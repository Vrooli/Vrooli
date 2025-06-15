import { cn } from "../../utils/tailwind-theme.js";
import { RadioProps } from "./types.js";

interface RadioStyleOptions {
    color?: RadioProps["color"];
    customColor?: string;
    size?: RadioProps["size"];
    disabled?: boolean;
    checked?: boolean;
    sx?: RadioProps["sx"];
}

const sizeClasses = {
    sm: {
        container: "tw-h-8",
        radioWrapper: "tw-w-6 tw-h-6",
        radioOuter: "tw-w-4 tw-h-4",
        radioInner: "tw-w-3.5 tw-h-3.5",
    },
    md: {
        container: "tw-h-10",
        radioWrapper: "tw-w-8 tw-h-8",
        radioOuter: "tw-w-5 tw-h-5",
        radioInner: "tw-w-4.5 tw-h-4.5",
    },
    lg: {
        container: "tw-h-12",
        radioWrapper: "tw-w-10 tw-h-10",
        radioOuter: "tw-w-6 tw-h-6",
        radioInner: "tw-w-5.5 tw-h-5.5",
    },
};

// Color mapping for consistent styling
const colorStyles = {
    primary: {
        border: "tw-border-blue-600",
        bg: "tw-bg-blue-600",
        hover: "hover:tw-bg-blue-600/10",
        focus: "peer-focus-visible:tw-ring-blue-600/30",
    },
    secondary: {
        border: "tw-border-green-600",
        bg: "tw-bg-green-600",
        hover: "hover:tw-bg-green-600/10",
        focus: "peer-focus-visible:tw-ring-green-600/30",
    },
    danger: {
        border: "tw-border-red-600",
        bg: "tw-bg-red-600",
        hover: "hover:tw-bg-red-600/10",
        focus: "peer-focus-visible:tw-ring-red-600/30",
    },
    success: {
        border: "tw-border-green-500",
        bg: "tw-bg-green-500",
        hover: "hover:tw-bg-green-500/10",
        focus: "peer-focus-visible:tw-ring-green-500/30",
    },
    warning: {
        border: "tw-border-amber-600",
        bg: "tw-bg-amber-600",
        hover: "hover:tw-bg-amber-600/10",
        focus: "peer-focus-visible:tw-ring-amber-600/30",
    },
    info: {
        border: "tw-border-sky-600",
        bg: "tw-bg-sky-600",
        hover: "hover:tw-bg-sky-600/10",
        focus: "peer-focus-visible:tw-ring-sky-600/30",
    },
};

export function getRadioStyles({
    color = "primary",
    customColor,
    size = "md",
    disabled = false,
    checked = false,
    sx,
}: RadioStyleOptions) {
    const sizeStyles = sizeClasses[size];
    const colorStyle = colorStyles[color] || colorStyles.primary;
    
    // Custom color handling
    const isCustom = color === "custom" && customColor;
    const customStyles = isCustom ? {
        borderColor: customColor,
        backgroundColor: customColor,
        hoverBg: `${customColor}1A`, // 10% opacity
        focusRing: `${customColor}4D`, // 30% opacity
    } : null;

    return {
        container: cn(
            "tw-inline-flex tw-items-center tw-cursor-pointer tw-select-none",
            sizeStyles.container,
            disabled && "tw-cursor-not-allowed tw-opacity-50",
        ),
        input: "tw-sr-only tw-peer", // Screen reader only with peer class
        radioWrapper: cn(
            "tw-relative tw-inline-flex tw-items-center tw-justify-center tw-rounded-full tw-transition-colors",
            "tw-p-1", // Add padding to reduce hover area
            !disabled && !isCustom && colorStyle.hover,
            !disabled && "peer-focus-visible:tw-ring-2 peer-focus-visible:tw-ring-offset-2",
            !disabled && !isCustom && colorStyle.focus,
        ),
        radioOuter: cn(
            "tw-relative tw-flex tw-items-center tw-justify-center tw-rounded-full tw-transition-all",
            "tw-border-2 tw-border-solid tw-bg-white",
            sizeStyles.radioOuter,
            disabled ? "tw-border-gray-400" : (checked && !isCustom ? colorStyle.border : "tw-text-text-secondary tw-border-current"),
        ),
        radioInner: cn(
            "tw-absolute tw-rounded-full tw-transition-transform",
            sizeStyles.radioInner,
            checked ? "tw-scale-100" : "tw-scale-0",
            checked && !isCustom && (disabled ? "tw-bg-gray-400" : colorStyle.bg),
        ),
        ripple: cn(
            "tw-absolute tw-rounded-full tw-bg-current tw-opacity-30",
            "tw-pointer-events-none",
            "[animation:radioRipple_0.6s_ease-out_forwards]",
        ),
        // Custom color inline styles
        ...(isCustom && {
            customStyles: {
                outer: {
                    borderColor: checked ? customColor : 'var(--text-secondary)',
                },
                inner: {
                    ...(checked && { backgroundColor: customColor }),
                },
                hover: {
                    backgroundColor: customStyles?.hoverBg,
                },
                focus: {
                    ringColor: customStyles?.focusRing,
                },
            },
        }),
    };
}

// Export all possible Tailwind classes for purge to detect them
export const RADIO_TAILWIND_CLASSES = [
    // Border colors
    "tw-border-blue-600",
    "tw-border-green-600",
    "tw-border-red-600",
    "tw-border-green-500",
    "tw-border-amber-600",
    "tw-border-sky-600",
    "tw-border-gray-400",
    "tw-border-gray-800",
    // Background colors
    "tw-bg-blue-600",
    "tw-bg-green-600",
    "tw-bg-red-600",
    "tw-bg-green-500",
    "tw-bg-amber-600",
    "tw-bg-sky-600",
    "tw-bg-gray-400",
    // Hover states
    "hover:tw-bg-blue-600/10",
    "hover:tw-bg-green-600/10",
    "hover:tw-bg-red-600/10",
    "hover:tw-bg-green-500/10",
    "hover:tw-bg-amber-600/10",
    "hover:tw-bg-sky-600/10",
    // Focus states
    "peer-focus-visible:tw-ring-blue-600/30",
    "peer-focus-visible:tw-ring-green-600/30",
    "peer-focus-visible:tw-ring-red-600/30",
    "peer-focus-visible:tw-ring-green-500/30",
    "peer-focus-visible:tw-ring-amber-600/30",
    "peer-focus-visible:tw-ring-sky-600/30",
];