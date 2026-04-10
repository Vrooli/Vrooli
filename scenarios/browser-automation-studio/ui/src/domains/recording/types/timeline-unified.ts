/**
 * Unified Timeline Types
 *
 * This module provides timeline data structures and conversion utilities that work
 * across both recording and execution modes. It's one of three reconciliation
 * systems in browser-automation-studio.
 *
 * ## Reconciliation Context
 *
 * This file implements AI step reconciliation - correlating AI navigation decisions
 * with recorded browser actions. See docs/architecture/reconciliation.md for the
 * full architecture covering all three systems.
 *
 * ## Key Responsibilities
 *
 * 1. **Type Unification**: Same TimelineItem structure for recording and execution
 * 2. **Format Conversion**: RecordedAction ↔ TimelineEntry ↔ TimelineItem
 * 3. **AI Correlation**: mergeActionsWithAISteps() matches AI reasoning to actions
 * 4. **Workflow Mapping**: Convert between workflow nodes and timeline items
 *
 * ## Related Files
 *
 * - api/services/workflow/sync.go (backend filesystem-DB sync)
 * - utils/mergeActions.ts (frontend action deduplication)
 * - RecordingSession.tsx (usage context, lines 639-651)
 *
 * See "UNIFIED RECORDING/EXECUTION MODEL" in shared.proto for proto design rationale.
 */

import type { RecordedAction, SelectorCandidate, SelectorSet, BoundingBox, ElementMeta } from './types';

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

// Import centralized timestamp utility
import { protoTimestampToDate } from '../../../utils/timestamps';

/**
 * Mode discriminator for timeline events.
 * - recording: Event from live recording session
 * - execution: Event from workflow execution playback
 */
export type TimelineMode = 'recording' | 'execution';

/**
 * Page event types for multi-tab recording.
 */
export type PageEventType = 'page_created' | 'page_navigated' | 'page_closed';

/**
 * AI metadata for actions performed by AI navigation.
 */
export interface AIMetadata {
  /** AI reasoning for why this action was taken */
  reasoning?: string;
  /** Tokens used for this step */
  tokensUsed?: {
    promptTokens: number;
    completionTokens: number;
    totalTokens: number;
  };
  /** Whether the AI achieved its goal with this action */
  goalAchieved?: boolean;
}

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
  /** Page ID for multi-tab recording */
  pageId?: string;
  /** Type of entry: action or page_event */
  entryType?: 'action' | 'page_event';
  /** Page event type (for page events only) */
  pageEventType?: PageEventType;
  /** Page title at the time of the event */
  pageTitle?: string;
  /** Whether this action was performed by AI navigation */
  isAI?: boolean;
  /** AI-specific metadata (reasoning, tokens, etc.) */
  aiMetadata?: AIMetadata;
}

/**
 * Convert a legacy RecordedAction to a TimelineItem for unified rendering.
 * Used when receiving RecordedAction from legacy API responses.
 *
 * @param action - The recorded action to convert
 * @param aiMetadata - Optional AI metadata if this action was performed by AI navigation
 */
