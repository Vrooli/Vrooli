# Summarize Model Benchmark

Date: 2026-05-17

Purpose: record the local evidence behind the default summarize model. Current
sources identify newer candidates, but the default is chosen from installed
models that actually run on this machine.

## Local Environment

- Ollama version: `0.11.7`
- Installed relevant text models: `llama3.2:3b`, `llama3.2:1b`,
  `qwen2.5:3b`, `mistral:latest`, `qwen3:1.7b`, `qwen3:4b`,
  `deepseek-r1:8b`, `llama3.1:8b`
- Recommended but not installed: `gemma3:4b`, `gemma3n:e2b`,
  `phi4-mini:3.8b`

## Smoke Benchmark

Command shape:

```bash
curl -H 'Content-Type: application/json' \
  --data '{"text":"<representative summarize failure RCA text>","level":"SUMMARIZE_LEVEL_MODERATE","model":"<model>","timeoutSeconds":80}' \
  http://localhost:19630/vrooli.audio_tools.v1.summarize.SummarizeService/Summarize
```

| Model | Installed | Reasoning | Wall time | Result |
|---|---:|---:|---:|---|
| `llama3.2:3b` | yes | no | 1.86s | Good enough; preserved STT-vs-Ollama provider distinction, timeout issue, model-selection fix. |
| `qwen2.5:3b` | yes | no | 2.48s | Good enough but slower than `llama3.2:3b` in this smoke. |
| `llama3.2:1b` | yes | no | 13.25s | Slower and lower quality; misstated Ollama itself as a reasoning model. |
| `mistral:latest` | yes | no | 17.40s | Slower than `llama3.2:3b`; acceptable wording but not a better default. |

## Decision

Keep `llama3.2:3b` as the default fallback. It is installed, non-reasoning,
fast in the local smoke, and good enough for TTS summarization.

Do not promote `gemma3:4b`, `gemma3n:e2b`, or `phi4-mini:3.8b` until an
operator explicitly installs them and they beat `llama3.2:3b` on latency and
quality. Do not use `qwen3:*` or `deepseek-r1:*` as defaults for TTS summaries
because they are reasoning models.
