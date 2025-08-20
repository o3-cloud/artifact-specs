package chunking

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/o3-cloud/artifact-specs/cli/internal/llm"
	"github.com/o3-cloud/artifact-specs/cli/internal/logging"
	"github.com/o3-cloud/artifact-specs/cli/internal/specs"
)

// MergeStrategy represents different approaches to merging chunk results
type MergeStrategy string

const (
	StrategyIncremental    MergeStrategy = "incremental"
	StrategyTwoPass       MergeStrategy = "two-pass"
	StrategyTemplateDriven MergeStrategy = "template-driven"
)

// MergeOptions contains configuration for merging chunks
type MergeOptions struct {
	Strategy     MergeStrategy
	Instructions string
	MaxRetries   int
	ShowStats    bool
}

// ChunkResult represents the result of processing a single chunk
type ChunkResult struct {
	ChunkIndex int
	Content    string
	JSONData   []byte
	Stats      *llm.CompletionResponse
	Error      error
}

// Merger handles merging results from multiple chunks
type Merger struct {
	spec   *specs.Spec
	client interface {
		Complete(context.Context, string, llm.CompletionOptions) (*llm.CompletionResponse, error)
	}
	options MergeOptions
}

// NewMerger creates a new chunk merger
func NewMerger(spec *specs.Spec, client interface {
	Complete(context.Context, string, llm.CompletionOptions) (*llm.CompletionResponse, error)
}, options MergeOptions) *Merger {
	return &Merger{
		spec:    spec,
		client:  client,
		options: options,
	}
}

// ProcessChunks processes multiple text chunks and merges the results
func (m *Merger) ProcessChunks(ctx context.Context, chunks []string) (*ChunkResult, error) {
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no chunks to process")
	}

	if len(chunks) == 1 {
		// Single chunk, process normally
		return m.processSingleChunk(ctx, chunks[0], 0)
	}

	switch m.options.Strategy {
	case StrategyIncremental:
		return m.processIncremental(ctx, chunks)
	case StrategyTwoPass:
		return m.processTwoPass(ctx, chunks)
	case StrategyTemplateDriven:
		return m.processTemplateDriven(ctx, chunks)
	default:
		return nil, fmt.Errorf("unknown merge strategy: %s", m.options.Strategy)
	}
}

// processIncremental processes chunks one by one, merging incrementally
func (m *Merger) processIncremental(ctx context.Context, chunks []string) (*ChunkResult, error) {
	logging.Info("Starting incremental processing", map[string]interface{}{
		"total_chunks": len(chunks),
		"strategy":     "incremental",
	})
	
	// Process first chunk
	logging.Info("Processing chunk", map[string]interface{}{
		"chunk_index": 0,
		"total":       len(chunks),
		"progress":    "1/" + fmt.Sprintf("%d", len(chunks)),
	})
	
	result, err := m.processSingleChunk(ctx, chunks[0], 0)
	if err != nil {
		return nil, fmt.Errorf("failed to process first chunk: %w", err)
	}
	
	logging.Info("First chunk processed successfully", map[string]interface{}{
		"chunk_index": 0,
	})

	// Process remaining chunks, merging with previous result
	for i := 1; i < len(chunks); i++ {
		logging.Info("Processing and merging chunk", map[string]interface{}{
			"chunk_index": i,
			"total":       len(chunks),
			"progress":    fmt.Sprintf("%d/%d", i+1, len(chunks)),
		})
		
		merged, err := m.mergeWithPrevious(ctx, result.JSONData, chunks[i], i)
		if err != nil {
			return nil, fmt.Errorf("failed to merge chunk %d: %w", i, err)
		}
		result = merged
		
		logging.Info("Chunk merged successfully", map[string]interface{}{
			"chunk_index": i,
			"progress":    fmt.Sprintf("%d/%d", i+1, len(chunks)),
		})
	}

	logging.Info("Incremental processing completed", map[string]interface{}{
		"total_chunks": len(chunks),
	})
	
	return result, nil
}

