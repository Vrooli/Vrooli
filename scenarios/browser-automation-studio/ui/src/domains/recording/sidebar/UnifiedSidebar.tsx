/**
 * UnifiedSidebar Component
 *
 * The unified sidebar container with tabbed navigation for Timeline, Auto,
 * Artifacts, and History tabs. Mode-aware: different tabs are shown based on
 * whether we're in recording or execution mode.
 *
 * Features:
 * - Timeline tab: Shows recorded/executed actions (both modes)
 * - Auto tab: AI navigation chat interface (recording mode only)
 * - Artifacts tab: Execution artifacts - screenshots, logs, network, etc. (execution mode only)
 * - History tab: Execution history for switching runs (execution mode only)
 * - Activity indicators for each tab
 * - Resizable width with persistence
 */

import { useEffect, useCallback, useMemo } from 'react';
import { TimelineTab, type TimelineTabProps } from './TimelineTab';
import { AutoTab, type AutoTabProps } from './AutoTab';
import { ArtifactsTab, type ArtifactsTabProps } from './ArtifactsTab';
import { HistoryTab, type HistoryTabProps } from './HistoryTab';
import { useUnifiedSidebar, type UseUnifiedSidebarOptions } from './useUnifiedSidebar';
import { getVisibleTabs, isTabVisible, getDefaultTab, type TabId, type ArtifactSubType } from './types';
import type { TimelineMode } from '../types/timeline-unified';

// ============================================================================
// Helpers
// ============================================================================

/** Detect if running on Mac for keyboard shortcut display */
function isMac(): boolean {
  if (typeof navigator === 'undefined') return false;
  return navigator.platform.toUpperCase().indexOf('MAC') >= 0;
}

// ============================================================================
// Types
// ============================================================================

/**
 * Props for TimelineTab passed through UnifiedSidebar.
 * Omits className as it's applied internally.
 */
export type TimelineTabPassthroughProps = Omit<TimelineTabProps, 'className'>;

/**
 * Props for AutoTab passed through UnifiedSidebar.
 * Omits className as it's applied internally.
 */
export type AutoTabPassthroughProps = Omit<AutoTabProps, 'className'>;

/**
 * Props for ArtifactsTab passed through UnifiedSidebar.
 * Omits className as it's applied internally.
 */
export type ArtifactsTabPassthroughProps = Omit<ArtifactsTabProps, 'className'>;

/**
 * Props for HistoryTab passed through UnifiedSidebar.
 * Omits className as it's applied internally.
 */
export type HistoryTabPassthroughProps = Omit<HistoryTabProps, 'className'>;

export interface UnifiedSidebarProps {
  /** Current mode (determines which tabs are visible) */
  mode: TimelineMode;
  /** Props to pass through to TimelineTab */
  timelineProps: TimelineTabPassthroughProps;
  /** Props to pass through to AutoTab */
  autoProps: AutoTabPassthroughProps;
  /** Props to pass through to ArtifactsTab (execution mode only) */
  artifactsProps?: ArtifactsTabPassthroughProps;
  /** Props to pass through to HistoryTab (execution mode only) */
  historyProps?: HistoryTabPassthroughProps;
  /** Initial tab to show (used when activeTab is not controlled) */
  initialTab?: TabId;
  /** Controlled active tab (use with onTabChange for controlled behavior) */
  activeTab?: TabId;
  /** Whether the sidebar is open */
  isOpen?: boolean;
  /** Callback when sidebar open state changes */
  onOpenChange?: (open: boolean) => void;
  /** Callback when tab changes */
  onTabChange?: (tab: TabId) => void;
  /** Controlled artifact sub-type (for ArtifactsTab) */
  artifactSubType?: ArtifactSubType;
  /** Callback when artifact sub-type changes */
  onArtifactSubTypeChange?: (subType: ArtifactSubType) => void;
  /** Optional className for the container */
  className?: string;
}

// ============================================================================
// Component
// ============================================================================

