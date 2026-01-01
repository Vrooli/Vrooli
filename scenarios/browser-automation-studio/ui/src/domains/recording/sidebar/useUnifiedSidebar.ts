/**
 * useUnifiedSidebar Hook
 *
 * Manages the unified sidebar state including:
 * - Active tab (timeline/auto)
 * - Sidebar width and resize handling
 * - Activity indicators for each tab
 *
 * All state is persisted to localStorage for user preference retention.
 */

import { useState, useCallback, useEffect, useRef } from 'react';
import {
  type TabId,
  type ArtifactSubType,
  SIDEBAR_MIN_WIDTH,
  SIDEBAR_MAX_WIDTH,
  SIDEBAR_DEFAULT_WIDTH,
  STORAGE_KEYS,
  DEFAULT_ARTIFACT_SUBTYPE,
} from './types';
import type { TimelineMode } from '../types/timeline-unified';

// ============================================================================
// LocalStorage Helpers
// ============================================================================

function getStoredNumber(key: string, defaultValue: number): number {
  if (typeof window === 'undefined') return defaultValue;
  try {
    const stored = localStorage.getItem(key);
    if (stored) {
      const parsed = parseInt(stored, 10);
      if (!isNaN(parsed)) return parsed;
    }
  } catch {
    // Ignore storage errors
  }
  return defaultValue;
}

function getStoredBoolean(key: string, defaultValue: boolean): boolean {
  if (typeof window === 'undefined') return defaultValue;
  try {
    const stored = localStorage.getItem(key);
    if (stored !== null) return stored === 'true';
  } catch {
    // Ignore storage errors
  }
  return defaultValue;
}

function getStoredTab(key: string, defaultValue: TabId): TabId {
  if (typeof window === 'undefined') return defaultValue;
  try {
    const stored = localStorage.getItem(key);
    // Handle new tab values and migrate old values
    if (stored === 'timeline' || stored === 'auto' || stored === 'artifacts' || stored === 'history') {
      return stored;
    }
    // Migrate old 'screenshots' or 'logs' to 'artifacts'
    if (stored === 'screenshots' || stored === 'logs') {
      return 'artifacts';
    }
  } catch {
    // Ignore storage errors
  }
  return defaultValue;
}

function getStoredArtifactSubType(key: string, defaultValue: ArtifactSubType): ArtifactSubType {
  if (typeof window === 'undefined') return defaultValue;
  try {
    const stored = localStorage.getItem(key);
    if (
      stored === 'screenshots' ||
      stored === 'execution-logs' ||
      stored === 'console' ||
      stored === 'network' ||
      stored === 'dom-snapshots'
    ) {
      return stored;
    }
  } catch {
    // Ignore storage errors
  }
  return defaultValue;
}

function setStoredValue(key: string, value: string | number | boolean): void {
  if (typeof window === 'undefined') return;
  try {
    localStorage.setItem(key, String(value));
  } catch {
    // Ignore storage errors
  }
}

// ============================================================================
// Hook Options
// ============================================================================

export interface UseUnifiedSidebarOptions {
  /** Initial tab to show (overrides localStorage) */
  initialTab?: TabId;
  /** Initial width (overrides localStorage) */
  initialWidth?: number;
  /** Whether sidebar is initially open */
  initialOpen?: boolean;
  /** Callback when tab changes */
  onTabChange?: (tab: TabId) => void;
  /** Current mode (recording or execution) - used for tab visibility */
  mode?: TimelineMode;
}

// ============================================================================
// Hook Return Type
// ============================================================================

export interface UseUnifiedSidebarReturn {
  // Visibility
  isOpen: boolean;
  setIsOpen: (open: boolean) => void;
  toggleOpen: () => void;

  // Tab management
  activeTab: TabId;
  setActiveTab: (tab: TabId) => void;

  // Artifact sub-type management
  artifactSubType: ArtifactSubType;
  setArtifactSubType: (subType: ArtifactSubType) => void;

  // Dimensions
  width: number;
  minWidth: number;
  maxWidth: number;

  // Resize handling
  isResizing: boolean;
  handleResizeStart: (e: React.MouseEvent) => void;

  // Activity indicators (flash when tab has new content)
  timelineActivity: boolean;
  autoActivity: boolean;
  artifactsActivity: boolean;
  historyActivity: boolean;
  setTimelineActivity: (active: boolean) => void;
  setAutoActivity: (active: boolean) => void;
  setArtifactsActivity: (active: boolean) => void;
  setHistoryActivity: (active: boolean) => void;
}

// ============================================================================
// Hook Implementation
// ============================================================================

