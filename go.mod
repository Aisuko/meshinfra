module github.com/Aisuko/meshinfra

go 1.14

require (
	github.com/gofrs/flock v0.7.1
	github.com/golang/mock v1.2.0
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.5.0
	github.com/stretchr/testify v1.4.0
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/yaml.v2 v2.2.8
	helm.sh/helm/v3 v3.1.2
	rsc.io/letsencrypt v0.0.3 // indirect
)

replace github.com/Aisuko/meshinfra/pkg/linkerd/mocks => ./pkg/linkerd/mocks
