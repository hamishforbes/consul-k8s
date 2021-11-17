package vault

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/hashicorp/consul-k8s/acceptance/framework/consul"
	"github.com/hashicorp/consul-k8s/acceptance/framework/helpers"
	"github.com/hashicorp/consul-k8s/acceptance/framework/logger"
	"github.com/hashicorp/consul-k8s/acceptance/framework/vault"
	"github.com/stretchr/testify/require"
	"testing"
)

// generateGossipSecret generates a random 32 byte secret returned as a base64 encoded string.
func generateGossipSecret() (string, error) {
	// This code was copied from Consul's Keygen command:
	// https://github.com/hashicorp/consul/blob/d652cc86e3d0322102c2b5e9026c6a60f36c17a5/command/keygen/keygen.go
	key := make([]byte, 32)
	n, err := rand.Reader.Read(key)
	if err != nil {
		return "", fmt.Errorf("error reading random data: %s", err)
	}
	if n != 32 {
		return "", fmt.Errorf("couldn't read enough entropy")
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// Installs Vault, bootstraps it with secrets, policies, and Kube Auth Method
// then creates a gossip encryption secret and uses this to bootstrap Consul.
func TestVault_BootstrapConsulGossipEncryptionKey(t *testing.T) {
	cfg := suite.Config()
	ctx := suite.Environment().DefaultContext(t)

	consulReleaseName := helpers.RandomName()
	vaultReleaseName := helpers.RandomName()
	consulClientServiceAccountName := fmt.Sprintf("%s-consul-client", consulReleaseName)
	consulServerServiceAccountName := fmt.Sprintf("%s-consul-server", consulReleaseName)

	vaultCluster := vault.NewVaultCluster(t, nil, ctx, cfg, vaultReleaseName)
	vaultCluster.Create(t, ctx)
	// Vault is now installed in the cluster.

	// Now fetch the Vault client so we can create the policies and secrets.
	vaultClient := vaultCluster.VaultClient(t)

	// Create the Vault Policy for the gossip key.
	logger.Log(t, "Creating the gossip policy")
	rules := `
path "consul/data/secret/gossip" {
  capabilities = ["read"]
}`
	err := vaultClient.Sys().PutPolicy("consul-gossip", rules)
	require.NoError(t, err)

	// Create the Auth Roles for consul-server + consul-client.
	logger.Log(t, "Creating the consul-server and consul-client-roles")
	params := map[string]interface{}{
		"bound_service_account_names":      consulClientServiceAccountName,
		"bound_service_account_namespaces": "default",
		"policies":                         "consul-gossip",
		"ttl":                              "24h",
	}
	_, err = vaultClient.Logical().Write("auth/kubernetes/role/consul-client", params)
	require.NoError(t, err)

	params = map[string]interface{}{
		"bound_service_account_names":      consulServerServiceAccountName,
		"bound_service_account_namespaces": "default",
		"policies":                         "consul-gossip",
		"ttl":                              "24h",
	}
	_, err = vaultClient.Logical().Write("auth/kubernetes/role/consul-server", params)
	require.NoError(t, err)

	gossipKey, err := generateGossipSecret()
	require.NoError(t, err)

	// Create the gossip secret.
	logger.Log(t, "Creating the gossip secret")
	params = map[string]interface{}{
		"data": map[string]interface{}{
			"gossip": gossipKey,
		},
	}
	_, err = vaultClient.Logical().Write("consul/data/secret/gossip", params)
	require.NoError(t, err)

	consulHelmValues := map[string]string{
		"server.enabled":  "true",
		"server.replicas": "1",

		"connectInject.enabled": "true",

		"global.secretsBackend.vault.enabled":          "true",
		"global.secretsBackend.vault.consulServerRole": "consul-server",
		"global.secretsBackend.vault.consulClientRole": "consul-client",

		"global.acls.manageSystemACLs":       "true",
		"global.tls.enabled":                 "true",
		"global.gossipEncryption.secretName": "consul/data/secret/gossip",
		"global.gossipEncryption.secretKey":  "gossip",
	}
	logger.Log(t, "Installing Consul")
	consulCluster := consul.NewHelmCluster(t, consulHelmValues, ctx, cfg, consulReleaseName)
	consulCluster.Create(t)

	// Validate that the gossip encryption key is set correctly.
	logger.Log(t, "Validating the gossip key has been set correctly.")
	consulClient := consulCluster.SetupConsulClient(t, true)
	keys, err := consulClient.Operator().KeyringList(nil)
	require.NoError(t, err)
	// We use keys[0] because KeyringList returns a list of keyrings for each dc, in this case there is only 1 dc.
	require.Equal(t, 1, keys[0].PrimaryKeys[gossipKey])
}

// Installs Vault, bootstraps it with secrets, policies, and Kube Auth Method
// then sets up a root CA, intermediate CA and bootstraps vault with the PKI engine
// for ServerTLS certs.
func TestVault_BootstrapConsulServerTLS(t *testing.T) {
	cfg := suite.Config()
	ctx := suite.Environment().DefaultContext(t)

	consulReleaseName := helpers.RandomName()
	vaultReleaseName := helpers.RandomName()
	consulServerServiceAccountName := fmt.Sprintf("%s-consul-server", consulReleaseName)
	consulClientServiceAccountName := fmt.Sprintf("%s-consul-client", consulReleaseName)

	vaultCluster := vault.NewVaultCluster(t, nil, ctx, cfg, vaultReleaseName)
	vaultCluster.Create(t, ctx)
	// Vault is now installed in the cluster.

	// Now fetch the Vault client so we can create the policies and secrets.
	vaultClient := vaultCluster.VaultClient(t)

	// Using : https://learn.hashicorp.com/tutorials/consul/vault-pki-consul-secure-tls

	// Generate the root CA.
	params := map[string]interface{}{
		"common_name": "dc1.consul",
		"ttl":         "24h",
	}
	_, err := vaultClient.Logical().Write("pki/root/generate/internal", params)
	require.NoError(t, err)

	// Configure the CA and CRL URLs.
	params = map[string]interface{}{
		"issuing_certificates":    "http://127.0.0.1:8200/v1/pki/ca",
		"crl_distribution_points": "http://127.0.0.1:8200/v1/pki/crl",
	}
	_, err = vaultClient.Logical().Write("pki/config/urls", params)
	require.NoError(t, err)

	// Generate an intermediate CA.
	params = map[string]interface{}{
		"common_name": "dc1.consul Intermediate Authority",
	}
	resp, err := vaultClient.Logical().Write("pki_int/intermediate/generate/internal", params)
	require.NoError(t, err)
	csr := resp.Data["csr"].(string)

	// Sign the CSR and import the certificate into Vault.
	params = map[string]interface{}{
		"csr":         csr,
		"common_name": "dc1.consul",
		"ttl":         "24h",
	}
	resp, err = vaultClient.Logical().Write("pki/root/sign-intermediate", params)
	require.NoError(t, err)
	intermediateCert := resp.Data["certificate"]

	params = map[string]interface{}{
		"certificate": intermediateCert,
	}
	_, err = vaultClient.Logical().Write("pki_int/intermediate/set-signed", params)
	require.NoError(t, err)

	// Create a Vault PKI Role
	params = map[string]interface{}{
		"allowed_domains":  "dc1.consul",
		"allow_subdomains": "true",
		"generate_lease":   "true",
		"max_ttl":          "1h",
	}

	_, err = vaultClient.Logical().Write("pki_int/roles/consul-server", params)
	require.NoError(t, err)

	rules := `
path "pki_int/issue/consul-server" {
  capabilities = ["create", "update"]
}`
	err = vaultClient.Sys().PutPolicy("consul-server", rules)
	require.NoError(t, err)

	logger.Log(t, "Creating the consul-server role.")
	params = map[string]interface{}{
		"bound_service_account_names":      consulServerServiceAccountName,
		"bound_service_account_namespaces": "default",
		"policies":                         "consul-server",
		"ttl":                              "24h",
	}
	_, err = vaultClient.Logical().Write("auth/kubernetes/role/consul-server", params)
	require.NoError(t, err)

	logger.Log(t, "Creating the consul-client role.")
	params["bound_service_account_names"] = consulClientServiceAccountName
	_, err = vaultClient.Logical().Write("auth/kubernetes/role/consul-client", params)
	require.NoError(t, err)

	consulHelmValues := map[string]string{
		"server.enabled":  "true",
		"server.replicas": "1",
		"global.secretsBackend.vault.consulServerRole": "consul-server",
		"global.secretsBackend.vault.consulClientRole": "consul-client",
		"server.serverCert.secretName":                 "pki_int/issue/consul-server",

		"connectInject.enabled": "true",

		"global.secretsBackend.vault.enabled": "true",
		"global.tls.enabled":                  "true",
		"global.tls.httpsOnly":                "false",
		"global.tls.enableAutoEncrypt":        "true",
	}

	logger.Log(t, "Installing Consul")
	consulCluster := consul.NewHelmCluster(t, consulHelmValues, ctx, cfg, consulReleaseName)
	consulCluster.Create(t)
	/*
		// Validate that the gossip encryption key is set correctly.
		logger.Log(t, "Validating the gossip key has been set correctly.")
		consulClient := consulCluster.SetupConsulClient(t, true)
		keys, err := consulClient.Operator().KeyringList(nil)
		require.NoError(t, err)
		// we use keys[0] because KeyringList returns a list of keyrings for each dc, in this case there is only 1 dc.
		require.Equal(t, 1, keys[0].PrimaryKeys[gossipKey])

	*/
}