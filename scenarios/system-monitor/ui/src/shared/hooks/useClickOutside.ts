import { useEffect } from 'react';
import type { RefObject } from 'react';

/**
 * Calls `onClickOutside` when a mousedown event occurs outside all provided refs.
 * Only active when `enabled` is true (defaults to true).
 */
export const useClickOutside = (
  refs: RefObject<Element | null> | RefObject<Element | null>[],
  onClickOutside: () => void,
  enabled = true
) => {
  useEffect(() => {
    if (!enabled) {
      return;
    }

    const refList = Array.isArray(refs) ? refs : [refs];

    const handleMouseDown = (event: MouseEvent) => {
      const target = event.target as Node;
      const isInside = refList.some(ref => ref.current?.contains(target));
      if (!isInside) {
        onClickOutside();
      }
    };

    document.addEventListener('mousedown', handleMouseDown);
    return () => { document.removeEventListener('mousedown', handleMouseDown); };
  }, [refs, onClickOutside, enabled]);
};
