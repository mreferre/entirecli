package kirocli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/entireio/cli/cmd/entire/cli/transcript"
)

func TestGetTranscriptPosition(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "session.jsonl")

	// Write a JSONL file with 3 lines
	content := `{"type":"user","uuid":"u1","message":{"content":"hello"}}
{"type":"assistant","uuid":"a1","message":{"content":[{"type":"text","text":"hi"}]}}
{"type":"user","uuid":"u2","message":{"content":"bye"}}
`
	if err := os.WriteFile(transcriptPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	ag := &KiroCLIAgent{}
	pos, err := ag.GetTranscriptPosition(transcriptPath)
	if err != nil {
		t.Fatalf("GetTranscriptPosition() error = %v", err)
	}

	if pos != 3 {
		t.Errorf("GetTranscriptPosition() = %d, want 3", pos)
	}
}

func TestGetTranscriptPosition_EmptyFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "empty.jsonl")

	// Write an empty file
	if err := os.WriteFile(transcriptPath, []byte{}, 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	ag := &KiroCLIAgent{}
	pos, err := ag.GetTranscriptPosition(transcriptPath)
	if err != nil {
		t.Fatalf("GetTranscriptPosition() error = %v", err)
	}

	if pos != 0 {
		t.Errorf("GetTranscriptPosition() = %d, want 0 for empty file", pos)
	}
}

func TestGetTranscriptPosition_NonExistentFile(t *testing.T) {
	t.Parallel()

	ag := &KiroCLIAgent{}
	pos, err := ag.GetTranscriptPosition("/nonexistent/path.jsonl")
	if err != nil {
		t.Fatalf("GetTranscriptPosition() error = %v, want nil for non-existent file", err)
	}

	if pos != 0 {
		t.Errorf("GetTranscriptPosition() = %d, want 0 for non-existent file", pos)
	}
}

func TestGetTranscriptPosition_EmptyPath(t *testing.T) {
	t.Parallel()

	ag := &KiroCLIAgent{}
	pos, err := ag.GetTranscriptPosition("")
	if err != nil {
		t.Fatalf("GetTranscriptPosition() error = %v", err)
	}

	if pos != 0 {
		t.Errorf("GetTranscriptPosition() = %d, want 0 for empty path", pos)
	}
}

func TestExtractModifiedFilesFromOffset(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "session.jsonl")

	// Write a JSONL file with file modification tool calls
	content := `{"type":"user","uuid":"u1","message":{"content":"write files"}}
{"type":"assistant","uuid":"a1","message":{"content":[{"type":"tool_use","name":"fs_write","input":{"file_path":"foo.go"}}]}}
{"type":"assistant","uuid":"a2","message":{"content":[{"type":"tool_use","name":"edit","input":{"file_path":"bar.go"}}]}}
{"type":"assistant","uuid":"a3","message":{"content":[{"type":"tool_use","name":"bash","input":{"command":"ls"}}]}}
{"type":"assistant","uuid":"a4","message":{"content":[{"type":"tool_use","name":"write","input":{"file_path":"baz.go"}}]}}
`
	if err := os.WriteFile(transcriptPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	ag := &KiroCLIAgent{}
	files, pos, err := ag.ExtractModifiedFilesFromOffset(transcriptPath, 0)
	if err != nil {
		t.Fatalf("ExtractModifiedFilesFromOffset() error = %v", err)
	}

	if pos != 5 {
		t.Errorf("ExtractModifiedFilesFromOffset() position = %d, want 5", pos)
	}

	// Should have foo.go, bar.go, baz.go (bash not included)
	if len(files) != 3 {
		t.Errorf("ExtractModifiedFilesFromOffset() got %d files, want 3: %v", len(files), files)
	}

	hasFile := func(name string) bool {
		for _, f := range files {
			if f == name {
				return true
			}
		}
		return false
	}

	if !hasFile("foo.go") {
		t.Error("missing foo.go in extracted files")
	}
	if !hasFile("bar.go") {
		t.Error("missing bar.go in extracted files")
	}
	if !hasFile("baz.go") {
		t.Error("missing baz.go in extracted files")
	}
}

