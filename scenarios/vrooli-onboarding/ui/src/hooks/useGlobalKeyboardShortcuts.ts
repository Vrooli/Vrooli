import { useEffect, useRef } from "react";
import { emitShortcutIntent } from "@vrooli/iframe-bridge";

/**
 * Registers global Alt+N keyboard shortcuts for view switching.
 * Ignores shortcuts when focus is inside text inputs.
 * Emits shortcut intents via iframe-bridge for host interop.
 */
export function useGlobalKeyboardShortcuts(
  viewIds: readonly string[],
  onSwitchView: (index: number) => void,
) {
  const onSwitchViewRef = useRef(onSwitchView);
  onSwitchViewRef.current = onSwitchView;

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!e.altKey || e.ctrlKey || e.metaKey) return;
      const target = e.target;
      if (
        target instanceof HTMLInputElement ||
        target instanceof HTMLTextAreaElement ||
        (target instanceof HTMLElement && target.isContentEditable)
      ) {
        return;
      }
      const keyIndex = parseInt(e.key, 10) - 1;
      if (keyIndex >= 0 && keyIndex < viewIds.length) {
        e.preventDefault();
        onSwitchViewRef.current(keyIndex);
        emitShortcutIntent({
          action: `view.switch.${viewIds[keyIndex]}`,
          outcome: "handled",
          chord: `Alt+${keyIndex + 1}`,
          source: "keyboard",
        });
      }
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [viewIds]);
}
