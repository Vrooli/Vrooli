import { useEffect, useCallback, useState } from 'react';
import type { CSSProperties, MutableRefObject, Dispatch, SetStateAction } from 'react';

interface UseMenuPositioningOptions {
  isOpen: boolean;
  buttonRef: MutableRefObject<HTMLButtonElement | null>; // Used for reference tracking, not accessed directly
  popoverRef: MutableRefObject<HTMLDivElement | null>;
  updateAnchor: () => DOMRect | null;
  computeMenuStyle: (anchorRect: DOMRect, popover: HTMLDivElement | null) => CSSProperties | undefined;
}

interface UseMenuPositioningReturn {
  anchorRect: DOMRect | null;
  menuStyle: CSSProperties | undefined;
  setAnchorRect: Dispatch<SetStateAction<DOMRect | null>>;
  setMenuStyle: Dispatch<SetStateAction<CSSProperties | undefined>>;
}

/**
 * Custom hook to handle menu positioning logic for dropdown menus.
 * Consolidates anchor positioning, menu styling, and resize/scroll handling.
 */
export const useMenuPositioning = ({
  isOpen,
  buttonRef: _buttonRef, // Passed for caller convenience but not used in hook
  popoverRef,
  updateAnchor,
  computeMenuStyle,
}: UseMenuPositioningOptions): UseMenuPositioningReturn => {
  const [anchorRect, setAnchorRect] = useState<DOMRect | null>(null);
  const [menuStyle, setMenuStyle] = useState<CSSProperties | undefined>(undefined);

  const scheduleMenuStyleUpdate = useCallback((
    rect: DOMRect | null,
    popover: MutableRefObject<HTMLDivElement | null>,
  ) => {
    if (!rect) {
      setMenuStyle(undefined);
      return;
    }

    const apply = (popoverElement: HTMLDivElement | null) => {
      setMenuStyle(computeMenuStyle(rect, popoverElement));
    };

    apply(popover.current);

    if (!popover.current && typeof window !== 'undefined') {
      requestAnimationFrame(() => {
        apply(popover.current);
      });
    }
  }, [computeMenuStyle]);

  useEffect(() => {
    if (!isOpen) {
      setMenuStyle(undefined);
      return;
    }

    const rect = updateAnchor();
    scheduleMenuStyleUpdate(rect ?? anchorRect, popoverRef);

    const handleResizeOrScroll = () => {
      const nextRect = updateAnchor();
      scheduleMenuStyleUpdate(nextRect ?? anchorRect, popoverRef);
    };

    window.addEventListener('resize', handleResizeOrScroll);
    window.addEventListener('scroll', handleResizeOrScroll, true);

    return () => {
      window.removeEventListener('resize', handleResizeOrScroll);
      window.removeEventListener('scroll', handleResizeOrScroll, true);
    };
  }, [anchorRect, isOpen, scheduleMenuStyleUpdate, updateAnchor, popoverRef]);

  return {
    anchorRect,
    menuStyle,
    setAnchorRect,
    setMenuStyle,
  };
};
