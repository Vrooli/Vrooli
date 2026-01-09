/**
 * AI Conversation Types
 *
 * Types for the chat-based AI conversation interface.
 * Re-exports relevant types from sidebar domain for convenience.
 */

// Re-export message types from sidebar domain
export type {
  AIMessage,
  AIMessageRole,
  AIMessageStatus,
  AISettings,
} from '../sidebar/types';

export {
  createUserMessage,
  createAssistantMessage,
  createSystemMessage,
  DEFAULT_AI_SETTINGS,
} from '../sidebar/types';

// Re-export navigation types for convenience
export type {
  AINavigationStep,
  HumanInterventionState,
  TokenUsage,
  VisionModelSpec,
} from '../ai-navigation/types';

export { VISION_MODELS } from '../ai-navigation/types';
