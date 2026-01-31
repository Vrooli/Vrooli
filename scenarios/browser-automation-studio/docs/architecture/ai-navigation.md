# AI Navigation Architecture

_Last reviewed: 2026-01-30_

## Overview

The AI navigation feature (also called "autopilot") enables users to control browser sessions using natural language prompts. A vision-language model observes the browser state via annotated screenshots and decides what actions to take to accomplish the user's goal.

## Navigator Abstraction

The AI navigation system uses a pluggable navigator architecture that allows multiple navigation backends with different capabilities, credit policies, and client source restrictions.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     VisionNavigationHandler                                 │
│                              │                                              │
│                    NavigatorRegistry.SelectNavigator()                      │
│                              │                                              │
│           ┌──────────────────┼──────────────────┐                           │
│           ▼                                     ▼                           │
│  PlaywrightVisionNavigator          ClaudeCodeVisionNavigator               │
│  (UI, CLI, API)                     (CLI only, future)                      │
│           │                                     │                           │
│           ▼                                     ▼                           │
│  playwright-driver                  claude CLI --chrome                     │
└─────────────────────────────────────────────────────────────────────────────┘
```

### VisionNavigator Interface

Each navigator implements the `VisionNavigator` interface:

```go
type VisionNavigator interface {
    Navigate(ctx context.Context, req NavigationRequest) (NavigationHandle, error)
    CreditPolicy() CreditPolicy
    ClientSourcePolicy() ClientSourcePolicy
    Type() NavigatorType
    IsAvailable(ctx context.Context) bool
    Description() string
    UnavailableReason(ctx context.Context) string
}
```

### Available Navigators

| Navigator | Status | Description | Allowed Sources |
|-----------|--------|-------------|-----------------|
| `playwright` | Available | Vision navigation via playwright-driver | UI, CLI, API |
| `claude_code` | Stub (future) | Navigation via Claude Code CLI with Chrome | CLI only |

## Visual Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              UI LAYER (React)                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌──────────────────┐    ┌────────────────────┐    ┌──────────────────────┐ │
│  │    AutoTab.tsx   │───▶│ useAIConversation  │───▶│   useAINavigation    │ │
│  │  (Chat Interface)│    │ (Message History)  │    │  (State + API Calls) │ │
│  └──────────────────┘    └────────────────────┘    └──────────┬───────────┘ │
│         ▲                                                      │            │
│         │                    WebSocket Events                  │            │
│         └──────────────────────────────────────────────────────┼────────────│
└─────────────────────────────────────────────────────────────────────────────┘
                                                                 │
                              HTTP POST /api/v1/ai-navigate      │
                                                                 ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           API LAYER (Go Backend)                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌────────────────────────────┐      ┌────────────────────────────────────┐ │
│  │  VisionNavigationHandler   │      │        Credit Service              │ │
│  │  - Selects navigator       │◀────▶│  - Policy-based checking           │ │
│  │  - Validates entitlements  │      │  - Per-step charging               │ │
│  │  - Broadcasts WebSocket    │      └────────────────────────────────────┘ │
│  └─────────────┬──────────────┘                                             │
│                │                                                            │
│    ┌───────────┴───────────┐                                                │
│    ▼                       ▼                                                │
│  NavigatorRegistry    CreditPolicy                                          │
│  - SelectNavigator()  - ShouldChargeCredits()                               │
│  - ListNavigators()   - BypassConditions                                    │
│                                                                             │
│                │  Forward to driver         Callback from driver            │
│                ▼                                    ▲                       │
│  ┌────────────────────────┐         ┌───────────────┴────────────────────┐  │
│  │ POST /session/:id/     │         │ POST /api/v1/internal/ai-navigate/ │  │
│  │      ai-navigate       │         │           callback                 │  │
│  └────────────────────────┘         └────────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
                                                                 ▲
                              HTTP Forward                       │ Callbacks
                                     │                           │
                                     ▼                           │
┌─────────────────────────────────────────────────────────────────────────────┐
│                      PLAYWRIGHT DRIVER (Node.js)                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                         Vision Agent                                  │   │
│  │  ┌────────────────────────────────────────────────────────────────┐  │   │
│  │  │                    Observe-Decide-Act Loop                     │  │   │
│  │  │                                                                │  │   │
│  │  │   ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐    │  │   │
│  │  │   │ OBSERVE │───▶│ DECIDE  │───▶│   ACT   │───▶│  EMIT   │────│  │   │
│  │  │   │         │    │         │    │         │    │         │    │  │   │
│  │  │   │Screenshot│   │ Vision  │    │Execute  │    │Callback │    │  │   │
│  │  │   │+Elements │   │ Model   │    │ Action  │    │  POST   │    │  │   │
│  │  │   └─────────┘    └─────────┘    └─────────┘    └─────────┘    │  │   │
│  │  │        ▲                                            │         │  │   │
│  │  │        └────────────────────────────────────────────┘         │  │   │
│  │  │                      Loop until goal or max steps             │  │   │
│  │  └────────────────────────────────────────────────────────────────┘  │   │
│  │                                                                       │   │
│  │  Supporting Components:                                               │   │
│  │  - Screenshot Capture    - Element Annotator                          │   │
│  │  - Action Executor       - Loop Detection                             │   │
│  │  - CAPTCHA Detection     - Callback Emitter                           │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│                                    │                                        │
│                                    ▼                                        │
│                          ┌─────────────────┐                                │
│                          │  Playwright     │                                │
│                          │  Browser Page   │                                │
│                          └─────────────────┘                                │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Credit Policies

Each navigator declares its own credit policy, which the handler checks before execution.

### CreditPolicy Structure

```go
type CreditPolicy struct {
    RequiresCredits  bool
    OperationType    credits.OperationType
    PerStepCharging  bool
    CreditsPerStep   int
    BypassConditions []BypassCondition
}
```

### Navigator Credit Policies

| Navigator | RequiresCredits | CreditsPerStep | Bypass Conditions |
|-----------|-----------------|----------------|-------------------|
| Playwright | Yes | 2 | `byok`, `resource_openrouter` |
| ClaudeCode | No | 0 | `local_execution` |

### Bypass Conditions

| Condition | Description |
|-----------|-------------|
| `byok` | User provided their own API key (Bring Your Own Key) |
| `resource_openrouter` | Using resource openrouter (server-provided key) |
| `local_execution` | Running locally without external API calls |

## Client Source Restriction

The system tracks client sources via the `X-Client-Source` header to restrict certain navigators.

### ClientSource Values

| Source | Description | Header Value |
|--------|-------------|--------------|
| `ui` | Web UI client | `ui` |
| `cli` | Command-line interface | `cli` |
| `api` | Direct API call | `api` (default) |

### Navigator Allowed Sources

| Navigator | Allowed Sources |
|-----------|-----------------|
| Playwright | All (UI, CLI, API) |
| ClaudeCode | CLI only |

## Data Flow Sequence

```
User types prompt
       │
       ▼
