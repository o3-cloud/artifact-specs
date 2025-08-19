package render

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "strings"

	"github.com/o3-cloud/artifact-specs/cli/internal/chunking"
	"github.com/o3-cloud/artifact-specs/cli/internal/config"
	"github.com/o3-cloud/artifact-specs/cli/internal/io"
	"github.com/o3-cloud/artifact-specs/cli/internal/llm"
	"github.com/o3-cloud/artifact-specs/cli/internal/logging"
	"github.com/o3-cloud/artifact-specs/cli/internal/specs"
	"github.com/o3-cloud/artifact-specs/cli/internal/validate"
)

type Renderer struct {
	spec      *specs.Spec
	validator *validate.Validator
	client    llmClient
}

type RenderOptions struct {
	SaveJSON          bool
	OutputPath        string
	ShowStats         bool
	StreamOutput      bool
	NoValidate        bool
	ChunkSize         int
	MergeStrategy     string
	MergeInstructions string
}

type RenderResult struct {
	JSON       []byte
	Markdown   string
	JSONPath   string
	Stats      *llm.CompletionResponse
}

func NewRenderer(spec *specs.Spec, client llmClient) (*Renderer, error) {
	validator, err := validate.NewValidator(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to create validator: %w", err)
	}
	
	return &Renderer{
		spec:      spec,
		validator: validator,
		client:    client,
	}, nil
}

func (r *Renderer) Render(ctx context.Context, input string, options RenderOptions) (*RenderResult, error) {
	if strings.TrimSpace(input) == "" {
		return nil, fmt.Errorf("input is empty")
	}

    // Step 1: Extract structured JSON
    logging.Info("Step 1: Extracting structured data...")
	
	// Check if chunking is needed
	tokenCounter := chunking.NewTokenCounter()
	inputTokens := tokenCounter.CountTokens(input)
	chunkSize := options.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 20000 // Default chunk size
	}

	var extractedData []byte
	var extractResponse *llm.CompletionResponse
	var err error
	
	if inputTokens <= chunkSize {
		logging.Debug("Input fits in single chunk, processing normally", map[string]interface{}{
			"tokens": inputTokens,
			"limit":  chunkSize,
		})

		// Single chunk processing
		extractPrompt, err := r.createExtractionPrompt(input)
		if err != nil {
			return nil, fmt.Errorf("failed to create extraction prompt: %w", err)
		}

		if options.NoValidate {
			extractResponse, err = r.client.Complete(ctx, extractPrompt, llm.CompletionOptions{ForceJSON: true})
			if err != nil {
				return nil, fmt.Errorf("extraction failed: %w", err)
			}
			extractedData = []byte(extractResponse.Content)
		} else {
			var validationResult *validate.ValidationResult
			extractedData, validationResult, err = r.validator.ValidateAndRetry(ctx, r.client, extractPrompt, 2)
			if err != nil {
				return nil, fmt.Errorf("extraction failed: %w", err)
			}

            if !validationResult.Valid {
                logging.Warn("Final result failed validation", map[string]interface{}{"errors": validationResult.FormatErrors()})
            }
		}
	} else {
		logging.Info("Input exceeds chunk limit, using chunked processing", map[string]interface{}{
			"tokens":   inputTokens,
			"limit":    chunkSize,
			"strategy": options.MergeStrategy,
		})

		// Chunked processing
		chunker := chunking.NewChunker(chunkSize)
		chunks, err := chunker.ChunkText(input)
		if err != nil {
			return nil, fmt.Errorf("failed to chunk input: %w", err)
		}

		// Log chunking details
		logging.Info("Input successfully chunked", map[string]interface{}{
			"total_tokens":   inputTokens,
			"chunk_limit":    chunkSize,
			"chunk_count":    len(chunks),
			"avg_chunk_size": inputTokens / len(chunks),
		})

		// Parse merge strategy
		strategy := options.MergeStrategy
		if strategy == "" {
			strategy = "incremental"
		}
		
		var mergeStrategy chunking.MergeStrategy
		switch strategy {
		case "incremental":
			mergeStrategy = chunking.StrategyIncremental
		case "two-pass":
			mergeStrategy = chunking.StrategyTwoPass
		case "template-driven":
			mergeStrategy = chunking.StrategyTemplateDriven
		default:
			return nil, fmt.Errorf("unknown merge strategy: %s", strategy)
		}

		// Create merger
		merger := chunking.NewMerger(r.spec, r.client, chunking.MergeOptions{
			Strategy:     mergeStrategy,
			Instructions: options.MergeInstructions,
			MaxRetries:   2,
			ShowStats:    options.ShowStats,
		})

		// Process chunks
		result, err := merger.ProcessChunks(ctx, chunks)
		if err != nil {
			return nil, fmt.Errorf("failed to process chunks: %w", err)
		}

		extractedData = result.JSONData
		extractResponse = result.Stats

		// Validate final result if requested
            if !options.NoValidate {
                validationResult := r.validator.Validate(extractedData)
                if !validationResult.Valid {
                    logging.Warn("Final result failed validation", map[string]interface{}{"errors": validationResult.FormatErrors()})
                }
            }
	}
	
    logging.Info("Step 1: ✓ Extraction completed")
	
    // Step 2: Verbalize to Markdown
    logging.Info("Step 2: Generating Markdown...")
	
	verbalizePrompt, err := r.createVerbalizationPrompt(extractedData)
	if err != nil {
		return nil, fmt.Errorf("failed to create verbalization prompt: %w", err)
	}
	
	var markdownResponse *llm.CompletionResponse
	var markdown string
	
	if options.StreamOutput {
		var callback llm.StreamCallback
		if !options.ShowStats {
                // Stream directly to stdout if not showing stats
                callback = func(content string) error {
                    fmt.Print(content)
                    return nil
                }
		} else {
			// Accumulate for later output if showing stats
			callback = func(content string) error {
				markdown += content
				return nil
			}
		}
		
		markdownResponse, err = r.client.CompleteStream(ctx, verbalizePrompt, callback, llm.CompletionOptions{})
	} else {
		markdownResponse, err = r.client.Complete(ctx, verbalizePrompt, llm.CompletionOptions{})
		markdown = markdownResponse.Content
	}
	
	if err != nil {
		return nil, fmt.Errorf("verbalization failed: %w", err)
	}
	
    logging.Info("Step 2: ✓ Markdown generation completed")
	
	// Save outputs
	result := &RenderResult{
		JSON:     extractedData,
		Markdown: markdown,
		Stats:    markdownResponse,
	}
	
	// Save JSON if requested
	if options.SaveJSON {
		jsonPath, err := r.saveJSON(extractedData, options.OutputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to save JSON: %w", err)
		}
		result.JSONPath = jsonPath
	}
	
	// Write markdown output
	if options.OutputPath != "" && !options.StreamOutput {
		writer := io.NewOutputWriter(options.OutputPath)
		if err := writer.WriteOutput(markdown); err != nil {
			return nil, fmt.Errorf("failed to write markdown output: %w", err)
		}
	} else if !options.StreamOutput {
		// Write to stdout if not already streamed
		fmt.Print(markdown)
	}
	
	return result, nil
}

