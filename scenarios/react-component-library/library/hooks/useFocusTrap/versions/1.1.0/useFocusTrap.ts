/**
 * @libraryId react-component-library:useFocusTrap
 * @displayName useFocusTrap
 * @version 1.1.0
 * @tags ["accessibility","focus","overlay"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { useEffect, type RefObject } from "react";

const focusableSelector =
  'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])';

// Listen at the document boundary and resolve the surface ref for each Tab.
// A Portal assigns its child ref only after hydration, which can be later than
// this hook's first effect. Capturing the element during that first effect
// silently leaves the modal untrapped for its entire open lifetime.
export function useFocusTrap(active: boolean, containerRef: RefObject<HTMLElement | null>): void {
  useEffect(() => {
    if (!active || typeof document === "undefined") return;

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Tab") return;
      const panel = containerRef.current;
      const activeElement = document.activeElement;
      if (!panel || !activeElement || !panel.contains(activeElement)) return;

      const focusable = Array.from(panel.querySelectorAll<HTMLElement>(focusableSelector)).filter(
        (element) => !element.hidden && element.getAttribute("aria-hidden") !== "true",
      );
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (!first || !last) return;

      if (event.shiftKey && activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    };

    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [active, containerRef]);
}
