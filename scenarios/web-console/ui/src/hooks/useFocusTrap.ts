/**
 * @vrooliComponentSource react-component-library:DrawerShell
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 66af1418-3596-413a-b978-2a70b7bc1511
 * @vrooliComponentAppliedAt 2026-07-14T03:49:23Z
 * @vrooliComponentSourceSha256 f829278e7555040abdfe731c380911da3dce0a2724cd464abddc757c16625a89
 * @vrooliComponentDriftHash f829278e7555040abdfe731c380911da3dce0a2724cd464abddc757c16625a89
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { useEffect, type RefObject } from "react";

const FOCUSABLE_SELECTOR =
  'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])';

/**
 * Traps keyboard focus inside `containerRef` while `active` is true: Tab and
 * Shift+Tab cycle through the container's focusable controls instead of leaking
 * to elements behind an overlay. Centralizing the listener here (a hook) keeps
 * overlay components free of scattered `addEventListener('keydown')` calls,
 * which fight host-frame spatial navigation.
 */
export function useFocusTrap(active: boolean, containerRef: RefObject<HTMLElement | null>): void {
  useEffect(() => {
    if (!active) return;
    const panel = containerRef.current;
    if (!panel) return;
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key !== "Tab") return;
      const focusable = Array.from(panel.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR)).filter(
        (el) => el.offsetParent !== null || el === document.activeElement,
      );
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (!first || !last) return;
      const activeEl = document.activeElement as HTMLElement | null;
      if (e.shiftKey) {
        if (activeEl === first || activeEl === null || !panel.contains(activeEl)) {
          e.preventDefault();
          last.focus();
        }
      } else if (activeEl === last || activeEl === null || !panel.contains(activeEl)) {
        e.preventDefault();
        first.focus();
      }
    };
    panel.addEventListener("keydown", onKeyDown);
    return () => panel.removeEventListener("keydown", onKeyDown);
  }, [active, containerRef]);
}