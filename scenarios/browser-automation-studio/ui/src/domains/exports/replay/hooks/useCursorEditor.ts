/**
 * useCursorEditor hook
 *
 * Manages cursor position editing via drag interactions on the replay canvas.
 * Handles pointer events and cursor override state mutations.
 */

import { useState, useCallback, type HTMLAttributes } from 'react';
import type {
  ReplayFrame,
  CursorAnimationOverride,
  CursorOverrideMap,
  CursorSpeedProfile,
  CursorPathStyle,
} from '../types';
import { clampNormalizedPoint, normalizeOverride } from '../utils/cursorMath';

export interface DragState {
  frameId: string;
  pointerId: number;
}

export interface UseCursorEditorOptions {
  currentFrame: ReplayFrame | null;
  screenshotRef: React.RefObject<HTMLDivElement | null>;
  canEditCursor: boolean;
  basePathStyle: CursorPathStyle;
  baseSpeedProfile: CursorSpeedProfile;
}

export interface UseCursorEditorResult {
  dragState: DragState | null;
  cursorOverrides: CursorOverrideMap;
  setCursorOverrides: React.Dispatch<React.SetStateAction<CursorOverrideMap>>;
  updateCursorOverride: (
    frameId: string,
    mutator: (previous: CursorAnimationOverride | undefined) => CursorAnimationOverride | null | undefined,
  ) => void;
  pointerEventProps: HTMLAttributes<HTMLDivElement>;
  isDraggingCursor: boolean;
}

export function useCursorEditor({
  currentFrame,
  screenshotRef,
  canEditCursor,
  basePathStyle,
  baseSpeedProfile,
}: UseCursorEditorOptions): UseCursorEditorResult {
  const [dragState, setDragState] = useState<DragState | null>(null);
  const [cursorOverrides, setCursorOverrides] = useState<CursorOverrideMap>({});

  const updateCursorOverride = useCallback(
    (
      frameId: string,
      mutator: (previous: CursorAnimationOverride | undefined) => CursorAnimationOverride | null | undefined,
    ) => {
      setCursorOverrides((prev) => {
        const previous = prev[frameId];
        const next = normalizeOverride(mutator(previous), basePathStyle, baseSpeedProfile);
        if (!next) {
          if (previous === undefined) {
            return prev;
          }
          const { [frameId]: _removed, ...rest } = prev;
          return rest;
        }
        return { ...prev, [frameId]: next };
      });
    },
    [basePathStyle, baseSpeedProfile],
  );

  const updateDragPosition = useCallback(
    (clientX: number, clientY: number) => {
      if (!screenshotRef.current || !currentFrame || !canEditCursor) {
        return;
      }
      const rect = screenshotRef.current.getBoundingClientRect();
      if (!rect.width || !rect.height) {
        return;
      }
      const normalized = clampNormalizedPoint({
        x: (clientX - rect.left) / rect.width,
        y: (clientY - rect.top) / rect.height,
      });

      updateCursorOverride(currentFrame.id, (previous) => {
        const existing = previous ?? {};
        const next: CursorAnimationOverride = {
          ...existing,
          target: normalized,
        };
        if (existing.path === undefined) {
          next.path = [];
        }
        return next;
      });
    },
    [canEditCursor, currentFrame, screenshotRef, updateCursorOverride],
  );

  const handlePointerDown = useCallback(
    (event: React.PointerEvent<HTMLDivElement>) => {
      if (!canEditCursor || !currentFrame) {
        return;
      }
      event.preventDefault();
      event.stopPropagation();
      setDragState({ frameId: currentFrame.id, pointerId: event.pointerId });
      if (!event.currentTarget.hasPointerCapture(event.pointerId)) {
        event.currentTarget.setPointerCapture(event.pointerId);
      }
      updateDragPosition(event.clientX, event.clientY);
    },
    [canEditCursor, currentFrame, updateDragPosition],
  );

  const handlePointerMove = useCallback(
    (event: React.PointerEvent<HTMLDivElement>) => {
      if (!canEditCursor || !dragState || dragState.pointerId !== event.pointerId || dragState.frameId !== currentFrame?.id) {
        return;
      }
      event.preventDefault();
      updateDragPosition(event.clientX, event.clientY);
    },
    [canEditCursor, currentFrame?.id, dragState, updateDragPosition],
  );

  const endDrag = useCallback(
    (event: React.PointerEvent<HTMLDivElement>) => {
      if (!canEditCursor || !dragState || dragState.pointerId !== event.pointerId) {
        return;
      }
      if (event.currentTarget.hasPointerCapture(event.pointerId)) {
        event.currentTarget.releasePointerCapture(event.pointerId);
      }
      setDragState(null);
    },
    [canEditCursor, dragState],
  );

  const handlePointerLeave = useCallback(
    (event: React.PointerEvent<HTMLDivElement>) => {
      if (!canEditCursor || !dragState?.pointerId) {
        return;
      }
      endDrag(event);
    },
    [canEditCursor, dragState?.pointerId, endDrag],
  );

  const pointerEventProps: HTMLAttributes<HTMLDivElement> = canEditCursor
    ? {
        onPointerDown: handlePointerDown,
        onPointerMove: handlePointerMove,
        onPointerUp: endDrag,
        onPointerCancel: endDrag,
        onPointerLeave: handlePointerLeave,
      }
    : {};

  const isDraggingCursor = dragState?.frameId === currentFrame?.id;

  return {
    dragState,
    cursorOverrides,
    setCursorOverrides,
    updateCursorOverride,
    pointerEventProps,
    isDraggingCursor,
  };
}