┌──────────────────┐
│ 1. UI sends POST │ ──────────────────────────────────────┐
│    to /ai-navigate│                                       │
└──────────────────┘                                       │
       │                                                   │
       ▼                                                   │
┌──────────────────┐                                       │
│ 2. Handler selects│                                       │
│    navigator from │                                       │
│    registry       │                                       │
└──────────────────┘                                       │
       │                                                   │
       ▼                                                   │
┌──────────────────┐                                       │
│ 3. Checks credit │                                       │
│    policy and    │                                       │
│    entitlements  │                                       │
└──────────────────┘                                       │
       │                                                   │
       ▼                                                   │
┌──────────────────┐     ┌─────────────────────────────────┤
│ 4. Navigator     │     │                                 │
│    forwards to   │     │  Returns 202 Accepted           │
│    driver        │     │  immediately                    │
└──────────────────┘     └─────────────────────────────────┘
       │
       ▼
┌────────────────────────────────────────────────────────────────────┐
│                    VISION AGENT LOOP                                │
├────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌────────────┐   ┌────────────┐   ┌────────────┐   ┌────────────┐ │
│  │  OBSERVE   │   │   DECIDE   │   │    ACT     │   │    EMIT    │ │
│  │            │   │            │   │            │   │            │ │
│  │ -Screenshot│──▶│ -Send to   │──▶│ -Execute   │──▶│ -POST step │ │
│  │ -Annotate  │   │  vision LLM│   │  action    │   │  callback  │ │
│  │  elements  │   │ -Get next  │   │ -Wait for  │   │  to backend│ │
│  │ -Check for │   │  action    │   │  page load │   │            │ │
│  │  CAPTCHA   │   │            │   │            │   │            │ │
│  └────────────┘   └────────────┘   └────────────┘   └─────┬──────┘ │
│                                                           │        │
│        ▲                                                  │        │
│        └──────────────────────────────────────────────────┘        │
│                    (repeat until goal/max/loop/human)              │
│                                                                     │
└────────────────────────────────────────────────────────────────────┘
       │
       ▼ (Callbacks)
