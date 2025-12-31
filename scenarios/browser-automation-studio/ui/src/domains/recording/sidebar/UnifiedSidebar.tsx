/**
 * UnifiedSidebar Component
 *
 * The unified sidebar container with tabbed navigation for Timeline, Auto,
 * Screenshots, and Logs tabs. Mode-aware: different tabs are shown based
 * on whether we're in recording or execution mode.
 *
 * Features:
 * - Timeline tab: Shows recorded/executed actions (both modes)
 * - Auto tab: AI navigation chat interface (recording mode only)
 * - Screenshots tab: Execution screenshots (execution mode only)
 * - Logs tab: Execution logs (execution mode only)
 * - Activity indicators for each tab
 * - Resizable width with persistence
 */

import { useEffect, useCallback, useMemo } from 'react';
import { TimelineTab, type TimelineTabProps } from './TimelineTab';
import { AutoTab, type AutoTabProps } from './AutoTab';
import { ScreenshotsTab } from './ScreenshotsTab';
import { LogsTab } from './LogsTab';
import { useUnifiedSidebar, type UseUnifiedSidebarOptions } from './useUnifiedSidebar';
import { getVisibleTabs, isTabVisible, getDefaultTab, type TabId } from './types';
import type { TimelineMode } from '../types/timeline-unified';
import type { Screenshot, LogEntry, Execution } from '@/domains/executions';

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
 * Props for ScreenshotsTab passed through UnifiedSidebar.
 */
export interface ScreenshotsTabPassthroughProps {
  screenshots: Screenshot[];
  selectedIndex?: number;
  onSelectScreenshot?: (index: number) => void;
  executionStatus?: Execution['status'];
}

/**
 * Props for LogsTab passed through UnifiedSidebar.
 */
export interface LogsTabPassthroughProps {
  logs: LogEntry[];
  filter?: 'all' | 'error' | 'warning' | 'info' | 'success';
  onFilterChange?: (filter: 'all' | 'error' | 'warning' | 'info' | 'success') => void;
  executionStatus?: Execution['status'];
}

export interface UnifiedSidebarProps {
  /** Current mode (determines which tabs are visible) */
  mode: TimelineMode;
  /** Props to pass through to TimelineTab */
  timelineProps: TimelineTabPassthroughProps;
  /** Props to pass through to AutoTab */
  autoProps: AutoTabPassthroughProps;
  /** Props to pass through to ScreenshotsTab (execution mode only) */
  screenshotsProps?: ScreenshotsTabPassthroughProps;
  /** Props to pass through to LogsTab (execution mode only) */
  logsProps?: LogsTabPassthroughProps;
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
  screenshotsProps,
  logsProps,
  initialTab,
  activeTab: controlledActiveTab,
  isOpen: controlledIsOpen,
  onOpenChange,
  onTabChange,
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
    width,
    isResizing,
    handleResizeStart,
    timelineActivity,
    autoActivity,
    screenshotsActivity,
    logsActivity,
    setTimelineActivity,
    setAutoActivity,
    setScreenshotsActivity,
    setLogsActivity,
  } = useUnifiedSidebar(sidebarOptions);

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

  // Show screenshots activity when screenshots change (only if not viewing screenshots)
  useEffect(() => {
    if (activeTab !== 'screenshots' && screenshotsProps?.screenshots && screenshotsProps.screenshots.length > 0) {
      setScreenshotsActivity(true);
    }
  }, [screenshotsProps?.screenshots?.length, activeTab, setScreenshotsActivity]);

  // Show logs activity when logs change (only if not viewing logs)
  useEffect(() => {
    if (activeTab !== 'logs' && logsProps?.logs && logsProps.logs.length > 0) {
      setLogsActivity(true);
    }
  }, [logsProps?.logs?.length, activeTab, setLogsActivity]);

  // Clear activity when switching to that tab
  useEffect(() => {
    if (activeTab === 'timeline') {
      setTimelineActivity(false);
    } else if (activeTab === 'auto') {
      setAutoActivity(false);
    } else if (activeTab === 'screenshots') {
      setScreenshotsActivity(false);
    } else if (activeTab === 'logs') {
      setLogsActivity(false);
    }
  }, [activeTab, setTimelineActivity, setAutoActivity, setScreenshotsActivity, setLogsActivity]);

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
      case 'screenshots': return screenshotsActivity;
      case 'logs': return logsActivity;
      default: return false;
    }
  }, [timelineActivity, autoActivity, screenshotsActivity, logsActivity]);

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
        {activeTab === 'screenshots' && screenshotsProps && (
          <ScreenshotsTab {...screenshotsProps} className="h-full" />
        )}
        {activeTab === 'logs' && logsProps && (
          <LogsTab {...logsProps} className="h-full" />
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
