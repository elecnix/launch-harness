# launch-harness

One command to launch any coding-agent harness — Claude Code, Codex, OpenCode, Pi, Cline, Droid, Qwen Code and more — configured to talk to **your own inference provider**.

```
harness-launch claude --model gpt-oss:20b
harness-launch codex --url http://localhost:1234
HARNESS_PROVIDER_URL=https://api.example.com harness-launch opencode
```

## What it does

`harness-launch` is a provider-agnostic extraction of Ollama's `ollama launch`
integration launcher. Instead of being wired to an Ollama server, it points each
harness at whatever endpoint you provide:

1. `--url` flag
2. `$HARNESS_PROVIDER_URL` environment variable
3. `http://127.0.0.1:11434` (default)

Per harness it knows how to:

- **discover** the harness binary (PATH, `~/.local/bin`, npm/uv/bun locations) and
  **auto-install** it when missing,
- **inject your provider** via the mechanism each harness supports:
  - env vars (`ANTHROPIC_BASE_URL`, `COPILOT_PROVIDER_BASE_URL`, inline config JSON, …),
  - config-file surgery with backup + `--restore` (Codex `config.toml` profiles,
    Droid `settings.json`, Pi `models.json`, Cline, VS Code settings, …),
  - inline CLI flags (`kimi --config '<json>'`),
- **launch** the harness as a child process with your model selected.

## Supported integrations

| Integration | Harness | Aliases |
|---|---|---|
| `claude` | Claude Code | |
| `codex` | OpenAI Codex CLI | |
| `chatgpt` | ChatGPT desktop app | `codex-app`, `codex-desktop`, `codex-gui` |
| `opencode` | OpenCode | |
| `copilot` | GitHub Copilot CLI | `copilot-cli` |
| `cline` | Cline | |
| `droid` | Factory Droid | |
| `dsh` | DeepSeek Harness | `deepseek-harness` |
| `hermes` | Hermes Agent (Nous Research) | |
| `hermes-desktop` | Hermes Desktop | |
| `kimi` | Kimi Code CLI | |
| `muse` | Muse Code | `muse-code` |
| `omp` | OMP | |
| `openclaw` | OpenClaw | `clawdbot`, `moltbot` |
| `pi` | Pi coding agent | |
| `pool` | Poolside | |
| `qwen` | Qwen Code | |
| `vscode` | VS Code | `code` |

(`claude-desktop` from the original Ollama launcher is omitted: it requires the
Ollama desktop app's embedded Anthropic-compatibility gateway.)

## Usage

```
harness-launch [INTEGRATION] [flags] [-- EXTRA_ARGS...]

Flags:
  -m, --model string   model to launch with
  -u, --url string     provider base URL (or $HARNESS_PROVIDER_URL)
  -y, --yes            auto-approve confirmation prompts
      --restore        restore the integration's original configuration
      --config         write the integration configuration without launching
  -v, --version        print the version
```

Examples:

```bash
# Claude Code against a local server
harness-launch claude --model qwen3-coder

# Codex against LM Studio, passing extra flags through to codex
harness-launch codex --url http://localhost:1234 -- --sandbox workspace-write

# Persist configuration only (no launch)
harness-launch droid --model glm-5.2 --config

# Undo config changes made by a previous launch
harness-launch codex --restore

# Everything through the environment
export HARNESS_PROVIDER_URL=http://localhost:8080
harness-launch opencode
```

Backups of any modified config files are kept in `~/.launch-harness/backup/`.

## Building

Requires Go 1.24+.

```bash
go build .
go test ./...
```

## Origin and license

This project is extracted from [ollama/ollama](https://github.com/ollama/ollama)
(`cmd/launch`), stripped of its server dependencies and reworked to target
arbitrary providers. The integration adapters, install discovery, backup/restore
machinery and per-harness configuration knowledge come from that codebase; the
server coupling, model inventory, cloud catalog and TUI layers were removed.

MIT licensed — see [LICENSE](LICENSE). Copyright for the original code belongs
to Ollama.
