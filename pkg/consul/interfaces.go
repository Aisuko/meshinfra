package consul

import "helm.sh/helm/v3/pkg/release"

//go:generate mockgen -source ./interfaces.go -destination ./mocks/mock_interfaces.go

// Consul interface is used to define the way how to transform the chart
type Consul interface {
	AddRepo() (err error)
	UpdateRepo()
	TranformChart() (*release.Release, error)
}
