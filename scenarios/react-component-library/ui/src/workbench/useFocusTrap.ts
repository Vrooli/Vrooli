import { useEffect, useRef } from "react";
import { useGlobalKeydown } from "../hooks/useGlobalKeydown";

const FOCUSABLE = [
  "a[href]",
  "button:not([disabled])",
  "input:not([disabled])",
  "select:not([disabled])",
  "textarea:not([disabled])",
  "[tabindex]:not([tabindex='-1'])",
].join(",");

/**
 * Keeps keyboard focus inside a modal surface, closes it on Escape, and
 * restores the trigger's focus on teardown. The workbench uses this for the
 * command palette, the design-studio phone sheets and Present mode.
 */
export function useFocusTrap<T extends HTMLElement>(
  active: boolean,
  onClose?: () => void,
  initialFocusSelector?: string,
) {
  const ref = useRef<T>(null);
  const closeRef = useRef(onClose);
  closeRef.current = onClose;
  const focusable = () =>
    Array.from(ref.current?.querySelectorAll<HTMLElement>(FOCUSABLE) ?? []).filter((element) => {
      if (element.hidden) return false;
      const style = getComputedStyle(element);
      return style.display !== "none" && style.visibility !== "hidden";
    });

  useGlobalKeydown(
    (event) => {
      const root = ref.current;
      if (event.key === "Escape") {
        event.preventDefault();
        closeRef.current?.();
        return;
      }
      if (event.key !== "Tab") return;
      const items = focusable();
      if (items.length === 0) return;
      const firstItem = items[0];
      const lastItem = items[items.length - 1];
      const current = document.activeElement;
      if (!firstItem || !lastItem) return;
      if (event.shiftKey && (current === firstItem || !root?.contains(current))) {
        event.preventDefault();
        lastItem.focus();
      } else if (!event.shiftKey && (current === lastItem || !root?.contains(current))) {
        event.preventDefault();
        firstItem.focus();
      }
    },
    { target: "document", enabled: active },
  );

  useEffect(() => {
    if (!active) return;
    const previous = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const root = ref.current;
    const preferred = initialFocusSelector
      ? root?.querySelector<HTMLElement>(initialFocusSelector)
      : undefined;
    (preferred && focusable().includes(preferred) ? preferred : focusable()[0])?.focus();
    return () => {
      previous?.focus();
    };
  }, [active, initialFocusSelector]);

  return ref;
}
