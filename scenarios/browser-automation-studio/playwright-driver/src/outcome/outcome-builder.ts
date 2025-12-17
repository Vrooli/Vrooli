/**
 * Outcome Builder
 *
 * Domain logic for constructing StepOutcome from handler results.
 * This module owns the transformation from internal HandlerResult to proto StepOutcome.
 *
 * PROTO TYPES:
 * This module uses proto-generated types from @vrooli/proto-types.
 * The output is serialized to snake_case JSON via protojson for wire compatibility.
 *
 * Responsibilities:
 * - Build proto StepOutcome from handler execution results
 * - Serialize StepOutcome to JSON wire format (DriverOutcome)
 * - Normalize telemetry data for contract compliance
 */

import { create, toJson } from '@bufbuild/protobuf';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import {
  StepOutcomeSchema,
  DriverScreenshotSchema,
  DOMSnapshotSchema,
  DriverConsoleLogEntrySchema,
  DriverNetworkEventSchema,
  StepFailureSchema,
  FailureSource,
  type StepOutcome,
} from '@vrooli/proto-types/browser-automation-studio/v1/execution/driver_pb';
import {
  BoundingBoxSchema,
} from '@vrooli/proto-types/browser-automation-studio/v1/base/geometry_pb';
import {
  ElementFocusSchema,
} from '@vrooli/proto-types/browser-automation-studio/v1/timeline/entry_pb';

import { safeDuration, validateStepIndex, safeSerializable } from '../utils';
import type { BaseExecutionResult, HandlerError } from './types';

// Re-export HandlerError for consumers that import from this module
export type { HandlerError };

// =============================================================================
// HANDLER RESULT TYPE
// =============================================================================
// HandlerError is now imported from types.ts (single source of truth)

/**
 * Result of handler execution with optional extracted data.
 * Handlers return this, which is then converted to proto StepOutcome.
 *
 * Extends BaseExecutionResult with handler-specific telemetry fields
 * (screenshots, DOM snapshots, console logs, network events).
 *
 * Note: Handlers don't track their own duration - that's added by the calling
 * route layer when building StepOutcome. Hence durationMs is optional via base.
 *
 * @see BaseExecutionResult - Base interface defining success/error contract
 * @see ActionReplayResult - Recording replay variant in recording/action-executor.ts
 * @see HandlerAdapterResult - Minimal adapter variant in recording/handler-adapter.ts
 */
export interface HandlerResult extends Omit<BaseExecutionResult, 'error'> {
  /** Handler-specific error with flexible kind type */
  error?: HandlerError;
  /** Extracted data from the page (text, attributes, etc.) */
  extracted_data?: Record<string, unknown>;
  /** Information about the focused element */
  focus?: {
    selector?: string;
    bounding_box?: {
      x: number;
      y: number;
      width: number;
      height: number;
    };
  };
  /** Screenshot captured during execution */
  screenshot?: Screenshot;
  /** DOM snapshot captured during execution */
  domSnapshot?: DOMSnapshot;
  /** Console log entries collected during execution */
  consoleLogs?: ConsoleLogEntry[];
  /** Network events captured during execution */
  networkEvents?: NetworkEvent[];
  /** Assertion outcome for assert instructions */
  assertion?: HandlerAssertionOutcome;
}

// =============================================================================
// HANDLER ASSERTION OUTCOME
// =============================================================================
// Handler-friendly assertion result type (uses strings, not proto JsonValue).
// Converted to proto AssertionOutcome in buildStepOutcome.

/**
 * Handler assertion outcome (handler-friendly format).
 * Uses plain strings for expected/actual instead of proto JsonValue.
 */
export interface HandlerAssertionOutcome {
  mode?: string;
  selector?: string;
  expected?: string;
  actual?: string;
  success: boolean;
  negated?: boolean;
  caseSensitive?: boolean;
  message?: string;
}

// =============================================================================
// TELEMETRY TYPES
// =============================================================================
// These types are used for telemetry data within handlers and outcome building.
// They use snake_case for wire format compatibility with the Go API.

/**
 * Screenshot telemetry data.
 */
export interface Screenshot {
  base64?: string;
  media_type?: string;
  capture_time?: string;
  width?: number;
  height?: number;
  hash?: string;
  from_cache?: boolean;
  truncated?: boolean;
  source?: string;
}

/**
 * DOM snapshot telemetry data.
 */
export interface DOMSnapshot {
  html?: string;
  preview?: string;
  hash?: string;
  collected_at?: string;
  truncated?: boolean;
}

/**
 * Console log entry telemetry.
 */
export interface ConsoleLogEntry {
  type: string;
  text: string;
  timestamp: string;
  stack?: string;
  location?: string;
}

