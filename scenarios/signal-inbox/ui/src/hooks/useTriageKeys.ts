import { useEffect } from "react";
import { emitShortcutIntent } from "@vrooli/iframe-bridge";

export function useTriageKeys(enabled: boolean, actions: { accept(): void; drop(): void; annotate(): void }) {
  useEffect(() => {
    if (!enabled) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.metaKey || event.ctrlKey || event.altKey || event.target instanceof HTMLInputElement || event.target instanceof HTMLTextAreaElement) return;
      if (event.key === "a") {
        actions.accept();
        emitShortcutIntent({ action: "triage.accept", outcome: "handled", chord: "A", source: "keyboard" });
      }
      if (event.key === "d") {
        actions.drop();
        emitShortcutIntent({ action: "triage.drop", outcome: "handled", chord: "D", source: "keyboard" });
      }
      if (event.key === "n") {
        actions.annotate();
        emitShortcutIntent({ action: "triage.annotate", outcome: "handled", chord: "N", source: "keyboard" });
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [actions, enabled]);
}
