/**
 * Execution Module
 *
 * Core domain logic for executing browser automation instructions.
 *
 * STABILITY: STABLE CORE
 *
 * This module provides the pure execution pipeline, separated from:
 * - HTTP routing concerns (routes/)
 * - Session lifecycle management (session/)
 * - Caching infrastructure (infra/)
 *
 * @module execution
 */

export {
  executeInstruction,
  validateInstruction,
  createInstructionKey,
  type ExecutionContext,
  type ExecutionResult,
  type ValidationResult,
  type ValidationError,
} from './instruction-executor';
