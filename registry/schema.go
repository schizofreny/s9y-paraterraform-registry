package registry

// Formula root formula definition
type Formula struct {
	Name      string
	Goversion string
	Versions  []struct {
		Git       string
		Name      string
		Script    []string
		Ref       string
		Artifacts []Artifact
	}
}

// Version of formula
type Version struct {
	Git  string `yaml:"git"`
	Name string `yaml:"name"`
}

// Artifact of version
type Artifact struct {
	Type string
	Arch string
	Path string
}
