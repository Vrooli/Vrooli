import { useEffect } from 'react';

/** Locks body scroll while active and restores on cleanup. */
export function useBodyScrollLock(isActive: boolean) {
  useEffect(() => {
    if (!isActive) {
      return;
    }

    const originalOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';

    return () => {
      document.body.style.overflow = originalOverflow;
    };
  }, [isActive]);
}
