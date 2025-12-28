import { useState, useRef, useEffect, useCallback, type ReactNode } from "react";
import { cn } from "../../lib/utils";

type TooltipSide = "top" | "bottom" | "left" | "right";

interface TooltipProps {
  content: string;
  children: ReactNode;
  side?: TooltipSide;
  delay?: number;
  className?: string;
}

// Minimum space required between tooltip and viewport edge
const VIEWPORT_PADDING = 8;
// Estimated tooltip height for boundary checks
const TOOLTIP_HEIGHT_ESTIMATE = 32;

function getFlippedSide(side: TooltipSide): TooltipSide {
  const flips: Record<TooltipSide, TooltipSide> = {
    top: "bottom",
    bottom: "top",
    left: "right",
    right: "left",
  };
  return flips[side];
}

function shouldFlip(
  side: TooltipSide,
  triggerRect: DOMRect
): boolean {
  switch (side) {
    case "top":
      // Not enough space above - flip to bottom
      return triggerRect.top < TOOLTIP_HEIGHT_ESTIMATE + VIEWPORT_PADDING;
    case "bottom":
      // Not enough space below - flip to top
      return window.innerHeight - triggerRect.bottom < TOOLTIP_HEIGHT_ESTIMATE + VIEWPORT_PADDING;
    case "left":
      // Not enough space on left - flip to right
      return triggerRect.left < 100 + VIEWPORT_PADDING;
    case "right":
      // Not enough space on right - flip to left
      return window.innerWidth - triggerRect.right < 100 + VIEWPORT_PADDING;
    default:
      return false;
  }
}

export function Tooltip({
  content,
  children,
  side = "top",
  delay = 300,
  className
}: TooltipProps) {
  const [isVisible, setIsVisible] = useState(false);
  const [effectiveSide, setEffectiveSide] = useState<TooltipSide>(side);
  const timeoutRef = useRef<ReturnType<typeof setTimeout>>();
  const triggerRef = useRef<HTMLDivElement>(null);

  const showTooltip = useCallback(() => {
    timeoutRef.current = setTimeout(() => {
      // Check viewport boundaries before showing
      if (triggerRef.current) {
        const rect = triggerRef.current.getBoundingClientRect();
        if (shouldFlip(side, rect)) {
          setEffectiveSide(getFlippedSide(side));
        } else {
          setEffectiveSide(side);
        }
      }
      setIsVisible(true);
    }, delay);
  }, [delay, side]);

  const hideTooltip = useCallback(() => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
    }
    setIsVisible(false);
  }, []);

  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  const positionClasses: Record<TooltipSide, string> = {
    top: "bottom-full left-1/2 -translate-x-1/2 mb-2",
    bottom: "top-full left-1/2 -translate-x-1/2 mt-2",
    left: "right-full top-1/2 -translate-y-1/2 mr-2",
    right: "left-full top-1/2 -translate-y-1/2 ml-2",
  };

  return (
    <div
      ref={triggerRef}
      className="relative inline-flex"
      onMouseEnter={showTooltip}
      onMouseLeave={hideTooltip}
      onFocus={showTooltip}
      onBlur={hideTooltip}
    >
      {children}
      {isVisible && content && (
        <div
          role="tooltip"
          className={cn(
            "absolute z-50 px-2.5 py-1.5 text-xs font-medium text-white bg-slate-800 rounded-md shadow-lg border border-white/10 whitespace-nowrap pointer-events-none animate-in fade-in-0 zoom-in-95 duration-100",
            positionClasses[effectiveSide],
            className
          )}
        >
          {content}
        </div>
      )}
    </div>
  );
}
