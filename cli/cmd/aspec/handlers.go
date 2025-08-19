package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/o3-cloud/artifact-specs/cli/internal/chunking"
	"github.com/o3-cloud/artifact-specs/cli/internal/config"
	"github.com/o3-cloud/artifact-specs/cli/internal/io"
	"github.com/o3-cloud/artifact-specs/cli/internal/llm"
	"github.com/o3-cloud/artifact-specs/cli/internal/logging"
	"github.com/o3-cloud/artifact-specs/cli/internal/render"
	"github.com/o3-cloud/artifact-specs/cli/internal/specs"
	"github.com/o3-cloud/artifact-specs/cli/internal/validate"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func runListCommand(cmd *cobra.Command, args []string) error {
	manager, err := specs.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create specs manager: %w", err)
	}
	
	specType, _ := cmd.Flags().GetString("type")
	search, _ := cmd.Flags().GetString("search")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	yamlOutput, _ := cmd.Flags().GetBool("yaml")
	
	var st specs.SpecType
	if specType == "extractors" {
		st = specs.Extractors
	} else {
		st = specs.Artifacts
	}
	
	specList, err := manager.ListSpecs(st, search)
	if err != nil {
		return fmt.Errorf("failed to list specs: %w", err)
	}
	
	if jsonOutput {
		data, err := json.MarshalIndent(specList, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}
	
	if yamlOutput {
		data, err := yaml.Marshal(specList)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}
		fmt.Print(string(data))
		return nil
	}
	
	// Table format
	if len(specList) == 0 {
		fmt.Printf("No %s found", specType)
		if search != "" {
			fmt.Printf(" matching '%s'", search)
		}
		fmt.Println()
		return nil
	}
	
	fmt.Printf("%-20s %-40s %-10s %-10s\n", "SLUG", "TITLE", "TYPE", "REF")
	fmt.Println(strings.Repeat("-", 80))
	
	for _, spec := range specList {
		title := spec.Title
		if len(title) > 40 {
			title = title[:37] + "..."
		}
		fmt.Printf("%-20s %-40s %-10s %-10s\n", spec.Slug, title, spec.Type, spec.Ref)
	}
	
	return nil
}

func runShowCommand(cmd *cobra.Command, args []string) error {
	specIdentifier, _ := cmd.Flags().GetString("spec")
	specType, _ := cmd.Flags().GetString("type")
	specURL, _ := cmd.Flags().GetString("spec-url")
	specPath, _ := cmd.Flags().GetString("spec-path")
	format, _ := cmd.Flags().GetString("format")
	
	if specIdentifier == "" && specURL == "" && specPath == "" {
		return fmt.Errorf("must specify --spec, --spec-url, or --spec-path")
	}
	
	manager, err := specs.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create specs manager: %w", err)
	}
	
	var spec *specs.Spec
	
	if specURL != "" {
		spec, err = manager.GetSpecByURL(specURL)
	} else if specPath != "" {
		spec, err = manager.GetSpecByPath(specPath)
	} else {
		var st specs.SpecType
		if specType == "extractors" {
			st = specs.Extractors
		} else {
			st = specs.Artifacts
		}
		spec, err = manager.GetSpec(st, specIdentifier)
	}
	
	if err != nil {
		return fmt.Errorf("failed to get spec: %w", err)
	}
	
	if format == "yaml" {
		var schemaDoc interface{}
		if err := json.Unmarshal(spec.Schema, &schemaDoc); err != nil {
			return fmt.Errorf("failed to parse schema: %w", err)
		}
		
		yamlData, err := yaml.Marshal(schemaDoc)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}
		
		fmt.Print(string(yamlData))
	} else {
		// JSON format with pretty printing
		var schemaDoc interface{}
		if err := json.Unmarshal(spec.Schema, &schemaDoc); err != nil {
			return fmt.Errorf("failed to parse schema: %w", err)
		}
		
		jsonData, err := json.MarshalIndent(schemaDoc, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format JSON: %w", err)
		}
		
		fmt.Println(string(jsonData))
	}
	
	return nil
}

