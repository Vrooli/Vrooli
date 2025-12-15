/**
 * Recording Type Conversion Utilities
 *
 * This module handles conversion from RawBrowserEvent (browser-originated)
 * to proto TimelineEntry (the canonical wire format).
 *
 * DESIGN:
 * The recording controller captures raw browser events via page.exposeFunction().
 * These are then converted directly to TimelineEntry for:
 *   - In-memory buffering during recording
 *   - WebSocket streaming to the UI
 *   - HTTP callback streaming to the API
 *   - Batch retrieval via REST endpoints
 *
 * This ensures the wire format uses proto types as the single source of truth.
 *
 * PROTO-FIRST ARCHITECTURE:
 * RawBrowserEvent → TimelineEntry (proto) → JSON (wire format)
 * No intermediate types - we go directly from browser boundary to proto.
 */

import { create, toJson } from '@bufbuild/protobuf';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { v4 as uuidv4 } from 'uuid';
import {
  TimelineEntrySchema,
  type TimelineEntry,
} from '@vrooli/proto-types/browser-automation-studio/v1/timeline/entry_pb';
import {
  ActionDefinitionSchema,
  ActionMetadataSchema,
  NavigateParamsSchema,
  ClickParamsSchema,
  InputParamsSchema,
  ScrollParamsSchema,
  SelectParamsSchema,
  HoverParamsSchema,
  FocusParamsSchema,
  BlurParamsSchema,
  KeyboardParamsSchema,
  ActionType,
  MouseButton,
  KeyboardModifier,
  type ActionDefinition,
} from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import {
  ActionTelemetrySchema,
  type ActionTelemetry,
} from '@vrooli/proto-types/browser-automation-studio/v1/domain/telemetry_pb';
import {
  SelectorCandidateSchema,
  ElementMetaSchema,
  type SelectorCandidate,
  type ElementMeta,
} from '@vrooli/proto-types/browser-automation-studio/v1/domain/selectors_pb';
import {
  SelectorType,
  EventContextSchema,
  RecordingSource,
  type EventContext,
} from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';
import {
  BoundingBoxSchema,
  PointSchema,
  type BoundingBox,
  type Point,
} from '@vrooli/proto-types/browser-automation-studio/v1/base/geometry_pb';

// =============================================================================
// RAW TYPES (browser-originated, before conversion to proto)
// =============================================================================
// These types represent data as received from the browser injector script.
// The browser can't import proto types, so it sends plain objects with
// string enum values that we convert here.

/**
 * Raw selector candidate as sent by browser injector.
 * Uses string type instead of proto enum since browser can't import proto.
 */
export interface RawSelectorCandidate {
  type: string;
  value: string;
  confidence: number;
  specificity: number;
}

/**
 * Raw element metadata as sent by browser injector.
 */
export interface RawElementMeta {
  tagName: string;
  id?: string;
  className?: string;
  innerText?: string;
  attributes?: Record<string, string>;
  isVisible: boolean;
  isEnabled: boolean;
  role?: string;
  ariaLabel?: string;
}

/**
 * Raw selector set as sent by browser injector.
 */
export interface RawSelectorSet {
  primary: string;
  candidates: RawSelectorCandidate[];
}

/**
 * Raw browser event captured by injected JavaScript.
 * This is the format received from page.exposeFunction() before conversion to proto.
 */
export interface RawBrowserEvent {
  actionType: string;
  timestamp: number;
  selector: RawSelectorSet;
  elementMeta: RawElementMeta;
  boundingBox?: { x: number; y: number; width: number; height: number };
  cursorPos?: { x: number; y: number };
  url: string;
  frameId?: string | null;
  payload?: Record<string, unknown>;
}

// =============================================================================
// INTERNAL TYPES (for conversion context)
// =============================================================================

/**
 * Conversion context passed to rawBrowserEventToTimelineEntry.
 */
export interface ConversionContext {
  sessionId: string;
  sequenceNum: number;
}


// =============================================================================
// ENUM CONVERSION MAPS
// =============================================================================

/**
 * Map from raw action type string to proto ActionType enum.
 */
