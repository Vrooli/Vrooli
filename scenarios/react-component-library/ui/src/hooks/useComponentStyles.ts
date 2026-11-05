import { useInsertionEffect } from "react";

/**
 * Attribute stamped on every stylesheet this module injects. Tests and the
 * `<style dangerouslySetInnerHTML>` guard key off it, so keep it stable.
 */
export const COMPONENT_STYLE_ID_ATTRIBUTE = "data-rcl-style-id";

interface Registration {
  element: HTMLStyleElement;
  /**
   * Number of currently-mounted component instances that asked for this
   * stylesheet. The element is removed only when this reaches zero, so a
   * sheet is never pulled out from under a sibling that still needs it.
   */
  refCount: number;
}

/**
 * Module-level registry keyed by stylesheet id. One entry per unique id, no
 * matter how many components or instances ask for it.
 */
const registry = new Map<string, Registration>();

function hasDOM(): boolean {
  return typeof document !== "undefined";
}

function acquire(id: string, css: string): void {
  if (!hasDOM()) return;

  const existing = registry.get(id);
  if (existing) {
    existing.refCount += 1;
    // Defensive re-attach: a test harness (or a host that wipes `document.head`)
    // can detach the element behind our back. Without this the refcount would
    // keep the registry entry alive while no stylesheet is actually applied.
    if (!existing.element.isConnected) {
      document.head.appendChild(existing.element);
    }
    return;
  }

  const element = document.createElement("style");
  element.setAttribute(COMPONENT_STYLE_ID_ATTRIBUTE, id);
  // `.textContent` on a created element is a plain text write — no HTML parser,
  // no injection surface. This is the safe replacement for the per-instance
  // `<style dangerouslySetInnerHTML>` blocks these components used to render.
  element.textContent = css;
  // Put library rules before the app's own head stylesheets. Consumer classes
  // can therefore win an equal-specificity cascade tie without depending on
  // which component instance rendered first.
  document.head.insertBefore(element, document.head.firstChild);
  registry.set(id, { element, refCount: 1 });
}

function release(id: string): void {
  if (!hasDOM()) return;

  const entry = registry.get(id);
  if (!entry) return;

  entry.refCount -= 1;
  if (entry.refCount > 0) return;

  registry.delete(id);
  entry.element.remove();
}

/**
 * Injects a component stylesheet into `document.head` exactly once per unique
 * `id`, however many components or instances request it.
 *
 * Ordering: sheets land in the order their first instance mounts. Insertion
 * effects run child-before-parent, so a nested component's sheet lands before
 * its ancestor's. Every RCL sheet is namespaced under its own `data-rcl-*`
 * root, so no two sheets that can nest declare the same selector and the
 * cascade result is unchanged.
 *
 * Ids are per component *and version*: two catalog versions whose CSS differs
 * (`TreeView` and `ui/TreeView` both target `[data-rcl-tree]`) must not share
 * an id, or one silently overrides the other. Sharing an id across files is
 * correct only when the CSS is byte-identical — `Pressable`/`ControlBase`
 * copies are, and `useComponentStyles.test.tsx` pins that.
 *
 * SSR/no-DOM safe: insertion effects do not run during server rendering, and
 * every DOM touch is guarded on `document` existing.
 */
export function useComponentStyles(id: string, css: string): void {
  // `useInsertionEffect` is React's designated slot for style injection: it
  // runs before layout effects, so the stylesheet is applied before anything
  // measures the DOM. A plain `useEffect` would let layout reads observe an
  // unstyled frame.
  useInsertionEffect(() => {
    acquire(id, css);
    return () => {
      release(id);
    };
  }, [id, css]);
}
