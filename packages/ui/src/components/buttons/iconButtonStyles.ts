import { cn } from "../../utils/tailwind-theme.js";

// Define types here to avoid circular imports
export type IconButtonVariant = "solid" | "transparent" | "space" | "custom" | "neon";
export type IconButtonSize = "sm" | "md" | "lg" | number;

/**
 * IconButton component styling utilities and configuration
 * Provides consistent styling patterns for icon buttons following Tailwind patterns
 */

// Configuration object for all icon button-related constants
export const ICON_BUTTON_CONFIG = {
    // Default sizes for icon buttons
    DEFAULT_SIZES: {
        sm: 32,
        md: 48,
        lg: 64,
    },
    
    // Ripple effect configuration
    RIPPLE: {
        RADIUS: 25,
        ANIMATION_DURATION: 0.6,
    },
    
    // Padding calculations based on size
    PADDING_BREAKPOINTS: {
        NONE: 16,    // <= 16px: no padding
        SMALL: 32,   // <= 32px: p-1
        MEDIUM: 48,  // <= 48px: p-2
        // > 48px: p-3
    },
} as const;

// Type-safe color system for icon button variants
export const ICON_BUTTON_COLORS = {
    // Ripple colors for each variant
    RIPPLE: {
        solid: "rgba(255, 255, 255, 0.5)",
        transparent: "rgba(100, 100, 100, 0.3)",
        space: "rgba(15, 170, 170, 0.8)",
        custom: "rgba(255, 255, 255, 0.5)",
        neon: "rgba(0, 255, 127, 0.6)",
    },
    
    // CSS variables for space variant (matching button space variant)
    SPACE: {
        border: {
            start: "#16a361", // secondary.main equivalent
            mid: "#0faaaa",   // primary.main equivalent  
            end: "#16a361",
        },
        background: {
            start: "#16a361", // secondary.main
            mid: "#0faaaa",   // primary.main
            end: "#16a361",
        },
        glow: "rgba(22, 163, 97, 0.4)",
        sweep: {
            light: "rgba(255, 255, 255, 0.2)",
            dark: "rgba(255, 255, 255, 0.1)",
        },
    },
    
    // Solid variant gradient colors
    SOLID: {
        gradientLight: "var(--icon-button-gradient-light)",
        gradientDark: "var(--icon-button-gradient-dark)",
    },
    
    // Transparent variant hover colors
    TRANSPARENT: {
        hover: "var(--icon-button-transparent-hover)",
        active: "var(--icon-button-transparent-active)",
    },
    
    // Neon variant colors
    NEON: {
        base: "#00ff7f",
        glow: "rgba(0, 255, 127, 0.4)",
        shadowInner: "rgba(0, 255, 127, 0.8)",
        shadowOuter: "rgba(0, 255, 127, 0.3)",
    },
} as const;

// Variant styles mapping - organized for better maintainability
export const ICON_VARIANT_STYLES: Record<IconButtonVariant, string> = {
    solid: cn(
        "tw-icon-button-solid",
        "tw-cursor-pointer tw-border-0",
        "focus:tw-ring-1 focus:tw-ring-secondary-main focus:tw-ring-offset-1 focus:tw-ring-opacity-50",
    ),
    
    transparent: cn(
        "tw-icon-button-transparent",
        "tw-cursor-pointer tw-border-0",
        "focus:tw-ring-1 focus:tw-ring-gray-400 focus:tw-ring-offset-1 focus:tw-ring-opacity-50",
    ),
    
    space: cn(
        "tw-group tw-cursor-pointer tw-border-0",
        "tw-transition-all tw-duration-300",
        "hover:tw-scale-105",
        "tw-shadow-md hover:tw-shadow-lg",
        // Direct gradient application
        "tw-bg-gradient-to-r tw-from-secondary-main tw-to-primary-main",
        "hover:tw-from-secondary-dark hover:tw-to-primary-dark",
        "hover:tw-shadow-secondary-main/40",
        "focus:tw-ring-1 focus:tw-ring-secondary-main focus:tw-ring-offset-1 focus:tw-ring-opacity-50",
        // Add anti-aliasing and smoother rendering
        "tw-antialiased",
        "[backface-visibility:hidden]",
        "[transform:translateZ(0)]",
    ),
    
    custom: cn(
        "tw-cursor-pointer tw-border-0",
        "tw-transition-all tw-duration-200",
        "focus:tw-ring-1 focus:tw-ring-offset-1 focus:tw-ring-opacity-50",
        "tw-shadow-md hover:tw-shadow-lg",
    ),
    
    neon: cn(
        "tw-icon-button-neon tw-group",
        "tw-cursor-pointer",
        "tw-transition-all tw-duration-300",
        "focus:tw-ring-1 focus:tw-ring-green-400 focus:tw-ring-offset-1 focus:tw-ring-opacity-50",
        "tw-antialiased",
        "[backface-visibility:hidden]",
        "[transform:translateZ(0)]",
    ),
};

// Base styles that apply to all icon button variants
export const BASE_ICON_BUTTON_STYLES = cn(
    // Layout and positioning
    "tw-relative tw-rounded-full",
    "tw-inline-flex tw-items-center tw-justify-center",
    
    // Transitions and interactions
    "tw-transition-all tw-duration-150",
    "tw-outline-none tw-overflow-hidden",
    
    // Focus accessibility
    "focus:tw-ring-offset-background",
);

/**
 * Helper function to get numeric size from IconButtonSize
 */
export const getNumericSize = (size: IconButtonSize): number => {
    if (typeof size === "number") return size;
    return ICON_BUTTON_CONFIG.DEFAULT_SIZES[size] || ICON_BUTTON_CONFIG.DEFAULT_SIZES.md;
};

