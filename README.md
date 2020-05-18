# s9y-paraterraform-registry
- automagically builds terraform providers from git
- publishes build as github artifact
- generate para.idx.yaml for https://github.com/paraterraform/para

# usage
- add formula
- travis build artifacts
- add this to para.cfg.yaml in terraform project

```index: https://raw.githubusercontent.com/schizofreny/s9y-paraterraform-registry/master/para.idx.yaml```

# dev
GITHUB_TOKEN="xxx" GITHUB_ORGANIZATION="schizofreny" GITHUB_PROJECT="s9y-paraterraform-registry" go run .