┌──────────────────┐
│ 5. Navigator     │
│    handles step  │
│    callbacks     │
└──────────────────┘
       │
       ▼
┌──────────────────┐
│ 6. Charges       │
│    credits per   │
│    policy        │
└──────────────────┘
       │
       ▼
┌──────────────────┐
│ 7. Broadcasts    │──────────▶  WebSocket  ──────────▶  UI updates
│    via WebSocket │                                     in real-time
└──────────────────┘
```

## Key Files

### Vision Service Package

| Layer | File | Purpose |
|-------|------|---------|
| **API** | `api/services/vision/navigator.go` | VisionNavigator interface, NavigationHandle |
| **API** | `api/services/vision/types.go` | NavigationRequest, NavigationStep, NavigationResult |
| **API** | `api/services/vision/policy.go` | CreditPolicy, ClientSourcePolicy, BypassCondition |
| **API** | `api/services/vision/registry.go` | NavigatorRegistry (discovery + selection) |
| **API** | `api/services/vision/playwright_navigator.go` | Playwright implementation |
| **API** | `api/services/vision/claudecode_navigator.go` | Claude Code stub (future) |

### UI Layer

| File | Purpose |
|------|---------|
| [CODE: ui/src/domains/recording/sidebar/AutoTab.tsx] | Chat interface for AI navigation |
| [CODE: ui/src/domains/recording/ai-conversation/useAIConversation.ts] | Message history management |
| [CODE: ui/src/domains/recording/ai-navigation/useAINavigation.ts] | Navigation state & API calls |
| [CODE: ui/src/domains/recording/ai-navigation/types.ts] | TypeScript type definitions |
| [CODE: ui/src/domains/recording/ai-navigation/HumanInterventionOverlay.tsx] | Human intervention UI |

### API Layer (Go)

| File | Purpose |
|------|---------|
| [CODE: api/handlers/ai/vision_navigation.go] | Handler using navigator registry |
| [CODE: api/handlers/handler.go] | Route registration |
| [CODE: api/main.go] | Registry initialization |

### Playwright Driver (Node.js)

| File | Purpose |
|------|---------|
| [CODE: playwright-driver/src/routes/session-ai-navigate.ts] | Route handlers |
| [CODE: playwright-driver/src/ai/vision-agent/agent.ts] | Core vision agent loop |

## Navigation State Machine

```
     ┌─────────┐
     │  idle   │
     └────┬────┘
          │ start
          ▼
   ┌────────────────┐          ┌───────────────────┐
   │   navigating   │─────────▶│  awaiting_human   │
   └───────┬────────┘  captcha └─────────┬─────────┘
           │                             │ resume
           │◀────────────────────────────┘
           │
     ┌─────┴─────┬──────────────┬──────────────┬───────────────┐
     ▼           ▼              ▼              ▼               ▼
┌─────────┐ ┌────────┐ ┌──────────────┐ ┌─────────────┐ ┌──────────┐
│completed│ │ failed │ │max_steps_    │ │loop_detected│ │ aborted  │
│         │ │        │ │reached       │ │             │ │          │
└─────────┘ └────────┘ └──────────────┘ └─────────────┘ └──────────┘
```

### Status Definitions

| Status | Description |
|--------|-------------|
| `idle` | No navigation in progress |
| `navigating` | Actively processing steps |
| `awaiting_human` | Paused for human intervention (CAPTCHA, verification) |
| `completed` | Goal achieved successfully |
| `failed` | Error occurred during navigation |
| `max_steps_reached` | Hit configured step limit |
| `loop_detected` | Agent stuck in repetitive actions |
| `aborted` | User cancelled the navigation |

## API Endpoints

### Public Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/v1/ai-navigate/navigators` | List available navigators |
| POST | `/api/v1/ai-navigate` | Start AI navigation (accepts `navigator_type`) |
| GET | `/api/v1/ai-navigate/:id/status` | Get navigation status |
| POST | `/api/v1/ai-navigate/:id/abort` | Abort navigation |
| POST | `/api/v1/ai-navigate/:id/resume` | Resume after human intervention |

