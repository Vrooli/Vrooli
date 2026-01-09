/**
 * Vision Client Module
 *
 * This module provides types, clients, and utilities for vision model integration.
 */

// Types
export * from './types';

// Model Registry
export {
  MODEL_REGISTRY,
  getModelSpec,
  getAllModels,
  getRecommendedModels,
  getModelsByTier,
  getModelsByProvider,
  calculateCost,
  supportsComputerUse,
  supportsElementLabels,
  getDefaultModelId,
} from './model-registry';

// Mock Client
export {
  MockVisionClient,
  createMockVisionClient,
  createHappyPathMock,
  createNeverCompleteMock,
  type MockVisionClientConfig,
  type QueuedResponse,
} from './mock';

// OpenRouter Client
export {
  OpenRouterVisionClient,
  createOpenRouterClient,
  type OpenRouterClientConfig,
} from './openrouter';

// Claude Computer Use Client
export {
  ClaudeComputerUseClient,
  createClaudeComputerUseClient,
  type ClaudeComputerUseClientConfig,
} from './claude-computer-use';

// Prompts
export {
  generateSystemPrompt,
  generateUserPrompt,
  formatElementLabelsCompact,
  generateContinuationPrompt,
  generateVerificationPrompt,
} from './prompts';

// Factory
export {
  createVisionClient,
  createMockClient,
  getModelInfo,
  isModelSupported,
  getSupportedModelIds,
  type VisionClientConfig,
} from './factory';
