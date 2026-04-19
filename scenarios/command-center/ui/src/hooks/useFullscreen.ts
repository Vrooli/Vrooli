import { useCallback, useEffect, useState } from "react";

export interface UseFullscreenReturn {
  isFullscreen: boolean;
  enter: () => Promise<void>;
  exit: () => Promise<void>;
  toggle: () => Promise<void>;
}

/**
 * Imperative fullscreen hook. Does NOT auto-invoke — browsers require a user
 * gesture before calling `requestFullscreen`, so callers must wire up a
 * button/keypress handler that calls {@link UseFullscreenReturn.enter}.
 */
export function useFullscreen(): UseFullscreenReturn {
  const [isFullscreen, setIsFullscreen] = useState<boolean>(() => {
    if (typeof document === "undefined") return false;
    return document.fullscreenElement !== null;
  });

  useEffect(() => {
    if (typeof document === "undefined") return;
    const handleChange = () => {
      setIsFullscreen(document.fullscreenElement !== null);
    };
    document.addEventListener("fullscreenchange", handleChange);
    return () => {
      document.removeEventListener("fullscreenchange", handleChange);
    };
  }, []);

  const enter = useCallback(async () => {
    if (typeof document === "undefined") return;
    const root = document.documentElement;
    if (document.fullscreenElement !== null) return;
    if (typeof root.requestFullscreen !== "function") return;
    try {
      await root.requestFullscreen();
    } catch {
      /* fullscreen rejected */
    }
  }, []);

  const exit = useCallback(async () => {
    if (typeof document === "undefined") return;
    if (document.fullscreenElement === null) return;
    if (typeof document.exitFullscreen !== "function") return;
    try {
      await document.exitFullscreen();
    } catch {
      /* not in fullscreen */
    }
  }, []);

  const toggle = useCallback(async () => {
    if (typeof document === "undefined") return;
    if (document.fullscreenElement === null) {
      await enter();
    } else {
      await exit();
    }
  }, [enter, exit]);

  return { isFullscreen, enter, exit, toggle };
}
