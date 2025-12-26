# Vision Agent Implementation Plan

**Date**: 2025-12-26
**Status**: ✅ COMPLETE
**Related Research**: [AI Browser Automation Research](../research/ai-browser-automation-research.md)

> All 6 phases have been implemented. The vision-based AI navigation feature is ready for production testing.

---

## Executive Summary

This document outlines a **clean, testable architecture** for adding AI-driven browser navigation to browser-automation-studio. The design follows **screaming architecture** principles where folder structure clearly communicates intent, establishes **clear testing seams** at every boundary, and maintains **strict separation of responsibilities**.

**Key Design Decisions:**
1. Vision agent loop lives in **playwright-driver** (direct browser access, minimal latency)
2. API acts as **orchestrator** (entitlements, credits, WebSocket broadcasting)
3. **Interface-first design** enables testing without real LLM calls
4. **Model-agnostic client** supports OpenRouter, Claude, and future providers

---

## Architecture Overview

### High-Level Data Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                   UI                                         │
│  ┌─────────────────┐  ┌─────────────────┐  ┌────────────────────────────┐   │
│  │ Browser Preview │  │ AI Prompt Panel │  │ Timeline (live updates)    │   │
│  │ (WebSocket)     │  │ "Order chicken" │  │ ✓ Click "Menu"             │   │
│  └─────────────────┘  └────────┬────────┘  │ ● Type "chicken"...        │   │
│                                │           └────────────────────────────┘   │
└────────────────────────────────┼────────────────────────▲───────────────────┘
                                 │                        │
                    POST /ai-navigate                WebSocket events
                    { prompt, model }                { step, action, screenshot }
                                 │                        │
┌────────────────────────────────┼────────────────────────┼───────────────────┐
│                               API                       │                    │
│  ┌─────────────────────────────┴─────────────────────────────────────────┐  │
│  │                    VisionNavigationHandler                             │  │
│  │  1. Validate entitlements (tier must support vision)                  │  │
│  │  2. Check credit balance                                              │  │
│  │  3. Forward to playwright-driver with callback URL                    │  │
│  │  4. Return navigation_id immediately (async operation)                │  │
│  └─────────────────────────────┬─────────────────────────────────────────┘  │
│                                │                        ▲                    │
│                     POST /session/{id}/ai-navigate     │ Step callbacks      │
│                     { prompt, model, maxSteps }        │ POST /callback      │
│                                │                        │                    │
│  ┌─────────────────────────────┴────────────────────────┴────────────────┐  │
│  │                    VisionNavigationCallbackHandler                     │  │
│  │  1. Receive step event from driver                                    │  │
│  │  2. Convert to TimelineEntry                                          │  │
│  │  3. Deduct credits (tokens used)                                      │  │
│  │  4. Broadcast via WebSocket hub                                       │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         playwright-driver                                    │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                         VisionAgent                                    │  │
│  │  ┌─────────────────────────────────────────────────────────────────┐  │  │
│  │  │                    Observe → Decide → Act Loop                   │  │  │
│  │  │                                                                  │  │  │
│  │  │  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────────┐  │  │  │
│  │  │  │ Screenshot│──►│ Annotate │──►│ Vision   │──►│ Parse Action │  │  │  │
│  │  │  │ Capture   │   │ Elements │   │ Model    │   │ Response     │  │  │  │
│  │  │  └──────────┘   └──────────┘   │ Client   │   └──────┬───────┘  │  │  │
│  │  │       ▲                        └──────────┘          │          │  │  │
│  │  │       │                                              ▼          │  │  │
│  │  │  ┌────┴─────┐                              ┌──────────────────┐ │  │  │
│  │  │  │ Emit Step│◄─────────────────────────────│ Execute Action   │ │  │  │
│  │  │  │ Callback │                              │ (Playwright)     │ │  │  │
│  │  │  └──────────┘                              └──────────────────┘ │  │  │
│  │  └─────────────────────────────────────────────────────────────────┘  │  │
│  │                                │                                       │  │
│  │                                ▼                                       │  │
│  │  ┌───────────────────────────────────────────────────────────────────┐│  │
│  │  │                    Existing Playwright Browser                     ││  │
│  │  │                    (same one user sees streaming)                  ││  │
│  │  └───────────────────────────────────────────────────────────────────┘│  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Component Design

