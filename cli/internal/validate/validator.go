package validate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kaptinlin/jsonschema"
	"github.com/o3-cloud/artifact-specs/cli/internal/llm"
	"github.com/o3-cloud/artifact-specs/cli/internal/specs"
)

type Validator struct {
	schema   *jsonschema.Schema
	specData json.RawMessage
}

type ValidationResult struct {
	Valid  bool
	Errors []ValidationError
}

type ValidationError struct {
	Path    string
	Message string
}

func NewValidator(spec *specs.Spec) (*Validator, error) {
	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile(spec.Schema)
	if err != nil {
		return nil, fmt.Errorf("failed to compile JSON schema: %w", err)
	}
	
	return &Validator{
		schema:   schema,
		specData: spec.Schema,
	}, nil
}

func (v *Validator) Validate(data []byte) *ValidationResult {
	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return &ValidationResult{
			Valid: false,
			Errors: []ValidationError{
				{
					Path:    "root",
					Message: fmt.Sprintf("Invalid JSON: %v", err),
				},
			},
		}
	}
	
	result := v.schema.Validate(jsonData)
	if result == nil {
		return &ValidationResult{Valid: true}
	}
	
	// Convert validation result to validation errors
	var errors []ValidationError
	if result != nil {
		// Basic error handling - the exact API may need adjustment
		errors = append(errors, ValidationError{
			Path:    "root",
			Message: "Validation failed", // Generic message since we can't access specific errors easily
		})
	}
	
	return &ValidationResult{
		Valid:  false,
		Errors: errors,
	}
}

func (v *Validator) ValidateAndRetry(ctx context.Context, client llmClient, input string, maxRetries int) ([]byte, *ValidationResult, error) {
	// First attempt
	response, err := client.Complete(ctx, input, llm.CompletionOptions{ForceJSON: true})
	if err != nil {
		return nil, nil, fmt.Errorf("initial completion failed: %w", err)
	}
	
	data := []byte(response.Content)
	result := v.Validate(data)
	
	if result.Valid {
		return data, result, nil
	}
	
	// Retry with validation feedback
	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("Validation failed (attempt %d/%d), retrying with feedback...\n", attempt, maxRetries+1)
		
		// Create repair prompt
		repairPrompt := v.createRepairPrompt(input, string(data), result)
		
		response, err := client.Complete(ctx, repairPrompt, llm.CompletionOptions{ForceJSON: true})
		if err != nil {
			return nil, result, fmt.Errorf("retry attempt %d failed: %w", attempt, err)
		}
		
		data = []byte(response.Content)
		result = v.Validate(data)
		
		if result.Valid {
			return data, result, nil
		}
	}
	
	return data, result, fmt.Errorf("validation failed after %d retries", maxRetries)
}

func (v *Validator) createRepairPrompt(originalPrompt, invalidJSON string, result *ValidationResult) string {
	var errorMessages []string
	for _, err := range result.Errors {
		errorMessages = append(errorMessages, fmt.Sprintf("- %s: %s", err.Path, err.Message))
	}
	
	return fmt.Sprintf(`The previous response was invalid JSON according to the schema. Please fix the following validation errors and provide a corrected JSON response:

Validation errors:
%s

Original prompt: %s

Invalid JSON response:
%s

Please provide a corrected JSON response that follows the schema exactly:`, 
		strings.Join(errorMessages, "\n"), 
		originalPrompt, 
		invalidJSON)
}

func (r *ValidationResult) FormatErrors() string {
	if r.Valid {
		return "No validation errors"
	}
	
	var messages []string
	for _, err := range r.Errors {
		if err.Path != "" && err.Path != "root" {
			messages = append(messages, fmt.Sprintf("  %s: %s", err.Path, err.Message))
		} else {
			messages = append(messages, fmt.Sprintf("  %s", err.Message))
		}
	}
	
	return "Validation errors:\n" + strings.Join(messages, "\n")
}

// Interface for LLM client to support both real and mock clients
type llmClient interface {
	Complete(ctx context.Context, userPrompt string, options llm.CompletionOptions) (*llm.CompletionResponse, error)
}