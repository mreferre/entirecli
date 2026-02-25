package kirocli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/entireio/cli/cmd/entire/cli/agent"
	"github.com/entireio/cli/cmd/entire/cli/jsonutil"
	"github.com/entireio/cli/cmd/entire/cli/paths"
)

// Ensure KiroCLIAgent implements HookSupport
var _ agent.HookSupport = (*KiroCLIAgent)(nil)

// Kiro CLI hook names - these become subcommands under `entire hooks kiro`
const (
	HookNameAgentSpawn       = "agent-spawn"
	HookNameUserPromptSubmit = "user-prompt-submit"
	HookNamePreToolUse       = "pre-tool-use"
	HookNamePostToolUse      = "post-tool-use"
	HookNameStop             = "stop"
)

// KiroSettingsFileName is the settings file used by Kiro CLI.
const KiroSettingsFileName = "settings.json"

// metadataDenyRule blocks Kiro from reading Entire session metadata
const metadataDenyRule = "Read(./.entire/metadata/**)"

// entireHookPrefixes are command prefixes that identify Entire hooks (both old and new formats)
var entireHookPrefixes = []string{
	"entire ",
	"go run ${KIRO_PROJECT_DIR}/cmd/entire/main.go ",
}

// HookNames returns the hook verbs this agent supports.
func (k *KiroCLIAgent) HookNames() []string {
	return []string{
		HookNameAgentSpawn,
		HookNameUserPromptSubmit,
		HookNamePreToolUse,
		HookNamePostToolUse,
		HookNameStop,
	}
}

// ParseHookEvent translates a Kiro CLI hook into a normalized lifecycle Event.
// Returns nil if the hook has no lifecycle significance.
func (k *KiroCLIAgent) ParseHookEvent(hookName string, stdin io.Reader) (*agent.Event, error) {
	switch hookName {
	case HookNameAgentSpawn:
		return k.parseSessionStart(stdin)
	case HookNameUserPromptSubmit:
		return k.parseTurnStart(stdin)
	case HookNameStop:
		return k.parseTurnEnd(stdin)
	case HookNamePreToolUse, HookNamePostToolUse:
		// Tool hooks are handled outside the generic dispatcher for now.
		return nil, nil //nolint:nilnil // nil event = no lifecycle action
	default:
		return nil, nil //nolint:nilnil // Unknown hooks have no lifecycle action
	}
}

// --- Internal hook parsing functions ---

func (k *KiroCLIAgent) parseSessionStart(stdin io.Reader) (*agent.Event, error) {
	raw, err := agent.ReadAndParseHookInput[sessionInfoRaw](stdin)
	if err != nil {
		return nil, err
	}
	return &agent.Event{
		Type:       agent.SessionStart,
		SessionID:  raw.SessionID,
		SessionRef: raw.TranscriptPath,
		Timestamp:  time.Now(),
	}, nil
}

func (k *KiroCLIAgent) parseTurnStart(stdin io.Reader) (*agent.Event, error) {
	raw, err := agent.ReadAndParseHookInput[userPromptSubmitRaw](stdin)
	if err != nil {
		return nil, err
	}
	return &agent.Event{
		Type:       agent.TurnStart,
		SessionID:  raw.SessionID,
		SessionRef: raw.TranscriptPath,
		Prompt:     raw.Prompt,
		Timestamp:  time.Now(),
	}, nil
}

func (k *KiroCLIAgent) parseTurnEnd(stdin io.Reader) (*agent.Event, error) {
	raw, err := agent.ReadAndParseHookInput[sessionInfoRaw](stdin)
	if err != nil {
		return nil, err
	}
	return &agent.Event{
		Type:       agent.TurnEnd,
		SessionID:  raw.SessionID,
		SessionRef: raw.TranscriptPath,
		Timestamp:  time.Now(),
	}, nil
}

