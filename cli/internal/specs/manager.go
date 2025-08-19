package specs

import (
    "fmt"
    "sort"
    "strings"

    "github.com/o3-cloud/artifact-specs/cli/internal/logging"
)

type Manager struct {
	cache  *Cache
	github *GitHubClient
}

func NewManager() (*Manager, error) {
	cache, err := NewCache()
	if err != nil {
		return nil, err
	}
	
	logging.Debug("Created specs manager with cache", map[string]interface{}{
		"cache_dir": cache.Dir(),
	})
	
	return &Manager{
		cache:  cache,
		github: NewGitHubClient(),
	}, nil
}

func (m *Manager) UpdateSpecs(repo Repository) error {
	// Fetch artifacts
	artifacts, err := m.github.FetchSpecs(repo, Artifacts)
	if err != nil {
		return fmt.Errorf("failed to fetch artifacts: %w", err)
	}
	
	// Fetch extractors
	extractors, err := m.github.FetchSpecs(repo, Extractors)
	if err != nil {
		return fmt.Errorf("failed to fetch extractors: %w", err)
	}
	
	// Save to cache
	for _, spec := range artifacts {
		if err := m.cache.SaveSpec(spec); err != nil {
			return fmt.Errorf("failed to cache artifact spec %s: %w", spec.Slug, err)
		}
	}
	
	for _, spec := range extractors {
		if err := m.cache.SaveSpec(spec); err != nil {
			return fmt.Errorf("failed to cache extractor spec %s: %w", spec.Slug, err)
		}
	}
	
    logging.Info("Specs updated", map[string]interface{}{"artifacts": len(artifacts), "extractors": len(extractors)})
    return nil
}

func (m *Manager) ListSpecs(specType SpecType, search string) ([]*Spec, error) {
	logging.Debug("Listing specs", map[string]interface{}{
		"type":   specType,
		"search": search,
	})
	
	specs, err := m.cache.SearchSpecs(specType, search)
	if err != nil {
		return nil, err
	}
	
	logging.Debug("Found specs", map[string]interface{}{
		"count": len(specs),
	})
	
	// Sort by slug for consistent output
	sort.Slice(specs, func(i, j int) bool {
		return specs[i].Slug < specs[j].Slug
	})
	
	return specs, nil
}

func (m *Manager) GetSpec(specType SpecType, identifier string) (*Spec, error) {
	// First try exact slug match
	spec, err := m.cache.LoadSpec(specType, identifier)
	if err == nil {
		return spec, nil
	}
	
	// Try fuzzy matching
	specs, err := m.cache.SearchSpecs(specType, identifier)
	if err != nil {
		return nil, err
	}
	
	if len(specs) == 0 {
		return nil, fmt.Errorf("no specs found matching '%s'", identifier)
	}
	
	if len(specs) == 1 {
		return specs[0], nil
	}
	
	// Multiple matches - find best match
	var exactMatches []*Spec
	var partialMatches []*Spec
	
	lower := strings.ToLower(identifier)
	
	for _, spec := range specs {
		if strings.ToLower(spec.Slug) == lower {
			exactMatches = append(exactMatches, spec)
		} else if strings.Contains(strings.ToLower(spec.Slug), lower) {
			partialMatches = append(partialMatches, spec)
		}
	}
	
	if len(exactMatches) == 1 {
		return exactMatches[0], nil
	}
	
	if len(exactMatches) > 1 {
		return nil, fmt.Errorf("multiple exact matches for '%s': %v", identifier, getSpeSlugs(exactMatches))
	}
	
    if len(partialMatches) == 1 {
        logging.Warn("Using fuzzy match", map[string]interface{}{"match": partialMatches[0].Slug, "requested": identifier})
        return partialMatches[0], nil
    }
	
	return nil, fmt.Errorf("multiple matches for '%s': %v", identifier, getSpeSlugs(specs))
}

func (m *Manager) GetSpecByURL(url string) (*Spec, error) {
	return m.github.FetchSpecByURL(url)
}

func (m *Manager) GetSpecByPath(path string) (*Spec, error) {
	return LoadSpecFromFile(path)
}

func getSpeSlugs(specs []*Spec) []string {
	var slugs []string
	for _, spec := range specs {
		slugs = append(slugs, spec.Slug)
	}
	return slugs
}