export function UnifiedSidebar({
  mode,
  timelineProps,
  autoProps,
  artifactsProps,
  historyProps,
  initialTab,
  activeTab: controlledActiveTab,
  isOpen: controlledIsOpen,
  onOpenChange,
  onTabChange,
  artifactSubType: controlledArtifactSubType,
  onArtifactSubTypeChange,
  className = '',
}: UnifiedSidebarProps) {
  // Get visible tabs for current mode
  const visibleTabs = useMemo(() => getVisibleTabs(mode), [mode]);

  const sidebarOptions: UseUnifiedSidebarOptions = {
    initialTab: controlledActiveTab ?? initialTab,
    initialOpen: controlledIsOpen,
    onTabChange,
    mode,
  };

  const {
    isOpen,
    setIsOpen,
    activeTab: internalActiveTab,
    setActiveTab,
    artifactSubType: internalArtifactSubType,
    setArtifactSubType,
    width,
    isResizing,
    handleResizeStart,
    timelineActivity,
    autoActivity,
    artifactsActivity,
    historyActivity,
    setTimelineActivity,
    setAutoActivity,
    setArtifactsActivity,
    setHistoryActivity,
  } = useUnifiedSidebar(sidebarOptions);

  // Use controlled artifactSubType if provided, otherwise use internal state
  const artifactSubType = controlledArtifactSubType ?? internalArtifactSubType;

  // Use controlled activeTab if provided, otherwise use internal state
  const activeTab = controlledActiveTab ?? internalActiveTab;

  // Auto-switch to valid tab when mode changes and current tab becomes invisible
  useEffect(() => {
    if (!isTabVisible(activeTab, mode)) {
      const defaultTab = getDefaultTab(mode);
      setActiveTab(defaultTab);
    }
  }, [mode, activeTab, setActiveTab]);

  // Sync controlled active tab
  useEffect(() => {
    if (controlledActiveTab !== undefined && controlledActiveTab !== internalActiveTab) {
      setActiveTab(controlledActiveTab);
    }
  }, [controlledActiveTab, internalActiveTab, setActiveTab]);

  // Sync controlled open state
  useEffect(() => {
    if (controlledIsOpen !== undefined && controlledIsOpen !== isOpen) {
      setIsOpen(controlledIsOpen);
    }
  }, [controlledIsOpen, isOpen, setIsOpen]);

  // Notify parent of open state changes
  useEffect(() => {
    if (onOpenChange && controlledIsOpen !== isOpen) {
      onOpenChange(isOpen);
    }
  }, [isOpen, onOpenChange, controlledIsOpen]);

  // Show timeline activity when actions change (only if not viewing timeline)
  useEffect(() => {
    if (activeTab !== 'timeline' && timelineProps.actions.length > 0) {
      setTimelineActivity(true);
    }
  }, [timelineProps.actions.length, activeTab, setTimelineActivity]);

  // Show auto activity when AI messages change (only if not viewing auto)
  useEffect(() => {
    if (activeTab !== 'auto' && autoProps.messages.length > 0) {
      setAutoActivity(true);
    }
  }, [autoProps.messages.length, activeTab, setAutoActivity]);

  // Show artifacts activity when artifacts change (only if not viewing artifacts)
  useEffect(() => {
    if (activeTab !== 'artifacts' && artifactsProps?.screenshots && artifactsProps.screenshots.length > 0) {
      setArtifactsActivity(true);
    }
  }, [artifactsProps?.screenshots?.length, activeTab, setArtifactsActivity]);

  // Sync controlled artifactSubType
  useEffect(() => {
    if (controlledArtifactSubType !== undefined && controlledArtifactSubType !== internalArtifactSubType) {
      setArtifactSubType(controlledArtifactSubType);
    }
  }, [controlledArtifactSubType, internalArtifactSubType, setArtifactSubType]);

  // Notify parent of artifact sub-type changes
  useEffect(() => {
    if (onArtifactSubTypeChange && controlledArtifactSubType !== artifactSubType) {
      onArtifactSubTypeChange(artifactSubType);
    }
  }, [artifactSubType, onArtifactSubTypeChange, controlledArtifactSubType]);

  // Clear activity when switching to that tab
  useEffect(() => {
    if (activeTab === 'timeline') {
      setTimelineActivity(false);
    } else if (activeTab === 'auto') {
      setAutoActivity(false);
    } else if (activeTab === 'artifacts') {
      setArtifactsActivity(false);
    } else if (activeTab === 'history') {
      setHistoryActivity(false);
    }
  }, [activeTab, setTimelineActivity, setAutoActivity, setArtifactsActivity, setHistoryActivity]);

  const handleTabClick = useCallback(
    (tab: TabId) => {
      setActiveTab(tab);
    },
    [setActiveTab]
  );

  // Keyboard shortcuts: Cmd+1/2/3 (Mac) or Ctrl+1/2/3 (Windows/Linux)
  // Only for visible tabs
  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      // Check for modifier key (Cmd on Mac, Ctrl on Windows/Linux)
      const modifierKey = isMac() ? e.metaKey : e.ctrlKey;
      if (!modifierKey) return;

      // Switch tabs with number keys based on visible tabs
      const keyNum = parseInt(e.key, 10);
      if (keyNum >= 1 && keyNum <= visibleTabs.length) {
        e.preventDefault();
        const targetTab = visibleTabs[keyNum - 1];
        if (targetTab) {
          setActiveTab(targetTab.id);
        }
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, setActiveTab, visibleTabs]);

  // Get keyboard shortcut modifier for tooltips
  const modifierKey = useMemo(() => (isMac() ? '⌘' : 'Ctrl+'), []);

  // Get activity state for each tab
  const getActivityForTab = useCallback((tabId: TabId): boolean => {
    switch (tabId) {
      case 'timeline': return timelineActivity;
      case 'auto': return autoActivity;
      case 'artifacts': return artifactsActivity;
      case 'history': return historyActivity;
      default: return false;
    }
  }, [timelineActivity, autoActivity, artifactsActivity, historyActivity]);

  if (!isOpen) {
    return null;
  }

  return (
    <div
      className={`relative flex flex-col bg-white dark:bg-gray-900 border-r border-gray-200 dark:border-gray-700 ${className}`}
      style={{ width: `${width}px`, minWidth: `${width}px` }}
    >
      {/* Tab Header - dynamically render visible tabs */}
      <div className="flex items-center border-b border-gray-200 dark:border-gray-700" role="tablist">
        {visibleTabs.map((tab) => (
          <TabButton
            key={tab.id}
            label={tab.label}
            tabId={tab.id}
            isActive={activeTab === tab.id}
            hasActivity={getActivityForTab(tab.id)}
            onClick={handleTabClick}
            shortcut={`${modifierKey}${tab.shortcutKey}`}
            tooltip={tab.tooltip}
          />
        ))}
      </div>

      {/* Tab Content */}
      <div className="flex-1 overflow-hidden">
        {activeTab === 'timeline' && <TimelineTab {...timelineProps} className="h-full" />}
        {activeTab === 'auto' && <AutoTab {...autoProps} className="h-full" />}
        {activeTab === 'artifacts' && artifactsProps && (
          <ArtifactsTab
            {...artifactsProps}
            activeSubType={artifactSubType}
            onSubTypeChange={(subType) => {
              setArtifactSubType(subType);
              onArtifactSubTypeChange?.(subType);
            }}
            className="h-full"
          />
        )}
        {activeTab === 'history' && historyProps && (
          <HistoryTab {...historyProps} className="h-full" />
        )}
      </div>

      {/* Resize Handle */}
      <div
        className={`absolute top-0 right-0 w-1 h-full cursor-col-resize hover:bg-blue-500/50 active:bg-blue-500 transition-colors ${
          isResizing ? 'bg-blue-500' : ''
        }`}
        onMouseDown={handleResizeStart}
        role="separator"
        aria-orientation="vertical"
        aria-label="Resize sidebar"
        aria-valuenow={width}
      />
    </div>
  );
}

// ============================================================================
// Tab Button Sub-component
// ============================================================================

interface TabButtonProps {
  label: string;
  tabId: TabId;
  isActive: boolean;
  hasActivity: boolean;
  onClick: (tab: TabId) => void;
  shortcut: string;
  tooltip: string;
}

function TabButton({
  label,
  tabId,
  isActive,
  hasActivity,
  onClick,
  shortcut,
  tooltip,
}: TabButtonProps) {
  return (
    <button
      onClick={() => onClick(tabId)}
      className={`relative flex-1 px-4 py-2.5 text-sm font-medium transition-colors ${
        isActive
          ? 'text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/20 border-b-2 border-blue-500'
          : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-800'
      }`}
      role="tab"
      aria-selected={isActive}
      aria-controls={`${tabId}-panel`}
      title={`${tooltip} (${shortcut})`}
    >
      {label}
      {/* Activity indicator dot */}
      {hasActivity && !isActive && (
        <span
          className="absolute top-2 right-2 w-2 h-2 bg-blue-500 rounded-full animate-pulse"
          aria-label="New activity"
        />
      )}
    </button>
  );
}
