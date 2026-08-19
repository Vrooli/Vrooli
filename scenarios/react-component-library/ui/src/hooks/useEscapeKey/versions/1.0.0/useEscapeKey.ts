/**
 * @vrooliComponentSource hooks.use-escape-key
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption e149b4c5-b7ec-40be-9efa-4943fe378e4a
 * @vrooliComponentAppliedAt 2026-08-11T01:05:38Z
 * @vrooliComponentSourceSha256 bb73b1ee71bc0c70b3d2a35942b7401d43121b20abb8f099ead741fa08d7dbb6
 * @vrooliComponentDriftHash bb73b1ee71bc0c70b3d2a35942b7401d43121b20abb8f099ead741fa08d7dbb6
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { useEffect } from "react";

/**
 * Invokes `onEscape` when the Escape key is pressed while `active` is true.
 * Keeping the listener in a shared hook prevents overlay components from
 * competing with host-frame spatial navigation through ad-hoc listeners.
 */
export function useEscapeKey(active: boolean, onEscape: () => void): void {
  useEffect(() => {
    if (!active) return;
    if (typeof window === "undefined") return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onEscape();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [active, onEscape]);
}
