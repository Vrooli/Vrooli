import { cn } from "../../utils/tailwind-theme.js";
import type { ButtonVariant, ButtonSize, ButtonBorderRadius } from "./Button.js";

/**
 * Button component styling utilities and configuration
 * Provides consistent styling patterns for Tailwind components
 */

// Configuration object for all button-related constants
export const BUTTON_CONFIG = {
    // Spinner thickness by size
    SPINNER_THICKNESS: {
        sm: 2,
        md: 3,
        lg: 3,
        icon: 3,
    },
    
    // Loading spinner sizes
    SPINNER_SIZES: {
        sm: 16,
        md: 20,
        lg: 24,
        icon: 20,
    },
    
    // Ripple effect configuration
    RIPPLE: {
        RADIUS: 25,
        ANIMATION_DURATION: 0.6,
    },
} as const;

// Type-safe color system for button variants
export const BUTTON_COLORS = {
    // Ripple colors for each variant
    RIPPLE: {
        primary: "rgba(255, 255, 255, 0.5)",
        secondary: "rgba(0, 0, 0, 0.3)",
        outline: "rgba(22, 163, 97, 0.3)",
        ghost: "rgba(22, 163, 97, 0.3)",
        danger: "rgba(255, 255, 255, 0.5)",
        space: "rgba(15, 170, 170, 0.8)",
        custom: "rgba(255, 255, 255, 0.3)",
        neon: "rgba(0, 255, 127, 0.6)",
    },
    
    // CSS variables for space variant - using visible gradient like signup button
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
    
    // Neon variant colors
    NEON: {
        base: "#00ff7f",
        glow: "rgba(0, 255, 127, 0.4)",
        shadowInner: "rgba(0, 255, 127, 0.8)",
        shadowOuter: "rgba(0, 255, 127, 0.3)",
    },
} as const;

// Variant styles mapping - organized for better maintainability
export const VARIANT_STYLES: Record<ButtonVariant, string> = {
    primary: cn(
        "tw-bg-secondary-main tw-text-white tw-border-0",
        "hover:tw-bg-secondary-dark",
        "focus:tw-ring-1 focus:tw-ring-secondary-main focus:tw-ring-offset-1 focus:tw-ring-opacity-50",
        "tw-shadow-md hover:tw-shadow-lg",
    ),
    
    secondary: cn(
        "tw-bg-gray-200 tw-text-gray-800 tw-border-0",
        "hover:tw-bg-gray-300",
        "focus:tw-ring-1 focus:tw-ring-gray-400 focus:tw-ring-offset-1 focus:tw-ring-opacity-50",
        "tw-shadow-sm hover:tw-shadow-md",
    ),
    
    outline: cn(
        "tw-bg-transparent tw-text-secondary-main",
        "tw-border-2 tw-border-solid tw-border-current",
        "hover:tw-bg-secondary-main hover:tw-text-white hover:tw-border-secondary-main",
        "focus:tw-ring-1 focus:tw-ring-secondary-main focus:tw-ring-offset-1 focus:tw-ring-opacity-50",
    ),
    
    ghost: cn(
        "tw-bg-transparent tw-text-secondary-main tw-border-0",
        "hover:tw-bg-secondary-main hover:tw-bg-opacity-10",
        "focus:tw-ring-1 focus:tw-ring-secondary-main focus:tw-ring-offset-1 focus:tw-ring-opacity-50",
    ),
    
    danger: cn(
        "tw-bg-red-600 tw-text-white tw-border-0",
        "hover:tw-bg-red-700",
        "focus:tw-ring-1 focus:tw-ring-red-600 focus:tw-ring-offset-1 focus:tw-ring-opacity-50",
        "tw-shadow-md hover:tw-shadow-lg",
    ),
    
    space: cn(
        "tw-text-white tw-group tw-border-0",
        "tw-transition-all tw-duration-300",
        "hover:tw-scale-105",
        "tw-shadow-md hover:tw-shadow-lg",
        // Direct gradient application
        "tw-bg-gradient-to-r tw-from-secondary-main tw-to-primary-main",
        "hover:tw-from-secondary-dark hover:tw-to-primary-dark",
        "hover:tw-shadow-secondary-main/40",
    ),
    
    custom: cn(
        "tw-transition-all tw-duration-200 tw-border-0",
        "focus:tw-ring-1 focus:tw-ring-offset-1 focus:tw-ring-opacity-50",
        "tw-shadow-md hover:tw-shadow-lg",
    ),
    
    neon: cn(
        "tw-button-neon-container",
        "tw-text-white tw-group",
    ),
};

