import { useMemo, useRef, useState } from 'react';
import { useRecordMode } from '../hooks/useRecordMode';
import { useActionSelection } from '../hooks/useActionSelection';
import { usePages } from '../hooks/usePages';
import { useWorkspaceTimeline } from '../hooks/useWorkspaceTimeline';
import { mergeTimelineItemsWithAISteps, recordingEntryToRecordedAction, recordingEntryToTimelineItem, type TimelineItem, type TimelineMode } from '../types/timeline-unified';
import type { RecordedAction } from '../types/types';
import type { TimelineAction } from '../api/schemas';
import { fromJson, type JsonValue } from '@bufbuild/protobuf';
import { TimelineEntrySchema as ProtoTimelineEntrySchema } from '@vrooli/proto-types/browser-automation-studio/v1/timeline/entry_pb';
import { useUnifiedSidebar, useAISettings } from '../sidebar';
import { useAIConversation } from '../ai-conversation';
import { useSessionStore } from '../stores/sessionStore';

export interface RecordingModeStateConfig {
  sessionId: string | null;
  mode: TimelineMode;
  executionId?: string | null;
  autoStartAI?: boolean;
  aiModel?: string;
  aiMaxSteps?: number;
  workflowNodes?: import('./useExecutionModeState').WorkflowNode[];
  workflowEdges?: import('./useExecutionModeState').WorkflowEdge[];
}

const PAGE_COLORS = [
  'bg-blue-500', 'bg-green-500', 'bg-purple-500', 'bg-orange-500',
  'bg-pink-500', 'bg-cyan-500', 'bg-yellow-500', 'bg-red-500',
] as const;

