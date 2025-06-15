import { cn } from "../../utils/tailwind-theme.js";
import { CheckboxProps } from "./types.js";

interface CheckboxStyleOptions {
    color?: CheckboxProps["color"];
    customColor?: string;
    size?: CheckboxProps["size"];
    disabled?: boolean;
    checked?: boolean;
    indeterminate?: boolean;
    sx?: CheckboxProps["sx"];
}

const sizeClasses = {
    sm: {
        container: "tw-h-8",
        checkboxWrapper: "tw-w-8 tw-h-8",
        checkboxBox: "tw-w-4 tw-h-4",
        checkmark: "tw-w-3 tw-h-3",
    },
    md: {
        container: "tw-h-10",
        checkboxWrapper: "tw-w-10 tw-h-10",
        checkboxBox: "tw-w-5 tw-h-5",
        checkmark: "tw-w-4 tw-h-4",
    },
    lg: {
        container: "tw-h-12",
        checkboxWrapper: "tw-w-12 tw-h-12",
        checkboxBox: "tw-w-6 tw-h-6",
        checkmark: "tw-w-5 tw-h-5",
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

export function getCheckboxStyles({
    color = "primary",
    customColor,
    size = "md",
    disabled = false,
    checked = false,
    indeterminate = false,
    sx,
}: CheckboxStyleOptions) {
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
        checkboxWrapper: cn(
            "tw-relative tw-inline-flex tw-items-center tw-justify-center tw-rounded-md tw-transition-colors",
            sizeStyles.checkboxWrapper,
            !disabled && !isCustom && colorStyle.hover,
            !disabled && "peer-focus-visible:tw-ring-2 peer-focus-visible:tw-ring-offset-2",
            !disabled && !isCustom && colorStyle.focus,
        ),
        checkboxBox: cn(
            "tw-relative tw-flex tw-items-center tw-justify-center tw-rounded tw-border-2 tw-border-solid tw-transition-all",
            sizeStyles.checkboxBox,
            // Background color - white by default, colored when checked
            (checked || indeterminate) ? (!isCustom && (disabled ? "tw-bg-gray-400" : colorStyle.bg)) : "tw-bg-white",
            // Border color
            disabled ? "tw-border-gray-400" : ((checked || indeterminate) && !isCustom ? colorStyle.border : "tw-text-text-secondary tw-border-current"),
        ),
        checkmark: cn(
            "tw-absolute tw-text-white tw-transition-transform",
            sizeStyles.checkmark,
            (checked || indeterminate) ? "tw-scale-100" : "tw-scale-0",
        ),
        ripple: cn(
            "tw-absolute tw-rounded-full tw-bg-current tw-opacity-30",
            "tw-pointer-events-none",
            "[animation:radioRipple_0.6s_ease-out_forwards]",
        ),
        // Custom color inline styles
        ...(isCustom && {
            customStyles: {
                box: {
                    borderColor: (checked || indeterminate) ? customColor : 'var(--text-secondary)',
                    ...((checked || indeterminate) && { 
                        backgroundColor: customColor,
                    }),
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