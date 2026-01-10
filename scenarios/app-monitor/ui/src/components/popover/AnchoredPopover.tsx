import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import type { CSSProperties, ReactNode, RefObject } from 'react';
import { PREVIEW_UI } from '../views/previewConstants';

export type PopoverPlacement = 'bottom-end' | 'top-end';

export type PopoverViewport = {
  width: number;
  height: number;
};

export type PopoverSize = {
  width: number;
  height: number;
};

export type PopoverPositioningOptions = {
  placement?: PopoverPlacement;
  offset?: number;
  margin?: number;
};

export type PopoverStyleResult = {
  style: CSSProperties;
  placement: PopoverPlacement;
};

const resolvePlacement = (
  preferred: PopoverPlacement,
  anchorRect: DOMRect,
  popoverSize: PopoverSize,
  viewport: PopoverViewport,
  offset: number,
  margin: number,
): PopoverPlacement => {
  const hasRoomBelow = anchorRect.bottom + offset + popoverSize.height + margin <= viewport.height;
  const hasRoomAbove = anchorRect.top - offset - popoverSize.height - margin >= 0;
  const anchorCenterY = anchorRect.top + anchorRect.height / 2;

  if (preferred === 'bottom-end') {
    if (hasRoomBelow) {
      return 'bottom-end';
    }
    if (hasRoomAbove) {
      return 'top-end';
    }
  }

  if (preferred === 'top-end') {
    if (hasRoomAbove) {
      return 'top-end';
    }
    if (hasRoomBelow) {
      return 'bottom-end';
    }
  }

  return anchorCenterY < viewport.height / 2 ? 'bottom-end' : 'top-end';
};

export const computeAnchoredPopoverStyle = (
  anchorRect: DOMRect,
  popoverSize: PopoverSize,
  viewport: PopoverViewport,
  options: PopoverPositioningOptions = {},
): PopoverStyleResult => {
  const margin = options.margin ?? PREVIEW_UI.FLOATING_MARGIN;
  const offset = options.offset ?? PREVIEW_UI.MENU_OFFSET;
  const preferred = options.placement ?? 'bottom-end';

  const placement = resolvePlacement(
    preferred,
    anchorRect,
    popoverSize,
    viewport,
    offset,
    margin,
  );

  let top = placement === 'bottom-end'
    ? anchorRect.bottom + offset
    : anchorRect.top - offset - popoverSize.height;

  const maxTop = viewport.height - margin - popoverSize.height;
  const minTop = margin;
  top = Math.min(Math.max(top, minTop), maxTop);

  let left = anchorRect.right;
  const maxLeft = viewport.width - margin;
  const minLeft = popoverSize.width > 0 ? popoverSize.width + margin : margin;
  left = Math.min(Math.max(left, minLeft), maxLeft);

  return {
    placement,
    style: {
      top: `${Math.round(top)}px`,
      left: `${Math.round(left)}px`,
      transform: 'translateX(-100%)',
      transformOrigin: placement === 'bottom-end' ? 'top right' : 'bottom right',
    },
  };
};

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

export type AnchoredPopoverProps = {
  isOpen: boolean;
  portalHost: HTMLElement | null;
  popoverRef: RefObject<HTMLDivElement>;
  style?: CSSProperties;
  placement?: PopoverPlacement;
  className?: string;
  role?: string;
  children: ReactNode;
};

export const AnchoredPopover = ({
  isOpen,
  portalHost,
  popoverRef,
  style,
  placement,
  className,
  role,
  children,
}: AnchoredPopoverProps) => {
  if (!portalHost || !isOpen || !style) {
    return null;
  }

  return createPortal(
    <div
      className={className}
      role={role}
      ref={popoverRef}
      style={style}
      data-placement={placement}
    >
      {children}
    </div>,
    portalHost,
  );
};
