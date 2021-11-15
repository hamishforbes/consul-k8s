module github.com/hashicorp/consul-k8s/cli

go 1.16

require (
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/bgentry/speakeasy v0.1.0
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/fatih/color v1.12.0
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/hashicorp/consul-k8s/charts v0.0.0-00010101000000-000000000000
	github.com/hashicorp/go-hclog v0.16.2
	github.com/hashicorp/go-multierror v1.1.0 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/kr/text v0.2.0
	github.com/mattn/go-isatty v0.0.13
	github.com/mitchellh/cli v1.1.2
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/olekukonko/tablewriter v0.0.4
	github.com/onsi/gomega v1.15.0 // indirect
	github.com/posener/complete v1.2.3
	github.com/stretchr/testify v1.7.0
	go.starlark.net v0.0.0-20200707032745-474f21a9602d // indirect
	golang.org/x/sys v0.0.0-20211013075003-97ac67df715c // indirect
	google.golang.org/appengine v1.6.7 // indirect
	helm.sh/helm/v3 v3.6.1
	k8s.io/api v0.22.2
	k8s.io/apiextensions-apiserver v0.22.2 // indirect
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