func runPromptCommand(cmd *cobra.Command, args []string) error {
	spec, err := getSpecFromFlags(cmd)
	if err != nil {
		return err
	}
	
	model, _ := cmd.Flags().GetString("model")
	outputPath, _ := cmd.Flags().GetString("out")
	streaming, _ := cmd.Flags().GetBool("stream")
	noStreaming, _ := cmd.Flags().GetBool("no-stream")
	
	if noStreaming {
		streaming = false
	}
	
	client, err := llm.NewClient(model)
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}
	
	// Create prompt generation prompt
	promptText := fmt.Sprintf("Turn this JSON schema spec into a plain English prompt:\n\n%s", string(spec.Schema))
	
	ctx := context.Background()
	
	var response *llm.CompletionResponse
	
	if streaming {
		var callback llm.StreamCallback
		if outputPath == "" {
			callback = func(content string) error {
				fmt.Print(content)
				return nil
			}
		}
		
		response, err = client.CompleteStream(ctx, promptText, callback, llm.CompletionOptions{})
	} else {
		response, err = client.Complete(ctx, promptText, llm.CompletionOptions{})
	}
	
	if err != nil {
		return fmt.Errorf("failed to generate prompt: %w", err)
	}
	
	if outputPath != "" {
		writer := io.NewOutputWriter(outputPath)
		if err := writer.WriteOutput(response.Content); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
	} else if !streaming {
		fmt.Print(response.Content)
	}
	
	return nil
}

func runExtractCommand(cmd *cobra.Command, args []string) error {
	spec, err := getSpecFromFlags(cmd)
	if err != nil {
		return err
	}
	
	inputPath, _ := cmd.Flags().GetString("in")
	outputPath, _ := cmd.Flags().GetString("out")
	model, _ := cmd.Flags().GetString("model")
	maxRetries, _ := cmd.Flags().GetInt("max-retries")
	noValidate, _ := cmd.Flags().GetBool("no-validate")
	enableValidate, _ := cmd.Flags().GetBool("validate")
	compact, _ := cmd.Flags().GetBool("compact")
	showStats, _ := cmd.Flags().GetBool("stats")
	chunkSize, _ := cmd.Flags().GetInt("chunk-size")
	mergeStrategy, _ := cmd.Flags().GetString("merge-strategy")
	mergeInstructions, _ := cmd.Flags().GetString("merge-instructions")
	
	// If --validate is explicitly set, override --no-validate
	if enableValidate {
		noValidate = false
	}
	
	// Read input
	reader := io.NewInputReader(inputPath)
	input, err := reader.ReadInput()
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	
	if strings.TrimSpace(input) == "" {
		return fmt.Errorf("input is empty")
	}
	
	// Create LLM client
	client, err := llm.NewClient(model)
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}
	
	ctx := context.Background()
	
	// Check if chunking is needed
	tokenCounter := chunking.NewTokenCounter()
	inputTokens := tokenCounter.CountTokens(input)

	var jsonData []byte
	var response *llm.CompletionResponse
	
	if inputTokens <= chunkSize {
		logging.Debug("Input fits in single chunk, processing normally", map[string]interface{}{
			"tokens": inputTokens,
			"limit":  chunkSize,
		})

		// Single chunk processing (existing logic)
		extractPrompt := createExtractionPrompt(spec, input)

		if noValidate {
			response, err = client.Complete(ctx, extractPrompt, llm.CompletionOptions{ForceJSON: true})
			if err != nil {
				return fmt.Errorf("extraction failed: %w", err)
			}
			jsonData = []byte(response.Content)
		} else {
			validator, err := validate.NewValidator(spec)
			if err != nil {
				return fmt.Errorf("failed to create validator: %w", err)
			}

			var validationResult *validate.ValidationResult
			jsonData, validationResult, err = validator.ValidateAndRetry(ctx, client, extractPrompt, maxRetries)
			if err != nil {
				return fmt.Errorf("extraction failed: %w", err)
			}

			if !validationResult.Valid {
				fmt.Fprintf(os.Stderr, "Warning: Final result failed validation: %s\n", validationResult.FormatErrors())
				os.Exit(3)
			}
		}
	} else {
		logging.Info("Input exceeds chunk limit, using chunked processing", map[string]interface{}{
			"tokens":   inputTokens,
			"limit":    chunkSize,
			"strategy": mergeStrategy,
		})

		// Chunked processing
		chunker := chunking.NewChunker(chunkSize)
		chunks, err := chunker.ChunkText(input)
		if err != nil {
			return fmt.Errorf("failed to chunk input: %w", err)
		}

		// Log chunking details
		logging.Info("Input successfully chunked", map[string]interface{}{
			"total_tokens":   inputTokens,
			"chunk_limit":    chunkSize,
			"chunk_count":    len(chunks),
			"avg_chunk_size": inputTokens / len(chunks),
		})

		// Log individual chunk sizes if verbose
		for i, chunk := range chunks {
			chunkTokens := tokenCounter.CountTokens(chunk)
			logging.Debug("Chunk details", map[string]interface{}{
				"chunk_index": i,
				"tokens":      chunkTokens,
				"characters":  len(chunk),
			})
		}

		// Parse merge strategy
		var strategy chunking.MergeStrategy
		switch mergeStrategy {
		case "incremental":
			strategy = chunking.StrategyIncremental
		case "two-pass":
			strategy = chunking.StrategyTwoPass
		case "template-driven":
			strategy = chunking.StrategyTemplateDriven
		default:
			return fmt.Errorf("unknown merge strategy: %s", mergeStrategy)
		}

		// Create merger
		merger := chunking.NewMerger(spec, client, chunking.MergeOptions{
			Strategy:     strategy,
			Instructions: mergeInstructions,
			MaxRetries:   maxRetries,
			ShowStats:    showStats,
		})
		
		// Process chunks
		result, err := merger.ProcessChunks(ctx, chunks)
		if err != nil {
			return fmt.Errorf("failed to process chunks: %w", err)
		}
		
		jsonData = result.JSONData
		response = result.Stats

		// Validate final result if requested
		if !noValidate {
			validator, err := validate.NewValidator(spec)
			if err != nil {
				return fmt.Errorf("failed to create validator: %w", err)
			}

			validationResult := validator.Validate(jsonData)
			if !validationResult.Valid {
				fmt.Fprintf(os.Stderr, "Warning: Final result failed validation: %s\n", validationResult.FormatErrors())
				os.Exit(3)
			}
		}
	}

	// Format output
	var outputData []byte
	if compact {
		outputData = jsonData
	} else {
		var formatted interface{}
		if err := json.Unmarshal(jsonData, &formatted); err != nil {
			return fmt.Errorf("failed to parse extracted JSON: %w", err)
		}
		
		outputData, err = json.MarshalIndent(formatted, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format JSON: %w", err)
		}
	}
	
	// Write output
	if outputPath != "" {
		writer := io.NewOutputWriter(outputPath)
		if err := writer.WriteOutput(string(outputData)); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
	} else {
		fmt.Println(string(outputData))
	}
	
	// Show stats if requested
	if showStats && response != nil {
		llm.PrintStats(response, os.Stderr)
	}
	
	return nil
}

