package common

import (
	"fmt"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/chart"
)

// Debug as a common tool function for transform function
func Debug(format string, args ...interface{}) {
	format = fmt.Sprintf("[debug] %s\n", format)
	fmt.Printf(format, args...)
}

// IsChartInstallable is a tool function to check the chart can be used
func IsChartInstallable(ch *chart.Chart) (bool, error) {
	switch ch.Metadata.Type {
	case "", "application":
		return true, nil
	}
	return false, errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}
