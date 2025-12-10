/**
 * Outcome Builder
 *
 * Domain logic for constructing StepOutcome and DriverOutcome from handler results.
 * This module owns the transformation from internal HandlerResult to wire-format outcomes.
 *
 * Responsibilities:
 * - Build StepOutcome from handler execution results
 * - Convert StepOutcome to DriverOutcome (wire format with flattened fields)
 * - Normalize telemetry data for contract compliance
 */

import type { HandlerResult } from '../handlers/base';
import type { CompiledInstruction, StepOutcome, DriverOutcome, Screenshot, DOMSnapshot } from '../types';
import { safeDuration, validateStepIndex, safeSerializable } from '../utils';

/**
 * Parameters for building a step outcome
 */
export interface BuildOutcomeParams {
  instruction: CompiledInstruction;
  result: HandlerResult;
  startedAt: Date;
  completedAt: Date;
  finalUrl: string;
  screenshot?: Screenshot & { base64?: string };
  domSnapshot?: DOMSnapshot;
  consoleLogs?: StepOutcome['console_logs'];
  networkEvents?: StepOutcome['network'];
}

/**
 * Build a StepOutcome from execution results.
 *
 * This is the canonical way to construct outcomes - handlers should not
 * build StepOutcome directly but instead return HandlerResult.
 */
export function buildStepOutcome(params: BuildOutcomeParams): StepOutcome {
  const {
    instruction,
    result,
    startedAt,
    completedAt,
    finalUrl,
    screenshot,
    domSnapshot,
    consoleLogs,
    networkEvents,
  } = params;

  // Hardened: Validate step index and calculate safe duration
  const validatedIndex = validateStepIndex(instruction.index, 'buildStepOutcome');
  const durationMs = safeDuration(startedAt, completedAt);

  const outcome: StepOutcome = {
    schema_version: 'automation-step-outcome-v1',
    payload_version: '1',
    step_index: validatedIndex,
    attempt: 1, // TODO: Track actual attempt number when retry logic is implemented
    node_id: instruction.node_id,
    step_type: instruction.type,
    success: result.success,
    started_at: startedAt.toISOString(),
    completed_at: completedAt.toISOString(),
    duration_ms: durationMs,
    final_url: finalUrl,
  };

  // Add telemetry (exclude base64 from screenshot - that's wire format only)
  if (screenshot) {
    outcome.screenshot = {
      media_type: screenshot.media_type,
      capture_time: screenshot.capture_time,
      width: screenshot.width,
      height: screenshot.height,
      hash: screenshot.hash,
      from_cache: screenshot.from_cache,
      truncated: screenshot.truncated,
      source: screenshot.source,
    };
  }

  if (domSnapshot) {
    outcome.dom_snapshot = domSnapshot;
  }

  if (consoleLogs && consoleLogs.length > 0) {
    outcome.console_logs = consoleLogs;
  }

  if (networkEvents && networkEvents.length > 0) {
    outcome.network = networkEvents;
  }

  // Add extracted data from handler
  // Hardened: Ensure extracted data is JSON-serializable
  if (result.extracted_data) {
    outcome.extracted_data = safeSerializable(result.extracted_data, 'extracted_data');
  }

  // Add focused element info
  if (result.focus) {
    outcome.focused_element = {
      selector: result.focus.selector || '',
      bounding_box: result.focus.bounding_box,
    };
  }

  // Add failure details
  if (!result.success && result.error) {
    // Hardened: Validate that error.kind is a valid FailureKind value
    // Invalid kinds default to 'engine' to prevent type errors downstream
    const validFailureKinds = ['engine', 'infra', 'orchestration', 'user', 'timeout', 'cancelled'] as const;
    const rawKind = result.error.kind;
    const kind: typeof validFailureKinds[number] = (
      rawKind && validFailureKinds.includes(rawKind as typeof validFailureKinds[number])
    ) ? (rawKind as typeof validFailureKinds[number]) : 'engine';

    outcome.failure = {
      kind,
      code: result.error.code,
      message: result.error.message,
      retryable: result.error.retryable || false,
    };
  }

  return outcome;
}

/**
 * Convert StepOutcome to DriverOutcome (wire format).
 *
 * The driver wire format uses flat fields for screenshot/DOM data
 * instead of nested objects, to simplify Go parsing.
 */
export function toDriverOutcome(
  outcome: StepOutcome,
  screenshotData?: { base64?: string; media_type?: string; width?: number; height?: number },
  domSnapshot?: DOMSnapshot
): DriverOutcome {
  // Start with all StepOutcome fields, then add flat fields
  const driverOutcome: DriverOutcome = {
    ...outcome,
    // Add flat screenshot fields
    screenshot_base64: screenshotData?.base64,
    screenshot_media_type: screenshotData?.media_type,
    screenshot_width: screenshotData?.width,
    screenshot_height: screenshotData?.height,
    // Add flat DOM fields
    dom_html: domSnapshot?.html,
    dom_preview: domSnapshot?.preview,
  };

  // Remove nested objects that are replaced by flat fields
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  delete (driverOutcome as any).screenshot;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  delete (driverOutcome as any).dom_snapshot;

  return driverOutcome;
}
