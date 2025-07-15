import type { HTMLAttributes, ElementType, ReactNode } from "react";
import { forwardRef } from "react";
import { cn } from "../../utils/tailwind-theme.js";

// Export types for external use
export type StackDirection = "row" | "column" | "row-reverse" | "column-reverse";
export type StackJustify = "start" | "end" | "center" | "between" | "around" | "evenly";
export type StackAlign = "start" | "end" | "center" | "baseline" | "stretch";
export type StackWrap = "nowrap" | "wrap" | "wrap-reverse";
export type StackSpacing = "none" | "xs" | "sm" | "md" | "lg" | "xl" | "2xl";

export interface StackProps extends HTMLAttributes<HTMLElement> {
    /** HTML element type to render */
    component?: ElementType;
    /** Flex direction */
    direction?: StackDirection;
    /** Justify content alignment */
    justify?: StackJustify;
    /** Align items alignment */
    align?: StackAlign;
    /** Flex wrap behavior */
    wrap?: StackWrap;
    /** Spacing between items */
    spacing?: StackSpacing;
    /** Whether the stack should take full width */
    fullWidth?: boolean;
    /** Whether the stack should take full height */
    fullHeight?: boolean;
    /** Whether to divide items with a visual separator */
    divider?: boolean;
    /** Stack content */
    children?: ReactNode;
}

/**
 * Stack direction configuration
 */
const STACK_DIRECTION = {
    row: "tw-flex-row",
    column: "tw-flex-col",
    "row-reverse": "tw-flex-row-reverse",
    "column-reverse": "tw-flex-col-reverse",
} as const;

/**
 * Stack justify content configuration
 */
const STACK_JUSTIFY = {
    start: "tw-justify-start",
    end: "tw-justify-end",
    center: "tw-justify-center",
    between: "tw-justify-between",
    around: "tw-justify-around",
    evenly: "tw-justify-evenly",
} as const;

/**
 * Stack align items configuration
 */
const STACK_ALIGN = {
    start: "tw-items-start",
    end: "tw-items-end",
    center: "tw-items-center",
    baseline: "tw-items-baseline",
    stretch: "tw-items-stretch",
} as const;

/**
 * Stack wrap configuration
 */
const STACK_WRAP = {
    nowrap: "tw-flex-nowrap",
    wrap: "tw-flex-wrap",
    "wrap-reverse": "tw-flex-wrap-reverse",
} as const;

/**
 * Stack spacing configuration for row direction
 */
const STACK_SPACING_ROW = {
    none: "",
    xs: "tw-space-x-1",
    sm: "tw-space-x-2",
    md: "tw-space-x-4",
    lg: "tw-space-x-6",
    xl: "tw-space-x-8",
    "2xl": "tw-space-x-12",
} as const;

/**
 * Stack spacing configuration for column direction
 */
const STACK_SPACING_COLUMN = {
    none: "",
    xs: "tw-space-y-1",
    sm: "tw-space-y-2",
    md: "tw-space-y-4",
    lg: "tw-space-y-6",
    xl: "tw-space-y-8",
    "2xl": "tw-space-y-12",
} as const;

/**
 * Stack divider configuration for row direction
 */
const STACK_DIVIDER_ROW = {
    none: "",
    xs: "tw-divide-x tw-divide-x-1 tw-divide-gray-300",
    sm: "tw-divide-x tw-divide-x-2 tw-divide-gray-300",
    md: "tw-divide-x tw-divide-x-4 tw-divide-gray-300",
    lg: "tw-divide-x tw-divide-x-6 tw-divide-gray-300",
    xl: "tw-divide-x tw-divide-x-8 tw-divide-gray-300",
    "2xl": "tw-divide-x tw-divide-x-12 tw-divide-gray-300",
} as const;

/**
 * Stack divider configuration for column direction
 */
const STACK_DIVIDER_COLUMN = {
    none: "",
    xs: "tw-divide-y tw-divide-y-1 tw-divide-gray-300",
    sm: "tw-divide-y tw-divide-y-2 tw-divide-gray-300",
    md: "tw-divide-y tw-divide-y-4 tw-divide-gray-300",
    lg: "tw-divide-y tw-divide-y-6 tw-divide-gray-300",
    xl: "tw-divide-y tw-divide-y-8 tw-divide-gray-300",
    "2xl": "tw-divide-y tw-divide-y-12 tw-divide-gray-300",
} as const;

/**
 * Builds the complete class string for the Stack component
 */
