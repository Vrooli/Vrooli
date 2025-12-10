/**
 * Type definitions matching Go automation/contracts package.
 * These must stay in sync with the Go contract definitions.
 *
 * Reference: api/automation/contracts/*.go
 *
 * STABILITY: STABLE CONTRACT
 *
 * These types define the wire format between the Playwright driver and the Go API.
 * Changes here require coordinated updates to the Go contracts package.
 *
 * - CompiledInstruction: Stable - input from Go
 * - StepOutcome: Stable - output to Go
 * - DriverOutcome: Stable - wire format
 * - Telemetry types: Stable - only additive changes
 *
 * Adding new fields is safe (Go uses omitempty).
 * Removing or renaming fields is a BREAKING CHANGE.
 */

export const STEP_OUTCOME_SCHEMA_VERSION = 'automation-step-outcome-v1';
export const PAYLOAD_VERSION = '1';

/**
 * CompiledInstruction - matches Go contracts.CompiledInstruction
 */
export interface CompiledInstruction {
  index: number;
  node_id: string;
  type: string;
  params: Record<string, unknown>;
  preload_html?: string;
  context?: Record<string, unknown>;
  metadata?: Record<string, string>;
}

/**
 * StepOutcome - matches Go contracts.StepOutcome
 */
export interface StepOutcome {
  schema_version: string;
  payload_version: string;
  execution_id?: string;
  correlation_id?: string;
  step_index: number;
  attempt: number;
  node_id: string;
  step_type: string;
  instruction?: string;
  success: boolean;
  started_at: string; // ISO 8601
  completed_at?: string; // ISO 8601
  duration_ms?: number;
  final_url?: string;
  screenshot?: Screenshot;
  dom_snapshot?: DOMSnapshot;
  console_logs?: ConsoleLogEntry[];
  network?: NetworkEvent[];
  extracted_data?: Record<string, unknown>;
  assertion?: AssertionOutcome;
  condition?: ConditionOutcome;
  element_bounding_box?: BoundingBox;
  click_position?: Point;
  focused_element?: ElementFocus;
  highlight_regions?: HighlightRegion[];
  mask_regions?: MaskRegion[];
  zoom_factor?: number;
  cursor_trail?: CursorPosition[];
  notes?: Record<string, string>;
  failure?: StepFailure;
}

/**
 * StepFailure - matches Go contracts.StepFailure
 */
export interface StepFailure {
  kind: FailureKind;
  code?: string;
  message?: string;
  fatal?: boolean;
  retryable?: boolean;
  occurred_at?: string; // ISO 8601
  details?: Record<string, unknown>;
  source?: FailureSource;
}

export type FailureKind =
  | 'engine'
  | 'infra'
  | 'orchestration'
  | 'user'
  | 'timeout'
  | 'cancelled';

export type FailureSource = 'engine' | 'executor' | 'recorder';

export interface Screenshot {
  media_type?: string;
  capture_time?: string; // ISO 8601
  width?: number;
  height?: number;
  hash?: string;
  from_cache?: boolean;
  truncated?: boolean;
  source?: string;
}

export interface DOMSnapshot {
  html?: string;
  preview?: string;
  hash?: string;
  collected_at?: string; // ISO 8601
  truncated?: boolean;
}

export interface ConsoleLogEntry {
  type: 'log' | 'warn' | 'error' | 'info' | 'debug';
  text: string;
  timestamp: string; // ISO 8601
  stack?: string;
  location?: string;
}

export interface NetworkEvent {
  type: 'request' | 'response' | 'failure';
  url: string;
  method?: string;
  resource_type?: string;
  status?: number;
  ok?: boolean;
  failure?: string;
  timestamp: string; // ISO 8601
  request_headers?: Record<string, string>;
  response_headers?: Record<string, string>;
  request_body_preview?: string;
  response_body_preview?: string;
  truncated?: boolean;
}

export interface AssertionOutcome {
  mode?: string;
  selector?: string;
  expected?: unknown;
  actual?: unknown;
  success: boolean;
  negated?: boolean;
  case_sensitive?: boolean;
  message?: string;
}

export interface ConditionOutcome {
  type?: string;
  outcome: boolean;
  negated?: boolean;
  operator?: string;
  variable?: string;
  selector?: string;
  expression?: string;
  actual?: unknown;
  expected?: unknown;
}

export interface BoundingBox {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface Point {
  x: number;
  y: number;
}

export interface ElementFocus {
  selector: string;
  bounding_box?: BoundingBox;
}

export interface HighlightRegion {
  selector: string;
  bounding_box?: BoundingBox;
  padding?: number;
  color?: string;
}

export interface MaskRegion {
  selector: string;
  bounding_box?: BoundingBox;
  opacity?: number;
}

export interface CursorPosition {
  point: Point;
  recorded_at?: string; // ISO 8601
  elapsed_ms?: number;
}

/**
 * Driver-specific outcome that includes inline data
 * (base64 screenshot, DOM HTML) before being decoded
 * into the StepOutcome format.
 */
export interface DriverOutcome extends Omit<StepOutcome, 'screenshot' | 'dom_snapshot'> {
  screenshot_base64?: string;
  screenshot_media_type?: string;
  screenshot_width?: number;
  screenshot_height?: number;
  dom_html?: string;
  dom_preview?: string;
  video_path?: string;
  trace_path?: string;
  har_path?: string;
}
