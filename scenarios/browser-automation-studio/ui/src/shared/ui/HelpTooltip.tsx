import { HelpCircle } from "lucide-react";
import Tooltip from "./Tooltip";

interface HelpTooltipProps {
  /** The help text to display */
  content: string;
  /** Position of the tooltip relative to the icon */
  position?: "top" | "bottom" | "left" | "right";
  /** Size of the help icon */
  size?: number;
  /** Additional CSS class for the icon */
  className?: string;
  /** Aria label for accessibility */
  ariaLabel?: string;
}

/**
 * A small help icon that shows a tooltip with explanatory text on hover.
 * Use this component to provide contextual help for complex UI elements.
 *
 * @example
 * <HelpTooltip content="This selector targets elements on the page" />
 *
 * @example
 * <label className="flex items-center gap-1">
 *   Element Selector
 *   <HelpTooltip content="Use CSS selectors like #id or .class" position="right" />
 * </label>
 */
function HelpTooltip({
  content,
  position = "top",
  size = 14,
  className = "",
  ariaLabel = "Help information",
}: HelpTooltipProps) {
  return (
    <Tooltip content={content} position={position}>
      <button
        type="button"
        className={`inline-flex items-center justify-center text-gray-400 hover:text-gray-200 transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-flow-accent/60 rounded-full ${className}`}
        aria-label={ariaLabel}
        tabIndex={0}
      >
        <HelpCircle size={size} />
      </button>
    </Tooltip>
  );
}

export default HelpTooltip;
