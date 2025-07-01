import type { HTMLAttributes, ElementType, ReactNode } from "react";
import { forwardRef } from "react";
import { cn } from "../../utils/tailwind-theme.js";

// Export types for external use
export type TypographyVariant = 
    | "h1" | "h2" | "h3" | "h4" | "h5" | "h6"
    | "subtitle1" | "subtitle2"
    | "body1" | "body2"
    | "caption" | "overline"
    | "display1" | "display2";

export type TypographyAlign = "left" | "center" | "right" | "justify";
export type TypographyColor = "primary" | "secondary" | "success" | "warning" | "error" | "info" | "text" | "inherit";
export type TypographyWeight = "light" | "normal" | "medium" | "semibold" | "bold";
export type TypographySpacing = "none" | "tight" | "normal" | "relaxed" | "loose";

export interface TypographyProps extends HTMLAttributes<HTMLElement> {
    /** HTML element type to render */
    component?: ElementType;
    /** Typography variant that determines size and weight */
    variant?: TypographyVariant;
    /** Text alignment */
    align?: TypographyAlign;
    /** Text color theme */
    color?: TypographyColor;
    /** Font weight override */
    weight?: TypographyWeight;
    /** Whether text should be uppercase */
    uppercase?: boolean;
    /** Whether text should have no wrap */
    noWrap?: boolean;
    /** Whether text should be truncated with ellipsis */
    truncate?: boolean;
    /** Control vertical spacing (margin) - defaults to browser defaults */
    spacing?: TypographySpacing;
    /** Typography content */
    children?: ReactNode;
}

/**
 * Typography variant styles configuration
 */
const TYPOGRAPHY_VARIANTS = {
    // Display variants
    display1: "tw-text-6xl tw-font-bold tw-leading-tight",
    display2: "tw-text-5xl tw-font-bold tw-leading-tight",
    
    // Heading variants
    h1: "tw-text-4xl tw-font-bold tw-leading-tight",
    h2: "tw-text-3xl tw-font-semibold tw-leading-tight",
    h3: "tw-text-2xl tw-font-semibold tw-leading-tight",
    h4: "tw-text-xl tw-font-semibold tw-leading-snug",
    h5: "tw-text-lg tw-font-medium tw-leading-snug",
    h6: "tw-text-base tw-font-medium tw-leading-normal",
    
    // Subtitle variants
    subtitle1: "tw-text-lg tw-font-normal tw-leading-relaxed",
    subtitle2: "tw-text-base tw-font-normal tw-leading-relaxed",
    
    // Body variants
    body1: "tw-text-base tw-font-normal tw-leading-relaxed",
    body2: "tw-text-sm tw-font-normal tw-leading-relaxed",
    
    // Utility variants
    caption: "tw-text-xs tw-font-normal tw-leading-normal",
    overline: "tw-text-xs tw-font-medium tw-leading-normal tw-uppercase tw-tracking-wide",
} as const;

/**
 * Typography alignment configuration
 */
const TYPOGRAPHY_ALIGN = {
    left: "tw-text-left",
    center: "tw-text-center",
    right: "tw-text-right",
    justify: "tw-text-justify",
} as const;

/**
 * Typography color configuration
 */
const TYPOGRAPHY_COLOR = {
    primary: "tw-text-primary",
    secondary: "tw-text-secondary",
    success: "tw-text-green-600",
    warning: "tw-text-orange-600",
    error: "tw-text-red-600",
    info: "tw-text-blue-600",
    text: "tw-text-foreground",
    inherit: "tw-text-inherit",
} as const;

/**
 * Typography weight configuration
 */
const TYPOGRAPHY_WEIGHT = {
    light: "tw-font-light",
    normal: "tw-font-normal",
    medium: "tw-font-medium",
    semibold: "tw-font-semibold",
    bold: "tw-font-bold",
} as const;

/**
 * Typography spacing configuration
 */
const TYPOGRAPHY_SPACING = {
    none: "tw-my-0",
    tight: "tw-my-1",      // 0.25rem = 4px
    normal: "tw-my-2",     // 0.5rem = 8px
    relaxed: "tw-my-4",    // 1rem = 16px
    loose: "tw-my-6",      // 1.5rem = 24px
} as const;

/**
 * Default HTML elements for each variant
 */
const DEFAULT_VARIANT_ELEMENTS: Record<TypographyVariant, ElementType> = {
    display1: "h1",
    display2: "h1",
    h1: "h1",
    h2: "h2",
    h3: "h3",
    h4: "h4",
    h5: "h5",
    h6: "h6",
    subtitle1: "h6",
    subtitle2: "h6",
    body1: "p",
    body2: "p",
    caption: "span",
    overline: "span",
};

/**
 * Builds the complete class string for the Typography component
 */
function buildTypographyClasses({
    variant,
    align,
    color,
    weight,
    spacing,
    uppercase,
    noWrap,
    truncate,
    className,
}: {
    variant: TypographyVariant;
    align: TypographyAlign;
    color: TypographyColor;
    weight?: TypographyWeight;
    spacing?: TypographySpacing;
    uppercase: boolean;
    noWrap: boolean;
    truncate: boolean;
    className?: string;
}) {
    return cn(
        // Base typography styles
        "tw-box-border",
        
        // Variant styles (includes default weight)
        TYPOGRAPHY_VARIANTS[variant],
        
        // Override weight if specified
        weight && TYPOGRAPHY_WEIGHT[weight],
        
        // Alignment
        TYPOGRAPHY_ALIGN[align],
        
        // Color
        TYPOGRAPHY_COLOR[color],
        
        // Spacing
        spacing && TYPOGRAPHY_SPACING[spacing],
        
        // Text transformations
        uppercase && "tw-uppercase",
        noWrap && "tw-whitespace-nowrap",
        truncate && "tw-truncate",
        
        // Custom className
        className,
    );
}