func (r *Renderer) createExtractionPrompt(input string) (string, error) {
	// Get schema title or use spec slug
	schemaTitle := r.spec.Title
	if schemaTitle == "" {
		schemaTitle = r.spec.Slug
	}
	
	// Use config-based template rendering
	return config.RenderExtractionPrompt(schemaTitle, input, string(r.spec.Schema))
}

func (r *Renderer) createVerbalizationPrompt(jsonData []byte) (string, error) {
	// Pretty-print the JSON for better readability in the prompt
	var formatted interface{}
	json.Unmarshal(jsonData, &formatted)
	prettyJSON, _ := json.MarshalIndent(formatted, "", "  ")
	
	return config.RenderVerbalizationPrompt(string(prettyJSON))
}

func (r *Renderer) saveJSON(jsonData []byte, outputPath string) (string, error) {
	var jsonPath string
	
	if outputPath != "" {
		// Generate JSON path based on output path
		jsonPath = io.GenerateOutputPath(outputPath, ".json")
	} else {
		// Use default path
		jsonPath = "./out.json"
	}
	
	// Pretty-print the JSON
	var formatted interface{}
	if err := json.Unmarshal(jsonData, &formatted); err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON for formatting: %w", err)
	}
	
	prettyJSON, err := json.MarshalIndent(formatted, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}
	
	// Ensure directory exists
	dir := filepath.Dir(jsonPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory: %w", err)
		}
	}
	
	// Write JSON file
	if err := os.WriteFile(jsonPath, prettyJSON, 0644); err != nil {
		return "", fmt.Errorf("failed to write JSON file: %w", err)
	}
	
    logging.Info("Intermediate JSON saved", map[string]interface{}{"path": jsonPath})
    return jsonPath, nil
}

// Interface for LLM client to support both real and mock clients
type llmClient interface {
	Complete(ctx context.Context, userPrompt string, options llm.CompletionOptions) (*llm.CompletionResponse, error)
	CompleteStream(ctx context.Context, userPrompt string, callback llm.StreamCallback, options llm.CompletionOptions) (*llm.CompletionResponse, error)
}
