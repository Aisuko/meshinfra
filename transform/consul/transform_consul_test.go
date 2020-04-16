package consul

import (
	"fmt"
	"testing"
)

var (
	chartName        = "consul"
	releaseName      = "consul-0.19.0"
	namespace        = "default"
	repoName         = "incubator"
	chartRepoAddress = "https://aisuko.github.io/adapter-charts/incubator"
	args             = map[string]string{}
	isHa             = false
)

func TestExeTransformConsul(t *testing.T) {
	strManifest, err := ExeTransformConsul(chartName, releaseName, namespace, repoName, chartRepoAddress, isHa, args)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(strManifest)
}
