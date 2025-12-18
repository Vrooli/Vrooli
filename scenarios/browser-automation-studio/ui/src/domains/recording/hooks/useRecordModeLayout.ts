/**
 * useRecordModeLayout Hook
 *
 * Manages the layout state for the redesigned Record Mode page.
 * All layout preferences are persisted to localStorage.
 *
 * Layout state includes:
 * - Main content view ('live' or 'timeline')
 * - Sidebar content ('timeline' | 'preview' | null)
 * - Sidebar width
 * - Action bar auto-hide preference
 * - Mini preview position (when timeline is in main view)
 */

import { useCallback, useEffect, useState } from 'react';

const STORAGE_KEY = 'record-mode-layout';

/** What's shown in the main content area */
export type MainContentView = 'live' | 'timeline';

/** What's shown in the sidebar (or null if closed) */
export type SidebarView = 'timeline' | 'preview' | null;

interface LayoutState {
  /** Current main content view */
  mainView: MainContentView;
  /** Current sidebar content (null = closed) */
  sidebarView: SidebarView;
  /** Sidebar width in pixels */
  sidebarWidth: number;
  /** Whether the floating action bar should auto-hide */
  actionBarAutoHide: boolean;
}

const DEFAULT_STATE: LayoutState = {
  mainView: 'live',
  sidebarView: null,
  sidebarWidth: 360,
  actionBarAutoHide: false,
};

const MIN_SIDEBAR_WIDTH = 280;
const MAX_SIDEBAR_WIDTH = 600;

/** Load layout state from localStorage */
function loadLayoutState(): LayoutState {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      const parsed = JSON.parse(stored) as Partial<LayoutState>;
      return {
        mainView: parsed.mainView === 'timeline' ? 'timeline' : 'live',
        sidebarView: parsed.sidebarView === 'timeline' || parsed.sidebarView === 'preview' ? parsed.sidebarView : null,
        sidebarWidth: typeof parsed.sidebarWidth === 'number'
          ? Math.min(MAX_SIDEBAR_WIDTH, Math.max(MIN_SIDEBAR_WIDTH, parsed.sidebarWidth))
          : DEFAULT_STATE.sidebarWidth,
        actionBarAutoHide: typeof parsed.actionBarAutoHide === 'boolean' ? parsed.actionBarAutoHide : DEFAULT_STATE.actionBarAutoHide,
      };
    }
  } catch {
    // localStorage unavailable or invalid JSON
  }
  return DEFAULT_STATE;
}

/** Save layout state to localStorage */
function saveLayoutState(state: LayoutState): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  } catch {
    // localStorage unavailable
  }
}

export interface RecordModeLayoutState {
  /** Current main content view */
  mainView: MainContentView;
  /** Current sidebar content */
  sidebarView: SidebarView;
  /** Sidebar width in pixels */
  sidebarWidth: number;
  /** Whether action bar auto-hides */
  actionBarAutoHide: boolean;
  /** Whether sidebar is open */
  isSidebarOpen: boolean;

  /** Set the main view */
  setMainView: (view: MainContentView) => void;
  /** Set sidebar view (null to close) */
  setSidebarView: (view: SidebarView) => void;
  /** Toggle sidebar open/closed */
  toggleSidebar: () => void;
  /** Set sidebar width */
  setSidebarWidth: (width: number) => void;
  /** Handle sidebar resize start (returns cleanup) */
  handleSidebarResizeStart: (event: React.MouseEvent) => void;
  /** Set action bar auto-hide */
  setActionBarAutoHide: (autoHide: boolean) => void;

  /** Swap main and sidebar views */
  swapViews: () => void;
  /** Open timeline in sidebar */
  openTimelineSidebar: () => void;
  /** Open timeline in full view (for workflow creation) */
  openTimelineFullView: () => void;
  /** Return to live preview as main view */
  openLiveFullView: () => void;
}

