/**
 * Vision Agent Module
 *
 * This module provides the vision-based AI navigation agent.
 * The agent orchestrates the observe-decide-act loop for
 * AI-driven browser automation.
 */

// Types
export * from './types';

// Agent implementation
export {
  createVisionAgent,
  createConsoleLogger,
  createNoopLogger,
  type VisionAgentConfig,
} from './agent';

// Loop detection (for configuration and testing)
export {
  createLoopDetector,
  serializeAction,
  createActionContext,
  createScrollContext,
  createHistoryEntry,
  DEFAULT_LOOP_CONFIG,
  type LoopDetectionConfig,
  type ScrollLoopConfig,
  type ClickLoopConfig,
  type DefaultLoopConfig,
  type ActionContext,
  type ScrollContext,
  type ClickContext,
  type ActionHistoryEntry,
  type LoopDetectionResult,
  type LoopDetector,
} from './loop-detection';
