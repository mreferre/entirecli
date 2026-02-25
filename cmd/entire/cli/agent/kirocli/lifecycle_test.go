package kirocli

import (
	"strings"
	"testing"

	"github.com/entireio/cli/cmd/entire/cli/agent"
)

func TestHookNames_AllExpected(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}

	names := ag.HookNames()
	if len(names) != 5 {
		t.Errorf("HookNames() returned %d names, want 5", len(names))
	}

	expected := map[string]bool{
		HookNameAgentSpawn:       false,
		HookNameUserPromptSubmit: false,
		HookNamePreToolUse:       false,
		HookNamePostToolUse:      false,
		HookNameStop:             false,
	}

	for _, name := range names {
		if _, ok := expected[name]; ok {
			expected[name] = true
		} else {
			t.Errorf("HookNames() returned unexpected name: %q", name)
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("HookNames() missing expected name: %q", name)
		}
	}
}

func TestParseHookEvent_SessionStart(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}

	input := `{"hook_event_name":"AgentSpawn","cwd":"/repo","session_id":"sess-123","transcript_path":"/home/.kiro/sessions/sess-123.jsonl"}`
	reader := strings.NewReader(input)

	event, err := ag.ParseHookEvent(HookNameAgentSpawn, reader)
	if err != nil {
		t.Fatalf("ParseHookEvent() error = %v", err)
	}

	if event == nil {
		t.Fatal("ParseHookEvent() returned nil event")
	}

	if event.Type != agent.SessionStart {
		t.Errorf("event.Type = %v, want %v", event.Type, agent.SessionStart)
	}
	if event.SessionID != "sess-123" {
		t.Errorf("event.SessionID = %q, want %q", event.SessionID, "sess-123")
	}
	if event.SessionRef != "/home/.kiro/sessions/sess-123.jsonl" {
		t.Errorf("event.SessionRef = %q, want %q", event.SessionRef, "/home/.kiro/sessions/sess-123.jsonl")
	}
	if event.Timestamp.IsZero() {
		t.Error("event.Timestamp should not be zero")
	}
}

func TestParseHookEvent_TurnStart(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}

	input := `{"hook_event_name":"UserPromptSubmit","cwd":"/repo","session_id":"sess-123","transcript_path":"/home/.kiro/sessions/sess-123.jsonl","prompt":"write a hello world"}`
	reader := strings.NewReader(input)

	event, err := ag.ParseHookEvent(HookNameUserPromptSubmit, reader)
	if err != nil {
		t.Fatalf("ParseHookEvent() error = %v", err)
	}

	if event == nil {
		t.Fatal("ParseHookEvent() returned nil event")
	}

	if event.Type != agent.TurnStart {
		t.Errorf("event.Type = %v, want %v", event.Type, agent.TurnStart)
	}
	if event.SessionID != "sess-123" {
		t.Errorf("event.SessionID = %q, want %q", event.SessionID, "sess-123")
	}
	if event.Prompt != "write a hello world" {
		t.Errorf("event.Prompt = %q, want %q", event.Prompt, "write a hello world")
	}
	if event.SessionRef != "/home/.kiro/sessions/sess-123.jsonl" {
		t.Errorf("event.SessionRef = %q, want %q", event.SessionRef, "/home/.kiro/sessions/sess-123.jsonl")
	}
}

func TestParseHookEvent_TurnEnd(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}

	input := `{"hook_event_name":"Stop","cwd":"/repo","session_id":"sess-456","transcript_path":"/home/.kiro/sessions/sess-456.jsonl"}`
	reader := strings.NewReader(input)

	event, err := ag.ParseHookEvent(HookNameStop, reader)
	if err != nil {
		t.Fatalf("ParseHookEvent() error = %v", err)
	}

	if event == nil {
		t.Fatal("ParseHookEvent() returned nil event")
	}

	if event.Type != agent.TurnEnd {
		t.Errorf("event.Type = %v, want %v", event.Type, agent.TurnEnd)
	}
	if event.SessionID != "sess-456" {
		t.Errorf("event.SessionID = %q, want %q", event.SessionID, "sess-456")
	}
}

func TestParseHookEvent_ToolHooks(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}

	tests := []struct {
		hookName string
	}{
		{HookNamePreToolUse},
		{HookNamePostToolUse},
	}

	for _, tc := range tests {
		t.Run(tc.hookName, func(t *testing.T) {
			t.Parallel()
			// Tool hooks return nil events (no lifecycle action)
			input := `{"hook_event_name":"PreToolUse","cwd":"/repo","session_id":"sess-123","transcript_path":"/home/.kiro/sessions/sess-123.jsonl","tool_name":"fs_write"}`
			reader := strings.NewReader(input)

			event, err := ag.ParseHookEvent(tc.hookName, reader)
			if err != nil {
				t.Fatalf("ParseHookEvent(%q) error = %v", tc.hookName, err)
			}

			if event != nil {
				t.Errorf("ParseHookEvent(%q) returned non-nil event, expected nil for tool hooks", tc.hookName)
			}
		})
	}
}

func TestParseHookEvent_UnknownHook(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}

	input := `{"hook_event_name":"UnknownHook","cwd":"/repo","session_id":"sess-123"}`
	reader := strings.NewReader(input)

	event, err := ag.ParseHookEvent("unknown-hook", reader)
	if err != nil {
		t.Fatalf("ParseHookEvent() error = %v", err)
	}

	if event != nil {
		t.Errorf("ParseHookEvent(unknown-hook) = %v, want nil", event)
	}
}

func TestParseHookEvent_EmptyInput(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}

	reader := strings.NewReader("")

	_, err := ag.ParseHookEvent(HookNameAgentSpawn, reader)
	if err == nil {
		t.Error("ParseHookEvent() should return error for empty input")
	}
}

func TestParseHookEvent_InvalidJSON(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}

	reader := strings.NewReader("not valid json")

	_, err := ag.ParseHookEvent(HookNameAgentSpawn, reader)
	if err == nil {
		t.Error("ParseHookEvent() should return error for invalid JSON")
	}
}
