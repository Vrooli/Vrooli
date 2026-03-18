export type PopoverPlacement = "top" | "bottom";

interface RectLike {
  top: number;
  left: number;
  width: number;
  height: number;
  bottom: number;
  right: number;
}

interface PopoverPositionInput {
  anchorRect: RectLike;
  viewportWidth: number;
  viewportHeight: number;
  preferredWidth: number;
  panelHeight: number;
  viewportPadding?: number;
  anchorGap?: number;
  minimumHeight?: number;
}

export interface PopoverPosition {
  top: number;
  left: number;
  width: number;
  maxHeight: number;
  placement: PopoverPlacement;
}

export function getPopoverPosition({
  anchorRect,
  viewportWidth,
  viewportHeight,
  preferredWidth,
  panelHeight,
  viewportPadding = 8,
  anchorGap = 8,
  minimumHeight = 180,
}: PopoverPositionInput): PopoverPosition {
  const width = Math.min(preferredWidth, Math.max(220, viewportWidth - (viewportPadding * 2)));
  const availableBelow = viewportHeight - anchorRect.bottom - anchorGap - viewportPadding;
  const availableAbove = anchorRect.top - anchorGap - viewportPadding;
  const placeAbove = availableBelow < minimumHeight && availableAbove > availableBelow;
  const placement: PopoverPlacement = placeAbove ? "top" : "bottom";
  const availableHeight = Math.max(
    minimumHeight,
    placement === "top" ? availableAbove : availableBelow
  );
  const height = Math.min(panelHeight, availableHeight);
  const unclampedTop = placement === "top"
    ? anchorRect.top - anchorGap - height
    : anchorRect.bottom + anchorGap;
  const unclampedLeft = anchorRect.right - width;

  return {
    top: clamp(unclampedTop, viewportPadding, viewportHeight - height - viewportPadding),
    left: clamp(unclampedLeft, viewportPadding, viewportWidth - width - viewportPadding),
    width,
    maxHeight: availableHeight,
    placement,
  };
}

function clamp(value: number, min: number, max: number): number {
  if (max < min) return min;
  return Math.min(Math.max(value, min), max);
}
