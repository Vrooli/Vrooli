import { useEffect, useRef } from 'react';

export function useBodyScrollLock(locked: boolean) {
  const previousOverflowRef = useRef<string | null>(null);

  useEffect(() => {
    if (typeof document === 'undefined') {
      return undefined;
    }

    const body = document.body;

    if (locked) {
      if (previousOverflowRef.current === null) {
        previousOverflowRef.current = body.style.overflow;
      }
      body.style.overflow = 'hidden';
    } else if (previousOverflowRef.current !== null) {
      body.style.overflow = previousOverflowRef.current;
      previousOverflowRef.current = null;
    }

    return () => {
      if (previousOverflowRef.current !== null) {
        body.style.overflow = previousOverflowRef.current;
        previousOverflowRef.current = null;
      } else if (locked) {
        body.style.overflow = '';
      }
    };
  }, [locked]);
}
