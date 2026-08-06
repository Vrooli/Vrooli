/** @vrooliComponentSource hooks.use-outside-interaction */
import { useEffect, useRef, type RefObject } from "react";

export interface OutsideInteractionOptions {
  active?: boolean;
  surfaceRef: RefObject<HTMLElement | null>;
  excludeRefs?: Array<RefObject<HTMLElement | null>>;
  onPointerDownOutside?: (event: PointerEvent) => void;
  onFocusOutside?: (event: FocusEvent) => void;
  onEscape?: (event: KeyboardEvent) => void;
  dismissOnPointerDown?: boolean;
  dismissOnFocus?: boolean;
  dismissOnEscape?: boolean;
}

function eventPathContains(event: Event, node: Node | null) {
  if (!node) return false;
  const path =
    typeof event.composedPath === "function" ? event.composedPath() : [];
  return path.includes(node) || node.contains(event.target as Node | null);
}

function isInside(
  event: Event,
  surfaceRef: OutsideInteractionOptions["surfaceRef"],
  excludeRefs: OutsideInteractionOptions["excludeRefs"],
) {
  if (eventPathContains(event, surfaceRef.current)) return true;
  return (excludeRefs ?? []).some((ref) =>
    eventPathContains(event, ref.current),
  );
}

/**
 * Coordinates dismissal across portals and nested layers without assuming
 * that the event target shares a DOM subtree with the surface.
 */
export function useOutsideInteraction({
  active = true,
  surfaceRef,
  excludeRefs = [],
  onPointerDownOutside,
  onFocusOutside,
  onEscape,
  dismissOnPointerDown = true,
  dismissOnFocus = true,
  dismissOnEscape = true,
}: OutsideInteractionOptions) {
  const callbacks = useRef({ onPointerDownOutside, onFocusOutside, onEscape });
  const lastPointerTarget = useRef<EventTarget | null>(null);
  callbacks.current = { onPointerDownOutside, onFocusOutside, onEscape };

  useEffect(() => {
    if (!active || typeof document === "undefined") return;
    const pointerDown = (event: PointerEvent) => {
      if (
        !dismissOnPointerDown ||
        event.defaultPrevented ||
        isInside(event, surfaceRef, excludeRefs)
      )
        return;
      lastPointerTarget.current = event.target;
      callbacks.current.onPointerDownOutside?.(event);
    };
    const click = (event: MouseEvent) => {
      if (
        !dismissOnPointerDown ||
        event.defaultPrevented ||
        isInside(event, surfaceRef, excludeRefs)
      )
        return;
      if (lastPointerTarget.current === event.target) {
        lastPointerTarget.current = null;
        return;
      }
      callbacks.current.onPointerDownOutside?.(
        event as unknown as PointerEvent,
      );
    };
    const focusIn = (event: FocusEvent) => {
      if (
        !dismissOnFocus ||
        event.defaultPrevented ||
        isInside(event, surfaceRef, excludeRefs)
      )
        return;
      callbacks.current.onFocusOutside?.(event);
    };
    const keyDown = (event: KeyboardEvent) => {
      if (dismissOnEscape && event.key === "Escape" && !event.defaultPrevented)
        callbacks.current.onEscape?.(event);
    };
    document.addEventListener("pointerdown", pointerDown, true);
    document.addEventListener("click", click, true);
    document.addEventListener("focusin", focusIn, true);
    window.addEventListener("keydown", keyDown, true);
    return () => {
      document.removeEventListener("pointerdown", pointerDown, true);
      document.removeEventListener("click", click, true);
      document.removeEventListener("focusin", focusIn, true);
      window.removeEventListener("keydown", keyDown, true);
    };
  }, [
    active,
    dismissOnEscape,
    dismissOnFocus,
    dismissOnPointerDown,
    excludeRefs,
    surfaceRef,
  ]);
}
