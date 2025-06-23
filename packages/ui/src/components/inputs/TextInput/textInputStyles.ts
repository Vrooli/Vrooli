import { cn } from "../../../utils/tailwind-theme.js";
import type { TextInputVariant, TextInputSize } from "./types.js";

export const TEXT_INPUT_CONFIG = {
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
        sm: "tw-px-3 tw-py-1",
        md: "tw-px-3 tw-py-2",
        lg: "tw-px-4 tw-py-3",
    },
} as const;

export const BASE_TEXT_INPUT_STYLES = cn(
    "tw-w-full",
    "tw-transition-all tw-duration-200",
    "tw-outline-none",
    "disabled:tw-cursor-not-allowed disabled:tw-opacity-50",
    "placeholder:tw-text-text-secondary",
    "tw-resize-none"  // For textarea elements
);

export const VARIANT_STYLES: Record<TextInputVariant, string> = {
    outline: cn(
        "tw-border tw-rounded-md",
        "tw-bg-background-paper",
        "tw-border-gray-400",
        "hover:tw-border-gray-500",
        "focus:tw-border-primary-main focus-within:tw-border-primary-main",
        "focus:tw-ring-2 focus:tw-ring-offset-2 focus-within:tw-ring-2 focus-within:tw-ring-offset-2",
        "focus:tw-ring-primary-main focus-within:tw-ring-primary-main",
        "tw-text-text-primary"
    ),
    filled: cn(
        "tw-border tw-rounded-md",
        "tw-bg-gray-200",
        "tw-border-transparent",
        "hover:tw-bg-gray-300",
        "focus:tw-bg-background-paper focus-within:tw-bg-background-paper",
        "focus:tw-border-primary-main focus-within:tw-border-primary-main",
        "focus:tw-ring-2 focus:tw-ring-offset-2 focus-within:tw-ring-2 focus-within:tw-ring-offset-2",
        "focus:tw-ring-primary-main focus-within:tw-ring-primary-main",
        "tw-text-text-primary"
    ),
    underline: cn(
        "tw-bg-transparent",
        "tw-border-0 tw-border-b-2",
        "tw-border-gray-400",
        "tw-rounded-none",
        "hover:tw-border-gray-500",
        "focus:tw-border-primary-main focus-within:tw-border-primary-main",
        "focus:tw-ring-0 focus-within:tw-ring-0",
        "focus:tw-rounded-md focus-within:tw-rounded-md", // Add border radius when focused
        "tw-text-text-primary"
    ),
};

export const SIZE_STYLES: Record<TextInputSize, string> = {
    sm: cn(TEXT_INPUT_CONFIG.HEIGHTS.sm, TEXT_INPUT_CONFIG.FONT_SIZES.sm, TEXT_INPUT_CONFIG.PADDING.sm),
    md: cn(TEXT_INPUT_CONFIG.HEIGHTS.md, TEXT_INPUT_CONFIG.FONT_SIZES.md, TEXT_INPUT_CONFIG.PADDING.md),
    lg: cn(TEXT_INPUT_CONFIG.HEIGHTS.lg, TEXT_INPUT_CONFIG.FONT_SIZES.lg, TEXT_INPUT_CONFIG.PADDING.lg),
};

export const ERROR_STYLES = cn(
    "tw-border-danger-main",
    "focus:tw-border-danger-main focus-within:tw-border-danger-main",
    "focus:tw-ring-danger-main focus-within:tw-ring-danger-main"
);

export const FULL_WIDTH_STYLES = "tw-w-full";

export function buildTextInputClasses({
    variant,
    size,
    error,
    disabled,
    fullWidth,
    hasAdornments,
}: {
    variant: TextInputVariant;
    size: TextInputSize;
    error?: boolean;
    disabled?: boolean;
    fullWidth?: boolean;
    hasAdornments?: boolean;
}) {
    return cn(
        BASE_TEXT_INPUT_STYLES,
        VARIANT_STYLES[variant],
        SIZE_STYLES[size],
        error && ERROR_STYLES,
        fullWidth && FULL_WIDTH_STYLES
    );
}

// Build container classes (same as input but without padding)
export function buildContainerClasses({
    variant,
    size,
    error,
    disabled,
    fullWidth,
}: {
    variant: TextInputVariant;
    size: TextInputSize;
    error?: boolean;
    disabled?: boolean;
    fullWidth?: boolean;
}) {
    return cn(
        BASE_TEXT_INPUT_STYLES,
        VARIANT_STYLES[variant],
        // Only height, not padding
        size === "sm" && TEXT_INPUT_CONFIG.HEIGHTS.sm,
        size === "md" && TEXT_INPUT_CONFIG.HEIGHTS.md,
        size === "lg" && TEXT_INPUT_CONFIG.HEIGHTS.lg,
        error && ERROR_STYLES,
        fullWidth && FULL_WIDTH_STYLES,
        "tw-p-0" // Override all padding since adornments handle their own
    );
}