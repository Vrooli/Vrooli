/** @libraryId react-component-library:StyleSheet */
/** @version 1.0.0 */
/** @vrooliComponentSource react-component-library:StyleSheet */
import { useInsertionEffect } from "react";

const mountedKeys = new Set<string>();

export function useLibraryStyleSheet(key: string, css: string): void {
  useInsertionEffect(() => {
    if (typeof document === "undefined" || !css.trim()) return;
    const escapedKey = key.replace(/[^a-zA-Z0-9_-]/g, "\\$&");
    const selector = `style[data-rcl-sheet="${escapedKey}"]`;
    if (document.querySelector(selector)) {
      mountedKeys.add(key);
      return;
    }
    // A host or test harness may clear the head behind the module-level set.
    // Treat that as a detached sheet and restore it instead of leaving the
    // document without the foundation rules.
    mountedKeys.delete(key);
    const style = document.createElement("style");
    style.dataset.rclSheet = key;
    style.setAttribute("data-rcl-sheet", key);
    style.textContent = css;
    // Library rules must precede consumer styles so an ordinary consumer
    // class wins an equal-specificity cascade tie without render-order tricks.
    document.head.insertBefore(style, document.head.firstChild);
    mountedKeys.add(key);
  }, [key, css]);
}

export interface StyleSheetProps {
  name: string;
  css: string;
}

// StyleSheet is the JSX-safe bridge for assets whose style declaration is
// selected by composition. It emits no DOM node; the hook owns one head node
// per key for the whole page.
export function StyleSheet({ name, css }: StyleSheetProps): null {
  useLibraryStyleSheet(name, css);
  return null;
}
