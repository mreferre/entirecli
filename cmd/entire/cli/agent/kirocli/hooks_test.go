package kirocli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/entireio/cli/cmd/entire/cli/agent"
)

// Compile-time check
var _ agent.HookSupport = (*KiroCLIAgent)(nil)

// Note: Hook tests cannot use t.Parallel() because t.Chdir() modifies process state.

func TestInstallHooks_FreshInstall(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	ag := &KiroCLIAgent{}

	count, err := ag.InstallHooks(false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 hooks installed, got %d", count)
	}

	// Verify settings file was created
	settingsPath := filepath.Join(dir, ".kiro", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("settings file not created: %v", err)
	}

	content := string(data)
	// Check for production hook commands
	if !strings.Contains(content, "entire hooks kiro stop") {
		t.Error("settings file does not contain stop hook")
	}
	if !strings.Contains(content, "entire hooks kiro agent-spawn") {
		t.Error("settings file does not contain agent-spawn hook")
	}
	if !strings.Contains(content, "entire hooks kiro user-prompt-submit") {
		t.Error("settings file does not contain user-prompt-submit hook")
	}
	// Should use production command
	if strings.Contains(content, "go run") {
		t.Error("settings file contains 'go run' in production mode")
	}

	// Verify JSON structure is valid
	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Errorf("settings file is not valid JSON: %v", err)
	}
	if _, ok := settings["hooks"]; !ok {
		t.Error("settings file missing 'hooks' key")
	}
	if _, ok := settings["permissions"]; !ok {
		t.Error("settings file missing 'permissions' key")
	}
}

func TestInstallHooks_Idempotent(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	ag := &KiroCLIAgent{}

	// First install
	count1, err := ag.InstallHooks(false, false)
	if err != nil {
		t.Fatalf("first install failed: %v", err)
	}
	if count1 != 3 {
		t.Errorf("first install: expected 3, got %d", count1)
	}

	// Second install — should be idempotent
	count2, err := ag.InstallHooks(false, false)
	if err != nil {
		t.Fatalf("second install failed: %v", err)
	}
	if count2 != 0 {
		t.Errorf("second install: expected 0 (idempotent), got %d", count2)
	}
}

func TestInstallHooks_LocalDev(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	ag := &KiroCLIAgent{}

	count, err := ag.InstallHooks(true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 hooks installed, got %d", count)
	}

	settingsPath := filepath.Join(dir, ".kiro", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("settings file not created: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "go run ${KIRO_PROJECT_DIR}/cmd/entire/main.go") {
		t.Error("local dev mode: settings file should contain 'go run' command")
	}
	if !strings.Contains(content, "hooks kiro stop") {
		t.Error("local dev mode: settings file should contain 'hooks kiro stop'")
	}
}

func TestInstallHooks_ForceReinstall(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	ag := &KiroCLIAgent{}

	// First install
	if _, err := ag.InstallHooks(false, false); err != nil {
		t.Fatalf("first install failed: %v", err)
	}

	// Force reinstall
	count, err := ag.InstallHooks(false, true)
	if err != nil {
		t.Fatalf("force install failed: %v", err)
	}
	if count != 3 {
		t.Errorf("force install: expected 3, got %d", count)
	}
}

func TestInstallHooks_PreservesExistingSettings(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	// Create existing settings with custom hooks
	kiroDir := filepath.Join(dir, ".kiro")
	if err := os.MkdirAll(kiroDir, 0o750); err != nil {
		t.Fatalf("failed to create .kiro dir: %v", err)
	}

	existingSettings := `{
  "someOtherKey": "someValue",
  "hooks": {
    "CustomHook": [{"matcher": "", "hooks": [{"type": "command", "command": "my-custom-hook"}]}]
  }
}
`
	if err := os.WriteFile(filepath.Join(kiroDir, "settings.json"), []byte(existingSettings), 0o600); err != nil {
		t.Fatalf("failed to write existing settings: %v", err)
	}

	ag := &KiroCLIAgent{}
	_, err := ag.InstallHooks(false, false)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(kiroDir, "settings.json"))
	if err != nil {
		t.Fatalf("failed to read settings: %v", err)
	}

	content := string(data)
	// Should preserve existing settings
	if !strings.Contains(content, "someOtherKey") {
		t.Error("existing settings key was lost")
	}
	if !strings.Contains(content, "CustomHook") {
		t.Error("existing custom hook was lost")
	}
	if !strings.Contains(content, "my-custom-hook") {
		t.Error("existing custom hook command was lost")
	}
	// Should add Entire hooks
	if !strings.Contains(content, "entire hooks kiro stop") {
		t.Error("stop hook was not added")
	}
}

