package llm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type MockClient struct {
	responses map[string]string
	model     string
}

func NewMockClient(model string) *MockClient {
	return &MockClient{
		responses: make(map[string]string),
		model:     model,
	}
}

func (m *MockClient) LoadFixture(fixturePath string) error {
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		return fmt.Errorf("failed to load mock fixture: %w", err)
	}
	
	// Use a simple key for now - in practice you might want to hash the prompt
	key := filepath.Base(fixturePath)
	m.responses[key] = string(content)
	
	return nil
}

func (m *MockClient) SetResponse(key, response string) {
	m.responses[key] = response
}

func (m *MockClient) Complete(ctx context.Context, userPrompt string, options CompletionOptions) (*CompletionResponse, error) {
	// For mock, we'll use a default response if no specific one is set
	response := "Mock response"
	if len(m.responses) > 0 {
		// Use the first available response
		for _, resp := range m.responses {
			response = resp
			break
		}
	}
	
	return &CompletionResponse{
		Content: response,
		TokensUsed: TokenUsage{
			PromptTokens:     100, // Mock values
			CompletionTokens: 50,
			TotalTokens:      150,
		},
		Duration: time.Millisecond * 100,
		Model:    m.model,
	}, nil
}

func (m *MockClient) CompleteStream(ctx context.Context, userPrompt string, callback StreamCallback, options CompletionOptions) (*CompletionResponse, error) {
	response := "Mock response"
	if len(m.responses) > 0 {
		for _, resp := range m.responses {
			response = resp
			break
		}
	}
	
	// Simulate streaming by calling callback with chunks
	if callback != nil {
		words := []string{"Mock", " ", "streaming", " ", "response"}
		for _, word := range words {
			if err := callback(word); err != nil {
				return nil, err
			}
			time.Sleep(time.Millisecond * 10) // Simulate delay
		}
	}
	
	return &CompletionResponse{
		Content: response,
		TokensUsed: TokenUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
		Duration: time.Millisecond * 100,
		Model:    m.model,
	}, nil
}

func (m *MockClient) SetSystemPrompt(prompt string) {
	// Mock client doesn't need to do anything with system prompt
}