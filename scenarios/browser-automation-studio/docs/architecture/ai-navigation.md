# AI Navigation Architecture

_Last reviewed: 2026-01-30_

## Overview

The AI navigation feature (also called "autopilot") enables users to control browser sessions using natural language prompts. A vision-language model observes the browser state via annotated screenshots and decides what actions to take to accomplish the user's goal.

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
│  │  - Validates entitlements  │◀────▶│  - Pre-check credits               │ │
│  │  - Tracks active sessions  │      │  - Per-step charging               │ │
│  │  - Broadcasts WebSocket    │      └────────────────────────────────────┘ │
│  └─────────────┬──────────────┘                                             │
│                │                                                            │
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
│ 2. Backend checks│                                       │
│    credits/tier  │                                       │
└──────────────────┘                                       │
       │                                                   │
       ▼                                                   │
┌──────────────────┐     ┌─────────────────────────────────┤
│ 3. Forwards to   │     │                                 │
│    playwright-   │     │  Returns 202 Accepted           │
│    driver        │     │  immediately                    │
└──────────────────┘     └─────────────────────────────────┘
       │
       ▼
┌──────────────────┐
│ 4. Driver starts │
│    Vision Agent  │
│    in background │
└──────────────────┘
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
│ 5. Backend       │
│    receives step │
│    callbacks     │
└──────────────────┘
       │
       ▼
┌──────────────────┐
│ 6. Charges       │
│    credits per   │
│    step          │
└──────────────────┘
       │
       ▼
┌──────────────────┐
│ 7. Broadcasts    │──────────▶  WebSocket  ──────────▶  UI updates
│    via WebSocket │                                     in real-time
└──────────────────┘
```

## Key Components

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
| [CODE: api/handlers/ai/vision_navigation.go] | Main handler & orchestration |
| [CODE: api/handlers/handler.go:424-446] | Route registration |

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
| POST | `/api/v1/ai-navigate` | Start AI navigation |
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

## Credit Integration

- **Pre-flight check**: Validates user can perform AI operation before starting
- **Per-step charging**: Credits deducted after each step based on token usage
- **BYOK bypass**: Users with own API keys skip credit checks
- **Operation code**: `credits.OpAIVisionNavigate`

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

1. **Async Background Processing**: Navigation runs in background (202 Accepted), events pushed via WebSocket
2. **Callback-based Progress**: Driver POSTs to callback URL, backend broadcasts via WebSocket
3. **Session Tracking**: `ActiveNavigations` map tracks per-session state for abort/resume
4. **Human-in-the-Loop**: Automatic CAPTCHA detection + AI-requested pauses support human intervention
5. **Loop Detection**: Prevents infinite action repetition by tracking action history
6. **Credit Pre-flight**: Validates ability to perform operation before starting
7. **Stateless Validation**: Each endpoint validates independently

## Related Documentation

- [DOC: docs/plans/vision-agent-implementation-plan.md] - Original implementation plan
- [DOC: docs/architecture/execution.md] - Workflow execution architecture
- [DOC: docs/research/ai-browser-automation-research.md] - Research on AI browser automation approaches
