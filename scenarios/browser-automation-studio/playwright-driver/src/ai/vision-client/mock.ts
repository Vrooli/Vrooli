/**
 * Mock Vision Client
 *
 * STABILITY: TESTING UTILITY
 *
 * This module provides a mock implementation of VisionModelClient
 * for testing the vision agent without making real LLM calls.
 */

import type {
  VisionModelClient,
  VisionAnalysisRequest,
  VisionAnalysisResponse,
  VisionModelSpec,
  TokenUsage,
} from './types';
import type { BrowserAction } from '../action/types';
import { getModelSpec, getDefaultModelId } from './model-registry';

/**
 * Configuration for MockVisionClient.
 */
export interface MockVisionClientConfig {
  /** Model to simulate */
  modelId?: string;

  /** Default confidence score */
  defaultConfidence?: number;

  /** Simulated latency in ms */
  latencyMs?: number;

  /** Whether to throw errors */
  shouldFail?: boolean;

  /** Error message when failing */
  failureMessage?: string;
}

/**
 * Queued response for sequential testing.
 */
export interface QueuedResponse {
  action: BrowserAction;
  reasoning: string;
  goalAchieved: boolean;
  confidence?: number;
  tokensUsed?: TokenUsage;
}

/**
 * Mock implementation of VisionModelClient for testing.
 *
 * Usage:
 * ```typescript
 * const mock = createMockVisionClient();
 *
 * // Queue specific responses
 * mock.queueResponse({
 *   action: { type: 'click', elementId: 5 },
 *   reasoning: 'Clicking login button',
 *   goalAchieved: false,
 * });
 *
 * // Or set a default response
 * mock.setDefaultResponse({
 *   action: { type: 'scroll', direction: 'down' },
 *   reasoning: 'Scrolling to find content',
 *   goalAchieved: false,
 * });
 * ```
 */
export class MockVisionClient implements VisionModelClient {
  private responseQueue: QueuedResponse[] = [];
  private defaultResponse: QueuedResponse | null = null;
  private readonly config: Required<MockVisionClientConfig>;
  private readonly modelSpec: VisionModelSpec;
  private callCount = 0;
  private callHistory: VisionAnalysisRequest[] = [];

  constructor(config: MockVisionClientConfig = {}) {
    this.config = {
      modelId: config.modelId ?? getDefaultModelId(),
      defaultConfidence: config.defaultConfidence ?? 0.9,
      latencyMs: config.latencyMs ?? 0,
      shouldFail: config.shouldFail ?? false,
      failureMessage: config.failureMessage ?? 'Mock client simulated failure',
    };
    this.modelSpec = getModelSpec(this.config.modelId);
  }

  /**
   * Queue a response for the next analyze call.
   * Responses are consumed in order.
   */
  queueResponse(response: QueuedResponse): void {
    this.responseQueue.push(response);
  }

  /**
   * Queue multiple responses at once.
   */
  queueResponses(responses: QueuedResponse[]): void {
    this.responseQueue.push(...responses);
  }

  /**
   * Set a default response used when queue is empty.
   */
  setDefaultResponse(response: QueuedResponse): void {
    this.defaultResponse = response;
  }

  /**
   * Clear all queued responses and default.
   */
  clearResponses(): void {
    this.responseQueue = [];
    this.defaultResponse = null;
  }

  /**
   * Get the number of times analyze() was called.
   */
  getCallCount(): number {
    return this.callCount;
  }

  /**
   * Get the history of all analyze() calls.
   */
  getCallHistory(): VisionAnalysisRequest[] {
    return [...this.callHistory];
  }

  /**
   * Get the last analyze() request.
   */
  getLastCall(): VisionAnalysisRequest | undefined {
    return this.callHistory[this.callHistory.length - 1];
  }

  /**
   * Reset call tracking.
   */
  resetTracking(): void {
    this.callCount = 0;
    this.callHistory = [];
  }

  /**
   * Configure the mock to fail on next call.
   */
  setFailMode(shouldFail: boolean, message?: string): void {
    this.config.shouldFail = shouldFail;
    if (message) {
      this.config.failureMessage = message;
    }
  }

  async analyze(request: VisionAnalysisRequest): Promise<VisionAnalysisResponse> {
    // Track call
    this.callCount++;
    this.callHistory.push(request);

    // Simulate latency
    if (this.config.latencyMs > 0) {
      await new Promise((resolve) => setTimeout(resolve, this.config.latencyMs));
    }

    // Simulate failure
    if (this.config.shouldFail) {
      throw new Error(this.config.failureMessage);
    }

    // Get response from queue or default
    const queued = this.responseQueue.shift();
    const response = queued ?? this.defaultResponse;

    if (!response) {
      throw new Error(
        'MockVisionClient: No response available. ' +
        'Call queueResponse() or setDefaultResponse() before analyze().'
      );
    }

    const tokensUsed: TokenUsage = response.tokensUsed ?? {
      promptTokens: 1000,
      completionTokens: 50,
      totalTokens: 1050,
    };

    return {
      action: response.action,
      reasoning: response.reasoning,
      goalAchieved: response.goalAchieved,
      confidence: response.confidence ?? this.config.defaultConfidence,
      tokensUsed,
      rawResponse: JSON.stringify(response.action),
    };
  }

  getModelSpec(): VisionModelSpec {
    return this.modelSpec;
  }
}

/**
 * Factory function to create a MockVisionClient.
 */
export function createMockVisionClient(
  config?: MockVisionClientConfig
): MockVisionClient {
  return new MockVisionClient(config);
}

/**
 * Create a mock client pre-configured for a "happy path" test.
 * Returns a client that will:
 * 1. Click on element 5
 * 2. Type "test@example.com" in element 3
 * 3. Complete successfully
 */
export function createHappyPathMock(): MockVisionClient {
  const mock = createMockVisionClient();

  mock.queueResponses([
    {
      action: { type: 'click', elementId: 5 },
      reasoning: 'Clicking on the login button',
      goalAchieved: false,
    },
    {
      action: { type: 'type', elementId: 3, text: 'test@example.com' },
      reasoning: 'Entering email address',
      goalAchieved: false,
    },
    {
      action: { type: 'done', success: true, result: 'Login completed' },
      reasoning: 'Login successful',
      goalAchieved: true,
    },
  ]);

  return mock;
}

/**
 * Create a mock client that never achieves the goal.
 * Useful for testing max steps behavior.
 */
export function createNeverCompleteMock(): MockVisionClient {
  const mock = createMockVisionClient();

  mock.setDefaultResponse({
    action: { type: 'scroll', direction: 'down' },
    reasoning: 'Looking for the target element',
    goalAchieved: false,
  });

  return mock;
}
