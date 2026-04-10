import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { CSSProperties, RefObject } from 'react';
import type { PopoverPlacement, PopoverPositioningOptions } from './anchoredPopoverUtils';
import { computeAnchoredPopoverStyle } from './anchoredPopoverUtils';

export type UseAnchoredPopoverOptions = PopoverPositioningOptions & {
  isOpen: boolean;
  anchorRef: RefObject<HTMLElement>;
  popoverRef: RefObject<HTMLDivElement>;
};

export type UseAnchoredPopoverReturn = {
  style: CSSProperties | undefined;
  placement: PopoverPlacement;
  updatePosition: () => void;
};

export const useAnchoredPopover = ({
  isOpen,
  anchorRef,
  popoverRef,
  placement,
  offset,
  margin,
}: UseAnchoredPopoverOptions): UseAnchoredPopoverReturn => {
  const [style, setStyle] = useState<CSSProperties | undefined>(undefined);
  const [resolvedPlacement, setResolvedPlacement] = useState<PopoverPlacement>('bottom-end');
  const followUpRef = useRef<number | null>(null);

  const options = useMemo(() => ({ placement, offset, margin }), [placement, offset, margin]);

  const updatePosition = useCallback(() => {
    if (typeof window === 'undefined') {
      return { measured: false, width: 0, height: 0 };
    }

    const anchor = anchorRef.current;
    if (!anchor) {
      return { measured: false, width: 0, height: 0 };
    }

    const anchorRect = anchor.getBoundingClientRect();
    const popoverRect = popoverRef.current?.getBoundingClientRect();
    const width = popoverRect?.width ?? 0;
    const height = popoverRect?.height ?? 0;
    const popoverSize = { width, height };
    const viewport = {
      width: window.innerWidth,
      height: window.innerHeight,
    };

    const result = computeAnchoredPopoverStyle(anchorRect, popoverSize, viewport, options);
    setStyle(result.style);
    setResolvedPlacement(result.placement);
    return { measured: Boolean(popoverRect), width, height };
  }, [anchorRef, options, popoverRef]);

  useEffect(() => {
    if (!isOpen) {
      setStyle(undefined);
      return;
    }

    const schedule = typeof window !== 'undefined' && typeof window.requestAnimationFrame === 'function'
      ? window.requestAnimationFrame
      : (callback: FrameRequestCallback) => window.setTimeout(callback, 0);
    const cancel = typeof window !== 'undefined' && typeof window.cancelAnimationFrame === 'function'
      ? window.cancelAnimationFrame
      : window.clearTimeout;

    const maxAttempts = 6;
    const scheduleFollowUp = (attempt: number) => {
      followUpRef.current = schedule(() => {
        const next = updatePosition();
        const needsFollowUp = !next.measured || next.width === 0 || next.height === 0;
        if (needsFollowUp && attempt < maxAttempts) {
          scheduleFollowUp(attempt + 1);
        } else {
          followUpRef.current = null;
        }
      });
    };

    const initial = updatePosition();
    if (!initial.measured || initial.width === 0 || initial.height === 0) {
      scheduleFollowUp(0);
    }

    const handleResizeOrScroll = () => updatePosition();
    window.addEventListener('resize', handleResizeOrScroll);
    window.addEventListener('scroll', handleResizeOrScroll, true);

    return () => {
      if (followUpRef.current !== null) {
        cancel(followUpRef.current);
        followUpRef.current = null;
      }
      window.removeEventListener('resize', handleResizeOrScroll);
      window.removeEventListener('scroll', handleResizeOrScroll, true);
    };
  }, [isOpen, updatePosition]);

  return {
    style,
    placement: resolvedPlacement,
    updatePosition,
  };
};
