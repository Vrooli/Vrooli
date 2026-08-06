import { useEffect, type RefObject } from "react";

const focusableSelector =
  'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])';

export function useFocusTrap(
  active: boolean,
  containerRef: RefObject<HTMLElement | null>,
): void {
  useEffect(() => {
    if (!active) return;
    const panel = containerRef.current;
    if (!panel) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Tab") return;
      const focusable = Array.from(
        panel.querySelectorAll<HTMLElement>(focusableSelector),
      ).filter(
        (element) =>
          element.offsetParent !== null || element === document.activeElement,
      );
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      const current = document.activeElement as HTMLElement | null;
      if (!first || !last) return;
      if (
        event.shiftKey &&
        (current === first || current === null || !panel.contains(current))
      ) {
        event.preventDefault();
        last.focus();
      }
      if (
        !event.shiftKey &&
        (current === last || current === null || !panel.contains(current))
      ) {
        event.preventDefault();
        first.focus();
      }
    };
    panel.addEventListener("keydown", onKeyDown);
    return () => panel.removeEventListener("keydown", onKeyDown);
  }, [active, containerRef]);
}
