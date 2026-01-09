# AI Browser Automation Research Report

**Date**: 2025-12-26
**Purpose**: Evaluate approaches for implementing AI-driven browser automation on the Record page
**Recommendation**: Build custom vision agent in playwright-driver using OpenRouter

---

## Executive Summary

After extensive research, I've identified **three viable approaches** for implementing AI browser automation on the Record page. The key constraint is that the AI must control **the browser visible to the user**, not a separate cloud-hosted browser.

---

## Part 1: Browser-Use Ecosystem Analysis

### 1.1 Browser-Use Node SDK

**Source**: [browser-use/browser-use-node](https://github.com/browser-use/browser-use-node)

| Aspect | Finding |
|--------|---------|
| Architecture | **Cloud-only** - Tasks execute on Browser-Use infrastructure |
| Existing Browser Support | Cannot inject existing Playwright page/context |
| Local Control | Not supported - SDK is a thin client to cloud API |
| Streaming | `stream()` and `watch()` methods for step updates |

**Verdict**: Not suitable - we need local browser control.

---

### 1.2 Browser-Use Python Library

**Source**: [browser-use/browser-use](https://github.com/browser-use/browser-use)

#### Critical Architecture Change
Browser-Use has **migrated from Playwright to pure CDP** as of August 2025:

> "We decided to drop Playwright entirely and just speak the browser's native tongue: CDP."
> — [Closer to the Metal: Leaving Playwright for CDP](https://browser-use.com/posts/playwright-to-cdp)

**Why they migrated**:
- Playwright's Node.js relay introduced latency (2nd network hop)
- Thousands of CDP calls for element inspection suffered
- Edge cases difficult to resolve with Playwright abstraction

**Performance gains**:
- Massively increased speed for element extraction and screenshots
- New async reaction capabilities
- Proper cross-origin iframe support

#### Connecting to Existing Browsers

From [Real Browser docs](https://docs.browser-use.com/customize/browser/real-browser):

```python
browser = Browser(config=BrowserConfig(
    executable_path="/usr/bin/google-chrome",
    user_data_dir="~/.config/google-chrome",
    profile_directory="Default"
))
```

**Critical limitation**: "You need to fully close chrome before running" - cannot attach to already-running browser.

#### Callback Mechanisms

From [Agent Lifecycle](https://deepwiki.com/sandeepsalwan1/browser-use/2.1-agent-lifecycle):

```python
agent = Agent(
    task="...",
    llm=llm,
    browser=browser
)

# Register step callback
agent.register_new_step_callback(
    lambda state, output, step_num: record_to_timeline(state, output, step_num)
)

# Register completion callback
agent.register_done_callback(
    lambda history: finalize_recording(history)
)

history = await agent.run()
```

Available callbacks:
- `register_new_step_callback(BrowserStateSummary, AgentOutput, step_number)`
- `register_done_callback(AgentHistoryList)`
- `register_external_agent_status_raise_error_callback()`

**Verdict**: Promising for callbacks, but Python-only and can't attach to running browser.

---

### 1.3 Browser-Use Cloud API

**Source**: [Browser-Use Cloud](https://cloud.browser-use.com/)

| Feature | Status |
|---------|--------|
| Step-by-step mode | Not documented - runs full task |
| Visible to user | Runs on their infrastructure |
| Model selection | GPT-4o, Claude, Gemini, etc. |
| Streaming | Not documented |

**Verdict**: Not suitable - browser runs remotely, user can't see it.

---

### 1.4 BU-30B-A3B-Preview Model

**Source**: [Hugging Face](https://huggingface.co/browser-use/bu-30b-a3b-preview)

| Property | Value |
|----------|-------|
| Base Model | Qwen3-VL-30B-A3B-Instruct |
| Parameters | 30B total, 3B active (MoE) |
| Context | 65,536 tokens |
| Input | Screenshots + DOM (vision-language) |
| Hardware | Single GPU (~60-70GB VRAM for BF16) |
| Cost | 200 tasks per $1 |

**Running locally**:
```bash
vllm serve browser-use/bu-30b-a3b-preview \
  --max-model-len 65536 \
  --host 0.0.0.0 --port 8000
```

**Using with browser-use**:
```python
from browser_use import Agent, ChatOpenAI

llm = ChatOpenAI(
    base_url='http://localhost:8000/v1',
    model='browser-use/bu-30b-a3b-preview',
    temperature=0.6
)
agent = Agent(task="...", llm=llm)
```

**Verdict**: Excellent model, but requires 60GB+ VRAM for local hosting. Could use via [FriendliAI](https://friendli.ai/model/browser-use/bu-30b-a3b-preview) or Browser-Use Cloud.

---

## Part 2: Alternative Approaches

### 2.1 Claude Computer Use

**Source**: [Claude Docs](https://platform.claude.com/docs/en/agents-and-tools/tool-use/computer-use-tool)

Claude has a native computer use capability:

```python
tools = [
    {
        "type": "computer_20251124",  # or computer_20250124
        "name": "computer",
        "display_width_px": 1024,
        "display_height_px": 768
    }
]
```

**Supported actions**:
- `screenshot` - Capture current screen
- `mouse_move` - Move cursor to coordinates
- `left_click`, `right_click`, `double_click`
- `type` - Type text
- `key` - Press key combinations
- `scroll` - Scroll in direction
- `zoom` (Opus 4.5 only)

**How it works**:
1. Claude receives screenshot
2. Analyzes UI elements using vision
3. Outputs action with pixel coordinates
4. Your code executes the action
5. Take new screenshot, repeat

**Best practices**:
- Keep resolution at or below 1024x768 for speed/accuracy
- Prompt: "After each step, take a screenshot and evaluate if you achieved the right outcome"

**Verdict**: Strong candidate - we control the browser, Claude just decides actions.

---

### 2.2 OpenRouter Vision Models

**Source**: [OpenRouter](https://openrouter.ai/qwen/qwen3-vl-30b-a3b-instruct)

Qwen3-VL-30B-A3B-Instruct (base for BU-30B):

| Property | Value |
|----------|-------|
| Input Cost | $0.15/M tokens |
| Output Cost | $0.60/M tokens |
| Context | 262,144 tokens |
| Capabilities | GUI automation, visual coding, spatial grounding |

Other options via OpenRouter:
- GPT-4o / GPT-4o-mini (vision)
- Claude Sonnet 4 / Opus 4.5 (vision)
- Gemini 2.5 Flash/Pro (vision)

**Verdict**: Excellent for cost-effective vision-based automation.

---

### 2.3 GPT-4V/GPT-4o + Custom Agent

**Source**: [GPT-4V-Act](https://github.com/ddupont808/GPT-4V-Act)

Pattern used by multiple projects:

1. **Screenshot** the page
2. **Annotate** interactive elements with numbered labels
3. **Send** to vision model with task
4. **Parse** response for action (e.g., "click element 34")
5. **Execute** action via Playwright/CDP
6. **Repeat** until task complete

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ Screenshot  │────►│ Vision LLM  │────►│ Parse Action│
│ + Annotate  │     │ (GPT-4o)    │     │ & Execute   │
└─────────────┘     └─────────────┘     └─────────────┘
       ▲                                       │
       └───────────────────────────────────────┘
```

**Verdict**: Full control, works with any vision model.

---

## Part 3: Recommended Architecture

Given the constraints (user must see browser, integrate with Record page, timeline recording), we recommend:

### Option A: Build Custom Vision Agent in playwright-driver (Recommended)

```
┌─────────────────────────────────────────────────────────────────┐
│                         UI (Record Page)                         │
│  ┌───────────────┐  ┌─────────────────┐  ┌───────────────────┐  │
│  │ Browser Frame │  │ AI Prompt Input │  │ Timeline (live)   │  │
│  │ (streaming)   │  │ "Order chicken" │  │ * Click #login    │  │
│  └───────────────┘  └─────────────────┘  │ * Type "user..."  │  │
│                                          │ * Click #submit   │  │
│                                          └───────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
         ▲ WebSocket                    │
         │ (stream frames/actions)      │ POST /ai-navigate
         │                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                         API (Go)                                 │
│  * Validate entitlements (credits, tier, API key)               │
│  * Forward request to playwright-driver                          │
│  * Stream actions back via WebSocket                             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   playwright-driver (Node.js)                    │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                    AI Navigation Loop                       │ │
│  │                                                             │ │
│  │  1. Screenshot current page                                 │ │
│  │  2. Extract DOM + annotate elements                         │ │
│  │  3. Send to Vision Model (OpenRouter/Claude)                │ │
│  │  4. Parse action from response                              │ │
│  │  5. Execute action via Playwright                           │ │
│  │  6. Emit action to timeline (WebSocket)                     │ │
│  │  7. Check if goal achieved                                  │ │
│  │  8. Repeat until done or max steps                          │ │
│  └────────────────────────────────────────────────────────────┘ │
│                              │                                   │
│                              ▼                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │              Existing Playwright Browser                    │ │
│  │              (same one user sees streaming)                 │ │
│  └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### Why This Approach

| Benefit | Explanation |
|---------|-------------|
| **Same browser** | User sees AI navigating in real-time |
| **Timeline integration** | Each action emits to existing timeline system |
| **Model flexibility** | Claude, GPT-4o, Qwen3-VL via OpenRouter |
| **No Python dependency** | Stays in Node.js/Go stack |
| **Reuses existing code** | DOM extraction, screenshots already built |
| **Entitlement ready** | Plugs into existing credit system |

### Model Options

| Model | Via | Cost | Quality |
|-------|-----|------|---------|
| Claude Sonnet 4 | Anthropic API | $3/$15 per M tokens | Excellent |
| GPT-4o | OpenRouter | $2.50/$10 per M tokens | Excellent |
| Qwen3-VL-30B | OpenRouter | $0.15/$0.60 per M tokens | Very Good |
| GPT-4o-mini | OpenRouter | $0.15/$0.60 per M tokens | Good |
| BU-30B-A3B | FriendliAI/Self-host | Variable | Excellent (for browser tasks) |

---

### Option B: Hybrid with Browser-Use Cloud

If you want to leverage browser-use's specialized model (BU-30B):

```
playwright-driver                    Browser-Use Cloud
       │                                    │
       ├──1. Screenshot + DOM ─────────────►│
       │                                    │
       │◄──2. Next action to take ──────────┤
       │                                    │
       ├──3. Execute action locally         │
       │                                    │
       └──4. Repeat ───────────────────────►│
```

**Challenge**: Browser-Use Cloud API doesn't have a documented "get next action" endpoint - it runs full tasks. Would need to:
1. Check if they have an undocumented step-by-step API
2. Or self-host BU-30B locally (requires 60GB+ VRAM)
3. Or use Qwen3-VL-30B via OpenRouter (base model, similar capabilities)

---

## Part 4: Implementation Considerations

### 4.1 What Already Exists in browser-automation-studio

| Component | Location | Reusable? |
|-----------|----------|-----------|
| Screenshot capture | `api/handlers/ai/screenshots.go` | Yes |
| DOM extraction | `api/handlers/ai/dom.go` | Yes |
| Element annotation | `api/handlers/ai/element_extraction.go` | Yes |
| OpenRouter client | `api/services/ai/client.go` | Yes |
| Ollama integration | `api/handlers/ai/ollama_suggestions.go` | Text-only |
| WebSocket streaming | `api/websocket/` | Yes |
| Timeline recording | `api/automation/telemetry/` | Yes |
| Entitlements | `api/services/entitlement/` | Yes |

### 4.2 What Needs to Be Built

| Component | Location | Effort |
|-----------|----------|--------|
| Vision model client | `playwright-driver/src/ai/` | Medium |
| AI navigation loop | `playwright-driver/src/ai/navigator.ts` | Medium |
| Action parser | `playwright-driver/src/ai/action-parser.ts` | Low |
| Element annotator | `playwright-driver/src/ai/annotator.ts` | Medium |
| Timeline bridge | `playwright-driver/src/ai/timeline-emitter.ts` | Low |
| UI prompt input | `ui/src/domains/record/AIPromptPanel.tsx` | Low |
| Settings UI | Add to `AISettingsSection.tsx` | Low |

### 4.3 Settings & Entitlements

Add to AI Settings:
- **Model selection**: Claude / GPT-4o / Qwen3-VL / GPT-4o-mini
- **API key** (BYOK): OpenRouter key or use server-side
- **Max steps**: Limit per navigation session (default: 20)

Credit consumption:
- Deduct credits per AI navigation step or per session
- Different rates for different models

---

## Part 5: Conclusion

### Recommendation

**Build a custom vision agent in playwright-driver** using:

1. **OpenRouter** for model access (Qwen3-VL-30B or GPT-4o)
2. **Existing infrastructure** for screenshots, DOM, WebSocket
3. **Simple observe-decide-act loop** in TypeScript

This gives you:
- User sees AI navigating live
- Timeline fills out in real-time
- Model flexibility (swap easily)
- Cost control via OpenRouter
- No new dependencies (Python, browser-use)
- Integrates with existing entitlement system

### Why Not Browser-Use Directly

| Reason | Impact |
|--------|--------|
| Cloud API runs remote browser | User can't see navigation |
| Node SDK is cloud-only wrapper | Same problem |
| Python library needs Python | Adds complexity to Node.js driver |
| Can't attach to running browser | Would need to restart browser |
| CDP migration = no Playwright | Would need to rewrite driver |

### Next Steps

1. Prototype vision agent loop in playwright-driver
2. Test with Qwen3-VL-30B via OpenRouter (cheapest)
3. Add UI controls to Record page
4. Wire up entitlements and settings
5. Benchmark and tune (max steps, screenshot size, etc.)

---

## Sources

- [Browser-Use Node SDK](https://github.com/browser-use/browser-use-node)
- [Browser-Use Python Library](https://github.com/browser-use/browser-use)
- [Browser-Use CDP Migration](https://browser-use.com/posts/playwright-to-cdp)
- [Browser-Use Agent Lifecycle](https://deepwiki.com/sandeepsalwan1/browser-use/2.1-agent-lifecycle)
- [BU-30B-A3B-Preview on Hugging Face](https://huggingface.co/browser-use/bu-30b-a3b-preview)
- [Browser-Use Cloud](https://cloud.browser-use.com/)
- [Claude Computer Use Tool](https://platform.claude.com/docs/en/agents-and-tools/tool-use/computer-use-tool)
- [OpenRouter Qwen3-VL-30B](https://openrouter.ai/qwen/qwen3-vl-30b-a3b-instruct)
- [GPT-4V-Act](https://github.com/ddupont808/GPT-4V-Act)
- [GitHub Issue #457 - Playwright Context Injection](https://github.com/browser-use/browser-use/issues/457)
