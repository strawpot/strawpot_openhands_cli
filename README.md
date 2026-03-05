# strawpot_openhands_cli

StrawPot wrapper for [OpenHands CLI](https://github.com/OpenHands/OpenHands-CLI). Translates StrawPot's agent protocol into OpenHands CLI flags.

## Overview

This wrapper provides two subcommands:

- **`setup`** — Runs `openhands login` for interactive authentication
- **`build`** — Translates StrawPot protocol args to an OpenHands CLI command, returning JSON: `{"cmd": [...], "cwd": "...", "env": {...}}`

OpenHands-specific behavior:

- Always runs in `--headless --json` mode for non-interactive automation
- Combines role/memory prompts and task into a single instructions file (`-f`) since OpenHands doesn't support separate system prompts
- Model selection via `LLM_MODEL` env var (returned in the `env` field) with `--override-with-envs`

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/strawpot/strawpot_openhands_cli/main/strawpot_openhands/install.sh | sh
```

## Development

```sh
cd openhands/wrapper
go test -v ./...
go build -o strawpot_openhands .
```

## License

MIT
