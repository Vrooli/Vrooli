import { useCallback, useEffect, useRef, useState } from 'react';

const STORAGE_KEYS = {
  WIDTH: 'record-mode-timeline-width',
  OPEN: 'record-mode-timeline-open',
} as const;

const DEFAULT_WIDTH = 360;
const MIN_WIDTH = 240;
const MAX_WIDTH = 640;

interface TimelinePanelState {
  isSidebarOpen: boolean;
  timelineWidth: number;
  isResizingSidebar: boolean;
  handleSidebarToggle: () => void;
  handleSidebarResizeStart: (event: React.MouseEvent<HTMLDivElement>) => void;
}

export function useTimelinePanel(): TimelinePanelState {
  const [isSidebarOpen, setIsSidebarOpen] = useState<boolean>(() => {
    if (typeof window === 'undefined') return true;
    try {
      const stored = window.localStorage.getItem(STORAGE_KEYS.OPEN);
      return stored === null ? true : stored === 'true';
    } catch {
      return true;
    }
  });

  const [timelineWidth, setTimelineWidth] = useState<number>(() => {
    if (typeof window === 'undefined') return DEFAULT_WIDTH;
    try {
      const stored = window.localStorage.getItem(STORAGE_KEYS.WIDTH);
      const parsed = stored ? Number(stored) : DEFAULT_WIDTH;
      if (Number.isNaN(parsed)) return DEFAULT_WIDTH;
      return Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, parsed));
    } catch {
      return DEFAULT_WIDTH;
    }
  });

  const [isResizingSidebar, setIsResizingSidebar] = useState(false);
  const resizeStartXRef = useRef(0);
  const resizeStartWidthRef = useRef(DEFAULT_WIDTH);

  useEffect(() => {
    try {
      window.localStorage.setItem(STORAGE_KEYS.OPEN, isSidebarOpen ? 'true' : 'false');
    } catch {
      // best-effort persistence only
    }
  }, [isSidebarOpen]);

  useEffect(() => {
    try {
      window.localStorage.setItem(STORAGE_KEYS.WIDTH, String(timelineWidth));
    } catch {
      // best-effort persistence only
    }
  }, [timelineWidth]);

  const handleSidebarToggle = useCallback(() => {
    setIsSidebarOpen((prev) => !prev);
  }, []);

  const handleSidebarResizeStart = useCallback((event: React.MouseEvent<HTMLDivElement>) => {
    event.preventDefault();
    resizeStartXRef.current = event.clientX;
    resizeStartWidthRef.current = timelineWidth;
    setIsResizingSidebar(true);

    const onMouseMove = (moveEvent: MouseEvent) => {
      const delta = moveEvent.clientX - resizeStartXRef.current;
      const next = Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, resizeStartWidthRef.current + delta));
      setTimelineWidth(next);
    };

    const onMouseUp = () => {
      setIsResizingSidebar(false);
      window.removeEventListener('mousemove', onMouseMove);
      window.removeEventListener('mouseup', onMouseUp);
    };

    window.addEventListener('mousemove', onMouseMove);
    window.addEventListener('mouseup', onMouseUp);
  }, [timelineWidth]);

  return {
    isSidebarOpen,
    timelineWidth,
    isResizingSidebar,
    handleSidebarToggle,
    handleSidebarResizeStart,
  };
}
