import { cn } from "../../utils/tailwind-theme.js";

// Slider configuration constants
export const SLIDER_CONFIG = {
    DEFAULT_SIZE: { width: 200, height: 6 },
    THUMB_SIZE: { sm: 16, md: 20, lg: 24 },
    ANIMATION: { 
        DURATION: "0.2s", 
        EASING: "cubic-bezier(0.2, 0, 0, 1)", 
    },
    TOUCH_TARGET: 44, // Minimum touch target size for accessibility
} as const;

// Color schemes for different variants
export const SLIDER_COLORS = {
    TRACK_FILLED: {
        default: "tw-bg-blue-600",
        primary: "tw-bg-blue-600",
        secondary: "tw-bg-gray-600",
        success: "tw-bg-green-600",
        warning: "tw-bg-yellow-600",
        danger: "tw-bg-red-600",
        space: "tw-bg-gradient-to-r tw-from-purple-600 tw-to-blue-600",
        neon: "tw-bg-gradient-to-r tw-from-cyan-400 tw-to-purple-500",
        custom: "",
    },
    TRACK_EMPTY: {
        default: "",
        primary: "",
        secondary: "",
        success: "",
        warning: "",
        danger: "",
        space: "tw-bg-gray-800 dark:tw-bg-gray-900",
        neon: "tw-bg-black dark:tw-bg-gray-900",
        custom: "",
    },
    THUMB: {
        default: "tw-bg-white tw-border-2 tw-border-blue-600",
        primary: "tw-bg-white tw-border-2 tw-border-blue-600",
        secondary: "tw-bg-white tw-border-2 tw-border-gray-600",
        success: "tw-bg-white tw-border-2 tw-border-green-600",
        warning: "tw-bg-white tw-border-2 tw-border-yellow-600",
        danger: "tw-bg-white tw-border-2 tw-border-red-600",
        space: "tw-bg-white tw-border-2 tw-border-purple-400 tw-shadow-lg tw-shadow-purple-500/30",
        neon: "tw-bg-white tw-border-2 tw-border-cyan-400 tw-shadow-lg tw-shadow-cyan-500/50",
        custom: "tw-bg-white tw-border-2",
    },
    FOCUS_RING: {
        default: "focus:tw-ring-2 focus:tw-ring-blue-500/30 focus:tw-ring-offset-2",
        primary: "focus:tw-ring-2 focus:tw-ring-blue-500/30 focus:tw-ring-offset-2",
        secondary: "focus:tw-ring-2 focus:tw-ring-gray-500/30 focus:tw-ring-offset-2",
        success: "focus:tw-ring-2 focus:tw-ring-green-500/30 focus:tw-ring-offset-2",
        warning: "focus:tw-ring-2 focus:tw-ring-yellow-500/30 focus:tw-ring-offset-2",
        danger: "focus:tw-ring-2 focus:tw-ring-red-500/30 focus:tw-ring-offset-2",
        space: "focus:tw-ring-2 focus:tw-ring-purple-500/30 focus:tw-ring-offset-2 focus:tw-ring-offset-gray-800",
        neon: "focus:tw-ring-2 focus:tw-ring-cyan-500/50 focus:tw-ring-offset-2 focus:tw-ring-offset-black",
        custom: "focus:tw-ring-2 focus:tw-ring-blue-500/30 focus:tw-ring-offset-2",
    },
} as const;

// Size configurations
export const SLIDER_SIZES = {
    CONTAINER: {
        sm: "tw-py-2",
        md: "tw-py-3",
        lg: "tw-py-4",
    },
    TRACK: {
        sm: "tw-h-1",
        md: "tw-h-1.5", 
        lg: "tw-h-2",
    },
    THUMB: {
        sm: "tw-w-4 tw-h-4",
        md: "tw-w-5 tw-h-5",
        lg: "tw-w-6 tw-h-6",
    },
} as const;

// Base styles for slider components
export const BASE_SLIDER_STYLES = {
    container: "tw-relative tw-flex tw-items-center tw-cursor-pointer tw-w-full",
    track: cn(
        "tw-relative tw-rounded-full tw-w-full",
        "tw-transition-all tw-duration-200 tw-ease-out",
    ),
    trackFilled: cn(
        "tw-absolute tw-left-0 tw-top-0 tw-h-full tw-rounded-full",
        "tw-transition-all tw-duration-200 tw-ease-out",
        "tw-transform tw-origin-left",
    ),
    trackFilledGradient: cn(
        "tw-absolute tw-left-0 tw-top-0 tw-h-full tw-rounded-full tw-w-full",
        "tw-transition-all tw-duration-200 tw-ease-out",
    ),
    thumb: cn(
        "tw-absolute tw-top-1/2 tw-transform tw--translate-y-1/2 tw--translate-x-1/2",
        "tw-rounded-full tw-cursor-grab tw-z-10",
        "tw-transition-all tw-duration-200 tw-ease-out",
        "tw-transform-gpu tw-will-change-transform",
        "hover:tw-scale-110",
        "active:tw-cursor-grabbing active:tw-scale-95",
    ),
    label: "tw-text-sm tw-font-medium tw-text-gray-700 dark:tw-text-gray-300 tw-mb-2",
    valueDisplay: cn(
        "tw-text-xs tw-font-semibold tw-text-gray-600 dark:tw-text-gray-400",
        "tw-absolute tw--top-8 tw-left-1/2 tw-transform tw--translate-x-1/2",
        "tw-bg-gray-900 tw-text-white tw-px-2 tw-py-1 tw-rounded",
        "tw-opacity-0 tw-transition-opacity tw-duration-200",
        "tw-pointer-events-none tw-z-20",
    ),
    marks: "tw-absolute tw-top-full tw-left-0 tw-w-full tw-flex tw-justify-between tw-mt-2",
    mark: "tw-text-xs tw-text-gray-500 dark:tw-text-gray-400",
    disabled: "tw-opacity-50 tw-cursor-not-allowed",
} as const;

