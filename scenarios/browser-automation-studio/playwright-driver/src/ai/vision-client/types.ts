/**
 * Vision Model Client Types
 *
 * STABILITY: STABLE CONTRACT
 *
 * This module defines the interface for vision model clients.
 * Implementations include OpenRouter, Claude Computer Use, and Mock (for testing).
 *
 * TESTING SEAM: Mock this interface for unit tests to avoid real LLM calls.
 */

import type { BrowserAction } from '../action/types';

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

/**
 * Request for vision model analysis.
 */
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

/**
 * Response from vision model analysis.
 */
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

/**
 * Token usage for credit calculation.
 */
export interface TokenUsage {
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
}

/**
 * Model specification for registry.
 */
export interface VisionModelSpec {
  /** Internal identifier */
  id: string;
  /** API model string */
  apiModelId: string;
  /** UI display name */
  displayName: string;
  /** Provider */
  provider: 'openrouter' | 'anthropic' | 'ollama';
  /** USD per 1M input tokens */
  inputCostPer1MTokens: number;
  /** USD per 1M output tokens */
  outputCostPer1MTokens: number;
  /** Max context tokens */
  maxContextTokens: number;
  /** Claude-specific feature */
  supportsComputerUse: boolean;
  /** Can use numbered labels */
  supportsElementLabels: boolean;
  /** Show in recommended list */
  recommended: boolean;
  /** For entitlement gating */
  tier: 'budget' | 'standard' | 'premium';
}

/**
 * Element label for annotated screenshots.
 */
export interface ElementLabel {
  /** Label number shown on screenshot */
  id: number;
  /** CSS selector for the element */
  selector: string;
  /** Element tag name */
  tagName: string;
  /** Bounding box */
  bounds: {
    x: number;
    y: number;
    width: number;
    height: number;
  };
  /** Text content (truncated) */
  text?: string;
  /** Role/type of element */
  role?: string;
  /** Placeholder text */
  placeholder?: string;
  /** Aria label */
  ariaLabel?: string;
}

/**
 * Conversation message for multi-step reasoning.
 */
export interface ConversationMessage {
  role: 'user' | 'assistant' | 'system';
  content: string;
  /** Screenshot for this message (if applicable) */
  screenshot?: Buffer;
}

/**
 * Error thrown by vision model clients.
 */
export class VisionModelError extends Error {
  constructor(
    message: string,
    public readonly code: VisionErrorCode,
    public readonly retryable: boolean = false,
    public readonly cause?: Error
  ) {
    super(message);
    this.name = 'VisionModelError';
  }
}

/**
 * Vision model error codes.
 */
export type VisionErrorCode =
  | 'RATE_LIMITED'
  | 'QUOTA_EXCEEDED'
  | 'INVALID_API_KEY'
  | 'MODEL_UNAVAILABLE'
  | 'CONTEXT_TOO_LONG'
  | 'PARSE_ERROR'
  | 'NETWORK_ERROR'
  | 'UNKNOWN';
