// Package kirocli implements the Agent interface for Kiro CLI.
package kirocli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/entireio/cli/cmd/entire/cli/agent"
	"github.com/entireio/cli/cmd/entire/cli/paths"
)

//nolint:gochecknoinits // Agent self-registration is the intended pattern
func init() {
	agent.Register(agent.AgentNameKiro, NewKiroCLIAgent)
}

// KiroCLIAgent implements the Agent interface for Kiro CLI.
//
//nolint:revive // KiroCLIAgent is clearer than Agent in this context
type KiroCLIAgent struct{}

// NewKiroCLIAgent creates a new Kiro CLI agent instance.
func NewKiroCLIAgent() agent.Agent {
	return &KiroCLIAgent{}
}

// --- Identity ---

// Name returns the agent registry key.
func (k *KiroCLIAgent) Name() agent.AgentName {
	return agent.AgentNameKiro
}

// Type returns the agent type identifier.
func (k *KiroCLIAgent) Type() agent.AgentType {
	return agent.AgentTypeKiro
}

// Description returns a human-readable description.
func (k *KiroCLIAgent) Description() string {
	return "Kiro CLI - AI-powered terminal coding agent"
}

// IsPreview returns whether the agent integration is in preview.
func (k *KiroCLIAgent) IsPreview() bool {
	return true
}

// DetectPresence checks if Kiro CLI is configured in the repository.
func (k *KiroCLIAgent) DetectPresence() (bool, error) {
	// Get worktree root to check for .kiro directory
	// This is needed because the CLI may be run from a subdirectory
	repoRoot, err := paths.WorktreeRoot()
	if err != nil {
		// Not in a git repo, fall back to CWD-relative check
		repoRoot = "."
	}

	// Check for .kiro directory
	kiroDir := filepath.Join(repoRoot, ".kiro")
	if _, err := os.Stat(kiroDir); err == nil {
		return true, nil
	}
	return false, nil
}

// ProtectedDirs returns directories that Kiro uses for config/state.
func (k *KiroCLIAgent) ProtectedDirs() []string {
	return []string{".kiro"}
}

// --- Transcript Storage ---

// ReadTranscript reads the raw transcript bytes for a session.
func (k *KiroCLIAgent) ReadTranscript(sessionRef string) ([]byte, error) {
	data, err := os.ReadFile(sessionRef) //nolint:gosec // Path from agent hook
	if err != nil {
		return nil, fmt.Errorf("failed to read kiro transcript: %w", err)
	}
	return data, nil
}

// ChunkTranscript splits a JSONL transcript at line boundaries.
// Kiro CLI uses JSONL format (similar to Claude Code).
func (k *KiroCLIAgent) ChunkTranscript(content []byte, maxSize int) ([][]byte, error) {
	chunks, err := agent.ChunkJSONL(content, maxSize)
	if err != nil {
		return nil, fmt.Errorf("failed to chunk JSONL transcript: %w", err)
	}
	return chunks, nil
}

// ReassembleTranscript concatenates JSONL chunks with newlines.
func (k *KiroCLIAgent) ReassembleTranscript(chunks [][]byte) ([]byte, error) {
	return agent.ReassembleJSONL(chunks), nil
}

// --- Legacy methods ---

// GetSessionID extracts the session ID from hook input.
func (k *KiroCLIAgent) GetSessionID(input *agent.HookInput) string {
	return input.SessionID
}

// GetSessionDir returns the directory where Kiro stores session transcripts.
func (k *KiroCLIAgent) GetSessionDir(repoPath string) (string, error) {
	// Check for test environment override
	if override := os.Getenv("ENTIRE_TEST_KIRO_PROJECT_DIR"); override != "" {
		return override, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	projectDir := SanitizePathForKiro(repoPath)
	return filepath.Join(homeDir, ".kiro", "projects", projectDir), nil
}

// ResolveSessionFile returns the path to a Kiro session file.
// Kiro names files directly as <id>.jsonl (similar to Claude Code).
func (k *KiroCLIAgent) ResolveSessionFile(sessionDir, agentSessionID string) string {
	return filepath.Join(sessionDir, agentSessionID+".jsonl")
}

// ReadSession reads session data from Kiro's storage.
func (k *KiroCLIAgent) ReadSession(input *agent.HookInput) (*agent.AgentSession, error) {
	if input.SessionRef == "" {
		return nil, errors.New("session reference (transcript path) is required")
	}

	// Read the raw JSONL file
	data, err := os.ReadFile(input.SessionRef) //nolint:gosec // Path from agent hook
	if err != nil {
		return nil, fmt.Errorf("failed to read transcript: %w", err)
	}

	return &agent.AgentSession{
		SessionID:  input.SessionID,
		AgentName:  k.Name(),
		SessionRef: input.SessionRef,
		NativeData: data,
	}, nil
}

// WriteSession writes session data for resumption.
func (k *KiroCLIAgent) WriteSession(session *agent.AgentSession) error {
	if session == nil {
		return errors.New("session is nil")
	}

	// Verify this session belongs to Kiro CLI
	if session.AgentName != "" && session.AgentName != k.Name() {
		return fmt.Errorf("session belongs to agent %q, not %q", session.AgentName, k.Name())
	}

	if session.SessionRef == "" {
		return errors.New("session reference (transcript path) is required")
	}

	if len(session.NativeData) == 0 {
		return errors.New("session has no native data to write")
	}

	// Write the raw JSONL data
	if err := os.WriteFile(session.SessionRef, session.NativeData, 0o600); err != nil {
		return fmt.Errorf("failed to write transcript: %w", err)
	}

	return nil
}

// FormatResumeCommand returns the command to resume a Kiro CLI session.
func (k *KiroCLIAgent) FormatResumeCommand(sessionID string) string {
	return "kiro-cli --resume " + sessionID
}

// SanitizePathForKiro converts a path to Kiro's project directory format.
// Replaces any non-alphanumeric character with a dash.
var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]`)

// SanitizePathForKiro converts a path to a safe directory name.
func SanitizePathForKiro(path string) string {
	return nonAlphanumericRegex.ReplaceAllString(path, "-")
}
