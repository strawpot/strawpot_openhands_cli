# StrawPot OpenHands CLI

A Go wrapper that translates [StrawPot](https://github.com/strawpot) protocol arguments into [OpenHands CLI](https://github.com/OpenHands/OpenHands-CLI) flags. It acts as a pure translation layer — process management, sessions, and infrastructure are handled by StrawPot core.

## Prerequisites

- [OpenHands CLI](https://github.com/OpenHands/OpenHands-CLI) (`pip install openhands-ai`)
- An LLM API key for your chosen model provider

## Installation

```sh
curl -fsSL https://raw.githubusercontent.com/strawpot/strawpot_openhands_cli/main/strawpot_openhands/install.sh | sh
```

This downloads a pre-built binary for your platform (macOS/Linux, amd64/arm64) to `/usr/local/bin`. Override the install directory with `INSTALL_DIR`:

```sh
INSTALL_DIR=~/.local/bin curl -fsSL ... | sh
```

## Usage

The wrapper exposes two subcommands:

### `setup`

Runs `openhands login` to authenticate.

```sh
strawpot_openhands setup
```

### `build`

Translates StrawPot protocol flags into an OpenHands CLI command and outputs it as JSON.

```sh
strawpot_openhands build \
  --agent-workspace-dir /path/to/workspace \
  --working-dir /path/to/project \
  --task "fix the bug" \
  --config '{"model":"gpt-4o"}'
```

Output:

```json
{
  "cmd": ["openhands", "--headless", "--json", "-t", "fix the bug", "--override-with-envs"],
  "cwd": "/path/to/project",
  "env": {"LLM_MODEL": "gpt-4o"}
}
```

#### Build flags

| Flag | Required | Description |
|---|---|---|
| `--agent-workspace-dir` | Yes | Workspace directory for instructions and symlinks |
| `--working-dir` | No | Working directory for the command (`cwd` in output) |
| `--task` | No | Task prompt (passed as `-t`, or combined into instructions file when prompts are present) |
| `--config` | No | JSON config object (default: `{}`) |
| `--role-prompt` | No | Role prompt text (written to `instructions.md`) |
| `--memory-prompt` | No | Memory/context prompt (appended to `instructions.md`) |
| `--skills-dir` | No | Directory with skill subdirectories (symlinked to `skills/`) |
| `--roles-dir` | No | Directory with role subdirectories (repeatable, symlinked to `roles/`) |
| `--agent-id` | No | Agent identifier |

## Configuration

### Config JSON

Pass via `--config`:

| Key | Type | Default | Description |
|---|---|---|---|
| `model` | string | _(none)_ | Model override (passed via `LLM_MODEL` env var with `--override-with-envs`) |

### Environment variables

| Variable | Description |
|---|---|
| `LLM_API_KEY` | LLM API key for the configured model provider |

### Notes

- OpenHands headless mode always runs in auto-approve mode — there is no opt-out.
- Since OpenHands CLI does not support a separate `--system-prompt` flag, role/memory prompts and task are combined into a single `instructions.md` file and passed via `-f`. When only a task is provided (no prompts), it is passed directly via `-t`.
- Model selection uses the `LLM_MODEL` environment variable (returned in the `env` field of the JSON output) rather than a CLI flag.

## Development

```sh
cd openhands/wrapper
go test -v ./...
```

Releases are built with [GoReleaser](https://goreleaser.com/) and published automatically via GitHub Actions.

## License

See [LICENSE](LICENSE) for details.
