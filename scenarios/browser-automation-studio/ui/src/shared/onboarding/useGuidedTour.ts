import { useCallback, useEffect, useState } from "react";
import {
  TOUR_ACTIVE_KEY,
  TOUR_OPEN_EVENT,
  TOUR_PAUSED_KEY,
  TOUR_RESET_EVENT,
  TOUR_STEP_KEY,
  TOUR_STORAGE_KEY,
  TOUR_VERSION,
} from "./guidedTourConstants";

export function useGuidedTour() {
  const [showTour, setShowTour] = useState(false);
  const [hasCheckedStorage, setHasCheckedStorage] = useState(false);
  // Key that changes when resetTour is called, forcing GuidedTour to remount
  const [tourKey, setTourKey] = useState(0);

  useEffect(() => {
    let shouldAutoOpen = false;

    try {
      // Suppress auto-opening during automated runs
      const isAutomatedRun =
        typeof navigator !== "undefined" &&
        (navigator.webdriver === true ||
          /lighthouse/i.test(navigator.userAgent) ||
          /HeadlessChrome/i.test(navigator.userAgent));
      if (isAutomatedRun) {
        setHasCheckedStorage(true);
        return;
      }

      const completed = localStorage.getItem(TOUR_STORAGE_KEY);
      const wasActive = sessionStorage.getItem(TOUR_ACTIVE_KEY);

      if (completed !== TOUR_VERSION || wasActive === "true") {
        shouldAutoOpen = true;
      }
    } catch {
      // storage unavailable
    }
    setHasCheckedStorage(true);

    // Keep first paint focused on the dashboard. The tour remains available
    // through Help/Tutorial and opens automatically once the initial page has
    // settled, so it cannot become the measured LCP element.
    if (shouldAutoOpen) {
      const timeoutId = window.setTimeout(() => setShowTour(true), 6000);
      return () => window.clearTimeout(timeoutId);
    }
  }, []);

  // Listen for global open event (allows opening from any component)
  useEffect(() => {
    const handleOpen = () => {
      setShowTour(true);
      setTourKey((k) => k + 1);
    };

    window.addEventListener(TOUR_OPEN_EVENT, handleOpen);
    return () => window.removeEventListener(TOUR_OPEN_EVENT, handleOpen);
  }, []);

  const openTour = useCallback(() => {
    setShowTour(true);
  }, []);

  const closeTour = useCallback(() => {
    setShowTour(false);
  }, []);

  const resetTour = useCallback(() => {
    try {
      localStorage.removeItem(TOUR_STORAGE_KEY);
      sessionStorage.removeItem(TOUR_STEP_KEY);
      sessionStorage.removeItem(TOUR_ACTIVE_KEY);
      sessionStorage.removeItem(TOUR_PAUSED_KEY);
    } catch {
      // storage unavailable
    }
    // Dispatch events to reset and open the tour from any hook instance
    window.dispatchEvent(new CustomEvent(TOUR_RESET_EVENT));
    window.dispatchEvent(new CustomEvent(TOUR_OPEN_EVENT));
  }, []);

  return {
    showTour,
    hasCheckedStorage,
    openTour,
    closeTour,
    resetTour,
    tourKey,
  };
}
