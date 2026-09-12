/**
 * Graph workspace keyboard shortcuts.
 *
 * 1-3: switch operator surfaces
 * L: cycle layout mode
 * Esc: deselect graph node
 * Ctrl+K: preserved for host switcher
 */

import { useCallback } from "react";
import {
  emitShortcutIntent,
  HOST_SHORTCUT_ACTION_OPEN_GLOBAL_SWITCHER,
} from "@vrooli/iframe-bridge";
import { useGraphDataStore } from "../stores/graph-data-store";
import { useGraphUIStore } from "../stores/graph-ui-store";
import { useGlobalKeyDown } from "../../../hooks/useGlobalKeyDown";
import type { AppGraphLens } from "../../../app/routes/route-paths";

function isInputElement(el: HTMLElement): boolean {
  return (
    el.tagName === "INPUT" ||
    el.tagName === "TEXTAREA" ||
    el.isContentEditable ||
    el.closest(".monaco-editor") !== null
  );
}

const LENS_MAP: Record<string, AppGraphLens> = {
  "1": "plan",
  "2": "graph",
  "3": "stats",
};

interface GraphShortcutHandlers {
  onLensChange: (lens: AppGraphLens) => void;
  onDeselectNode: () => void;
  onSettingsToggle: () => void;
  onReturnToAtlas: () => void;
  onToggleCommandPost?: () => void;
  focusNodeId: string | null;
}

export function useGraphKeyboardShortcuts(handlers: GraphShortcutHandlers): void {
  const cycleLayoutMode = useGraphUIStore((s) => s.cycleLayoutMode);
  const selectedNodeId = useGraphUIStore((s) => s.selectedNodeId);
  const lens = useGraphDataStore((s) => s.lens);

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

      // Number keys 1-3 — surface switching
      if (!mod && event.key in LENS_MAP) {
        const nextLens = LENS_MAP[event.key];
        if (nextLens) {
          handlers.onLensChange(nextLens);
        }
        return;
      }

      // P — Toggle Command Post
      if (!mod && event.key.toLowerCase() === "p") {
        handlers.onToggleCommandPost?.();
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

      // Escape — Deselect graph node.
      if (event.key === "Escape") {
        if (selectedNodeId) {
          handlers.onDeselectNode();
        }
        return;
      }
    },
    [handlers, cycleLayoutMode, lens, selectedNodeId],
  );

  useGlobalKeyDown(handleKeyDown);
}
