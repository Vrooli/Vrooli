import { cn } from "../../../utils/tailwind-theme.js";
import type { DialogPosition, DialogSize, DialogVariant } from "./Dialog.js";

/**
 * Dialog component styling utilities and configuration
 * Provides consistent styling patterns for Tailwind dialog components
 */

// Configuration object for dialog-related constants
export const DIALOG_CONFIG = {
    // Animation durations
    ANIMATION: {
        OVERLAY_DURATION: "tw-duration-200",
        CONTENT_DURATION: "tw-duration-300",
    },

    // Z-index values
    Z_INDEX: {
        OVERLAY: 50,
        CONTENT: 50,
    },

    // Special effects configuration
    EFFECTS: {
        SPACE_STAR_COUNT: 100,
        NEON_BLUR_AMOUNT: 20,
    },
} as const;

// Dialog size dimensions
export const DIALOG_SIZES = {
    sm: "tw-max-w-sm tw-w-[calc(100vw-2rem)]",
    md: "tw-max-w-md tw-w-[calc(100vw-2rem)]",
    lg: "tw-max-w-lg tw-w-[calc(100vw-2rem)]",
    xl: "tw-max-w-xl tw-w-[calc(100vw-2rem)]",
    full: "tw-max-w-none tw-w-full tw-h-full tw-m-0",
} as const;

// Dialog position styles
export const DIALOG_POSITIONS = {
    center: "tw-items-center tw-justify-center tw-min-h-full",
    top: "tw-items-start tw-justify-center tw-pt-16",
    bottom: "tw-items-end tw-justify-center tw-pb-16",
    left: "tw-items-center tw-justify-start tw-pl-16 tw-min-h-full",
    right: "tw-items-center tw-justify-end tw-pr-16 tw-min-h-full",
} as const;

// Variant-specific styles
export const DIALOG_STYLES = {
    overlay: {
        base: cn(
            "tw-fixed tw-inset-0",
            "tw-flex",
            "tw-overflow-y-auto", // Allow scrolling if dialog is taller than viewport
            "tw-transition-opacity",
            DIALOG_CONFIG.ANIMATION.OVERLAY_DURATION,
            "tw-z-50",
        ),
    },

    dialog: {
        base: cn(
            "tw-relative",
            "tw-w-full",
            "tw-transition-all",
            DIALOG_CONFIG.ANIMATION.CONTENT_DURATION,
            "tw-transform",
            "tw-scale-100 tw-opacity-100",
            "data-[state=closed]:tw-scale-95 data-[state=closed]:tw-opacity-0",
        ),
    },

    content: {
        base: cn(
            "tw-relative",
            "tw-bg-background-paper",
            "tw-rounded-2xl", // Increased border radius
            "tw-shadow-xl",
            "tw-overflow-hidden",
            "tw-flex tw-flex-col",
            "tw-border tw-border-gray-200 dark:tw-border-gray-700",
        ),

        variants: {
            default: "",
            danger: "",
            success: "",
            space: cn(
                "tw-border tw-border-cyan-400/30",
                "tw-bg-black", // Black background to see stars
                "tw-text-white", // White text for space theme
                "[&_*]:tw-text-white", // All child elements get white text
                "tw-shadow-2xl tw-shadow-cyan-500/20",
                "tw-relative tw-overflow-hidden",
                "[&_.tw-dialog-space-stars]:tw-opacity-100", // Ensure stars are visible
            ),

            neon: cn(
                "tw-border-2 tw-border-green-400/50",
                "tw-bg-black",
                "tw-text-white", // White text for neon theme
                "[&_*]:tw-text-white", // All child elements get white text
                "tw-shadow-2xl",
                "tw-relative tw-overflow-hidden",
                "[box-shadow:0_0_30px_rgba(0,255,127,0.5),inset_0_0_20px_rgba(0,255,127,0.1)]",
            ),
        },
    },

    title: {
        base: cn(
            "tw-text-lg tw-font-semibold",
            "tw-text-text-primary",
        ),
    },

    actions: {
        base: cn(
            "tw-flex tw-items-center tw-justify-end",
            "tw-gap-2",
            "tw-px-6 tw-py-4",
            "tw-border-t tw-border-gray-200 dark:tw-border-gray-700",
        ),
        variants: {
            default: "tw-bg-background-paper",
            danger: "tw-bg-background-paper",
            success: "tw-bg-background-paper",
            space: "tw-bg-transparent tw-border-t-cyan-400/30",
            neon: "tw-bg-transparent tw-border-t-green-400/30",
        },
    },
} as const;

// Build overlay classes
export function buildOverlayClasses(enableBackgroundBlur: boolean, className?: string) {
    return cn(
        DIALOG_STYLES.overlay.base,
        enableBackgroundBlur ? "tw-bg-black/50 tw-backdrop-blur-sm" : "tw-bg-transparent",
        className,
    );
}

// Build dialog classes based on props
interface BuildDialogClassesProps {
    variant: DialogVariant;
    size: DialogSize;
    position: DialogPosition;
    className?: string;
}

export function buildDialogClasses({
    size,
    position,
    className,
}: BuildDialogClassesProps) {
    const positionClasses = DIALOG_POSITIONS[position];
    return cn(
        positionClasses,
        className,
    );
}

// Build content classes based on props
interface BuildContentClassesProps {
    variant: DialogVariant;
    className?: string;
}

export function buildContentClasses({
    variant,
    className,
}: BuildContentClassesProps) {
    return cn(
        DIALOG_STYLES.content.base,
        DIALOG_STYLES.content.variants[variant],
        className,
    );
}

// Build title classes
export function buildTitleClasses(className?: string) {
    return cn(DIALOG_STYLES.title.base, className);
}

// Build actions classes
export function buildActionsClasses(variant: DialogVariant, className?: string) {
    return cn(
        DIALOG_STYLES.actions.base,
        DIALOG_STYLES.actions.variants[variant],
        className,
    );
}

// Helper to get dialog wrapper classes (includes size)
export function getDialogWrapperClasses(size: DialogSize, draggable?: boolean, isDragging?: boolean) {
    return cn(
        DIALOG_STYLES.dialog.base,
        DIALOG_SIZES[size],
        size !== "full" && !draggable && "tw-my-8 tw-max-h-[calc(100vh-4rem)]", // Ensure dialog fits in viewport with margin
        draggable && "tw-max-h-[calc(100vh-2rem)] tw-dialog-draggable", // Smaller margin for draggable dialogs + performance class
        isDragging && "tw-dialog-dragging", // Performance class for dragging state
    );
}
