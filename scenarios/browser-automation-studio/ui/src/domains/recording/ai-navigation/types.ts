/**
 * AI Navigation Types
 *
 * Types for the AI vision-based browser navigation feature.
 * These mirror the Go API types in vision_navigation.go
 */

/**
 * Vision model specification for UI display.
 */
export interface VisionModelSpec {
  id: string;
  displayName: string;
  provider: 'openrouter' | 'anthropic' | 'ollama';
  inputCostPer1MTokens: number;
  outputCostPer1MTokens: number;
  tier: 'budget' | 'standard' | 'premium';
  recommended: boolean;
}

/**
 * Available vision models.
 */
export const VISION_MODELS: VisionModelSpec[] = [
  {
    id: 'qwen3-vl-30b',
    displayName: 'Qwen3-VL-30B',
    provider: 'openrouter',
    inputCostPer1MTokens: 0.15,
    outputCostPer1MTokens: 0.60,
    tier: 'budget',
    recommended: true,
  },
  {
    id: 'gpt-4o',
    displayName: 'GPT-4o',
    provider: 'openrouter',
    inputCostPer1MTokens: 2.50,
    outputCostPer1MTokens: 10.00,
    tier: 'standard',
    recommended: true,
  },
  {
    id: 'gpt-4o-mini',
    displayName: 'GPT-4o Mini',
    provider: 'openrouter',
    inputCostPer1MTokens: 0.15,
    outputCostPer1MTokens: 0.60,
    tier: 'budget',
    recommended: false,
  },
  {
    id: 'claude-sonnet-4',
    displayName: 'Claude Sonnet 4',
    provider: 'anthropic',
    inputCostPer1MTokens: 3.00,
    outputCostPer1MTokens: 15.00,
    tier: 'premium',
    recommended: true,
  },
];

/**
 * Request to start AI navigation.
 */
export interface AINavigateRequest {
  sessionId: string;
  prompt: string;
  model: string;
  maxSteps?: number;
  apiKey?: string;
}

/**
 * Response when AI navigation starts.
 */
export interface AINavigateResponse {
  navigationId: string;
  status: string;
  model: string;
  maxSteps: number;
  estimatedCost?: number;
}

/**
 * Token usage for credit tracking.
 */
export interface TokenUsage {
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
}

/**
 * Browser action from vision model.
 */
export interface BrowserAction {
  type: 'click' | 'type' | 'scroll' | 'navigate' | 'hover' | 'select' | 'wait' | 'keypress' | 'done';
  elementId?: number;
  coordinates?: { x: number; y: number };
  text?: string;
  direction?: 'up' | 'down' | 'left' | 'right';
  url?: string;
  key?: string;
  result?: string;
  success?: boolean;
}

/**
 * AI navigation step event (received via WebSocket).
 */
export interface AINavigationStepEvent {
  type: 'ai_navigation_step';
  navigationId: string;
  sessionId: string;
  stepNumber: number;
  action: BrowserAction;
  reasoning: string;
  currentUrl: string;
  goalAchieved: boolean;
  tokensUsed: TokenUsage;
  durationMs: number;
  error?: string;
  timestamp: string;
}

/**
 * AI navigation complete event (received via WebSocket).
 */
export interface AINavigationCompleteEvent {
  type: 'ai_navigation_complete';
  navigationId: string;
  sessionId: string;
  status: 'completed' | 'failed' | 'aborted' | 'max_steps_reached' | 'loop_detected';
  totalSteps: number;
  totalTokens: number;
  totalDurationMs: number;
  finalUrl: string;
  error?: string;
  summary?: string;
  timestamp: string;
}

/**
 * Navigation status response.
 */
export interface NavigationStatusResponse {
  navigationId: string;
  sessionId: string;
  status: string;
  stepCount: number;
  totalTokens: number;
  startedAt: string;
}

/**
 * State of an AI navigation session.
 */
export interface AINavigationState {
  isNavigating: boolean;
  navigationId: string | null;
  prompt: string;
  model: string;
  steps: AINavigationStep[];
  status: 'idle' | 'navigating' | 'completed' | 'failed' | 'aborted' | 'max_steps_reached' | 'loop_detected';
  totalTokens: number;
  error: string | null;
}

/**
 * A single step in an AI navigation session.
 * Used for timeline display.
 */
export interface AINavigationStep {
  id: string;
  stepNumber: number;
  action: BrowserAction;
  reasoning: string;
  currentUrl: string;
  goalAchieved: boolean;
  tokensUsed: TokenUsage;
  durationMs: number;
  error?: string;
  timestamp: Date;
}