// processTwoPass processes all chunks individually, then merges all results
func (m *Merger) processTwoPass(ctx context.Context, chunks []string) (*ChunkResult, error) {
	logging.Info("Starting two-pass processing", map[string]interface{}{
		"total_chunks": len(chunks),
		"strategy":     "two-pass",
	})
	
	// Phase 1: Process all chunks independently
	logging.Info("Phase 1: Processing all chunks independently", map[string]interface{}{
		"phase": 1,
	})
	
	var chunkResults []*ChunkResult
	for i, chunk := range chunks {
		logging.Info("Processing chunk", map[string]interface{}{
			"chunk_index": i,
			"total":       len(chunks),
			"progress":    fmt.Sprintf("%d/%d", i+1, len(chunks)),
			"phase":       1,
		})
		
		result, err := m.processSingleChunk(ctx, chunk, i)
		if err != nil {
			return nil, fmt.Errorf("failed to process chunk %d: %w", i, err)
		}
		chunkResults = append(chunkResults, result)
		
		logging.Info("Chunk processed successfully", map[string]interface{}{
			"chunk_index": i,
			"phase":       1,
		})
	}

	// Phase 2: Merge all results
	logging.Info("Phase 2: Merging all results", map[string]interface{}{
		"phase":        2,
		"chunk_count":  len(chunkResults),
	})
	
	result, err := m.mergeAllResults(ctx, chunkResults)
	if err != nil {
		return nil, err
	}
	
	logging.Info("Two-pass processing completed", map[string]interface{}{
		"total_chunks": len(chunks),
	})
	
	return result, nil
}

// processTemplateDriven uses schema knowledge to intelligently merge chunks
func (m *Merger) processTemplateDriven(ctx context.Context, chunks []string) (*ChunkResult, error) {
	logging.Info("Starting template-driven processing", map[string]interface{}{
		"total_chunks": len(chunks),
		"strategy":     "template-driven",
	})
	
	// Process all chunks
	var chunkResults []*ChunkResult
	for i, chunk := range chunks {
		logging.Info("Processing chunk", map[string]interface{}{
			"chunk_index": i,
			"total":       len(chunks),
			"progress":    fmt.Sprintf("%d/%d", i+1, len(chunks)),
		})
		
		result, err := m.processSingleChunk(ctx, chunk, i)
		if err != nil {
			return nil, fmt.Errorf("failed to process chunk %d: %w", i, err)
		}
		chunkResults = append(chunkResults, result)
		
		logging.Info("Chunk processed successfully", map[string]interface{}{
			"chunk_index": i,
		})
	}

	// Use schema-driven merging
	logging.Info("Starting schema-driven merging", map[string]interface{}{
		"chunk_count": len(chunkResults),
	})
	
	result, err := m.mergeByTemplate(ctx, chunkResults)
	if err != nil {
		return nil, err
	}
	
	logging.Info("Template-driven processing completed", map[string]interface{}{
		"total_chunks": len(chunks),
	})
	
	return result, nil
}

// processSingleChunk extracts data from a single chunk
func (m *Merger) processSingleChunk(ctx context.Context, chunk string, index int) (*ChunkResult, error) {
	prompt := m.createExtractionPrompt(chunk)
	
	logging.Debug("Generated chunk extraction prompt", map[string]interface{}{
		"chunk_index": index,
		"prompt":      prompt,
	})
	
	response, err := m.client.Complete(ctx, prompt, llm.CompletionOptions{ForceJSON: true})
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	return &ChunkResult{
		ChunkIndex: index,
		Content:    chunk,
		JSONData:   []byte(response.Content),
		Stats:      response,
		Error:      nil,
	}, nil
}

// mergeWithPrevious merges a new chunk with the previous accumulated result
func (m *Merger) mergeWithPrevious(ctx context.Context, previousJSON []byte, newChunk string, chunkIndex int) (*ChunkResult, error) {
	prompt := m.createMergePrompt(previousJSON, newChunk)
	
	logging.Debug("Generated chunk merge prompt", map[string]interface{}{
		"chunk_index": chunkIndex,
		"prompt":      prompt,
	})
	
	response, err := m.client.Complete(ctx, prompt, llm.CompletionOptions{ForceJSON: true})
	if err != nil {
		return nil, fmt.Errorf("merge failed: %w", err)
	}

	return &ChunkResult{
		ChunkIndex: chunkIndex,
		Content:    newChunk,
		JSONData:   []byte(response.Content),
		Stats:      response,
		Error:      nil,
	}, nil
}

