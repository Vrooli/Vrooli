import { useMemo, useRef, useState } from 'react';
import { useRecordMode } from '../hooks/useRecordMode';
import { useActionSelection } from '../hooks/useActionSelection';
import { usePages } from '../hooks/usePages';
import { useTimeline } from '../hooks/useTimeline';
import { useUnifiedTimeline } from '../hooks/useUnifiedTimeline';
import { attachTimelinePageIdentities, mergeTimelineItemsWithAISteps, recordingEntryToTimelineItem, type TimelineMode } from '../types/timeline-unified';
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
  const { entries: timelineEntries } = useTimeline({ sessionId, pages: pages.openPages });
  const { items: timelineItems, isLive: isTimelineLive } = useUnifiedTimeline({
    mode,
    executionId,
    initialTimelineItems: mode === 'recording' ? timelineEntries.map(recordingEntryToTimelineItem) : undefined,
    workflowNodes: mode === 'execution' ? workflowNodes : undefined,
    workflowEdges: mode === 'execution' ? workflowEdges : undefined,
  });
  const identifiedActions = useMemo(
    () => attachTimelinePageIdentities(recordMode.actions, timelineEntries),
    [recordMode.actions, timelineEntries],
  );
  const timelineDisplayItems = useMemo(
    () => mode !== 'recording' || aiSteps.length === 0
      ? timelineItems
      : mergeTimelineItemsWithAISteps(timelineItems, aiSteps),
    [mode, timelineItems, aiSteps],
  );
  const pageColorMap = useMemo(() => {
    const colors = new Map<string, typeof PAGE_COLORS[number]>();
    pages.openPages.forEach((page, index) => colors.set(page.id, PAGE_COLORS[index % PAGE_COLORS.length] ?? PAGE_COLORS[0]));
    return colors;
  }, [pages.openPages]);
  const timelineItemCount = mode === 'recording' ? timelineDisplayItems.length : timelineItems.length;
  const selection = useActionSelection({ actionCount: timelineItemCount });
  const selectedActionIndices = useMemo(() => {
    if (mode !== 'recording') return selection.selectedIndicesArray;
    const selected: number[] = [];
    let actionIndex = 0;
    timelineDisplayItems.forEach((item, index) => {
      if (item?.entryType === 'page_event') return;
      if (selection.selectedIndices.has(index)) selected.push(actionIndex);
      actionIndex += 1;
    });
    return selected;
  }, [mode, selection.selectedIndicesArray, selection.selectedIndices, timelineDisplayItems]);
  const handleDeleteAction = (index: number) => recordMode.deleteAction(index);
  const handleEditSelector = (index: number, selector: string) => recordMode.updateSelector(index, selector);
  const handleEditPayload = (index: number, payload: Record<string, unknown>) => recordMode.updatePayload(index, payload);

  return {
    ...recordMode,
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
    recentActivityPageId, timelineEntries, timelineItems, isTimelineLive, identifiedActions,
    timelineDisplayItems, pageColorMap, timelineItemCount, selectedActionIndices,
    handleDeleteAction, handleEditSelector, handleEditPayload,
  };
}
