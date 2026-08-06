import { useFocusReturn } from "./useFocusReturn";
import type { RefObject } from "react";
export function Default() {
  const ref = useFocusReturn(false);
  return (
    <button ref={ref as RefObject<HTMLButtonElement>} type="button">
      Focus return
    </button>
  );
}
