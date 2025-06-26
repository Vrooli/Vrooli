import type { Theme } from "@mui/material";

/**
 * Creates theme-aware Tailwind CSS classes that sync with MUI theme.
 * All classes use the 'tw-' prefix to avoid conflicts during migration.
 */
export function createTailwindThemeClasses(theme: Theme) {
    return {
        // Background colors
        bgDefault: "tw-bg-background-default",
        bgPaper: "tw-bg-background-paper",
        
        // Text colors  
        textPrimary: "tw-text-text-primary",
        textSecondary: "tw-text-text-secondary",
        
        // Dynamic font sizes
        textXs: "tw-text-dynamic-xs",
        textSm: "tw-text-dynamic-sm", 
        textBase: "tw-text-dynamic-base",
        textLg: "tw-text-dynamic-lg",
        textXl: "tw-text-dynamic-xl",
        
        // Common patterns
        buttonPrimary: "tw-bg-secondary-main tw-text-white tw-px-4 tw-py-2 tw-rounded hover:tw-bg-secondary-dark tw-transition-all tw-shadow-md hover:tw-shadow-lg",
        buttonSecondary: "tw-bg-gray-200 tw-text-gray-800 tw-px-4 tw-py-2 tw-rounded hover:tw-bg-gray-300 tw-transition-all tw-shadow-md hover:tw-shadow-lg",
        cardContainer: "tw-bg-background-paper tw-rounded-lg tw-shadow-md tw-p-4",
        
        // Spacing utilities
        spacing: (value: number) => `tw-p-${value}`,
        margin: (value: number) => `tw-m-${value}`,
        padding: (value: number) => `tw-p-${value}`,
        
        // Flexbox utilities
        flexCenter: "tw-flex tw-items-center tw-justify-center",
        flexBetween: "tw-flex tw-items-center tw-justify-between",
        flexColumn: "tw-flex tw-flex-col",
        
        // Border utilities
        border: "tw-border tw-border-gray-300",
        borderRadius: "tw-rounded",
        borderRadiusLg: "tw-rounded-lg",
        
        // Shadow utilities
        shadow: "tw-shadow",
        shadowMd: "tw-shadow-md",
        shadowLg: "tw-shadow-lg",
    };
}

/**
 * Helper function to combine multiple class names, filtering out undefined values
 */
export function cn(...classes: (string | undefined | false)[]) {
    return classes.filter(Boolean).join(" ");
}
