// DOC: docs/concepts/ARCHITECTURE.md#ui-layer
import { useState, useRef, useCallback, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { X } from "lucide-react";
import { listInformation, updateInformation, deleteInformation } from "../lib/api";
import { CANVAS_ZOOM_MIN, CANVAS_ZOOM_MAX, CANVAS_ZOOM_IN_FACTOR, CANVAS_ZOOM_OUT_FACTOR } from "../lib/config";
import { useMutationErrors } from "../hooks/useMutationErrors";
import { ErrorBanner } from "./ErrorBanner";
import type { Information } from "../lib/types";

interface Props {
  schemeId: string;
}

interface DragState {
  itemId: string;
  startX: number;
  startY: number;
  origX: number;
  origY: number;
}

export function CanvasView({ schemeId }: Props) {
  const qc = useQueryClient();
  const { data: items = [], isLoading } = useQuery({
    queryKey: ["information", schemeId],
    queryFn: () => listInformation(schemeId),
  });

  const [pan, setPan] = useState({ x: 0, y: 0 });
  const [zoom, setZoom] = useState(1);
  const [, setDrag] = useState<DragState | null>(null);
  const canvasRef = useRef<HTMLDivElement>(null);
  const zoomRef = useRef(zoom);
  zoomRef.current = zoom;

  const updateMut = useMutation({
    mutationFn: ({ id, x, y }: { id: string; x: number; y: number }) =>
      updateInformation(schemeId, id, { canvas_x: x, canvas_y: y }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["information", schemeId] }),
  });

  const deleteMut = useMutation({
    mutationFn: (id: string) => deleteInformation(schemeId, id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["information", schemeId] }),
  });

  const handleWheel = useCallback((e: React.WheelEvent) => {
    e.preventDefault();
    const delta = e.deltaY > 0 ? CANVAS_ZOOM_OUT_FACTOR : CANVAS_ZOOM_IN_FACTOR;
    setZoom((z) => Math.max(CANVAS_ZOOM_MIN, Math.min(CANVAS_ZOOM_MAX, z * delta)));
  }, []);

  // Track item-drag cleanup so listeners can be removed on unmount or interruption.
  // The cleanup function removes the window listeners without committing the position.
  const dragCleanupRef = useRef<(() => void) | null>(null);

  const handleMouseDown = (e: React.MouseEvent, item: Information) => {
    e.stopPropagation();
    const dragState: DragState = { itemId: item.id, startX: e.clientX, startY: e.clientY, origX: item.canvas_x, origY: item.canvas_y };
    setDrag(dragState);

    // Attach window-level listeners so mouseup outside the canvas is still captured
    const onMove = (me: MouseEvent) => {
      const dx = (me.clientX - dragState.startX) / zoomRef.current;
      const dy = (me.clientY - dragState.startY) / zoomRef.current;
      const el = document.querySelector<HTMLElement>(`[data-item-id="${dragState.itemId}"]`);
      if (el) {
        el.style.left = `${dragState.origX + dx}px`;
        el.style.top = `${dragState.origY + dy}px`;
      }
    };
    const onUp = (me: MouseEvent) => {
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
      dragCleanupRef.current = null;
      const dx = (me.clientX - dragState.startX) / zoomRef.current;
      const dy = (me.clientY - dragState.startY) / zoomRef.current;
      updateMut.mutate({ id: dragState.itemId, x: dragState.origX + dx, y: dragState.origY + dy });
      setDrag(null);
    };
    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
    dragCleanupRef.current = () => {
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
    };
  };

  // Clear drag state when scheme changes to avoid stale item references
  useEffect(() => {
    setDrag(null);
    setPan({ x: 0, y: 0 });
    setZoom(1);
  }, [schemeId]);

  // Track pan-drag listeners so they can be cleaned up on unmount
  const panCleanupRef = useRef<(() => void) | null>(null);

  useEffect(() => {
    return () => {
      panCleanupRef.current?.();
      dragCleanupRef.current?.();
    };
  }, []);

  const handleCanvasMouseDown = (e: React.MouseEvent) => {
    if (e.target === canvasRef.current) {
      const startX = e.clientX;
      const startY = e.clientY;
      const origPan = { ...pan };

      const onMove = (me: MouseEvent) => {
        setPan({ x: origPan.x + (me.clientX - startX), y: origPan.y + (me.clientY - startY) });
      };
      const onUp = () => {
        window.removeEventListener("mousemove", onMove);
        window.removeEventListener("mouseup", onUp);
        panCleanupRef.current = null;
      };
      window.addEventListener("mousemove", onMove);
      window.addEventListener("mouseup", onUp);
      panCleanupRef.current = onUp;
    }
  };

  const { activeError, resetAll } = useMutationErrors([updateMut, deleteMut]);

  return (
    <div
      ref={canvasRef}
      data-testid="canvas-view"
      className="flex-1 relative overflow-hidden bg-slate-950 cursor-grab active:cursor-grabbing"
      onWheel={handleWheel}
      onMouseDown={handleCanvasMouseDown}
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
      <div className="absolute bottom-3 right-3 text-xs text-slate-600">
        {Math.round(zoom * 100)}%
      </div>
    </div>
  );
}
