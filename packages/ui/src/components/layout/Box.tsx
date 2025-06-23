import type { HTMLAttributes, ElementType, ReactNode } from "react";
import { forwardRef } from "react";
import { cn } from "../../utils/tailwind-theme.js";

// Export types for external use
export type BoxVariant = "default" | "paper" | "outlined" | "elevated" | "subtle";
export type BoxPadding = "none" | "xs" | "sm" | "md" | "lg" | "xl";
export type BoxBorderRadius = "none" | "sm" | "md" | "lg" | "xl" | "full";

export interface BoxProps extends HTMLAttributes<HTMLElement> {
    /** HTML element type to render */
    component?: ElementType;
    /** Visual style variant of the box */
    variant?: BoxVariant;
    /** Padding size */
    padding?: BoxPadding;
    /** Border radius size */
    borderRadius?: BoxBorderRadius;
    /** Whether the box should take full width */
    fullWidth?: boolean;
    /** Whether the box should take full height */
    fullHeight?: boolean;
    /** Box content */
    children?: ReactNode;
}

/**
 * Box variant styles configuration
 */
const BOX_VARIANTS = {
    default: "tw-bg-background-default",
    paper: "tw-bg-background-paper tw-shadow-sm",
    outlined: "tw-bg-background-paper tw-border tw-border-gray-300",
    elevated: "tw-bg-background-paper tw-shadow-md",
    subtle: "tw-bg-gray-50",
} as const;

/**
 * Box padding size configuration
 */
const BOX_PADDING = {
    none: "",
    xs: "tw-p-1",
    sm: "tw-p-2", 
    md: "tw-p-4",
    lg: "tw-p-6",
    xl: "tw-p-8",
} as const;

/**
 * Box border radius configuration
 */
const BOX_BORDER_RADIUS = {
    none: "tw-rounded-none",
    sm: "tw-rounded-sm",
    md: "tw-rounded-md",
    lg: "tw-rounded-lg",
    xl: "tw-rounded-xl",
    full: "tw-rounded-full",
} as const;

/**
 * Builds the complete class string for the Box component
 */
function buildBoxClasses({
    variant,
    padding,
    borderRadius,
    fullWidth,
    fullHeight,
    className,
}: {
    variant: BoxVariant;
    padding: BoxPadding;
    borderRadius: BoxBorderRadius;
    fullWidth: boolean;
    fullHeight: boolean;
    className?: string;
}) {
    return cn(
        // Base box styles
        "tw-box-border",
        
        // Variant styles
        BOX_VARIANTS[variant],
        
        // Padding
        BOX_PADDING[padding],
        
        // Border radius
        BOX_BORDER_RADIUS[borderRadius],
        
        // Size modifiers
        fullWidth && "tw-w-full",
        fullHeight && "tw-h-full",
        
        // Custom className
        className,
    );
}

/**
 * A flexible Tailwind CSS box component for layout and containing content.
 * 
 * Features:
 * - 5 visual variants (default, paper, outlined, elevated, subtle)
 * - 6 padding sizes from none to xl
 * - 6 border radius options from none to full
 * - Customizable HTML element type (div by default)
 * - Full width and height options
 * - Theme-aware colors and shadows
 * 
 * @example
 * ```tsx
 * // Basic container
 * <Box variant="paper" padding="md" borderRadius="lg">
 *   Content here
 * </Box>
 * 
 * // Custom element type
 * <Box component="section" variant="outlined" padding="lg">
 *   Section content
 * </Box>
 * 
 * // Full width card
 * <Box variant="elevated" padding="lg" borderRadius="md" fullWidth>
 *   Card content
 * </Box>
 * ```
 */
export const Box = forwardRef<HTMLElement, BoxProps>(
    (
        {
            component: Component = "div",
            variant = "default",
            padding = "none",
            borderRadius = "none",
            fullWidth = false,
            fullHeight = false,
            className,
            children,
            ...props
        },
        ref,
    ) => {
        const boxClasses = buildBoxClasses({
            variant,
            padding,
            borderRadius,
            fullWidth,
            fullHeight,
            className,
        });

        return (
            <Component
                ref={ref}
                className={boxClasses}
                {...props}
            >
                {children}
            </Component>
        );
    },
);

Box.displayName = "Box";

/**
 * Pre-configured box components for common use cases
 */
export const BoxFactory = {
    /** Basic container with paper background */
    Paper: (props: Omit<BoxProps, "variant">) => (
        <Box variant="paper" {...props} />
    ),
    /** Card-like container with elevation */
    Card: (props: Omit<BoxProps, "variant" | "padding" | "borderRadius">) => (
        <Box variant="elevated" padding="lg" borderRadius="lg" {...props} />
    ),
    /** Outlined container */
    Outlined: (props: Omit<BoxProps, "variant">) => (
        <Box variant="outlined" {...props} />
    ),
    /** Subtle background container */
    Subtle: (props: Omit<BoxProps, "variant">) => (
        <Box variant="subtle" {...props} />
    ),
    /** Full width container */
    FullWidth: (props: Omit<BoxProps, "fullWidth">) => (
        <Box fullWidth {...props} />
    ),
    /** Flex container with center alignment */
    FlexCenter: (props: BoxProps) => (
        <Box className={cn("tw-flex tw-items-center tw-justify-center", props.className)} {...props} />
    ),
    /** Flex container with space between alignment */
    FlexBetween: (props: BoxProps) => (
        <Box className={cn("tw-flex tw-items-center tw-justify-between", props.className)} {...props} />
    ),
} as const;