export function useRecordModeLayout(): RecordModeLayoutState {
  const [state, setState] = useState<LayoutState>(loadLayoutState);
  const [isResizing, setIsResizing] = useState(false);

  // Persist state changes
  useEffect(() => {
    saveLayoutState(state);
  }, [state]);

  // Handle resize drag
  useEffect(() => {
    if (!isResizing) return;

    let startX = 0;
    let startWidth = state.sidebarWidth;

    const handleMouseMove = (event: MouseEvent) => {
      // Sidebar is on the left, so positive delta increases width
      const delta = event.clientX - startX;
      const newWidth = Math.min(MAX_SIDEBAR_WIDTH, Math.max(MIN_SIDEBAR_WIDTH, startWidth + delta));
      setState((prev) => ({ ...prev, sidebarWidth: newWidth }));
    };

    const handleMouseUp = () => {
      setIsResizing(false);
      document.body.style.userSelect = '';
      document.body.style.cursor = '';
    };

    // Capture start position on first move
    const handleFirstMove = (event: MouseEvent) => {
      startX = event.clientX;
      startWidth = state.sidebarWidth;
      window.removeEventListener('mousemove', handleFirstMove);
      window.addEventListener('mousemove', handleMouseMove);
    };

    window.addEventListener('mousemove', handleFirstMove);
    window.addEventListener('mouseup', handleMouseUp);
    document.body.style.userSelect = 'none';
    document.body.style.cursor = 'col-resize';

    return () => {
      window.removeEventListener('mousemove', handleFirstMove);
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseup', handleMouseUp);
      document.body.style.userSelect = '';
      document.body.style.cursor = '';
    };
  }, [isResizing, state.sidebarWidth]);

  const setMainView = useCallback((view: MainContentView) => {
    setState((prev) => ({ ...prev, mainView: view }));
  }, []);

  const setSidebarView = useCallback((view: SidebarView) => {
    setState((prev) => ({ ...prev, sidebarView: view }));
  }, []);

  const toggleSidebar = useCallback(() => {
    setState((prev) => {
      if (prev.sidebarView === null) {
        // Open sidebar - default to timeline if main is live, preview if main is timeline
        const newSidebarView = prev.mainView === 'live' ? 'timeline' : 'preview';
        return { ...prev, sidebarView: newSidebarView };
      }
      return { ...prev, sidebarView: null };
    });
  }, []);

  const setSidebarWidth = useCallback((width: number) => {
    const clampedWidth = Math.min(MAX_SIDEBAR_WIDTH, Math.max(MIN_SIDEBAR_WIDTH, width));
    setState((prev) => ({ ...prev, sidebarWidth: clampedWidth }));
  }, []);

  const handleSidebarResizeStart = useCallback((event: React.MouseEvent) => {
    event.preventDefault();
    setIsResizing(true);
  }, []);

  const setActionBarAutoHide = useCallback((autoHide: boolean) => {
    setState((prev) => ({ ...prev, actionBarAutoHide: autoHide }));
  }, []);

  const swapViews = useCallback(() => {
    setState((prev) => {
      if (prev.mainView === 'live' && prev.sidebarView === 'timeline') {
        return { ...prev, mainView: 'timeline', sidebarView: 'preview' };
      }
      if (prev.mainView === 'timeline' && prev.sidebarView === 'preview') {
        return { ...prev, mainView: 'live', sidebarView: 'timeline' };
      }
      // If sidebar is closed, just swap main view
      if (prev.sidebarView === null) {
        return {
          ...prev,
          mainView: prev.mainView === 'live' ? 'timeline' : 'live',
        };
      }
      // General swap
      const newMain: MainContentView = prev.sidebarView === 'timeline' ? 'timeline' : 'live';
      const newSidebar: SidebarView = prev.mainView === 'live' ? 'preview' : 'timeline';
      return { ...prev, mainView: newMain, sidebarView: newSidebar };
    });
  }, []);

  const openTimelineSidebar = useCallback(() => {
    setState((prev) => ({
      ...prev,
      mainView: 'live',
      sidebarView: 'timeline',
    }));
  }, []);

  const openTimelineFullView = useCallback(() => {
    setState((prev) => ({
      ...prev,
      mainView: 'timeline',
      sidebarView: 'preview',
    }));
  }, []);

  const openLiveFullView = useCallback(() => {
    setState((prev) => ({
      ...prev,
      mainView: 'live',
      sidebarView: null,
    }));
  }, []);

  return {
    mainView: state.mainView,
    sidebarView: state.sidebarView,
    sidebarWidth: state.sidebarWidth,
    actionBarAutoHide: state.actionBarAutoHide,
    isSidebarOpen: state.sidebarView !== null,

    setMainView,
    setSidebarView,
    toggleSidebar,
    setSidebarWidth,
    handleSidebarResizeStart,
    setActionBarAutoHide,

    swapViews,
    openTimelineSidebar,
    openTimelineFullView,
    openLiveFullView,
  };
}
