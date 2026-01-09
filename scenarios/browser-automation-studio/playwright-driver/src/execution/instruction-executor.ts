/**
 * Instruction Executor
 *
 * Core domain service for executing browser automation instructions.
 *
 * RESPONSIBILITY:
 * This service owns the pure execution logic, extracted from session-run.ts
 * to provide a clean separation between:
 * - HTTP routing concerns (session-run.ts)
 * - Execution domain logic (this file)
 *
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ WHAT THIS SERVICE DOES:                                                 │
 * │                                                                         │
 * │ 1. VALIDATION - Validates instruction structure before execution        │
 * │ 2. HANDLER DISPATCH - Finds and invokes the appropriate handler         │
 * │ 3. TELEMETRY - Orchestrates telemetry collection during execution       │
 * │ 4. OUTCOME BUILDING - Transforms results to StepOutcome wire format     │
 * │                                                                         │
 * │ WHAT IT DOES NOT DO:                                                    │
 * │                                                                         │
 * │ - HTTP request/response handling (route layer)                          │
 * │ - Session phase management (session manager)                            │
 * │ - Idempotency caching (infra layer)                                     │
 * │ - Replay detection (route layer)                                        │
 * └─────────────────────────────────────────────────────────────────────────┘
 *
 * DESIGN DECISIONS:
 *
 * 1. Stateless service - receives all context through execute() params
 * 2. Returns typed result - ExecutionResult with outcome + telemetry
 * 3. Pure domain logic - no HTTP, no caching, no session state mutation
 *
 * @module execution/instruction-executor
 */

import type { HandlerRegistry } from '../handlers';
import { type HandlerContext } from '../handlers/base';
import type { Metrics } from '../utils/metrics';
import {
  CompiledInstructionSchema,
  toHandlerInstruction,
  parseProtoLenient,
  type HandlerInstruction,
  type StepOutcome,
} from '../proto';
import { TelemetryOrchestrator, type StepTelemetry } from '../telemetry';
import {
  buildStepOutcome,
  toDriverOutcome,
  type HandlerResult,
  type DriverOutcome,
} from '../outcome';
import { logger, scopedLog, LogContext } from '../utils';

// =============================================================================
// Types
// =============================================================================

/**
 * ExecutionContext is now an alias for HandlerContext.
 *
 * UNIFIED CONTEXT: Both the instruction executor and handlers use the same
 * context type to eliminate the mental mapping between two similar types.
 * The HandlerContext in handlers/base.ts is the canonical definition.
 *
 * This alias is retained for backward compatibility with code that imports
 * ExecutionContext from this module.
 */
export type ExecutionContext = HandlerContext;

/**
 * Result of instruction execution.
 * Contains both the wire-format outcome and collected telemetry.
 */
export interface ExecutionResult {
  /** Whether execution succeeded */
  success: boolean;
  /** Proto StepOutcome (internal format) */
  outcome: StepOutcome;
  /** Wire format outcome for HTTP response */
  driverOutcome: DriverOutcome;
  /** Collected telemetry data */
  telemetry: StepTelemetry;
  /** Handler result (raw, before outcome building) */
  handlerResult: HandlerResult;
  /** Execution duration in milliseconds */
  durationMs: number;
  /** The parsed instruction (for caching/logging) */
  instruction: HandlerInstruction;
}

/**
 * Validation error returned when instruction structure is invalid.
 */
export interface ValidationError {
  code: 'INVALID_INSTRUCTION';
  message: string;
}

/**
 * Result of instruction validation.
 */
export type ValidationResult =
  | { valid: true; instruction: HandlerInstruction }
  | { valid: false; error: ValidationError };

// =============================================================================
// Validation
// =============================================================================

/**
 * Validate that a raw instruction object has all required fields.
 * Accepts both wire format (snake_case: node_id) and proto format (camelCase: nodeId).
 *
 * @returns Error message if invalid, null if valid
 */
function validateInstructionStructure(rawInstruction: unknown): string | null {
  if (!rawInstruction || typeof rawInstruction !== 'object') {
    return 'Missing or invalid instruction: must be an object';
  }
  const inst = rawInstruction as Record<string, unknown>;
  if (typeof inst.index !== 'number') return 'Missing or invalid instruction.index: must be a number';
  // Accept both node_id (wire format) and nodeId (proto format)
  const nodeId = inst.node_id ?? inst.nodeId;
  if (!nodeId || typeof nodeId !== 'string') return 'Missing or invalid instruction.node_id: must be a non-empty string';
  const hasLegacyType = typeof inst.type === 'string' && inst.type.length > 0;
  const action = inst.action as Record<string, unknown> | undefined;
  const actionType = action?.type;
  const hasTypedAction = typeof actionType === 'string' || typeof actionType === 'number';
  if (!hasLegacyType && !hasTypedAction) return 'Missing or invalid instruction.type: must be a non-empty string';
  if (hasLegacyType && (!inst.params || typeof inst.params !== 'object')) return 'Missing or invalid instruction.params: must be an object';
  return null;
}

