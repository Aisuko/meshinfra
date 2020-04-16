package linkerd

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

var (
	chartName        = "linkerd2"
	releaseName      = "linkerd2-2.7.0"
	namespace        = "linkerd"
	repoName         = "stable"
	chartRepoAddress = "https://aisuko.github.io/adapter-charts/stable"
	args             = map[string]string{
		"--set":      "identity.issuer.crtExpiry",
		"--set-file": "global.identityTrustAnchorsPEM,identity.issuer.tls.crtPEM,identity.issuer.tls.keyPEM",
	}
	isHa = true
)

func TestTransformLinkerd(t *testing.T) {

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	pathCaCrt := wd + "/testdata/ca.crt"

	pathIssuerCrt := wd + "/testdata/issuer.crt"

	pathIssuerKey := wd + "/testdata/issuer.key"

	extraCert := &ExtraCert{}
	extraCert.reader(pathCaCrt, t)
	extraCert.reader(pathIssuerCrt, t)
	extraCert.reader(pathIssuerKey, t)

	args["--set"] = string("identity.issuer.crtExpiry=2021-04-10T19:49:28Z")
	args["--set-file"] = fmt.Sprintf("global.identityTrustAnchorsPEM=%s,identity.issuer.tls.crtPEM=%s,identity.issuer.tls.keyPEM=%s", extraCert.bySlice[0], extraCert.bySlice[1], extraCert.bySlice[2])

	fmt.Println(args["--set-file"])

	strManifest, err := ExeTransformLinkerd(chartName, releaseName, namespace, repoName, chartRepoAddress, isHa, args)
	if err != nil {
		t.Fatal(err)
	}
	// fmt.Println(strManifest)
	bol := strings.ContainsAny("values-ha.yaml", strManifest)
	if !bol {
		t.Log("Can not find the values-ha.yaml string")
	} else {
		t.Log("Linkerd was deployed with High-Availability scenario succeed")
	}

}

func (extracert *ExtraCert) reader(pathOfFile string, t *testing.T) {
	byteContext, err := ioutil.ReadFile(pathOfFile)
	if err != nil {
		t.Fatal(err)
	}
	extracert.bySlice = append(extracert.bySlice, string(byteContext))
}

type ExtraCert struct {
	bySlice []string
}
