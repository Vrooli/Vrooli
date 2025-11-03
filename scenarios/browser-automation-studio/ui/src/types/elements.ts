/**
 * Shared types for element selection and information
 */

export interface SelectorOption {
  selector: string;
  type: string;
  robustness: number;
  fallback: boolean;
}

export interface BoundingBox {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface ElementInfo {
  text: string;
  tagName: string;
  type: string;
  selectors: SelectorOption[];
  boundingBox: BoundingBox;
  bounding_box?: BoundingBox;
  confidence: number;
  category: string;
  attributes: Record<string, string>;
}

export interface ElementHierarchyEntry {
  element: ElementInfo;
  selector: string;
  depth: number;
  path: string[];
  pathSummary?: string;
}

export interface ElementCoordinateResponse {
  element: ElementInfo | null;
  candidates: ElementHierarchyEntry[];
  selectedIndex: number;
}

/**
 * Timeline frame data from API
 */
export interface TimelineFramePayload {
  screenshot?: {
    url?: string;
    thumbnail_url?: string;
    artifact_id?: string;
    width?: number;
    height?: number;
    content_type?: string;
    size_bytes?: number;
  } | null;
  focused_element?: {
    selector?: string;
    bounding_box?: BoundingBox;
    boundingBox?: BoundingBox;
  };
  focusedElement?: {
    selector?: string;
    bounding_box?: BoundingBox;
    boundingBox?: BoundingBox;
  };
  element_bounding_box?: BoundingBox;
  elementBoundingBox?: BoundingBox;
  click_position?: { x: number; y: number };
  clickPosition?: { x: number; y: number };
  cursor_trail?: Array<{ x: number; y: number; timestamp?: number }>;
  cursorTrail?: Array<{ x: number; y: number; timestamp?: number }>;
  highlight_regions?: BoundingBox[];
  highlightRegions?: BoundingBox[];
  mask_regions?: BoundingBox[];
  maskRegions?: BoundingBox[];
  step_index?: number;
  node_id?: string;
  step_type?: string;
  status?: string;
  success?: boolean;
  duration_ms?: number;
  durationMs?: number;
  total_duration_ms?: number;
  totalDurationMs?: number;
  progress?: number;
  final_url?: string;
  finalUrl?: string;
  error?: string;
  extracted_data_preview?: unknown;
  extractedDataPreview?: unknown;
  console_log_count?: number;
  consoleLogCount?: number;
  network_event_count?: number;
  networkEventCount?: number;
  timeline_artifact_id?: string;
  zoom_factor?: number;
  zoomFactor?: number;
  retry_attempt?: number;
  retryAttempt?: number;
  retry_max_attempts?: number;
  retryMaxAttempts?: number;
  retry_configured?: number;
  retryConfigured?: number;
  retry_delay_ms?: number;
  retryDelayMs?: number;
  retry_backoff_factor?: number;
  retryBackoffFactor?: number;
  retry_history?: Array<{
    attempt?: number;
    duration_ms?: number;
    success?: boolean;
    error?: string;
  }>;
  retryHistory?: Array<{
    attempt?: number;
    duration_ms?: number;
    success?: boolean;
    error?: string;
  }>;
  dom_snapshot_preview?: string;
  domSnapshotPreview?: string;
  dom_snapshot_artifact_id?: string;
  domSnapshotArtifactId?: string;
  started_at?: string | Date;
  assertion?: AssertionPayload;
  artifacts?: Array<{
    type?: string;
    id?: string;
    payload?: {
      html?: string;
      [key: string]: unknown;
    };
  }>;
}

/**
 * Assertion data in event payloads
 */
export interface AssertionPayload {
  selector?: string;
  mode?: string;
  success?: boolean;
  message?: string;
}

/**
 * Console log entry in telemetry
 */
export interface ConsoleLogEntry {
  type?: string;
  text?: string;
  timestamp?: string | number;
}

/**
 * Network event entry in telemetry
 */
export interface NetworkEventEntry {
  type?: string;
  method?: string;
  status?: number;
  url?: string;
  failure?: string;
  timestamp?: string | number;
}

/**
 * Telemetry payload structure
 */
export interface TelemetryPayload {
  console_logs?: ConsoleLogEntry[];
  network_events?: NetworkEventEntry[];
}