### Folder Structure (Screaming Architecture)

```
playwright-driver/src/
├── ai/                                    # ← SCREAMS "AI Navigation"
│   ├── vision-agent/                      # ← SCREAMS "Vision-Based Agent"
│   │   ├── agent.ts                       # Main orchestrator (observe-decide-act)
│   │   ├── agent.test.ts                  # Unit tests with mocked dependencies
│   │   ├── types.ts                       # NavigationConfig, NavigationStep, etc.
│   │   └── config.ts                      # VisionAgentConfig (max steps, timeouts)
│   │
│   ├── vision-client/                     # ← SCREAMS "Vision Model Client"
│   │   ├── types.ts                       # VisionModelClient interface
│   │   ├── factory.ts                     # createVisionClient(model, apiKey)
│   │   ├── openrouter.ts                  # OpenRouter implementation
│   │   ├── claude-computer-use.ts         # Claude Computer Use implementation
│   │   ├── model-registry.ts              # Model specs (costs, limits, capabilities)
│   │   ├── openrouter.test.ts
│   │   └── mock.ts                        # MockVisionClient for testing
│   │
│   ├── screenshot/                        # ← SCREAMS "Screenshot Operations"
│   │   ├── capture.ts                     # captureScreenshot(page): Buffer
│   │   ├── annotate.ts                    # annotateWithLabels(img, elements): AnnotatedImage
│   │   ├── types.ts                       # ElementLabel, AnnotatedImage
│   │   └── annotate.test.ts
│   │
│   ├── action/                            # ← SCREAMS "Action Parsing & Execution"
│   │   ├── types.ts                       # BrowserAction union type
│   │   ├── parser.ts                      # parseLLMResponse(text): BrowserAction
│   │   ├── executor.ts                    # executeAction(page, action): StepOutcome
│   │   ├── parser.test.ts
│   │   └── executor.test.ts
│   │
│   └── emitter/                           # ← SCREAMS "Event Emission"
│       ├── types.ts                       # NavigationStepEvent
│       ├── callback-emitter.ts            # Emit via HTTP callback to API
│       └── callback-emitter.test.ts
│
├── handlers/                              # Existing handlers
│   ├── session.ts
│   ├── record.ts
│   └── ai-navigation.ts                   # NEW: Handle /session/:id/ai-navigate
│
└── server.ts                              # Existing server
```

```
api/
├── handlers/
│   ├── ai/
│   │   ├── ai_analysis.go                 # Existing DOM-based analysis
│   │   ├── vision_navigation.go           # NEW: POST /api/v1/ai-navigate
│   │   └── vision_callback.go             # NEW: POST /api/v1/internal/vision-callback
│   └── ...
│
├── services/
│   ├── ai/
│   │   ├── client.go                      # Existing OpenRouter/Ollama client
│   │   ├── vision_models.go               # NEW: Model registry + cost calculation
│   │   └── vision_orchestrator.go         # NEW: Coordinate driver + credits
│   └── entitlement/
│       └── ai_credits.go                  # Existing - extend for vision
│
└── websocket/
    └── hub.go                             # Existing - add vision event types
```

```
ui/src/domains/record/
├── AINavigationPanel/                     # NEW
│   ├── AINavigationPanel.tsx              # Prompt input + model selector + controls
│   ├── AINavigationPanel.module.css
│   ├── useAINavigation.ts                 # Hook: submit, abort, status
│   └── types.ts
│
├── Timeline/                              # Existing - extend
│   └── AINavigationStep.tsx               # NEW: Render AI steps with reasoning
│
└── ...
```

---

## Data Contracts

### TypeScript Interfaces (playwright-driver)

