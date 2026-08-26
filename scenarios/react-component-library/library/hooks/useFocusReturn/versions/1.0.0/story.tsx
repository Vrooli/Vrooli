import { useFocusReturn } from "./useFocusReturn";
import type { RefObject } from "react";
export function Default() {
  const ref = useFocusReturn(false);
  return (
    <button data-testid="hooks.use-focus-return" ref={ref as RefObject<HTMLButtonElement>} type="button">
      Focus return
    </button>
  );
}