/**
 * Helper function to calculate padding based on size
 * Returns appropriate Tailwind padding class
 */
export const calculatePadding = (size: number): string => {
    if (size <= ICON_BUTTON_CONFIG.PADDING_BREAKPOINTS.NONE) return "tw-p-0";
    if (size <= ICON_BUTTON_CONFIG.PADDING_BREAKPOINTS.SMALL) return "tw-p-1";
    if (size <= ICON_BUTTON_CONFIG.PADDING_BREAKPOINTS.MEDIUM) return "tw-p-2";
    return "tw-p-3";
};

/**
 * Helper to create ripple style object for icon buttons
 * Uses the same pattern as button ripples but with icon button dimensions
 */
export const createIconRippleStyle = (
    ripple: { x: number; y: number },
    color: string,
) => {
    // Provide fallback color if undefined
    const safeColor = color || "rgba(255, 255, 255, 0.5)";
    const fadeColor = safeColor.includes("rgba") 
        ? safeColor.replace(/[\d.]+(?=\))/, "0.1")
        : "rgba(255, 255, 255, 0.1)";
    
    return {
        left: ripple.x - ICON_BUTTON_CONFIG.RIPPLE.RADIUS,
        top: ripple.y - ICON_BUTTON_CONFIG.RIPPLE.RADIUS,
        width: ICON_BUTTON_CONFIG.RIPPLE.RADIUS * 2,
        height: ICON_BUTTON_CONFIG.RIPPLE.RADIUS * 2,
        background: `radial-gradient(circle, ${safeColor} 0%, ${fadeColor} 50%, transparent 70%)`,
    };
};

/**
 * Utility function to build complete icon button classes
 * Follows the pattern: base + variant + padding + state + custom
 */
export const buildIconButtonClasses = ({
    variant = "transparent",
    size,
    disabled = false,
    className,
}: {
    variant?: IconButtonVariant;
    size: number; // Already converted to numeric
    disabled?: boolean;
    className?: string;
}) => {
    return cn(
        // Base styles
        BASE_ICON_BUTTON_STYLES,
        
        // Padding based on size
        calculatePadding(size),
        
        // Variant styles
        ICON_VARIANT_STYLES[variant],
        
        // State styles
        disabled && "tw-opacity-50 tw-cursor-not-allowed tw-pointer-events-none",
        
        // Custom overrides
        className,
    );
};

/**
 * Get ripple color for a specific variant
 */
export const getRippleColor = (variant: IconButtonVariant): string => {
    return ICON_BUTTON_COLORS.RIPPLE[variant] || "rgba(255, 255, 255, 0.5)";
};

/**
 * Calculate contrast text color based on background color
 * Uses WCAG luminance formula to determine if text should be light or dark
 * @param bgColor - Background color in hex format (e.g., "#9333EA")
 * @returns "#000000" for light backgrounds, "#FFFFFF" for dark backgrounds
 */
export const getContrastTextColor = (bgColor: string): string => {
    // Remove # if present and convert to RGB
    const hex = bgColor.replace("#", "");
    const r = parseInt(hex.substr(0, 2), 16);
    const g = parseInt(hex.substr(2, 2), 16);
    const b = parseInt(hex.substr(4, 2), 16);
    
    // Calculate relative luminance using WCAG formula
    const luminance = (0.299 * r + 0.587 * g + 0.114 * b) / 255;
    
    // Return black text for light backgrounds, white for dark
    return luminance > 0.5 ? "#000000" : "#FFFFFF";
};

/**
 * Generate custom icon button styles with proper contrast
 * @param color - Background color in hex format
 * @returns Style object for custom colored icon button
 */
export const getCustomIconButtonStyle = (color: string) => ({
    backgroundColor: color,
    color: getContrastTextColor(color),
    borderColor: color,
    // Ensure icons and other child elements inherit the text color
    fill: getContrastTextColor(color),
    // Add hover effects
    "&:hover": {
        backgroundColor: color,
        opacity: 0.9,
    },
    "&:active": {
        backgroundColor: color,
        opacity: 0.8,
    },
});

/**
 * Style generators for space variant background layers
 * Uses a single-layer approach to avoid aliasing issues
 */
export const SPACE_BACKGROUND_STYLES = {
    // Single unified background with border and gradient
    main: {
        background: `
            radial-gradient(ellipse at center, ${ICON_BUTTON_COLORS.SPACE.glow} 0%, transparent 70%),
            linear-gradient(135deg, ${ICON_BUTTON_COLORS.SPACE.background.start} 0%, ${ICON_BUTTON_COLORS.SPACE.background.mid} 50%, ${ICON_BUTTON_COLORS.SPACE.background.end} 100%)
        `,
        border: "2px solid transparent",
        backgroundClip: "padding-box",
        boxShadow: `
            0 0 0 2px transparent,
            inset 0 0 0 1px rgba(255, 255, 255, 0.1)
        `,
    },
    // Border gradient using pseudo-element approach
    borderGradient: {
        background: `linear-gradient(135deg, ${ICON_BUTTON_COLORS.SPACE.border.start} 0%, ${ICON_BUTTON_COLORS.SPACE.border.mid} 50%, ${ICON_BUTTON_COLORS.SPACE.border.end} 100%)`,
    },
    sweep: {
        background: `linear-gradient(110deg, transparent 25%, ${ICON_BUTTON_COLORS.SPACE.sweep.light} 45%, ${ICON_BUTTON_COLORS.SPACE.sweep.dark} 55%, transparent 75%)`,
        backgroundSize: "200% 100%",
        animation: "iconButtonGradientSweep 3s ease-in-out infinite",
    },
} as const;

/**
 * Style generators for neon variant with glowing effects
 */
// Neon styles are handled entirely by CSS classes