// Build slider container classes
export const buildSliderContainerClasses = ({
    size = "md",
    disabled = false,
    className,
}: {
    size?: "sm" | "md" | "lg";
    disabled?: boolean;
    className?: string;
}) => {
    return cn(
        BASE_SLIDER_STYLES.container,
        SLIDER_SIZES.CONTAINER[size],
        disabled && BASE_SLIDER_STYLES.disabled,
        className,
    );
};

// Build track classes
export const buildTrackClasses = ({
    size = "md",
    variant = "default",
}: {
    size?: "sm" | "md" | "lg";
    variant?: keyof typeof SLIDER_COLORS.TRACK_EMPTY;
}) => {
    return cn(
        BASE_SLIDER_STYLES.track,
        SLIDER_SIZES.TRACK[size],
        SLIDER_COLORS.TRACK_EMPTY[variant],
    );
};

// Get track background style for inline styling
export const getTrackBackgroundStyle = (variant: keyof typeof SLIDER_COLORS.TRACK_EMPTY) => {
    if (variant === "space" || variant === "neon") {
        return {};
    }
    
    return {
        backgroundColor: "#dee2e6", // Light gray for light mode
    };
};

// Build filled track classes
export const buildFilledTrackClasses = ({
    variant = "default",
    customColor,
}: {
    variant?: keyof typeof SLIDER_COLORS.TRACK_FILLED;
    customColor?: string;
}) => {
    if (variant === "custom" && customColor) {
        return cn(BASE_SLIDER_STYLES.trackFilled);
    }
    
    // For gradient variants, use special gradient track
    if (variant === "space" || variant === "neon") {
        return cn(
            BASE_SLIDER_STYLES.trackFilledGradient,
            SLIDER_COLORS.TRACK_FILLED[variant],
        );
    }
    
    return cn(
        BASE_SLIDER_STYLES.trackFilled,
        SLIDER_COLORS.TRACK_FILLED[variant],
    );
};

// Build thumb classes
export const buildThumbClasses = ({
    size = "md",
    variant = "default",
    disabled = false,
    isDragging = false,
    customColor,
}: {
    size?: "sm" | "md" | "lg";
    variant?: keyof typeof SLIDER_COLORS.THUMB;
    disabled?: boolean;
    isDragging?: boolean;
    customColor?: string;
}) => {
    if (variant === "custom" && customColor) {
        return cn(
            BASE_SLIDER_STYLES.thumb,
            SLIDER_SIZES.THUMB[size],
            SLIDER_COLORS.THUMB[variant],
            SLIDER_COLORS.FOCUS_RING.default,
            disabled && BASE_SLIDER_STYLES.disabled,
            isDragging && "tw-scale-110 tw-cursor-grabbing",
        );
    }
    
    return cn(
        BASE_SLIDER_STYLES.thumb,
        SLIDER_SIZES.THUMB[size],
        SLIDER_COLORS.THUMB[variant],
        SLIDER_COLORS.FOCUS_RING[variant],
        disabled && BASE_SLIDER_STYLES.disabled,
        isDragging && "tw-scale-110 tw-cursor-grabbing",
    );
};

// Utility function for calculating slider position
export const calculateSliderPosition = (
    value: number,
    min: number,
    max: number,
): number => {
    if (max === min) return 0;
    return Math.max(0, Math.min(100, ((value - min) / (max - min)) * 100));
};

// Utility function for calculating value from position
export const calculateValueFromPosition = (
    position: number,
    min: number,
    max: number,
    step?: number,
): number => {
    const range = max - min;
    let value = min + (position / 100) * range;
    
    if (step && step > 0) {
        value = Math.round(value / step) * step;
    }
    
    return Math.max(min, Math.min(max, value));
};

// Generate custom styles for custom variant
export const getCustomSliderStyle = (color?: string) => {
    if (!color) return {};
    
    return {
        "--slider-color": color,
        "--slider-color-rgb": color.replace("#", "").match(/.{2}/g)?.map(hex => parseInt(hex, 16)).join(", ") || "59, 130, 246",
    } as React.CSSProperties;
};

// Utility to format values with proper decimal places
export const formatSliderValue = (value: number, step?: number): string => {
    if (!step || step >= 1) {
        return Math.round(value).toString();
    }
    
    // Count decimal places in step
    const stepString = step.toString();
    const decimals = stepString.includes(".") ? stepString.split(".")[1].length : 0;
    
    return Number(value.toFixed(decimals)).toString();
};
