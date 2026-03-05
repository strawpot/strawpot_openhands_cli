package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseBuildArgs_AllFlags(t *testing.T) {
	args := []string{
		"--agent-id", "agent-1",
		"--working-dir", "/work",
		"--agent-workspace-dir", "/workspace",
		"--role-prompt", "You are a helper",
		"--memory-prompt", "Remember this",
		"--task", "do something",
		"--config", `{"model":"gpt-4o"}`,
		"--skills-dir", "/skills",
		"--roles-dir", "/roles1",
		"--roles-dir", "/roles2",
	}

	ba := parseBuildArgs(args)

	if ba.AgentID != "agent-1" {
		t.Errorf("AgentID = %q, want %q", ba.AgentID, "agent-1")
	}
	if ba.WorkingDir != "/work" {
		t.Errorf("WorkingDir = %q, want %q", ba.WorkingDir, "/work")
	}
	if ba.AgentWorkspaceDir != "/workspace" {
		t.Errorf("AgentWorkspaceDir = %q, want %q", ba.AgentWorkspaceDir, "/workspace")
	}
	if ba.RolePrompt != "You are a helper" {
		t.Errorf("RolePrompt = %q, want %q", ba.RolePrompt, "You are a helper")
	}
	if ba.MemoryPrompt != "Remember this" {
		t.Errorf("MemoryPrompt = %q, want %q", ba.MemoryPrompt, "Remember this")
	}
	if ba.Task != "do something" {
		t.Errorf("Task = %q, want %q", ba.Task, "do something")
	}
	if ba.Config != `{"model":"gpt-4o"}` {
		t.Errorf("Config = %q, want %q", ba.Config, `{"model":"gpt-4o"}`)
	}
	if ba.SkillsDir != "/skills" {
		t.Errorf("SkillsDir = %q, want %q", ba.SkillsDir, "/skills")
	}
	if len(ba.RolesDirs) != 2 || ba.RolesDirs[0] != "/roles1" || ba.RolesDirs[1] != "/roles2" {
		t.Errorf("RolesDirs = %v, want [/roles1 /roles2]", ba.RolesDirs)
	}
}

func TestParseBuildArgs_Empty(t *testing.T) {
	ba := parseBuildArgs([]string{})

	if ba.Config != "{}" {
		t.Errorf("Config default = %q, want %q", ba.Config, "{}")
	}
	if ba.AgentID != "" {
		t.Errorf("AgentID = %q, want empty", ba.AgentID)
	}
	if ba.RolesDirs != nil {
		t.Errorf("RolesDirs = %v, want nil", ba.RolesDirs)
	}
}

func TestParseBuildArgs_TrailingFlag(t *testing.T) {
	// A flag without a value at the end should not panic
	ba := parseBuildArgs([]string{"--agent-id"})

	if ba.AgentID != "" {
		t.Errorf("AgentID = %q, want empty (trailing flag ignored)", ba.AgentID)
	}
}

func TestCmdBuild_MinimalArgs(t *testing.T) {
	tmpDir := t.TempDir()
	wsDir := filepath.Join(tmpDir, "workspace")

	args := []string{
		"--agent-workspace-dir", wsDir,
		"--working-dir", "/project",
	}

	output := captureBuildOutput(t, args)

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	cmd, ok := result["cmd"].([]interface{})
	if !ok {
		t.Fatal("Missing 'cmd' in output")
	}

	if len(cmd) < 1 || cmd[0] != "openhands" {
		t.Errorf("cmd should start with openhands, got %v", cmd[0])
	}
	assertContains(t, cmd, "--headless")
	assertContains(t, cmd, "--json")

	if result["cwd"] != "/project" {
		t.Errorf("cwd = %v, want /project", result["cwd"])
	}

	// No instructions file should be created when no prompts
	instructionsFile := filepath.Join(wsDir, "instructions.md")
	if _, err := os.Stat(instructionsFile); err == nil {
		t.Error("instructions.md should not be created when no prompts given")
	}

	// No env key when no model
	if _, ok := result["env"]; ok {
		t.Error("env should not be present when no model configured")
	}
}

