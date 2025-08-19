package chunking

import (
	"strings"
	"unicode/utf8"
)

// TokenCounter provides token counting functionality
type TokenCounter struct {
	// Simple approximation: ~4 characters per token for most text
	// This is a rough estimate and could be improved with proper tokenization
	charsPerToken float64
}

// NewTokenCounter creates a new token counter
func NewTokenCounter() *TokenCounter {
	return &TokenCounter{
		charsPerToken: 4.0, // GPT-style approximation
	}
}

// CountTokens estimates the token count for the given text
func (tc *TokenCounter) CountTokens(text string) int {
	if text == "" {
		return 0
	}
	
	charCount := utf8.RuneCountInString(text)
	tokenEstimate := int(float64(charCount) / tc.charsPerToken)
	
	// Add some buffer for special tokens, formatting, etc.
	return int(float64(tokenEstimate) * 1.1)
}

// Chunker splits text into chunks based on token limits and semantic boundaries
type Chunker struct {
	tokenCounter *TokenCounter
	maxTokens    int
}

// NewChunker creates a new text chunker
func NewChunker(maxTokens int) *Chunker {
	return &Chunker{
		tokenCounter: NewTokenCounter(),
		maxTokens:    maxTokens,
	}
}

// ChunkText splits text into chunks, preferring semantic boundaries
func (c *Chunker) ChunkText(text string) ([]string, error) {
	totalTokens := c.tokenCounter.CountTokens(text)
	if totalTokens <= c.maxTokens {
		return []string{text}, nil
	}
	
	var chunks []string
	remaining := text
	
	for len(remaining) > 0 && c.tokenCounter.CountTokens(remaining) > c.maxTokens {
		
		chunk, rest := c.findOptimalChunk(remaining)
		if chunk == "" {
			// Fallback: force split if no good boundary found
			chunk, rest = c.forceSplit(remaining)
		}
		chunks = append(chunks, chunk)
		remaining = rest
	}
	
	// Add remaining text if any
	if len(strings.TrimSpace(remaining)) > 0 {
		chunks = append(chunks, remaining)
	}
	
	return chunks, nil
}

// findOptimalChunk finds the best chunk that fits within token limits
func (c *Chunker) findOptimalChunk(text string) (chunk, remaining string) {
	if c.tokenCounter.CountTokens(text) <= c.maxTokens {
		return text, ""
	}
	
	// Try different semantic boundaries in order of preference
	boundaries := []string{
		"\n\n\n",    // Multiple blank lines
		"\n\n",      // Paragraph breaks
		"\n",        // Line breaks
		". ",        // Sentence endings
		"? ",        // Question endings
		"! ",        // Exclamation endings
		", ",        // Comma breaks
		" ",         // Word boundaries
	}
	
	bestChunk := ""
	bestRemaining := text
	
	for _, boundary := range boundaries {
		if chunk, rest := c.tryBoundary(text, boundary); chunk != "" {
			if len(chunk) > len(bestChunk) {
				bestChunk = chunk
				bestRemaining = rest
			}
		}
	}
	
	return bestChunk, bestRemaining
}

// tryBoundary attempts to split text at the given boundary within token limits
func (c *Chunker) tryBoundary(text, boundary string) (chunk, remaining string) {
	parts := strings.Split(text, boundary)
	if len(parts) <= 1 {
		return "", text
	}
	
	var current strings.Builder
	var used int
	
	for i, part := range parts {
		candidate := current.String()
		if i > 0 {
			candidate += boundary
		}
		candidate += part
		
		if c.tokenCounter.CountTokens(candidate) > c.maxTokens {
			if used == 0 {
				// First part is too big, return empty to try next boundary
				return "", text
			}
			// Return what we have so far
			remaining := strings.Join(parts[used:], boundary)
			return current.String(), remaining
		}
		
		if i > 0 {
			current.WriteString(boundary)
		}
		current.WriteString(part)
		used = i + 1
	}
	
	// All parts fit
	return current.String(), ""
}

// forceSplit performs a hard split when no good boundary is found
func (c *Chunker) forceSplit(text string) (chunk, remaining string) {
	// Estimate character limit based on token limit
	charLimit := int(float64(c.maxTokens) * c.tokenCounter.charsPerToken * 0.8) // Buffer
	
	if len(text) <= charLimit {
		return text, ""
	}
	
	// Find the last word boundary before the limit
	cutPoint := charLimit
	for cutPoint > 0 && cutPoint < len(text) {
		if text[cutPoint] == ' ' || text[cutPoint] == '\n' || text[cutPoint] == '\t' {
			break
		}
		cutPoint--
	}
	
	if cutPoint == 0 {
		cutPoint = charLimit // No word boundary found, hard cut
	}
	
	return text[:cutPoint], strings.TrimSpace(text[cutPoint:])
}

// ValidateBoundary checks if a boundary is reasonable for chunking
func (c *Chunker) ValidateBoundary(text, boundary string) bool {
	// Simple validation: boundary should appear in text and create meaningful splits
	parts := strings.Split(text, boundary)
	return len(parts) > 1 && len(parts) < len(text)/10 // Reasonable number of parts
}

// GetEstimatedChunkCount returns an estimate of how many chunks the text will create
func (c *Chunker) GetEstimatedChunkCount(text string) int {
	totalTokens := c.tokenCounter.CountTokens(text)
	if totalTokens <= c.maxTokens {
		return 1
	}
	return (totalTokens + c.maxTokens - 1) / c.maxTokens // Ceiling division
}