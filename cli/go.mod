module github.com/hashicorp/consul-k8s/cli

go 1.16

require (
	github.com/bgentry/speakeasy v0.1.0
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/fatih/color v1.12.0
	github.com/hashicorp/consul-k8s/charts v0.0.0-00010101000000-000000000000
	github.com/hashicorp/consul-k8s/control-plane v0.0.0-20211108194959-03d4b932256b
	github.com/hashicorp/go-hclog v0.16.2
	github.com/kr/text v0.2.0
	github.com/mattn/go-isatty v0.0.13
	github.com/mitchellh/cli v1.1.2
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/olekukonko/tablewriter v0.0.4
	github.com/posener/complete v1.2.3
	github.com/stretchr/testify v1.7.0
	go.starlark.net v0.0.0-20200707032745-474f21a9602d // indirect
	golang.org/x/sys v0.0.0-20211013075003-97ac67df715c // indirect
	helm.sh/helm/v3 v3.6.1
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/cli-runtime v0.21.0
	k8s.io/client-go v0.22.2
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/yaml v1.2.0
)

// This replace directive is to avoid having to manually bump the version of the charts module upon changes to the Helm
// chart. When the CLI compiles, all changes to the local charts directory are picked up automatically. This directive
// works because of the monorepo setup, where the charts module and CLI module are in the same repository. Otherwise,
// this won't work.
replace github.com/hashicorp/consul-k8s/charts => ../charts
