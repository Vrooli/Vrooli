import { cn } from "../../../utils/tailwind-theme.js";
import type { SwitchVariant, SwitchSize } from "./Switch.js";

/**
 * Switch component styling utilities and configuration
 * Provides consistent styling patterns for Tailwind components
 */

// Configuration object for all switch-related constants
export const SWITCH_CONFIG = {
    // Track dimensions by size
    TRACK_DIMENSIONS: {
        sm: { width: 32, height: 18, padding: 2 },
        md: { width: 44, height: 24, padding: 3 },
        lg: { width: 56, height: 30, padding: 4 },
    },
    
    // Thumb dimensions by size
    THUMB_DIMENSIONS: {
        sm: { size: 14 },
        md: { size: 18 },
        lg: { size: 22 },
    },
    
    // Animation configuration
    ANIMATION: {
        DURATION: "0.2s",
        EASING: "cubic-bezier(0.2, 0, 0, 1)",
    },
} as const;

// Type-safe color system for switch variants
export const SWITCH_COLORS = {
    // Track colors (off state)
    TRACK_OFF: {
        default: "tw-bg-gray-300 dark:tw-bg-gray-600",
        success: "tw-bg-gray-300 dark:tw-bg-gray-600",
        warning: "tw-bg-gray-300 dark:tw-bg-gray-600",
        danger: "tw-bg-gray-300 dark:tw-bg-gray-600",
        space: "tw-bg-gray-800 dark:tw-bg-gray-900",
        neon: "tw-bg-gray-800 dark:tw-bg-gray-900",
        theme: "tw-bg-sky-200",
        custom: "tw-bg-gray-300 dark:tw-bg-gray-600",
    },
    
    // Track colors (on state)
    TRACK_ON: {
        default: "tw-bg-blue-600 dark:tw-bg-blue-500",
        success: "tw-bg-green-600 dark:tw-bg-green-500",
        warning: "tw-bg-orange-500 dark:tw-bg-orange-400",
        danger: "tw-bg-red-600 dark:tw-bg-red-500",
        space: "tw-bg-gradient-to-r tw-from-cyan-500 tw-to-blue-600",
        neon: "tw-bg-green-400",
        theme: "tw-bg-slate-800",
        custom: "tw-bg-purple-600",
    },
    
    // Thumb colors
    THUMB: {
        default: "tw-bg-white tw-border-gray-300 dark:tw-border-gray-500",
        success: "tw-bg-white tw-border-gray-300 dark:tw-border-gray-500",
        warning: "tw-bg-white tw-border-gray-300 dark:tw-border-gray-500",
        danger: "tw-bg-white tw-border-gray-300 dark:tw-border-gray-500",
        space: "tw-bg-white tw-shadow-lg tw-shadow-cyan-500/30",
        neon: "tw-bg-white tw-shadow-lg tw-shadow-green-400/50",
        theme: "tw-bg-transparent",
        custom: "tw-bg-white tw-border-gray-300 dark:tw-border-gray-500",
    },
    
    // Focus ring colors
    FOCUS_RING: {
        default: "focus:tw-ring-blue-500/30",
        success: "focus:tw-ring-green-500/30",
        warning: "focus:tw-ring-orange-500/30",
        danger: "focus:tw-ring-red-500/30",
        space: "focus:tw-ring-cyan-500/30",
        neon: "focus:tw-ring-green-400/30",
        theme: "focus:tw-ring-yellow-500/30",
        custom: "focus:tw-ring-purple-500/30",
    },
} as const;

// Base styles for all switches
export const BASE_SWITCH_STYLES = cn(
    // Layout and positioning
    "tw-relative tw-inline-flex tw-items-center tw-cursor-pointer",
    "tw-transition-all tw-duration-200 tw-ease-in-out",
    
    // Focus styles
    "focus:tw-outline-none focus:tw-ring-2 focus:tw-ring-offset-2",
    "focus:tw-ring-offset-white dark:focus:tw-ring-offset-gray-900",
    
    // Disabled state
    "disabled:tw-opacity-50 disabled:tw-cursor-not-allowed disabled:tw-pointer-events-none",
    
    // Accessibility
    "tw-select-none"
);

