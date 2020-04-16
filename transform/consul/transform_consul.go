package consul

import (
	"os"

	transform "github.com/Aisuko/meshinfra/transform"
	"helm.sh/helm/v3/pkg/cli"
)

// TranformConsul is used to handler the parameters and transform Linekrd chart to Kubernetes mainifest
type TranformConsul struct {
	transformInfo *transform.Transform
}

var settings = cli.New()

func (tc *TranformConsul) transformConsul() (string, error) {
	_ = os.Setenv("HELM_NAMESPACE", tc.transformInfo.Namespace)

	err := tc.transformInfo.AddNewChartRepo(settings)
	if err != nil {
		return "", err
	}

	tc.transformInfo.UpdateRepo(settings)

	release, err := tc.transformInfo.RenderChart(settings)

	if err != nil {
		return "", err
	}

	return release.Manifest, nil
}

// ExeTransformConsul is used to execute the transforming
func ExeTransformConsul(chartName, releaseName, namespace, repoName, chartRepoAddress string, isHa bool, args map[string]string) (string, error) {
	t := &TranformConsul{
		&transform.Transform{
			ChartName:        chartName,
			ReleaseName:      releaseName,
			Namespace:        namespace,
			RepoName:         repoName,
			ChartRepoAddress: chartRepoAddress,
			IsHa:             isHa,
			Args:             args,
		},
	}
	return t.transformConsul()
}
