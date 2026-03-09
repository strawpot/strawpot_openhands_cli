---
name: strawpot-openhands
description: OpenHands CLI agent
metadata:
  strawpot:
    bin:
      macos: strawpot_openhands
      linux: strawpot_openhands
    install:
      macos: curl -fsSL https://raw.githubusercontent.com/strawpot/strawpot_openhands_cli/main/strawpot_openhands/install.sh | sh
      linux: curl -fsSL https://raw.githubusercontent.com/strawpot/strawpot_openhands_cli/main/strawpot_openhands/install.sh | sh
    tools:
      openhands:
        description: OpenHands CLI
        install:
          macos: pip install openhands-ai
          linux: pip install openhands-ai
    params:
      model:
        type: string
        description: Model override (passed via LLM_MODEL env var)
      dangerously_skip_permissions:
        type: boolean
        default: true
        description: Skip permission prompts (headless mode always auto-approves)
    env:
      LLM_API_KEY:
        required: false
        description: LLM API key (if needed for the configured model)
---

# OpenHands CLI Agent

Runs OpenHands CLI as a subprocess. Supports headless non-interactive mode,
custom model selection via environment variables, and skill-based prompt augmentation.
