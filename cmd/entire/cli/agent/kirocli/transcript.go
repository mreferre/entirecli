package kirocli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/entireio/cli/cmd/entire/cli/agent"
	"github.com/entireio/cli/cmd/entire/cli/transcript"
)

// TranscriptLine is an alias to the shared transcript.Line type.
// Kiro CLI uses JSONL format like Claude Code.
type TranscriptLine = transcript.Line

// Type aliases for internal use.
type (
	userMessage      = transcript.UserMessage
	assistantMessage = transcript.AssistantMessage
	toolInput        = transcript.ToolInput
)

// SerializeTranscript converts transcript lines back to JSONL bytes.
func SerializeTranscript(lines []TranscriptLine) ([]byte, error) {
	var buf bytes.Buffer
	for _, line := range lines {
		data, err := json.Marshal(line)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal line: %w", err)
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}

// ExtractModifiedFiles extracts files modified by tool calls from transcript.
func ExtractModifiedFiles(lines []TranscriptLine) []string {
	fileSet := make(map[string]bool)
	var files []string

	for _, line := range lines {
		if line.Type != transcript.TypeAssistant {
			continue
		}

		var msg assistantMessage
		if err := json.Unmarshal(line.Message, &msg); err != nil {
			continue
		}

		for _, block := range msg.Content {
			if block.Type != transcript.ContentTypeToolUse {
				continue
			}

			// Check if it's a file modification tool
			isModifyTool := false
			for _, name := range FileModificationTools {
				if block.Name == name {
					isModifyTool = true
					break
				}
			}

			if !isModifyTool {
				continue
			}

			var input toolInput
			if err := json.Unmarshal(block.Input, &input); err != nil {
				continue
			}

			file := input.FilePath
			if file != "" && !fileSet[file] {
				fileSet[file] = true
				files = append(files, file)
			}
		}
	}

	return files
}

// ExtractLastUserPrompt extracts the last user message from transcript.
func ExtractLastUserPrompt(lines []TranscriptLine) string {
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i].Type != transcript.TypeUser {
			continue
		}

		var msg userMessage
		if err := json.Unmarshal(lines[i].Message, &msg); err != nil {
			continue
		}

		// Handle string content
		if str, ok := msg.Content.(string); ok {
			return str
		}

		// Handle array content (text blocks)
		if arr, ok := msg.Content.([]interface{}); ok {
			var texts []string
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					if m["type"] == transcript.ContentTypeText {
						if text, ok := m["text"].(string); ok {
							texts = append(texts, text)
						}
					}
				}
			}
			if len(texts) > 0 {
				return strings.Join(texts, "\n\n")
			}
		}
	}
	return ""
}

// --- TranscriptAnalyzer interface implementation ---

// GetTranscriptPosition returns the current line count of a Kiro transcript.
// Kiro CLI uses JSONL format, so position is the number of lines.
// This is a lightweight operation that only counts lines without parsing JSON.
// Returns 0 if the file doesn't exist or is empty.
func (k *KiroCLIAgent) GetTranscriptPosition(path string) (int, error) {
	if path == "" {
		return 0, nil
	}

	file, err := os.Open(path) //nolint:gosec // Path comes from Kiro CLI transcript location
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to open transcript file: %w", err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	lineCount := 0

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				if len(line) > 0 {
					lineCount++ // Count final line without trailing newline
				}
				break
			}
			return 0, fmt.Errorf("failed to read transcript: %w", err)
		}
		lineCount++
	}

	return lineCount, nil
}

