package kirocli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/entireio/cli/cmd/entire/cli/agent"
)

func TestNewKiroCLIAgent(t *testing.T) {
	t.Parallel()
	ag := NewKiroCLIAgent()
	if ag == nil {
		t.Fatal("NewKiroCLIAgent() returned nil")
	}
	_, ok := ag.(*KiroCLIAgent)
	if !ok {
		t.Error("NewKiroCLIAgent() did not return *KiroCLIAgent")
	}
}

func TestName(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}
	if ag.Name() != agent.AgentNameKiro {
		t.Errorf("Name() = %q, want %q", ag.Name(), agent.AgentNameKiro)
	}
}

func TestType(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}
	if ag.Type() != agent.AgentTypeKiro {
		t.Errorf("Type() = %q, want %q", ag.Type(), agent.AgentTypeKiro)
	}
}

func TestDescription(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}
	desc := ag.Description()
	if desc != "Kiro CLI - AI-powered terminal coding agent" {
		t.Errorf("Description() = %q, want %q", desc, "Kiro CLI - AI-powered terminal coding agent")
	}
}

func TestIsPreview(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}
	if !ag.IsPreview() {
		t.Error("IsPreview() = false, want true")
	}
}

func TestProtectedDirs(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}
	dirs := ag.ProtectedDirs()
	if len(dirs) != 1 || dirs[0] != ".kiro" {
		t.Errorf("ProtectedDirs() = %v, want [.kiro]", dirs)
	}
}

func TestResolveSessionFile(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}
	result := ag.ResolveSessionFile("/home/user/.kiro/projects/foo", "abc-123-def")
	expected := "/home/user/.kiro/projects/foo/abc-123-def.jsonl"
	if result != expected {
		t.Errorf("ResolveSessionFile() = %q, want %q", result, expected)
	}
}

func TestGetSessionID(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}
	input := &agent.HookInput{
		SessionID: "test-session-123",
	}
	if ag.GetSessionID(input) != "test-session-123" {
		t.Errorf("GetSessionID() = %q, want %q", ag.GetSessionID(input), "test-session-123")
	}
}

func TestFormatResumeCommand(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}
	result := ag.FormatResumeCommand("test-session-123")
	expected := "kiro-cli --resume test-session-123"
	if result != expected {
		t.Errorf("FormatResumeCommand() = %q, want %q", result, expected)
	}
}

func TestSanitizePathForKiro(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected string
	}{
		{"/home/user/projects/my-project", "-home-user-projects-my-project"},
		{"/usr/local/src", "-usr-local-src"},
		{"simple", "simple"},
	}
	for _, tc := range tests {
		result := SanitizePathForKiro(tc.input)
		if result != tc.expected {
			t.Errorf("SanitizePathForKiro(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestDetectPresence(t *testing.T) {
	// Cannot use t.Parallel() because subtests use t.Chdir

	t.Run("with .kiro directory", func(t *testing.T) {
		// Create a temp directory with .kiro
		tmpDir := t.TempDir()
		t.Chdir(tmpDir)

		// Initialize git repo (needed for WorktreeRoot)
		if err := os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755); err != nil {
			t.Fatalf("failed to create .git dir: %v", err)
		}

		// Create .kiro directory
		if err := os.MkdirAll(filepath.Join(tmpDir, ".kiro"), 0o755); err != nil {
			t.Fatalf("failed to create .kiro dir: %v", err)
		}

		ag := &KiroCLIAgent{}
		present, err := ag.DetectPresence()
		if err != nil {
			t.Errorf("DetectPresence() error = %v", err)
		}
		if !present {
			t.Error("DetectPresence() = false, want true when .kiro exists")
		}
	})

	t.Run("without .kiro directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Chdir(tmpDir)

		// Initialize git repo (needed for WorktreeRoot)
		if err := os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755); err != nil {
			t.Fatalf("failed to create .git dir: %v", err)
		}

		ag := &KiroCLIAgent{}
		present, err := ag.DetectPresence()
		if err != nil {
			t.Errorf("DetectPresence() error = %v", err)
		}
		if present {
			t.Error("DetectPresence() = true, want false when .kiro does not exist")
		}
	})
}

