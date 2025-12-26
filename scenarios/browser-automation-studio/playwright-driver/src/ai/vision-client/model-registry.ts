/**
 * Model Registry
 *
 * STABILITY: STABLE CONTRACT
 *
 * This module contains specifications for all supported vision models.
 * Used for model selection, cost calculation, and entitlement gating.
 */

import type { VisionModelSpec } from './types';

/**
 * Registry of all supported vision models.
 */
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

/**
 * Get a model specification by ID.
 * @throws Error if model not found
 */
export function getModelSpec(modelId: string): VisionModelSpec {
  const spec = MODEL_REGISTRY[modelId];
  if (!spec) {
    const available = Object.keys(MODEL_REGISTRY).join(', ');
    throw new Error(
      `Unknown vision model: ${modelId}. Available models: ${available}`
    );
  }
  return spec;
}

/**
 * Get all available models.
 */
export function getAllModels(): VisionModelSpec[] {
  return Object.values(MODEL_REGISTRY);
}

/**
 * Get recommended models for display.
 */
export function getRecommendedModels(): VisionModelSpec[] {
  return Object.values(MODEL_REGISTRY).filter((spec) => spec.recommended);
}

/**
 * Get models by tier for entitlement checking.
 */
export function getModelsByTier(
  tier: 'budget' | 'standard' | 'premium'
): VisionModelSpec[] {
  return Object.values(MODEL_REGISTRY).filter((spec) => spec.tier === tier);
}

/**
 * Get models by provider.
 */
export function getModelsByProvider(
  provider: 'openrouter' | 'anthropic' | 'ollama'
): VisionModelSpec[] {
  return Object.values(MODEL_REGISTRY).filter(
    (spec) => spec.provider === provider
  );
}

/**
 * Calculate cost for token usage.
 * @returns Cost in USD
 */
export function calculateCost(
  modelId: string,
  promptTokens: number,
  completionTokens: number
): number {
  const spec = getModelSpec(modelId);
  const inputCost = (promptTokens / 1_000_000) * spec.inputCostPer1MTokens;
  const outputCost =
    (completionTokens / 1_000_000) * spec.outputCostPer1MTokens;
  return inputCost + outputCost;
}

/**
 * Check if a model supports computer use (Claude-specific).
 */
export function supportsComputerUse(modelId: string): boolean {
  return getModelSpec(modelId).supportsComputerUse;
}

/**
 * Check if a model supports element labels.
 */
export function supportsElementLabels(modelId: string): boolean {
  return getModelSpec(modelId).supportsElementLabels;
}

/**
 * Get the default model ID.
 */
export function getDefaultModelId(): string {
  return 'qwen3-vl-30b';
}