/**
 * Network event telemetry.
 */
export interface NetworkEvent {
  type: string;
  url: string;
  method?: string;
  resource_type?: string;
  status?: number;
  ok?: boolean;
  failure?: string;
  timestamp: string;
  request_headers?: Record<string, string>;
  response_headers?: Record<string, string>;
  request_body_preview?: string;
  response_body_preview?: string;
  truncated?: boolean;
}

/**
 * DriverOutcome - Wire format for HTTP responses.
 * This is what the Go API expects to receive.
 */
export interface DriverOutcome {
  [key: string]: unknown;
}

// =============================================================================
// ENUM CONVERSION HELPERS
// =============================================================================

// Import normalizeFailureKind from shared types module
import { normalizeFailureKind } from './types';

// Alias for backward compatibility within this file
const toFailureKind = normalizeFailureKind;

// =============================================================================
// CONSTANTS
// =============================================================================

const SCHEMA_VERSION = 'automation-step-outcome-v1';
const PAYLOAD_VERSION = '1';

// =============================================================================
// BUILD PARAMS TYPE
// =============================================================================

import type { HandlerInstruction } from '../proto';

/**
 * Parameters for building a step outcome
 */
export interface BuildOutcomeParams {
  instruction: HandlerInstruction;
  result: HandlerResult;
  startedAt: Date;
  completedAt: Date;
  finalUrl: string;
  screenshot?: Screenshot;
  domSnapshot?: DOMSnapshot;
  consoleLogs?: ConsoleLogEntry[];
  networkEvents?: NetworkEvent[];
}

// =============================================================================
// MAIN BUILDER FUNCTION
// =============================================================================

/**
 * Build a proto StepOutcome from execution results.
 *
 * This is the canonical way to construct outcomes - handlers should not
 * build StepOutcome directly but instead return HandlerResult.
 *
 * @returns Proto StepOutcome message
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

  // Validate and compute timing
  const validatedIndex = validateStepIndex(instruction.index, 'buildStepOutcome');
  const durationMs = safeDuration(startedAt, completedAt);

  // Create base outcome
  const outcome = create(StepOutcomeSchema, {
    schemaVersion: SCHEMA_VERSION,
    payloadVersion: PAYLOAD_VERSION,
    stepIndex: validatedIndex,
    attempt: 1, // TODO: Track actual attempt number when retry logic is implemented
    nodeId: instruction.nodeId,
    stepType: instruction.type,
    success: result.success,
    startedAt: timestampFromDate(startedAt),
    completedAt: timestampFromDate(completedAt),
    durationMs,
    finalUrl,
    notes: {},
  });

  // Add screenshot telemetry
  if (screenshot) {
    outcome.screenshot = create(DriverScreenshotSchema, {
      data: screenshot.base64 ? base64ToUint8Array(screenshot.base64) : new Uint8Array(),
      mediaType: screenshot.media_type ?? 'image/png',
      width: screenshot.width ?? 0,
      height: screenshot.height ?? 0,
      hash: screenshot.hash,
      fromCache: screenshot.from_cache ?? false,
      truncated: screenshot.truncated ?? false,
      source: screenshot.source,
      captureTime: screenshot.capture_time ? timestampFromDate(new Date(screenshot.capture_time)) : undefined,
    });
  }

  // Add DOM snapshot
  if (domSnapshot) {
    outcome.domSnapshot = create(DOMSnapshotSchema, {
      html: domSnapshot.html,
      preview: domSnapshot.preview,
      hash: domSnapshot.hash,
      collectedAt: domSnapshot.collected_at ? timestampFromDate(new Date(domSnapshot.collected_at)) : undefined,
      truncated: domSnapshot.truncated ?? false,
    });
  }

  // Add console logs
  if (consoleLogs && consoleLogs.length > 0) {
    outcome.consoleLogs = consoleLogs.map(log =>
      create(DriverConsoleLogEntrySchema, {
        type: log.type,
        text: log.text,
        timestamp: timestampFromDate(new Date(log.timestamp)),
        stack: log.stack,
        location: log.location,
      })
    );
  }

  // Add network events
  if (networkEvents && networkEvents.length > 0) {
    outcome.networkEvents = networkEvents.map(event =>
      create(DriverNetworkEventSchema, {
        type: event.type,
        url: event.url,
        method: event.method,
        resourceType: event.resource_type,
        status: event.status,
        ok: event.ok,
        failure: event.failure,
        timestamp: timestampFromDate(new Date(event.timestamp)),
        requestHeaders: event.request_headers ?? {},
        responseHeaders: event.response_headers ?? {},
        requestBodyPreview: event.request_body_preview,
        responseBodyPreview: event.response_body_preview,
        truncated: event.truncated ?? false,
      })
    );
  }

  // Add extracted data
  if (result.extracted_data) {
    const safeData = safeSerializable(result.extracted_data, 'extracted_data');
    outcome.extractedData = objectToJsonValueMap(safeData);
  }

  // Add focused element info
  if (result.focus) {
    outcome.focusedElement = create(ElementFocusSchema, {
      selector: result.focus.selector ?? '',
      boundingBox: result.focus.bounding_box
        ? create(BoundingBoxSchema, result.focus.bounding_box)
        : undefined,
    });
  }

  // Add failure details
  if (!result.success && result.error) {
    outcome.failure = create(StepFailureSchema, {
      kind: toFailureKind(result.error.kind),
      code: result.error.code,
      message: result.error.message,
      retryable: result.error.retryable ?? false,
      fatal: false,
      source: FailureSource.ENGINE,
    });
  }

  return outcome;
}

// =============================================================================
// SERIALIZATION
// =============================================================================

/**
 * Convert StepOutcome to DriverOutcome (wire format).
 *
 * The driver wire format uses the proto JSON serialization which produces
 * snake_case field names for Go API compatibility.
 *
 * For screenshots, we need to include the base64 data as a flat field
 * for backward compatibility with the Go API's current parsing.
 */
