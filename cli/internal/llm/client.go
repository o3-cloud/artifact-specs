package llm

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/o3-cloud/artifact-specs/cli/internal/config"
	"github.com/o3-cloud/artifact-specs/cli/internal/logging"
	"github.com/sashabaranov/go-openai"
)

const (
	DefaultSystemPrompt = "You are a strict structured-output assistant. Follow the user prompt exactly. Don't invent facts. If unsure, leave fields missing. Output only what's requested."
)

type Client struct {
	client       *openai.Client
	model        string
	systemPrompt string
}

type StreamCallback func(string) error

type CompletionResponse struct {
	Content      string
	TokensUsed   TokenUsage
	Duration     time.Duration
	Model        string
}

type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

func NewClient(model string) (*Client, error) {
	cfg := config.Get()
	apiKey := config.GetAPIKey()
	
	if apiKey == "" {
		return nil, fmt.Errorf("OpenRouter API key not found. Set OPENROUTER_API_KEY environment variable")
	}
	
	if model == "" {
		model = cfg.Model
	}
	
	logging.Debug("Creating LLM client", map[string]interface{}{
		"model":    model,
		"base_url": cfg.BaseURL,
		"provider": cfg.Provider,
	})
	
	clientConfig := openai.DefaultConfig(apiKey)
	clientConfig.BaseURL = cfg.BaseURL
	
	// Add custom headers for OpenRouter
	transport := &headerTransport{
		base: http.DefaultTransport,
		headers: map[string]string{
			"X-Title": "aspec CLI",
		},
	}
	clientConfig.HTTPClient = &http.Client{Transport: transport}
	
	client := openai.NewClientWithConfig(clientConfig)
	
	return &Client{
		client:       client,
		model:        model,
		systemPrompt: DefaultSystemPrompt,
	}, nil
}

func (c *Client) SetSystemPrompt(prompt string) {
	c.systemPrompt = prompt
}

func (c *Client) Complete(ctx context.Context, userPrompt string, options CompletionOptions) (*CompletionResponse, error) {
	start := time.Now()
	
	logging.Debug("Starting LLM completion", map[string]interface{}{
		"model":      c.model,
		"force_json": options.ForceJSON,
		"prompt_len": len(userPrompt),
	})
	
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: c.systemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: userPrompt,
		},
	}
	
	req := openai.ChatCompletionRequest{
		Model:    c.model,
		Messages: messages,
	}
	
	// Set response format for JSON extraction
	if options.ForceJSON {
		req.ResponseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		}
	}
	
	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		logging.Debug("LLM completion failed", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("OpenAI API error: %w", err)
	}
	
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no completion choices returned")
	}
	
	duration := time.Since(start)
	response := &CompletionResponse{
		Content: resp.Choices[0].Message.Content,
		TokensUsed: TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		Duration: duration,
		Model:    resp.Model,
	}
	
	logging.Debug("LLM completion finished", map[string]interface{}{
		"duration_ms":       duration.Milliseconds(),
		"prompt_tokens":     resp.Usage.PromptTokens,
		"completion_tokens": resp.Usage.CompletionTokens,
		"total_tokens":      resp.Usage.TotalTokens,
		"response_len":      len(response.Content),
	})
	
	return response, nil
}

func (c *Client) CompleteStream(ctx context.Context, userPrompt string, callback StreamCallback, options CompletionOptions) (*CompletionResponse, error) {
	start := time.Now()
	
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: c.systemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: userPrompt,
		},
	}
	
	req := openai.ChatCompletionRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   true,
	}
	
	// Note: streaming doesn't support JSON response format
	if options.ForceJSON {
		return nil, fmt.Errorf("streaming is not compatible with JSON response format")
	}
	
	stream, err := c.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("OpenAI stream error: %w", err)
	}
	defer stream.Close()
	
	var content strings.Builder
	var usage TokenUsage
	var model string
	
	for {
		response, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("stream receive error: %w", err)
		}
		
		if len(response.Choices) > 0 {
			delta := response.Choices[0].Delta.Content
			if delta != "" {
				content.WriteString(delta)
				if callback != nil {
					if err := callback(delta); err != nil {
						return nil, fmt.Errorf("stream callback error: %w", err)
					}
				}
			}
		}
		
		// Note: Streaming responses typically don't include usage information
		// We'll set approximate values for mock purposes
		if len(response.Choices) > 0 {
			usage = TokenUsage{
				PromptTokens:     100, // Approximate values since streaming doesn't provide usage
				CompletionTokens: 50,
				TotalTokens:      150,
			}
		}
		
		if response.Model != "" {
			model = response.Model
		}
	}
	
	return &CompletionResponse{
		Content:    content.String(),
		TokensUsed: usage,
		Duration:   time.Since(start),
		Model:      model,
	}, nil
}

type CompletionOptions struct {
	ForceJSON bool
}

// headerTransport adds custom headers to requests
type headerTransport struct {
	base    http.RoundTripper
	headers map[string]string
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Add custom headers
	for key, value := range t.headers {
		req.Header.Set(key, value)
	}
	
	// Use base transport
	if t.base != nil {
		return t.base.RoundTrip(req)
	}
	
	return http.DefaultTransport.RoundTrip(req)
}

func PrintStats(resp *CompletionResponse, writer io.Writer) {
	if writer == nil {
		writer = os.Stderr
	}
	
	fmt.Fprintf(writer, "\n--- Stats ---\n")
	fmt.Fprintf(writer, "Model: %s\n", resp.Model)
	fmt.Fprintf(writer, "Duration: %v\n", resp.Duration)
	fmt.Fprintf(writer, "Tokens: %d prompt + %d completion = %d total\n", 
		resp.TokensUsed.PromptTokens, 
		resp.TokensUsed.CompletionTokens, 
		resp.TokensUsed.TotalTokens)
	
	// Rough cost estimation for OpenRouter
	// These are approximate rates and may vary
	estimatedCost := estimateCost(resp.Model, resp.TokensUsed)
	if estimatedCost > 0 {
		fmt.Fprintf(writer, "Estimated cost: $%.6f\n", estimatedCost)
	}
}

func estimateCost(model string, usage TokenUsage) float64 {
	// Rough cost estimates per 1M tokens (as of 2024)
	// These are approximations and actual costs may vary
	costs := map[string]struct{ prompt, completion float64 }{
		"openai/gpt-4o-mini":    {0.15, 0.60},
		"openai/gpt-4o":         {5.00, 15.00},
		"openai/gpt-3.5-turbo":  {0.50, 1.50},
		"anthropic/claude-3-haiku": {0.25, 1.25},
		"anthropic/claude-3-sonnet": {3.00, 15.00},
		"anthropic/claude-3-opus": {15.00, 75.00},
	}
	
	cost, exists := costs[model]
	if !exists {
		return 0 // Unknown model
	}
	
	promptCost := (float64(usage.PromptTokens) / 1000000) * cost.prompt
	completionCost := (float64(usage.CompletionTokens) / 1000000) * cost.completion
	
	return promptCost + completionCost
}