func runRenderCommand(cmd *cobra.Command, args []string) error {
	spec, err := getSpecFromFlags(cmd)
	if err != nil {
		return err
	}
	
	inputPath, _ := cmd.Flags().GetString("in")
	outputPath, _ := cmd.Flags().GetString("out")
	model, _ := cmd.Flags().GetString("model")
	saveJSON, _ := cmd.Flags().GetBool("save-json")
	showStats, _ := cmd.Flags().GetBool("stats")
	streaming, _ := cmd.Flags().GetBool("stream")
	noStreaming, _ := cmd.Flags().GetBool("no-stream")
	noValidate, _ := cmd.Flags().GetBool("no-validate")
	enableValidate, _ := cmd.Flags().GetBool("validate")
	
	if noStreaming {
		streaming = false
	}
	
	// If --validate is explicitly set, override --no-validate
	if enableValidate {
		noValidate = false
	}
	
	// Read input
	reader := io.NewInputReader(inputPath)
	input, err := reader.ReadInput()
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	
	if strings.TrimSpace(input) == "" {
		return fmt.Errorf("input is empty")
	}
	
	// Create LLM client
	client, err := llm.NewClient(model)
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}
	
	// Create renderer
	renderer, err := render.NewRenderer(spec, client)
	if err != nil {
		return fmt.Errorf("failed to create renderer: %w", err)
	}
	
	// Render
	ctx := context.Background()
	result, err := renderer.Render(ctx, input, render.RenderOptions{
		SaveJSON:     saveJSON,
		OutputPath:   outputPath,
		ShowStats:    showStats,
		StreamOutput: streaming,
		NoValidate:   noValidate,
	})
	if err != nil {
		return fmt.Errorf("render failed: %w", err)
	}
	
	// Show stats if requested
	if showStats && result.Stats != nil {
		llm.PrintStats(result.Stats, os.Stderr)
	}
	
	return nil
}