export function toDriverOutcome(
  outcome: StepOutcome,
  screenshotData?: { base64?: string; media_type?: string; width?: number; height?: number },
  domSnapshot?: DOMSnapshot
): DriverOutcome {
  // Serialize proto to JSON (produces snake_case)
  const json = toJson(StepOutcomeSchema, outcome) as Record<string, unknown>;

  // For backward compatibility, add flat screenshot fields
  // The Go API expects these flat fields in addition to the nested screenshot object
  if (screenshotData?.base64) {
    json['screenshot_base64'] = screenshotData.base64;
    json['screenshot_media_type'] = screenshotData.media_type;
    json['screenshot_width'] = screenshotData.width;
    json['screenshot_height'] = screenshotData.height;
  }

  // Add flat DOM fields
  if (domSnapshot) {
    json['dom_html'] = domSnapshot.html;
    json['dom_preview'] = domSnapshot.preview;
  }

  return json;
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

/**
 * Convert base64 string to Uint8Array.
 */
function base64ToUint8Array(base64: string): Uint8Array {
  const binaryString = Buffer.from(base64, 'base64');
  return new Uint8Array(binaryString);
}

/**
 * Convert a plain object to a map of proto JsonValue messages.
 */
function objectToJsonValueMap(obj: Record<string, unknown>): Record<string, import('@vrooli/proto-types/common/v1/types_pb').JsonValue> {
  // Dynamic require to avoid circular dependency issues
  const { create: createProto } = require('@bufbuild/protobuf');
  const { JsonValueSchema: JVS, JsonObjectSchema: JOS, JsonListSchema: JLS } = require('@vrooli/proto-types/common/v1/types_pb');

  function toJsonValue(value: unknown): import('@vrooli/proto-types/common/v1/types_pb').JsonValue {
    if (value === null || value === undefined) {
      return createProto(JVS, { kind: { case: 'nullValue', value: 0 } });
    }
    if (typeof value === 'boolean') {
      return createProto(JVS, { kind: { case: 'boolValue', value } });
    }
    if (typeof value === 'number') {
      if (Number.isInteger(value)) {
        return createProto(JVS, { kind: { case: 'intValue', value: BigInt(value) } });
      }
      return createProto(JVS, { kind: { case: 'doubleValue', value } });
    }
    if (typeof value === 'string') {
      return createProto(JVS, { kind: { case: 'stringValue', value } });
    }
    if (Array.isArray(value)) {
      return createProto(JVS, {
        kind: {
          case: 'listValue',
          value: createProto(JLS, { values: value.map(toJsonValue) }),
        },
      });
    }
    if (typeof value === 'object') {
      const fields: Record<string, import('@vrooli/proto-types/common/v1/types_pb').JsonValue> = {};
      for (const [k, v] of Object.entries(value as Record<string, unknown>)) {
        fields[k] = toJsonValue(v);
      }
      return createProto(JVS, {
        kind: {
          case: 'objectValue',
          value: createProto(JOS, { fields }),
        },
      });
    }
    return createProto(JVS, { kind: { case: 'stringValue', value: String(value) } });
  }

  const result: Record<string, import('@vrooli/proto-types/common/v1/types_pb').JsonValue> = {};
  for (const [key, value] of Object.entries(obj)) {
    result[key] = toJsonValue(value);
  }
  return result;
}
