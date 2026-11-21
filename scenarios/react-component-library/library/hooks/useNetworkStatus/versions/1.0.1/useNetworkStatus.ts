/**
 * @libraryId react-component-library:useNetworkStatus
 * @displayName useNetworkStatus
 * @description A best-effort connectivity primitive combining browser signals with optional application health checks, rather than treating the navigator online flag as authoritative.
 * @version 1.0.1
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:useNetworkStatus
 * @vrooliComponentSourceSlot hooks.use-network-status */
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