// InstallHooks installs Kiro CLI hooks in .kiro/settings.json.
// If force is true, removes existing Entire hooks before installing.
// Returns the number of hooks installed.
func (k *KiroCLIAgent) InstallHooks(localDev bool, force bool) (int, error) {
	// Use repo root instead of CWD to find .kiro directory
	repoRoot, err := paths.WorktreeRoot()
	if err != nil {
		// Fallback to CWD if not in a git repo (e.g., during tests)
		repoRoot, err = os.Getwd() //nolint:forbidigo // Intentional fallback when WorktreeRoot() fails (tests run outside git repos)
		if err != nil {
			return 0, fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	settingsPath := filepath.Join(repoRoot, ".kiro", KiroSettingsFileName)

	// Read existing settings if they exist
	var rawSettings map[string]json.RawMessage
	var rawHooks map[string]json.RawMessage
	var rawPermissions map[string]json.RawMessage

	existingData, readErr := os.ReadFile(settingsPath) //nolint:gosec // path is constructed from repo root + settings file name
	if readErr == nil {
		if err := json.Unmarshal(existingData, &rawSettings); err != nil {
			return 0, fmt.Errorf("failed to parse existing settings.json: %w", err)
		}
		if hooksRaw, ok := rawSettings["hooks"]; ok {
			if err := json.Unmarshal(hooksRaw, &rawHooks); err != nil {
				return 0, fmt.Errorf("failed to parse hooks in settings.json: %w", err)
			}
		}
		if permRaw, ok := rawSettings["permissions"]; ok {
			if err := json.Unmarshal(permRaw, &rawPermissions); err != nil {
				return 0, fmt.Errorf("failed to parse permissions in settings.json: %w", err)
			}
		}
	} else {
		rawSettings = make(map[string]json.RawMessage)
	}

	if rawHooks == nil {
		rawHooks = make(map[string]json.RawMessage)
	}
	if rawPermissions == nil {
		rawPermissions = make(map[string]json.RawMessage)
	}

	// Parse only the hook types we need to modify
	var agentSpawn, userPromptSubmit, stop []KiroHookMatcher
	parseKiroHookType(rawHooks, "AgentSpawn", &agentSpawn)
	parseKiroHookType(rawHooks, "UserPromptSubmit", &userPromptSubmit)
	parseKiroHookType(rawHooks, "Stop", &stop)

	// If force is true, remove all existing Entire hooks first
	if force {
		agentSpawn = removeEntireHooks(agentSpawn)
		userPromptSubmit = removeEntireHooks(userPromptSubmit)
		stop = removeEntireHooks(stop)
	}

	// Define hook commands
	var agentSpawnCmd, userPromptSubmitCmd, stopCmd string
	if localDev {
		agentSpawnCmd = "go run ${KIRO_PROJECT_DIR}/cmd/entire/main.go hooks kiro agent-spawn"
		userPromptSubmitCmd = "go run ${KIRO_PROJECT_DIR}/cmd/entire/main.go hooks kiro user-prompt-submit"
		stopCmd = "go run ${KIRO_PROJECT_DIR}/cmd/entire/main.go hooks kiro stop"
	} else {
		agentSpawnCmd = "entire hooks kiro agent-spawn"
		userPromptSubmitCmd = "entire hooks kiro user-prompt-submit"
		stopCmd = "entire hooks kiro stop"
	}

	count := 0

	// Add hooks if they don't exist
	if !hookCommandExists(agentSpawn, agentSpawnCmd) {
		agentSpawn = addHookToMatcher(agentSpawn, "", agentSpawnCmd)
		count++
	}
	if !hookCommandExists(userPromptSubmit, userPromptSubmitCmd) {
		userPromptSubmit = addHookToMatcher(userPromptSubmit, "", userPromptSubmitCmd)
		count++
	}
	if !hookCommandExists(stop, stopCmd) {
		stop = addHookToMatcher(stop, "", stopCmd)
		count++
	}

	// Add permissions.deny rule if not present
	permissionsChanged := false
	var denyRules []string
	if denyRaw, ok := rawPermissions["deny"]; ok {
		if err := json.Unmarshal(denyRaw, &denyRules); err != nil {
			return 0, fmt.Errorf("failed to parse permissions.deny in settings.json: %w", err)
		}
	}
	if !containsString(denyRules, metadataDenyRule) {
		denyRules = append(denyRules, metadataDenyRule)
		denyJSON, err := json.Marshal(denyRules)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal permissions.deny: %w", err)
		}
		rawPermissions["deny"] = denyJSON
		permissionsChanged = true
	}

	if count == 0 && !permissionsChanged {
		return 0, nil // All hooks and permissions already installed
	}

	// Marshal modified hook types back to rawHooks
	marshalKiroHookType(rawHooks, "AgentSpawn", agentSpawn)
	marshalKiroHookType(rawHooks, "UserPromptSubmit", userPromptSubmit)
	marshalKiroHookType(rawHooks, "Stop", stop)

	// Marshal hooks and update raw settings
	hooksJSON, err := json.Marshal(rawHooks)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal hooks: %w", err)
	}
	rawSettings["hooks"] = hooksJSON

	// Marshal permissions and update raw settings
	permJSON, err := json.Marshal(rawPermissions)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal permissions: %w", err)
	}
	rawSettings["permissions"] = permJSON

	// Write back to file
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o750); err != nil {
		return 0, fmt.Errorf("failed to create .kiro directory: %w", err)
	}

	output, err := jsonutil.MarshalIndentWithNewline(rawSettings, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, output, 0o600); err != nil {
		return 0, fmt.Errorf("failed to write settings.json: %w", err)
	}

	return count, nil
}