/**
 * A flexible Tailwind CSS typography component for text content.
 * 
 * Features:
 * - 12 semantic variants from display text to captions
 * - 4 text alignment options
 * - 8 color themes including semantic colors
 * - 5 font weight options
 * - 5 spacing options to control vertical margins
 * - Text transformation options (uppercase, nowrap, truncate)
 * - Semantic HTML element mapping by default
 * - Custom element type override support
 * 
 * @example
 * ```tsx
 * // Page heading
 * <Typography variant="h1" color="primary">
 *   Welcome to our app
 * </Typography>
 * 
 * // Dialog title with no spacing
 * <Typography variant="h4" spacing="none">
 *   Confirm Action
 * </Typography>
 * 
 * // Body text with custom styling
 * <Typography variant="body1" align="center" weight="medium">
 *   This is important body text
 * </Typography>
 * 
 * // Custom element type
 * <Typography component="span" variant="caption" color="secondary">
 *   Small inline text
 * </Typography>
 * 
 * // Truncated text
 * <Typography variant="body2" truncate>
 *   This very long text will be truncated with ellipsis when it overflows
 * </Typography>
 * ```
 */
export const Typography = forwardRef<HTMLElement, TypographyProps>(
    (
        {
            component,
            variant = "body1",
            align = "left",
            color = "text",
            weight,
            spacing,
            uppercase = false,
            noWrap = false,
            truncate = false,
            className,
            children,
            ...props
        },
        ref,
    ) => {
        // Determine the component to render
        const Component = component || DEFAULT_VARIANT_ELEMENTS[variant] || "p";

        const typographyClasses = buildTypographyClasses({
            variant,
            align,
            color,
            weight,
            spacing,
            uppercase,
            noWrap,
            truncate,
            className,
        });

        return (
            <Component
                ref={ref}
                className={typographyClasses}
                {...props}
            >
                {children}
            </Component>
        );
    },
);

Typography.displayName = "Typography";

/**
 * Pre-configured typography components for common use cases
 */
export const TypographyFactory = {
    /** Large display heading */
    Display1: (props: Omit<TypographyProps, "variant">) => (
        <Typography variant="display1" {...props} />
    ),
    /** Medium display heading */
    Display2: (props: Omit<TypographyProps, "variant">) => (
        <Typography variant="display2" {...props} />
    ),
    /** Main page heading */
    H1: (props: Omit<TypographyProps, "variant">) => (
        <Typography variant="h1" {...props} />
    ),
    /** Section heading */
    H2: (props: Omit<TypographyProps, "variant">) => (
        <Typography variant="h2" {...props} />
    ),
    /** Subsection heading */
    H3: (props: Omit<TypographyProps, "variant">) => (
        <Typography variant="h3" {...props} />
    ),
    /** Minor heading */
    H4: (props: Omit<TypographyProps, "variant">) => (
        <Typography variant="h4" {...props} />
    ),
    /** Small heading */
    H5: (props: Omit<TypographyProps, "variant">) => (
        <Typography variant="h5" {...props} />
    ),
    /** Tiny heading */
    H6: (props: Omit<TypographyProps, "variant">) => (
        <Typography variant="h6" {...props} />
    ),
    /** Large subtitle */
    Subtitle1: (props: Omit<TypographyProps, "variant">) => (
        <Typography variant="subtitle1" {...props} />
    ),
    /** Small subtitle */
    Subtitle2: (props: Omit<TypographyProps, "variant">) => (
        <Typography variant="subtitle2" {...props} />
    ),
    /** Primary body text */
    Body1: (props: Omit<TypographyProps, "variant">) => (
        <Typography variant="body1" {...props} />
    ),
    /** Secondary body text */
    Body2: (props: Omit<TypographyProps, "variant">) => (
        <Typography variant="body2" {...props} />
    ),
    /** Small caption text */
    Caption: (props: Omit<TypographyProps, "variant">) => (
        <Typography variant="caption" {...props} />
    ),
    /** Uppercase label text */
    Overline: (props: Omit<TypographyProps, "variant">) => (
        <Typography variant="overline" {...props} />
    ),
    
    // Utility factory components
    /** Centered text */
    Centered: (props: TypographyProps) => (
        <Typography align="center" {...props} />
    ),
    /** Primary colored text */
    Primary: (props: TypographyProps) => (
        <Typography color="primary" {...props} />
    ),
    /** Secondary colored text */
    Secondary: (props: TypographyProps) => (
        <Typography color="secondary" {...props} />
    ),
    /** Error colored text */
    Error: (props: TypographyProps) => (
        <Typography color="error" {...props} />
    ),
    /** Success colored text */
    Success: (props: TypographyProps) => (
        <Typography color="success" {...props} />
    ),
    /** Bold text */
    Bold: (props: TypographyProps) => (
        <Typography weight="bold" {...props} />
    ),
    /** Truncated text */
    Truncated: (props: TypographyProps) => (
        <Typography truncate {...props} />
    ),
} as const;
