/**
 * @libraryId react-component-library:useDelayedPending
 * @displayName useDelayedPending
 * @description A loading-state primitive that suppresses indicators for operations fast enough not to need them, avoiding transient spinner flashes.
 * @version 1.0.1
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:useDelayedPending
 * @vrooliComponentSourceSlot hooks.use-delayed-pending */
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