export function useRecordingModeState({
  sessionId,
  mode,
  executionId,
  autoStartAI,
  aiModel,
  aiMaxSteps,
  workflowNodes,
  workflowEdges,
}: RecordingModeStateConfig) {
  const [recentActivityPageId, setRecentActivityPageId] = useState<string | null>(null);
  const switchPageRef = useRef<((pageId: string) => Promise<void>) | null>(null);
  const {
    isOpen: isSidebarOpen,
    setIsOpen: setSidebarOpen,
    toggleOpen: handleSidebarToggle,
    activeTab: sidebarActiveTab,
    setActiveTab: setSidebarTab,
    setAutoActivity,
  } = useUnifiedSidebar({ initialTab: autoStartAI ? 'auto' : 'timeline' });
  const { settings: aiSettings, updateSettings: updateAISettings } = useAISettings({
    initialSettings: { model: aiModel, maxSteps: aiMaxSteps },
  });
  const {
    messages: aiMessages,
    sendMessage: aiSendMessage,
    abortNavigation: aiAbortNavigation,
    resumeNavigation: aiResumeNavigation,
    clearConversation: aiClearConversation,
    isNavigating: aiIsNavigating,
    navigationSteps: aiSteps,
    availableModels: aiAvailableModels,
    humanIntervention: aiHumanIntervention,
  } = useAIConversation({
    sessionId,
    settings: aiSettings,
    onTimelineAction: () => setAutoActivity(true),
  });
  const recordMode = useRecordMode({ sessionId });
  const pages = usePages({
    sessionId,
    onPageCreated: (page) => {
      setRecentActivityPageId(page.id);
      setTimeout(() => setRecentActivityPageId(null), 2000);
      if (useSessionStore.getState().activePageId !== page.id) void switchPageRef.current?.(page.id);
    },
  });
  switchPageRef.current = pages.switchToPage;
  const {
    entries: timelineEntries,
    executionItems: timelineItems,
    updateEntries,
    isLive: isTimelineLive,
  } = useWorkspaceTimeline({
    mode,
    isRecording: recordMode.isRecording,
    sessionId,
    pages: pages.openPages,
    executionId,
    workflowNodes,
    workflowEdges,
  });
  const identifiedActions = useMemo(() => timelineEntries
    .map(recordingEntryToRecordedAction)
    .filter((action): action is RecordedAction => action !== undefined), [timelineEntries]);
  const recordedTimelineItems = useMemo(() => timelineEntries.map(recordingEntryToTimelineItem), [timelineEntries]);
  const timelineDisplayItems = useMemo(
    () => mode !== 'recording' ? timelineItems : mergeTimelineItemsWithAISteps(recordedTimelineItems, aiSteps),
    [mode, timelineItems, recordedTimelineItems, aiSteps],
  );
  const pageColorMap = useMemo(() => {
    const colors = new Map<string, typeof PAGE_COLORS[number]>();
    pages.openPages.forEach((page, index) => colors.set(page.id, PAGE_COLORS[index % PAGE_COLORS.length] ?? PAGE_COLORS[0]));
    return colors;
  }, [pages.openPages]);
  const timelineItemCount = mode === 'recording' ? timelineDisplayItems.length : timelineItems.length;
  const selection = useActionSelection({ actionCount: timelineItemCount });
  const selectedActionIds = useMemo(() => selection.selectedIndicesArray
    .map((index) => timelineDisplayItems[index])
    .filter((item): item is TimelineItem => item !== undefined && item.entryType !== 'page_event')
    .map((item) => item.id), [selection.selectedIndicesArray, timelineDisplayItems]);
  const updateAction = (timelineIndex: number, update: (entry: TimelineAction) => TimelineAction) => {
    updateEntries((current) => current.map((entry, index) =>
      index === timelineIndex && entry.type === 'action' ? { ...entry, entry: update(entry.entry) } : entry));
  };
  const handleDeleteAction = (timelineIndex: number) => updateEntries((current) =>
    current[timelineIndex]?.type === 'action' ? current.filter((_, index) => index !== timelineIndex) : current);
  const handleEditSelector = (timelineIndex: number, selector: string) => updateAction(timelineIndex, (entry) => {
    const action = entry.action;
    const params = action?.params;
    if (!action || !params || !params.value || !('selector' in params.value)) return entry;
    return { ...entry, action: { ...action, params: { ...params, value: { ...params.value, selector } } } } as TimelineAction;
  });
  const handleEditPayload = (timelineIndex: number, payload: Record<string, unknown>) => updateAction(timelineIndex, (entry) => {
    const action = entry.action;
    const params = action?.params;
    if (!action || !params) return entry;
    const aliases: Record<string, string> = { text: 'value', targetUrl: 'url', scrollX: 'x', scrollY: 'y', selectedText: 'label' };
    const patch = Object.fromEntries(Object.entries(payload).map(([key, value]) => [aliases[key] ?? key, value]));
    return { ...entry, action: { ...action, params: { ...params, value: { ...(params.value as Record<string, unknown>), ...patch } } } } as TimelineAction;
  });
  const clearActions = () => updateEntries((current) => current.filter((entry) => entry.type !== 'action'));
  const insertAction = (data: { actionType: string; payload?: Record<string, unknown>; selector?: string }) => {
    const actionCase = data.actionType === 'select' ? 'selectOption' : data.actionType;
    const actionEnum = `ACTION_TYPE_${data.actionType.replace(/[A-Z]/g, (letter) => `_${letter}`).toUpperCase()}`;
    const value: Record<string, unknown> = { ...(data.payload ?? {}), ...(data.selector ? { selector: data.selector } : {}) };
    const extractType = ({ text: 'EXTRACT_TYPE_TEXT', innerHTML: 'EXTRACT_TYPE_INNER_HTML', attribute: 'EXTRACT_TYPE_ATTRIBUTE', value: 'EXTRACT_TYPE_VALUE' } as Record<string, string>)[String(value.extractType)] ?? 'EXTRACT_TYPE_TEXT';
    const assertionMode = `ASSERTION_MODE_${String(value.mode ?? 'exists').replace(/[A-Z]/g, (letter) => `_${letter}`).toUpperCase()}`;
    const params = data.actionType === 'navigate' ? { navigate: { url: String(value.targetUrl ?? value.url ?? '') } }
      : data.actionType === 'input' ? { input: { selector: String(value.selector ?? ''), value: String(value.text ?? value.value ?? '') } }
      : data.actionType === 'click' ? { click: { selector: String(value.selector ?? ''), button: typeof value.button === 'string' ? `MOUSE_BUTTON_${value.button.toUpperCase()}` : value.button, clickCount: value.clickCount, modifiers: Array.isArray(value.modifiers) ? value.modifiers.map((modifier) => `KEYBOARD_MODIFIER_${String(modifier).toUpperCase()}`) : [] } }
      : data.actionType === 'assert' ? { assert: { selector: String(value.selector ?? ''), mode: assertionMode, expected: value.expected, attributeName: value.attribute, caseSensitive: value.caseSensitive } }
      : data.actionType === 'scroll' ? { scroll: value }
      : data.actionType === 'select' ? { selectOption: { selector: String(value.selector ?? ''), selectBy: { case: 'value', value: String(value.value ?? '') } } }
      : data.actionType === 'wait' ? { wait: value.waitType === 'time'
        ? { durationMs: Number(value.duration ?? value.durationMs ?? 1000) }
        : { selector: String(value.selector ?? ''), state: value.state ? `WAIT_STATE_${String(value.state).toUpperCase()}` : undefined } }
      : data.actionType === 'screenshot' ? { screenshot: { fullPage: Boolean(value.fullPage), selector: value.selector } }
      : data.actionType === 'extract' ? { extract: { selector: String(value.selector ?? ''), extractType, attributeName: value.attribute, storeAs: value.variableName } }
      : data.actionType === 'setVariable' ? { setVariable: { name: String(value.name ?? ''), sourceType: 'SET_VARIABLE_SOURCE_TYPE_STATIC', value: value.value } }
      : data.actionType === 'evaluate' ? { evaluate: { expression: String(value.script ?? ''), storeResult: value.variableName } }
      : { [actionCase]: value };
    const sequenceNum = timelineEntries.filter((entry) => entry.type === 'action').length;
    const raw = {
      id: `manual-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
      sequenceNum,
      timestamp: { seconds: String(Math.floor(Date.now() / 1000)), nanos: (Date.now() % 1000) * 1_000_000 },
      action: { type: actionEnum, ...params, ...(data.actionType === 'screenshot' && value.name ? { metadata: { label: String(value.name) } } : {}) },
      context: { sessionId: sessionId ?? '' },
    };
    const generated = fromJson(ProtoTimelineEntrySchema, raw as unknown as JsonValue);
    updateEntries((current) => [...current, { type: 'action', pageId: pages.activePageId ?? '', entry: generated }]);
  };
  const lowConfidenceCount = 0;
  const mediumConfidenceCount = identifiedActions.filter((action) => action.selector).length;

  return {
    ...recordMode,
    clearActions, insertAction, lowConfidenceCount, mediumConfidenceCount,
    openPages: pages.openPages,
    activePageId: pages.activePageId,
    switchToPage: pages.switchToPage,
    closePage: pages.closePage,
    createPage: pages.createPage,
    isPagesLoading: pages.isLoading,
    ...selection,
    isSidebarOpen, setSidebarOpen, handleSidebarToggle, sidebarActiveTab, setSidebarTab, setAutoActivity,
    aiSettings, updateAISettings, aiMessages, aiSendMessage, aiAbortNavigation, aiResumeNavigation,
    aiClearConversation, aiIsNavigating, aiSteps, aiAvailableModels, aiHumanIntervention,
    recentActivityPageId, timelineEntries, timelineItems: mode === 'recording' ? recordedTimelineItems : timelineItems, isTimelineLive, identifiedActions,
    timelineDisplayItems, pageColorMap, timelineItemCount, selectedActionIds,
    handleDeleteAction, handleEditSelector, handleEditPayload,
  };
}
