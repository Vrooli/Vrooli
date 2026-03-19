import { useState, useEffect } from "react";

/** Debounces a string value by the given delay (ms). */
export function useDebouncedValue(value: string, delay: number) {
  const [debounced, setDebounced] = useState(value);
  const [isPending, setIsPending] = useState(false);

  useEffect(() => {
    if (value === debounced) {
      setIsPending(false);
      return;
    }
    setIsPending(true);
    const id = setTimeout(() => {
      setDebounced(value);
      setIsPending(false);
    }, delay);
    return () => clearTimeout(id);
  }, [value, delay, debounced]);

  return { debounced, isPending };
}
