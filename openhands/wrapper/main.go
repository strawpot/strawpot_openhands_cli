// OpenHands CLI wrapper — translates StrawPot protocol to OpenHands CLI.
//
// This wrapper is a pure translation layer: it maps StrawPot protocol args
// to "openhands" CLI flags.  It does NOT manage processes, sessions, or any
// infrastructure — that is handled by WrapperRuntime in StrawPot core.
//
// Subcommands: setup, build
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: wrapper <setup|build> [args...]")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "setup":
		cmdSetup()
	case "build":
		cmdBuild(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", os.Args[1])
		os.Exit(1)
	}
}

// ---------------------------------------------------------------------------
// setup
// ---------------------------------------------------------------------------

func cmdSetup() {
	openhandsPath, err := exec.LookPath("openhands")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: openhands CLI not found on PATH.")
		fmt.Fprintln(os.Stderr, "Install it with: pip install openhands-ai")
		os.Exit(1)
	}

	cmd := exec.Command(openhandsPath, "login")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}

// ---------------------------------------------------------------------------
// build
// ---------------------------------------------------------------------------

type buildArgs struct {
	AgentID           string
	WorkingDir        string
	AgentWorkspaceDir string
	RolePrompt        string
	MemoryPrompt      string
	Task              string
	Config            string
	SkillsDir         string
	RolesDirs         []string
}

func parseBuildArgs(args []string) buildArgs {
	var ba buildArgs
	ba.Config = "{}"

	for i := 0; i < len(args); i++ {
		if i+1 >= len(args) {
			break
		}
		switch args[i] {
		case "--agent-id":
			i++
			ba.AgentID = args[i]
		case "--working-dir":
			i++
			ba.WorkingDir = args[i]
		case "--agent-workspace-dir":
			i++
			ba.AgentWorkspaceDir = args[i]
		case "--role-prompt":
			i++
			ba.RolePrompt = args[i]
		case "--memory-prompt":
			i++
			ba.MemoryPrompt = args[i]
		case "--task":
			i++
			ba.Task = args[i]
		case "--config":
			i++
			ba.Config = args[i]
		case "--skills-dir":
			i++
			ba.SkillsDir = args[i]
		case "--roles-dir":
			i++
			ba.RolesDirs = append(ba.RolesDirs, args[i])
		}
	}
	return ba
}

// symlink creates a symlink from dst pointing to src.
func symlink(src, dst string) error {
	return os.Symlink(src, dst)
}

func cmdBuild(args []string) {
	ba := parseBuildArgs(args)

	// Parse config JSON
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(ba.Config), &config); err != nil {
		config = map[string]interface{}{}
	}

	// Validate required args
	if ba.AgentWorkspaceDir == "" {
		fmt.Fprintln(os.Stderr, "Error: --agent-workspace-dir is required")
		os.Exit(1)
	}

	if err := os.MkdirAll(ba.AgentWorkspaceDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create workspace dir: %v\n", err)
		os.Exit(1)
	}

	// Build instructions file content from role + memory prompts.
	var promptParts []string
	if ba.RolePrompt != "" {
		promptParts = append(promptParts, ba.RolePrompt)
	}
	if ba.MemoryPrompt != "" {
		promptParts = append(promptParts, ba.MemoryPrompt)
	}

	hasPrompts := len(promptParts) > 0
	instructionsFile := filepath.Join(ba.AgentWorkspaceDir, "instructions.md")

	// When both prompts and task exist, combine them into one file since
	// OpenHands -f and -t are alternatives (not combinable).
	if hasPrompts && ba.Task != "" {
		promptParts = append(promptParts, ba.Task)
		if err := os.WriteFile(instructionsFile, []byte(strings.Join(promptParts, "\n\n")), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write instructions file: %v\n", err)
			os.Exit(1)
		}
	} else if hasPrompts {
		if err := os.WriteFile(instructionsFile, []byte(strings.Join(promptParts, "\n\n")), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write instructions file: %v\n", err)
			os.Exit(1)
		}
	}

	// Symlink each subdirectory in skills-dir into skills/<name>/
	if ba.SkillsDir != "" {
		entries, err := os.ReadDir(ba.SkillsDir)
		if err == nil && len(entries) > 0 {
			skillsTarget := filepath.Join(ba.AgentWorkspaceDir, "skills")
			if err := os.MkdirAll(skillsTarget, 0o755); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create skills dir: %v\n", err)
				os.Exit(1)
			}
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				src := filepath.Join(ba.SkillsDir, entry.Name())
				link := filepath.Join(skillsTarget, entry.Name())
				if err := symlink(src, link); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to link skill %s: %v\n", entry.Name(), err)
					os.Exit(1)
				}
			}
		}
	}

	// Symlink each subdirectory from each roles-dir into roles/<name>/
	for _, rolesDir := range ba.RolesDirs {
		if rolesDir == "" {
			continue
		}
		entries, err := os.ReadDir(rolesDir)
		if err != nil || len(entries) == 0 {
			continue
		}
		rolesTarget := filepath.Join(ba.AgentWorkspaceDir, "roles")
		if err := os.MkdirAll(rolesTarget, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create roles dir: %v\n", err)
			os.Exit(1)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			src := filepath.Join(rolesDir, entry.Name())
			link := filepath.Join(rolesTarget, entry.Name())
			// Skip if already exists (first wins)
			if _, err := os.Lstat(link); err == nil {
				continue
			}
			if err := symlink(src, link); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to link role %s: %v\n", entry.Name(), err)
				os.Exit(1)
			}
		}
	}

	// Build openhands command
	cmd := []string{"openhands"}

	// Always headless for non-interactive use
	cmd = append(cmd, "--headless")

	// Always JSON output for machine parsing
	cmd = append(cmd, "--json")

	// Task or instructions file
	if hasPrompts {
		// Use -f (prompts written to file, task appended if present)
		cmd = append(cmd, "-f", instructionsFile)
	} else if ba.Task != "" {
		// No prompts, just task — use -t
		cmd = append(cmd, "-t", ba.Task)
	}

	// Model via env var (OpenHands has no -m flag)
	env := map[string]string{}
	if model, ok := config["model"].(string); ok && model != "" {
		env["LLM_MODEL"] = model
		cmd = append(cmd, "--override-with-envs")
	}

	// Output JSON
	result := map[string]interface{}{
		"cmd": cmd,
		"cwd": ba.WorkingDir,
	}
	if len(env) > 0 {
		result["env"] = env
	}

	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to encode JSON: %v\n", err)
		os.Exit(1)
	}
}