```typescript
// ═══════════════════════════════════════════════════════════════════════════
// ai/vision-client/types.ts - Vision Model Client Interface
// ═══════════════════════════════════════════════════════════════════════════

/**
 * Interface for vision model clients.
 * Testing seam: Mock this interface for unit tests.
 */
export interface VisionModelClient {
  /**
   * Analyze a screenshot and determine the next browser action.
   */
  analyze(request: VisionAnalysisRequest): Promise<VisionAnalysisResponse>;

  /**
   * Get model metadata for cost calculation.
   */
  getModelSpec(): VisionModelSpec;
}

export interface VisionAnalysisRequest {
  /** Raw screenshot (PNG/JPEG) */
  screenshot: Buffer;

  /** Screenshot with numbered element labels (optional optimization) */
  annotatedScreenshot?: Buffer;

  /** Element metadata for label-based selection */
  elementLabels?: ElementLabel[];

  /** User's goal (e.g., "Order chicken from the menu") */
  goal: string;

  /** Conversation history for multi-step reasoning */
  conversationHistory: ConversationMessage[];

  /** Current URL for context */
  currentUrl: string;
}

export interface VisionAnalysisResponse {
  /** The action to perform */
  action: BrowserAction;

  /** Model's reasoning (for UI display) */
  reasoning: string;

  /** Whether the goal has been achieved */
  goalAchieved: boolean;

  /** Confidence score (0-1) */
  confidence: number;

  /** Token usage for credit calculation */
  tokensUsed: TokenUsage;

  /** Raw response for debugging */
  rawResponse?: string;
}

export interface TokenUsage {
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
}
```

```typescript
// ═══════════════════════════════════════════════════════════════════════════
// ai/action/types.ts - Browser Action Types
// ═══════════════════════════════════════════════════════════════════════════

/**
 * Union type for all browser actions.
 * Each action maps to a Playwright operation.
 */
export type BrowserAction =
  | ClickAction
  | TypeAction
  | ScrollAction
  | NavigateAction
  | HoverAction
  | SelectAction
  | WaitAction
  | KeyPressAction
  | DoneAction;

export interface ClickAction {
  type: 'click';
  /** Element label number from annotated screenshot */
  elementId?: number;
  /** Pixel coordinates (fallback if no elementId) */
  coordinates?: { x: number; y: number };
  /** Click variant */
  variant?: 'left' | 'right' | 'double';
}

export interface TypeAction {
  type: 'type';
  /** Text to type */
  text: string;
  /** Element label number (optional - uses focused element if omitted) */
  elementId?: number;
  /** Whether to clear existing text first */
  clearFirst?: boolean;
}

export interface ScrollAction {
  type: 'scroll';
  direction: 'up' | 'down' | 'left' | 'right';
  /** Scroll amount in pixels (default: viewport height/width) */
  amount?: number;
}

export interface NavigateAction {
  type: 'navigate';
  url: string;
}

export interface HoverAction {
  type: 'hover';
  elementId?: number;
  coordinates?: { x: number; y: number };
}

export interface SelectAction {
  type: 'select';
  elementId: number;
  value: string;
}

export interface WaitAction {
  type: 'wait';
  /** Milliseconds to wait */
  ms?: number;
  /** Selector to wait for */
  selector?: string;
}

export interface KeyPressAction {
  type: 'keypress';
  key: string; // e.g., "Enter", "Escape", "Tab"
  modifiers?: ('Control' | 'Alt' | 'Shift' | 'Meta')[];
}

export interface DoneAction {
  type: 'done';
  /** Summary of what was accomplished */
  result: string;
  /** Whether the goal was successfully achieved */
  success: boolean;
}
```

```typescript
// ═══════════════════════════════════════════════════════════════════════════
// ai/vision-agent/types.ts - Vision Agent Types
// ═══════════════════════════════════════════════════════════════════════════

import type { Page } from 'playwright';
import type { BrowserAction } from '../action/types';
import type { TokenUsage } from '../vision-client/types';

/**
 * Configuration for a navigation session.
 */
export interface NavigationConfig {
  /** User's goal prompt */
  prompt: string;

  /** Playwright page to control */
  page: Page;

  /** Maximum steps before stopping */
  maxSteps: number;

  /** Model identifier (e.g., "qwen3-vl-30b") */
  model: string;

  /** API key for the model provider */
  apiKey: string;

  /** Callback invoked after each step */
  onStep: (step: NavigationStep) => Promise<void>;

  /** Callback URL for API to receive events */
  callbackUrl: string;

  /** Navigation session ID for correlation */
  navigationId: string;

  /** Optional: Abort signal for cancellation */
  abortSignal?: AbortSignal;
}

/**
 * Data emitted after each navigation step.
 */
export interface NavigationStep {
  navigationId: string;
  stepNumber: number;
  action: BrowserAction;
  reasoning: string;
  screenshot: Buffer;
  annotatedScreenshot?: Buffer;
  currentUrl: string;
  tokensUsed: TokenUsage;
  durationMs: number;
  goalAchieved: boolean;
  error?: string;
}

/**
 * Final result of a navigation session.
 */
export interface NavigationResult {
  navigationId: string;
  status: 'completed' | 'failed' | 'aborted' | 'max_steps_reached';
  totalSteps: number;
  totalTokens: number;
  totalDurationMs: number;
  finalUrl: string;
  error?: string;
}

/**
 * Vision Agent interface.
 * Testing seam: Mock this for integration tests.
 */
export interface VisionAgent {
  navigate(config: NavigationConfig): Promise<NavigationResult>;
  abort(): void;
}
```

