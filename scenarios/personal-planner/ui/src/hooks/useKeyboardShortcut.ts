import { useEffect } from "react";
import { emitShortcutIntent } from "@vrooli/iframe-bridge";

export function useKeyboardShortcut(key: string, onTrigger: () => void) {
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === key.toLowerCase()) {
        event.preventDefault();
        emitShortcutIntent({ action: "personal-planner.global-capture", outcome: "handled", chord: `${event.metaKey ? "Meta" : "Control"}+${key.toUpperCase()}`, source: "keyboard" });
        onTrigger();
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [key, onTrigger]);
}
