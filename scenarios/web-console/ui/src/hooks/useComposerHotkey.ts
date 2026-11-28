import { useEffect } from "react";

/**
 * Desktop keyboard shortcut (Ctrl/Cmd+Shift+K) that opens the full-screen
 * composer. Escape (owned by the DrawerShell) closes it, so this only ever
 * opens. Lives in a hook so the keydown listener stays out of components,
 * satisfying the host-frame interop convention.
 */
export function useComposerHotkey(onOpen: () => void): void {
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.shiftKey && (e.key === "K" || e.key === "k")) {
        e.preventDefault();
        onOpen();
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [onOpen]);
}