```typescript
// ═══════════════════════════════════════════════════════════════════════════
// ai/vision-client/model-registry.ts - Model Specifications
// ═══════════════════════════════════════════════════════════════════════════

export interface VisionModelSpec {
  id: string;                              // Internal identifier
  apiModelId: string;                      // API model string
  displayName: string;                     // UI display name
  provider: 'openrouter' | 'anthropic' | 'ollama';
  inputCostPer1MTokens: number;            // USD
  outputCostPer1MTokens: number;           // USD
  maxContextTokens: number;
  supportsComputerUse: boolean;            // Claude-specific feature
  supportsElementLabels: boolean;          // Can use numbered labels
  recommended: boolean;                    // Show in recommended list
  tier: 'budget' | 'standard' | 'premium'; // For entitlement gating
}

export const MODEL_REGISTRY: Record<string, VisionModelSpec> = {
  'qwen3-vl-30b': {
    id: 'qwen3-vl-30b',
    apiModelId: 'qwen/qwen3-vl-30b-a3b-instruct',
    displayName: 'Qwen3-VL-30B',
    provider: 'openrouter',
    inputCostPer1MTokens: 0.15,
    outputCostPer1MTokens: 0.60,
    maxContextTokens: 262144,
    supportsComputerUse: false,
    supportsElementLabels: true,
    recommended: true,
    tier: 'budget',
  },
  'gpt-4o': {
    id: 'gpt-4o',
    apiModelId: 'openai/gpt-4o',
    displayName: 'GPT-4o',
    provider: 'openrouter',
    inputCostPer1MTokens: 2.50,
    outputCostPer1MTokens: 10.00,
    maxContextTokens: 128000,
    supportsComputerUse: false,
    supportsElementLabels: true,
    recommended: true,
    tier: 'standard',
  },
  'gpt-4o-mini': {
    id: 'gpt-4o-mini',
    apiModelId: 'openai/gpt-4o-mini',
    displayName: 'GPT-4o Mini',
    provider: 'openrouter',
    inputCostPer1MTokens: 0.15,
    outputCostPer1MTokens: 0.60,
    maxContextTokens: 128000,
    supportsComputerUse: false,
    supportsElementLabels: true,
    recommended: false,
    tier: 'budget',
  },
  'claude-sonnet-4': {
    id: 'claude-sonnet-4',
    apiModelId: 'anthropic/claude-sonnet-4-20250514',
    displayName: 'Claude Sonnet 4',
    provider: 'anthropic',
    inputCostPer1MTokens: 3.00,
    outputCostPer1MTokens: 15.00,
    maxContextTokens: 200000,
    supportsComputerUse: true,
    supportsElementLabels: true,
    recommended: true,
    tier: 'premium',
  },
  'claude-opus-4': {
    id: 'claude-opus-4',
    apiModelId: 'anthropic/claude-opus-4-20250514',
    displayName: 'Claude Opus 4',
    provider: 'anthropic',
    inputCostPer1MTokens: 15.00,
    outputCostPer1MTokens: 75.00,
    maxContextTokens: 200000,
    supportsComputerUse: true,
    supportsElementLabels: true,
    recommended: false,
    tier: 'premium',
  },
};
```

### Go Contracts (API)