export function useUnifiedSidebar(
  options: UseUnifiedSidebarOptions = {}
): UseUnifiedSidebarReturn {
  const { initialTab, initialWidth, initialOpen, onTabChange } = options;

  // Core state - check if initialOpen was explicitly provided
  const [isOpen, setIsOpenState] = useState(() =>
    initialOpen !== undefined ? initialOpen : getStoredBoolean(STORAGE_KEYS.SIDEBAR_OPEN, true)
  );

  const [activeTab, setActiveTabState] = useState<TabId>(() =>
    initialTab ?? getStoredTab(STORAGE_KEYS.SIDEBAR_TAB, 'timeline')
  );

  const [width, setWidth] = useState(() =>
    initialWidth ?? getStoredNumber(STORAGE_KEYS.SIDEBAR_WIDTH, SIDEBAR_DEFAULT_WIDTH)
  );

  // Resize state
  const [isResizing, setIsResizing] = useState(false);
  const resizeStartXRef = useRef(0);
  const resizeStartWidthRef = useRef(0);

  // Artifact sub-type state
  const [artifactSubType, setArtifactSubTypeState] = useState<ArtifactSubType>(() =>
    getStoredArtifactSubType(STORAGE_KEYS.ARTIFACT_SUBTYPE, DEFAULT_ARTIFACT_SUBTYPE)
  );

  // Activity indicators
  const [timelineActivity, setTimelineActivity] = useState(false);
  const [autoActivity, setAutoActivity] = useState(false);
  const [artifactsActivity, setArtifactsActivity] = useState(false);
  const [historyActivity, setHistoryActivity] = useState(false);
  const timelineActivityTimerRef = useRef<ReturnType<typeof setTimeout>>();
  const autoActivityTimerRef = useRef<ReturnType<typeof setTimeout>>();
  const artifactsActivityTimerRef = useRef<ReturnType<typeof setTimeout>>();
  const historyActivityTimerRef = useRef<ReturnType<typeof setTimeout>>();

  // Persist open state
  const setIsOpen = useCallback((open: boolean) => {
    setIsOpenState(open);
    setStoredValue(STORAGE_KEYS.SIDEBAR_OPEN, open);
  }, []);

  const toggleOpen = useCallback(() => {
    setIsOpenState((prev) => {
      const next = !prev;
      setStoredValue(STORAGE_KEYS.SIDEBAR_OPEN, next);
      return next;
    });
  }, []);

  // Persist tab state
  const setActiveTab = useCallback(
    (tab: TabId) => {
      setActiveTabState(tab);
      setStoredValue(STORAGE_KEYS.SIDEBAR_TAB, tab);
      onTabChange?.(tab);
    },
    [onTabChange]
  );

  // Persist artifact sub-type
  const setArtifactSubType = useCallback((subType: ArtifactSubType) => {
    setArtifactSubTypeState(subType);
    setStoredValue(STORAGE_KEYS.ARTIFACT_SUBTYPE, subType);
  }, []);

  // Activity indicator with auto-clear
  const setTimelineActivityWithClear = useCallback((active: boolean) => {
    if (timelineActivityTimerRef.current) {
      clearTimeout(timelineActivityTimerRef.current);
    }
    setTimelineActivity(active);
    if (active) {
      timelineActivityTimerRef.current = setTimeout(() => {
        setTimelineActivity(false);
      }, 2000);
    }
  }, []);

  const setAutoActivityWithClear = useCallback((active: boolean) => {
    if (autoActivityTimerRef.current) {
      clearTimeout(autoActivityTimerRef.current);
    }
    setAutoActivity(active);
    if (active) {
      autoActivityTimerRef.current = setTimeout(() => {
        setAutoActivity(false);
      }, 2000);
    }
  }, []);

  const setArtifactsActivityWithClear = useCallback((active: boolean) => {
    if (artifactsActivityTimerRef.current) {
      clearTimeout(artifactsActivityTimerRef.current);
    }
    setArtifactsActivity(active);
    if (active) {
      artifactsActivityTimerRef.current = setTimeout(() => {
        setArtifactsActivity(false);
      }, 2000);
    }
  }, []);

  const setHistoryActivityWithClear = useCallback((active: boolean) => {
    if (historyActivityTimerRef.current) {
      clearTimeout(historyActivityTimerRef.current);
    }
    setHistoryActivity(active);
    if (active) {
      historyActivityTimerRef.current = setTimeout(() => {
        setHistoryActivity(false);
      }, 2000);
    }
  }, []);

  // Cleanup activity timers
  useEffect(() => {
    return () => {
      if (timelineActivityTimerRef.current) clearTimeout(timelineActivityTimerRef.current);
      if (autoActivityTimerRef.current) clearTimeout(autoActivityTimerRef.current);
      if (artifactsActivityTimerRef.current) clearTimeout(artifactsActivityTimerRef.current);
      if (historyActivityTimerRef.current) clearTimeout(historyActivityTimerRef.current);
    };
  }, []);

  // Resize handling
  const handleResizeStart = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setIsResizing(true);
    resizeStartXRef.current = e.clientX;
    resizeStartWidthRef.current = width;
  }, [width]);

  // Mouse move handler for resize
  useEffect(() => {
    if (!isResizing) return;

    const handleMouseMove = (e: MouseEvent) => {
      const delta = e.clientX - resizeStartXRef.current;
      const newWidth = Math.min(
        SIDEBAR_MAX_WIDTH,
        Math.max(SIDEBAR_MIN_WIDTH, resizeStartWidthRef.current + delta)
      );
      setWidth(newWidth);
    };

    const handleMouseUp = () => {
      setIsResizing(false);
      // Persist final width
      setStoredValue(STORAGE_KEYS.SIDEBAR_WIDTH, width);
    };

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);

    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
    };
  }, [isResizing, width]);

  // Persist width when resize ends
  useEffect(() => {
    if (!isResizing) {
      setStoredValue(STORAGE_KEYS.SIDEBAR_WIDTH, width);
    }
  }, [isResizing, width]);

  return {
    // Visibility
    isOpen,
    setIsOpen,
    toggleOpen,

    // Tab management
    activeTab,
    setActiveTab,

    // Artifact sub-type management
    artifactSubType,
    setArtifactSubType,

    // Dimensions
    width,
    minWidth: SIDEBAR_MIN_WIDTH,
    maxWidth: SIDEBAR_MAX_WIDTH,

    // Resize handling
    isResizing,
    handleResizeStart,

    // Activity indicators
    timelineActivity,
    autoActivity,
    artifactsActivity,
    historyActivity,
    setTimelineActivity: setTimelineActivityWithClear,
    setAutoActivity: setAutoActivityWithClear,
    setArtifactsActivity: setArtifactsActivityWithClear,
    setHistoryActivity: setHistoryActivityWithClear,
  };
}
