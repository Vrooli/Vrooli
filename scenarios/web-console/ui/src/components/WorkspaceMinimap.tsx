import { useCallback, useEffect, useRef, useState } from "react";
import type { RefObject } from "react";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import {
  buildMinimapRowMarkers,
  viewportFromScrollMetrics,
  scrollTopFromMinimapPointer,
  MINIMAP_MIN_OVERFLOW_PX,
} from "../lib/minimapLayout";

interface WorkspaceMinimapProps {
  scrollRef: RefObject<HTMLDivElement | null>;
  rowCount: number;
}

type ScrollMetrics = {
  scrollTop: number;
  scrollHeight: number;
  clientHeight: number;
};

export default function WorkspaceMinimap({ scrollRef, rowCount }: WorkspaceMinimapProps) {
  const isMinimapVisible = useWorkspaceStore((s) => s.isMinimapVisible);
  const railRef = useRef<HTMLDivElement>(null);
  const draggingRef = useRef(false);

  const [metrics, setMetrics] = useState<ScrollMetrics>({
    scrollTop: 0,
    scrollHeight: 0,
    clientHeight: 0,
  });

  // Track scroll metrics via scroll + resize listeners
  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;

    const update = () => {
      setMetrics({
        scrollTop: el.scrollTop,
        scrollHeight: el.scrollHeight,
        clientHeight: el.clientHeight,
      });
    };

    update();
    el.addEventListener("scroll", update, { passive: true });
    const observer = new ResizeObserver(update);
    observer.observe(el);

    return () => {
      el.removeEventListener("scroll", update);
      observer.disconnect();
    };
  }, [scrollRef]);

  const viewport = viewportFromScrollMetrics(metrics);
  const markers = buildMinimapRowMarkers(rowCount);

  const jumpTo = useCallback(
    (pointerOffsetY: number) => {
      const rail = railRef.current;
      const el = scrollRef.current;
      if (!rail || !el) return;
      const top = scrollTopFromMinimapPointer(
        pointerOffsetY,
        rail.clientHeight,
        el.scrollHeight,
        el.clientHeight,
      );
      el.scrollTo({ top, behavior: "auto" });
    },
    [scrollRef],
  );

  const handlePointerDown = useCallback(
    (e: React.PointerEvent) => {
      e.preventDefault();
      (e.currentTarget as HTMLElement).setPointerCapture?.(e.pointerId);
      draggingRef.current = true;
      const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
      jumpTo(e.clientY - rect.top);
    },
    [jumpTo],
  );

  const handlePointerMove = useCallback(
    (e: React.PointerEvent) => {
      if (!draggingRef.current) return;
      const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
      jumpTo(e.clientY - rect.top);
    },
    [jumpTo],
  );

  const handlePointerUp = useCallback(() => {
    draggingRef.current = false;
  }, []);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      const el = scrollRef.current;
      if (!el) return;

      let next: number | null = null;
      switch (e.key) {
        case "ArrowDown":
          next = el.scrollTop + 120;
          break;
        case "ArrowUp":
          next = el.scrollTop - 120;
          break;
        case "PageDown":
          next = el.scrollTop + el.clientHeight;
          break;
        case "PageUp":
          next = el.scrollTop - el.clientHeight;
          break;
        case "Home":
          next = 0;
          break;
        case "End":
          next = el.scrollHeight - el.clientHeight;
          break;
        default:
          return;
      }
      e.preventDefault();
      el.scrollTo({ top: Math.max(0, next), behavior: "auto" });
    },
    [scrollRef],
  );

  if (!isMinimapVisible) return null;
  if (viewport.maxScrollable <= MINIMAP_MIN_OVERFLOW_PX) return null;

  // Determine active row from viewport center
  const viewportCenter = viewport.topPercent + viewport.heightPercent / 2;
  const activeRowIndex = markers.length > 0
    ? Math.min(
        markers.length - 1,
        Math.max(0, Math.floor((viewportCenter / 100) * markers.length)),
      )
    : -1;

  return (
    <aside className="wc-minimap" data-testid="workspace-minimap">
      <div
        ref={railRef}
        className="wc-minimap-rail"
        role="slider"
        aria-label="Scroll position"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={Math.round(viewport.topPercent)}
        tabIndex={0}
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={handlePointerUp}
        onPointerCancel={handlePointerUp}
        onKeyDown={handleKeyDown}
      >
        <div className="wc-minimap-markers">
          {markers.map((m) => (
            <div
              key={m.rowIndex}
              className={`wc-minimap-marker${m.rowIndex === activeRowIndex ? " wc-minimap-marker--active" : ""}`}
              style={{
                top: `${m.topPercent}%`,
                height: `${m.heightPercent}%`,
              }}
            />
          ))}
        </div>
        <div
          className="wc-minimap-viewport"
          style={{
            top: `${viewport.topPercent}%`,
            height: `${viewport.heightPercent}%`,
          }}
        />
      </div>
    </aside>
  );
}