// mergeAllResults merges multiple chunk results into a single result
func (m *Merger) mergeAllResults(ctx context.Context, results []*ChunkResult) (*ChunkResult, error) {
	if len(results) == 1 {
		return results[0], nil
	}

	var jsonResults []json.RawMessage
	for _, result := range results {
		jsonResults = append(jsonResults, json.RawMessage(result.JSONData))
	}

	prompt := m.createConsolidationPrompt(jsonResults)
	
	logging.Debug("Generated consolidation prompt", map[string]interface{}{
		"chunk_count": len(results),
		"prompt":      prompt,
	})
	response, err := m.client.Complete(ctx, prompt, llm.CompletionOptions{ForceJSON: true})
	if err != nil {
		return nil, fmt.Errorf("consolidation failed: %w", err)
	}

	return &ChunkResult{
		ChunkIndex: -1, // Combined result
		Content:    "consolidated",
		JSONData:   []byte(response.Content),
		Stats:      response,
		Error:      nil,
	}, nil
}

// mergeByTemplate uses the JSON schema to guide merging
func (m *Merger) mergeByTemplate(ctx context.Context, results []*ChunkResult) (*ChunkResult, error) {
	// For now, use the same logic as mergeAllResults but with schema guidance
	// This could be enhanced to analyze the schema and merge fields intelligently
	return m.mergeAllResults(ctx, results)
}

// createExtractionPrompt creates a prompt for extracting from a single chunk
func (m *Merger) createExtractionPrompt(chunk string) string {
	schemaTitle := m.spec.Title
	if schemaTitle == "" {
		schemaTitle = m.spec.Slug
	}

	return fmt.Sprintf(`Extract structured data from the following input according to the "%s" specification.

Instructions:
- Extract only information that is explicitly present in this chunk
- Do not invent or infer information not directly stated
- Leave fields empty/null if the information is not available in this chunk
- Follow the JSON schema structure exactly
- This is part of a larger document, so partial information is expected

Input:
%s

JSON Schema Reference:
%s

Provide the extracted data as valid JSON:`, 
		schemaTitle, 
		chunk, 
		string(m.spec.Schema))
}

// createMergePrompt creates a prompt for merging new chunk data with previous results
func (m *Merger) createMergePrompt(previousJSON []byte, newChunk string) string {
	instructions := m.options.Instructions
	if instructions == "" {
		instructions = `Merge the new data with the existing data:
- Combine arrays by appending new items
- Update object fields with new information
- Preserve existing data when not conflicted
- Use new data to fill in missing fields
- When data conflicts, prefer the new data`
	}

	schemaTitle := m.spec.Title
	if schemaTitle == "" {
		schemaTitle = m.spec.Slug
	}

	return fmt.Sprintf(`You have existing extracted data and a new chunk of text to process. Merge the new information with the existing data according to the "%s" specification.

%s

Existing extracted data:
%s

New chunk to merge:
%s

JSON Schema Reference:
%s

Provide the merged result as valid JSON that follows the schema:`, 
		schemaTitle,
		instructions,
		string(previousJSON), 
		newChunk,
		string(m.spec.Schema))
}

// createConsolidationPrompt creates a prompt for consolidating multiple results
func (m *Merger) createConsolidationPrompt(jsonResults []json.RawMessage) string {
	instructions := m.options.Instructions
	if instructions == "" {
		instructions = `Consolidate all the partial results into a single comprehensive result:
- Merge arrays by combining all items
- Merge objects by combining all fields
- Remove duplicates where appropriate
- Ensure the final result follows the schema structure`
	}

	var resultsStr strings.Builder
	for i, result := range jsonResults {
		resultsStr.WriteString(fmt.Sprintf("Result %d:\n%s\n\n", i+1, string(result)))
	}

	return fmt.Sprintf(`Consolidate the following partial extraction results into a single comprehensive result.

%s

Partial results to consolidate:
%s

JSON Schema Reference:
%s

Provide the consolidated result as valid JSON:`,
		instructions,
		resultsStr.String(),
		string(m.spec.Schema))
}