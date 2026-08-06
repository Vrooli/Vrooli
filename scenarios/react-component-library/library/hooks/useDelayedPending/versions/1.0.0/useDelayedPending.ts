/** @vrooliComponentSource hooks.use-delayed-pending */
import { useEffect, useState } from "react";

export function useDelayedPending(pending: boolean, delay = 150) {
  const [visible, setVisible] = useState(false);
  useEffect(() => {
    if (!pending) {
      setVisible(false);
      return;
    }
    const timer = window.setTimeout(() => setVisible(true), delay);
    return () => window.clearTimeout(timer);
  }, [delay, pending]);
  return visible;
}
