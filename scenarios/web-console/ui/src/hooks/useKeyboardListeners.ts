import { useEffect, useRef, type RefObject } from "react";
import { emitShortcutIntent } from "@vrooli/iframe-bridge";
import { matchesShortcut, parseShortcut } from "../lib/shortcutParser";

type KeyHandler = (event: KeyboardEvent) => void;

/** Keeps window-level keyboard listeners in the hook layer for iframe safety. */
export function useWindowKeyDown(active: boolean, handler: KeyHandler): void {
  const handlerRef = useRef(handler);
  handlerRef.current = handler;

  useEffect(() => {
    if (!active) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.ctrlKey || event.metaKey || event.altKey) {
        emitShortcutIntent({
          action: "keyboard.shortcut",
          chord: [event.ctrlKey && "Ctrl", event.metaKey && "Meta", event.altKey && "Alt", event.shiftKey && "Shift", event.key]
            .filter(Boolean)
            .join("+"),
          source: "keyboard",
        });
      }
      handlerRef.current(event);
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [active]);
}

/** Releases a scroll pin when the user interacts with a scrollable element. */
export function useReleaseOnElementInteraction(
  elementRef: RefObject<HTMLElement | null>,
  onRelease: () => void,
): void {
  const releaseRef = useRef(onRelease);
  releaseRef.current = onRelease;

  useEffect(() => {
    const element = elementRef.current;
    if (!element) return;
    const release = () => releaseRef.current();
    element.addEventListener("wheel", release, { passive: true });
    element.addEventListener("touchstart", release, { passive: true });
    element.addEventListener("pointerdown", release, { passive: true });
    element.addEventListener("keydown", release);
    return () => {
      element.removeEventListener("wheel", release);
      element.removeEventListener("touchstart", release);
      element.removeEventListener("pointerdown", release);
      element.removeEventListener("keydown", release);
    };
  }, [elementRef]);
}

/** Captures the configured voice shortcut before xterm processes the key. */
export function useTerminalVoiceShortcut(
  elementRef: RefObject<HTMLElement | null>,
  shortcut: string,
  onStart?: () => void,
  onStop?: () => void,
): void {
  const callbacksRef = useRef({ onStart, onStop });
  callbacksRef.current = { onStart, onStop };

  useEffect(() => {
    const element = elementRef.current;
    if (!element || !onStart || !onStop) return;
    const parsed = parseShortcut(shortcut);
    if (!parsed) return;

    const handler = (event: KeyboardEvent) => {
      if (!matchesShortcut(event, parsed)) return;
      event.preventDefault();
      event.stopPropagation();
      if (event.type === "keydown") callbacksRef.current.onStart?.();
      if (event.type === "keyup") callbacksRef.current.onStop?.();
    };

    element.addEventListener("keydown", handler, { capture: true });
    element.addEventListener("keyup", handler, { capture: true });
    return () => {
      element.removeEventListener("keydown", handler, { capture: true });
      element.removeEventListener("keyup", handler, { capture: true });
    };
  }, [elementRef, onStart, onStop, shortcut]);
}
