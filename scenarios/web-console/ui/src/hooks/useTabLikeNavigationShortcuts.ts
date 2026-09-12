import { useEffect } from "react";
import type { PaneMetadata } from "../stores/useWorkspaceStore";

interface UseTabLikeNavigationShortcutsOptions {
  enabled: boolean;
  panes: PaneMetadata[];
  activePane: string | null;
  onActivatePane: (sessionId: string) => void;
  onClosePane: (sessionId: string) => void;
}

export function useTabLikeNavigationShortcuts({
  enabled,
  panes,
  activePane,
  onActivatePane,
  onClosePane,
}: UseTabLikeNavigationShortcutsOptions): void {
  useEffect(() => {
    if (!enabled) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      const activeIdx = panes.findIndex((pane) => pane.sessionId === activePane);

      if (event.ctrlKey && event.key === "Tab") {
        event.preventDefault();
        if (panes.length === 0) return;
        const direction = event.shiftKey ? -1 : 1;
        const nextIdx = (activeIdx + direction + panes.length) % panes.length;
        const nextPane = panes[nextIdx];
        if (nextPane) onActivatePane(nextPane.sessionId);
        return;
      }

      if (event.ctrlKey && !event.shiftKey && !event.altKey && /^[1-9]$/.test(event.key)) {
        const idx = Number.parseInt(event.key, 10) - 1;
        if (idx < panes.length) {
          event.preventDefault();
          const targetPane = panes[idx];
          if (targetPane) onActivatePane(targetPane.sessionId);
        }
        return;
      }

      if (event.ctrlKey && !event.shiftKey && !event.altKey && event.key === "w") {
        event.preventDefault();
        if (activePane) onClosePane(activePane);
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [activePane, enabled, onActivatePane, onClosePane, panes]);
}
