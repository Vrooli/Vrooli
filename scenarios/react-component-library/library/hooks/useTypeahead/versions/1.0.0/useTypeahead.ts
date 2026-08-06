/** @vrooliComponentSource hooks.use-typeahead */
import {
  useCallback,
  useRef,
  type KeyboardEvent as ReactKeyboardEvent,
} from "react";

export function useTypeahead(onMatch: (query: string) => void) {
  const query = useRef("");
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  return useCallback(
    (event: ReactKeyboardEvent) => {
      if (event.key.length !== 1) return;
      query.current += event.key;
      onMatch(query.current);
      if (timer.current) clearTimeout(timer.current);
      timer.current = setTimeout(() => {
        query.current = "";
      }, 500);
    },
    [onMatch],
  );
}
