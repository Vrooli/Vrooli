import { useCallback, useEffect, useRef } from "react";

/** Records a native-color value only after blur or component unmount. */
export function useDeferredColorCommit(onCommit?: (color: string) => void) {
  const pending = useRef<string | null>(null);
  const flush = useCallback(() => {
    const color = pending.current;
    pending.current = null;
    if (color) onCommit?.(color);
  }, [onCommit]);
  useEffect(() => flush, [flush]);
  return {
    park: (color: string) => {
      pending.current = color;
    },
    flush,
  };
}