func TestUninstallHooks(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	ag := &KiroCLIAgent{}

	if _, err := ag.InstallHooks(false, false); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	if err := ag.UninstallHooks(); err != nil {
		t.Fatalf("uninstall failed: %v", err)
	}

	settingsPath := filepath.Join(dir, ".kiro", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("settings file should still exist: %v", err)
	}

	content := string(data)
	if strings.Contains(content, "entire hooks kiro") {
		t.Error("Entire hooks still present after uninstall")
	}
}

func TestUninstallHooks_PreservesOtherHooks(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	// Create settings with both Entire and custom hooks
	kiroDir := filepath.Join(dir, ".kiro")
	if err := os.MkdirAll(kiroDir, 0o750); err != nil {
		t.Fatalf("failed to create .kiro dir: %v", err)
	}

	settingsWithMixed := `{
  "hooks": {
    "Stop": [{"matcher": "", "hooks": [
      {"type": "command", "command": "entire hooks kiro stop"},
      {"type": "command", "command": "my-custom-stop-hook"}
    ]}]
  }
}
`
	if err := os.WriteFile(filepath.Join(kiroDir, "settings.json"), []byte(settingsWithMixed), 0o600); err != nil {
		t.Fatalf("failed to write settings: %v", err)
	}

	ag := &KiroCLIAgent{}
	if err := ag.UninstallHooks(); err != nil {
		t.Fatalf("uninstall failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(kiroDir, "settings.json"))
	if err != nil {
		t.Fatalf("failed to read settings: %v", err)
	}

	content := string(data)
	if strings.Contains(content, "entire hooks kiro") {
		t.Error("Entire hooks still present after uninstall")
	}
	if !strings.Contains(content, "my-custom-stop-hook") {
		t.Error("custom hook was incorrectly removed during uninstall")
	}
}

func TestUninstallHooks_NoFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	ag := &KiroCLIAgent{}

	// Should not error when no settings file exists
	if err := ag.UninstallHooks(); err != nil {
		t.Fatalf("uninstall with no file should not error: %v", err)
	}
}

func TestAreHooksInstalled(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	ag := &KiroCLIAgent{}

	if ag.AreHooksInstalled() {
		t.Error("hooks should not be installed initially")
	}

	if _, err := ag.InstallHooks(false, false); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	if !ag.AreHooksInstalled() {
		t.Error("hooks should be installed after InstallHooks")
	}

	if err := ag.UninstallHooks(); err != nil {
		t.Fatalf("uninstall failed: %v", err)
	}

	if ag.AreHooksInstalled() {
		t.Error("hooks should not be installed after UninstallHooks")
	}
}

func TestAreHooksInstalled_LocalDev(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	ag := &KiroCLIAgent{}

	// Install with localDev=true
	if _, err := ag.InstallHooks(true, false); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	// Should still detect hooks installed (checks for both prod and dev formats)
	if !ag.AreHooksInstalled() {
		t.Error("hooks should be detected even when installed in local dev mode")
	}
}

func TestHookNames(t *testing.T) {
	t.Parallel()
	ag := &KiroCLIAgent{}

	names := ag.HookNames()
	if len(names) != 5 {
		t.Errorf("expected 5 hook names, got %d", len(names))
	}

	expected := []string{
		HookNameAgentSpawn,
		HookNameUserPromptSubmit,
		HookNamePreToolUse,
		HookNamePostToolUse,
		HookNameStop,
	}

	for _, exp := range expected {
		found := false
		for _, name := range names {
			if name == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected hook name %q not found", exp)
		}
	}
}

func TestIsEntireHook(t *testing.T) {
	t.Parallel()

	tests := []struct {
		command  string
		expected bool
	}{
		{"entire hooks kiro stop", true},
		{"entire rewind kiro", true},
		{"go run ${KIRO_PROJECT_DIR}/cmd/entire/main.go hooks kiro stop", true},
		{"my-custom-hook", false},
		{"other-command --flag", false},
		{"", false},
	}

	for _, tc := range tests {
		if got := isEntireHook(tc.command); got != tc.expected {
			t.Errorf("isEntireHook(%q) = %v, want %v", tc.command, got, tc.expected)
		}
	}
}
