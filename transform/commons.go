package transform

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
)

// Transform is used to provider the basement parameters to the transform process
type Transform struct {
	ChartName        string
	ReleaseName      string
	Namespace        string
	RepoName         string
	ChartRepoAddress string
	Args             map[string]string
	IsHa             bool
}

// AddNewChartRepo is used to add repo for meshinfra
func (t *Transform) AddNewChartRepo(settings *cli.EnvSettings) (err error) {
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
		t.Debug("Read %s repo file error", t.RepoName)
		return err
	}

	var f repo.File
	if err := yaml.Unmarshal(b, &f); err != nil {
		return err
	}

	if f.Has(t.RepoName) {
		t.Debug("Repository name %s already exists", t.RepoName)
	}

	c := repo.Entry{
		Name: t.RepoName,
		URL:  t.ChartRepoAddress,
	}

	r, err := repo.NewChartRepository(&c, getter.All(settings))
	if err != nil {
		return err
	}

	if _, err := r.DownloadIndexFile(); err != nil {
		err := errors.Wrapf(err, "looks like %q is not a valid chart repository or cannot be reached", t.ChartRepoAddress)
		return err
	}

	f.Update(&c)

	if err := f.WriteFile(repoFile, 0644); err != nil {
		t.Debug("Add the %s chart repo failed", t.RepoName)
		return err
	}

	return nil
}

// UpdateRepo is used to update the chart repo index.yaml
func (t *Transform) UpdateRepo(settings *cli.EnvSettings) {
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
				t.Debug("Update %s repo index failed", t.RepoName)
			}
		}(re)
	}
	wg.Wait()
	t.Debug("Update %s repo index succeed", t.RepoName)
}

// RenderChart is used to  tranform the chart to Kubernetes manifest
func (t *Transform) RenderChart(settings *cli.EnvSettings) (*release.Release, error) {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), t.Debug); err != nil {
		return nil, err
	}

	client := action.NewInstall(actionConfig)

	if client.Version == "" && client.Devel {
		client.Version = ">0.0.0-0"
	}

	client.ReleaseName = t.ReleaseName

	cp, err := client.ChartPathOptions.LocateChart(fmt.Sprintf("%s/%s", t.RepoName, t.ChartName), settings)
	if err != nil {
		return nil, err
	}
	t.Debug("CHART PATH: %s\n", cp)

	p := getter.All(settings)
	valueOpts := &values.Options{}
	vals, err := valueOpts.MergeValues(p)
	if err != nil {
		return nil, err
	}

	//Add args
	// if err = strvals.ParseInto(t.Args["--set"], vals); err != nil {
	// 	return nil, (errors.Wrap(err, "failed parsing --set data"))
	// }

	// Check chart dependencies to make sure all are present in /charts
	chartRequested, err := loader.Load(cp)

	if err != nil {
		return nil, err
	}

	validInstallableChart, err := t.IsChartInstallable(chartRequested)
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

	release, err := client.Run(chartRequested, vals)
	if err != nil {
		return nil, err
	}

	return release, nil
}

// Debug is used default output the context to stout
func (t *Transform) Debug(format string, args ...interface{}) {
	format = fmt.Sprintf("[debug] %s\n", format)
	fmt.Printf(format, args...)
}

// IsChartInstallable is used to check that the chart can be installed
func (t *Transform) IsChartInstallable(ch *chart.Chart) (bool, error) {
	switch ch.Metadata.Type {
	case "", "application":
		return true, nil
	}
	return false, errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}