func TestAgentRegistration(t *testing.T) {
	t.Parallel()
	// Verify the agent is registered in the registry
	names := agent.List()
	found := false
	for _, name := range names {
		if name == agent.AgentNameKiro {
			found = true
			break
		}
	}
	if !found {
		t.Error("Kiro agent not found in agent.List()")
	}

	// Verify we can get the agent
	ag, err := agent.Get(agent.AgentNameKiro)
	if err != nil {
		t.Fatalf("agent.Get(AgentNameKiro) error = %v", err)
	}
	if ag.Name() != agent.AgentNameKiro {
		t.Errorf("agent.Get(AgentNameKiro).Name() = %q, want %q", ag.Name(), agent.AgentNameKiro)
	}
}

func TestGetSessionDir(t *testing.T) {
	ag := &KiroCLIAgent{}

	// Test with override env var
	t.Setenv("ENTIRE_TEST_KIRO_PROJECT_DIR", "/test/override")

	dir, err := ag.GetSessionDir("/some/repo")
	if err != nil {
		t.Fatalf("GetSessionDir() error = %v", err)
	}
	if dir != "/test/override" {
		t.Errorf("GetSessionDir() = %q, want /test/override", dir)
	}
}

func TestGetSessionDir_DefaultPath(t *testing.T) {
	ag := &KiroCLIAgent{}

	// Make sure env var is not set
	t.Setenv("ENTIRE_TEST_KIRO_PROJECT_DIR", "")

	dir, err := ag.GetSessionDir("/some/repo")
	if err != nil {
		t.Fatalf("GetSessionDir() error = %v", err)
	}

	// Should be an absolute path containing .kiro/projects
	if !filepath.IsAbs(dir) {
		t.Errorf("GetSessionDir() should return absolute path, got %q", dir)
	}
}

func TestReadSession(t *testing.T) {
	tempDir := t.TempDir()

	// Create a transcript file
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")
	transcriptContent := `{"type":"user","uuid":"u1","message":{"content":"hello"}}
{"type":"assistant","uuid":"a1","message":{"content":[{"type":"text","text":"hi"}]}}
`
	if err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o644); err != nil {
		t.Fatalf("failed to write transcript: %v", err)
	}

	ag := &KiroCLIAgent{}
	input := &agent.HookInput{
		SessionID:  "test-session",
		SessionRef: transcriptPath,
	}

	session, err := ag.ReadSession(input)
	if err != nil {
		t.Fatalf("ReadSession() error = %v", err)
	}

	if session.SessionID != "test-session" {
		t.Errorf("SessionID = %q, want test-session", session.SessionID)
	}
	if session.AgentName != agent.AgentNameKiro {
		t.Errorf("AgentName = %q, want %q", session.AgentName, agent.AgentNameKiro)
	}
	if len(session.NativeData) == 0 {
		t.Error("NativeData is empty")
	}
}

func TestReadSession_NoSessionRef(t *testing.T) {
	ag := &KiroCLIAgent{}
	input := &agent.HookInput{SessionID: "test-session"}

	_, err := ag.ReadSession(input)
	if err == nil {
		t.Error("ReadSession() should error when SessionRef is empty")
	}
}

func TestWriteSession(t *testing.T) {
	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")

	ag := &KiroCLIAgent{}
	session := &agent.AgentSession{
		SessionID:  "test-session",
		AgentName:  agent.AgentNameKiro,
		SessionRef: transcriptPath,
		NativeData: []byte(`{"type":"user","uuid":"u1","message":{"content":"hello"}}`),
	}

	err := ag.WriteSession(session)
	if err != nil {
		t.Fatalf("WriteSession() error = %v", err)
	}

	// Verify file was written
	data, err := os.ReadFile(transcriptPath)
	if err != nil {
		t.Fatalf("failed to read transcript: %v", err)
	}

	if string(data) != `{"type":"user","uuid":"u1","message":{"content":"hello"}}` {
		t.Errorf("transcript content mismatch, got %q", string(data))
	}
}

func TestWriteSession_Nil(t *testing.T) {
	ag := &KiroCLIAgent{}

	err := ag.WriteSession(nil)
	if err == nil {
		t.Error("WriteSession(nil) should error")
	}
}

