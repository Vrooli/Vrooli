/**
 * @libraryId react-component-library:useTypeahead
 * @displayName useTypeahead
 * @description A buffered keyboard-search primitive that moves through a collection using normalized localized labels with predictable timeout behavior.
 * @version 1.0.1
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:useTypeahead
 * @vrooliComponentSourceSlot hooks.use-typeahead */
import { useCallback, useRef, type KeyboardEvent as ReactKeyboardEvent } from "react";

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
