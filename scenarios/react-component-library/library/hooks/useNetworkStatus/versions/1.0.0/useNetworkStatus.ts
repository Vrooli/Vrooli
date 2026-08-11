/** @vrooliComponentSource hooks.use-network-status */
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
