/**
 * @vrooliComponentSource react-component-library:DrawerShell
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 66af1418-3596-413a-b978-2a70b7bc1511
 * @vrooliComponentAppliedAt 2026-07-14T03:49:23Z
 * @vrooliComponentSourceSha256 e6f7d9ded4ec985c1b121b56fbdc166a5ccf63684bb36429a8f3e8dd0c00b293
 * @vrooliComponentDriftHash e6f7d9ded4ec985c1b121b56fbdc166a5ccf63684bb36429a8f3e8dd0c00b293
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { useEffect } from "react";

/**
 * Invokes `onEscape` when the Escape key is pressed while `active` is true.
 * Centralizing the listener here keeps overlay components free of raw
 * `addEventListener` calls (which fight host-frame spatial navigation).
 */
export function useEscapeKey(active: boolean, onEscape: () => void): void {
  useEffect(() => {
    if (!active) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onEscape();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [active, onEscape]);
}