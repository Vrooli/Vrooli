import { cn } from "../../utils/tailwind-theme.js";

export const BOTTOM_NAV_CONFIG = {
    HEIGHT: "48px", // Based on clickSize.minHeight
    Z_INDEX: 50,
    TRANSITION_DURATION: 200,
} as const;

export const bottomNavClasses = cn(
    // Base layout
    "tw-fixed tw-bottom-0 tw-w-full",
    "tw-flex tw-items-center tw-justify-around",
    // Height and padding
    "tw-h-12", // 48px
    "tw-pb-[env(safe-area-inset-bottom)]",
    "tw-pl-[calc(4px+env(safe-area-inset-left))]",
    "tw-pr-[calc(4px+env(safe-area-inset-right))]",
    // Colors - using CSS variables for theming
    "tw-bg-[var(--primary-dark)]",
    // Z-index
    "tw-z-50",
    // Responsive - only show on mobile
    "tw-flex md:tw-hidden",
);

export const navActionClasses = cn(
    // Layout
    "tw-flex tw-flex-col tw-items-center tw-justify-center",
    "tw-flex-1",
    // Sizing
    "tw-min-w-[58px]", // Matches original minWidth
    "tw-h-full",
    "tw-py-1",
    // Typography
    "tw-text-xs tw-font-medium",
    "tw-no-underline", // Remove link underline
    // Colors - white text on primary dark background
    "tw-text-white",
    // Transitions
    "tw-transition-all tw-duration-200",
    // States
    "hover:tw-bg-white/10",
    "active:tw-bg-white/20",
    // Focus
    "focus:tw-outline-none",
    "focus:tw-ring-2 focus:tw-ring-inset focus:tw-ring-white/30",
    // Disable text selection
    "tw-select-none",
    // Cursor
    "tw-cursor-pointer",
);

export const navActionLabelClasses = cn(
    "tw-mt-1",
    "tw-text-[10px]",
    "tw-leading-tight",
);

export const badgeWrapperClasses = cn(
    "tw-relative",
    "tw-inline-flex",
);

export const badgeClasses = cn(
    // Position
    "tw-absolute tw-top-0 tw-right-0",
    "tw-translate-x-1/2 -tw-translate-y-1/2",
    // Size
    "tw-min-w-[16px] tw-h-4",
    "tw-px-1",
    // Shape
    "tw-rounded-full",
    // Layout
    "tw-flex tw-items-center tw-justify-center",
    // Typography
    "tw-text-[10px] tw-font-medium",
    // Colors - using error/danger colors from CSS variables
    "tw-bg-[var(--danger-main)] tw-text-white",
);