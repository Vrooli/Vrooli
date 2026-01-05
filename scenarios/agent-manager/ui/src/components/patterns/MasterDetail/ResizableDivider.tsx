import * as React from "react";
import { cn } from "../../../lib/utils";

interface ResizableDividerProps {
  /** Mouse down handler to start resize */
  onMouseDown: (e: React.MouseEvent) => void;
  /** Whether resize is currently in progress */
  isResizing?: boolean;
  /** Whether the divider is disabled (e.g., when panels are collapsed) */
  disabled?: boolean;
  /** Additional CSS classes */
  className?: string;
}

/**
 * Draggable vertical divider for resizing panels.
 * Changes cursor to col-resize on hover and shows visual feedback.
 */
export function ResizableDivider({
  onMouseDown,
  isResizing = false,
  disabled = false,
  className,
}: ResizableDividerProps) {
  return (
    <div
      className={cn(
        // Base styles
        "w-1 shrink-0 transition-colors",
        // Interactive states
        !disabled && "cursor-col-resize hover:bg-primary/50",
        // Default and active colors
        isResizing ? "bg-primary/50" : "bg-border",
        // Disabled state
        disabled && "cursor-default opacity-50",
        className
      )}
      onMouseDown={disabled ? undefined : onMouseDown}
      role="separator"
      aria-orientation="vertical"
      aria-disabled={disabled}
    />
  );
}
