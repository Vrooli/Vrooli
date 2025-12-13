/**
 * Unified Types - Primary Types for Browser Automation Studio
 *
 * This module provides the canonical types for recording, workflow storage,
 * and execution in BAS. These types are aligned with the proto schema
 * defined in packages/proto/schemas/browser-automation-studio/v1/unified.proto.
 *
 * KEY TYPES:
 * - TimelineEvent: Unified streaming format for recording and execution events
 * - ActionDefinition: Core action type used across recording, workflows, and execution
 * - WorkflowNodeV2: Workflow storage format with ActionDefinition
 *
 * CONVERSION UTILITIES:
 * - recordedActionToTimelineEvent: Legacy RecordedAction → TimelineEvent
 * - timelineEventToRecordedAction: TimelineEvent → Legacy RecordedAction (for backward compat)
 * - timelineEventToWorkflowNode: TimelineEvent → WorkflowNodeV2 (for saving recordings)
 *
 * NEW CODE SHOULD USE THESE TYPES DIRECTLY rather than the legacy types.ts.
 *
 * See: docs/plans/bas-unified-timeline-workflow-types.md
 */

import type { RecordedAction, SelectorCandidate as LegacySelectorCandidate, ElementMeta as LegacyElementMeta } from './types';
import type { BoundingBox, Point } from '../types/contracts';

// =============================================================================
// Re-export proto-generated types for convenience
// =============================================================================

// These types are generated from unified.proto
// We define interfaces here that mirror them for TypeScript use
// Eventually these should be imported directly from the generated code

export type ActionType =
  | 'navigate'
  | 'click'
  | 'input'
  | 'wait'
  | 'assert'
  | 'scroll'
  | 'select'
  | 'evaluate'
  | 'keyboard'
  | 'hover'
  | 'screenshot'
  | 'focus'
  | 'blur';

export interface SelectorCandidate {
  type: string;
  value: string;
  confidence: number;
  specificity: number;
}