func getSpecFromFlags(cmd *cobra.Command) (*specs.Spec, error) {
	specIdentifier, _ := cmd.Flags().GetString("spec")
	specType, _ := cmd.Flags().GetString("type")
	specURL, _ := cmd.Flags().GetString("spec-url")
	specPath, _ := cmd.Flags().GetString("spec-path")
	
	logging.Debug("Getting spec from flags", map[string]interface{}{
		"spec":      specIdentifier,
		"type":      specType,
		"spec_url":  specURL,
		"spec_path": specPath,
	})
	
	if specIdentifier == "" && specURL == "" && specPath == "" {
		return nil, fmt.Errorf("must specify --spec, --spec-url, or --spec-path")
	}
	
	manager, err := specs.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create specs manager: %w", err)
	}
	
	if specURL != "" {
		logging.Debug("Loading spec from URL", map[string]interface{}{
			"url": specURL,
		})
		return manager.GetSpecByURL(specURL)
	}
	
	if specPath != "" {
		logging.Debug("Loading spec from file path", map[string]interface{}{
			"path": specPath,
		})
		return manager.GetSpecByPath(specPath)
	}
	
	var st specs.SpecType
	if specType == "extractors" {
		st = specs.Extractors
	} else {
		st = specs.Artifacts
	}
	
	logging.Debug("Loading spec from cache", map[string]interface{}{
		"identifier": specIdentifier,
		"type":       st,
	})
	
	return manager.GetSpec(st, specIdentifier)
}

func createExtractionPrompt(spec *specs.Spec, input string) string {
	schemaTitle := spec.Title
	if schemaTitle == "" {
		schemaTitle = spec.Slug
	}
	
	return fmt.Sprintf(`Extract structured data from the following input according to the "%s" specification.

Instructions:
- Extract only information that is explicitly present in the input
- Do not invent or infer information not directly stated
- Leave fields empty/null if the information is not available
- Follow the JSON schema structure exactly
- Ensure all required fields are present

Input:
%s

JSON Schema Reference:
%s

Provide the extracted data as valid JSON:`, 
		schemaTitle, 
		input, 
		string(spec.Schema))
}

func runValidateCommand(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("must provide JSON file to validate")
	}
	
	jsonFile := args[0]
	
	// Read JSON file
	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		return fmt.Errorf("failed to read JSON file: %w", err)
	}
	
	// Get spec - can be from file schema or flags
	var spec *specs.Spec
	
	// Try to extract $schema from JSON first
	var jsonDoc map[string]interface{}
	if err := json.Unmarshal(jsonData, &jsonDoc); err != nil {
		return fmt.Errorf("invalid JSON file: %w", err)
	}
	
	if schemaURL, ok := jsonDoc["$schema"].(string); ok && schemaURL != "" {
		// Try to load from $schema reference
		logging.Info("Using $schema reference from JSON file", map[string]interface{}{
			"schema": schemaURL,
		})
		
		manager, err := specs.NewManager()
		if err != nil {
			return fmt.Errorf("failed to create specs manager: %w", err)
		}
		
		spec, err = manager.GetSpecByURL(schemaURL)
		if err != nil {
			logging.Warn("Failed to load schema from $schema reference, falling back to flags")
			spec, err = getSpecFromFlags(cmd)
			if err != nil {
				return fmt.Errorf("failed to get spec from flags: %w", err)
			}
		}
	} else {
		// Use spec from flags
		spec, err = getSpecFromFlags(cmd)
		if err != nil {
			return fmt.Errorf("failed to get spec: %w", err)
		}
	}
	
	// Validate
	validator, err := validate.NewValidator(spec)
	if err != nil {
		return fmt.Errorf("failed to create validator: %w", err)
	}
	
	result := validator.Validate(jsonData)
	
	if result.Valid {
		fmt.Printf("✓ JSON is valid according to %s schema\n", spec.Slug)
		return nil
	}
	
	fmt.Printf("✗ JSON validation failed:\n%s\n", result.FormatErrors())
	os.Exit(3)
	return nil
}

