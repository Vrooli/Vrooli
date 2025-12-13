/**
 * Unified Timeline Types
 *
 * This module bridges between the legacy RecordedAction types used in the UI
 * and the new unified TimelineEvent types from the proto definitions.
 *
 * The unified types enable:
 * 1. Same data structure for recording and execution
 * 2. Real-time execution viewing on the Record page
 * 3. Simpler data transformation between layers
 */

import type { RecordedAction, SelectorCandidate, SelectorSet, BoundingBox, ElementMeta } from '../types';

// Re-export the proto types for convenience
// These are generated from packages/proto/schemas/browser-automation-studio/v1/unified.proto
export type {
  TimelineEvent,
  ActionDefinition,
  ActionTelemetry,
  RecordingEventData,
  ExecutionEventData,
  WorkflowNodeV2,
  ActionMetadata,
  ActionType,
} from '@vrooli/proto-types/browser-automation-studio/v1/unified_pb';

export {
  ActionType as ActionTypeEnum,
} from '@vrooli/proto-types/browser-automation-studio/v1/unified_pb';

// Import proto types for use in conversions
import type {
  TimelineEvent,
} from '@vrooli/proto-types/browser-automation-studio/v1/unified_pb';

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
  // Raw event for detailed views
  rawEvent?: TimelineEvent;
  // Legacy action for backward compatibility
  legacyAction?: RecordedAction;
}

/**
 * Convert a legacy RecordedAction to a TimelineItem for unified rendering.
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
    legacyAction: action,
  };
}

/**
 * Convert a TimelineEvent (proto) to a TimelineItem for unified rendering.
 */
export function timelineEventToTimelineItem(event: TimelineEvent): TimelineItem {
  const actionType = getActionTypeString(event.action?.type);
  const mode: TimelineMode = event.modeData.case === 'recording' ? 'recording' : 'execution';

  let selector: string | undefined;
  // Try to get selector from action params
  if (event.action?.params.case === 'click') {
    selector = event.action.params.value.selector;
  } else if (event.action?.params.case === 'input') {
    selector = event.action.params.value.selector;
  } else if (event.action?.params.case === 'hover') {
    selector = event.action.params.value.selector;
  } else if (event.action?.params.case === 'focus') {
    selector = event.action.params.value.selector;
  }

  // Extract success/error for execution mode
  const executionData = event.modeData.case === 'execution' ? event.modeData.value : undefined;

  return {
    id: event.id,
    sequenceNum: event.sequenceNum,
    timestamp: event.timestamp ? timestampToDate(event.timestamp) : new Date(),
    durationMs: event.durationMs,
    actionType,
    selector,
    url: event.telemetry?.url,
    success: mode === 'execution' ? executionData?.success : true,
    error: executionData?.error,
    mode,
    rawEvent: event,
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
 */
function getActionTypeString(type: number | undefined): string {
  switch (type) {
    case 1: return 'navigate';
    case 2: return 'click';
    case 3: return 'input';
    case 4: return 'wait';
    case 5: return 'assert';
    case 6: return 'scroll';
    case 7: return 'select';
    case 8: return 'evaluate';
    case 9: return 'keyboard';
    case 10: return 'hover';
    case 11: return 'screenshot';
    case 12: return 'focus';
    case 13: return 'blur';
    default: return 'unknown';
  }
}

/**
 * Convert a TimelineEvent back to a RecordedAction for legacy component compatibility.
 * This is useful during the migration period.
 */
export function timelineEventToRecordedAction(event: TimelineEvent): RecordedAction | null {
  if (!event.action) return null;

  const actionType = getActionTypeString(event.action.type);
  const selector = extractSelector(event);
  const elementMeta = extractElementMeta(event);
  const boundingBox = extractBoundingBox(event);

  // Extract session/execution ID from modeData
  let sessionId = '';
  if (event.modeData.case === 'recording') {
    sessionId = event.modeData.value.sessionId;
  } else if (event.modeData.case === 'execution') {
    sessionId = event.modeData.value.executionId;
  }

  return {
    id: event.id,
    sessionId,
    sequenceNum: event.sequenceNum,
    timestamp: event.timestamp ? timestampToDate(event.timestamp).toISOString() : new Date().toISOString(),
    durationMs: event.durationMs,
    actionType: actionType as RecordedAction['actionType'],
    confidence: event.action.metadata?.confidence ?? 1.0,
    selector,
    elementMeta,
    boundingBox,
    payload: extractPayload(event),
    url: event.telemetry?.url ?? '',
    frameId: event.telemetry?.frameId,
    cursorPos: event.telemetry?.cursorPosition
      ? { x: event.telemetry.cursorPosition.x, y: event.telemetry.cursorPosition.y }
      : undefined,
  };
}

function extractSelector(event: TimelineEvent): SelectorSet | undefined {
  // Get selector from params
  let primary: string | undefined;
  const params = event.action?.params;

  if (params?.case === 'click') primary = params.value.selector;
  else if (params?.case === 'input') primary = params.value.selector;
  else if (params?.case === 'hover') primary = params.value.selector;
  else if (params?.case === 'focus') primary = params.value.selector;
  else if (params?.case === 'assert') primary = params.value.selector;
  else if (params?.case === 'selectOption') primary = params.value.selector;
  else if (params?.case === 'scroll') primary = params.value.selector;

  if (!primary) return undefined;

  // Get candidates from recording data or metadata
  const candidates: SelectorCandidate[] = [];
  const recordingData = event.modeData.case === 'recording' ? event.modeData.value : undefined;
  const protoSelectorCandidates =
    recordingData?.selectorCandidates ?? event.action?.metadata?.selectorCandidates ?? [];

  for (const c of protoSelectorCandidates) {
    candidates.push({
      type: c.type as SelectorCandidate['type'],
      value: c.value,
      confidence: c.confidence,
      specificity: c.specificity,
    });
  }

  return { primary, candidates };
}

function extractElementMeta(event: TimelineEvent): ElementMeta | undefined {
  const snapshot = event.action?.metadata?.elementSnapshot;
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

function extractBoundingBox(event: TimelineEvent): BoundingBox | undefined {
  const box = event.telemetry?.elementBoundingBox ?? event.action?.metadata?.recordedBoundingBox;
  if (!box) return undefined;

  return {
    x: box.x,
    y: box.y,
    width: box.width,
    height: box.height,
  };
}

function extractPayload(event: TimelineEvent): RecordedAction['payload'] {
  const params = event.action?.params;
  if (!params || params.case === undefined) return {};

  switch (params.case) {
    case 'click':
      return {
        button: params.value.button as 'left' | 'right' | 'middle',
        clickCount: params.value.clickCount,
        modifiers: params.value.modifiers as Array<'ctrl' | 'shift' | 'alt' | 'meta'>,
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
 * Type guard to check if a WebSocket message contains a timeline_event field.
 */
export function hasTimelineEvent(message: unknown): message is { timeline_event: unknown } {
  return (
    typeof message === 'object' &&
    message !== null &&
    'timeline_event' in message &&
    message.timeline_event !== null
  );
}

/**
 * Parse a timeline_event from a WebSocket message.
 * Returns null if parsing fails.
 */
export function parseTimelineEvent(timelineEventJson: unknown): TimelineEvent | null {
  if (!timelineEventJson || typeof timelineEventJson !== 'object') {
    return null;
  }

  // The timeline_event is already JSON - we need to convert it to the proto type
  // For now, we return it as-is since the proto types are interfaces
  // In a full implementation, you'd use protojson to properly deserialize
  return timelineEventJson as TimelineEvent;
}
