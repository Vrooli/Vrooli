/**
 * Vision Client Factory
 *
 * STABILITY: STABLE CONTRACT
 *
 * This module provides factory functions to create vision model clients.
 * It abstracts the creation of different client implementations based on:
 * - Provider-neutral AI Gateway route profiles
 * - Provider-neutral client construction
 *
 * Tests provide a local fake at the VisionModelClient boundary.
 */

import type { VisionModelClient, VisionModelSpec } from './types';
import { AIGatewayVisionClient, normalizeGatewayProfile } from './gateway';

/**
 * Configuration for creating a vision client.
 */
export interface VisionClientConfig {
  /** Provider-neutral gateway profile. */
  modelId?: string;

  /** Optional gateway URL override, primarily for tests and external drivers. */
  gatewayUrl?: string;

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
 * const client = createVisionClient({ modelId: 'local_first' });
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
  const profile = normalizeGatewayProfile(config.modelId);
  return new AIGatewayVisionClient({
    gatewayUrl: config.gatewayUrl,
    profile,
    timeoutMs: config.timeoutMs,
  });
}

/**
 * Get the model specification without creating a client.
 *
 * Useful for displaying route info in the UI before selection.
 *
 * @param modelId - AI Gateway route profile
 * @returns Model specification
 */
export function getModelInfo(modelId: string): VisionModelSpec {
  const profile = normalizeGatewayProfile(modelId);
  return {
    id: profile,
    displayName: profile === 'local_first' ? 'Local-first vision' : 'Hosted vision',
    provider: 'ai-gateway',
    supportsComputerUse: false,
    supportsElementLabels: true,
    recommended: profile === 'local_first',
    tier: profile === 'local_first' ? 'local' : 'remote',
  };
}

/**
 * Check if a model ID is valid and supported.
 *
 * @param modelId - Model ID to check
 * @returns true if the model is supported
 */
export function isModelSupported(modelId: string): boolean {
  return ['local_first', 'remote_only', 'local-first', 'remote-only'].includes(modelId.trim().toLowerCase());
}

/**
 * Get all currently supported model IDs.
 *
 * @returns Array of supported model IDs
 */
export function getSupportedModelIds(): string[] {
  return ['local_first', 'remote_only'];
}
