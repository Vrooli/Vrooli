# AI Navigation Architecture

_Last reviewed: 2026-01-30_

_Lifecycle ownership rechecked: 2026-09-29_

## Overview

The AI navigation feature (also called "autopilot") enables users to control browser sessions using natural language prompts. A vision-language model observes the browser state via annotated screenshots and decides what actions to take to accomplish the user's goal.

## AI Gateway provider boundary

The Playwright vision client sends the inner multimodal decision request to
AI Gateway's `InferenceService.Run` contract. BAS supplies the intent role
(`extract.structured`), a provider-neutral route profile, the ordered
conversation turns, and inline screenshot attachments. AI Gateway and the
resource providers own credentials, concrete model selection, retries,
provider request shape, and reported token usage.

The UI exposes only two route profiles: `local_first` (local multimodal
preference with reviewed hosted fallback) and `remote_only` (hosted through
AI Gateway). BAS retains the browser-specific observe/decide/act loop, element
numbering, action parsing, loop detection, and human-intervention behavior.
Each prior user frame remains attached to its original turn, and credit
accounting consumes usage returned by AI Gateway rather than a BAS price table.

Playwright-backed navigation is the sole production engine. The API owns
navigation identity, authorization, lifecycle tracking and action recording;
playwright-driver owns the observe/decide/act loop and calls AI Gateway for
provider-neutral model inference. Text analysis also uses the shared OpenRouter
model service boundary.

## Navigator Abstraction

The API registry exposes the canonical Playwright navigator for discovery and
request validation. Production wiring registers exactly one navigator.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     VisionNavigationHandler                                 │
│                              │                                              │
│                    NavigatorRegistry.SelectNavigator()                      │
│                              │                                              │
│                              ▼                                              │
│                    PlaywrightVisionNavigator                               │
│                              │                                              │
│                              ▼                                              │
│                       playwright-driver                                    │
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
| Playwright | Yes | 2 | AI Gateway usage and entitlement policy |

### Bypass Conditions

| Condition | Description |
|-----------|-------------|
| `ai_gateway` | AI Gateway reports provider-neutral usage and route evidence |
| `entitlement` | An approved entitlement or credit policy permits execution |
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

### UI Layer

| File | Purpose |
|------|---------|
| [CODE: ui/src/domains/recording/sidebar/AutoTab.tsx] | Chat interface for AI navigation |
| [CODE: ui/src/domains/recording/ai-conversation/useAIConversation.ts] | Message history management |
| [CODE: ui/src/domains/recording/ai-navigation/useAINavigation.ts] | Start/abort/resume requests and navigation projection |
| [CODE: ui/src/domains/recording/ai-navigation/useAINavigationRuntime.ts] | Identity, cancellation, lifecycle reset and shared command/event refs |
| [CODE: ui/src/domains/recording/ai-navigation/useAINavigationEvents.ts] | Incremental WebSocket event admission |
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

## Current lifecycle ownership and recovery contract

The browser callback is admitted and normalized once by
`api/services/vision/playwright_navigator.go`. Under the session owner it
rejects terminal or duplicate steps, creates one redacted event projection,
then reuses that projection for bounded history, recording callbacks and
WebSocket fan-out. Consumers must not independently redact or reinterpret the
same callback envelope.

The UI has deliberately separate observation responsibilities:

- `navigationEvents.ts` parses the wire envelope and recovery snapshot.
- `useAINavigationEvents.ts` admits low-latency WebSocket events and fences
  stale generations.
- `useAINavigationRuntime.ts` owns the identity refs, cancellation controllers,
  reset lifecycle and one shared command/event ref contract, including the
  generation predicate and current-command state updater. It is the only owner
  that creates those refs; consumers receive the same fenced object.
- `useAINavigation.ts` sends start/abort/resume requests through the API and
  projects their server-assigned navigation identity.
- `useAINavigation.ts` composes the runtime, event path and server-owned status
  wait used after transport loss. WebSocket and
  recovery projections remain separate transport owners, but share the same
  navigation identity and generation fences.
- `useAIConversation.ts` owns conversation-message projection.
- `utils/actionDisplay.ts` is the presentation redaction boundary for action
  details; it is not a substitute for API-side redaction.

WebSocket delivery is the low-latency path. A status-wait failure is an
`observation_unavailable` transport state, not proof that navigation failed:
the navigation identity remains available for stop/recovery and a successor
cannot start until the server operation settles. A valid current-session live
step is authoritative transport recovery and returns the projection to
`navigating`, clearing only the stale observer error. Abort, replacement and
unmount cancel the outstanding status wait. Managed shutdown gives HTTP drain
and owner cleanup separate budgets; cleanup remains mandatory after a drain
deadline and active requests are force-closed so browser and sidecar owners
cannot survive the process boundary.

This boundary is source- and focused-test verified only until a fresh managed
build and qualification receipt bind it to runtime behavior.

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

- [DOC: docs/architecture/driver-interface.md] - Driver interface and navigator architecture
- [DOC: docs/architecture/recording.md] - Manual recording architecture (contrast with AI navigation)
- [DOC: docs/plans/README.md#historical-source-files] - Original implementation plan
- [DOC: docs/architecture/execution.md] - Workflow execution architecture
- [DOC: docs/research/ai-browser-automation-research.md] - Research on AI browser automation approaches