// UninstallHooks removes Entire hooks from Kiro CLI settings.
func (k *KiroCLIAgent) UninstallHooks() error {
	// Use repo root to find .kiro directory when run from a subdirectory
	repoRoot, err := paths.WorktreeRoot()
	if err != nil {
		repoRoot = "." // Fallback to CWD if not in a git repo
	}
	settingsPath := filepath.Join(repoRoot, ".kiro", KiroSettingsFileName)
	data, err := os.ReadFile(settingsPath) //nolint:gosec // path is constructed from repo root + fixed path
	if err != nil {
		return nil //nolint:nilerr // No settings file means nothing to uninstall
	}

	var rawSettings map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawSettings); err != nil {
		return fmt.Errorf("failed to parse settings.json: %w", err)
	}

	// rawHooks preserves unknown hook types
	var rawHooks map[string]json.RawMessage
	if hooksRaw, ok := rawSettings["hooks"]; ok {
		if err := json.Unmarshal(hooksRaw, &rawHooks); err != nil {
			return fmt.Errorf("failed to parse hooks: %w", err)
		}
	}
	if rawHooks == nil {
		rawHooks = make(map[string]json.RawMessage)
	}

	// Parse only the hook types we need to modify
	var agentSpawn, userPromptSubmit, stop []KiroHookMatcher
	parseKiroHookType(rawHooks, "AgentSpawn", &agentSpawn)
	parseKiroHookType(rawHooks, "UserPromptSubmit", &userPromptSubmit)
	parseKiroHookType(rawHooks, "Stop", &stop)

	// Remove Entire hooks from all hook types
	agentSpawn = removeEntireHooks(agentSpawn)
	userPromptSubmit = removeEntireHooks(userPromptSubmit)
	stop = removeEntireHooks(stop)

	// Marshal modified hook types back to rawHooks
	marshalKiroHookType(rawHooks, "AgentSpawn", agentSpawn)
	marshalKiroHookType(rawHooks, "UserPromptSubmit", userPromptSubmit)
	marshalKiroHookType(rawHooks, "Stop", stop)

	// Also remove the metadata deny rule from permissions
	var rawPermissions map[string]json.RawMessage
	if permRaw, ok := rawSettings["permissions"]; ok {
		if err := json.Unmarshal(permRaw, &rawPermissions); err != nil {
			// If parsing fails, just skip permissions cleanup
			rawPermissions = nil
		}
	}

	if rawPermissions != nil {
		if denyRaw, ok := rawPermissions["deny"]; ok {
			var denyRules []string
			if err := json.Unmarshal(denyRaw, &denyRules); err == nil {
				// Filter out the metadata deny rule
				filteredRules := make([]string, 0, len(denyRules))
				for _, rule := range denyRules {
					if rule != metadataDenyRule {
						filteredRules = append(filteredRules, rule)
					}
				}
				if len(filteredRules) > 0 {
					denyJSON, err := json.Marshal(filteredRules)
					if err == nil {
						rawPermissions["deny"] = denyJSON
					}
				} else {
					// Remove empty deny array
					delete(rawPermissions, "deny")
				}
			}
		}

		// If permissions is empty, remove it entirely
		if len(rawPermissions) > 0 {
			permJSON, err := json.Marshal(rawPermissions)
			if err == nil {
				rawSettings["permissions"] = permJSON
			}
		} else {
			delete(rawSettings, "permissions")
		}
	}

	// Marshal hooks back (preserving unknown hook types)
	if len(rawHooks) > 0 {
		hooksJSON, err := json.Marshal(rawHooks)
		if err != nil {
			return fmt.Errorf("failed to marshal hooks: %w", err)
		}
		rawSettings["hooks"] = hooksJSON
	} else {
		delete(rawSettings, "hooks")
	}

	// Write back
	output, err := jsonutil.MarshalIndentWithNewline(rawSettings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}
	if err := os.WriteFile(settingsPath, output, 0o600); err != nil {
		return fmt.Errorf("failed to write settings.json: %w", err)
	}
	return nil
}