func TestWriteSession_WrongAgent(t *testing.T) {
	ag := &KiroCLIAgent{}
	session := &agent.AgentSession{
		AgentName:  "claude-code",
		SessionRef: "/path/to/file",
		NativeData: []byte("{}"),
	}

	err := ag.WriteSession(session)
	if err == nil {
		t.Error("WriteSession() should error for wrong agent")
	}
}

func TestWriteSession_NoSessionRef(t *testing.T) {
	ag := &KiroCLIAgent{}
	session := &agent.AgentSession{
		AgentName:  agent.AgentNameKiro,
		NativeData: []byte("{}"),
	}

	err := ag.WriteSession(session)
	if err == nil {
		t.Error("WriteSession() should error when SessionRef is empty")
	}
}

func TestWriteSession_NoNativeData(t *testing.T) {
	ag := &KiroCLIAgent{}
	session := &agent.AgentSession{
		AgentName:  agent.AgentNameKiro,
		SessionRef: "/path/to/file",
	}

	err := ag.WriteSession(session)
	if err == nil {
		t.Error("WriteSession() should error when NativeData is empty")
	}
}

// Chunking tests

func TestChunkTranscript_SmallContent(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}

	content := []byte(`{"type":"user","uuid":"u1","message":{"content":"hello"}}
{"type":"assistant","uuid":"a1","message":{"content":[{"type":"text","text":"hi"}]}}
`)

	chunks, err := ag.ChunkTranscript(content, agent.MaxChunkSize)
	if err != nil {
		t.Fatalf("ChunkTranscript() error = %v", err)
	}
	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk, got %d", len(chunks))
	}
}

func TestChunkTranscript_LargeContent(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}

	// Create a JSONL transcript with many lines that exceeds maxSize
	var lines []byte
	for i := range 100 {
		line := []byte(`{"type":"user","uuid":"u` + string(rune('0'+i%10)) + `","message":{"content":"message with some content to make it larger xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}}` + "\n")
		lines = append(lines, line...)
	}

	// Use a small maxSize to force chunking
	maxSize := 5000
	chunks, err := ag.ChunkTranscript(lines, maxSize)
	if err != nil {
		t.Fatalf("ChunkTranscript() error = %v", err)
	}

	if len(chunks) < 2 {
		t.Errorf("Expected at least 2 chunks for large content, got %d", len(chunks))
	}

	// Verify reassembly gives back all lines
	reassembled, err := ag.ReassembleTranscript(chunks)
	if err != nil {
		t.Fatalf("ReassembleTranscript() error = %v", err)
	}

	// Count lines in original and reassembled
	originalLines := countLines(lines)
	reassembledLines := countLines(reassembled)

	if reassembledLines != originalLines {
		t.Errorf("Reassembled line count = %d, want %d", reassembledLines, originalLines)
	}
}

func countLines(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	count := 0
	for _, b := range data {
		if b == '\n' {
			count++
		}
	}
	return count
}

func TestChunkTranscript_EmptyContent(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}

	chunks, err := ag.ChunkTranscript([]byte{}, agent.MaxChunkSize)
	if err != nil {
		t.Fatalf("ChunkTranscript() error = %v", err)
	}
	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for empty content, got %d", len(chunks))
	}
}

func TestChunkTranscript_RoundTrip(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}

	// Create a realistic JSONL transcript
	original := []byte(`{"type":"user","uuid":"u1","message":{"content":"Write a hello world program"}}
{"type":"assistant","uuid":"a1","message":{"content":[{"type":"tool_use","name":"fs_write","input":{"file_path":"main.go"}}]}}
{"type":"user","uuid":"u2","message":{"content":"Now add a function"}}
{"type":"assistant","uuid":"a2","message":{"content":[{"type":"tool_use","name":"edit","input":{"file_path":"main.go"}}]}}
`)

	// Use small maxSize to force chunking
	maxSize := 200
	chunks, err := ag.ChunkTranscript(original, maxSize)
	if err != nil {
		t.Fatalf("ChunkTranscript() error = %v", err)
	}

	reassembled, err := ag.ReassembleTranscript(chunks)
	if err != nil {
		t.Fatalf("ReassembleTranscript() error = %v", err)
	}

	if string(reassembled) != string(original) {
		t.Errorf("Round-trip content mismatch:\ngot:  %q\nwant: %q", string(reassembled), string(original))
	}
}

