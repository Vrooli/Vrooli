/** @vrooliComponentSource hooks.use-drag */
import { useCallback, type PointerEvent as ReactPointerEvent } from "react";

export function useDrag(onMove?: (event: globalThis.PointerEvent) => void) {
  const onPointerMove = useCallback(
    (event: ReactPointerEvent) => onMove?.(event.nativeEvent),
    [onMove],
  );
  return { onPointerMove };
}
