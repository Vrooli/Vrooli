import { useState, useCallback, useEffect, useRef } from "react";

/**
 * Manages a two-step confirm flow: first click sets confirming=true,
 * second click (confirm) triggers the action. Auto-resets after `timeout` ms.
 */
export function useConfirmAction(onConfirm: () => void, timeout = 5000) {
  const [confirming, setConfirming] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const clearTimer = useCallback(() => {
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  useEffect(() => () => clearTimer(), [clearTimer]);

  const requestConfirm = useCallback(() => {
    setConfirming(true);
    clearTimer();
    timerRef.current = setTimeout(() => setConfirming(false), timeout);
  }, [timeout, clearTimer]);

  const confirm = useCallback(() => {
    clearTimer();
    setConfirming(false);
    onConfirm();
  }, [onConfirm, clearTimer]);

  const cancel = useCallback(() => {
    clearTimer();
    setConfirming(false);
  }, [clearTimer]);

  return { confirming, requestConfirm, confirm, cancel };
}
