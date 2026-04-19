import { useCallback, useEffect, useRef, useState } from "react";

export interface UseWakeLockReturn {
  /** True while the Screen Wake Lock sentinel is held. */
  isActive: boolean;
  /** Imperatively request the wake lock. Safe to call multiple times. */
  request: () => Promise<void>;
  /** Imperatively release the wake lock. Safe to call when inactive. */
  release: () => void;
}

/**
 * Imperative Wake Lock hook. Does NOT auto-invoke — callers must call
 * {@link UseWakeLockReturn.request} themselves (usually after a user gesture,
 * which browsers require anyway).
 *
 * Re-acquires automatically when the document returns to the foreground while
 * the sentinel is expected to be active.
 */
export function useWakeLock(): UseWakeLockReturn {
  const [isActive, setIsActive] = useState(false);
  const sentinelRef = useRef<WakeLockSentinel | null>(null);
  const shouldHoldRef = useRef(false);

  const request = useCallback(async () => {
    if (typeof navigator === "undefined") return;
    // `navigator.wakeLock` may be missing on older browsers even though the
    // DOM typing advertises it as always-present. Guard via `in` so we never
    // dereference undefined on legacy platforms.
    if (!("wakeLock" in navigator)) return;
    const wakeLock = navigator.wakeLock;
    try {
      const sentinel = await wakeLock.request("screen");
      sentinelRef.current = sentinel;
      shouldHoldRef.current = true;
      setIsActive(true);
      sentinel.addEventListener("release", () => {
        if (sentinelRef.current === sentinel) {
          sentinelRef.current = null;
          setIsActive(false);
        }
      });
    } catch {
      setIsActive(false);
    }
  }, []);

  const release = useCallback(() => {
    shouldHoldRef.current = false;
    const sentinel = sentinelRef.current;
    if (!sentinel) {
      setIsActive(false);
      return;
    }
    sentinelRef.current = null;
    setIsActive(false);
    void sentinel.release().catch(() => {
      /* already released */
    });
  }, []);

  // Re-acquire when the tab returns to the foreground, if the caller hasn't
  // explicitly released.
  useEffect(() => {
    const handleVisibility = () => {
      if (
        document.visibilityState === "visible" &&
        shouldHoldRef.current &&
        !sentinelRef.current
      ) {
        void request();
      }
    };
    document.addEventListener("visibilitychange", handleVisibility);
    return () => {
      document.removeEventListener("visibilitychange", handleVisibility);
    };
  }, [request]);

  // Ensure the sentinel is released on unmount.
  useEffect(() => {
    return () => {
      const sentinel = sentinelRef.current;
      if (sentinel) {
        sentinelRef.current = null;
        void sentinel.release().catch(() => {
          /* already released */
        });
      }
    };
  }, []);

  return { isActive, request, release };
}
