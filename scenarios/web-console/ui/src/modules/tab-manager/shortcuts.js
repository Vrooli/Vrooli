import { shortcutButtons, state } from "../state.js";
import { closeDrawer, isDrawerFloating } from "../drawer.js";
import { tabCallbacks } from "./callbacks.js";

export function initializeShortcutButtons() {
  shortcutButtons.forEach((button) => {
    button.addEventListener("click", () => {
      const command = (button.dataset.command || "").trim();
      const id = button.dataset.shortcutId || "unknown";
      tabCallbacks.onShortcut?.({ command, id });
      if (command && state.drawer && state.drawer.open && isDrawerFloating()) {
        closeDrawer();
      }
    });
  });
}