```go
// ═══════════════════════════════════════════════════════════════════════════
// handlers/ai/vision_navigation.go - Request/Response Types
// ═══════════════════════════════════════════════════════════════════════════

// AINavigateRequest is the request body for POST /api/v1/ai-navigate
type AINavigateRequest struct {
    SessionID string `json:"session_id" validate:"required,uuid"`
    Prompt    string `json:"prompt" validate:"required,min=1,max=2000"`
    Model     string `json:"model" validate:"required"`
    MaxSteps  int    `json:"max_steps" validate:"min=1,max=100"`
}

// AINavigateResponse is returned immediately (async operation)
type AINavigateResponse struct {
    NavigationID string `json:"navigation_id"`
    Status       string `json:"status"` // "started"
    EstimatedCost float64 `json:"estimated_cost"` // Based on model + max steps
}

// NavigationStepCallback is received from playwright-driver
type NavigationStepCallback struct {
    NavigationID      string         `json:"navigation_id"`
    StepNumber        int            `json:"step_number"`
    Action            BrowserAction  `json:"action"`
    Reasoning         string         `json:"reasoning"`
    Screenshot        string         `json:"screenshot"` // base64
    CurrentURL        string         `json:"current_url"`
    TokensUsed        TokenUsage     `json:"tokens_used"`
    DurationMs        int64          `json:"duration_ms"`
    GoalAchieved      bool           `json:"goal_achieved"`
    Error             string         `json:"error,omitempty"`
}

// BrowserAction mirrors TypeScript union type
type BrowserAction struct {
    Type        string      `json:"type"` // click, type, scroll, etc.
    ElementID   *int        `json:"element_id,omitempty"`
    Coordinates *Coordinate `json:"coordinates,omitempty"`
    Text        string      `json:"text,omitempty"`
    Direction   string      `json:"direction,omitempty"`
    URL         string      `json:"url,omitempty"`
    Key         string      `json:"key,omitempty"`
    Result      string      `json:"result,omitempty"`
    Success     *bool       `json:"success,omitempty"`
}

// TokenUsage for credit calculation
type TokenUsage struct {
    PromptTokens     int `json:"prompt_tokens"`
    CompletionTokens int `json:"completion_tokens"`
    TotalTokens      int `json:"total_tokens"`
}
```

---

## Testing Strategy

### Testing Seams

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Testing Seam Diagram                               │
│                                                                              │
│  Each interface is a testing seam where we can inject mocks                 │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                         VisionAgent                                  │    │
│  │                                                                      │    │
│  │    ┌──────────────────┐    ┌──────────────────┐                     │    │
│  │    │ VisionModelClient│◄───│ MockVisionClient │  ← TESTING SEAM 1   │    │
│  │    │    (interface)   │    │ (returns canned  │                     │    │
│  │    └────────┬─────────┘    │  responses)      │                     │    │
│  │             │              └──────────────────┘                     │    │
│  │             ▼                                                       │    │
│  │    ┌──────────────────┐    ┌──────────────────┐                     │    │
│  │    │ ScreenshotCapture│◄───│ MockScreenshot   │  ← TESTING SEAM 2   │    │
│  │    │    (interface)   │    │ (returns test    │                     │    │
│  │    └────────┬─────────┘    │  images)         │                     │    │
│  │             │              └──────────────────┘                     │    │
│  │             ▼                                                       │    │
│  │    ┌──────────────────┐    ┌──────────────────┐                     │    │
│  │    │ ActionExecutor   │◄───│ MockExecutor     │  ← TESTING SEAM 3   │    │
│  │    │    (interface)   │    │ (records calls,  │                     │    │
│  │    └────────┬─────────┘    │  simulates page) │                     │    │
│  │             │              └──────────────────┘                     │    │
│  │             ▼                                                       │    │
│  │    ┌──────────────────┐    ┌──────────────────┐                     │    │
│  │    │ StepEmitter      │◄───│ MockEmitter      │  ← TESTING SEAM 4   │    │
│  │    │    (interface)   │    │ (captures events)│                     │    │
│  │    └──────────────────┘    └──────────────────┘                     │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Test Levels

| Level | What | How | Coverage Target |
|-------|------|-----|-----------------|
| **Unit** | Individual functions | Mock all dependencies | 90%+ |
| **Integration** | Component interactions | Mock external services only | 80%+ |
| **E2E** | Full navigation flow | Real browser, mock LLM | Critical paths |

### Example Unit Test