func TestExtractModifiedFilesFromOffset_WithOffset(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "session.jsonl")

	// Write a JSONL file with file modification tool calls
	content := `{"type":"assistant","uuid":"a1","message":{"content":[{"type":"tool_use","name":"fs_write","input":{"file_path":"old.go"}}]}}
{"type":"assistant","uuid":"a2","message":{"content":[{"type":"tool_use","name":"fs_write","input":{"file_path":"new.go"}}]}}
`
	if err := os.WriteFile(transcriptPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	ag := &KiroCLIAgent{}
	// Start from offset 1 (skip first line)
	files, pos, err := ag.ExtractModifiedFilesFromOffset(transcriptPath, 1)
	if err != nil {
		t.Fatalf("ExtractModifiedFilesFromOffset() error = %v", err)
	}

	if pos != 2 {
		t.Errorf("ExtractModifiedFilesFromOffset() position = %d, want 2", pos)
	}

	// Should only have new.go (old.go is before offset)
	if len(files) != 1 {
		t.Errorf("ExtractModifiedFilesFromOffset() got %d files, want 1: %v", len(files), files)
	}

	if len(files) > 0 && files[0] != "new.go" {
		t.Errorf("ExtractModifiedFilesFromOffset() got %q, want new.go", files[0])
	}
}

func TestExtractModifiedFilesFromOffset_NoFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "session.jsonl")

	// Write a JSONL file with no file modification tools
	content := `{"type":"user","uuid":"u1","message":{"content":"do something"}}
{"type":"assistant","uuid":"a1","message":{"content":[{"type":"text","text":"done"}]}}
`
	if err := os.WriteFile(transcriptPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	ag := &KiroCLIAgent{}
	files, pos, err := ag.ExtractModifiedFilesFromOffset(transcriptPath, 0)
	if err != nil {
		t.Fatalf("ExtractModifiedFilesFromOffset() error = %v", err)
	}

	if pos != 2 {
		t.Errorf("ExtractModifiedFilesFromOffset() position = %d, want 2", pos)
	}

	if len(files) != 0 {
		t.Errorf("ExtractModifiedFilesFromOffset() got %d files, want 0", len(files))
	}
}

func TestExtractModifiedFilesFromOffset_EmptyPath(t *testing.T) {
	t.Parallel()

	ag := &KiroCLIAgent{}
	files, pos, err := ag.ExtractModifiedFilesFromOffset("", 0)
	if err != nil {
		t.Fatalf("ExtractModifiedFilesFromOffset() error = %v", err)
	}

	if pos != 0 {
		t.Errorf("ExtractModifiedFilesFromOffset() position = %d, want 0", pos)
	}

	if files != nil {
		t.Errorf("ExtractModifiedFilesFromOffset() files = %v, want nil", files)
	}
}

func TestExtractPrompts(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "session.jsonl")

	// Write a JSONL file with user prompts
	content := `{"type":"user","uuid":"u1","message":{"content":"first prompt"}}
{"type":"assistant","uuid":"a1","message":{"content":[{"type":"text","text":"response 1"}]}}
{"type":"user","uuid":"u2","message":{"content":"second prompt"}}
{"type":"assistant","uuid":"a2","message":{"content":[{"type":"text","text":"response 2"}]}}
`
	if err := os.WriteFile(transcriptPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	ag := &KiroCLIAgent{}
	prompts, err := ag.ExtractPrompts(transcriptPath, 0)
	if err != nil {
		t.Fatalf("ExtractPrompts() error = %v", err)
	}

	if len(prompts) != 2 {
		t.Errorf("ExtractPrompts() got %d prompts, want 2", len(prompts))
	}

	if len(prompts) >= 2 {
		if prompts[0] != "first prompt" {
			t.Errorf("prompts[0] = %q, want %q", prompts[0], "first prompt")
		}
		if prompts[1] != "second prompt" {
			t.Errorf("prompts[1] = %q, want %q", prompts[1], "second prompt")
		}
	}
}

func TestExtractPrompts_WithOffset(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "session.jsonl")

	// Write a JSONL file with user prompts
	content := `{"type":"user","uuid":"u1","message":{"content":"old prompt"}}
{"type":"assistant","uuid":"a1","message":{"content":[{"type":"text","text":"response 1"}]}}
{"type":"user","uuid":"u2","message":{"content":"new prompt"}}
`
	if err := os.WriteFile(transcriptPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	ag := &KiroCLIAgent{}
	// Start from offset 2 (skip first two lines)
	prompts, err := ag.ExtractPrompts(transcriptPath, 2)
	if err != nil {
		t.Fatalf("ExtractPrompts() error = %v", err)
	}

	if len(prompts) != 1 {
		t.Errorf("ExtractPrompts() got %d prompts, want 1", len(prompts))
	}

	if len(prompts) >= 1 && prompts[0] != "new prompt" {
		t.Errorf("prompts[0] = %q, want %q", prompts[0], "new prompt")
	}
}

