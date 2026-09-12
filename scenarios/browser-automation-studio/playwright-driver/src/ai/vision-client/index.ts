/**
 * Vision Client Module
 *
 * This module provides types, clients, and utilities for vision model integration.
 */

// Types
export * from './types';

// AI Gateway client
export {
  AIGatewayVisionClient,
  createAIGatewayVisionClient,
  normalizeGatewayProfile,
  type AIGatewayVisionClientConfig,
} from './gateway';

// Mock Client
export {
  MockVisionClient,
  createMockVisionClient,
  createHappyPathMock,
  createNeverCompleteMock,
  type MockVisionClientConfig,
  type QueuedResponse,
} from './mock';

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
