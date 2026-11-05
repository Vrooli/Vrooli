# R-1 — Non-text usage by runner (2026-10-29)

Scan: 677 Codex rollouts (12 GB) and 241 Claude transcripts (212 sub-agent files), last 14
days. Scripts and raw output: `/tmp/claude-1000/-home-matthalloran8-Vrooli/76d970de-82d0-4c52-b559-ceadb6acbf61/scratchpad/r1/` (temporary).
**F** = fact (scan/code/docs), **H** = hypothesis. Serves SPEC AM-3.

## Codex

| Modality | Transcript evidence | Charge | AM today | Capture per turn |
|---|---|---|---|---|
| Images viewed by tools | `custom_tool_call_output.output[].type=input_image` (base64; `detail` high ×4765, original ×32) and `event_msg.item_completed` `ImageView{path}`; from `tools.view_image` inside `exec` **F**: 4787 images / 2730 turns | Inside `last_token_usage.input_tokens`, no separate counter **F**. API rule: 32px patches ×1.2, high capped at 2500 patches **F** | Kept as RawJSON including base64 (`codex.go:1039`); `item_completed` ignored **F** | `images_in{count, source=tool, detail, w×h, bytes, path}` plus `est_tokens` (estimated). Strip base64 |
| Images attached by user | `message:user` `input_image` **F**: 4 | Same | Dropped | Same, `source=user` |
| Image generation | `image_gen__imagegen` tool; 0 calls in window, 6 `image_generation_end` in older files **F** | Plan: shared limits, "3–5× faster"; credits after limits (GPT-Image-2 200/750 credits per 1M in/out) **F** | Not handled | `images_out{count, model, size, quality}`, charge unknown |
| Web search | 0 in window; 89 `web_search_call` and 657 `web_search_end` in older files **F** | API $10/1k plus tokens **F**; plan unknown | Not handled | `web_search{count}` as its own line |
| Audio | 1 `input_audio` **F** | Unknown | RawJSON only | `audio_in{count}` |
| MCP connectors | 4108 `McpToolCall` items **F** | Tokens **H** | Not parsed | `tool_kind=mcp:<server>.<tool>` |
| Credits pool | `rate_limits.credits{has_credits, unlimited, balance}`, `plan_type`, `spend_control_reached`, `individual_limit` **F** | Separate pool; balance 62500, unchanged **F** | Dropped by `codexRateLimits` **F** | Credits snapshot per turn; charge only when it moves |

Vrooli scenarios called through the shell (`image-tools` ×2512, `whisper*` ×110, `tts*` ×39) are
scenario costs, not runner meters. Tag them `tool_kind=scenario:<name>`; their cost comes from the
scenario's own records **H**.

## Claude Code

| Modality | Transcript evidence | Charge | AM today | Capture per turn |
|---|---|---|---|---|
| Images viewed (Read, screenshots, Chrome) | `image` block in `tool_result` and `toolUseResult.isImage` **F**: 94 | Input/cache tokens ≈ w×h/750, capped **F**; local jumps fit (1200×800 → 1383 vs 1280) **F** | Dropped (text only) **F** | `images_in{count, source, media_type, w×h, est_tokens}` |
| Images attached by user | `user` `image` blocks **F**: 38 | Same | Dropped | Same, `source=user` |
| Web search | 32 `WebSearch` calls; `usage.server_tool_use.web_search_requests` = 0 on all 12,620 rows **F** | API $10/1k **F**; probably a side request outside the transcript **H** | `ServerToolUseRequests` never set **F** | Count `WebSearch` calls, marked metered outside the transcript |
| Web fetch | 37 `WebFetch` calls; `toolUseResult{bytes, code, url}` **F** | No fee beyond tokens **F**; side-model summary not logged **H** | Not parsed | `web_fetch{count, bytes}`, side cost unknown |
| New usage fields | `iterations[]`, `fallback_credit` (null), `speed`, `output_tokens_details` **F** | `fallback_credit` probably records extra-usage draws **H** | Dropped | Persist raw per turn |

## OpenCode and Grok (codec code only)

- OpenCode: parts carry `tokens` and `cost`; media parts are not modelled **H**. Tag `tool_kind` from `part.tool`.
- Grok: no usage/cost/image fields (`SupportsCostTracking:false`) **F**. Everything stays unknown.

## Attribution approach (W5)

1. Per-turn `modality[]` (`text`, `image_in`, `image_out`, `web_search`, `web_fetch`, `audio_in`)
   and `tool_kind`.
2. Work charged outside the token counters gets its own line marked known/estimated/unknown: Codex
   credits, API search fees, the image-generation multiplier, scenario costs.
3. Image-input tokens are already inside input tokens. Report them as an estimated share of input,
   never as an extra cost.
4. Recommendations: flag high-resolution screenshots (Codex `detail:original`, Claude images over
   about 1.5k tokens) and repeated re-views of the same image path.

## Adjacent defects

- The Codex codec stores full base64 images in run events (RawJSON/RawOutputJSON), which bloats storage.
- 603k Codex `item_completed` frames are ignored (ImageView, McpToolCall, ContextCompaction, CommandExecution).
- Claude `ServerToolUseRequests` (`domain/run_configuration.go`) is never filled.
