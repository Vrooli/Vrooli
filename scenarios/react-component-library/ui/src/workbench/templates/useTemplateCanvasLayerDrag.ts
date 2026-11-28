import { useCallback, useRef, useState, type MouseEvent, type PointerEvent } from "react";
import type { LayerDropPosition } from "../studio/layerMovement";
import { CONTAINER_TYPES } from "../studio/model";
import { findNode, type TemplateNode } from "./spec";

interface Options {
  spec: TemplateNode;
  interactive: boolean;
  onSelect?: (id: string) => void;
  onMoveLayer?: (sourceId: string, targetId: string, position: LayerDropPosition) => void;
}

export function useTemplateCanvasLayerDrag({ spec, interactive, onSelect, onMoveLayer }: Options) {
  const rootRef = useRef<HTMLDivElement>(null);
  const [canvasDrag, setCanvasDrag] = useState<{
    sourceId: string;
    targetId: string | null;
    position: LayerDropPosition;
  } | null>(null);
  const dragDidMove = useRef(false);
  const canvasDragRef = useRef<{
    sourceId: string;
    pointerId: number;
    startX: number;
    startY: number;
    started: boolean;
    targetId: string | null;
    position: LayerDropPosition;
  } | null>(null);

  const dropPositionAt = useCallback(
    (target: HTMLElement, clientX: number, clientY: number): LayerDropPosition => {
      const nodeId = target.dataset.nodeId;
      const node = nodeId ? findNode(spec, nodeId) : undefined;
      const rect = target.getBoundingClientRect();
      const horizontal = window
        .getComputedStyle(target.parentElement ?? target)
        .flexDirection.startsWith("row");
      const ratio = horizontal
        ? (clientX - rect.left) / Math.max(rect.width, 1)
        : (clientY - rect.top) / Math.max(rect.height, 1);
      if (node && CONTAINER_TYPES.has(node.type) && ratio > 0.25 && ratio < 0.75) return "inside";
      return ratio < 0.5 ? "before" : "after";
    },
    [spec],
  );

  const onPointerDown = (event: PointerEvent<HTMLDivElement>) => {
    if (!interactive || !onMoveLayer || (event.button !== 0 && event.pointerType === "mouse"))
      return;
    const target = (event.target as HTMLElement).closest<HTMLElement>("[data-node-id]");
    const sourceId = target?.dataset.nodeId;
    if (!sourceId || sourceId === spec.id) return;
    canvasDragRef.current = {
      sourceId,
      pointerId: event.pointerId,
      startX: event.clientX,
      startY: event.clientY,
      started: false,
      targetId: null,
      position: "after",
    };
    try {
      event.currentTarget.setPointerCapture(event.pointerId);
    } catch {
      // Pointer capture is unavailable in a few browser/test surfaces.
    }
    event.preventDefault();
    onSelect?.(sourceId);
  };

  const onPointerMove = (event: PointerEvent<HTMLDivElement>) => {
    const drag = canvasDragRef.current;
    if (!drag || drag.pointerId !== event.pointerId) return;
    if (!drag.started && Math.hypot(event.clientX - drag.startX, event.clientY - drag.startY) < 5)
      return;
    drag.started = true;
    dragDidMove.current = true;
    const hit = document.elementFromPoint(event.clientX, event.clientY);
    const target = hit?.closest<HTMLElement>("[data-node-id]");
    const targetId =
      target && rootRef.current?.contains(target) && target.dataset.nodeId !== drag.sourceId
        ? (target.dataset.nodeId ?? null)
        : null;
    drag.targetId = targetId;
    if (target && targetId) drag.position = dropPositionAt(target, event.clientX, event.clientY);
    setCanvasDrag({ sourceId: drag.sourceId, targetId, position: drag.position });
    event.preventDefault();
  };

  const onPointerUp = (event: PointerEvent<HTMLDivElement>) => {
    const drag = canvasDragRef.current;
    if (!drag || drag.pointerId !== event.pointerId) return;
    canvasDragRef.current = null;
    try {
      event.currentTarget.releasePointerCapture(event.pointerId);
    } catch {
      // The pointer may have left the artboard before release.
    }
    if (drag.started && drag.targetId) {
      event.preventDefault();
      onMoveLayer?.(drag.sourceId, drag.targetId, drag.position);
      onSelect?.(drag.sourceId);
    }
    setCanvasDrag(null);
  };

  const onClick = (event: MouseEvent<HTMLDivElement>) => {
    if (dragDidMove.current) {
      dragDidMove.current = false;
      event.preventDefault();
      return;
    }
    if (!interactive) return;
    const target = (event.target as HTMLElement).closest<HTMLElement>("[data-node-id]");
    if (target?.dataset.nodeId) {
      event.stopPropagation();
      onSelect?.(target.dataset.nodeId);
    }
  };

  return { rootRef, canvasDrag, onPointerDown, onPointerMove, onPointerUp, onClick };
}
