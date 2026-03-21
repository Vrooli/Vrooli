// DOC: docs/concepts/ARCHITECTURE.md#ui-layer
import { useState, useRef, useCallback, useEffect, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { X } from "lucide-react";
import { listInformation, updateInformation, deleteInformation } from "../lib/api";
import { CANVAS_ZOOM_MIN, CANVAS_ZOOM_MAX, CANVAS_ZOOM_IN_FACTOR, CANVAS_ZOOM_OUT_FACTOR, CANVAS_PAN_STEP } from "../lib/config";
import { useMutationErrors } from "../hooks/useMutationErrors";
import { useWindowDrag } from "../hooks/useWindowDrag";
import { ErrorBanner } from "./ErrorBanner";
import { KeyboardShortcutHelp } from "./KeyboardShortcutHelp";
import type { Information } from "../lib/types";

interface Props {
  schemeId: string;
}

export function CanvasView({ schemeId }: Props) {
  const qc = useQueryClient();
  const { data: items = [], isLoading } = useQuery({
    queryKey: ["information", schemeId],
    queryFn: () => listInformation(schemeId),
  });

  const [pan, setPan] = useState({ x: 0, y: 0 });
  const [zoom, setZoom] = useState(1);
  const [showHelp, setShowHelp] = useState(false);
  const canvasRef = useRef<HTMLDivElement>(null);

  // Track the item being dragged so the move/end callbacks can reference it
  const dragItemRef = useRef<{ id: string; origX: number; origY: number } | null>(null);

  const updateMut = useMutation({
    mutationFn: ({ id, x, y }: { id: string; x: number; y: number }) =>
      updateInformation(schemeId, id, { canvas_x: x, canvas_y: y }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["information", schemeId] }),
  });

  const deleteMut = useMutation({
    mutationFn: (id: string) => deleteInformation(schemeId, id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["information", schemeId] }),
  });

  // Item drag — moves the DOM element during drag, commits position on drop
  const itemDragCallbacks = useMemo(
    () => ({
      onMove: (dx: number, dy: number) => {
        const item = dragItemRef.current;
        if (!item) return;
        const el = document.querySelector<HTMLElement>(`[data-item-id="${item.id}"]`);
        if (el) {
          el.style.left = `${item.origX + dx}px`;
          el.style.top = `${item.origY + dy}px`;
        }
      },
      onEnd: (dx: number, dy: number) => {
        const item = dragItemRef.current;
        if (!item) return;
        updateMut.mutate({ id: item.id, x: item.origX + dx, y: item.origY + dy });
        dragItemRef.current = null;
      },
    }),
    [],
  );
  const zoomRef = useRef(zoom);
  zoomRef.current = zoom;
  const { startDrag: startItemDrag } = useWindowDrag({
    ...itemDragCallbacks,
    get scale() { return 1 / zoomRef.current; },
  });

  // Pan drag — updates pan state during drag
  const panOriginRef = useRef({ x: 0, y: 0 });
  const panDragCallbacks = useMemo(
    () => ({
      onMove: (dx: number, dy: number) => {
        const orig = panOriginRef.current;
        setPan({ x: orig.x + dx, y: orig.y + dy });
      },
    }),
    [],
  );
  const { startDrag: startPanDrag } = useWindowDrag(panDragCallbacks);

  const handleWheel = useCallback((e: React.WheelEvent) => {
    e.preventDefault();
    const delta = e.deltaY > 0 ? CANVAS_ZOOM_OUT_FACTOR : CANVAS_ZOOM_IN_FACTOR;
    setZoom((z) => Math.max(CANVAS_ZOOM_MIN, Math.min(CANVAS_ZOOM_MAX, z * delta)));
  }, []);

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    switch (e.key) {
      case "ArrowUp":
        e.preventDefault();
        setPan((p) => ({ ...p, y: p.y + CANVAS_PAN_STEP }));
        break;
      case "ArrowDown":
        e.preventDefault();
        setPan((p) => ({ ...p, y: p.y - CANVAS_PAN_STEP }));
        break;
      case "ArrowLeft":
        e.preventDefault();
        setPan((p) => ({ ...p, x: p.x + CANVAS_PAN_STEP }));
        break;
      case "ArrowRight":
        e.preventDefault();
        setPan((p) => ({ ...p, x: p.x - CANVAS_PAN_STEP }));
        break;
      case "+":
      case "=":
        e.preventDefault();
        setZoom((z) => Math.min(CANVAS_ZOOM_MAX, z * CANVAS_ZOOM_IN_FACTOR));
        break;
      case "-":
        e.preventDefault();
        setZoom((z) => Math.max(CANVAS_ZOOM_MIN, z * CANVAS_ZOOM_OUT_FACTOR));
        break;
      case "?":
        e.preventDefault();
        setShowHelp((v) => !v);
        break;
    }
  }, []);

  const handleMouseDown = (e: React.MouseEvent, item: Information) => {
    e.stopPropagation();
    dragItemRef.current = { id: item.id, origX: item.canvas_x, origY: item.canvas_y };
    startItemDrag(e);
  };

  // Clear drag state when scheme changes to avoid stale item references
  useEffect(() => {
    dragItemRef.current = null;
    setPan({ x: 0, y: 0 });
    setZoom(1);
  }, [schemeId]);

  const handleCanvasMouseDown = (e: React.MouseEvent) => {
    if (e.target === canvasRef.current) {
      panOriginRef.current = { ...pan };
      startPanDrag(e);
    }
  };

  const { activeError, resetAll } = useMutationErrors([updateMut, deleteMut]);

  return (
    <div
      ref={canvasRef}
      data-testid="canvas-view"
      className="flex-1 relative overflow-hidden bg-slate-950 cursor-grab active:cursor-grabbing focus:outline-none focus:ring-1 focus:ring-white/20"
      onWheel={handleWheel}
      onMouseDown={handleCanvasMouseDown}
      onKeyDown={handleKeyDown}
      tabIndex={0}
      role="application"
      aria-label="Spatial canvas. Use arrow keys to pan, plus and minus to zoom."
    >
      {activeError && (
        <div className="absolute top-2 left-2 right-2 z-10">
          <ErrorBanner
            error={activeError}
            onRetry={resetAll}
            onDismiss={resetAll}
          />
        </div>
      )}
      {isLoading && items.length === 0 && (
        <div className="absolute inset-0 flex items-center justify-center">
          <p className="text-sm text-slate-500">Loading items...</p>
        </div>
      )}
      <div
        style={{
          transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom})`,
          transformOrigin: "0 0",
        }}
        className="absolute inset-0"
      >
        {items.map((item) => (
          <div
            key={item.id}
            data-item-id={item.id}
            data-testid="canvas-node"
            onMouseDown={(e) => handleMouseDown(e, item)}
            style={{ left: item.canvas_x, top: item.canvas_y }}
            className="absolute group rounded-lg border border-white/10 bg-slate-800/90 p-3 min-w-[140px] max-w-[280px] cursor-move select-none shadow-lg hover:border-white/20"
          >
            <button
              onClick={(e) => {
                e.stopPropagation();
                deleteMut.mutate(item.id);
              }}
              className="absolute -top-2 -right-2 p-0.5 rounded-full bg-slate-700 text-slate-400 opacity-0 group-hover:opacity-100 hover:text-red-400"
              aria-label="Delete item"
            >
              <X className="h-3 w-3" aria-hidden="true" />
            </button>
            <div className="text-[10px] uppercase tracking-wider text-slate-500 mb-1">{item.type}</div>
            <div className="text-sm text-slate-200 whitespace-pre-wrap break-words">{item.content}</div>
          </div>
        ))}
      </div>
      <KeyboardShortcutHelp open={showHelp} onClose={() => setShowHelp(false)} />
      <div className="absolute bottom-3 right-3 flex items-center gap-2 text-xs text-slate-600" aria-hidden="true">
        <span className="opacity-60">Press ? for shortcuts</span>
        <span>{Math.round(zoom * 100)}%</span>
      </div>
      <div className="sr-only" aria-live="polite" role="status">
        Zoom {Math.round(zoom * 100)}%
      </div>
    </div>
  );
}