func TestCmdBuild_TaskOnly(t *testing.T) {
	tmpDir := t.TempDir()
	wsDir := filepath.Join(tmpDir, "workspace")

	args := []string{
		"--agent-workspace-dir", wsDir,
		"--working-dir", "/project",
		"--task", "fix the bug",
	}

	output := captureBuildOutput(t, args)

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	cmd := result["cmd"].([]interface{})
	assertSequence(t, cmd, "-t", "fix the bug")

	// No instructions file when only task (no prompts)
	instructionsFile := filepath.Join(wsDir, "instructions.md")
	if _, err := os.Stat(instructionsFile); err == nil {
		t.Error("instructions.md should not be created when only task given")
	}
}

func TestCmdBuild_WithModel(t *testing.T) {
	tmpDir := t.TempDir()
	wsDir := filepath.Join(tmpDir, "workspace")

	args := []string{
		"--agent-workspace-dir", wsDir,
		"--working-dir", "/project",
		"--task", "fix the bug",
		"--config", `{"model":"gpt-4o"}`,
	}

	output := captureBuildOutput(t, args)

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	cmd := result["cmd"].([]interface{})
	assertContains(t, cmd, "--override-with-envs")

	// Check env field
	envRaw, ok := result["env"]
	if !ok {
		t.Fatal("Missing 'env' in output when model specified")
	}
	env := envRaw.(map[string]interface{})
	if env["LLM_MODEL"] != "gpt-4o" {
		t.Errorf("LLM_MODEL = %v, want gpt-4o", env["LLM_MODEL"])
	}
}

func TestCmdBuild_NoModelNoEnv(t *testing.T) {
	tmpDir := t.TempDir()
	wsDir := filepath.Join(tmpDir, "workspace")

	args := []string{
		"--agent-workspace-dir", wsDir,
		"--working-dir", "/project",
	}

	output := captureBuildOutput(t, args)

	var result map[string]interface{}
	json.Unmarshal(output, &result)

	if _, ok := result["env"]; ok {
		t.Error("env should not be present when no model configured")
	}

	cmd := result["cmd"].([]interface{})
	for _, v := range cmd {
		if v == "--override-with-envs" {
			t.Error("--override-with-envs should not be present when no model configured")
		}
	}
}

func TestCmdBuild_PromptsOnly(t *testing.T) {
	tmpDir := t.TempDir()
	wsDir := filepath.Join(tmpDir, "workspace")

	args := []string{
		"--agent-workspace-dir", wsDir,
		"--role-prompt", "You are a coder",
		"--memory-prompt", "Use Go",
	}

	output := captureBuildOutput(t, args)

	var result map[string]interface{}
	json.Unmarshal(output, &result)

	cmd := result["cmd"].([]interface{})
	instructionsFile := filepath.Join(wsDir, "instructions.md")
	assertSequence(t, cmd, "-f", instructionsFile)

	// Verify instructions.md content
	content, err := os.ReadFile(instructionsFile)
	if err != nil {
		t.Fatalf("Failed to read instructions.md: %v", err)
	}

	expected := "You are a coder\n\nUse Go"
	if string(content) != expected {
		t.Errorf("instructions.md = %q, want %q", string(content), expected)
	}

	// -t should NOT be in cmd
	for _, v := range cmd {
		if v == "-t" {
			t.Error("cmd should not contain -t when only prompts given")
		}
	}
}

func TestCmdBuild_PromptsAndTask(t *testing.T) {
	tmpDir := t.TempDir()
	wsDir := filepath.Join(tmpDir, "workspace")

	args := []string{
		"--agent-workspace-dir", wsDir,
		"--role-prompt", "You are a coder",
		"--memory-prompt", "Use Go",
		"--task", "fix the bug",
	}

	output := captureBuildOutput(t, args)

	var result map[string]interface{}
	json.Unmarshal(output, &result)

	cmd := result["cmd"].([]interface{})
	instructionsFile := filepath.Join(wsDir, "instructions.md")
	assertSequence(t, cmd, "-f", instructionsFile)

	// Verify task is appended to instructions file
	content, err := os.ReadFile(instructionsFile)
	if err != nil {
		t.Fatalf("Failed to read instructions.md: %v", err)
	}

	expected := "You are a coder\n\nUse Go\n\nfix the bug"
	if string(content) != expected {
		t.Errorf("instructions.md = %q, want %q", string(content), expected)
	}

	// -t should NOT be in cmd (task is in the file)
	for _, v := range cmd {
		if v == "-t" {
			t.Error("cmd should not contain -t when prompts and task combined in file")
		}
	}
}