```typescript
// ai/action/parser.test.ts

import { describe, it, expect } from 'vitest';
import { parseLLMResponse } from './parser';

describe('parseLLMResponse', () => {
  it('parses click action with element ID', () => {
    const response = `
      I can see the login button labeled [5].
      I'll click on it to proceed.

      ACTION: click(5)
    `;

    const action = parseLLMResponse(response);

    expect(action).toEqual({
      type: 'click',
      elementId: 5,
    });
  });

  it('parses click action with coordinates when no element ID', () => {
    const response = `
      The button is at coordinates (150, 300).

      ACTION: click(150, 300)
    `;

    const action = parseLLMResponse(response);

    expect(action).toEqual({
      type: 'click',
      coordinates: { x: 150, y: 300 },
    });
  });

  it('parses type action', () => {
    const response = `
      I'll type the username into the input field [3].

      ACTION: type(3, "john.doe@example.com")
    `;

    const action = parseLLMResponse(response);

    expect(action).toEqual({
      type: 'type',
      elementId: 3,
      text: 'john.doe@example.com',
    });
  });

  it('parses done action', () => {
    const response = `
      The order has been placed successfully.

      ACTION: done(true, "Order #12345 placed for chicken")
    `;

    const action = parseLLMResponse(response);

    expect(action).toEqual({
      type: 'done',
      success: true,
      result: 'Order #12345 placed for chicken',
    });
  });

  it('throws on unrecognized action format', () => {
    const response = 'I am confused and cannot proceed.';

    expect(() => parseLLMResponse(response)).toThrow(
      'Could not parse action from LLM response'
    );
  });
});
```

### Example Integration Test

```typescript
// ai/vision-agent/agent.test.ts

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { createVisionAgent } from './agent';
import { createMockVisionClient } from '../vision-client/mock';
import { createMockPage } from '../../test-utils/mock-playwright';

describe('VisionAgent', () => {
  let mockClient: ReturnType<typeof createMockVisionClient>;
  let mockPage: ReturnType<typeof createMockPage>;
  let capturedSteps: NavigationStep[];

  beforeEach(() => {
    mockClient = createMockVisionClient();
    mockPage = createMockPage();
    capturedSteps = [];
  });

  it('executes multi-step navigation until goal achieved', async () => {
    // Configure mock to return sequence of actions
    mockClient.queueResponse({
      action: { type: 'click', elementId: 5 },
      reasoning: 'Clicking login button',
      goalAchieved: false,
      confidence: 0.9,
      tokensUsed: { promptTokens: 1000, completionTokens: 50, totalTokens: 1050 },
    });
    mockClient.queueResponse({
      action: { type: 'type', elementId: 3, text: 'user@example.com' },
      reasoning: 'Entering email',
      goalAchieved: false,
      confidence: 0.95,
      tokensUsed: { promptTokens: 1100, completionTokens: 60, totalTokens: 1160 },
    });
    mockClient.queueResponse({
      action: { type: 'done', success: true, result: 'Logged in successfully' },
      reasoning: 'Login complete',
      goalAchieved: true,
      confidence: 0.99,
      tokensUsed: { promptTokens: 1200, completionTokens: 40, totalTokens: 1240 },
    });

    const agent = createVisionAgent({ client: mockClient });

    const result = await agent.navigate({
      prompt: 'Log into the website',
      page: mockPage,
      maxSteps: 10,
      model: 'qwen3-vl-30b',
      apiKey: 'test-key',
      navigationId: 'nav-123',
      callbackUrl: 'http://localhost/callback',
      onStep: async (step) => { capturedSteps.push(step); },
    });

    expect(result.status).toBe('completed');
    expect(result.totalSteps).toBe(3);
    expect(capturedSteps).toHaveLength(3);
    expect(mockPage.click).toHaveBeenCalledWith('[data-ai-label="5"]');
    expect(mockPage.fill).toHaveBeenCalledWith('[data-ai-label="3"]', 'user@example.com');
  });

  it('stops at max steps if goal not achieved', async () => {
    // Configure mock to never achieve goal
    mockClient.setDefaultResponse({
      action: { type: 'scroll', direction: 'down' },
      reasoning: 'Looking for the button',
      goalAchieved: false,
      confidence: 0.5,
      tokensUsed: { promptTokens: 1000, completionTokens: 50, totalTokens: 1050 },
    });

    const agent = createVisionAgent({ client: mockClient });

    const result = await agent.navigate({
      prompt: 'Find something that does not exist',
      page: mockPage,
      maxSteps: 3,
      model: 'qwen3-vl-30b',
      apiKey: 'test-key',
      navigationId: 'nav-456',
      callbackUrl: 'http://localhost/callback',
      onStep: async (step) => { capturedSteps.push(step); },
    });

    expect(result.status).toBe('max_steps_reached');
    expect(result.totalSteps).toBe(3);
  });

  it('can be aborted mid-navigation', async () => {
    const abortController = new AbortController();

    mockClient.setDefaultResponse({
      action: { type: 'wait', ms: 1000 },
      reasoning: 'Waiting',
      goalAchieved: false,
      confidence: 0.5,
      tokensUsed: { promptTokens: 1000, completionTokens: 50, totalTokens: 1050 },
    });

    const agent = createVisionAgent({ client: mockClient });

    // Abort after first step
    setTimeout(() => abortController.abort(), 100);

    const result = await agent.navigate({
      prompt: 'Some task',
      page: mockPage,
      maxSteps: 10,
      model: 'qwen3-vl-30b',
      apiKey: 'test-key',
      navigationId: 'nav-789',
      callbackUrl: 'http://localhost/callback',
      onStep: async () => {},
      abortSignal: abortController.signal,
    });

    expect(result.status).toBe('aborted');
  });
});
```

