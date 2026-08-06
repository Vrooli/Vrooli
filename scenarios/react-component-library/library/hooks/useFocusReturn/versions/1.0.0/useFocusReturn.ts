/** @vrooliComponentSource hooks.use-focus-return */
import { useEffect, useRef } from "react";

export function useFocusReturn(active: boolean) {
  const ref = useRef<HTMLElement | null>(null);
  useEffect(() => {
    if (active) ref.current = document.activeElement as HTMLElement;
    else if (ref.current) ref.current.focus();
  }, [active]);
  return ref;
}