function buildStackClasses({
    direction,
    justify,
    align,
    wrap,
    spacing,
    fullWidth,
    fullHeight,
    divider,
    className,
}: {
    direction: StackDirection;
    justify: StackJustify;
    align: StackAlign;
    wrap: StackWrap;
    spacing: StackSpacing;
    fullWidth: boolean;
    fullHeight: boolean;
    divider: boolean;
    className?: string;
}) {
    const isRow = direction === "row" || direction === "row-reverse";
    const spacingClasses = isRow ? STACK_SPACING_ROW[spacing] : STACK_SPACING_COLUMN[spacing];
    const dividerClasses = divider ? (isRow ? STACK_DIVIDER_ROW[spacing] : STACK_DIVIDER_COLUMN[spacing]) : "";

    return cn(
        // Base flex styles
        "tw-flex",
        
        // Direction
        STACK_DIRECTION[direction],
        
        // Justify content
        STACK_JUSTIFY[justify],
        
        // Align items
        STACK_ALIGN[align],
        
        // Wrap
        STACK_WRAP[wrap],
        
        // Spacing (use divider classes if divider is enabled, otherwise use spacing)
        divider ? dividerClasses : spacingClasses,
        
        // Size modifiers
        fullWidth && "tw-w-full",
        fullHeight && "tw-h-full",
        
        // Custom className
        className,
    );
}

/**
 * A flexible Tailwind CSS stack component for arranging items in a flex layout.
 * 
 * Features:
 * - 4 flex directions (row, column, row-reverse, column-reverse)
 * - 6 justify content options (start, end, center, between, around, evenly)
 * - 5 align items options (start, end, center, baseline, stretch)
 * - 3 wrap options (nowrap, wrap, wrap-reverse)
 * - 7 spacing sizes from none to 2xl
 * - Optional visual dividers between items
 * - Customizable HTML element type (div by default)
 * - Full width and height options
 * 
 * @example
 * ```tsx
 * // Vertical stack with spacing
 * <Stack direction="column" spacing="md">
 *   <div>Item 1</div>
 *   <div>Item 2</div>
 *   <div>Item 3</div>
 * </Stack>
 * 
 * // Horizontal stack with dividers
 * <Stack direction="row" spacing="sm" divider>
 *   <span>Link 1</span>
 *   <span>Link 2</span>
 *   <span>Link 3</span>
 * </Stack>
 * 
 * // Centered flex container
 * <Stack direction="row" justify="center" align="center" fullWidth>
 *   <button>Primary Action</button>
 *   <button>Secondary Action</button>
 * </Stack>
 * ```
 */
export const Stack = forwardRef<HTMLElement, StackProps>(
    (
        {
            component: Component = "div",
            direction = "column",
            justify = "start",
            align = "start",
            wrap = "nowrap",
            spacing = "none",
            fullWidth = false,
            fullHeight = false,
            divider = false,
            className,
            children,
            ...props
        },
        ref,
    ) => {
        const stackClasses = buildStackClasses({
            direction,
            justify,
            align,
            wrap,
            spacing,
            fullWidth,
            fullHeight,
            divider,
            className,
        });

        return (
            <Component
                ref={ref}
                className={stackClasses}
                {...props}
            >
                {children}
            </Component>
        );
    },
);

Stack.displayName = "Stack";

/**
 * Pre-configured stack components for common use cases
 */
export const StackFactory = {
    /** Vertical stack with medium spacing */
    Vertical: (props: Omit<StackProps, "direction">) => (
        <Stack direction="column" spacing="md" {...props} />
    ),
    /** Horizontal stack with medium spacing */
    Horizontal: (props: Omit<StackProps, "direction">) => (
        <Stack direction="row" spacing="md" {...props} />
    ),
    /** Centered container (both axes) */
    Center: (props: Omit<StackProps, "justify" | "align">) => (
        <Stack justify="center" align="center" {...props} />
    ),
    /** Space between container */
    Between: (props: Omit<StackProps, "justify">) => (
        <Stack justify="between" {...props} />
    ),
    /** Full width horizontal stack */
    HorizontalFull: (props: Omit<StackProps, "direction" | "fullWidth">) => (
        <Stack direction="row" fullWidth {...props} />
    ),
    /** Full height vertical stack */
    VerticalFull: (props: Omit<StackProps, "direction" | "fullHeight">) => (
        <Stack direction="column" fullHeight {...props} />
    ),
    /** Navbar-style horizontal stack with space between */
    Navbar: (props: Omit<StackProps, "direction" | "justify" | "align" | "fullWidth">) => (
        <Stack direction="row" justify="between" align="center" fullWidth {...props} />
    ),
    /** Sidebar-style vertical stack with full height */
    Sidebar: (props: Omit<StackProps, "direction" | "fullHeight">) => (
        <Stack direction="column" fullHeight spacing="sm" {...props} />
    ),
    /** Button group with dividers */
    ButtonGroup: (props: Omit<StackProps, "direction" | "divider" | "spacing">) => (
        <Stack direction="row" divider spacing="sm" {...props} />
    ),
    /** Form fields vertical stack */
    Form: (props: Omit<StackProps, "direction" | "spacing" | "fullWidth">) => (
        <Stack direction="column" spacing="lg" fullWidth {...props} />
    ),
} as const;