// Size styles mapping
export const SIZE_STYLES: Record<ButtonSize, string> = {
    sm: "tw-h-8 tw-px-3 tw-text-xs",
    md: "tw-h-10 tw-px-4 tw-text-sm",
    lg: "tw-h-12 tw-px-6 tw-text-base",
    icon: "tw-h-10 tw-w-10 tw-p-0",
};

// Border radius styles mapping
export const BORDER_RADIUS_STYLES: Record<ButtonBorderRadius, string> = {
    none: "tw-rounded-none",
    minimal: "tw-rounded", // Default - 4px
    pill: "tw-rounded-full",
};

// Base styles that apply to all button variants (without border radius)
export const BASE_BUTTON_STYLES = cn(
    // Layout and alignment
    "tw-inline-flex tw-items-center tw-justify-center tw-gap-2",
    
    // Typography
    "tw-font-sans tw-font-medium tw-tracking-wider tw-uppercase",
    
    // Interactions and cursor
    "tw-cursor-pointer",
    
    // Transitions and interactions
    "tw-transition-all tw-duration-200",
    "tw-outline-none",
    "focus:tw-ring-offset-background",
    
    // Positioning for effects
    "tw-relative tw-overflow-hidden",
);

/**
 * Utility function to determine spinner variant based on button variant
 */
export const getSpinnerVariant = (variant: ButtonVariant): "current" | "secondary" | "white" => {
    if (variant === "outline" || variant === "ghost" || variant === "custom" || variant === "neon") {
        return "current";
    }
    if (variant === "secondary") {
        return "secondary";
    }
    return "white";
};

/**
 * Utility function to get spinner configuration for a given size
 */
export const getSpinnerConfig = (size: ButtonSize) => ({
    size: BUTTON_CONFIG.SPINNER_SIZES[size],
    thickness: BUTTON_CONFIG.SPINNER_THICKNESS[size],
});

/**
 * Utility function to build complete button classes
 * Follows the pattern: base + variant + size + borderRadius + state + custom
 */
export const buildButtonClasses = ({
    variant = "primary",
    size = "md",
    borderRadius = "minimal",
    fullWidth = false,
    disabled = false,
    className,
}: {
    variant?: ButtonVariant;
    size?: ButtonSize;
    borderRadius?: ButtonBorderRadius;
    fullWidth?: boolean;
    disabled?: boolean;
    className?: string;
}) => {
    return cn(
        // Base styles
        BASE_BUTTON_STYLES,
        
        // Width styles
        fullWidth ? "tw-w-full" : "tw-w-auto",
        
        // Special handling for space and neon variants
        (variant === "space" || variant === "neon") && "tw-group",
        
        // Variant styles
        VARIANT_STYLES[variant],
        
        // Size styles
        SIZE_STYLES[size],
        
        // Border radius styles
        BORDER_RADIUS_STYLES[borderRadius],
        
        // State styles
        disabled && "tw-opacity-50 tw-cursor-not-allowed tw-pointer-events-none tw-button-disabled",
        
        // Custom overrides
        className,
    );
};

/**
 * Helper to create ripple style object
 */
export const createRippleStyle = (
    ripple: { x: number; y: number },
    color: string,
) => {
    const fadeColor = color.includes("rgba") 
        ? color.replace(/[\d.]+(?=\))/, "0.1")
        : "rgba(255, 255, 255, 0.1)";
    
    return {
        left: ripple.x - BUTTON_CONFIG.RIPPLE.RADIUS,
        top: ripple.y - BUTTON_CONFIG.RIPPLE.RADIUS,
        width: BUTTON_CONFIG.RIPPLE.RADIUS * 2,
        height: BUTTON_CONFIG.RIPPLE.RADIUS * 2,
        background: `radial-gradient(circle, ${color} 0%, ${fadeColor} 50%, transparent 70%)`,
    };
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
 * Generate custom button styles with proper contrast
 * @param color - Background color in hex format
 * @returns Style object for custom colored button
 */
export const getCustomButtonStyle = (color: string) => ({
    backgroundColor: color,
    color: getContrastTextColor(color),
    borderColor: color,
    // Ensure icons and other child elements inherit the text color
    fill: getContrastTextColor(color),
});

/**
 * Generate focus ring color based on background color
 * Creates a semi-transparent version of the color for focus states
 */
export const getCustomFocusRingColor = (color: string): string => {
    const hex = color.replace("#", "");
    const r = parseInt(hex.substr(0, 2), 16);
    const g = parseInt(hex.substr(2, 2), 16);
    const b = parseInt(hex.substr(4, 2), 16);
    
    // Return semi-transparent version of the color
    return `rgba(${r}, ${g}, ${b}, 0.5)`;
};

/**
 * Style generators for neon variant with glowing effects
 */
// Neon styles are handled entirely by CSS classes
