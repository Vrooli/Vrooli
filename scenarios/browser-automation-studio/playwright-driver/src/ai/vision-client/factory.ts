/**
 * Vision Client Factory
 *
 * STABILITY: STABLE CONTRACT
 *
 * This module provides factory functions to create vision model clients.
 * It abstracts the creation of different client implementations based on:
 * - Model provider (OpenRouter, Anthropic, Ollama)
 * - Testing mode (mock vs real clients)
 *
 * TESTING SEAM: Use createMockVisionClient() in tests instead of real clients.
 */

import type { VisionModelClient, VisionModelSpec } from './types';
import { VisionModelError } from './types';
import { getModelSpec } from './model-registry';
import { ClaudeComputerUseClient } from './claude-computer-use';
import { OpenRouterVisionClient } from './openrouter';
import { MockVisionClient, type MockVisionClientConfig } from './mock';

/**
 * Configuration for creating a vision client.
 */
export interface VisionClientConfig {
  /** Model ID from registry */
  modelId: string;

  /** API key for the provider */
  apiKey: string;

  /** Request timeout in ms */
  timeoutMs?: number;

  /** Max retries on transient errors */
  maxRetries?: number;
}

/**
 * Create a vision model client for the specified model.
 *
 * This factory function selects the appropriate client implementation
 * based on the model's provider.
 *
 * @param config - Client configuration
 * @returns A VisionModelClient implementation
 * @throws VisionModelError if the model or provider is not supported
 *
 * @example
 * ```typescript
 * const client = createVisionClient({
 *   modelId: 'qwen3-vl-30b',
 *   apiKey: process.env.OPENROUTER_API_KEY!,
 * });
 *
 * const result = await client.analyze({
 *   screenshot: buffer,
 *   goal: 'Click the login button',
 *   currentUrl: 'https://example.com',
 *   conversationHistory: [],
 * });
 * ```
 */
export function createVisionClient(config: VisionClientConfig): VisionModelClient {
  const modelSpec = getModelSpec(config.modelId);

  switch (modelSpec.provider) {
    case 'openrouter':
      return new OpenRouterVisionClient({
        apiKey: config.apiKey,
        modelId: config.modelId,
        timeoutMs: config.timeoutMs,
        maxRetries: config.maxRetries,
      });

    case 'anthropic':
      return new ClaudeComputerUseClient({
        apiKey: config.apiKey,
        modelId: config.modelId,
        timeoutMs: config.timeoutMs,
        maxRetries: config.maxRetries,
      });

    case 'ollama':
      // TODO: Implement Ollama vision client
      throw new VisionModelError(
        `Ollama provider not yet implemented.`,
        'MODEL_UNAVAILABLE'
      );

    default: {
      const _exhaustive: never = modelSpec.provider;
      throw new VisionModelError(
        `Unknown provider: ${modelSpec.provider}`,
        'MODEL_UNAVAILABLE'
      );
    }
  }
}

/**
 * Create a mock vision client for testing.
 *
 * Use this in unit and integration tests to avoid real API calls.
 *
 * @param config - Optional mock configuration
 * @returns A MockVisionClient instance
 *
 * @example
 * ```typescript
 * const mock = createMockClient();
 * mock.queueResponse({
 *   action: { type: 'click', elementId: 5 },
 *   reasoning: 'Clicking login button',
 *   goalAchieved: false,
 * });
 *
 * const agent = createVisionAgent({ client: mock });
 * ```
 */
export function createMockClient(config?: MockVisionClientConfig): MockVisionClient {
  return new MockVisionClient(config);
}

/**
 * Get the model specification without creating a client.
 *
 * Useful for displaying model info in the UI before selection.
 *
 * @param modelId - Model ID from registry
 * @returns Model specification
 */
export function getModelInfo(modelId: string): VisionModelSpec {
  return getModelSpec(modelId);
}

/**
 * Check if a model ID is valid and supported.
 *
 * @param modelId - Model ID to check
 * @returns true if the model is supported
 */
export function isModelSupported(modelId: string): boolean {
  try {
    const spec = getModelSpec(modelId);
    // OpenRouter and Anthropic (Claude Computer Use) are supported
    return spec.provider === 'openrouter' || spec.provider === 'anthropic';
  } catch {
    return false;
  }
}

/**
 * Get all currently supported model IDs.
 *
 * @returns Array of supported model IDs
 */
export function getSupportedModelIds(): string[] {
  // Import dynamically to avoid circular dependency
  const { getAllModels } = require('./model-registry');
  return getAllModels()
    .filter((spec: VisionModelSpec) => spec.provider === 'openrouter' || spec.provider === 'anthropic')
    .map((spec: VisionModelSpec) => spec.id);
}
