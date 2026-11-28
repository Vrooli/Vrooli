import { useEffect } from "react";
import { emitShortcutIntent } from "@vrooli/iframe-bridge";
import { selectors } from "../consts/selectors";

export function useManifestKeyboardShortcuts(
  undo: () => void,
  redo: () => void,
) {
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      const target = event.target instanceof HTMLElement ? event.target : null;
      const isEditorTextarea = target?.getAttribute("data-testid") === selectors.manifest.input;
      const isInputElement =
        target?.tagName === "INPUT" ||
        (target?.tagName === "TEXTAREA" && !isEditorTextarea);

      if (isInputElement) return;

      if ((event.ctrlKey || event.metaKey) && event.key === "z") {
        event.preventDefault();
        if (event.shiftKey) {
          redo();
          emitShortcutIntent({
            action: "scenario-to-cloud.manifest.redo",
            outcome: "handled",
            chord: event.metaKey ? "Meta+Shift+Z" : "Ctrl+Shift+Z",
            source: "keyboard",
          });
        } else {
          undo();
          emitShortcutIntent({
            action: "scenario-to-cloud.manifest.undo",
            outcome: "handled",
            chord: event.metaKey ? "Meta+Z" : "Ctrl+Z",
            source: "keyboard",
          });
        }
      }
      if ((event.ctrlKey || event.metaKey) && event.key === "y") {
        event.preventDefault();
        redo();
        emitShortcutIntent({
          action: "scenario-to-cloud.manifest.redo",
          outcome: "handled",
          chord: event.metaKey ? "Meta+Y" : "Ctrl+Y",
          source: "keyboard",
        });
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [redo, undo]);
}
