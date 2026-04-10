import { useEffect, useCallback } from "react";
import { emitShortcutIntent, type BridgeShortcutIntent } from "@vrooli/iframe-bridge";

// ─────────────────────────────────────────────────────────────────────────────
// Keyboard Shortcut Hook
// [REQ:P0-001] Reference Scenario Registry - Keyboard navigation support
// ─────────────────────────────────────────────────────────────────────────────
//
// Single centralized keyboard shortcut manager. All app-level shortcuts should
// be registered through this hook rather than scattered across components.
//
// ╔══════════════════════════════════════════════════════════════════════════╗
// ║  INTEROP-CRITICAL: Iframe focus handling                                 ║
// ║                                                                          ║
// ║  When running in an iframe, keyboard events may need to be relayed to    ║
// ║  the parent. This hook follows the local-first/relay-on-noop pattern:    ║
// ║  - Handle shortcuts locally when possible                                ║
// ║  - Relay to parent via iframe-bridge for parent-scope actions            ║
// ║                                                                          ║
// ║  Uses emitShortcutIntent from @vrooli/iframe-bridge to notify parent     ║
// ║  of handled shortcuts. This allows the host to know what shortcuts       ║
// ║  the child scenario handles and avoid conflicts.                         ║
// ╚══════════════════════════════════════════════════════════════════════════╝
// ─────────────────────────────────────────────────────────────────────────────

export interface ShortcutConfig {
  /** Key to listen for (e.g., "r", "Escape", "/") */
  key: string;
  /** Modifier keys required */
  modifiers?: {
    ctrl?: boolean;
    alt?: boolean;
    shift?: boolean;
    meta?: boolean;
  };
  /** Handler function */
  handler: (event: KeyboardEvent) => void;
  /** Prevent default browser behavior */
  preventDefault?: boolean;
  /** Description for help displays */
  description?: string;
  /** Action name for iframe-bridge shortcut intent (defaults to description) */
  action?: string;
}

export interface UseKeyboardShortcutOptions {
  /** Disable all shortcuts temporarily */
  disabled?: boolean;
  /** Don't trigger when focus is in form elements */
  ignoreFormElements?: boolean;
}

const FORM_ELEMENTS = ["INPUT", "TEXTAREA", "SELECT"];

/**
 * Hook for registering keyboard shortcuts.
 *
 * @example
 * ```tsx
 * useKeyboardShortcut([
 *   { key: "r", handler: handleRefresh, description: "Refresh data" },
 *   { key: "Escape", handler: handleClose, description: "Close modal" }
 * ]);
 * ```
 */
export function useKeyboardShortcut(
  shortcuts: ShortcutConfig[],
  options: UseKeyboardShortcutOptions = {}
) {
  const { disabled = false, ignoreFormElements = true } = options;

  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (disabled) return;

      // Skip if focused on form element and option is set
      if (ignoreFormElements) {
        const target = event.target as HTMLElement;
        if (target && FORM_ELEMENTS.includes(target.tagName)) {
          return;
        }
      }

      for (const shortcut of shortcuts) {
        const { key, modifiers = {}, handler, preventDefault = true, action, description } = shortcut;

        // Check if key matches
        if (event.key.toLowerCase() !== key.toLowerCase()) continue;

        // Check modifiers
        const ctrlMatch = !!modifiers.ctrl === (event.ctrlKey || event.metaKey);
        const altMatch = !!modifiers.alt === event.altKey;
        const shiftMatch = !!modifiers.shift === event.shiftKey;

        if (ctrlMatch && altMatch && shiftMatch) {
          if (preventDefault) {
            event.preventDefault();
          }

          // Build chord string for iframe-bridge
          const chordParts: string[] = [];
          if (modifiers.ctrl) chordParts.push("Ctrl");
          if (modifiers.alt) chordParts.push("Alt");
          if (modifiers.shift) chordParts.push("Shift");
          chordParts.push(key.toUpperCase());
          const chord = chordParts.join("+");

          // Execute local handler
          handler(event);

          // Emit shortcut intent to parent iframe (if embedded)
          // This notifies the host about handled shortcuts to avoid conflicts
          const intent: BridgeShortcutIntent = {
            action: action ?? description ?? key,
            outcome: "handled",
            chord,
            source: "keyboard"
          };
          emitShortcutIntent(intent);

          return;
        }
      }
    },
    [shortcuts, disabled, ignoreFormElements]
  );

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [handleKeyDown]);
}

/**
 * Hook for common navigation shortcuts.
 * Provides standard shortcuts for refresh, back, etc.
 */
export function useNavigationShortcuts(handlers: {
  onRefresh?: () => void;
  onBack?: () => void;
  onHome?: () => void;
}) {
  const shortcuts: ShortcutConfig[] = [];

  if (handlers.onRefresh) {
    shortcuts.push({
      key: "r",
      handler: handlers.onRefresh,
      description: "Refresh data"
    });
  }

  if (handlers.onBack) {
    shortcuts.push({
      key: "Escape",
      handler: handlers.onBack,
      description: "Go back"
    });
  }

  if (handlers.onHome) {
    shortcuts.push({
      key: "h",
      handler: handlers.onHome,
      description: "Go to dashboard"
    });
  }

  useKeyboardShortcut(shortcuts);
}
