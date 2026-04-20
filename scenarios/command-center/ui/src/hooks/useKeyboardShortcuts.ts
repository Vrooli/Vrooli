import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { emitShortcutIntent } from "@vrooli/iframe-bridge";

export interface KeyboardShortcut {
  chord: string;
  action: string;
  path?: string;
  handler?: () => void;
}

export const KIOSK_SHORTCUTS: readonly KeyboardShortcut[] = [
  { chord: "1", action: "command-center.nav.mission-control", path: "/mission-control" },
  { chord: "2", action: "command-center.nav.hive", path: "/hive" },
  { chord: "3", action: "command-center.nav.forge", path: "/forge" },
  { chord: "4", action: "command-center.nav.ledger", path: "/ledger" },
  { chord: "5", action: "command-center.nav.broadcast", path: "/broadcast" },
  { chord: "6", action: "command-center.nav.panorama", path: "/panorama" },
] as const;

function isEditingTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  if (target.isContentEditable) return true;
  return target instanceof HTMLInputElement || target instanceof HTMLTextAreaElement;
}

/**
 * Registers keyboard shortcuts for dashboard navigation and relays intent to
 * the host frame via `emitShortcutIntent`. Safe when rendered standalone —
 * emitShortcutIntent no-ops outside an iframe.
 */
export function useKeyboardShortcuts(
  shortcuts: readonly KeyboardShortcut[] = KIOSK_SHORTCUTS,
): void {
  const navigate = useNavigate();

  useEffect(() => {
    const handler = (event: KeyboardEvent) => {
      if (event.metaKey || event.ctrlKey || event.altKey) return;
      if (isEditingTarget(event.target)) return;
      const match = shortcuts.find((s) => s.chord === event.key);
      if (!match) return;
      event.preventDefault();

      if (match.path) {
        navigate(match.path);
      }
      match.handler?.();

      emitShortcutIntent({
        action: match.action,
        outcome: "handled",
        chord: match.chord,
        source: "keyboard",
      });
    };

    window.addEventListener("keydown", handler);
    return () => {
      window.removeEventListener("keydown", handler);
    };
  }, [navigate, shortcuts]);
}