// ExtractModifiedFilesFromOffset extracts files modified since a given line number.
// For Kiro CLI (JSONL format), offset is the starting line number.
// Returns:
//   - files: list of file paths modified by Kiro (from Write/Edit tools)
//   - currentPosition: total number of lines in the file
//   - error: any error encountered during reading
func (k *KiroCLIAgent) ExtractModifiedFilesFromOffset(path string, startOffset int) (files []string, currentPosition int, err error) {
	if path == "" {
		return nil, 0, nil
	}

	file, openErr := os.Open(path) //nolint:gosec // Path comes from Kiro CLI transcript location
	if openErr != nil {
		return nil, 0, fmt.Errorf("failed to open transcript file: %w", openErr)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	var lines []TranscriptLine
	lineNum := 0

	for {
		lineData, readErr := reader.ReadBytes('\n')
		if readErr != nil && readErr != io.EOF {
			return nil, 0, fmt.Errorf("failed to read transcript: %w", readErr)
		}

		if len(lineData) > 0 {
			lineNum++
			if lineNum > startOffset {
				var line TranscriptLine
				if parseErr := json.Unmarshal(lineData, &line); parseErr == nil {
					lines = append(lines, line)
				}
				// Skip malformed lines silently
			}
		}

		if readErr == io.EOF {
			break
		}
	}

	return ExtractModifiedFiles(lines), lineNum, nil
}

// ExtractPrompts extracts user prompts from the transcript starting at the given line offset.
func (k *KiroCLIAgent) ExtractPrompts(sessionRef string, fromOffset int) ([]string, error) {
	lines, _, err := transcript.ParseFromFileAtLine(sessionRef, fromOffset)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transcript: %w", err)
	}

	var prompts []string
	for i := range lines {
		if lines[i].Type != transcript.TypeUser {
			continue
		}
		content := transcript.ExtractUserContent(lines[i].Message)
		if content != "" {
			prompts = append(prompts, content)
		}
	}
	return prompts, nil
}

// ExtractSummary extracts the last assistant message as a session summary.
func (k *KiroCLIAgent) ExtractSummary(sessionRef string) (string, error) {
	data, err := os.ReadFile(sessionRef) //nolint:gosec // Path comes from agent hook input
	if err != nil {
		return "", fmt.Errorf("failed to read transcript: %w", err)
	}

	lines, parseErr := transcript.ParseFromBytes(data)
	if parseErr != nil {
		return "", fmt.Errorf("failed to parse transcript: %w", parseErr)
	}

	// Walk backward to find last assistant text block
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i].Type != transcript.TypeAssistant {
			continue
		}
		var msg transcript.AssistantMessage
		if err := json.Unmarshal(lines[i].Message, &msg); err != nil {
			continue
		}
		for _, block := range msg.Content {
			if block.Type == transcript.ContentTypeText && block.Text != "" {
				return block.Text, nil
			}
		}
	}
	return "", nil
}

// --- TokenCalculator interface implementation ---

// messageWithUsage represents an assistant message with token usage data.
type messageWithUsage struct {
	ID    string       `json:"id"`
	Usage messageUsage `json:"usage"`
}

// messageUsage represents Anthropic-style token usage in a message.
type messageUsage struct {
	InputTokens              int `json:"input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	OutputTokens             int `json:"output_tokens"`
}

// CalculateTokenUsage computes token usage from the transcript starting at the given line offset.
// Kiro CLI may use Anthropic-style token usage format in assistant messages.
// Due to streaming, multiple transcript rows may share the same message.id.
// We deduplicate by taking the row with the highest output_tokens for each message.id.
func (k *KiroCLIAgent) CalculateTokenUsage(sessionRef string, fromOffset int) (*agent.TokenUsage, error) {
	if sessionRef == "" {
		return &agent.TokenUsage{}, nil
	}

	lines, _, err := transcript.ParseFromFileAtLine(sessionRef, fromOffset)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transcript: %w", err)
	}

	// Map from message.id to the usage with highest output_tokens
	usageByMessageID := make(map[string]messageUsage)

	for _, line := range lines {
		if line.Type != transcript.TypeAssistant {
			continue
		}

		var msg messageWithUsage
		if err := json.Unmarshal(line.Message, &msg); err != nil {
			continue
		}

		if msg.ID == "" {
			continue
		}

		// Keep the entry with highest output_tokens (final streaming state)
		existing, exists := usageByMessageID[msg.ID]
		if !exists || msg.Usage.OutputTokens > existing.OutputTokens {
			usageByMessageID[msg.ID] = msg.Usage
		}
	}

	// Sum up all unique messages
	usage := &agent.TokenUsage{
		APICallCount: len(usageByMessageID),
	}
	for _, u := range usageByMessageID {
		usage.InputTokens += u.InputTokens
		usage.CacheCreationTokens += u.CacheCreationInputTokens
		usage.CacheReadTokens += u.CacheReadInputTokens
		usage.OutputTokens += u.OutputTokens
	}

	return usage, nil
}
