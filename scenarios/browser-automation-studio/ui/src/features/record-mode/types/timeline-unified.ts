/**
 * Unified Timeline Types
 *
 * This module provides the unified TimelineEntry types from proto definitions
 * and conversion utilities for UI rendering.
 *
 * The unified types enable:
 * 1. Same data structure for recording and execution
 * 2. Real-time execution viewing on the Record page
 * 3. Simpler data transformation between layers
 *
 * See "UNIFIED RECORDING/EXECUTION MODEL" in shared.proto for design rationale.
 */

import type { RecordedAction, SelectorCandidate, SelectorSet, BoundingBox, ElementMeta } from '../types';

// Re-export the proto types for convenience
// These are generated from packages/proto/schemas/browser-automation-studio/v1/
export type {
  TimelineEntry,
  TimelineEntryAggregates,
  TimelineStreamMessage,
  TimelineStatusUpdate,
  TimelineHeartbeat,
  TimelineLog,
  TimelineArtifact,
  ElementFocus,
} from '@vrooli/proto-types/browser-automation-studio/v1/timeline/entry_pb';

export type {
  ActionDefinition,
  ActionMetadata,
} from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';

export type {
  ActionTelemetry,
} from '@vrooli/proto-types/browser-automation-studio/v1/domain/telemetry_pb';

export type {
  EventContext,
} from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';

export type {
  WorkflowNodeV2,
} from '@vrooli/proto-types/browser-automation-studio/v1/workflows/definition_pb';

// Import proto types for use in conversions
import type {
  TimelineEntry,
} from '@vrooli/proto-types/browser-automation-studio/v1/timeline/entry_pb';

// Import proto enums for type-safe conversions
import {
  ActionType as ProtoActionType,
  MouseButton as ProtoMouseButton,
  KeyboardModifier as ProtoKeyboardModifier,
} from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';

import {
  SelectorType as ProtoSelectorType,
} from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';

/**
 * Mode discriminator for timeline events.
 * - recording: Event from live recording session
 * - execution: Event from workflow execution playback
 */
export type TimelineMode = 'recording' | 'execution';

/**
 * A timeline item that can be rendered in the UI.
 * This is the common interface that works for both recording and execution.
 */
export interface TimelineItem {
  id: string;
  sequenceNum: number;
  timestamp: Date;
  durationMs?: number;
  actionType: string;
  selector?: string;
  url?: string;
  success?: boolean;
  error?: string;
  mode: TimelineMode;
  /** Raw TimelineEntry for detailed views */
  rawEntry?: TimelineEntry;
}

/**
 * Convert a legacy RecordedAction to a TimelineItem for unified rendering.
 * Used when receiving RecordedAction from legacy API responses.
 */
export function recordedActionToTimelineItem(action: RecordedAction): TimelineItem {
  return {
    id: action.id,
    sequenceNum: action.sequenceNum,
    timestamp: new Date(action.timestamp),
    durationMs: action.durationMs,
    actionType: action.actionType,
    selector: action.selector?.primary,
    url: action.url,
    success: true, // Recording actions are always successful captures
    mode: 'recording',
  };
}

/**
 * Convert a TimelineEntry (proto) to a TimelineItem for unified rendering.
 */
export function timelineEntryToTimelineItem(entry: TimelineEntry): TimelineItem {
  const actionType = getActionTypeString(entry.action?.type);
  // Determine mode from context.origin - sessionId indicates recording, executionId indicates execution
  const context = entry.context;
  const mode: TimelineMode = context?.origin?.case === 'sessionId' ? 'recording' : 'execution';

  let selector: string | undefined;
  // Try to get selector from action params
  if (entry.action?.params?.case === 'click') {
    selector = (entry.action.params.value as { selector?: string })?.selector;
  } else if (entry.action?.params?.case === 'input') {
    selector = (entry.action.params.value as { selector?: string })?.selector;
  } else if (entry.action?.params?.case === 'hover') {
    selector = (entry.action.params.value as { selector?: string })?.selector;
  } else if (entry.action?.params?.case === 'focus') {
    selector = (entry.action.params.value as { selector?: string })?.selector;
  }

  // Extract success/error from unified context
  return {
    id: entry.id,
    sequenceNum: entry.sequenceNum,
    timestamp: entry.timestamp ? timestampToDate(entry.timestamp) : new Date(),
    durationMs: entry.durationMs,
    actionType,
    selector,
    url: entry.telemetry?.url,
    success: context?.success ?? true,
    error: context?.error,
    mode,
    rawEntry: entry,
  };
}

/**
 * Convert a protobuf Timestamp to a JavaScript Date.
 */
