package linkerd

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	util "github.com/Aisuko/meshinfra/pkg/ioutil"

	transform "github.com/Aisuko/meshinfra/transform"
	"github.com/gofrs/flock"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/strvals"
)

// TranformLinkerd is used to handler the parameters and transform Linekrd chart to Kubernetes mainifest
type TranformLinkerd struct {
	transformInfo *transform.Transform
}

var settings = cli.New()

func (t *TranformLinkerd) addNewChartRepo() (err error) {
	repoFile := settings.RepositoryConfig

	//Ensure the file directory exists as it is required for file locking
	err = os.MkdirAll(filepath.Dir(repoFile), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	fileLock := flock.New(strings.Replace(repoFile, filepath.Ext(repoFile), ".lock", 1))
	lockCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()
	locked, err := fileLock.TryLockContext(lockCtx, time.Second)
	if err == nil && locked {
		defer util.SafeUnLock(fileLock, &err)
	}

	if err != nil {
		return err
	}

	// Need to check filepath
	b, err := ioutil.ReadFile(filepath.Clean(repoFile))
	if err != nil && !os.IsNotExist(err) {
		t.transformInfo.Debug("Read %s repo file error", t.transformInfo.RepoName)
		return err
	}

	var f repo.File
	if err := yaml.Unmarshal(b, &f); err != nil {
		return err
	}

	if f.Has(t.transformInfo.RepoName) {
		t.transformInfo.Debug("Repository name %s already exists", t.transformInfo.RepoName)
	}

	c := repo.Entry{
		Name: t.transformInfo.RepoName,
		URL:  t.transformInfo.ChartRepoAddress,
	}

	r, err := repo.NewChartRepository(&c, getter.All(settings))
	if err != nil {
		return err
	}

	if _, err := r.DownloadIndexFile(); err != nil {
		err := errors.Wrapf(err, "looks like %q is not a valid chart repository or cannot be reached", t.transformInfo.ChartRepoAddress)
		return err
	}

	f.Update(&c)

	if err := f.WriteFile(repoFile, 0644); err != nil {
		t.transformInfo.Debug("Add the %s chart repo failed", t.transformInfo.RepoName)
		return err
	}

	return nil
}

func (t *TranformLinkerd) updateRepo() {
	repoFile := settings.RepositoryConfig
	f, err := repo.LoadFile(repoFile)
	if os.IsNotExist(errors.Cause(err)) || len(f.Repositories) == 0 {
		log.Fatal(errors.New("No repositories found. You must add one before updating"))
	}

	var repos []*repo.ChartRepository
	for _, cfg := range f.Repositories {
		r, err := repo.NewChartRepository(cfg, getter.All(settings))
		if err != nil {
			log.Fatal(err)
		}
		repos = append(repos, r)
	}

	var wg sync.WaitGroup
	for _, re := range repos {
		wg.Add(1)
		go func(re *repo.ChartRepository) {
			defer wg.Done()
			if _, err := re.DownloadIndexFile(); err != nil {
				log.Fatal(err)
				t.transformInfo.Debug("Update %s repo index failed", t.transformInfo.RepoName)
			}
		}(re)
	}
	wg.Wait()
	t.transformInfo.Debug("Update %s repo index succeed", t.transformInfo.RepoName)
}

func (t *TranformLinkerd) renderChart() (*release.Release, error) {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), t.transformInfo.Debug); err != nil {
		return nil, err
	}

	client := action.NewInstall(actionConfig)

	if client.Version == "" && client.Devel {
		client.Version = ">0.0.0-0"
	}

	client.ReleaseName = t.transformInfo.ReleaseName

	cp, err := client.ChartPathOptions.LocateChart(fmt.Sprintf("%s/%s", t.transformInfo.RepoName, t.transformInfo.ChartName), settings)
	if err != nil {
		return nil, err
	}
	t.transformInfo.Debug("CHART PATH: %s\n", cp)

	p := getter.All(settings)
	valueOpts := &values.Options{}
	vals, err := valueOpts.MergeValues(p)
	if err != nil {
		return nil, err
	}

	//Add args
	if err = strvals.ParseInto(t.transformInfo.Args["--set"], vals); err != nil {
		return nil, (errors.Wrap(err, "failed parsing --set data"))
	}

	if err = strvals.ParseInto(t.transformInfo.Args["--set-file"], vals); err != nil {
		return nil, (errors.Wrap(err, "failed parsing --set-file data"))
	}

	// Check chart dependencies to make sure all are present in /charts
	chartRequested, err := loader.Load(cp)

	if err != nil {
		return nil, err
	}

	valsMerged, err := requestHa(t.transformInfo.IsHa, chartRequested, vals)

	if err != nil {
		t.transformInfo.Debug("Can not merge the value-ha.yaml %s\n", err)
		return nil, err
	}

	validInstallableChart, err := t.transformInfo.IsChartInstallable(chartRequested)
	if !validInstallableChart {
		return nil, err
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		// If CheckDependencies returns an error, we have unfulfilled dependencies.
		// As of Helm 2.4.0, this is treated as a stopping condition:
		// https://github.com/helm/helm/issues/2209
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			if client.DependencyUpdate {
				man := &downloader.Manager{
					Out:              os.Stdout,
					ChartPath:        cp,
					Keyring:          client.ChartPathOptions.Keyring,
					SkipUpdate:       false,
					Getters:          p,
					RepositoryConfig: settings.RepositoryConfig,
					RepositoryCache:  settings.RepositoryCache,
				}
				if err := man.Update(); err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}
	}

	client.Namespace = settings.Namespace()
	client.DryRun = true
	client.ClientOnly = true

	release, err := client.Run(chartRequested, valsMerged)
	if err != nil {
		return nil, err
	}

	return release, nil
}

func (t *TranformLinkerd) transformLinkerd() (string, error) {
	_ = os.Setenv("HELM_NAMESPACE", t.transformInfo.Namespace)

	err := t.addNewChartRepo()
	if err != nil {
		return "", err
	}

	t.updateRepo()

	release, err := t.renderChart()

	if err != nil {
		return "", err
	}

	return release.Manifest, nil
}

// ExeTransformLinkerd is used to execute the transforming
func ExeTransformLinkerd(chartName, releaseName, namespace, repoName, chartRepoAddress string, isHa bool, args map[string]string) (string, error) {
	t := &TranformLinkerd{
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
	return t.transformLinkerd()
}

func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = mergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

func requestHa(ha bool, chart *chart.Chart, vals map[string]interface{}) (map[string]interface{}, error) {
	var dataVH []byte
	for _, v := range chart.Files {
		if v.Name == "values-ha.yaml" {
			dataVH = v.Data
		}
	}
	currentMap := map[string]interface{}{}
	if err := yaml.Unmarshal(dataVH, &currentMap); err != nil {
		return nil, err
	}
	vals = mergeMaps(currentMap, vals)
	return vals, nil
}
