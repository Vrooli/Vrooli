import { useEffect } from "react";
import { emitShortcutIntent } from "@vrooli/iframe-bridge";

function isEditableTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  if (target.isContentEditable) return true;
  const tag = target.tagName.toLowerCase();
  return tag === "input" || tag === "textarea" || tag === "select";
}

function shortcutChord(event: KeyboardEvent): string {
  const key = event.key.length === 1 ? event.key.toLowerCase() : event.key;
  const modifiers: string[] = [];
  if (event.metaKey) modifiers.push("meta");
  if (event.ctrlKey) modifiers.push("ctrl");
  if (event.altKey) modifiers.push("alt");
  if (event.shiftKey && key.length > 1) modifiers.push("shift");
  return modifiers.length > 0 ? `${modifiers.join("+")}+${key}` : key;
}

export function useHostShortcutRelay() {
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.defaultPrevented || isEditableTarget(event.target)) return;
      if (!event.metaKey && !event.ctrlKey && !event.altKey) return;
      emitShortcutIntent({
        action: "react-component-library.unhandled-shortcut",
        outcome: "noop",
        chord: shortcutChord(event),
        source: "keyboard",
      });
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);
}
