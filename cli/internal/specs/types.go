package specs

import (
	"encoding/json"
	"time"
)

type SpecType string

const (
	Artifacts  SpecType = "artifacts"
	Extractors SpecType = "extractors"
)

type Spec struct {
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Type        SpecType  `json:"type"`
	Ref         string    `json:"ref"`
	Path        string    `json:"path"`
	URL         string    `json:"url"`
	Schema      json.RawMessage `json:"schema"`
	LastUpdated time.Time `json:"last_updated"`
}

type GitHubContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	SHA         string `json:"sha"`
	Size        int    `json:"size"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	GitURL      string `json:"git_url"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
	Content     string `json:"content,omitempty"`
	Encoding    string `json:"encoding,omitempty"`
}

type Repository struct {
	Owner string
	Name  string
	Ref   string
}

func (r Repository) String() string {
	return r.Owner + "/" + r.Name
}