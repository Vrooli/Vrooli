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

// Factory
export {
  createVisionClient,
  getModelInfo,
  isModelSupported,
  getSupportedModelIds,
  type VisionClientConfig,
} from './factory';
