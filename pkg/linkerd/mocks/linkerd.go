package mocks

import "github.com/stretchr/testify/mock"

type MockLinkerd struct {
	mock.Mock
}

func (mk *MockLinkerd) Deploy(kubeConfig, name, namespace, chartPath, valuesPath string, valueString map[string]string) error {
	args := mk.Called(kubeConfig, name, namespace, chartPath, valuesPath, valueString)

	return args.Error(0)
}

func (h *MockLinkerd) Destroy(kubeConfig, name, namespace string) error {
	args := h.Called(kubeConfig, name, namespace)

	return args.Error(0)
}