### Internal Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/api/v1/internal/ai-navigate/callback` | Receive step/completion callbacks from driver |

### Playwright Driver Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/session/:id/ai-navigate` | Start vision agent |
| POST | `/session/:id/ai-navigate/abort` | Abort vision agent |
| POST | `/session/:id/ai-navigate/resume` | Resume vision agent |
| GET | `/session/:id/ai-navigate/status` | Get agent status |

## CLI Commands

The CLI provides commands for AI navigation:

```bash
# List available navigation backends
browser-automation-studio ai navigators

# Start AI navigation with specific navigator
browser-automation-studio ai navigate \
    --session abc123 \
    --prompt "Click the login button" \
    --navigator playwright
```

## WebSocket Events

| Event Type | Direction | Description |
|------------|-----------|-------------|
| `ai_navigation_step` | Server → Client | Step completed with action details |
| `ai_navigation_awaiting_human` | Server → Client | Paused for human intervention |
| `ai_navigation_resumed` | Server → Client | Resumed after human action |
| `ai_navigation_complete` | Server → Client | Navigation finished (with status) |

## Vision Agent Loop Detail

The core loop in [CODE: playwright-driver/src/ai/vision-agent/agent.ts] follows an **Observe-Decide-Act** pattern:

### 1. OBSERVE Phase

- Capture screenshot of current browser state
- Annotate interactive elements with labels/bounding boxes
- Check for CAPTCHA indicators programmatically
- Return image + element list for vision model

### 2. DECIDE Phase

- Send screenshot + prompt + element labels to vision LLM
- Model returns:
  - Next action to take (click, type, scroll, navigate, etc.)
  - Reasoning for the action
  - Whether goal is achieved
  - Whether human intervention is needed

### 3. ACT Phase

- Execute the decided action on browser
- Wait for page to settle (network idle, animations complete)
- Track action in history for loop detection

### 4. EMIT Phase

- Create NavigationStep with action details, reasoning, screenshot, tokens
- POST callback to backend
- Backend broadcasts via WebSocket to UI

## Supported Vision Models

| Model | Tier | Notes |
|-------|------|-------|
| Qwen3-VL-30B | Budget | Recommended for cost efficiency |
| GPT-4o | Standard | Recommended for accuracy |
| GPT-4o Mini | Budget | Lower cost option |
| Claude Sonnet 4 | Premium | Highest quality |

## Human Intervention

The system supports pausing for human input when:

1. **CAPTCHA detected** - Programmatic detection before screenshot
2. **Verification required** - Login, 2FA, age gates
3. **Complex interaction** - AI determines human needed
4. **Screenshot failures** - Indicating blocked content

The UI shows [CODE: ui/src/domains/recording/ai-navigation/HumanInterventionOverlay.tsx] with:
- Intervention reason and type
- "I'm Done" button to resume

## Architectural Decisions

1. **Navigator Abstraction**: Pluggable navigator pattern enables different backends with distinct policies
2. **Policy-Based Credit Gating**: Each navigator declares its own credit policy, handler enforces uniformly
3. **Client Source Restriction**: `X-Client-Source` header enables CLI-only features
4. **Async Background Processing**: Navigation runs in background (202 Accepted), events pushed via WebSocket
5. **Callback-based Progress**: Driver POSTs to callback URL, backend broadcasts via WebSocket
6. **Session Tracking**: Navigator tracks per-session state for abort/resume
7. **Human-in-the-Loop**: Automatic CAPTCHA detection + AI-requested pauses support human intervention
8. **Loop Detection**: Prevents infinite action repetition by tracking action history

## Related Documentation

- [DOC: docs/plans/vision-agent-implementation-plan.md] - Original implementation plan
- [DOC: docs/architecture/execution.md] - Workflow execution architecture
- [DOC: docs/research/ai-browser-automation-research.md] - Research on AI browser automation approaches
