package kirocli

import "encoding/json"

// KiroSettings represents the .kiro/settings.json structure
type KiroSettings struct {
	Hooks KiroHooks `json:"hooks"`
}

// KiroHooks contains the hook configurations for Kiro CLI.
// Kiro uses hooks similar to Claude Code with these event types:
// - AgentSpawn: triggered when the agent is activated (session start)
// - UserPromptSubmit: triggered when user submits a prompt
// - PreToolUse: triggered before tool execution
// - PostToolUse: triggered after tool execution
// - Stop: triggered when assistant finishes responding
type KiroHooks struct {
	AgentSpawn       []KiroHookMatcher `json:"AgentSpawn,omitempty"`
	UserPromptSubmit []KiroHookMatcher `json:"UserPromptSubmit,omitempty"`
	PreToolUse       []KiroHookMatcher `json:"PreToolUse,omitempty"`
	PostToolUse      []KiroHookMatcher `json:"PostToolUse,omitempty"`
	Stop             []KiroHookMatcher `json:"Stop,omitempty"`
}

// KiroHookMatcher matches hooks to specific patterns (e.g., tool matchers)
type KiroHookMatcher struct {
	Matcher string          `json:"matcher"`
	Hooks   []KiroHookEntry `json:"hooks"`
}

// KiroHookEntry represents a single hook command
type KiroHookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// sessionInfoRaw is the JSON structure from hook events.
// This is the common format sent to hooks via stdin.
type sessionInfoRaw struct {
	HookEventName  string `json:"hook_event_name"`
	Cwd            string `json:"cwd"`
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
}

// userPromptSubmitRaw is the JSON structure from UserPromptSubmit hooks.
// Unlike other hooks, this includes the user's prompt text.
type userPromptSubmitRaw struct {
	HookEventName  string `json:"hook_event_name"`
	Cwd            string `json:"cwd"`
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Prompt         string `json:"prompt"`
}

// toolHookInputRaw is the JSON structure from PreToolUse/PostToolUse hooks.
// Contains tool invocation details for file modification tracking.
type toolHookInputRaw struct {
	HookEventName  string          `json:"hook_event_name"`
	Cwd            string          `json:"cwd"`
	SessionID      string          `json:"session_id"`
	TranscriptPath string          `json:"transcript_path"`
	ToolName       string          `json:"tool_name"`
	ToolInput      json.RawMessage `json:"tool_input"`
	ToolResponse   json.RawMessage `json:"tool_response,omitempty"`
}

// Tool names used in Kiro CLI transcripts.
// These are the tools that create or modify files.
const (
	ToolFsWrite = "fs_write"
	ToolWrite   = "write"
	ToolEdit    = "edit"
	ToolFsEdit  = "fs_edit"
)

// FileModificationTools lists tools that create or modify files
var FileModificationTools = []string{
	ToolFsWrite,
	ToolWrite,
	ToolEdit,
	ToolFsEdit,
}
