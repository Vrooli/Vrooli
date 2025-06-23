import { forwardRef } from "react";
import { cn } from "../../utils/tailwind-theme.js";

export type DividerOrientation = "horizontal" | "vertical";

export interface DividerProps {
    /** Orientation of the divider */
    orientation?: DividerOrientation;
    /** Additional className for custom styling */
    className?: string;
    /** Text content to display in the middle of the divider */
    children?: React.ReactNode;
    /** Position of text when children is provided */
    textAlign?: "left" | "center" | "right";
}

/**
 * A simple Tailwind CSS divider component matching FormDivider styling.
 * 
 * Features:
 * - Horizontal and vertical orientations
 * - Optional text/content in the middle
 * - Uses theme-aware colors (tw-bg-text-secondary tw-opacity-20)
 * - Full accessibility support
 * 
 * @example
 * ```tsx
 * // Simple horizontal divider
 * <Divider />
 * 
 * // Divider with text
 * <Divider>OR</Divider>
 * 
 * // Vertical divider
 * <Divider orientation="vertical" />
 * ```
 */
export const Divider = forwardRef<HTMLDivElement, DividerProps>(
    (
        {
            orientation = "horizontal",
            className,
            children,
            textAlign = "center",
        },
        ref,
    ) => {
        const isHorizontal = orientation === "horizontal";
        const hasChildren = Boolean(children);

        // Build divider line classes - matches FormDivider styling
        const lineClasses = cn(
            "tw-bg-text-secondary tw-opacity-20",
            isHorizontal ? "tw-h-0.5 tw-flex-grow" : "tw-w-0.5 tw-flex-grow",
        );

        // Container classes
        const containerClasses = cn(
            "tw-flex tw-items-center",
            isHorizontal ? "tw-w-full tw-flex-row" : "tw-h-full tw-flex-col",
            hasChildren && textAlign === "left" && "tw-justify-start",
            hasChildren && textAlign === "center" && "tw-justify-center", 
            hasChildren && textAlign === "right" && "tw-justify-end",
            className,
        );

        // Text/children wrapper classes
        const textClasses = cn(
            "tw-px-3 tw-text-sm tw-text-text-secondary",
            "tw-whitespace-nowrap",
        );

        // Divider with children
        if (hasChildren) {
            return (
                <div
                    ref={ref}
                    className={containerClasses}
                    role="separator"
                    aria-orientation={orientation}
                >
                    <div className={lineClasses} />
                    <div className={textClasses}>{children}</div>
                    <div className={lineClasses} />
                </div>
            );
        }

        // Simple divider without children - matches FormDivider exactly
        return (
            <div
                ref={ref}
                className={cn(
                    isHorizontal ? "tw-w-full tw-h-0.5" : "tw-h-full tw-w-0.5",
                    "tw-bg-text-secondary tw-opacity-20",
                    className,
                )}
                role="separator"
                aria-orientation={orientation}
            />
        );
    },
);

Divider.displayName = "Divider";

/**
 * Pre-configured divider components for common use cases
 */
export const DividerFactory = {
    /** Default horizontal divider */
    Horizontal: (props: Omit<DividerProps, "orientation">) => (
        <Divider orientation="horizontal" {...props} />
    ),
    /** Vertical divider */
    Vertical: (props: Omit<DividerProps, "orientation">) => (
        <Divider orientation="vertical" {...props} />
    ),
} as const;