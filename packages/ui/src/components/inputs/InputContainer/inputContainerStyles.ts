import { cn } from "../../../utils/tailwind-theme.js";
import type { InputVariant, InputSize } from "./types.js";

export const INPUT_CONTAINER_CONFIG = {
    HEIGHTS: {
        sm: "tw-h-8",
        md: "tw-h-10", 
        lg: "tw-h-12",
    },
    FONT_SIZES: {
        sm: "tw-text-sm",
        md: "tw-text-base",
        lg: "tw-text-lg",
    },
    PADDING: {
        // Note: horizontal padding is handled by adornments
        sm: "tw-py-1",
        md: "tw-py-2",
        lg: "tw-py-3",
    },
} as const;

export const BASE_INPUT_CONTAINER_STYLES = cn(
    "tw-w-full",
    "tw-transition-all tw-duration-200 tw-ease-in-out",
    "tw-outline-none",
    "tw-flex tw-items-center",
    "tw-relative",
    "tw-overflow-hidden",
    "tw-min-h-0", // Prevent flex item from growing
    // Better disabled states
    "disabled:tw-cursor-not-allowed",
    "disabled:tw-opacity-60",
    // Focus management
    "focus-within:tw-z-10", // Bring focused inputs above others
);

export const VARIANT_STYLES: Record<InputVariant, string> = {
    outline: cn(
        "tw-border tw-rounded-md",
        "tw-bg-background-paper",
        "tw-border-text-secondary/30", // More semantic border color
        "hover:tw-border-text-secondary/50",
        "focus-within:tw-border-primary-main",
        "focus-within:tw-ring-2 focus-within:tw-ring-offset-2",
        "focus-within:tw-ring-primary-main/20", // Softer ring
        "tw-text-text-primary",
        // Disabled states
        "disabled:tw-bg-background-default",
        "disabled:tw-border-text-secondary/20",
        "disabled:tw-text-text-secondary",
    ),
    filled: cn(
        "tw-border tw-rounded-md",
        "tw-bg-text-secondary/10", // Use theme-aware background
        "tw-border-transparent",
        "hover:tw-bg-text-secondary/15",
        "focus-within:tw-bg-background-paper",
        "focus-within:tw-border-primary-main",
        "focus-within:tw-ring-2 focus-within:tw-ring-offset-2",
        "focus-within:tw-ring-primary-main/20", // Softer ring
        "tw-text-text-primary",
        // Disabled states
        "disabled:tw-bg-text-secondary/5",
        "disabled:tw-border-transparent",
        "disabled:tw-text-text-secondary",
    ),
    underline: cn(
        "tw-bg-transparent",
        "tw-border-0 tw-border-b-2",
        "tw-border-text-secondary/40", // More semantic border color
        "tw-rounded-none",
        "hover:tw-border-text-secondary/60",
        "focus-within:tw-border-primary-main",
        "focus-within:tw-ring-0",
        "focus-within:tw-rounded-t-md", // Add top border radius when focused
        "tw-text-text-primary",
        // Disabled states
        "disabled:tw-border-text-secondary/20",
        "disabled:tw-text-text-secondary",
    ),
};

// Padding adjustments based on adornments
export const ADORNMENT_PADDING = {
    start: {
        sm: "tw-pl-3",
        md: "tw-pl-3",
        lg: "tw-pl-4",
    },
    end: {
        sm: "tw-pr-3",
        md: "tw-pr-3",
        lg: "tw-pr-4",
    },
} as const;

export const SIZE_STYLES: Record<InputSize, string> = {
    sm: cn(INPUT_CONTAINER_CONFIG.HEIGHTS.sm, INPUT_CONTAINER_CONFIG.FONT_SIZES.sm, INPUT_CONTAINER_CONFIG.PADDING.sm),
    md: cn(INPUT_CONTAINER_CONFIG.HEIGHTS.md, INPUT_CONTAINER_CONFIG.FONT_SIZES.md, INPUT_CONTAINER_CONFIG.PADDING.md),
    lg: cn(INPUT_CONTAINER_CONFIG.HEIGHTS.lg, INPUT_CONTAINER_CONFIG.FONT_SIZES.lg, INPUT_CONTAINER_CONFIG.PADDING.lg),
};

export const ERROR_STYLES = cn(
    "tw-border-danger-main",
    "focus-within:tw-border-danger-main",
    "focus-within:tw-ring-danger-main/20", // Softer error ring
    "hover:tw-border-danger-main",
);

export const FOCUSED_STYLES: Record<InputVariant, string> = {
    outline: cn(
        "tw-border-primary-main",
        "tw-ring-2 tw-ring-offset-2",
        "tw-ring-primary-main/20",
    ),
    filled: cn(
        "tw-bg-background-paper",
        "tw-border-primary-main",
        "tw-ring-2 tw-ring-offset-2",
        "tw-ring-primary-main/20",
    ),
    underline: cn(
        "tw-border-primary-main",
        "tw-rounded-t-md", // Only top radius for underline
    ),
};

// Disabled state overrides for all variants
export const DISABLED_STYLES = cn(
    "disabled:tw-cursor-not-allowed",
    "disabled:tw-opacity-60",
    "disabled:hover:tw-border-text-secondary/20", // Override hover states when disabled
    "disabled:focus-within:tw-ring-0", // Remove focus ring when disabled
    "disabled:focus-within:tw-border-text-secondary/20", // Override focus border when disabled
);

export function buildInputContainerClasses({
    variant,
    size,
    error,
    disabled,
    fullWidth,
    focused,
    hasStartAdornment,
    hasEndAdornment,
}: {
    variant: InputVariant;
    size: InputSize;
    error?: boolean;
    disabled?: boolean;
    fullWidth?: boolean;
    focused?: boolean;
    hasStartAdornment?: boolean;
    hasEndAdornment?: boolean;
}) {
    return cn(
        BASE_INPUT_CONTAINER_STYLES,
        VARIANT_STYLES[variant],
        SIZE_STYLES[size],
        error && !disabled && ERROR_STYLES, // Don't show error styles when disabled
        focused && !disabled && FOCUSED_STYLES[variant], // Don't show focus styles when disabled
        disabled && DISABLED_STYLES,
        // Add padding based on adornments
        !hasStartAdornment && ADORNMENT_PADDING.start[size],
        !hasEndAdornment && ADORNMENT_PADDING.end[size],
    );
}
