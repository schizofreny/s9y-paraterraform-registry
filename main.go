package main

import (
	"log"

	"github.com/kelseyhightower/envconfig"
	"github.com/schizofreny/s9y-paraterraform-registry/githubutils"
	"github.com/schizofreny/s9y-paraterraform-registry/registry"
)

type specification struct {
	Token        string `required:"true" envconfig:"GITHUB_TOKEN"`
	Organization string `required:"true" envconfig:"GITHUB_ORGANIZATION"`
	Project      string `required:"true" envconfig:"GITHUB_PROJECT"`
}

func main() {
	var s specification
	err := envconfig.Process("", &s)
	if err != nil {
		log.Fatal(err.Error())
	}

	_, err = registry.NewRegistry(githubutils.GithubRegistryConfig{
		Token:        s.Token,
		Organization: s.Organization,
		Repository:   s.Project,
	})
	if err != nil {
		panic(err)
	}

}
