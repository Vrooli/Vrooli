import { CSSProperties, MutableRefObject, useCallback, useLayoutEffect, useRef, useState } from 'react';

export type PopoverPlacement = 'bottom-start' | 'bottom-end' | 'top-start' | 'top-end';

interface UsePopoverPositionOptions {
  isOpen: boolean;
  /**
   * Placement priority, the hook will pick the first placement that fits within the viewport.
   * Defaults to ['bottom-start', 'top-start', 'bottom-end', 'top-end']
   */
  placementPriority?: PopoverPlacement[];
  /** Offset between the trigger and the floating element, in pixels */
  offset?: number;
  /** Minimum distance from viewport edges, in pixels */
  viewportMargin?: number;
  /** When true, the floating element will match the trigger's width */
  matchReferenceWidth?: boolean;
}

interface PopoverPositionResult {
  floatingStyles: CSSProperties;
  updatePosition: () => void;
}

const DEFAULT_PLACEMENTS: PopoverPlacement[] = [
  'bottom-start',
  'top-start',
  'bottom-end',
  'top-end',
];
const DEFAULT_PLACEMENT_SIGNATURE = DEFAULT_PLACEMENTS.join('|');

export function usePopoverPosition(
  referenceRef: MutableRefObject<HTMLElement | null>,
  floatingRef: MutableRefObject<HTMLElement | null>,
  options: UsePopoverPositionOptions,
): PopoverPositionResult {
  const {
    isOpen,
    offset = 8,
    viewportMargin = 8,
    placementPriority,
    matchReferenceWidth = false,
  } = options;

  const placementSignatureRef = useRef<string>(DEFAULT_PLACEMENT_SIGNATURE);
  const placementPriorityRef = useRef<PopoverPlacement[]>(DEFAULT_PLACEMENTS);

  // Keep placement preference stable even when callers pass new array literals each render.

  const nextPlacementSignature = placementPriority?.join('|') ?? DEFAULT_PLACEMENT_SIGNATURE;
  if (nextPlacementSignature !== placementSignatureRef.current) {
    placementSignatureRef.current = nextPlacementSignature;
    placementPriorityRef.current =
      placementPriority && placementPriority.length > 0 ? placementPriority : DEFAULT_PLACEMENTS;
  }

  const [floatingStyles, setFloatingStyles] = useState<CSSProperties>({
    position: 'fixed',
    visibility: 'hidden',
    opacity: 0,
  });

  const updatePosition = useCallback(() => {
    const referenceEl = referenceRef.current;
    const floatingEl = floatingRef.current;

    if (!referenceEl || !floatingEl) {
      return;
    }

    const triggerRect = referenceEl.getBoundingClientRect();
    const popoverRect = floatingEl.getBoundingClientRect();

    const viewportWidth = window.innerWidth;
    const viewportHeight = window.innerHeight;
    const effectiveWidth = matchReferenceWidth
      ? Math.max(popoverRect.width, triggerRect.width)
      : popoverRect.width;
    const effectiveHeight = popoverRect.height;

    const computePlacement = (placement: PopoverPlacement) => {
      const [verticalPlacement, horizontalPlacement] = placement.split('-') as ['top' | 'bottom', 'start' | 'end'];

      const top =
        verticalPlacement === 'bottom'
          ? triggerRect.bottom + offset
          : triggerRect.top - effectiveHeight - offset;

      const left =
        horizontalPlacement === 'end'
          ? triggerRect.right - effectiveWidth
          : triggerRect.left;

      return { placement, top, left };
    };

    const doesFitWithinViewport = (top: number, left: number) => {
      const withinHorizontalBounds =
        left >= viewportMargin && left + effectiveWidth <= viewportWidth - viewportMargin;
      const withinVerticalBounds =
        top >= viewportMargin && top + effectiveHeight <= viewportHeight - viewportMargin;

      return withinHorizontalBounds && withinVerticalBounds;
    };

    let chosenTop: number | undefined;
    let chosenLeft: number | undefined;

    for (const placement of placementPriorityRef.current) {
      const { top, left } = computePlacement(placement);
      if (doesFitWithinViewport(top, left)) {
        chosenTop = top;
        chosenLeft = left;
        break;
      }
    }

    if (chosenTop === undefined || chosenLeft === undefined) {
      const fallback = computePlacement(placementPriorityRef.current[0]);
      chosenTop = fallback.top;
      chosenLeft = fallback.left;

      if (chosenTop + effectiveHeight > viewportHeight - viewportMargin) {
        chosenTop = viewportHeight - effectiveHeight - viewportMargin;
      }
      if (chosenTop < viewportMargin) {
        chosenTop = viewportMargin;
      }
      if (chosenLeft + effectiveWidth > viewportWidth - viewportMargin) {
        chosenLeft = viewportWidth - effectiveWidth - viewportMargin;
      }
      if (chosenLeft < viewportMargin) {
        chosenLeft = viewportMargin;
      }
    }

    const nextStyles: CSSProperties = {
      position: 'fixed',
      top: Math.max(viewportMargin, Math.round(chosenTop)),
      left: Math.max(viewportMargin, Math.round(chosenLeft)),
      maxHeight: `calc(100vh - ${viewportMargin * 2}px)`,
      width: matchReferenceWidth ? Math.round(effectiveWidth) : undefined,
      visibility: 'visible',
      opacity: 1,
      zIndex: 50,
    };

    setFloatingStyles((current) => {
      const keys = Object.keys(nextStyles) as (keyof CSSProperties)[];
      const hasChanges = keys.some((key) => current[key] !== nextStyles[key]);
      return hasChanges ? nextStyles : current;
    });
  }, [offset, viewportMargin, matchReferenceWidth, referenceRef, floatingRef]);

  useLayoutEffect(() => {
    if (!isOpen) {
      setFloatingStyles((prev) => {
        const prevVisibility = (prev as CSSProperties & { visibility?: string }).visibility;
        const prevOpacity = (prev as CSSProperties & { opacity?: number }).opacity;
        if (prevVisibility === 'hidden' && prevOpacity === 0) {
          return prev;
        }
        return { ...prev, visibility: 'hidden', opacity: 0 };
      });
      return;
    }

    updatePosition();

    const handleScroll = () => updatePosition();
    const handleResize = () => updatePosition();

    window.addEventListener('resize', handleResize);
    window.addEventListener('scroll', handleScroll, true);

    const floatingEl = floatingRef.current;
    const resizeObserver = floatingEl && typeof ResizeObserver !== 'undefined'
      ? new ResizeObserver(() => updatePosition())
      : null;

    if (floatingEl && resizeObserver) {
      resizeObserver.observe(floatingEl);
    }

    return () => {
      window.removeEventListener('resize', handleResize);
      window.removeEventListener('scroll', handleScroll, true);
      if (floatingEl && resizeObserver) {
        resizeObserver.disconnect();
      }
    };
  }, [isOpen, updatePosition, floatingRef]);

  return { floatingStyles, updatePosition };
}
