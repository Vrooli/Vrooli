/**
 * @libraryId react-component-library:useFocusReturn
 * @displayName useFocusReturn
 * @description Production-ready useFocusReturn hook with SSR-safe lifecycle behavior.
 * @version 1.1.0
 * @tags ["runtime","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource hooks.use-focus-return */
import { useEffect, useRef, type RefObject } from "react";

export function useFocusReturn(active: boolean, returnFocusRef?: RefObject<HTMLElement | null>) {
  const ref = useRef<HTMLElement | null>(null);
  useEffect(() => {
    if (active) ref.current = document.activeElement as HTMLElement;
    else (returnFocusRef?.current ?? ref.current)?.focus();
  }, [active, returnFocusRef]);
  return ref;
}