const ACTION_TYPE_MAP: Record<string, ActionType> = {
  navigate: ActionType.NAVIGATE,
  click: ActionType.CLICK,
  type: ActionType.INPUT,
  input: ActionType.INPUT,
  scroll: ActionType.SCROLL,
  select: ActionType.SELECT,
  hover: ActionType.HOVER,
  focus: ActionType.FOCUS,
  blur: ActionType.BLUR,
  keypress: ActionType.KEYBOARD,
  keydown: ActionType.KEYBOARD,
  keyup: ActionType.KEYBOARD,
  change: ActionType.SELECT,
};

/**
 * Map string selector type to proto SelectorType enum.
 */
const SELECTOR_TYPE_MAP: Record<string, SelectorType> = {
  css: SelectorType.CSS,
  xpath: SelectorType.XPATH,
  id: SelectorType.ID,
  'data-testid': SelectorType.DATA_TESTID,
  'data-attr': SelectorType.DATA_TESTID,
  aria: SelectorType.ARIA,
  text: SelectorType.TEXT,
  role: SelectorType.ROLE,
  placeholder: SelectorType.PLACEHOLDER,
  'alt-text': SelectorType.ALT_TEXT,
  title: SelectorType.TITLE,
};

/**
 * Map legacy mouse button strings to proto MouseButton enum.
 */
const MOUSE_BUTTON_MAP: Record<string, MouseButton> = {
  left: MouseButton.LEFT,
  right: MouseButton.RIGHT,
  middle: MouseButton.MIDDLE,
};

/**
 * Map legacy modifier strings to proto KeyboardModifier enum.
 */
const MODIFIER_MAP: Record<string, KeyboardModifier> = {
  ctrl: KeyboardModifier.CTRL,
  shift: KeyboardModifier.SHIFT,
  alt: KeyboardModifier.ALT,
  meta: KeyboardModifier.META,
};

// =============================================================================
// MAIN CONVERSION FUNCTION
// =============================================================================

/**
 * Convert a RawBrowserEvent directly to a proto TimelineEntry.
 *
 * This is the primary conversion function for recording. It converts
 * browser-originated events directly to the canonical proto format,
 * eliminating the need for intermediate types.
 *
 * @param raw - Raw browser event from page.exposeFunction()
 * @param ctx - Conversion context (sessionId, sequenceNum)
 * @returns Proto TimelineEntry ready for buffering/streaming
 */
export function rawBrowserEventToTimelineEntry(
  raw: RawBrowserEvent,
  ctx: ConversionContext
): TimelineEntry {
  // Generate unique ID
  const id = uuidv4();

  // Normalize timestamp
  const timestamp = normalizeTimestamp(raw.timestamp);

  // Map action type
  const actionType = ACTION_TYPE_MAP[raw.actionType.toLowerCase()] ?? ActionType.UNSPECIFIED;

  // Convert raw types to proto
  const selectorCandidates = rawSelectorSetToProtoCandidates(raw.selector);
  const elementMeta = rawElementMetaToProto(raw.elementMeta);
  const boundingBox = raw.boundingBox ? rawBoundingBoxToProto(raw.boundingBox) : undefined;
  const cursorPos = raw.cursorPos ? rawPointToProto(raw.cursorPos) : undefined;

  // Build the entry
  const entry = create(TimelineEntrySchema, {
    id,
    sequenceNum: ctx.sequenceNum,
    timestamp: timestampFromDate(timestamp),
  });

  // Build action definition with params
  entry.action = buildActionDefinition(
    actionType,
    raw,
    selectorCandidates,
    elementMeta,
    boundingBox
  );

  // Build telemetry
  entry.telemetry = buildActionTelemetry(raw, boundingBox, cursorPos);

  // Build context
  entry.context = buildEventContext(ctx.sessionId);

  return entry;
}

/**
 * Create a navigate TimelineEntry for synthetic navigation events.
 * Used when capturing initial URL or URL changes during recording.
 */
export function createNavigateTimelineEntry(
  url: string,
  ctx: ConversionContext
): TimelineEntry {
  const id = uuidv4();
  const timestamp = new Date();

  const entry = create(TimelineEntrySchema, {
    id,
    sequenceNum: ctx.sequenceNum,
    timestamp: timestampFromDate(timestamp),
  });

  // Build navigate action
  entry.action = create(ActionDefinitionSchema, {
    type: ActionType.NAVIGATE,
    metadata: create(ActionMetadataSchema, {
      confidence: 1, // Navigation is always high confidence
      capturedAt: timestampFromDate(timestamp),
    }),
    params: {
      case: 'navigate',
      value: create(NavigateParamsSchema, { url }),
    },
  });

  // Build telemetry
  entry.telemetry = create(ActionTelemetrySchema, { url });

  // Build context
  entry.context = buildEventContext(ctx.sessionId);

  return entry;
}