---

## Responsibility Boundaries

| Component | Responsibilities | Does NOT Handle |
|-----------|-----------------|-----------------|
| **UI (AINavigationPanel)** | Collect prompt, show model selector, display progress | LLM calls, browser control |
| **API (VisionNavigationHandler)** | Validate entitlements, check credits, orchestrate | LLM calls, browser actions |
| **API (VisionCallbackHandler)** | Receive step events, deduct credits, broadcast WS | Browser control |
| **Driver (VisionAgent)** | Observe-decide-act loop, coordinate components | Credit management, auth |
| **Driver (VisionModelClient)** | Call OpenRouter/Claude API, parse response | Browser control, screenshots |
| **Driver (ScreenshotCapture)** | Capture page screenshot | Element detection |
| **Driver (ElementAnnotator)** | Add numbered labels to interactive elements | LLM calls |
| **Driver (ActionParser)** | Parse LLM response into BrowserAction | LLM calls, execution |
| **Driver (ActionExecutor)** | Execute BrowserAction via Playwright | LLM calls, parsing |
| **Driver (StepEmitter)** | Send step events to API callback | UI, WebSocket |

---

## Implementation Phases

### Phase 1: Core Infrastructure ✅ COMPLETE
1. ✅ Create folder structure in playwright-driver
2. ✅ Define TypeScript interfaces (types.ts files)
3. ✅ Implement MockVisionClient for testing
4. ✅ Implement action parser with tests (42 tests)
5. ✅ Implement action executor with tests (24 tests)

**Files created:**
- `src/ai/action/{types,parser,executor,index}.ts`
- `src/ai/vision-client/{types,mock,model-registry,index}.ts`
- `src/ai/screenshot/{types,capture,index}.ts`
- `src/ai/emitter/{types,callback-emitter,index}.ts`
- `src/ai/vision-agent/{types,index}.ts`
- `tests/unit/ai/action/{parser,executor}.test.ts`

### Phase 2: Vision Agent Loop ✅ COMPLETE
1. ✅ Implement screenshot capture (using existing Playwright)
2. ✅ Implement element annotator (metadata extraction from page)
3. ✅ Implement VisionAgent with observe-decide-act loop
4. ✅ Add comprehensive unit tests with mocks (15 tests)

**Files created:**
- `src/ai/screenshot/annotate.ts` - Element extraction and labeling
- `src/ai/screenshot/browser-globals.d.ts` - Browser type declarations
- `src/ai/vision-agent/agent.ts` - VisionAgent orchestrator
- `tests/unit/ai/vision-agent/agent.test.ts`

**Test Summary:** 81 tests passing (66 Phase 1 + 15 Phase 2)

