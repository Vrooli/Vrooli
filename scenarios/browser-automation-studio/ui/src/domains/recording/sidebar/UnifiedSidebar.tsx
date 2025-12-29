/**
 * UnifiedSidebar Component
 *
 * The unified sidebar container with tabbed navigation for Timeline and Auto tabs.
 * Manages tab switching, resize handle, and activity indicators.
 *
 * Features:
 * - Timeline tab: Shows recorded actions with selection controls
 * - Auto tab: AI navigation chat interface
 * - Activity indicators for each tab
 * - Resizable width with persistence
 */

import { useEffect, useCallback, useMemo } from 'react';
import { TimelineTab, type TimelineTabProps } from './TimelineTab';
import { AutoTab, type AutoTabProps } from './AutoTab';
import { useUnifiedSidebar, type UseUnifiedSidebarOptions } from './useUnifiedSidebar';
import type { TabId } from './types';

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

export interface UnifiedSidebarProps {
  /** Props to pass through to TimelineTab */
  timelineProps: TimelineTabPassthroughProps;
  /** Props to pass through to AutoTab */
  autoProps: AutoTabPassthroughProps;
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
  timelineProps,
  autoProps,
  initialTab,
  activeTab: controlledActiveTab,
  isOpen: controlledIsOpen,
  onOpenChange,
  onTabChange,
  className = '',
}: UnifiedSidebarProps) {
  const sidebarOptions: UseUnifiedSidebarOptions = {
    initialTab: controlledActiveTab ?? initialTab,
    initialOpen: controlledIsOpen,
    onTabChange,
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
    setTimelineActivity,
    setAutoActivity,
  } = useUnifiedSidebar(sidebarOptions);

  // Use controlled activeTab if provided, otherwise use internal state
  const activeTab = controlledActiveTab ?? internalActiveTab;

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

  // Clear activity when switching to that tab
  useEffect(() => {
    if (activeTab === 'timeline') {
      setTimelineActivity(false);
    } else if (activeTab === 'auto') {
      setAutoActivity(false);
    }
  }, [activeTab, setTimelineActivity, setAutoActivity]);

  const handleTabClick = useCallback(
    (tab: TabId) => {
      setActiveTab(tab);
    },
    [setActiveTab]
  );

  // Keyboard shortcuts: Cmd+1/2 (Mac) or Ctrl+1/2 (Windows/Linux)
  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      // Check for modifier key (Cmd on Mac, Ctrl on Windows/Linux)
      const modifierKey = isMac() ? e.metaKey : e.ctrlKey;
      if (!modifierKey) return;

      // Switch tabs with 1/2
      if (e.key === '1') {
        e.preventDefault();
        setActiveTab('timeline');
      } else if (e.key === '2') {
        e.preventDefault();
        setActiveTab('auto');
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, setActiveTab]);

  // Get keyboard shortcut modifier for tooltips
  const modifierKey = useMemo(() => (isMac() ? '⌘' : 'Ctrl+'), []);

  if (!isOpen) {
    return null;
  }

  return (
    <div
      className={`relative flex flex-col bg-white dark:bg-gray-900 border-r border-gray-200 dark:border-gray-700 ${className}`}
      style={{ width: `${width}px`, minWidth: `${width}px` }}
    >
      {/* Tab Header */}
      <div className="flex items-center border-b border-gray-200 dark:border-gray-700" role="tablist">
        <TabButton
          label="Timeline"
          tabId="timeline"
          isActive={activeTab === 'timeline'}
          hasActivity={timelineActivity}
          onClick={handleTabClick}
          shortcut={`${modifierKey}1`}
          tooltip="View recorded actions"
        />
        <TabButton
          label="Auto"
          tabId="auto"
          isActive={activeTab === 'auto'}
          hasActivity={autoActivity}
          onClick={handleTabClick}
          shortcut={`${modifierKey}2`}
          tooltip="AI-powered browser automation"
        />
      </div>

      {/* Tab Content */}
      <div className="flex-1 overflow-hidden">
        {activeTab === 'timeline' && <TimelineTab {...timelineProps} className="h-full" />}
        {activeTab === 'auto' && <AutoTab {...autoProps} className="h-full" />}
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
