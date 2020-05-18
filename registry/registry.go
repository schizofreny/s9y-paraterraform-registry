package registry

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Jeffail/gabs"
	gyaml "github.com/ghodss/yaml"
	"github.com/google/go-github/v31/github"
	"github.com/schizofreny/s9y-paraterraform-registry/dockerbuilder"
	"github.com/schizofreny/s9y-paraterraform-registry/githubutils"
	"github.com/schizofreny/s9y-paraterraform-registry/utils"
	"github.com/thoas/go-funk"
	"gopkg.in/yaml.v2"
)

var dockerWorkDir = "/usr/src/myapp"

// Registry struct
type Registry struct {
	github   *githubutils.GithubRegistry
	formulae []Formula
}

// NewRegistry initialize
func NewRegistry(githubConfig githubutils.GithubRegistryConfig) (*Registry, error) {

	ghr, err := githubutils.NewGithubRegistry(githubConfig)
	if err != nil {
		return nil, err
	}
	d := &Registry{
		github: ghr,
	}

	formulae, err := d.loadFormulae()
	if err != nil {
		return nil, err
	}
	log.Printf("====> Loaded %d formulae", len(formulae))
	d.formulae = formulae

	log.Printf("====> Validating artifacts")
	d.buildMissing()

	log.Printf("====> Building index")
	index, err := d.buildIndexYaml()
	if err != nil {
		return nil, err
	}

	log.Printf("====> Uploading index")
	err = d.github.UploadFileToGit("para.idx.yaml", index)
	if err != nil {
		return nil, err
	}

	return d, nil

}

func (r *Registry) loadFormulae() ([]Formula, error) {
	formulae := []Formula{}

	yamls, err := filepath.Glob("formulae/*.yaml")
	if err != nil {
		return nil, err
	}

	for _, f := range yamls {
		b, err := ioutil.ReadFile(f)
		if err != nil {
			return nil, err
		}

		var cfg Formula

		if err := yaml.Unmarshal(b, &cfg); err != nil {
			return nil, err
		}
		cfg.Name = utils.FileNameWithoutExtension(filepath.Base(f))
		formulae = append(formulae, cfg)
	}
	return formulae, nil
}

func artifactToFQN(artifact Artifact, formulaName string, versionName string) string {
	return fmt.Sprintf("terraform-%s-%s-%s-%s", artifact.Type, formulaName, versionName, artifact.Arch)
}

func (r *Registry) buildIndexYaml() ([]byte, error) {

	ghr, err := r.github.GithubArtifacts()
	if err != nil {
		return nil, err
	}

	assets := ghr.Assets

	jsonObj := gabs.New()

	for _, f := range r.formulae {
		for _, v := range f.Versions {
			for _, a := range v.Artifacts {
				fullName := artifactToFQN(a, f.Name, v.Name)

				ga := funk.Find(assets, func(gha *github.ReleaseAsset) bool {
					println(gha.GetName())
					parts := strings.Split(gha.GetLabel(), "@")
					if len(parts) != 2 {
						return false
					}
					return parts[0] == fullName

				})

				if ga == nil {
					return nil, fmt.Errorf("Asset %s not found on github", fullName)
				}
				asset := ga.(*github.ReleaseAsset)

				parts := strings.Split(asset.GetLabel(), "@")
				sha := parts[1]

				jsonObj.Set(asset.GetSize(), a.Type, f.Name, v.Name, a.Arch, "size")
				jsonObj.Set(fmt.Sprintf("sha256:%s", sha), a.Type, f.Name, v.Name, a.Arch, "digest")
				jsonObj.Set(asset.GetBrowserDownloadURL(), a.Type, f.Name, v.Name, a.Arch, "url")
			}
		}
	}

	bytes, err := gyaml.JSONToYAML([]byte(jsonObj.String()))

	return bytes, err
}

func (r *Registry) buildMissing() error {

	formulae := r.formulae

	ghr, err := r.github.GithubArtifacts()
	if err != nil {
		return err
	}

	existingAssets := funk.Map(ghr.Assets, func(asset *github.ReleaseAsset) string {
		name := asset.GetName()
		println(name)
		parts := strings.Split(name, "@")
		return parts[0]
	})

	for _, f := range formulae {
		for _, v := range f.Versions {

			missingArtifacts := funk.Filter(v.Artifacts, func(artifact Artifact) bool {
				artifactFQN := artifactToFQN(artifact, f.Name, v.Name)
				if funk.Contains(existingAssets, artifactFQN) == true {
					log.Printf("----> Already exists, skipping %s", artifactFQN)
					return false
				}
				return true
			}).([]Artifact)

			if len(missingArtifacts) == 0 {
				continue
			}

			log.Printf("====> Missing artifacts %v", missingArtifacts)

			container, err := dockerbuilder.NewGoDockerContainer(f.Goversion)

			if err != nil {
				return err
			}

			defer container.Kill()

			container.Exec([]string{"git", "clone", v.Git, dockerWorkDir}, "")
			container.Exec([]string{"git", "checkout", v.Ref}, "")

			container.ExecShellScriptLines(v.Script, dockerWorkDir)

			tmpdir, err := ioutil.TempDir("", f.Name)
			defer os.RemoveAll(tmpdir)

			for _, artifact := range missingArtifacts {
				artifactFQN := artifactToFQN(artifact, f.Name, v.Name)
				if err != nil {
					return err
				}
				dest := path.Join(tmpdir, artifactFQN)
				log.Printf("----> Uploading %s to github", dest)
				err := container.CopyFromContainer(path.Join(dockerWorkDir, artifact.Path), dest)
				if err != nil {
					return err
				}

				err = r.github.UploadArtifact(dest, artifactFQN)
				if err != nil {
					return err
				}

			}
		}
	}
	return nil
}