func runUpdateCommand(cmd *cobra.Command, args []string) error {
	ref, _ := cmd.Flags().GetString("ref")
	repoName, _ := cmd.Flags().GetString("repo")
	
	// Parse repo name
	parts := strings.Split(repoName, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository format, expected owner/name")
	}
	
	repo := specs.Repository{
		Owner: parts[0],
		Name:  parts[1],
		Ref:   ref,
	}
	
	manager, err := specs.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create specs manager: %w", err)
	}
	
	fmt.Printf("Updating specs from %s@%s...\n", repo.String(), ref)
	
	if err := manager.UpdateSpecs(repo); err != nil {
		return fmt.Errorf("failed to update specs: %w", err)
	}
	
	return nil
}

func runTestCommand(cmd *cobra.Command, args []string) error {
	specIdentifier, _ := cmd.Flags().GetString("spec")
	fixture, _ := cmd.Flags().GetString("fixture")
	expected, _ := cmd.Flags().GetString("expected")
	provider, _ := cmd.Flags().GetString("provider")
	mockFixture, _ := cmd.Flags().GetString("mock-fixture")
	
	if specIdentifier == "" {
		return fmt.Errorf("must specify --spec")
	}
	
	if fixture == "" {
		return fmt.Errorf("must specify --fixture")
	}
	
	manager, err := specs.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create specs manager: %w", err)
	}
	
	spec, err := manager.GetSpec(specs.Artifacts, specIdentifier)
	if err != nil {
		return fmt.Errorf("failed to get spec: %w", err)
	}
	
	// Read fixture input
	inputData, err := os.ReadFile(fixture)
	if err != nil {
		return fmt.Errorf("failed to read fixture input: %w", err)
	}
	
	var client interface {
		Complete(context.Context, string, llm.CompletionOptions) (*llm.CompletionResponse, error)
	}
	
	if provider == "mock" {
		mockClient := llm.NewMockClient("mock-model")
		if mockFixture != "" {
			if err := mockClient.LoadFixture(mockFixture); err != nil {
				return fmt.Errorf("failed to load mock fixture: %w", err)
			}
		}
		client = mockClient
	} else {
		realClient, err := llm.NewClient("")
		if err != nil {
			return fmt.Errorf("failed to create LLM client: %w", err)
		}
		client = realClient
	}
	
	// Run extraction
	extractPrompt := createExtractionPrompt(spec, string(inputData))
	
	ctx := context.Background()
	response, err := client.Complete(ctx, extractPrompt, llm.CompletionOptions{ForceJSON: true})
	if err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}
	
	// Validate result
	validator, err := validate.NewValidator(spec)
	if err != nil {
		return fmt.Errorf("failed to create validator: %w", err)
	}
	
	result := validator.Validate([]byte(response.Content))
	if !result.Valid {
		fmt.Printf("✗ Test failed: validation errors\n%s\n", result.FormatErrors())
		os.Exit(1)
	}
	
	// Compare with expected if provided
	if expected != "" {
		expectedData, err := os.ReadFile(expected)
		if err != nil {
			return fmt.Errorf("failed to read expected file: %w", err)
		}
		
		// Normalize JSON for comparison
		var actualJSON, expectedJSON interface{}
		json.Unmarshal([]byte(response.Content), &actualJSON)
		json.Unmarshal(expectedData, &expectedJSON)
		
		actualBytes, _ := json.Marshal(actualJSON)
		expectedBytes, _ := json.Marshal(expectedJSON)
		
		if string(actualBytes) != string(expectedBytes) {
			fmt.Println("✗ Test failed: output doesn't match expected")
			fmt.Println("Expected:")
			fmt.Println(string(expectedData))
			fmt.Println("Actual:")
			fmt.Println(response.Content)
			os.Exit(1)
		}
	}
	
	fmt.Printf("✓ Test passed for %s\n", spec.Slug)
	return nil
}

func runInitCommand(cmd *cobra.Command, args []string) error {
	if err := config.CreateDefaultConfig(); err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}
	
	fmt.Println("Configuration initialized successfully!")
	fmt.Println("Set your OpenRouter API key with: export OPENROUTER_API_KEY=your_key_here")
	
	return nil
}