// Track styles by variant
export const SWITCH_TRACK_STYLES = {
    default: cn(
        "tw-bg-gray-300 dark:tw-bg-gray-600",
        "data-[checked=true]:tw-bg-blue-600 dark:data-[checked=true]:tw-bg-blue-500",
        "tw-transition-colors tw-duration-200 tw-ease-in-out"
    ),
    success: cn(
        "tw-bg-gray-300 dark:tw-bg-gray-600",
        "data-[checked=true]:tw-bg-green-600 dark:data-[checked=true]:tw-bg-green-500",
        "tw-transition-colors tw-duration-200 tw-ease-in-out"
    ),
    warning: cn(
        "tw-bg-gray-300 dark:tw-bg-gray-600",
        "data-[checked=true]:tw-bg-orange-500 dark:data-[checked=true]:tw-bg-orange-400",
        "tw-transition-colors tw-duration-200 tw-ease-in-out"
    ),
    danger: cn(
        "tw-bg-gray-300 dark:tw-bg-gray-600",
        "data-[checked=true]:tw-bg-red-600 dark:data-[checked=true]:tw-bg-red-500",
        "tw-transition-colors tw-duration-200 tw-ease-in-out"
    ),
    space: cn(
        "tw-bg-gray-800 dark:tw-bg-gray-900 tw-border tw-border-gray-700",
        "data-[checked=true]:tw-bg-gradient-to-r data-[checked=true]:tw-from-cyan-500 data-[checked=true]:tw-to-blue-600",
        "data-[checked=true]:tw-shadow-lg data-[checked=true]:tw-shadow-cyan-500/30",
        "tw-transition-all tw-duration-300 tw-ease-in-out"
    ),
    neon: cn(
        "tw-bg-gray-800 dark:tw-bg-gray-900 tw-border tw-border-gray-700",
        "data-[checked=true]:tw-bg-green-400 data-[checked=true]:tw-border-green-400",
        "data-[checked=true]:tw-shadow-lg data-[checked=true]:tw-shadow-green-400/50",
        "tw-transition-all tw-duration-300 tw-ease-in-out"
    ),
    theme: cn(
        "tw-bg-sky-200 tw-border tw-border-sky-300",
        "data-[checked=true]:tw-bg-slate-800 data-[checked=true]:tw-border-slate-700",
        "tw-transition-all tw-duration-300 tw-ease-in-out"
    ),
    custom: cn(
        "tw-bg-gray-300 dark:tw-bg-gray-600",
        "tw-transition-colors tw-duration-200 tw-ease-in-out"
    ),
} as const;

// Thumb styles by variant
export const SWITCH_THUMB_STYLES = {
    default: cn(
        "tw-bg-white tw-border tw-border-gray-300 dark:tw-border-gray-500",
        "tw-rounded-full tw-shadow-sm",
        "tw-transition-all tw-duration-200 tw-ease-in-out"
    ),
    success: cn(
        "tw-bg-white tw-border tw-border-gray-300 dark:tw-border-gray-500",
        "tw-rounded-full tw-shadow-sm",
        "tw-transition-all tw-duration-200 tw-ease-in-out"
    ),
    warning: cn(
        "tw-bg-white tw-border tw-border-gray-300 dark:tw-border-gray-500",
        "tw-rounded-full tw-shadow-sm",
        "tw-transition-all tw-duration-200 tw-ease-in-out"
    ),
    danger: cn(
        "tw-bg-white tw-border tw-border-gray-300 dark:tw-border-gray-500",
        "tw-rounded-full tw-shadow-sm",
        "tw-transition-all tw-duration-200 tw-ease-in-out"
    ),
    space: cn(
        "tw-bg-white tw-border tw-border-gray-200",
        "tw-rounded-full tw-shadow-lg tw-shadow-cyan-500/20",
        "data-[checked=true]:tw-shadow-cyan-500/40",
        "tw-transition-all tw-duration-300 tw-ease-in-out"
    ),
    neon: cn(
        "tw-bg-white tw-border tw-border-gray-200",
        "tw-rounded-full tw-shadow-lg tw-shadow-green-400/20",
        "data-[checked=true]:tw-shadow-green-400/60 data-[checked=true]:tw-border-green-300",
        "tw-transition-all tw-duration-300 tw-ease-in-out"
    ),
    theme: cn(
        "tw-bg-transparent tw-border-0",
        "tw-rounded-full",
        "tw-transition-all tw-duration-300 tw-ease-in-out"
    ),
    custom: cn(
        "tw-bg-white tw-border tw-border-gray-300 dark:tw-border-gray-500",
        "tw-rounded-full tw-shadow-sm",
        "tw-transition-all tw-duration-200 tw-ease-in-out"
    ),
} as const;

// Label styles
export const SWITCH_LABEL_STYLES = cn(
    "tw-text-sm tw-font-medium tw-text-gray-700 dark:tw-text-gray-300",
    "tw-cursor-pointer tw-select-none"
);

/**
 * Utility function to get track dimensions for a given size
 */
export function getTrackDimensions(size: SwitchSize) {
    return SWITCH_CONFIG.TRACK_DIMENSIONS[size];
}

/**
 * Utility function to get thumb dimensions for a given size
 */
export function getThumbDimensions(size: SwitchSize) {
    return SWITCH_CONFIG.THUMB_DIMENSIONS[size];
}

/**
 * Utility function to calculate thumb position based on checked state and size
 */
export function getThumbPosition(checked: boolean, size: SwitchSize) {
    const { width, padding } = getTrackDimensions(size);
    const { size: thumbSize } = getThumbDimensions(size);
    
    if (checked) {
        return width - thumbSize - padding;
    }
    return padding;
}

/**
 * Utility function to build switch classes based on props
 */
export function buildSwitchClasses({
    variant,
    size,
    disabled,
    className,
}: {
    variant: SwitchVariant;
    size: SwitchSize;
    disabled: boolean;
    className?: string;
}) {
    return cn(
        BASE_SWITCH_STYLES,
        SWITCH_COLORS.FOCUS_RING[variant],
        disabled && "tw-opacity-50 tw-cursor-not-allowed",
        className
    );
}

/**
 * Utility function to get custom switch style for the custom variant
 */
export function getCustomSwitchStyle(color: string) {
    return {
        '--switch-custom-color': color,
        '--switch-custom-color-light': `${color}20`,
    } as React.CSSProperties;
}

/**
 * Utility function to get focus ring color for a variant
 */
export function getFocusRingColor(variant: SwitchVariant) {
    return SWITCH_COLORS.FOCUS_RING[variant];
}