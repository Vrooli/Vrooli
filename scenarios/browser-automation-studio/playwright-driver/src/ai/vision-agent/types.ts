/**
 * Vision Agent Types
 *
 * STABILITY: STABLE CONTRACT
 *
 * This module defines the types for the vision agent, which orchestrates
 * the observe-decide-act loop for AI-driven browser navigation.
 */

import type { Page } from 'playwright';
import type { BrowserAction } from '../action/types';
import type { TokenUsage, ElementLabel } from '../vision-client/types';

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

  /** Optional: Screenshot quality (0-100, default 80) */
  screenshotQuality?: number;

  /** Optional: Whether to capture full page screenshots */
  fullPageScreenshot?: boolean;
}

/**
 * Human intervention details when awaiting human action.
 */
export interface HumanInterventionDetails {
  /** Why human intervention is needed */
  reason: string;
  /** Instructions for the user */
  instructions?: string;
  /** Type of intervention needed */
  interventionType: string;
  /** What triggered the intervention */
  trigger: 'programmatic' | 'ai_requested';
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
  /** Element labels visible in this step */
  elementLabels?: ElementLabel[];
  /** Whether this step is awaiting human intervention */
  awaitingHuman?: boolean;
  /** Human intervention details if awaiting */
  humanIntervention?: HumanInterventionDetails;
}

/**
 * Final result of a navigation session.
 */
export interface NavigationResult {
  navigationId: string;
  status: NavigationStatus;
  totalSteps: number;
  totalTokens: number;
  totalDurationMs: number;
  finalUrl: string;
  error?: string;
  /** Summary of what was accomplished */
  summary?: string;
}

/**
 * Navigation session status.
 */
export type NavigationStatus =
  | 'completed'
  | 'failed'
  | 'aborted'
  | 'max_steps_reached'
  | 'loop_detected'
  | 'awaiting_human';

/**
 * Vision Agent interface.
 * Testing seam: Mock this for integration tests.
 */
export interface VisionAgent {
  /**
   * Execute navigation loop until goal achieved or max steps.
   */
  navigate(config: NavigationConfig): Promise<NavigationResult>;

  /**
   * Abort the current navigation.
   */
  abort(): void;

  /**
   * Check if navigation is currently in progress.
   */
  isNavigating(): boolean;

  /**
   * Resume navigation after human intervention.
   */
  resume(): void;

  /**
   * Check if navigation is paused awaiting human intervention.
   */
  isPaused(): boolean;
}

/**
 * Dependencies for VisionAgent construction.
 * Using dependency injection for testability.
 */
export interface VisionAgentDeps {
  /** Vision model client for LLM calls */
  visionClient: VisionModelClientInterface;

  /** Screenshot capture service */
  screenshotCapture: ScreenshotCaptureInterface;

  /** Element annotator service */
  annotator: ElementAnnotatorInterface;

  /** Action executor service */
  actionExecutor: ActionExecutorInterface;

  /** Step emitter service */
  stepEmitter: StepEmitterInterface;

  /** Logger */
  logger: LoggerInterface;
}

/**
 * Interface for screenshot capture.
 * Testing seam: Mock for unit tests.
 */
export interface ScreenshotCaptureInterface {
  capture(page: Page, options?: ScreenshotOptions): Promise<Buffer>;
}

/**
 * Screenshot capture options.
 */
export interface ScreenshotOptions {
  quality?: number;
  fullPage?: boolean;
  format?: 'png' | 'jpeg';
}

/**
 * Interface for element annotator.
 * Testing seam: Mock for unit tests.
 */
export interface ElementAnnotatorInterface {
  annotate(
    screenshot: Buffer,
    elements: ElementLabel[]
  ): Promise<AnnotatedScreenshot>;
}

/**
 * Result of element annotation.
 */
export interface AnnotatedScreenshot {
  /** Screenshot with numbered labels */
  image: Buffer;
  /** Element metadata keyed by label ID */
  labels: ElementLabel[];
}

/**
 * Interface for action executor.
 * Testing seam: Mock for unit tests.
 */
export interface ActionExecutorInterface {
  execute(
    page: Page,
    action: BrowserAction,
    elementLabels?: ElementLabel[]
  ): Promise<ActionExecutionResult>;
}

/**
 * Result of action execution.
 */
export interface ActionExecutionResult {
  success: boolean;
  error?: string;
  /** URL after action (may have changed) */
  newUrl?: string;
  /** Duration in milliseconds */
  durationMs: number;
  /**
   * Context for intelligent loop detection.
   * Contains metadata about action effectiveness (e.g., scroll position changes).
   */
  context?: ActionExecutionContext;
}

/**
 * Context captured during action execution for loop detection.
 * Different action types populate different fields.
 */
export interface ActionExecutionContext {
  /**
   * Scroll context - populated for scroll actions.
   * Contains position before/after to detect if scroll had effect.
   */
  scroll?: {
    positionBefore: { x: number; y: number };
    positionAfter: { x: number; y: number };
  };
}

/**
 * Interface for step emitter (callback to API).
 * Testing seam: Mock for unit tests.
 */
export interface StepEmitterInterface {
  emit(step: NavigationStep, callbackUrl: string): Promise<void>;
}

/**
 * Interface for vision model client.
 * Re-exported from vision-client/types.ts for convenience.
 */
export interface VisionModelClientInterface {
  analyze(request: VisionAnalysisRequestInterface): Promise<VisionAnalysisResponseInterface>;
  getModelSpec(): VisionModelSpecInterface;
}

// Re-export types for convenience
export interface VisionAnalysisRequestInterface {
  screenshot: Buffer;
  annotatedScreenshot?: Buffer;
  elementLabels?: ElementLabel[];
  goal: string;
  conversationHistory: ConversationMessageInterface[];
  currentUrl: string;
}

export interface VisionAnalysisResponseInterface {
  action: BrowserAction;
  reasoning: string;
  goalAchieved: boolean;
  confidence: number;
  tokensUsed: TokenUsage;
  rawResponse?: string;
}

export interface VisionModelSpecInterface {
  id: string;
  apiModelId: string;
  displayName: string;
  provider: 'openrouter' | 'anthropic' | 'ollama';
  inputCostPer1MTokens: number;
  outputCostPer1MTokens: number;
  maxContextTokens: number;
  supportsComputerUse: boolean;
  supportsElementLabels: boolean;
  recommended: boolean;
  tier: 'budget' | 'standard' | 'premium';
}

export interface ConversationMessageInterface {
  role: 'user' | 'assistant' | 'system';
  content: string;
  screenshot?: Buffer;
}

/**
 * Logger interface for dependency injection.
 */
export interface LoggerInterface {
  debug(message: string, meta?: Record<string, unknown>): void;
  info(message: string, meta?: Record<string, unknown>): void;
  warn(message: string, meta?: Record<string, unknown>): void;
  error(message: string, meta?: Record<string, unknown>): void;
}