func TestExtractSummary(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "session.jsonl")

	// Write a JSONL file with assistant responses
	content := `{"type":"user","uuid":"u1","message":{"content":"do something"}}
{"type":"assistant","uuid":"a1","message":{"content":[{"type":"text","text":"Here is the summary of what I did."}]}}
`
	if err := os.WriteFile(transcriptPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	ag := &KiroCLIAgent{}
	summary, err := ag.ExtractSummary(transcriptPath)
	if err != nil {
		t.Fatalf("ExtractSummary() error = %v", err)
	}

	expected := "Here is the summary of what I did."
	if summary != expected {
		t.Errorf("ExtractSummary() = %q, want %q", summary, expected)
	}
}

func TestExtractSummary_LastMessage(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "session.jsonl")

	// Write a JSONL file with multiple assistant responses
	content := `{"type":"assistant","uuid":"a1","message":{"content":[{"type":"text","text":"First response"}]}}
{"type":"user","uuid":"u1","message":{"content":"continue"}}
{"type":"assistant","uuid":"a2","message":{"content":[{"type":"text","text":"Final response"}]}}
`
	if err := os.WriteFile(transcriptPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	ag := &KiroCLIAgent{}
	summary, err := ag.ExtractSummary(transcriptPath)
	if err != nil {
		t.Fatalf("ExtractSummary() error = %v", err)
	}

	// Should return the last assistant message
	if summary != "Final response" {
		t.Errorf("ExtractSummary() = %q, want %q", summary, "Final response")
	}
}

func TestExtractSummary_NoAssistantMessage(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "session.jsonl")

	// Write a JSONL file with only user messages
	content := `{"type":"user","uuid":"u1","message":{"content":"hello"}}
`
	if err := os.WriteFile(transcriptPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	ag := &KiroCLIAgent{}
	summary, err := ag.ExtractSummary(transcriptPath)
	if err != nil {
		t.Fatalf("ExtractSummary() error = %v", err)
	}

	if summary != "" {
		t.Errorf("ExtractSummary() = %q, want empty string when no assistant message", summary)
	}
}

func TestSerializeTranscript(t *testing.T) {
	t.Parallel()

	lines := []TranscriptLine{
		{Type: "user", UUID: "u1"},
		{Type: "assistant", UUID: "a1"},
	}

	data, err := SerializeTranscript(lines)
	if err != nil {
		t.Fatalf("SerializeTranscript() error = %v", err)
	}

	// Parse back to verify round-trip
	parsed, err := transcript.ParseFromBytes(data)
	if err != nil {
		t.Fatalf("ParseFromBytes(serialized) error = %v", err)
	}

	if len(parsed) != 2 {
		t.Errorf("Round-trip got %d lines, want 2", len(parsed))
	}
}

func TestExtractModifiedFiles(t *testing.T) {
	t.Parallel()

	data := []byte(`{"type":"assistant","uuid":"a1","message":{"content":[{"type":"tool_use","name":"fs_write","input":{"file_path":"foo.go"}}]}}
{"type":"assistant","uuid":"a2","message":{"content":[{"type":"tool_use","name":"edit","input":{"file_path":"bar.go"}}]}}
{"type":"assistant","uuid":"a3","message":{"content":[{"type":"tool_use","name":"bash","input":{"command":"ls"}}]}}
{"type":"assistant","uuid":"a4","message":{"content":[{"type":"tool_use","name":"fs_write","input":{"file_path":"foo.go"}}]}}
`)

	lines, err := transcript.ParseFromBytes(data)
	if err != nil {
		t.Fatalf("ParseFromBytes() error = %v", err)
	}
	files := ExtractModifiedFiles(lines)

	// Should have foo.go and bar.go (deduplicated, bash not included)
	if len(files) != 2 {
		t.Errorf("ExtractModifiedFiles() got %d files, want 2: %v", len(files), files)
	}

	hasFile := func(name string) bool {
		for _, f := range files {
			if f == name {
				return true
			}
		}
		return false
	}

	if !hasFile("foo.go") {
		t.Error("missing foo.go in ExtractModifiedFiles result")
	}
	if !hasFile("bar.go") {
		t.Error("missing bar.go in ExtractModifiedFiles result")
	}
}

