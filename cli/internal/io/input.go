package io

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type InputReader struct {
	source string
}

func NewInputReader(source string) *InputReader {
	return &InputReader{source: source}
}

func (r *InputReader) ReadInput() (string, error) {
	if r.source == "" || r.source == "-" {
		return r.readStdin()
	}
	
	info, err := os.Stat(r.source)
	if err != nil {
		return "", fmt.Errorf("failed to stat input: %w", err)
	}
	
	if info.IsDir() {
		return r.readDirectory()
	}
	
	return r.readFile(r.source)
}

func (r *InputReader) readStdin() (string, error) {
	var content strings.Builder
	scanner := bufio.NewScanner(os.Stdin)
	
	for scanner.Scan() {
		content.WriteString(scanner.Text())
		content.WriteString("\n")
	}
	
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read stdin: %w", err)
	}
	
	return content.String(), nil
}

func (r *InputReader) readFile(path string) (string, error) {
	if isBinary(path) {
		fmt.Fprintf(os.Stderr, "Warning: Skipping binary file: %s\n", path)
		return "", nil
	}
	
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}
	
	return string(data), nil
}

func (r *InputReader) readDirectory() (string, error) {
	var files []string
	var content strings.Builder
	
	err := filepath.WalkDir(r.source, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() {
			return nil
		}
		
		// Skip hidden files and directories
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		
		// Skip binary files
		if isBinary(path) {
			fmt.Fprintf(os.Stderr, "Warning: Skipping binary file: %s\n", path)
			return nil
		}
		
		files = append(files, path)
		return nil
	})
	
	if err != nil {
		return "", fmt.Errorf("failed to walk directory: %w", err)
	}
	
	// Sort files for consistent output
	sort.Strings(files)
	
	for i, file := range files {
		if i > 0 {
			content.WriteString("\n\n")
		}
		
		// Write file separator
		relPath, err := filepath.Rel(r.source, file)
		if err != nil {
			relPath = file
		}
		content.WriteString(fmt.Sprintf("=== %s ===\n", relPath))
		
		// Read and append file content
		fileContent, err := r.readFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to read file %s: %v\n", file, err)
			continue
		}
		
		if fileContent != "" {
			content.WriteString(fileContent)
		}
	}
	
	return content.String(), nil
}

var binaryExtensions = map[string]bool{
	".pdf":  true,
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".gif":  true,
	".bmp":  true,
	".tiff": true,
	".webp": true,
	".ico":  true,
	".svg":  false, // SVG is text-based
	".zip":  true,
	".tar":  true,
	".gz":   true,
	".rar":  true,
	".7z":   true,
	".exe":  true,
	".dll":  true,
	".so":   true,
	".dylib": true,
	".bin":  true,
	".dat":  true,
	".db":   true,
	".sqlite": true,
	".mp3":  true,
	".wav":  true,
	".mp4":  true,
	".avi":  true,
	".mov":  true,
	".mkv":  true,
	".webm": true,
	".woff": true,
	".woff2": true,
	".ttf":  true,
	".otf":  true,
	".eot":  true,
}

func isBinary(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	
	// Check known extensions first
	if isBin, exists := binaryExtensions[ext]; exists {
		return isBin
	}
	
	// Use MIME type detection for unknown extensions
	mimeType := mime.TypeByExtension(ext)
	if mimeType != "" {
		// Consider non-text MIME types as binary
		return !strings.HasPrefix(mimeType, "text/") &&
		       !strings.HasPrefix(mimeType, "application/json") &&
		       !strings.HasPrefix(mimeType, "application/xml") &&
		       !strings.HasPrefix(mimeType, "application/javascript") &&
		       !strings.HasPrefix(mimeType, "application/x-yaml")
	}
	
	// If we can't determine from extension/MIME, try reading a small sample
	return isBinaryByContent(path)
}

func isBinaryByContent(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false // Assume text if we can't read
	}
	defer file.Close()
	
	// Read first 512 bytes to check for null bytes
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false // Assume text if we can't read
	}
	
	// Check for null bytes, which indicate binary content
	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return true
		}
	}
	
	return false
}