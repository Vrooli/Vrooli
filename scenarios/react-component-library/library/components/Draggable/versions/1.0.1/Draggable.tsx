/**
 * @libraryId react-component-library:Draggable
 * @displayName Draggable
 * @description A pointer-capturing and keyboard-operable direct-manipulation primitive with constraints, cancellation, announcements, and overlay-safe positioning.
 * @version 1.0.1
 * @tags ["manipulation","drag-drop","keyboard","motion","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

/** @vrooliComponentSource manipulation.draggable */
import { resolveStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { useEffect, useMemo, useRef, useState, type CSSProperties, type ReactNode } from "react";
import { useAnnounce } from "@vrooli/react-component-library/useAnnounce/1.0.0";
import { useDrag } from "@vrooli/react-component-library/useDrag/1.0.0";
import {
  createDragDropStore,
  type DragPosition,
} from "@vrooli/react-component-library/DragDropStore/1.0.0";

export interface DragBounds {
  left?: number;
  right?: number;
  top?: number;
  bottom?: number;
}
export interface DraggableProps {
  id: string;
  children: ReactNode;
  defaultPosition?: DragPosition;
  position?: DragPosition;
  onPositionChange?: (position: DragPosition) => void;
  onDragStart?: () => void;
  onDragEnd?: (position: DragPosition) => void;
  onCancel?: () => void;
  bounds?: DragBounds;
  step?: number;
  disabled?: boolean;
  label?: string;
  className?: string;
  style?: CSSProperties;
}

const styles = `
  [data-rcl-draggable] { position: relative; min-inline-size: 0; touch-action: none; cursor: grab; transform: translate3d(var(--rcl-drag-x), var(--rcl-drag-y), 0); transition: box-shadow var(--dur-quick, 160ms) var(--ease-standard, ease), opacity var(--dur-quick, 160ms) var(--ease-standard, ease); }
  [data-rcl-draggable]:hover { box-shadow: var(--elev-raised, 0 10px 24px rgb(15 23 42 / .12)); }
  [data-rcl-draggable]:focus-visible { outline: 3px solid color-mix(in srgb, var(--color-focus, #2563eb) 42%, transparent); outline-offset: 4px; }
  [data-rcl-draggable][data-dragging="true"] { z-index: 5; cursor: grabbing; box-shadow: var(--elev-overlay, 0 18px 42px rgb(15 23 42 / .2)); transition: none; }
  [data-rcl-draggable][data-disabled="true"] { cursor: not-allowed; opacity: .58; }
  [data-rcl-draggable-status] { position: absolute; inline-size: 1px; block-size: 1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; }
  @media (prefers-reduced-motion: reduce) { [data-rcl-draggable] { transition: none; } }
`;

function clamp(position: DragPosition, bounds: DragBounds): DragPosition {
  return {
    x: Math.max(bounds.left ?? -Infinity, Math.min(bounds.right ?? Infinity, position.x)),
    y: Math.max(bounds.top ?? -Infinity, Math.min(bounds.bottom ?? Infinity, position.y)),
  };
}

export const Draggable = withClassName(function Draggable({
  id,
  children,
  defaultPosition = { x: 0, y: 0 },
  position,
  onPositionChange,
  onDragStart,
  onDragEnd,
  onCancel,
  bounds = {},
  step = 8,
  disabled = false,
  label = resolveStrings("manipulation.draggable.draggable-item", "Draggable item"),
  className,
  style,
}: DraggableProps) {
  const announce = useAnnounce();
  const initialX = defaultPosition.x;
  const initialY = defaultPosition.y;
  const store = useMemo(
    () => createDragDropStore({ x: initialX, y: initialY }),
    [initialX, initialY],
  );
  const [localPosition, setLocalPosition] = useState(defaultPosition);
  const [, setStoreVersion] = useState(0);
  const resolved = position ?? localPosition;
  const dragOrigin = useRef(resolved);
  const lastPointer = useRef<{ x: number; y: number; time: number }>();
  const update = (next: DragPosition) => {
    const constrained = clamp(next, bounds);
    if (position === undefined) setLocalPosition(constrained);
    store.move(constrained);
    onPositionChange?.(constrained);
    return constrained;
  };
  const drag = useDrag({
    disabled,
    onStart: (start) => {
      dragOrigin.current = resolved;
      lastPointer.current = {
        x: start.x,
        y: start.y,
        time: performance.now(),
      };
      store.start(id, "pointer", resolved);
      onDragStart?.();
      announce(`${label} picked up. Use arrow keys to move, Escape to cancel.`);
    },
    onMove: (event, start) => {
      const next = update({
        x: dragOrigin.current.x + event.clientX - start.x,
        y: dragOrigin.current.y + event.clientY - start.y,
      });
      const previous = lastPointer.current;
      const now = performance.now();
      const elapsed = Math.max(now - (previous?.time ?? now), 1);
      store.move(next, {
        x: ((event.clientX - (previous?.x ?? event.clientX)) / elapsed) * 1000,
        y: ((event.clientY - (previous?.y ?? event.clientY)) / elapsed) * 1000,
      });
      lastPointer.current = { x: event.clientX, y: event.clientY, time: now };
    },
    onEnd: () => {
      lastPointer.current = undefined;
      store.end();
      onDragEnd?.(store.get().position);
      announce(
        `${label} dropped at ${Math.round(store.get().position.x)}, ${Math.round(store.get().position.y)}.`,
      );
    },
    onCancel: () => {
      lastPointer.current = undefined;
      store.cancel();
      if (position === undefined) setLocalPosition(defaultPosition);
      onCancel?.();
      announce(`${label} cancelled.`);
    },
    onKeyboardMove: (dx, dy) => {
      const current = store.get().position;
      const next = update({
        x: current.x + dx * step,
        y: current.y + dy * step,
      });
      store.start(id, "keyboard", next);
      announce(`${label} at ${Math.round(next.x)}, ${Math.round(next.y)}.`);
    },
    onKeyboardEnd: () => {
      store.end();
      onDragEnd?.(store.get().position);
      announce(
        `${label} dropped at ${Math.round(store.get().position.x)}, ${Math.round(store.get().position.y)}.`,
      );
    },
  });
  const { isDragging, ...dragHandlers } = drag;
  const state = store.get();
  useEffect(() => {
    return store.subscribe(() => {
      setLocalPosition(position ?? store.get().position);
      setStoreVersion((version) => version + 1);
    });
  }, [position, store]);
  const dragStyle: CSSProperties = {
    ...style,
    "--rcl-drag-x": `${resolved.x}px`,
    "--rcl-drag-y": `${resolved.y}px`,
  } as CSSProperties;
  return (
    <>
      <style data-rcl-draggable-styles dangerouslySetInnerHTML={{ __html: styles }} />
      <div data-testid="manipulation.draggable"
        data-rcl-draggable
        data-dragging={isDragging || state.phase !== "idle"}
        data-disabled={disabled || undefined}
        className={className}
        style={dragStyle}
        role="group"
        aria-label={label}
        tabIndex={disabled ? -1 : 0}
        {...dragHandlers}
      >
        <span data-rcl-draggable-status role="status" aria-live="polite">
          {state.phase === "keyboard" ? `Moving ${label}` : ""}
        </span>
        {children}
      </div>
    </>
  );
});
