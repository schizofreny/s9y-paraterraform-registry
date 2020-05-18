package githubutils

import (
	"context"
	"fmt"
	"log"
	"os"

	"golang.org/x/oauth2"

	"github.com/google/go-github/v31/github"
	"github.com/schizofreny/s9y-paraterraform-registry/utils"
)

const releaseName = "binaries"

// GithubRegistry struct
type GithubRegistry struct {
	client *github.Client
	ctx    context.Context
	config GithubRegistryConfig
}

// GithubRegistryConfig token and repository path
type GithubRegistryConfig struct {
	Token        string
	Organization string
	Repository   string
}

// NewGithubRegistry github api wrapper
func NewGithubRegistry(config GithubRegistryConfig) (*GithubRegistry, error) {

	ctx := context.Background()

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.Token},
	)

	tc := oauth2.NewClient(ctx, ts)

	d := &GithubRegistry{
		ctx:    context.Background(),
		client: github.NewClient(tc),
		config: config,
	}

	return d, nil
}

// GithubArtifacts find artifacts o registry release
func (c *GithubRegistry) GithubArtifacts() (*github.RepositoryRelease, error) {
	release, _, err := c.client.Repositories.GetReleaseByTag(c.ctx, c.config.Organization, c.config.Repository, releaseName)

	return release, err
}

// UploadArtifact uploads file to binaries release
func (c *GithubRegistry) UploadArtifact(filePath string, name string) error {
	release, _, err := c.client.Repositories.GetReleaseByTag(c.ctx, c.config.Organization, c.config.Repository, releaseName)

	if err != nil {
		panic(err)
	}

	uploadOptions := github.UploadOptions{
		Name:  name,
		Label: fmt.Sprintf("%s@%s", name, utils.Sha256sum(filePath)),
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}

	log.Printf("----> Uploading %s", name)
	_, _, err = c.client.Repositories.UploadReleaseAsset(c.ctx, c.config.Organization, c.config.Repository, release.GetID(), &uploadOptions, file)
	return err
}

// UploadFileToGit replace file in git
func (c *GithubRegistry) UploadFileToGit(path string, content []byte) error {
	// if file already exists, previous file sha1 must be used to replace content
	fileContent, _, _, err := c.client.Repositories.GetContents(c.ctx, c.config.Organization, c.config.Repository, path, &github.RepositoryContentGetOptions{})
	if err != nil {
		return err
	}

	shasm := ""
	if fileContent != nil {
		shasm = *fileContent.SHA
	}

	message := "Update para.idx.yaml"
	_, _, err = c.client.Repositories.UpdateFile(c.ctx, c.config.Organization, c.config.Repository, path, &github.RepositoryContentFileOptions{
		Content: content,
		Message: &message,
		SHA:     &shasm,
	})
	if err != nil {
		return err
	}
	return nil
}