// AreHooksInstalled checks if Entire hooks are installed.
func (k *KiroCLIAgent) AreHooksInstalled() bool {
	// Use repo root to find .kiro directory when run from a subdirectory
	repoRoot, err := paths.WorktreeRoot()
	if err != nil {
		repoRoot = "." // Fallback to CWD if not in a git repo
	}
	settingsPath := filepath.Join(repoRoot, ".kiro", KiroSettingsFileName)
	data, err := os.ReadFile(settingsPath) //nolint:gosec // path is constructed from repo root + fixed path
	if err != nil {
		return false
	}

	var settings KiroSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return false
	}

	// Check for at least one of our hooks (new or old format)
	return hookCommandExists(settings.Hooks.Stop, "entire hooks kiro stop") ||
		hookCommandExists(settings.Hooks.Stop, "go run ${KIRO_PROJECT_DIR}/cmd/entire/main.go hooks kiro stop")
}

// Helper functions for hook management

// parseKiroHookType parses a specific hook type from rawHooks into the target slice.
// Silently ignores parse errors (leaves target unchanged).
func parseKiroHookType(rawHooks map[string]json.RawMessage, hookType string, target *[]KiroHookMatcher) {
	if data, ok := rawHooks[hookType]; ok {
		//nolint:errcheck,gosec // Intentionally ignoring parse errors - leave target as nil/empty
		json.Unmarshal(data, target)
	}
}

// marshalKiroHookType marshals a hook type back to rawHooks.
// If the slice is empty, removes the key from rawHooks.
func marshalKiroHookType(rawHooks map[string]json.RawMessage, hookType string, matchers []KiroHookMatcher) {
	if len(matchers) == 0 {
		delete(rawHooks, hookType)
		return
	}
	data, err := json.Marshal(matchers)
	if err != nil {
		return // Silently ignore marshal errors (shouldn't happen)
	}
	rawHooks[hookType] = data
}

func hookCommandExists(matchers []KiroHookMatcher, command string) bool {
	for _, matcher := range matchers {
		for _, hook := range matcher.Hooks {
			if hook.Command == command {
				return true
			}
		}
	}
	return false
}

func addHookToMatcher(matchers []KiroHookMatcher, matcherName, command string) []KiroHookMatcher {
	entry := KiroHookEntry{
		Type:    "command",
		Command: command,
	}

	// If no matcher name, add to a matcher with empty string
	if matcherName == "" {
		for i, matcher := range matchers {
			if matcher.Matcher == "" {
				matchers[i].Hooks = append(matchers[i].Hooks, entry)
				return matchers
			}
		}
		return append(matchers, KiroHookMatcher{
			Matcher: "",
			Hooks:   []KiroHookEntry{entry},
		})
	}

	// Find or create matcher with the given name
	for i, matcher := range matchers {
		if matcher.Matcher == matcherName {
			matchers[i].Hooks = append(matchers[i].Hooks, entry)
			return matchers
		}
	}

	return append(matchers, KiroHookMatcher{
		Matcher: matcherName,
		Hooks:   []KiroHookEntry{entry},
	})
}

// isEntireHook checks if a command is an Entire hook (old or new format)
func isEntireHook(command string) bool {
	for _, prefix := range entireHookPrefixes {
		if strings.HasPrefix(command, prefix) {
			return true
		}
	}
	return false
}

// removeEntireHooks removes all Entire hooks from a list of matchers
func removeEntireHooks(matchers []KiroHookMatcher) []KiroHookMatcher {
	result := make([]KiroHookMatcher, 0, len(matchers))
	for _, matcher := range matchers {
		filteredHooks := make([]KiroHookEntry, 0, len(matcher.Hooks))
		for _, hook := range matcher.Hooks {
			if !isEntireHook(hook.Command) {
				filteredHooks = append(filteredHooks, hook)
			}
		}
		// Only keep the matcher if it has hooks remaining
		if len(filteredHooks) > 0 {
			matcher.Hooks = filteredHooks
			result = append(result, matcher)
		}
	}
	return result
}

// containsString checks if a slice contains a specific string.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
