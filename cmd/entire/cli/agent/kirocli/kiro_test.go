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
