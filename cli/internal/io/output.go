package io

import (
	"fmt"
	"os"
	"path/filepath"
)

type OutputWriter struct {
	path string
}

func NewOutputWriter(path string) *OutputWriter {
	return &OutputWriter{path: path}
}

func (w *OutputWriter) WriteOutput(content string) error {
	if w.path == "" {
		// Write to stdout
		fmt.Print(content)
		return nil
	}
	
	// Ensure directory exists
	dir := filepath.Dir(w.path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}
	
	// Write to file
	if err := os.WriteFile(w.path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}
	
	fmt.Fprintf(os.Stderr, "Output written to %s\n", w.path)
	return nil
}

func (w *OutputWriter) WriteJSON(content string, compact bool) error {
	if !compact {
		// Content is already formatted
		return w.WriteOutput(content)
	}
	
	// TODO: Implement JSON minification if needed
	return w.WriteOutput(content)
}

func GenerateOutputPath(basePath, extension string) string {
	if basePath == "" {
		return fmt.Sprintf("out%s", extension)
	}
	
	dir := filepath.Dir(basePath)
	base := filepath.Base(basePath)
	nameWithoutExt := base[:len(base)-len(filepath.Ext(base))]
	
	return filepath.Join(dir, nameWithoutExt+extension)
}