function timestampToDate(timestamp: { seconds?: bigint; nanos?: number }): Date {
  const seconds = Number(timestamp.seconds ?? 0n);
  const millis = Math.floor((timestamp.nanos ?? 0) / 1_000_000);
  return new Date(seconds * 1000 + millis);
}

/**
 * Convert ActionType enum to display string.
 * Uses proto-generated enum values for type safety.
 */
function getActionTypeString(type: number | undefined): string {
  switch (type) {
    case ProtoActionType.NAVIGATE: return 'navigate';
    case ProtoActionType.CLICK: return 'click';
    case ProtoActionType.INPUT: return 'input';
    case ProtoActionType.WAIT: return 'wait';
    case ProtoActionType.ASSERT: return 'assert';
    case ProtoActionType.SCROLL: return 'scroll';
    case ProtoActionType.SELECT: return 'select';
    case ProtoActionType.EVALUATE: return 'evaluate';
    case ProtoActionType.KEYBOARD: return 'keyboard';
    case ProtoActionType.HOVER: return 'hover';
    case ProtoActionType.SCREENSHOT: return 'screenshot';
    case ProtoActionType.FOCUS: return 'focus';
    case ProtoActionType.BLUR: return 'blur';
    default: return 'unknown';
  }
}

/**
 * Convert SelectorType enum to string.
 * Uses proto-generated enum values for type safety.
 */
function selectorTypeToString(type: number | string | undefined): string {
  if (typeof type === 'string') return type;
  switch (type) {
    case ProtoSelectorType.CSS: return 'css';
    case ProtoSelectorType.XPATH: return 'xpath';
    case ProtoSelectorType.ID: return 'id';
    case ProtoSelectorType.DATA_TESTID: return 'data-testid';
    case ProtoSelectorType.ARIA: return 'aria';
    case ProtoSelectorType.TEXT: return 'text';
    case ProtoSelectorType.ROLE: return 'role';
    case ProtoSelectorType.PLACEHOLDER: return 'placeholder';
    case ProtoSelectorType.ALT_TEXT: return 'alt-text';
    case ProtoSelectorType.TITLE: return 'title';
    default: return 'css';
  }
}

/**
 * Convert MouseButton enum to string.
 * Uses proto-generated enum values for type safety.
 */
function mouseButtonToString(button: number | string | undefined): 'left' | 'right' | 'middle' {
  if (typeof button === 'string') {
    if (button === 'left' || button === 'right' || button === 'middle') return button;
    return 'left';
  }
  switch (button) {
    case ProtoMouseButton.LEFT: return 'left';
    case ProtoMouseButton.RIGHT: return 'right';
    case ProtoMouseButton.MIDDLE: return 'middle';
    default: return 'left';
  }
}

/**
 * Convert KeyboardModifier array to string array.
 * Uses proto-generated enum values for type safety.
 */
function keyboardModifiersToStrings(modifiers: Array<number | string> | undefined): Array<'ctrl' | 'shift' | 'alt' | 'meta'> {
  if (!modifiers) return [];
  return modifiers.map((mod) => {
    if (typeof mod === 'string') {
      if (mod === 'ctrl' || mod === 'shift' || mod === 'alt' || mod === 'meta') return mod;
      return 'ctrl'; // fallback
    }
    switch (mod) {
      case ProtoKeyboardModifier.CTRL: return 'ctrl';
      case ProtoKeyboardModifier.SHIFT: return 'shift';
      case ProtoKeyboardModifier.ALT: return 'alt';
      case ProtoKeyboardModifier.META: return 'meta';
      default: return 'ctrl';
    }
  });
}

/**
 * Convert a TimelineEntry back to a RecordedAction for legacy component compatibility.
 * This is useful when interfacing with components that still expect RecordedAction.
 */
export function timelineEntryToRecordedAction(entry: TimelineEntry): RecordedAction | null {
  if (!entry.action) return null;

  const actionType = getActionTypeString(entry.action.type);
  const selector = extractSelector(entry);
  const elementMeta = extractElementMeta(entry);
  const boundingBox = extractBoundingBox(entry);

  // Extract session/execution ID from unified context.origin
  let sessionId = '';
  const context = entry.context;
  if (context?.origin?.case === 'sessionId') {
    sessionId = context.origin.value;
  } else if (context?.origin?.case === 'executionId') {
    sessionId = context.origin.value;
  }

  return {
    id: entry.id,
    sessionId,
    sequenceNum: entry.sequenceNum,
    timestamp: entry.timestamp ? timestampToDate(entry.timestamp).toISOString() : new Date().toISOString(),
    durationMs: entry.durationMs,
    actionType: actionType as RecordedAction['actionType'],
    confidence: entry.action.metadata?.confidence ?? 1.0,
    selector,
    elementMeta,
    boundingBox,
    payload: extractPayload(entry),
    url: entry.telemetry?.url ?? '',
    frameId: entry.telemetry?.frameId,
    cursorPos: entry.telemetry?.cursorPosition
      ? { x: entry.telemetry.cursorPosition.x, y: entry.telemetry.cursorPosition.y }
      : undefined,
  };
}

