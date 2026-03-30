/**
 * Graph workspace keyboard shortcuts.
 *
 * 1-3: switch lenses
 * L: cycle layout mode
 * Esc: close detail page or deselect node
 * Ctrl+K: preserved for host switcher
 */

import { useEffect, useCallback } from "react";
import {
  emitShortcutIntent,
  HOST_SHORTCUT_ACTION_OPEN_GLOBAL_SWITCHER,
} from "@vrooli/iframe-bridge";
import { useGraphDataStore } from "../stores/graph-data-store";
import { useGraphUIStore } from "../stores/graph-ui-store";
import { useDetailSelectionStore } from "../../../stores/detail-selection-store";
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
  onDeselectNode: () => void;
  onSettingsToggle: () => void;
  onReturnToAtlas: () => void;
  focusNodeId: string | null;
}

export function useGraphKeyboardShortcuts(handlers: GraphShortcutHandlers): void {
  const cycleLayoutMode = useGraphUIStore((s) => s.cycleLayoutMode);
  const selectedNodeId = useGraphUIStore((s) => s.selectedNodeId);
  const lens = useGraphDataStore((s) => s.lens);
  const detailSelection = useDetailSelectionStore((s) => s.selection);
  const clearDetailSelection = useDetailSelectionStore((s) => s.clearSelection);

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
        const nextLens = LENS_MAP[event.key];
        if (nextLens) {
          // Flow lens requires a focus node to be set.
          if (nextLens === "flow" && !handlers.focusNodeId) {
            return;
          }
          handlers.onLensChange(nextLens);
        }
        return;
      }

      // L — Cycle layout mode
      if (!mod && event.key.toLowerCase() === "l") {
        cycleLayoutMode(lens);
        return;
      }

      // Backspace / Alt+Left — Return to atlas
      if (event.key === "Backspace" || (event.altKey && event.key === "ArrowLeft")) {
        handlers.onReturnToAtlas();
        return;
      }

      // Escape — Close detail page, or deselect node
      if (event.key === "Escape") {
        if (detailSelection) {
          clearDetailSelection();
        } else if (selectedNodeId) {
          handlers.onDeselectNode();
        }
        return;
      }
    },
    [handlers, cycleLayoutMode, lens, selectedNodeId, detailSelection, clearDetailSelection],
  );

  useEffect(() => {
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);
}
