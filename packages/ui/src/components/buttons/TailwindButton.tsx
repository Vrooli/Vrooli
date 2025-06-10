import { forwardRef } from "react";
import type { ButtonHTMLAttributes, ReactNode } from "react";
import CircularProgress from "@mui/material/CircularProgress";
import { cn } from "../../utils/tailwind-theme.js";

// Define variant types
export type ButtonVariant = "primary" | "secondary" | "outline" | "ghost" | "danger";
export type ButtonSize = "sm" | "md" | "lg" | "icon";

export interface TailwindButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
    variant?: ButtonVariant;
    size?: ButtonSize;
    isLoading?: boolean;
    startIcon?: ReactNode;
    endIcon?: ReactNode;
    fullWidth?: boolean;
    children?: ReactNode;
}

// Variant styles mapping
const variantStyles: Record<ButtonVariant, string> = {
    primary: "tw-bg-secondary-main tw-text-white hover:tw-bg-secondary-dark focus:tw-ring-2 focus:tw-ring-secondary-main focus:tw-ring-offset-2 tw-shadow-md hover:tw-shadow-lg",
    secondary: "tw-bg-gray-200 tw-text-gray-800 hover:tw-bg-gray-300 focus:tw-ring-2 focus:tw-ring-gray-400 focus:tw-ring-offset-2 tw-shadow-sm hover:tw-shadow-md",
    outline: "tw-border tw-border-secondary-main tw-text-secondary-main hover:tw-bg-secondary-main hover:tw-text-white focus:tw-ring-2 focus:tw-ring-secondary-main focus:tw-ring-offset-2",
    ghost: "tw-text-secondary-main hover:tw-bg-secondary-main hover:tw-bg-opacity-10 focus:tw-ring-2 focus:tw-ring-secondary-main focus:tw-ring-offset-2",
    danger: "tw-bg-red-600 tw-text-white hover:tw-bg-red-700 focus:tw-ring-2 focus:tw-ring-red-600 focus:tw-ring-offset-2 tw-shadow-md hover:tw-shadow-lg",
};

// Size styles mapping
const sizeStyles: Record<ButtonSize, string> = {
    sm: "tw-h-8 tw-px-3 tw-text-xs",
    md: "tw-h-10 tw-px-4 tw-text-sm",
    lg: "tw-h-12 tw-px-6 tw-text-base",
    icon: "tw-h-10 tw-w-10 tw-p-0",
};

// Loading spinner sizes
const spinnerSizes: Record<ButtonSize, number> = {
    sm: 16,
    md: 20,
    lg: 24,
    icon: 20,
};

/**
 * A performant, accessible Tailwind CSS button component with multiple variants and sizes.
 * Supports loading states, icons, and full keyboard/screen reader support.
 */
export const TailwindButton = forwardRef<HTMLButtonElement, TailwindButtonProps>(
    (
        {
            variant = "primary",
            size = "md",
            isLoading = false,
            startIcon,
            endIcon,
            fullWidth = false,
            disabled = false,
            className,
            children,
            type = "button",
            ...props
        },
        ref
    ) => {
        // Determine if button should be disabled (either explicitly or when loading)
        const isDisabled = disabled || isLoading;

        // Build the complete className
        const buttonClasses = cn(
            // Base styles - always applied
            "tw-inline-flex tw-items-center tw-justify-center tw-gap-2",
            "tw-font-sans tw-font-medium tw-tracking-wider tw-uppercase",
            "tw-rounded tw-transition-all tw-duration-200",
            "tw-border-0 tw-outline-none",
            "focus:tw-ring-offset-background",
            
            // Width styles
            fullWidth ? "tw-w-full" : "tw-w-auto",
            
            // Variant styles
            variantStyles[variant],
            
            // Size styles
            sizeStyles[size],
            
            // Disabled/loading styles
            isDisabled && "tw-opacity-50 tw-cursor-not-allowed tw-pointer-events-none",
            
            // Custom className (allows overrides)
            className
        );

        // Render loading spinner or start icon
        const renderStartElement = () => {
            if (isLoading) {
                return (
                    <CircularProgress 
                        size={spinnerSizes[size]} 
                        sx={{ color: variant === "outline" || variant === "ghost" ? "currentColor" : "inherit" }}
                    />
                );
            }
            return startIcon || null;
        };

        return (
            <button
                ref={ref}
                type={type}
                disabled={isDisabled}
                className={buttonClasses}
                aria-busy={isLoading}
                aria-disabled={isDisabled}
                {...props}
            >
                {renderStartElement()}
                {children && <span>{children}</span>}
                {!isLoading && endIcon}
            </button>
        );
    }
);

TailwindButton.displayName = "TailwindButton";

// Export a typed component factory for common use cases
export const Button = {
    Primary: (props: Omit<TailwindButtonProps, "variant">) => (
        <TailwindButton variant="primary" {...props} />
    ),
    Secondary: (props: Omit<TailwindButtonProps, "variant">) => (
        <TailwindButton variant="secondary" {...props} />
    ),
    Outline: (props: Omit<TailwindButtonProps, "variant">) => (
        <TailwindButton variant="outline" {...props} />
    ),
    Ghost: (props: Omit<TailwindButtonProps, "variant">) => (
        <TailwindButton variant="ghost" {...props} />
    ),
    Danger: (props: Omit<TailwindButtonProps, "variant">) => (
        <TailwindButton variant="danger" {...props} />
    ),
};