/**
 * Convert a TimelineEntry to JSON wire format (snake_case).
 */
export function timelineEntryToJson(entry: TimelineEntry): Record<string, unknown> {
  return toJson(TimelineEntrySchema, entry) as Record<string, unknown>;
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

/**
 * Normalize timestamp from raw event.
 * Handles invalid/missing timestamps by falling back to current time.
 */
function normalizeTimestamp(rawTimestamp: number): Date {
  try {
    const date = new Date(rawTimestamp);
    if (Number.isNaN(date.getTime())) {
      return new Date();
    }
    return date;
  } catch {
    return new Date();
  }
}

/**
 * Build ActionDefinition from raw event data.
 */
function buildActionDefinition(
  actionType: ActionType,
  raw: RawBrowserEvent,
  selectorCandidates: SelectorCandidate[],
  elementMeta: ElementMeta,
  boundingBox?: BoundingBox
): ActionDefinition {
  const primarySelector = raw.selector?.primary ?? '';
  const confidence = calculateConfidence(actionType, raw.selector);
  const timestamp = normalizeTimestamp(raw.timestamp);

  const definition = create(ActionDefinitionSchema, {
    type: actionType,
    metadata: create(ActionMetadataSchema, {
      confidence,
      capturedAt: timestampFromDate(timestamp),
      selectorCandidates,
      elementSnapshot: elementMeta,
      capturedBoundingBox: boundingBox,
    }),
  });

  // Set type-specific params
  const payload = (raw.payload || {}) as Record<string, unknown>;

  switch (actionType) {
    case ActionType.NAVIGATE:
      definition.params = {
        case: 'navigate',
        value: create(NavigateParamsSchema, {
          url: (payload.targetUrl as string) ?? raw.url,
        }),
      };
      break;

    case ActionType.CLICK:
      definition.params = {
        case: 'click',
        value: create(ClickParamsSchema, {
          selector: primarySelector,
          button: MOUSE_BUTTON_MAP[(payload.button as string) ?? 'left'] ?? MouseButton.LEFT,
          clickCount: (payload.clickCount as number) ?? 1,
          modifiers: ((payload.modifiers as string[]) ?? [])
            .map((m) => MODIFIER_MAP[m])
            .filter((m): m is KeyboardModifier => m !== undefined),
        }),
      };
      break;

    case ActionType.INPUT:
      definition.params = {
        case: 'input',
        value: create(InputParamsSchema, {
          selector: primarySelector,
          value: (payload.text as string) ?? (payload.value as string) ?? '',
          clearFirst: (payload.clearFirst as boolean) ?? false,
          delayMs: (payload.delay as number) ?? undefined,
        }),
      };
      break;

    case ActionType.SCROLL:
      definition.params = {
        case: 'scroll',
        value: create(ScrollParamsSchema, {
          selector: primarySelector || undefined,
          x: (payload.scrollX as number) ?? undefined,
          y: (payload.scrollY as number) ?? undefined,
          deltaX: (payload.deltaX as number) ?? undefined,
          deltaY: (payload.deltaY as number) ?? undefined,
        }),
      };
      break;

    case ActionType.SELECT:
      definition.params = {
        case: 'selectOption',
        value: create(SelectParamsSchema, {
          selector: primarySelector,
          selectBy: payload.value
            ? { case: 'value', value: payload.value as string }
            : payload.selectedText
              ? { case: 'label', value: payload.selectedText as string }
              : payload.selectedIndex !== undefined
                ? { case: 'index', value: payload.selectedIndex as number }
                : undefined,
        }),
      };
      break;

    case ActionType.HOVER:
      definition.params = {
        case: 'hover',
        value: create(HoverParamsSchema, { selector: primarySelector }),
      };
      break;

    case ActionType.FOCUS:
      definition.params = {
        case: 'focus',
        value: create(FocusParamsSchema, { selector: primarySelector }),
      };
      break;

    case ActionType.BLUR:
      definition.params = {
        case: 'blur',
        value: create(BlurParamsSchema, { selector: primarySelector || undefined }),
      };
      break;

    case ActionType.KEYBOARD:
      definition.params = {
        case: 'keyboard',
        value: create(KeyboardParamsSchema, {
          key: (payload.key as string) ?? undefined,
          modifiers: ((payload.modifiers as string[]) ?? [])
            .map((m) => MODIFIER_MAP[m])
            .filter((m): m is KeyboardModifier => m !== undefined),
        }),
      };
      break;
  }

  return definition;
}

/**
 * Build ActionTelemetry from raw event data.
 */
function buildActionTelemetry(
  raw: RawBrowserEvent,
  boundingBox?: BoundingBox,
  cursorPos?: Point
): ActionTelemetry {
  return create(ActionTelemetrySchema, {
    url: raw.url || '',
    frameId: raw.frameId ?? undefined,
    elementBoundingBox: boundingBox,
    cursorPosition: cursorPos,
  });
}

/**
 * Build EventContext from session ID.
 */
function buildEventContext(sessionId: string): EventContext {
  return create(EventContextSchema, {
    origin: {
      case: 'sessionId',
      value: sessionId,
    },
    source: RecordingSource.AUTO,
  });
}

/**
 * Calculate confidence score for an action based on selector quality.
 */
function calculateConfidence(
  actionType: ActionType,
  selector?: RawSelectorSet
): number {
  // Actions without selectors don't have selector-based confidence issues
  if (actionType === ActionType.NAVIGATE || actionType === ActionType.SCROLL) {
    return 1;
  }

  if (!selector || !selector.candidates || selector.candidates.length === 0) {
    return 0.5;
  }

  const primaryCandidate = selector.candidates.find((c) => c.value === selector.primary);
  if (!primaryCandidate) {
    return 0.5;
  }

  // If we found a stable signal (data-testid/id/aria/data-*), bump to a safe floor
  const strongTypes = ['data-testid', 'id', 'aria', 'data-attr'];
  if (strongTypes.includes(primaryCandidate.type) && (primaryCandidate.confidence ?? 0.5) < 0.85) {
    return Math.max(primaryCandidate.confidence ?? 0.5, 0.85);
  }

  return primaryCandidate.confidence ?? 0.5;
}

// =============================================================================
// RAW TO PROTO CONVERSION FUNCTIONS
// =============================================================================

/**
 * Convert raw selector candidates to proto SelectorCandidate[].
 */
function rawSelectorSetToProtoCandidates(raw: RawSelectorSet): SelectorCandidate[] {
  if (!raw || !raw.candidates) {
    return [];
  }
  return raw.candidates.map((c) =>
    create(SelectorCandidateSchema, {
      type: SELECTOR_TYPE_MAP[c.type] ?? SelectorType.CSS,
      value: c.value,
      confidence: c.confidence,
      specificity: c.specificity,
    })
  );
}

/**
 * Convert raw element meta to proto ElementMeta.
 */
function rawElementMetaToProto(raw: RawElementMeta): ElementMeta {
  return create(ElementMetaSchema, {
    tagName: raw.tagName,
    id: raw.id ?? '',
    className: raw.className ?? '',
    innerText: raw.innerText ?? '',
    isVisible: raw.isVisible,
    isEnabled: raw.isEnabled,
    role: raw.role ?? '',
    ariaLabel: raw.ariaLabel ?? '',
    attributes: raw.attributes ?? {},
  });
}

/**
 * Convert raw bounding box to proto BoundingBox.
 */
function rawBoundingBoxToProto(raw: { x: number; y: number; width: number; height: number }): BoundingBox {
  return create(BoundingBoxSchema, {
    x: raw.x,
    y: raw.y,
    width: raw.width,
    height: raw.height,
  });
}

/**
 * Convert raw point to proto Point.
 */
function rawPointToProto(raw: { x: number; y: number }): Point {
  return create(PointSchema, {
    x: raw.x,
    y: raw.y,
  });
}

// =============================================================================
// RE-EXPORTS FOR CONVENIENCE
// =============================================================================

export type { TimelineEntry } from '@vrooli/proto-types/browser-automation-studio/v1/timeline/entry_pb';
export { TimelineEntrySchema } from '@vrooli/proto-types/browser-automation-studio/v1/timeline/entry_pb';
export { ActionType } from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
