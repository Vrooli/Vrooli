import { useEffect, useRef } from "react";

/**
 * Manages the Screen Wake Lock API to prevent the device screen from
 * dimming or locking. Designed for hands-free voice interaction where
 * the user may not touch the screen for extended periods.
 *
 * - Requests a `"screen"` wake lock when `enabled` is `true`.
 * - Re-acquires automatically when the page becomes visible again
 *   (browsers release wake locks when a tab is hidden).
 * - Silently no-ops on browsers that don't support the API.
 * - Never throws — all errors are caught and logged.
 */
export function useWakeLock(enabled: boolean): void {
  const sentinelRef = useRef<WakeLockSentinel | null>(null);
  const enabledRef = useRef(enabled);
  enabledRef.current = enabled;

  useEffect(() => {
    if (!("wakeLock" in navigator)) return;

    async function requestLock() {
      // Release any existing sentinel before acquiring a new one.
      if (sentinelRef.current) {
        try {
          await sentinelRef.current.release();
        } catch { /* already released */ }
        sentinelRef.current = null;
      }

      if (!enabledRef.current || document.visibilityState !== "visible") return;

      try {
        sentinelRef.current = await navigator.wakeLock.request("screen");
      } catch (err) {
        // Request can be denied due to low battery, power-save mode, or
        // permissions policy. Log but don't disrupt the app.
        console.debug("[useWakeLock] request denied:", err);
      }
    }

    async function releaseLock() {
      if (sentinelRef.current) {
        try {
          await sentinelRef.current.release();
        } catch { /* already released */ }
        sentinelRef.current = null;
      }
    }

    function handleVisibilityChange() {
      if (document.visibilityState === "visible" && enabledRef.current) {
        void requestLock();
      }
    }

    if (enabled) {
      void requestLock();
    } else {
      void releaseLock();
    }

    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => {
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      void releaseLock();
    };
  }, [enabled]);
}