export function recordedActionToTimelineItem(
  action: RecordedAction,
  aiMetadata?: AIMetadata,
): TimelineItem {
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
    pageId: action.pageId,
    entryType: 'action',
    pageTitle: action.pageTitle,
    isAI: aiMetadata !== undefined,
    aiMetadata,
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
    timestamp: entry.timestamp ? protoTimestampToDate(entry.timestamp) ?? new Date() : new Date(),
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
 * TimelineEntry type from useTimeline hook.
 * Re-declared here to avoid circular imports.
 */
export interface UseTimelineEntry {
  id: string;
  type: 'action' | 'page_event';
  timestamp: string;
  pageId: string;
  action?: {
    id: string;
    actionType: string;
    url?: string;
    sequenceNum: number;
    timestamp: string;
    selector?: { primary: string };
    payload?: Record<string, unknown>;
    confidence: number;
    pageTitle?: string;
  };
  pageEvent?: {
    id: string;
    type: 'page_created' | 'page_navigated' | 'page_closed';
    pageId: string;
    url?: string;
    title?: string;
    openerId?: string;
    timestamp: string;
  };
}

/**
 * Convert a useTimeline TimelineEntry to a TimelineItem.
 * Handles both action entries and page event entries.
 * This is for entries from the /timeline API endpoint (useTimeline hook).
 */
export function useTimelineEntryToTimelineItem(
  entry: UseTimelineEntry,
): TimelineItem {
  // Handle page events
  if (entry.type === 'page_event' && entry.pageEvent) {
    return {
      id: entry.id,
      sequenceNum: 0, // Page events don't have sequence numbers
      timestamp: new Date(entry.timestamp),
      actionType: entry.pageEvent.type, // page_created, page_navigated, page_closed
      mode: 'recording',
      pageId: entry.pageId,
      entryType: 'page_event',
      pageEventType: entry.pageEvent.type,
      url: entry.pageEvent.url,
      pageTitle: entry.pageEvent.title,
    };
  }

  // Handle action entries
  if (entry.action) {
    return {
      id: entry.action.id,
      sequenceNum: entry.action.sequenceNum,
      timestamp: new Date(entry.action.timestamp),
      actionType: entry.action.actionType,
      selector: entry.action.selector?.primary,
      url: entry.action.url,
      success: true, // Recording actions are successful captures
      mode: 'recording',
      pageId: entry.pageId,
      entryType: 'action',
      pageTitle: entry.action.pageTitle,
    };
  }

  // Fallback for malformed entries
  return {
    id: entry.id,
    sequenceNum: 0,
    timestamp: new Date(entry.timestamp),
    actionType: 'unknown',
    mode: 'recording',
    entryType: 'action',
  };
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
    timestamp: entry.timestamp ? (protoTimestampToDate(entry.timestamp)?.toISOString() ?? new Date().toISOString()) : new Date().toISOString(),
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
 * Execution status for timeline items during workflow execution.
 */
export type ExecutionStatus = 'pending' | 'running' | 'completed' | 'failed';

/**
 * Extended TimelineItem with execution status for workflow execution.
 */
export interface ExecutionTimelineItem extends TimelineItem {
  /** Execution status for this step */
  executionStatus: ExecutionStatus;
  /** Node ID from the workflow (for matching with execution events) */
  nodeId: string;
}

// ============================================================================
// Workflow ↔ Timeline Bidirectional Conversion
// ============================================================================

/**
 * Node types that should NOT appear in the timeline.
 * These are structural/control nodes, not action nodes.
 */
const NON_ACTION_NODE_TYPES = new Set([
  'start',
  'end',
  'entry',
  'exit',
  'group',
  'comment',
  'note',
  'annotation',
]);

/**
 * Check if a node is an action node that should appear in the timeline.
 * Handles both V2 format (node.action.type) and legacy format (node.type).
 */
function isActionNode(node: { type?: string; action?: { type: string } }): boolean {
  // Get the effective type from either format
  const nodeType = node.type ?? node.action?.type;
  if (!nodeType) return false;

  // Normalize V2 format (ACTION_TYPE_NAVIGATE -> navigate)
  const normalizedType = nodeType.replace('ACTION_TYPE_', '').toLowerCase();

  return !NON_ACTION_NODE_TYPES.has(normalizedType);
}

/**
 * Extract action type from a workflow node.
 * Handles both V2 format (node.action.type) and legacy format (node.type).
 */
function getNodeActionType(node: { type?: string; action?: { type: string } }): string {
  if (node.action?.type) {
    // V2 format: ACTION_TYPE_NAVIGATE -> navigate
    return node.action.type.replace('ACTION_TYPE_', '').toLowerCase();
  }
  if (node.type) {
    return node.type.toLowerCase();
  }
  return 'unknown';
}

/**
 * Convert workflow nodes to timeline items for pre-populating the timeline.
 * Uses STORED ORDER (array index order) - not topological sort.
 * This preserves the visual order from the workflow builder.
 *
 * @param nodes - React Flow nodes from the workflow
 * @param _edges - React Flow edges (currently unused, kept for future use)
 * @returns Timeline items with pending execution status
 */
export function workflowNodesToTimelineItems(
  nodes: Array<{ id: string; type?: string; data?: Record<string, unknown>; action?: { type: string; metadata?: { label?: string }; navigate?: { url?: string } } }>,
  _edges: Array<{ source: string; target: string }>
): ExecutionTimelineItem[] {
  console.log('[workflowNodesToTimelineItems] Input nodes:', nodes?.length ?? 0, nodes);

  if (!nodes || nodes.length === 0) {
    console.log('[workflowNodesToTimelineItems] No nodes provided');
    return [];
  }

  const items: ExecutionTimelineItem[] = [];
  let sequenceNum = 1;

  // Use stored order (array index) - filter out non-action nodes
  for (const node of nodes) {
    const nodeTypeInfo = {
      id: node.id,
      type: node.type,
      actionType: node.action?.type,
      isAction: isActionNode(node),
    };
    console.log('[workflowNodesToTimelineItems] Processing node:', nodeTypeInfo);

    // Skip non-action nodes
    if (!isActionNode(node)) {
      console.log('[workflowNodesToTimelineItems] Skipping non-action node:', node.id);
      continue;
    }

    const actionType = getNodeActionType(node);

    // Get label for display
    const label = node.action?.metadata?.label ?? (node.data?.label as string | undefined);

    // Get selector if available
    const selector = node.data?.selector as string | undefined;

    // Get URL for navigate nodes
    let url: string | undefined;
    if (actionType === 'navigate') {
      url = (node.data?.url as string) ?? node.action?.navigate?.url;
    }

    items.push({
      id: `workflow-step-${node.id}`,
      nodeId: node.id,
      sequenceNum: sequenceNum++,
      timestamp: new Date(), // Will be updated during execution
      actionType,
      selector,
      url,
      mode: 'execution',
      executionStatus: 'pending',
      entryType: 'action',
      // Use label as page title for display
      pageTitle: label,
    });
  }

  console.log('[workflowNodesToTimelineItems] Converted', items.length, 'of', nodes.length, 'nodes to timeline items');
  return items;
}

/**
 * Convert timeline items back to workflow nodes.
 * This is the reverse operation for editing workflows from timeline.
 *
 * @param items - Timeline items to convert
 * @returns Workflow nodes (without position data - caller must add)
 */
export function timelineItemsToWorkflowNodes(
  items: TimelineItem[]
): Array<{ id: string; type: string; data: Record<string, unknown> }> {
  return items
    .filter(item => item.entryType === 'action')
    .map(item => ({
      id: (item as ExecutionTimelineItem).nodeId ?? item.id,
      type: item.actionType,
      data: {
        label: item.pageTitle ?? item.actionType,
        selector: item.selector,
        url: item.url,
      },
    }));
}

/**
 * Update timeline items with execution event data.
 * Matches by node_id and updates status.
 *
 * @param items - Current timeline items
 * @param nodeId - Node ID from execution event
 * @param status - New execution status
 * @param error - Optional error message
 * @param durationMs - Optional duration in milliseconds
 * @returns Updated timeline items
 */
export function updateTimelineItemStatus(
  items: ExecutionTimelineItem[],
  nodeId: string,
  status: ExecutionStatus,
  error?: string,
  durationMs?: number
): ExecutionTimelineItem[] {
  return items.map(item => {
    if (item.nodeId === nodeId) {
      return {
        ...item,
        executionStatus: status,
        success: status === 'completed' ? true : status === 'failed' ? false : undefined,
        error,
        durationMs: durationMs ?? item.durationMs,
        timestamp: status === 'running' ? new Date() : item.timestamp,
      };
    }
    return item;
  });
}

/**
 * AI Navigation Step interface (imported types to avoid circular deps).
 * This mirrors the AINavigationStep from ai-navigation/types.ts.
 */
interface AIStepForMerge {
  id: string;
  stepNumber: number;
  action: {
    type: string;
    text?: string;
    url?: string;
  };
  reasoning: string;
  currentUrl: string;
  goalAchieved: boolean;
  tokensUsed: {
    promptTokens: number;
    completionTokens: number;
    totalTokens: number;
  };
  durationMs: number;
  error?: string;
  timestamp: Date;
}

/**
 * Normalize AI action types to match recorded action types.
 *
 * The AI navigation system uses different terminology than the browser recorder:
 * - AI says "type" for text input, recorder says "input"
 * - AI says "keypress" for keyboard events, recorder says "keyboard"
 * - AI says "done" when goal is achieved (no direct recorder equivalent)
 *
 * This normalization enables timestamp-based matching in mergeActionsWithAISteps().
 */
function normalizeActionType(aiActionType: string): string {
  const mapping: Record<string, string> = {
    type: 'input',      // AI "type" = recorder "input" (text entry)
    keypress: 'keyboard', // AI "keypress" = recorder "keyboard" (key events)
    done: 'wait',       // AI "done" has no direct equivalent
  };
  return mapping[aiActionType] ?? aiActionType;
}

/**
 * Correlate AI navigation steps with recorded browser actions.
 *
 * ## Problem
 *
 * When AI drives the browser, two parallel event streams exist:
 * 1. **AI steps**: High-level decisions with reasoning, token usage, goal status
 * 2. **Recorded actions**: Low-level browser events captured by the recorder
 *
 * Users want to see both together: what happened AND why the AI did it.
 *
 * ## Solution
 *
 * Match AI steps to recorded actions using:
 * 1. **Timestamp proximity**: Must be within 5 seconds of each other
 * 2. **Action type matching**: Types must match (after normalization)
 * 3. **Greedy best-match**: Select the closest match, consume it to prevent duplicates
 *
 * ## Why 5-second window?
 *
 * - Typical action latency: 50-500ms (network, rendering)
 * - AI processing time: 500ms-2s (LLM inference)
 * - Safety margin for edge cases
 * - Tight enough to avoid false positives with unrelated actions
 *
 * ## Why greedy matching?
 *
 * - Each AI step should match at most one recorded action
 * - Prevents the same reasoning from appearing on multiple actions
 * - Simple to understand and debug
 *
 * @param actions - Recorded actions from the browser session
 * @param aiSteps - AI navigation steps with reasoning and metadata
 * @returns TimelineItems with AI metadata attached where matches were found
 */
export function mergeActionsWithAISteps(
  actions: RecordedAction[],
  aiSteps: AIStepForMerge[],
): TimelineItem[] {
  // Fast path: no AI steps means no correlation needed
  if (aiSteps.length === 0) {
    return actions.map((action) => recordedActionToTimelineItem(action));
  }

  // Working copy of AI steps - we remove matched steps to prevent duplicate attribution.
  // Using splice() for removal makes this O(n*m) worst case, but m is typically small (<50).
  const unmatchedSteps = [...aiSteps];

  return actions.map((action) => {
    const actionTime = new Date(action.timestamp).getTime();
    const normalizedActionType = action.actionType;

    // Greedy best-match: find the AI step with minimum time delta that also matches type
    let bestMatchIndex = -1;
    let bestMatchTimeDiff = Infinity;

    for (let i = 0; i < unmatchedSteps.length; i++) {
      const step = unmatchedSteps[i];
      if (!step) continue;
      const stepTime = step.timestamp.getTime();
      const timeDiff = Math.abs(actionTime - stepTime);

      // Both criteria must be satisfied:
      // 1. Within 5-second window (see function docs for rationale)
      // 2. Action types match (after normalization)
      const normalizedStepType = normalizeActionType(step.action.type);
      if (timeDiff < 5000 && normalizedStepType === normalizedActionType) {
        if (timeDiff < bestMatchTimeDiff) {
          bestMatchTimeDiff = timeDiff;
          bestMatchIndex = i;
        }
      }
    }

    if (bestMatchIndex >= 0) {
      // Match found: consume the AI step (remove from pool) and attach its metadata
      const matchedStep = unmatchedSteps.splice(bestMatchIndex, 1)[0];
      if (matchedStep) {
        return recordedActionToTimelineItem(action, {
          reasoning: matchedStep.reasoning,
          tokensUsed: matchedStep.tokensUsed,
          goalAchieved: matchedStep.goalAchieved,
        });
      }
    }

    // No match: return action without AI context (human action or unmatched AI action)
    return recordedActionToTimelineItem(action);
  });
}