func TestReassembleTranscript_SingleChunk(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}

	content := []byte(`{"type":"user","uuid":"u1","message":{"content":"hello"}}
`)
	chunks := [][]byte{content}

	result, err := ag.ReassembleTranscript(chunks)
	if err != nil {
		t.Fatalf("ReassembleTranscript() error = %v", err)
	}

	if string(result) != string(content) {
		t.Errorf("ReassembleTranscript() = %q, want %q", string(result), string(content))
	}
}

func TestReassembleTranscript_MultipleChunks(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}

	// Note: ChunkJSONL trims trailing newlines from chunks
	chunk1 := []byte(`{"type":"user","uuid":"u1","message":{"content":"hello"}}`)
	chunk2 := []byte(`{"type":"assistant","uuid":"a1","message":{"content":[{"type":"text","text":"hi"}]}}`)
	chunks := [][]byte{chunk1, chunk2}

	result, err := ag.ReassembleTranscript(chunks)
	if err != nil {
		t.Fatalf("ReassembleTranscript() error = %v", err)
	}

	// ReassembleJSONL joins chunks with newlines
	expected := string(chunk1) + "\n" + string(chunk2)
	if string(result) != expected {
		t.Errorf("ReassembleTranscript() = %q, want %q", string(result), expected)
	}
}

func TestReassembleTranscript_EmptyChunks(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}

	result, err := ag.ReassembleTranscript([][]byte{})
	if err != nil {
		t.Fatalf("ReassembleTranscript() error = %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result for empty chunks, got %q", string(result))
	}
}

func TestChunkTranscript_SingleOversizedLine(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}

	// Create a single line that exceeds maxSize
	largeContent := `{"type":"user","uuid":"u1","message":{"content":"` + string(make([]byte, 1000)) + `"}}` + "\n"
	content := []byte(largeContent)

	// maxSize smaller than the single line
	maxSize := 100

	// ChunkJSONL returns an error when a single line exceeds maxSize (it can't split a JSON object)
	_, err := ag.ChunkTranscript(content, maxSize)
	if err == nil {
		t.Error("ChunkTranscript() should error when a single line exceeds maxSize")
	}
}

func TestChunkTranscript_PreservesLineOrder(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}

	// Create lines with numbered content to verify order
	var lines []byte
	for i := range 20 {
		line := []byte(`{"type":"user","uuid":"u` + string(rune('A'+i)) + `","message":{"content":"message-` + string(rune('A'+i)) + `"}}` + "\n")
		lines = append(lines, line...)
	}

	// Small maxSize to force multiple chunks
	chunks, err := ag.ChunkTranscript(lines, 200)
	if err != nil {
		t.Fatalf("ChunkTranscript() error = %v", err)
	}

	reassembled, err := ag.ReassembleTranscript(chunks)
	if err != nil {
		t.Fatalf("ReassembleTranscript() error = %v", err)
	}

	// Verify content is identical
	if string(reassembled) != string(lines) {
		t.Error("Line order not preserved after chunking and reassembly")
	}
}

func TestReadTranscript(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	// Create a transcript file
	transcriptPath := filepath.Join(tempDir, "session.jsonl")
	transcriptContent := `{"type":"user","uuid":"u1","message":{"content":"hello"}}
{"type":"assistant","uuid":"a1","message":{"content":[{"type":"text","text":"hi"}]}}
`
	if err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o644); err != nil {
		t.Fatalf("failed to write transcript: %v", err)
	}

	ag := &KiroCLIAgent{}
	data, err := ag.ReadTranscript(transcriptPath)
	if err != nil {
		t.Fatalf("ReadTranscript() error = %v", err)
	}

	if string(data) != transcriptContent {
		t.Errorf("ReadTranscript() content mismatch")
	}
}

func TestReadTranscript_NonExistent(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}

	_, err := ag.ReadTranscript("/nonexistent/path.jsonl")
	if err == nil {
		t.Error("ReadTranscript() should error for non-existent file")
	}
}
