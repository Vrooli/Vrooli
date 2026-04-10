import {
  useState,
  useEffect,
  useCallback,
  useRef,
  type MouseEvent as ReactMouseEvent,
  type MutableRefObject,
} from 'react';

const DEFAULT_WIDTH = 560;
const MIN_WIDTH = 320;
const MAX_WIDTH_RATIO = 0.75;
const SNAP_CLOSE_THRESHOLD = 200;

interface UseResizableSplitPanelOptions {
  defaultWidth?: number;
  minWidth?: number;
  maxWidthRatio?: number;
  anchor?: 'left' | 'right';
  storageKey?: string;
  snapCloseThreshold?: number;
}

interface UseResizableSplitPanelResult {
  width: number;
  isResizing: boolean;
  isCollapsed: boolean;
  containerRef: MutableRefObject<HTMLDivElement | null>;
  handleResizeStart: (e: ReactMouseEvent) => void;
  expand: () => void;
  collapse: () => void;
}

export function useResizableSplitPanel(
  options: UseResizableSplitPanelOptions = {}
): UseResizableSplitPanelResult {
  const {
    defaultWidth = DEFAULT_WIDTH,
    minWidth = MIN_WIDTH,
    maxWidthRatio = MAX_WIDTH_RATIO,
    anchor = 'left',
    storageKey = 'ecosystem-manager.markdownSplitWidth',
    snapCloseThreshold = SNAP_CLOSE_THRESHOLD,
  } = options;

  const containerRef = useRef<HTMLDivElement>(null);
  const resizeRef = useRef<{ startX: number; startWidth: number; maxWidth: number } | null>(null);
  const prevWidthRef = useRef<number>(defaultWidth);

  const [width, setWidth] = useState(() => {
    if (typeof window === 'undefined') return defaultWidth;
    const stored = localStorage.getItem(storageKey);
    if (stored) {
      const parsed = Number(stored);
      if (Number.isFinite(parsed) && (parsed === 0 || parsed >= minWidth)) {
        if (parsed > 0) prevWidthRef.current = parsed;
        return parsed;
      }
    }
    return defaultWidth;
  });

  const [isResizing, setIsResizing] = useState(false);

  useEffect(() => {
    if (typeof window !== 'undefined') {
      localStorage.setItem(storageKey, String(width));
    }
  }, [storageKey, width]);

  useEffect(() => {
    if (!containerRef.current || typeof ResizeObserver === 'undefined') return;

    const clamp = () => {
      if (!containerRef.current) return;
      const containerWidth = containerRef.current.clientWidth;
      const maxWidth = Math.floor(containerWidth * maxWidthRatio);

      setWidth((prev) => {
        if (prev === 0) return 0;
        if (prev > maxWidth) return Math.max(minWidth, maxWidth);
        if (prev < minWidth) return minWidth;
        return prev;
      });
    };

    clamp();
    const observer = new ResizeObserver(clamp);
    observer.observe(containerRef.current);

    return () => observer.disconnect();
  }, [maxWidthRatio, minWidth]);

  useEffect(() => {
    if (!isResizing) return;

    const handleMouseMove = (e: MouseEvent) => {
      if (!resizeRef.current) return;

      const delta = e.clientX - resizeRef.current.startX;
      const direction = anchor === 'right' ? -1 : 1;
      const rawWidth = resizeRef.current.startWidth + delta * direction;
      const clampedWidth = snapCloseThreshold > 0
        ? Math.max(0, Math.min(resizeRef.current.maxWidth, rawWidth))
        : Math.max(minWidth, Math.min(resizeRef.current.maxWidth, rawWidth));
      setWidth(clampedWidth);
    };

    const handleMouseUp = () => {
      setIsResizing(false);

      if (snapCloseThreshold > 0) {
        setWidth((prev) => {
          if (prev < snapCloseThreshold && prev < minWidth) {
            prevWidthRef.current = resizeRef.current?.startWidth ?? defaultWidth;
            resizeRef.current = null;
            return 0;
          }
          resizeRef.current = null;
          return prev < minWidth ? minWidth : prev;
        });
      } else {
        resizeRef.current = null;
      }

      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };

    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
    window.addEventListener('mousemove', handleMouseMove);
    window.addEventListener('mouseup', handleMouseUp);

    return () => {
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };
  }, [anchor, defaultWidth, isResizing, minWidth, snapCloseThreshold]);

  const handleResizeStart = useCallback(
    (e: ReactMouseEvent) => {
      e.preventDefault();
      if (!containerRef.current) return;

      const containerWidth = containerRef.current.clientWidth;
      const maxWidth = Math.floor(containerWidth * maxWidthRatio);

      resizeRef.current = {
        startX: e.clientX,
        startWidth: width,
        maxWidth,
      };
      setIsResizing(true);
    },
    [maxWidthRatio, width]
  );

  const expand = useCallback(() => {
    const restored = prevWidthRef.current > 0 ? prevWidthRef.current : defaultWidth;
    setWidth(restored);
  }, [defaultWidth]);

  const collapse = useCallback(() => {
    if (width > 0) {
      prevWidthRef.current = width;
    }
    setWidth(0);
  }, [width]);

  return {
    width,
    isResizing,
    isCollapsed: width === 0,
    containerRef,
    handleResizeStart,
    expand,
    collapse,
  };
}
