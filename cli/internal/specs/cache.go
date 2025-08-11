package specs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/o3-cloud/artifact-specs/cli/internal/config"
	"github.com/o3-cloud/artifact-specs/cli/internal/logging"
)

type Cache struct {
	cacheDir string
}

func NewCache() (*Cache, error) {
	cacheDir, err := config.GetCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache directory: %w", err)
	}
	
	return &Cache{cacheDir: cacheDir}, nil
}

func (c *Cache) ensureCacheDir() error {
	return os.MkdirAll(c.cacheDir, 0755)
}

func (c *Cache) SaveSpec(spec *Spec) error {
	if err := c.ensureCacheDir(); err != nil {
		return err
	}
	
	spec.LastUpdated = time.Now()
	
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal spec: %w", err)
	}
	
	filename := fmt.Sprintf("%s_%s.json", spec.Type, spec.Slug)
	path := filepath.Join(c.cacheDir, filename)
	
	return os.WriteFile(path, data, 0644)
}

func (c *Cache) LoadSpec(specType SpecType, slug string) (*Spec, error) {
	filename := fmt.Sprintf("%s_%s.json", specType, slug)
	path := filepath.Join(c.cacheDir, filename)
	
	logging.Trace("Loading spec from cache file", map[string]interface{}{
		"path": path,
		"slug": slug,
		"type": specType,
	})
	
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("spec %s not found in cache", slug)
		}
		return nil, fmt.Errorf("failed to read cached spec: %w", err)
	}
	
	var spec Spec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached spec: %w", err)
	}
	
	logging.Trace("Successfully loaded spec from cache", map[string]interface{}{
		"slug":         spec.Slug,
		"title":        spec.Title,
		"schema_bytes": len(spec.Schema),
	})
	
	return &spec, nil
}

func (c *Cache) ListSpecs(specType SpecType) ([]*Spec, error) {
	if err := c.ensureCacheDir(); err != nil {
		return nil, err
	}
	
	pattern := fmt.Sprintf("%s_*.json", specType)
	matches, err := filepath.Glob(filepath.Join(c.cacheDir, pattern))
	if err != nil {
		return nil, fmt.Errorf("failed to glob cache files: %w", err)
	}
	
	var specs []*Spec
	for _, match := range matches {
		data, err := os.ReadFile(match)
		if err != nil {
			continue // Skip files we can't read
		}
		
		var spec Spec
		if err := json.Unmarshal(data, &spec); err != nil {
			continue // Skip files we can't unmarshal
		}
		
		specs = append(specs, &spec)
	}
	
	return specs, nil
}

func (c *Cache) SearchSpecs(specType SpecType, search string) ([]*Spec, error) {
	logging.Debug("Searching cached specs", map[string]interface{}{
		"type":   specType,
		"search": search,
	})
	
	specs, err := c.ListSpecs(specType)
	if err != nil {
		return nil, err
	}
	
	logging.Debug("Loaded specs from cache", map[string]interface{}{
		"count": len(specs),
	})
	
	if search == "" {
		return specs, nil
	}
	
	search = strings.ToLower(search)
	var filtered []*Spec
	
	for _, spec := range specs {
		if strings.Contains(strings.ToLower(spec.Slug), search) ||
		   strings.Contains(strings.ToLower(spec.Title), search) {
			filtered = append(filtered, spec)
		}
	}
	
	logging.Debug("Filtered specs by search", map[string]interface{}{
		"search":         search,
		"filtered_count": len(filtered),
	})
	
	return filtered, nil
}

func (c *Cache) Clear() error {
	return os.RemoveAll(c.cacheDir)
}

func (c *Cache) Dir() string {
	return c.cacheDir
}