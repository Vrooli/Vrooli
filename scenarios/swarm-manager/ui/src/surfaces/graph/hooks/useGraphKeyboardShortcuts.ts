/**
 * Graph workspace keyboard shortcuts.
 *
 * 1-3: switch lenses
 * L: cycle layout mode
 * I: toggle inspector
 * Esc: deselect node / close inspector
 * Ctrl+K: preserved for host switcher
 */

import { useEffect, useCallback } from "react";
import {
  emitShortcutIntent,
  HOST_SHORTCUT_ACTION_OPEN_GLOBAL_SWITCHER,
} from "@vrooli/iframe-bridge";
import { useGraphUIStore } from "../stores/graph-ui-store";
import type { GraphLens } from "../stores/graph-data-store";

function isInputElement(el: HTMLElement): boolean {
  return (
    el.tagName === "INPUT" ||
    el.tagName === "TEXTAREA" ||
    el.isContentEditable ||
    el.closest(".monaco-editor") !== null
  );
}

const LENS_MAP: Record<string, GraphLens> = {
  "1": "topology",
  "2": "flow",
  "3": "operations",
};

interface GraphShortcutHandlers {
  onLensChange: (lens: GraphLens) => void;
  onInspectorClose: () => void;
  onSettingsToggle: () => void;
}

export function useGraphKeyboardShortcuts(handlers: GraphShortcutHandlers): void {
  const cycleLayoutMode = useGraphUIStore((s) => s.cycleLayoutMode);
  const toggleInspector = useGraphUIStore((s) => s.toggleInspector);
  const selectedNodeId = useGraphUIStore((s) => s.selectedNodeId);

  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      const target = event.target as HTMLElement;
      if (isInputElement(target)) return;

      const mod = event.metaKey || event.ctrlKey;

      // Ctrl/Cmd+K — Global search / switcher
      if (mod && event.key === "k") {
        event.preventDefault();
        emitShortcutIntent({
          action: HOST_SHORTCUT_ACTION_OPEN_GLOBAL_SWITCHER,
          outcome: "noop",
          chord: "mod+k",
          source: "keyboard",
        });
        return;
      }

      // Number keys 1-3 — Lens switching
      if (!mod && event.key in LENS_MAP) {
        handlers.onLensChange(LENS_MAP[event.key]!);
        return;
      }

      // L — Cycle layout mode
      if (!mod && event.key.toLowerCase() === "l") {
        cycleLayoutMode();
        return;
      }

      // I — Toggle inspector
      if (!mod && event.key.toLowerCase() === "i") {
        toggleInspector();
        return;
      }

      // Escape — Deselect node / close inspector
      if (event.key === "Escape") {
        if (selectedNodeId) {
          handlers.onInspectorClose();
        }
        return;
      }
    },
    [handlers, cycleLayoutMode, toggleInspector, selectedNodeId],
  );

  useEffect(() => {
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);
}
