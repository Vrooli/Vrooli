import { useState, useRef, useEffect, ReactNode } from "react";

interface TooltipProps {
  content: ReactNode;
  children: ReactNode;
  position?: "top" | "bottom" | "left" | "right";
  delay?: number;
  className?: string;
}

/**
 * A reusable tooltip component that provides helpful context on hover.
 * Supports different positions and customizable delay.
 */
function Tooltip({
  content,
  children,
  position = "top",
  delay = 200,
  className = "",
}: TooltipProps) {
  const [isVisible, setIsVisible] = useState(false);
  const [coords, setCoords] = useState({ x: 0, y: 0 });
  const timeoutRef = useRef<number | null>(null);
  const triggerRef = useRef<HTMLDivElement>(null);
  const tooltipRef = useRef<HTMLDivElement>(null);

  const showTooltip = () => {
    timeoutRef.current = window.setTimeout(() => {
      if (triggerRef.current) {
        const rect = triggerRef.current.getBoundingClientRect();

        let x = 0;
        let y = 0;

        switch (position) {
          case "top":
            x = rect.left + rect.width / 2;
            y = rect.top - 8;
            break;
          case "bottom":
            x = rect.left + rect.width / 2;
            y = rect.bottom + 8;
            break;
          case "left":
            x = rect.left - 8;
            y = rect.top + rect.height / 2;
            break;
          case "right":
            x = rect.right + 8;
            y = rect.top + rect.height / 2;
            break;
        }

        setCoords({ x, y });
      }
      setIsVisible(true);
    }, delay);
  };

  const hideTooltip = () => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }
    setIsVisible(false);
  };

  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  const getPositionStyles = () => {
    switch (position) {
      case "top":
        return {
          left: coords.x,
          top: coords.y,
          transform: "translate(-50%, -100%)",
        };
      case "bottom":
        return {
          left: coords.x,
          top: coords.y,
          transform: "translate(-50%, 0)",
        };
      case "left":
        return {
          left: coords.x,
          top: coords.y,
          transform: "translate(-100%, -50%)",
        };
      case "right":
        return {
          left: coords.x,
          top: coords.y,
          transform: "translate(0, -50%)",
        };
    }
  };

  const getArrowStyles = () => {
    switch (position) {
      case "top":
        return "bottom-[-4px] left-1/2 -translate-x-1/2 border-l-transparent border-r-transparent border-b-transparent border-t-gray-800";
      case "bottom":
        return "top-[-4px] left-1/2 -translate-x-1/2 border-l-transparent border-r-transparent border-t-transparent border-b-gray-800";
      case "left":
        return "right-[-4px] top-1/2 -translate-y-1/2 border-t-transparent border-b-transparent border-r-transparent border-l-gray-800";
      case "right":
        return "left-[-4px] top-1/2 -translate-y-1/2 border-t-transparent border-b-transparent border-l-transparent border-r-gray-800";
    }
  };

  return (
    <>
      <div
        ref={triggerRef}
        onMouseEnter={showTooltip}
        onMouseLeave={hideTooltip}
        onFocus={showTooltip}
        onBlur={hideTooltip}
        className="inline-flex"
      >
        {children}
      </div>

      {isVisible && (
        <div
          ref={tooltipRef}
          role="tooltip"
          style={getPositionStyles()}
          className={`fixed z-50 px-3 py-2 text-sm text-surface bg-gray-800 rounded-lg shadow-lg max-w-xs ${className}`}
        >
          {content}
          <div
            className={`absolute w-0 h-0 border-4 ${getArrowStyles()}`}
            aria-hidden="true"
          />
        </div>
      )}
    </>
  );
}

export default Tooltip;
