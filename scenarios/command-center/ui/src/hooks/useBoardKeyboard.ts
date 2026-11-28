import { useEffect } from "react";
import { emitShortcutIntent } from "@vrooli/iframe-bridge";
import type { BoardIntent } from "../lib/boardContext";

const CYCLE_EXEMPT_PATHS = new Set(["/settings"]);
const isInteractiveTarget = (target: EventTarget | null): boolean => {
  const element = target as Element | null;
  if (!element || typeof element.closest !== "function") return false;
  return element.closest("input, textarea, select, button, a[href], summary, [contenteditable]:not([contenteditable='false']), [role='button'], [role='link'], [role='checkbox'], [role='radio'], [role='slider'], [role='spinbutton'], [role='combobox'], [role='menuitem'], [role='switch']") !== null;
};
const KEY_INTENTS: Record<string, BoardIntent> = {
  arrowright: "navigate-right", arrowleft: "navigate-left", arrowup: "navigate-up", arrowdown: "navigate-down",
  "[": "navigate-beat-prev", "]": "navigate-beat-next", " ": "pause-cycle", f: "toggle-fullscreen", "?": "show-help", h: "show-help", enter: "inspect", escape: "back",
};

export function useBoardKeyboard(path: string, rooms: { id: string }[], dispatch: (intent: BoardIntent) => void, goTo: (path: string) => void): void {
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (CYCLE_EXEMPT_PATHS.has(path) || isInteractiveTarget(event.target)) return;
      const key = event.key.toLowerCase();
      if (/^[1-9]$/.test(key)) { const room = rooms[Number(key) - 1]; if (room) { dispatch("reveal-controls"); goTo(`/${room.id}`); } return; }
      const intent = KEY_INTENTS[key];
      if (!intent) { dispatch("reveal-controls"); return; }
      event.preventDefault();
      emitShortcutIntent({ action: `command-center.${intent}`, outcome: "handled", chord: event.key, source: "keyboard" });
      dispatch(intent);
    };
    const onPointerMove = () => dispatch("reveal-controls");
    window.addEventListener("keydown", onKey);
    window.addEventListener("pointermove", onPointerMove, { passive: true });
    return () => { window.removeEventListener("keydown", onKey); window.removeEventListener("pointermove", onPointerMove); };
  }, [dispatch, goTo, path, rooms]);
}
