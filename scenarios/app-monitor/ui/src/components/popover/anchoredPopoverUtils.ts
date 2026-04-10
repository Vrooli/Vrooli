import type { CSSProperties } from 'react';
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
