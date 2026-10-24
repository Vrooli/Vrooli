/**
 * @libraryId react-component-library:useFocusReturn
 * @displayName useFocusReturn
 * @description A focus-restoration primitive returning focus to the originating trigger, or to a deterministic fallback when that trigger no longer exists.
 * @version 1.1.1
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:useFocusReturn
 * @vrooliComponentSourceSlot hooks.use-focus-return */
import { useEffect, useRef, type RefObject } from "react";

export function useFocusReturn(active: boolean, returnFocusRef?: RefObject<HTMLElement | null>) {
  const ref = useRef<HTMLElement | null>(null);
  useEffect(() => {
    if (active) ref.current = document.activeElement as HTMLElement;
    else (returnFocusRef?.current ?? ref.current)?.focus();
  }, [active, returnFocusRef]);
  return ref;
}
