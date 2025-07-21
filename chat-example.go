package main

import (
	"bytes"
	"encoding/json"
	"net/http"
)

// For /v1/chat/completions endpoint
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

type ChatCompletionResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
}

func createChatRequest(systemPrompt, userPrompt string) ChatCompletionRequest {
	return ChatCompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   700,
		Temperature: 0.2,
	}
}

func sendChatRequest(endpoint, token string, request ChatCompletionRequest) (*ChatCompletionResponse, error) {
	jsonData, _ := json.Marshal(request)

	req, _ := http.NewRequest("POST", endpoint+"/v1/chat/completions", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// ... rest of HTTP handling
	return nil, nil
}

// Usage would be:
// systemPrompt := getSystemPrompt()
// userPrompt := "There's been a failed build! Here is the output:\n" + buildLogs
// request := createChatRequest(systemPrompt, userPrompt)
// response := sendChatRequest(endpoint, token, request)
// result := response.Choices[0].Message.Content