func TestCmdBuild_SkillsSymlinks(t *testing.T) {
	tmpDir := t.TempDir()
	wsDir := filepath.Join(tmpDir, "workspace")
	skillsDir := filepath.Join(tmpDir, "skills")

	// Create skill directories and a file (file should be skipped)
	os.MkdirAll(filepath.Join(skillsDir, "skill-a"), 0o755)
	os.MkdirAll(filepath.Join(skillsDir, "skill-b"), 0o755)
	os.WriteFile(filepath.Join(skillsDir, "not-a-dir.txt"), []byte("skip"), 0o644)

	args := []string{
		"--agent-workspace-dir", wsDir,
		"--skills-dir", skillsDir,
	}

	captureBuildOutput(t, args)

	// Verify symlinks created for directories only
	for _, name := range []string{"skill-a", "skill-b"} {
		link := filepath.Join(wsDir, "skills", name)
		target, err := os.Readlink(link)
		if err != nil {
			t.Errorf("Symlink %s not created: %v", name, err)
			continue
		}
		expected := filepath.Join(skillsDir, name)
		if target != expected {
			t.Errorf("Symlink %s -> %q, want %q", name, target, expected)
		}
	}

	// File should not be symlinked
	notADir := filepath.Join(wsDir, "skills", "not-a-dir.txt")
	if _, err := os.Lstat(notADir); err == nil {
		t.Error("non-directory 'not-a-dir.txt' should not be symlinked")
	}
}

func TestCmdBuild_RolesSymlinks(t *testing.T) {
	tmpDir := t.TempDir()
	wsDir := filepath.Join(tmpDir, "workspace")
	rolesDir1 := filepath.Join(tmpDir, "roles1")
	rolesDir2 := filepath.Join(tmpDir, "roles2")

	os.MkdirAll(filepath.Join(rolesDir1, "role-a"), 0o755)
	os.MkdirAll(filepath.Join(rolesDir2, "role-a"), 0o755) // duplicate — should be skipped
	os.MkdirAll(filepath.Join(rolesDir2, "role-b"), 0o755)

	args := []string{
		"--agent-workspace-dir", wsDir,
		"--roles-dir", rolesDir1,
		"--roles-dir", rolesDir2,
	}

	captureBuildOutput(t, args)

	// role-a should point to rolesDir1 (first wins)
	target, err := os.Readlink(filepath.Join(wsDir, "roles", "role-a"))
	if err != nil {
		t.Fatalf("role-a symlink not created: %v", err)
	}
	if target != filepath.Join(rolesDir1, "role-a") {
		t.Errorf("role-a -> %q, want first roles-dir", target)
	}

	// role-b should point to rolesDir2
	target, err = os.Readlink(filepath.Join(wsDir, "roles", "role-b"))
	if err != nil {
		t.Fatalf("role-b symlink not created: %v", err)
	}
	if target != filepath.Join(rolesDir2, "role-b") {
		t.Errorf("role-b -> %q, want %q", target, filepath.Join(rolesDir2, "role-b"))
	}
}

// --- helpers ---

// captureBuildOutput runs cmdBuild and captures its stdout JSON output.
func captureBuildOutput(t *testing.T, args []string) []byte {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stdout = w

	cmdBuild(args)

	w.Close()
	os.Stdout = oldStdout

	var buf [4096]byte
	n, _ := r.Read(buf[:])
	r.Close()

	return buf[:n]
}

func assertContains(t *testing.T, slice []interface{}, val string) {
	t.Helper()
	for _, v := range slice {
		if v == val {
			return
		}
	}
	t.Errorf("cmd %v does not contain %q", slice, val)
}

func assertSequence(t *testing.T, slice []interface{}, key, val string) {
	t.Helper()
	for i, v := range slice {
		if v == key && i+1 < len(slice) && slice[i+1] == val {
			return
		}
	}
	t.Errorf("cmd %v does not contain %q %q in sequence", slice, key, val)
}
