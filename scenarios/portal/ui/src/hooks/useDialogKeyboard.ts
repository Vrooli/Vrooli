import { emitShortcutIntent } from "@vrooli/iframe-bridge";
import { useEffect, type RefObject } from "react";

type DialogElement = HTMLElement;

/**
 * Keep keyboard handling for a modal dialog in one reusable hook.
 * The hook restores focus to the element that opened the dialog when it closes.
 */
export function useDialogKeyboard(
  dialogRef: RefObject<DialogElement | null>,
  initialFocusRef: RefObject<HTMLElement | null>,
  active: boolean,
  onEscape: () => void,
) {
  useEffect(() => {
    if (!active) return;

    const previous = document.activeElement;
    initialFocusRef.current?.focus();
    const dialog = dialogRef.current;
    const restoreFocus = () => {
      if (previous instanceof HTMLElement && previous.isConnected) previous.focus();
    };

    if (!dialog) return restoreFocus;

    const keyboard = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        emitShortcutIntent({
          action: "portal.dialog.quit",
          outcome: "handled",
          chord: "Escape",
          source: "keyboard",
        });
        onEscape();
        return;
      }
      if (event.key !== "Tab") return;

      const buttons = dialog.querySelectorAll<HTMLButtonElement>("button:not(:disabled)");
      const first = buttons[0];
      const last = buttons[buttons.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last?.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first?.focus();
      }
    };

    dialog.addEventListener("keydown", keyboard);
    return () => {
      dialog.removeEventListener("keydown", keyboard);
      restoreFocus();
    };
  }, [active, dialogRef, initialFocusRef, onEscape]);
}
