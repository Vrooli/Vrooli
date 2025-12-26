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