func TestExtractLastUserPrompt(t *testing.T) {
	t.Parallel()

	data := []byte(`{"type":"user","uuid":"u1","message":{"content":"first"}}
{"type":"assistant","uuid":"a1","message":{"content":[{"type":"text","text":"response"}]}}
{"type":"user","uuid":"u2","message":{"content":"last"}}
`)

	lines, err := transcript.ParseFromBytes(data)
	if err != nil {
		t.Fatalf("ParseFromBytes() error = %v", err)
	}

	prompt := ExtractLastUserPrompt(lines)
	if prompt != "last" {
		t.Errorf("ExtractLastUserPrompt() = %q, want %q", prompt, "last")
	}
}

func TestExtractLastUserPrompt_Empty(t *testing.T) {
	t.Parallel()

	var lines []TranscriptLine
	prompt := ExtractLastUserPrompt(lines)
	if prompt != "" {
		t.Errorf("ExtractLastUserPrompt() = %q, want empty string for empty input", prompt)
	}
}

func TestCalculateTokenUsage(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "session.jsonl")

	// Write a JSONL file with token usage data
	content := `{"type":"user","uuid":"u1","message":{"content":"hello"}}
{"type":"assistant","uuid":"a1","message":{"id":"msg-1","content":[{"type":"text","text":"hi"}],"usage":{"input_tokens":10,"output_tokens":5}}}
{"type":"assistant","uuid":"a2","message":{"id":"msg-2","content":[{"type":"text","text":"bye"}],"usage":{"input_tokens":15,"output_tokens":8}}}
`
	if err := os.WriteFile(transcriptPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	ag := &KiroCLIAgent{}
	usage, err := ag.CalculateTokenUsage(transcriptPath, 0)
	if err != nil {
		t.Fatalf("CalculateTokenUsage() error = %v", err)
	}

	if usage.APICallCount != 2 {
		t.Errorf("APICallCount = %d, want 2", usage.APICallCount)
	}
	if usage.InputTokens != 25 {
		t.Errorf("InputTokens = %d, want 25", usage.InputTokens)
	}
	if usage.OutputTokens != 13 {
		t.Errorf("OutputTokens = %d, want 13", usage.OutputTokens)
	}
}

func TestCalculateTokenUsage_EmptyPath(t *testing.T) {
	t.Parallel()

	ag := &KiroCLIAgent{}
	usage, err := ag.CalculateTokenUsage("", 0)
	if err != nil {
		t.Fatalf("CalculateTokenUsage() error = %v", err)
	}

	if usage.APICallCount != 0 {
		t.Errorf("APICallCount = %d, want 0", usage.APICallCount)
	}
}

func TestCalculateTokenUsage_Deduplication(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "session.jsonl")

	// Write a JSONL file with duplicate message IDs (streaming scenario)
	// The entry with highest output_tokens should be kept
	content := `{"type":"assistant","uuid":"a1","message":{"id":"msg-1","usage":{"input_tokens":10,"output_tokens":2}}}
{"type":"assistant","uuid":"a2","message":{"id":"msg-1","usage":{"input_tokens":10,"output_tokens":5}}}
{"type":"assistant","uuid":"a3","message":{"id":"msg-1","usage":{"input_tokens":10,"output_tokens":3}}}
`
	if err := os.WriteFile(transcriptPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	ag := &KiroCLIAgent{}
	usage, err := ag.CalculateTokenUsage(transcriptPath, 0)
	if err != nil {
		t.Fatalf("CalculateTokenUsage() error = %v", err)
	}

	// Should only count once with highest output_tokens (5)
	if usage.APICallCount != 1 {
		t.Errorf("APICallCount = %d, want 1", usage.APICallCount)
	}
	if usage.InputTokens != 10 {
		t.Errorf("InputTokens = %d, want 10", usage.InputTokens)
	}
	if usage.OutputTokens != 5 {
		t.Errorf("OutputTokens = %d, want 5", usage.OutputTokens)
	}
}
