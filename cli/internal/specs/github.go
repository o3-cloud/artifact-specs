package specs

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type GitHubClient struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewGitHubClient() *GitHubClient {
	return &GitHubClient{
		baseURL: "https://api.github.com",
		token:   os.Getenv("GITHUB_TOKEN"),
		client:  &http.Client{},
	}
}

func (g *GitHubClient) request(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if g.token != "" {
		req.Header.Set("Authorization", "token "+g.token)
	}
	
	return g.client.Do(req)
}

func (g *GitHubClient) FetchSpecs(repo Repository, specType SpecType) ([]*Spec, error) {
	path := fmt.Sprintf("specs/%s", specType)
	url := fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", g.baseURL, repo.String(), path, repo.Ref)
	
	resp, err := g.request(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch directory listing: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}
	
	var contents []GitHubContent
	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
		return nil, fmt.Errorf("failed to decode directory listing: %w", err)
	}
	
	var specs []*Spec
	for _, content := range contents {
		if !strings.HasSuffix(content.Name, ".schema.json") {
			continue
		}
		
		spec, err := g.fetchSpec(repo, specType, content)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to fetch spec %s: %v\n", content.Name, err)
			continue
		}
		
		specs = append(specs, spec)
	}
	
	return specs, nil
}

func (g *GitHubClient) fetchSpec(repo Repository, specType SpecType, content GitHubContent) (*Spec, error) {
	resp, err := g.request(content.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spec content: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}
	
	var fileContent GitHubContent
	if err := json.NewDecoder(resp.Body).Decode(&fileContent); err != nil {
		return nil, fmt.Errorf("failed to decode file content: %w", err)
	}
	
	if fileContent.Encoding != "base64" {
		return nil, fmt.Errorf("unexpected encoding: %s", fileContent.Encoding)
	}
	
	schemaData, err := base64.StdEncoding.DecodeString(fileContent.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 content: %w", err)
	}
	
	// Parse the schema to extract title
	var schemaDoc map[string]interface{}
	if err := json.Unmarshal(schemaData, &schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON: %w", err)
	}
	
	title := content.Name
	if t, ok := schemaDoc["title"].(string); ok && t != "" {
		title = t
	}
	
	slug := strings.TrimSuffix(content.Name, ".schema.json")
	
	return &Spec{
		Slug:   slug,
		Title:  title,
		Type:   specType,
		Ref:    repo.Ref,
		Path:   content.Path,
		URL:    content.HTMLURL,
		Schema: json.RawMessage(schemaData),
	}, nil
}

func (g *GitHubClient) FetchSpecByURL(url string) (*Spec, error) {
	resp, err := g.request(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spec from URL: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}
	
	var content GitHubContent
	if err := json.NewDecoder(resp.Body).Decode(&content); err != nil {
		return nil, fmt.Errorf("failed to decode spec content: %w", err)
	}
	
	if content.Encoding != "base64" {
		return nil, fmt.Errorf("unexpected encoding: %s", content.Encoding)
	}
	
	schemaData, err := base64.StdEncoding.DecodeString(content.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 content: %w", err)
	}
	
	// Parse the schema to extract title and determine type
	var schemaDoc map[string]interface{}
	if err := json.Unmarshal(schemaData, &schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON: %w", err)
	}
	
	title := content.Name
	if t, ok := schemaDoc["title"].(string); ok && t != "" {
		title = t
	}
	
	slug := strings.TrimSuffix(content.Name, ".schema.json")
	
	// Determine spec type from path
	specType := Artifacts
	if strings.Contains(content.Path, "extractors/") {
		specType = Extractors
	}
	
	return &Spec{
		Slug:   slug,
		Title:  title,
		Type:   specType,
		Path:   content.Path,
		URL:    content.HTMLURL,
		Schema: json.RawMessage(schemaData),
	}, nil
}

func LoadSpecFromFile(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec file: %w", err)
	}
	
	// Parse the schema to extract title
	var schemaDoc map[string]interface{}
	if err := json.Unmarshal(data, &schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON: %w", err)
	}
	
	filename := filepath.Base(path)
	title := filename
	if t, ok := schemaDoc["title"].(string); ok && t != "" {
		title = t
	}
	
	slug := strings.TrimSuffix(filename, ".schema.json")
	
	// Determine spec type from path
	specType := Artifacts
	if strings.Contains(path, "extractors") {
		specType = Extractors
	}
	
	return &Spec{
		Slug:   slug,
		Title:  title,
		Type:   specType,
		Path:   path,
		Schema: json.RawMessage(data),
	}, nil
}