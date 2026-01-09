import { useEffect, useRef } from 'react';

/**
 * Hook to track the previous value of a variable across renders.
 * Returns undefined on the first render.
 *
 * @param value - The current value to track
 * @returns The previous value from the last render
 */
export function usePrevious<T>(value: T): T | undefined {
  const ref = useRef<T>();

  useEffect(() => {
    ref.current = value;
  });

  return ref.current;
}
