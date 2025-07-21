package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Constants
const (
	defaultLogsFile     = "logs.txt"
	responseFile        = "response.json"
	defaultModel        = "ibm-granite/granite-3.1-8b-instruct"
	maxTokens           = 700
	temperature         = 0.2
	completionsEndpoint = "/v1/completions"
)

// CompletionRequest represents the API request payload
type CompletionRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

// CompletionResponse represents the API response
type CompletionResponse struct {
	Choices []struct {
		Text string `json:"text"`
	} `json:"choices"`
}

// getSystemPrompt returns the troubleshooting system prompt
func getSystemPrompt() string {
	return `You are an expert OpenShift CI/CD pipeline troubleshooting assistant. Analyze the provided build logs and provide a structured diagnosis.

**Your Task:**
1. **Identify Failures**: Extract specific errors, failed steps, and affected components
2. **Root Cause Analysis**: Determine the underlying cause of each failure
3. **Provide Solutions**: Give clear, actionable steps to resolve the issues
4. **Categorize Issues**: Label errors by type (resource, network, config, etc.)
5. **Suggest Prevention**: Recommend best practices to avoid similar failures

**Output Format:**
## Summary
- Brief overview of what failed

## Root Cause
- Primary cause of the failure
- Contributing factors

## Solutions
1. Immediate fixes (step-by-step)
2. Configuration changes needed
3. Commands to run (kubectl, oc, etc.)

## Error Category
- Primary: [Resource/Network/Configuration/Image/Permissions/Other]
- Secondary: [specific subcategory]

## Prevention
- Best practices to prevent recurrence
- Monitoring/alerting improvements

**Focus on:**
- Actionable solutions over theory
- Specific commands and configurations
- OpenShift/Kubernetes best practices
- Clear, concise explanations`
}

// readLogsFile reads and validates the build logs file
func readLogsFile(filename string) (string, error) {
	if filename == "" {
		return "", fmt.Errorf("filename cannot be empty")
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("logs file '%s' not found", filename)
		}
		return "", fmt.Errorf("failed to read logs file: %w", err)
	}

	if len(content) == 0 {
		return "", fmt.Errorf("logs file '%s' is empty", filename)
	}

	return strings.TrimSpace(string(content)), nil
}

// createAnalysisPrompt combines system prompt with build logs
func createAnalysisPrompt(buildLogs string) string {
	return fmt.Sprintf("%s\n\nUser Request:\nThere's been a failed build! Here is the output:\n%s\n\nAssistant:",
		getSystemPrompt(), buildLogs)
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: Could not load .env file, using system environment variables")
	}

	// Get required environment variables
	endpoint := strings.TrimSpace(os.Getenv("MODEL_API_ENDPOINT"))
	token := strings.TrimSpace(os.Getenv("MODEL_API_TOKEN"))

	if endpoint == "" || token == "" {
		fmt.Println("Error: MODEL_API_ENDPOINT and MODEL_API_TOKEN must be set")
		os.Exit(1)
	}

	// Determine logs file
	logsFile := defaultLogsFile
	if len(os.Args) > 1 {
		logsFile = strings.TrimSpace(os.Args[1])
	}

	// Read and analyze build logs
	fmt.Printf("Reading build logs from %s...\n", logsFile)
	buildLogs, err := readLogsFile(logsFile)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Create and send analysis request
	fmt.Println("Analyzing build logs with AI model...")
	request := CompletionRequest{
		Model:       defaultModel,
		Prompt:      createAnalysisPrompt(buildLogs),
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	response, err := sendRequest(endpoint, token, request)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Display and save results
	if len(response.Choices) > 0 {
		fmt.Println("\n=== AI Analysis Results ===")
		fmt.Println(response.Choices[0].Text)
	} else {
		fmt.Println("No analysis results received from the model")
		os.Exit(1)
	}

	if err := saveResponse(response); err != nil {
		fmt.Printf("Warning: Failed to save response: %v\n", err)
	} else {
		fmt.Printf("\nFull response saved to %s\n", responseFile)
	}
}

// sendRequest handles the HTTP request to the API
func sendRequest(endpoint, token string, request CompletionRequest) (*CompletionResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to create JSON: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint+completionsEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response CompletionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// saveResponse saves the API response to a JSON file
func saveResponse(response *CompletionResponse) error {
	file, err := os.Create(responseFile)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(response)
}