/**
 * Validate and parse a raw instruction.
 *
 * @param rawInstruction - Raw instruction object from request body
 * @returns Validation result with parsed instruction or error
 */
export function validateInstruction(rawInstruction: unknown): ValidationResult {
  const validationError = validateInstructionStructure(rawInstruction);
  if (validationError) {
    return {
      valid: false,
      error: { code: 'INVALID_INSTRUCTION', message: validationError },
    };
  }

  // Parse with proto schema and convert to handler-friendly format
  const protoInstruction = parseProtoLenient(CompiledInstructionSchema, rawInstruction);
  const instruction = toHandlerInstruction(protoInstruction);

  return { valid: true, instruction };
}

/**
 * Create a unique key for an instruction based on nodeId and index.
 * Used to track which instructions have been executed in a session.
 */
export function createInstructionKey(instruction: HandlerInstruction): string {
  return `${instruction.nodeId}:${instruction.index}`;
}

// =============================================================================
// Executor
// =============================================================================

/**
 * Execute a browser automation instruction.
 *
 * This is the core execution pipeline, extracted from session-run.ts for
 * cleaner separation of concerns. It handles:
 *
 * 1. Handler dispatch - finds the right handler for the instruction type
 * 2. Execution - runs the handler with proper context
 * 3. Telemetry - collects screenshots, DOM, console, network events
 * 4. Outcome building - transforms results to wire format
 *
 * @param instruction - The validated instruction to execute
 * @param context - Execution context (page, config, logger, etc.)
 * @param handlerRegistry - Registry of instruction handlers
 * @returns Execution result with outcome and telemetry
 */
export async function executeInstruction(
  instruction: HandlerInstruction,
  context: ExecutionContext,
  handlerRegistry: HandlerRegistry
): Promise<ExecutionResult> {
  const startedAt = new Date();

  logger.info(scopedLog(LogContext.INSTRUCTION, 'executing'), {
    sessionId: context.sessionId,
    type: instruction.type,
    stepIndex: instruction.index,
    nodeId: instruction.nodeId,
    selector: instruction.params.selector,
    url: instruction.params.url,
  });

  // Get handler for this instruction type
  const handler = handlerRegistry.getHandler(instruction);

  // Setup telemetry collection
  const telemetryOrchestrator = new TelemetryOrchestrator(context.page, context.config);
  telemetryOrchestrator.start();

  // Execute the instruction
  let handlerResult: HandlerResult;
  let instructionDuration: number;

  try {
    const instructionStart = Date.now();
    // Context is now unified - pass directly to handler
    handlerResult = await handler.execute(instruction, context);
    instructionDuration = Date.now() - instructionStart;
  } catch (error) {
    // Ensure telemetry is disposed on error
    telemetryOrchestrator.dispose();
    throw error;
  }

  // Record metrics
  recordMetrics(context.metrics, instruction.type, handlerResult, instructionDuration);

  // Collect telemetry
  const telemetry = await telemetryOrchestrator.collectForStep(handlerResult);
  telemetryOrchestrator.dispose();

  // Build outcome
  const completedAt = new Date();
  const outcome = buildStepOutcome({
    instruction,
    result: handlerResult,
    startedAt,
    completedAt,
    finalUrl: context.page.url(),
    screenshot: telemetry.screenshot,
    domSnapshot: telemetry.domSnapshot,
    consoleLogs: telemetry.consoleLogs,
    networkEvents: telemetry.networkEvents,
  });

  logger.info(scopedLog(LogContext.INSTRUCTION, handlerResult.success ? 'completed' : 'failed'), {
    sessionId: context.sessionId,
    type: instruction.type,
    stepIndex: instruction.index,
    success: handlerResult.success,
    durationMs: outcome.durationMs,
    finalUrl: context.page.url(),
    ...(handlerResult.error && {
      errorCode: handlerResult.error.code,
      errorKind: handlerResult.error.kind,
      errorMessage: handlerResult.error.message,
    }),
  });

  // Convert to wire format
  const driverOutcome = toDriverOutcome(outcome, telemetry.screenshot, telemetry.domSnapshot);

  return {
    success: handlerResult.success,
    outcome,
    driverOutcome,
    telemetry,
    handlerResult,
    durationMs: instructionDuration,
    instruction,
  };
}

// =============================================================================
// Helpers
// =============================================================================

/**
 * Record execution metrics.
 */
function recordMetrics(
  metrics: Metrics,
  instructionType: string,
  result: HandlerResult,
  durationMs: number
): void {
  metrics.instructionDuration.observe(
    { type: instructionType, success: String(result.success) },
    durationMs
  );
  if (!result.success) {
    metrics.instructionErrors.inc({
      type: instructionType,
      error_kind: result.error?.kind?.toString() || 'unknown',
    });
  }
}
