/**
 * Performance Tracing Module
 *
 * Driver-side CDP performance tracing + web-vitals capture for BAS's
 * CAPTURE_TYPE_PERFORMANCE artifact (P2 of the performance-health buildout).
 *
 * NOTE: distinct from `src/performance/` (frame-streaming pipeline timing).
 * This module captures the page's DevTools timeline trace, not the driver's
 * own frame-capture latency.
 */

export {
  PerformanceTracer,
  injectWebVitalsObserver,
  collectTrace,
  PERF_TRACE_FILE,
  PERF_WEB_VITALS_FILE,
  PERF_TRACE_CATEGORIES,
  type TracingCDP,
  type VitalsPage,
} from './performance-tracer';
export { WEB_VITALS_GLOBAL, WEB_VITALS_INIT_SCRIPT } from './web-vitals-script';
export {
  AccessibilitySnapshotter,
  normalizeAccessibilityTree,
  parseDomSnapshot,
  deriveStates,
  ACCESSIBILITY_SNAPSHOT_FILE,
  ACCESSIBILITY_CONTRACT,
  ACCESSIBILITY_MAX_NODES,
  type AccessibilityCDP,
  type RawAXNode,
  type AXValue,
  type AXProperty,
  type DomNodeInfo,
  type DOMSnapshotResult,
  type NormalizedAXNode,
  type AccessibilitySnapshot,
  type SnapshotMeta,
} from './accessibility-snapshot';
