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
  /** Provider-neutral AI Gateway routing profile. */
  profile: 'local_first' | 'remote_only';
  tier: 'local' | 'remote';
  recommended: boolean;
}

/**
 * Available vision models.
 */
export const VISION_MODELS: VisionModelSpec[] = [
  {
    id: 'local_first',
    displayName: 'Local-first vision',
    profile: 'local_first',
    tier: 'local',
    recommended: true,
  },
  {
    id: 'remote_only',
    displayName: 'Hosted vision',
    profile: 'remote_only',
    tier: 'remote',
    recommended: false,
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
  type: 'click' | 'type' | 'scroll' | 'navigate' | 'hover' | 'select' | 'wait' | 'keypress' | 'done' | 'request_human';
  elementId?: number;
  coordinates?: { x: number; y: number };
  text?: string;
  direction?: 'up' | 'down' | 'left' | 'right';
  url?: string;
  key?: string;
  result?: string;
  success?: boolean;
  // For request_human action
  reason?: string;
  instructions?: string;
  interventionType?: 'captcha' | 'verification' | 'complex_interaction' | 'login_required' | 'other';
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
  status: 'completed' | 'failed' | 'aborted' | 'max_steps_reached' | 'loop_detected' | 'awaiting_human';
  totalSteps: number;
  totalTokens: number;
  totalDurationMs: number;
  finalUrl: string;
  error?: string;
  summary?: string;
  timestamp: string;
}

/**
 * AI navigation awaiting human intervention event (received via WebSocket).
 */
export interface AINavigationAwaitingHumanEvent {
  type: 'ai_navigation_awaiting_human';
  navigationId: string;
  sessionId: string;
  stepNumber: number;
  reason: string;
  instructions?: string;
  interventionType: 'captcha' | 'verification' | 'complex_interaction' | 'login_required' | 'other';
  trigger: 'programmatic' | 'ai_requested';
  timestamp: string;
}

/**
 * AI navigation resumed event (received via WebSocket).
 */
export interface AINavigationResumedEvent {
  type: 'ai_navigation_resumed';
  navigationId: string;
  sessionId: string;
  timestamp: string;
}

/**
 * Human intervention state for UI display.
 */
export interface HumanInterventionState {
  reason: string;
  instructions?: string;
  interventionType: 'captcha' | 'verification' | 'complex_interaction' | 'login_required' | 'other';
  trigger: 'programmatic' | 'ai_requested';
  startedAt: Date;
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
  status: 'idle' | 'navigating' | 'aborting' | 'completed' | 'failed' | 'aborted' | 'max_steps_reached' | 'loop_detected' | 'awaiting_human';
  totalTokens: number;
  error: string | null;
  humanIntervention: HumanInterventionState | null;
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
