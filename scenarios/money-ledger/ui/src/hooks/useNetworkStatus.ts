/**
 * @vrooliComponentSource react-component-library:useNetworkStatus
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 4d11cc7f-ae55-45d7-b8d3-76906825794f
 * @vrooliComponentAppliedAt 2026-08-17T08:31:47Z
 * @vrooliComponentSourceSha256 37997d937bd2782f47e9b8dd6c318d6e023fd40fd67da51d4dc2a70b00750dc7
 * @vrooliComponentDriftHash 38b8b2d0c13a7918f6af3917a19413ae8dde269c4ba9b80708b17ef003623ea7
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { useSyncExternalStore } from "react";

export function useNetworkStatus() {
  return useSyncExternalStore(
    (onChange) => {
      if (typeof window === "undefined") return () => {};
      window.addEventListener("online", onChange);
      window.addEventListener("offline", onChange);
      return () => {
        window.removeEventListener("online", onChange);
        window.removeEventListener("offline", onChange);
      };
    },
    () => typeof navigator === "undefined" || navigator.onLine,
    () => true,
  );
}
