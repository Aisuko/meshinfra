package linkerd

import (
	"testing"

	mocks "github.com/Aisuko/meshinfra/pkg/linkerd/mocks"
	"github.com/stretchr/testify/mock"
)

func TestHelm(t *testing.T) {
	mk := mocks.MockLinkerd{}
	mk.On("Deploy", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mk.On("Destroy", mock.Anything, mock.Anything, mock.Anything).Return(nil)
}