function extractSelector(entry: TimelineEntry): SelectorSet | undefined {
  // Get selector from params
  let primary: string | undefined;
  const params = entry.action?.params;

  if (params?.case === 'click') primary = params.value.selector;
  else if (params?.case === 'input') primary = params.value.selector;
  else if (params?.case === 'hover') primary = params.value.selector;
  else if (params?.case === 'focus') primary = params.value.selector;
  else if (params?.case === 'assert') primary = params.value.selector;
  else if (params?.case === 'selectOption') primary = params.value.selector;
  else if (params?.case === 'scroll') primary = params.value.selector;

  if (!primary) return undefined;

  // Get candidates from metadata (unified - no more modeData separation)
  const candidates: SelectorCandidate[] = [];
  const protoSelectorCandidates = entry.action?.metadata?.selectorCandidates ?? [];

  for (const c of protoSelectorCandidates) {
    candidates.push({
      type: selectorTypeToString(c.type) as SelectorCandidate['type'],
      value: c.value,
      confidence: c.confidence,
      specificity: c.specificity,
    });
  }

  return { primary, candidates };
}

function extractElementMeta(entry: TimelineEntry): ElementMeta | undefined {
  const snapshot = entry.action?.metadata?.elementSnapshot;
  if (!snapshot) return undefined;

  const attributes: Record<string, string> = {};
  if (snapshot.attributes) {
    for (const [key, value] of Object.entries(snapshot.attributes)) {
      attributes[key] = value;
    }
  }

  return {
    tagName: snapshot.tagName,
    id: snapshot.id,
    className: snapshot.className,
    innerText: snapshot.innerText,
    attributes,
    isVisible: snapshot.isVisible,
    isEnabled: snapshot.isEnabled,
    role: snapshot.role,
    ariaLabel: snapshot.ariaLabel,
  };
}

function extractBoundingBox(entry: TimelineEntry): BoundingBox | undefined {
  // Use telemetry bounding box (live) or metadata captured bounding box (snapshot)
  // Note: recordedBoundingBox was renamed to capturedBoundingBox in unified model
  const box = entry.telemetry?.elementBoundingBox ?? entry.action?.metadata?.capturedBoundingBox;
  if (!box) return undefined;

  return {
    x: box.x,
    y: box.y,
    width: box.width,
    height: box.height,
  };
}

function extractPayload(entry: TimelineEntry): RecordedAction['payload'] {
  const params = entry.action?.params;
  if (!params || params.case === undefined) return {};

  switch (params.case) {
    case 'click':
      return {
        button: mouseButtonToString(params.value.button),
        clickCount: params.value.clickCount,
        modifiers: keyboardModifiersToStrings(params.value.modifiers),
      };
    case 'input':
      return {
        text: params.value.value,
        clearFirst: params.value.clearFirst,
      };
    case 'scroll':
      return {
        scrollX: params.value.x,
        scrollY: params.value.y,
        deltaX: params.value.deltaX,
        deltaY: params.value.deltaY,
      };
    case 'navigate':
      return {
        targetUrl: params.value.url,
      };
    case 'keyboard':
      return {
        key: params.value.key,
      };
    case 'selectOption':
      if (params.value.selectBy.case === 'value') {
        return { value: params.value.selectBy.value };
      } else if (params.value.selectBy.case === 'label') {
        return { selectedText: params.value.selectBy.value };
      } else if (params.value.selectBy.case === 'index') {
        return { selectedIndex: params.value.selectBy.value };
      }
      return {};
    default:
      return {};
  }
}

/**
 * Type guard to check if a WebSocket message contains a timeline_entry field.
 * Note: The Go API uses timeline_entry (V2 unified format).
 */
export function hasTimelineEntry(message: unknown): message is { timeline_entry: unknown } {
  return (
    typeof message === 'object' &&
    message !== null &&
    'timeline_entry' in message &&
    message.timeline_entry !== null
  );
}

/**
 * Parse a timeline_entry from a WebSocket message.
 * Returns null if parsing fails.
 */
export function parseTimelineEntry(timelineEntryJson: unknown): TimelineEntry | null {
  if (!timelineEntryJson || typeof timelineEntryJson !== 'object') {
    return null;
  }

  // The timeline_entry is already JSON - we need to convert it to the proto type
  // For now, we return it as-is since the proto types are interfaces
  // In a full implementation, you'd use protojson to properly deserialize
  return timelineEntryJson as TimelineEntry;
}