export interface ElementSnapshot {
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

export interface ActionMetadata {
  label?: string;
  selectorCandidates?: SelectorCandidate[];
  elementSnapshot?: ElementSnapshot;
  confidence?: number;
  recordedAt?: string;
  recordedBoundingBox?: BoundingBox;
}

// Action params interfaces
export interface NavigateParams {
  url: string;
  waitForSelector?: string;
  timeoutMs?: number;
  waitUntil?: string;
}

export interface ClickParams {
  selector: string;
  button?: string;
  clickCount?: number;
  delayMs?: number;
  modifiers?: string[];
  force?: boolean;
  scrollIntoView?: boolean;
}

export interface InputParams {
  selector: string;
  value: string;
  isSensitive?: boolean;
  submit?: boolean;
  clearFirst?: boolean;
  delayMs?: number;
}

export interface WaitParams {
  durationMs?: number;
  selector?: string;
  state?: string;
  timeoutMs?: number;
}

export interface AssertParams {
  selector: string;
  mode: string;
  expected?: unknown;
  negated?: boolean;
  caseSensitive?: boolean;
  attributeName?: string;
  failureMessage?: string;
  timeoutMs?: number;
}

export interface ScrollParams {
  selector?: string;
  x?: number;
  y?: number;
  behavior?: string;
  deltaX?: number;
  deltaY?: number;
}

export interface SelectParams {
  selector: string;
  value?: string;
  label?: string;
  index?: number;
  timeoutMs?: number;
}

export interface EvaluateParams {
  expression: string;
  storeResult?: string;
  args?: Record<string, unknown>;
}

export interface KeyboardParams {
  key?: string;
  keys?: string[];
  modifiers?: string[];
  action?: string;
}

export interface HoverParams {
  selector: string;
  timeoutMs?: number;
}

export interface ScreenshotParams {
  fullPage?: boolean;
  selector?: string;
  quality?: number;
}

export interface FocusParams {
  selector: string;
  scroll?: boolean;
  timeoutMs?: number;
}

export interface BlurParams {
  selector?: string;
  timeoutMs?: number;
}

export type ActionParams =
  | { case: 'navigate'; value: NavigateParams }
  | { case: 'click'; value: ClickParams }
  | { case: 'input'; value: InputParams }
  | { case: 'wait'; value: WaitParams }
  | { case: 'assert'; value: AssertParams }
  | { case: 'scroll'; value: ScrollParams }
  | { case: 'selectOption'; value: SelectParams }
  | { case: 'evaluate'; value: EvaluateParams }
  | { case: 'keyboard'; value: KeyboardParams }
  | { case: 'hover'; value: HoverParams }
  | { case: 'screenshot'; value: ScreenshotParams }
  | { case: 'focus'; value: FocusParams }
  | { case: 'blur'; value: BlurParams }
  | { case: undefined; value?: undefined };

export interface ActionDefinition {
  type: ActionType;
  params: ActionParams;
  metadata?: ActionMetadata;
}

// Telemetry types
export interface Screenshot {
  mediaType: string;
  captureTime?: string;
  width?: number;
  height?: number;
  artifactUrl?: string;
  thumbnailUrl?: string;
  base64Data?: string;
}

export interface ConsoleLogEntry {
  level: string;
  text: string;
  timestamp: string;
  stack?: string;
  location?: string;
}

export interface NetworkEvent {
  type: string;
  url: string;
  method?: string;
  resourceType?: string;
  status?: number;
  ok?: boolean;
  failure?: string;
  timestamp: string;
}

export interface HighlightRegion {
  selector: string;
  boundingBox?: BoundingBox;
  padding?: number;
  color?: string;
}

export interface MaskRegion {
  selector: string;
  boundingBox?: BoundingBox;
  opacity?: number;
}

export interface ActionTelemetry {
  url: string;
  frameId?: string;
  screenshot?: Screenshot;
  domSnapshotPreview?: string;
  domSnapshotHtml?: string;
  elementBoundingBox?: BoundingBox;
  clickPosition?: Point;
  cursorPosition?: Point;
  cursorTrail?: Point[];
  highlightRegions?: HighlightRegion[];
  maskRegions?: MaskRegion[];
  zoomFactor?: number;
  consoleLogs?: ConsoleLogEntry[];
  networkEvents?: NetworkEvent[];
}

// Recording-specific data
export interface RecordingEventData {
  sessionId: string;
  selectorCandidates: SelectorCandidate[];
  needsConfirmation: boolean;
  source: 'auto' | 'manual';
}

// Execution-specific data
export interface RetryAttempt {
  attempt: number;
  success: boolean;
  durationMs: number;
  error?: string;
}

export interface AssertionResult {
  mode: string;
  selector: string;
  expected?: unknown;
  actual?: unknown;
  success: boolean;
  negated: boolean;
  caseSensitive: boolean;
  message?: string;
}

export interface ExecutionEventData {
  executionId: string;
  nodeId: string;
  success: boolean;
  error?: string;
  errorCode?: string;
  attempt: number;
  maxAttempts?: number;
  retryDelayMs?: number;
  retryBackoffFactor?: number;
  retryHistory?: RetryAttempt[];
  assertion?: AssertionResult;
  extractedData?: Record<string, unknown>;
}

// The unified TimelineEvent
export interface TimelineEvent {
  id: string;
  sequenceNum: number;
  timestamp: string;
  durationMs?: number;
  action: ActionDefinition;
  telemetry: ActionTelemetry;
  recording?: RecordingEventData;
  execution?: ExecutionEventData;
}

// =============================================================================
// Conversion Functions
// =============================================================================

/**
 * Convert legacy ActionType string to unified ActionType.
 */
function mapActionType(legacyType: string): ActionType {
  const mapping: Record<string, ActionType> = {
    click: 'click',
    type: 'input',        // 'type' maps to 'input' in unified schema
    input: 'input',
    scroll: 'scroll',
    navigate: 'navigate',
    select: 'select',
    hover: 'hover',
    focus: 'focus',
    blur: 'blur',
    keypress: 'keyboard', // 'keypress' maps to 'keyboard' in unified schema
    keyboard: 'keyboard',
    wait: 'wait',
    assert: 'assert',
    screenshot: 'screenshot',
    evaluate: 'evaluate',
  };

  return mapping[legacyType] || 'click'; // Default to click if unknown
}

/**
 * Convert legacy SelectorCandidate to unified SelectorCandidate.
 */
function convertSelectorCandidate(legacy: LegacySelectorCandidate): SelectorCandidate {
  return {
    type: legacy.type,
    value: legacy.value,
    confidence: legacy.confidence,
    specificity: legacy.specificity,
  };
}

/**
 * Convert legacy ElementMeta to unified ElementSnapshot.
 */
function convertElementMeta(legacy: LegacyElementMeta): ElementSnapshot {
  return {
    tagName: legacy.tagName,
    id: legacy.id,
    className: legacy.className,
    innerText: legacy.innerText,
    attributes: legacy.attributes,
    isVisible: legacy.isVisible,
    isEnabled: legacy.isEnabled,
    role: legacy.role,
    ariaLabel: legacy.ariaLabel,
  };
}

/**
 * Build ActionParams from RecordedAction based on action type.
 */
function buildActionParams(action: RecordedAction): ActionParams {
  const selector = action.selector?.primary || '';
  // Use string comparison to handle both legacy and unified action types
  const actionType: string = action.actionType;

  switch (actionType) {
    case 'navigate':
      return {
        case: 'navigate',
        value: {
          url: action.typedAction && 'navigate' in action.typedAction
            ? action.typedAction.navigate.url
            : action.payload?.targetUrl as string || action.url,
          waitForSelector: action.typedAction && 'navigate' in action.typedAction
            ? action.typedAction.navigate.waitForSelector
            : undefined,
          timeoutMs: action.typedAction && 'navigate' in action.typedAction
            ? action.typedAction.navigate.timeoutMs
            : undefined,
        },
      };

    case 'click':
      return {
        case: 'click',
        value: {
          selector,
          button: action.typedAction && 'click' in action.typedAction
            ? action.typedAction.click.button
            : action.payload?.button as string,
          clickCount: action.typedAction && 'click' in action.typedAction
            ? action.typedAction.click.clickCount
            : action.payload?.clickCount as number,
          modifiers: action.payload?.modifiers as string[],
          scrollIntoView: action.typedAction && 'click' in action.typedAction
            ? action.typedAction.click.scrollIntoView
            : undefined,
        },
      };

    case 'type':
    case 'input':
      return {
        case: 'input',
        value: {
          selector,
          value: action.typedAction && 'input' in action.typedAction
            ? action.typedAction.input.value
            : action.payload?.text as string || '',
          isSensitive: action.typedAction && 'input' in action.typedAction
            ? action.typedAction.input.isSensitive
            : undefined,
          submit: action.typedAction && 'input' in action.typedAction
            ? action.typedAction.input.submit
            : undefined,
          clearFirst: action.payload?.clearFirst as boolean,
          delayMs: action.payload?.delay as number,
        },
      };

    case 'wait':
      return {
        case: 'wait',
        value: {
          durationMs: action.typedAction && 'wait' in action.typedAction
            ? action.typedAction.wait.durationMs
            : action.payload?.ms as number,
          selector: action.payload?.selector as string,
        },
      };

    case 'assert':
      return {
        case: 'assert',
        value: {
          selector,
          mode: action.typedAction && 'assert' in action.typedAction
            ? action.typedAction.assert.mode
            : 'exists',
          expected: action.typedAction && 'assert' in action.typedAction
            ? action.typedAction.assert.expected
            : undefined,
          negated: action.typedAction && 'assert' in action.typedAction
            ? action.typedAction.assert.negated
            : undefined,
          caseSensitive: action.typedAction && 'assert' in action.typedAction
            ? action.typedAction.assert.caseSensitive
            : undefined,
        },
      };

    case 'scroll':
      return {
        case: 'scroll',
        value: {
          selector: selector || undefined,
          x: action.payload?.scrollX as number,
          y: action.payload?.scrollY as number,
          deltaX: action.payload?.deltaX as number,
          deltaY: action.payload?.deltaY as number,
        },
      };

    case 'select':
      return {
        case: 'selectOption',
        value: {
          selector,
          value: action.payload?.value as string,
          label: action.payload?.selectedText as string,
          index: action.payload?.selectedIndex as number,
        },
      };

    case 'hover':
      return {
        case: 'hover',
        value: {
          selector,
        },
      };

    case 'focus':
      return {
        case: 'focus',
        value: {
          selector,
        },
      };

    case 'blur':
      return {
        case: 'blur',
        value: {
          selector: selector || undefined,
        },
      };

    case 'keypress':
      return {
        case: 'keyboard',
        value: {
          key: action.payload?.key as string,
          modifiers: action.payload?.modifiers as string[],
        },
      };

    default:
      // Fallback for unknown types
      return {
        case: 'click',
        value: {
          selector,
        },
      };
  }
}

/**
 * Convert a legacy RecordedAction to a unified TimelineEvent.
 *
 * This is the main conversion function used during the migration period.
 * The recording controller will emit TimelineEvents, but internally
 * may still use RecordedAction temporarily.
 */
export function recordedActionToTimelineEvent(action: RecordedAction): TimelineEvent {
  const actionType = mapActionType(action.actionType);

  // Build selector candidates from legacy format
  const selectorCandidates: SelectorCandidate[] = action.selector?.candidates
    ? action.selector.candidates.map(convertSelectorCandidate)
    : [];

  // Build element snapshot from legacy format
  const elementSnapshot = action.elementMeta
    ? convertElementMeta(action.elementMeta)
    : undefined;

  // Build ActionDefinition
  const actionDef: ActionDefinition = {
    type: actionType,
    params: buildActionParams(action),
    metadata: {
      label: generateActionLabel(action),
      selectorCandidates,
      elementSnapshot,
      confidence: action.confidence,
      recordedAt: action.timestamp,
      recordedBoundingBox: action.boundingBox,
    },
  };

  // Build ActionTelemetry
  const telemetry: ActionTelemetry = {
    url: action.url,
    frameId: action.frameId,
    elementBoundingBox: action.boundingBox,
    cursorPosition: action.cursorPos,
  };

  // Build RecordingEventData
  const recording: RecordingEventData = {
    sessionId: action.sessionId,
    selectorCandidates,
    needsConfirmation: selectorCandidates.length > 1 || action.confidence < 0.8,
    source: 'auto',
  };

  return {
    id: action.id,
    sequenceNum: action.sequenceNum,
    timestamp: action.timestamp,
    durationMs: action.durationMs,
    action: actionDef,
    telemetry,
    recording,
  };
}

/**
 * Generate a human-readable label for an action.
 */
function generateActionLabel(action: RecordedAction): string {
  const type = action.actionType;
  const element = action.elementMeta;

  // Try to create a descriptive label
  if (element) {
    const tag = element.tagName;
    const text = element.innerText?.substring(0, 30);
    const id = element.id;
    const ariaLabel = element.ariaLabel;

    if (ariaLabel) {
      return `${capitalize(type)}: ${ariaLabel}`;
    }
    if (text) {
      return `${capitalize(type)}: "${text}"`;
    }
    if (id) {
      return `${capitalize(type)}: #${id}`;
    }
    return `${capitalize(type)}: ${tag}`;
  }

  // Special cases
  if (type === 'navigate') {
    const url = action.payload?.targetUrl as string || action.url;
    try {
      const parsed = new URL(url);
      return `Navigate to ${parsed.pathname || '/'}`;
    } catch {
      return `Navigate to ${url}`;
    }
  }

  return capitalize(type);
}

function capitalize(str: string): string {
  return str.charAt(0).toUpperCase() + str.slice(1);
}

/**
 * Convert a TimelineEvent back to legacy RecordedAction format.
 *
 * This is useful for backward compatibility with existing code
 * that still expects RecordedAction.
 */
export function timelineEventToRecordedAction(event: TimelineEvent): RecordedAction {
  const { action, telemetry, recording } = event;

  // Map unified action type back to legacy
  const legacyTypeMap: Record<ActionType, string> = {
    navigate: 'navigate',
    click: 'click',
    input: 'type',  // 'input' maps back to 'type' in legacy
    wait: 'wait',
    assert: 'assert',
    scroll: 'scroll',
    select: 'select',
    evaluate: 'evaluate',
    keyboard: 'keypress', // 'keyboard' maps back to 'keypress' in legacy
    hover: 'hover',
    screenshot: 'screenshot',
    focus: 'focus',
    blur: 'blur',
  };

  const actionType = legacyTypeMap[action.type] || action.type;

  // Extract selector from params
  let primarySelector = '';
  if (action.params.case && action.params.value && 'selector' in action.params.value) {
    primarySelector = (action.params.value as { selector?: string }).selector || '';
  }

  // Build legacy selector set
  const selectorCandidates = action.metadata?.selectorCandidates || recording?.selectorCandidates || [];
  const selector = primarySelector || selectorCandidates[0]?.value
    ? {
        primary: primarySelector || selectorCandidates[0]?.value || '',
        candidates: selectorCandidates.map((c) => ({
          type: c.type as LegacySelectorCandidate['type'],
          value: c.value,
          confidence: c.confidence,
          specificity: c.specificity,
        })),
      }
    : undefined;

  // Build legacy element meta
  const elementSnapshot = action.metadata?.elementSnapshot;
  const elementMeta: LegacyElementMeta | undefined = elementSnapshot
    ? {
        tagName: elementSnapshot.tagName,
        id: elementSnapshot.id,
        className: elementSnapshot.className,
        innerText: elementSnapshot.innerText,
        attributes: elementSnapshot.attributes,
        isVisible: elementSnapshot.isVisible,
        isEnabled: elementSnapshot.isEnabled,
        role: elementSnapshot.role,
        ariaLabel: elementSnapshot.ariaLabel,
      }
    : undefined;

  // Build legacy payload from params
  const payload = buildLegacyPayload(action);

  return {
    id: event.id,
    sessionId: recording?.sessionId || '',
    sequenceNum: event.sequenceNum,
    timestamp: event.timestamp,
    durationMs: event.durationMs,
    actionType: actionType as RecordedAction['actionType'],
    confidence: action.metadata?.confidence || 1,
    selector,
    elementMeta,
    boundingBox: telemetry.elementBoundingBox || action.metadata?.recordedBoundingBox,
    cursorPos: telemetry.cursorPosition,
    url: telemetry.url,
    frameId: telemetry.frameId,
    payload,
  };
}

/**
 * Build legacy payload from unified ActionParams.
 * Uses type assertions to match the strict literal types in RecordedAction['payload'].
 */
function buildLegacyPayload(action: ActionDefinition): RecordedAction['payload'] {
  const params = action.params;
  if (!params.case || !params.value) {
    return undefined;
  }

  switch (params.case) {
    case 'navigate':
      return { targetUrl: params.value.url };

    case 'click':
      return {
        button: params.value.button as 'left' | 'right' | 'middle' | undefined,
        clickCount: params.value.clickCount,
        modifiers: params.value.modifiers as Array<'ctrl' | 'shift' | 'alt' | 'meta'> | undefined,
      };

    case 'input':
      return {
        text: params.value.value,
        clearFirst: params.value.clearFirst,
        delay: params.value.delayMs,
      };

    case 'wait':
      return {
        ms: params.value.durationMs,
        selector: params.value.selector,
      };

    case 'scroll':
      return {
        scrollX: params.value.x,
        scrollY: params.value.y,
        deltaX: params.value.deltaX,
        deltaY: params.value.deltaY,
      };

    case 'selectOption':
      return {
        value: params.value.value,
        selectedText: params.value.label,
        selectedIndex: params.value.index,
      };

    case 'keyboard':
      return {
        key: params.value.key,
        modifiers: params.value.modifiers as Array<'ctrl' | 'shift' | 'alt' | 'meta'> | undefined,
      };

    default:
      return undefined;
  }
}

// =============================================================================
// Workflow Conversion
// =============================================================================

export interface WorkflowNodeV2 {
  id: string;
  action: ActionDefinition;
  position?: { x: number; y: number };
  executionSettings?: {
    timeoutMs?: number;
    waitAfterMs?: number;
    continueOnError?: boolean;
  };
}

/**
 * Convert a TimelineEvent to a WorkflowNodeV2.
 *
 * This is used when saving a recording as a workflow.
 * By default, rich metadata is preserved for debugging/fallback purposes.
 */
export function timelineEventToWorkflowNode(
  event: TimelineEvent,
  nodeId: string,
  options?: {
    preserveMetadata?: boolean;
    position?: { x: number; y: number };
  }
): WorkflowNodeV2 {
  const { action } = event;
  const preserveMetadata = options?.preserveMetadata ?? true;

  const workflowAction: ActionDefinition = {
    type: action.type,
    params: action.params,
  };

  // Optionally preserve rich metadata from recording
  if (preserveMetadata && action.metadata) {
    workflowAction.metadata = {
      ...action.metadata,
      // Keep selector candidates for fallback during execution
      selectorCandidates: action.metadata.selectorCandidates,
      // Keep element snapshot for debugging
      elementSnapshot: action.metadata.elementSnapshot,
    };
  }

  return {
    id: nodeId,
    action: workflowAction,
    position: options?.position,
  };
}

/**
 * Convert multiple TimelineEvents to WorkflowNodeV2 array with auto-layout.
 */
export function timelineEventsToWorkflow(
  events: TimelineEvent[],
  options?: {
    preserveMetadata?: boolean;
    startPosition?: { x: number; y: number };
    nodeSpacing?: number;
  }
): WorkflowNodeV2[] {
  const startX = options?.startPosition?.x ?? 250;
  const startY = options?.startPosition?.y ?? 100;
  const spacing = options?.nodeSpacing ?? 120;

  return events.map((event, index) => {
    return timelineEventToWorkflowNode(event, `node_${index + 1}`, {
      preserveMetadata: options?.preserveMetadata,
      position: { x: startX, y: startY + index * spacing },
    });
  });
}