### Phase 3: OpenRouter Integration ✅ COMPLETE
1. ✅ Implement OpenRouter vision client
2. ✅ Add prompt engineering for browser automation
3. ✅ Add factory function for client creation
4. ✅ Add comprehensive unit tests (60 tests)

**Files created:**
- `src/ai/vision-client/openrouter.ts` - OpenRouter API client with retry logic
- `src/ai/vision-client/prompts.ts` - System/user prompts for browser automation
- `src/ai/vision-client/factory.ts` - Factory for creating vision clients
- `tests/unit/ai/vision-client/{openrouter,prompts,factory}.test.ts`

**Test Summary:** 141 tests passing (81 Phase 1+2 + 60 Phase 3)

### Phase 4: API Integration ✅ COMPLETE
1. ✅ Add /session/:id/ai-navigate handler in driver
2. ✅ Add POST /api/v1/ai-navigate in API
3. ✅ Add callback handler for step events
4. ✅ Wire up credit deduction
5. ✅ Add WebSocket event broadcasting

**Files created/modified:**
- `playwright-driver/src/routes/session-ai-navigate.ts` - Route handler for AI navigation in driver
- `api/handlers/ai/vision_navigation.go` - Go API handler with entitlement/credit integration
- `api/handlers/handler.go` - Added vision navigation handler to Handler struct
- `api/main.go` - Added routes for AI navigation endpoints

**API Endpoints:**
- `POST /api/v1/ai-navigate` - Start AI navigation
- `GET /api/v1/ai-navigate/{id}/status` - Get navigation status
- `POST /api/v1/ai-navigate/{id}/abort` - Abort navigation
- `POST /api/v1/internal/ai-navigate/callback` - Internal callback for step events

**Driver Endpoints:**
- `POST /session/:id/ai-navigate` - Start AI navigation for session
- `POST /session/:id/ai-navigate/abort` - Abort navigation
- `GET /session/:id/ai-navigate/status` - Get navigation status
- `GET /ai/models` - List supported vision models

**Test Summary:** 141 AI tests still passing

### Phase 5: UI Integration ✅ COMPLETE
1. ✅ Create AINavigationPanel component
2. ✅ Add model selector with cost display
3. ✅ Integrate with Timeline for live updates
4. ✅ Add abort functionality

**Files created:**
- `ui/src/domains/recording/ai-navigation/types.ts` - TypeScript types
- `ui/src/domains/recording/ai-navigation/useAINavigation.ts` - React hook
- `ui/src/domains/recording/ai-navigation/AINavigationPanel.tsx` - Main control panel
- `ui/src/domains/recording/ai-navigation/AINavigationStepCard.tsx` - Timeline step component
- `ui/src/domains/recording/ai-navigation/AINavigationView.tsx` - Combined view
- `ui/src/domains/recording/ai-navigation/index.ts` - Module exports

### Phase 6: Polish & Production ✅ COMPLETE
1. ✅ Add Claude Computer Use provider
2. ✅ Add comprehensive error handling (error boundary)
3. ✅ Add loading states and skeleton UI
4. ✅ Add keyboard shortcuts (Cmd/Ctrl+Enter, Escape)
5. ✅ Add model selection persistence to localStorage
6. ✅ Performance optimization for long sessions

**Files created:**
- `ui/src/domains/recording/ai-navigation/AINavigationErrorBoundary.tsx` - Error boundary component
- `ui/src/domains/recording/ai-navigation/AINavigationSkeleton.tsx` - Skeleton/loading components
- `playwright-driver/src/ai/vision-client/claude-computer-use.ts` - Claude Computer Use client

**Key Performance Optimizations:**
- Conversation history trimming (max 20 messages)
- Element label limiting (max 50 elements)
- Configurable via VisionAgentConfig

---

## Key Design Principles

1. **Interface First** - Every component depends on interfaces, not implementations
2. **Dependency Injection** - Components receive dependencies, enabling testing
3. **Single Responsibility** - Each class/function has one reason to change
4. **Explicit Over Implicit** - No magic; clear data flow
5. **Fail Fast** - Validate inputs at boundaries, throw descriptive errors
6. **Testable by Default** - Every public function should be unit-testable

---

## Related Documents

- [AI Browser Automation Research](../research/ai-browser-automation-research.md) - Background research on browser-use, Claude Computer Use, and OpenRouter options
