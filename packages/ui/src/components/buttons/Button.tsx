import type { ButtonHTMLAttributes, MouseEvent, ReactNode } from "react";
import { forwardRef, useCallback, useState } from "react";
import { cn } from "../../utils/tailwind-theme.js";
import { CircularProgress, OrbitalSpinner } from "../indicators/CircularProgress.js";

// Define variant types
export type ButtonVariant = "primary" | "secondary" | "outline" | "ghost" | "danger" | "space";
export type ButtonSize = "sm" | "md" | "lg" | "icon";

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
    variant?: ButtonVariant;
    size?: ButtonSize;
    isLoading?: boolean;
    loadingIndicator?: "circular" | "orbital";
    startIcon?: ReactNode;
    endIcon?: ReactNode;
    fullWidth?: boolean;
    children?: ReactNode;
}

// Variant styles mapping
const variantStyles: Record<ButtonVariant, string> = {
    primary: "tw-bg-secondary-main tw-text-white hover:tw-bg-secondary-dark focus:tw-ring-2 focus:tw-ring-secondary-main focus:tw-ring-offset-2 tw-shadow-md hover:tw-shadow-lg",
    secondary: "tw-bg-gray-200 tw-text-gray-800 hover:tw-bg-gray-300 focus:tw-ring-2 focus:tw-ring-gray-400 focus:tw-ring-offset-2 tw-shadow-sm hover:tw-shadow-md",
    outline: "tw-bg-transparent tw-border tw-border-secondary-main tw-text-secondary-main hover:tw-bg-secondary-main hover:tw-text-white focus:tw-ring-2 focus:tw-ring-secondary-main focus:tw-ring-offset-2",
    ghost: "tw-bg-transparent tw-text-secondary-main hover:tw-bg-secondary-main hover:tw-bg-opacity-10 focus:tw-ring-2 focus:tw-ring-secondary-main focus:tw-ring-offset-2",
    danger: "tw-bg-red-600 tw-text-white hover:tw-bg-red-700 focus:tw-ring-2 focus:tw-ring-red-600 focus:tw-ring-offset-2 tw-shadow-md hover:tw-shadow-lg",
    /* ✦ space variant: gradient border + inner padding */
    space: cn(
        "tw-relative tw-overflow-hidden tw-text-white",
        "tw-rounded-md tw-p-[2px]",
        "tw-bg-gradient-to-br tw-from-[#1a3a4a] tw-via-[#2a4a6a] tw-to-[#1a3a4a]",
        "tw-transition-all tw-duration-300 tw-shadow-lg hover:tw-shadow-cyan-400/25"
    ),
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

// Ripple effect interface
interface Ripple {
    id: number;
    x: number;
    y: number;
}

// Ripple colors for each variant
const rippleColors: Record<ButtonVariant, string> = {
    primary: "rgba(255, 255, 255, 0.5)",
    secondary: "rgba(0, 0, 0, 0.3)",
    outline: "rgba(22, 163, 97, 0.3)",
    ghost: "rgba(22, 163, 97, 0.3)",
    danger: "rgba(255, 255, 255, 0.5)",
    space: "rgba(15, 170, 170, 0.8)",
};

// Generic Ripple component
const RippleEffect = ({
    ripples,
    onRippleComplete,
    color
}: {
    ripples: Ripple[];
    onRippleComplete: (id: number) => void;
    color: string;
}) => {
    return (
        <>
            {ripples.map((ripple) => (
                <div
                    key={ripple.id}
                    className="tw-absolute tw-pointer-events-none tw-rounded-full"
                    style={{
                        left: ripple.x - 25,
                        top: ripple.y - 25,
                        width: 50,
                        height: 50,
                        background: `radial-gradient(circle, ${color} 0%, ${color.replace(/[\d.]+\)/, '0.1)')} 50%, transparent 70%)`,
                        animation: 'rippleExpand 0.6s ease-out forwards',
                    }}
                    onAnimationEnd={() => onRippleComplete(ripple.id)}
                />
            ))}

            {/* CSS animation for ripple */}
            <style jsx>{`
                @keyframes rippleExpand {
                    0% {
                        transform: scale(0);
                        opacity: 1;
                    }
                    100% {
                        transform: scale(4);
                        opacity: 0;
                    }
                }
            `}</style>
        </>
    );
};

// Space background component for the space variant - clean and functional
const SpaceBackground = ({
    ripples,
    onRippleComplete
}: {
    ripples: Ripple[];
    onRippleComplete: (id: number) => void;
}) => {
    return (
        <>
            {/* Border that matches the gradient theme */}
            <div
                className="tw-absolute tw-inset-0 tw-rounded"
                style={{
                    background: `linear-gradient(135deg, #1a3a4a 0%, #2a4a6a 50%, #1a3a4a 100%)`,
                }}
            />

            {/* Clean gradient background with subtle space feel */}
            <div
                className="tw-absolute tw-inset-0.5 tw-rounded"
                style={{
                    background: `linear-gradient(135deg, #0a1a2a 0%, #16213a 50%, #0a1a2a 100%)`,
                }}
            />

            {/* Subtle glow effect */}
            <div
                className="tw-absolute tw-inset-0.5 tw-rounded tw-opacity-60"
                style={{
                    background: `radial-gradient(ellipse at center, rgba(22, 163, 97, 0.15) 0%, transparent 70%)`,
                }}
            />

            {/* Improved animated gradient sweep on hover */}
            <div
                className="tw-absolute tw-inset-0.5 tw-rounded tw-opacity-0 group-hover:tw-opacity-100 tw-transition-opacity tw-duration-500 tw-pointer-events-none"
                style={{
                    background: `
                        linear-gradient(110deg, 
                            transparent 25%, 
                            rgba(15, 170, 170, 0.4) 45%, 
                            rgba(22, 163, 97, 0.4) 55%, 
                            transparent 75%
                        )
                    `,
                    backgroundSize: '200% 100%',
                    animation: ripples.length > 0 ? 'none' : 'gradientSweep 3s ease-in-out infinite',
                }}
            />

            {/* Click ripple effects */}
            {ripples.map((ripple) => (
                <div
                    key={ripple.id}
                    className="tw-absolute tw-pointer-events-none tw-rounded-full"
                    style={{
                        left: ripple.x - 25,
                        top: ripple.y - 25,
                        width: 50,
                        height: 50,
                        background: 'radial-gradient(circle, rgba(15, 170, 170, 0.8) 0%, rgba(22, 163, 97, 0.4) 50%, transparent 70%)',
                        animation: 'rippleExpand 0.6s ease-out forwards',
                    }}
                    onAnimationEnd={() => onRippleComplete(ripple.id)}
                />
            ))}

            {/* CSS animations */}
            <style jsx>{`
                @keyframes gradientSweep {
                    0% { 
                        background-position: -200% 0%; 
                        transform: skewX(-12deg);
                    }
                    50% { 
                        background-position: 200% 0%; 
                        transform: skewX(-12deg);
                    }
                    100% { 
                        background-position: -200% 0%; 
                        transform: skewX(-12deg);
                    }
                }
                
                @keyframes rippleExpand {
                    0% {
                        transform: scale(0);
                        opacity: 1;
                    }
                    100% {
                        transform: scale(4);
                        opacity: 0;
                    }
                }
            `}</style>
        </>
    );
};

/**
 * A performant, accessible Tailwind CSS button component with multiple variants and sizes.
 * Supports loading states, icons, and full keyboard/screen reader support.
 */
export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
    (
        {
            variant = "primary",
            size = "md",
            isLoading = false,
            loadingIndicator = "circular",
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

        // Ripple effect state for all variants
        const [ripples, setRipples] = useState<Ripple[]>([]);

        // Handle click for ripple effect
        const handleClick = useCallback((event: MouseEvent<HTMLButtonElement>) => {
            // Add ripple effect for all variants
            if (!isDisabled) {
                const rect = event.currentTarget.getBoundingClientRect();
                const x = event.clientX - rect.left;
                const y = event.clientY - rect.top;

                const newRipple: Ripple = {
                    id: Date.now(),
                    x,
                    y,
                };

                setRipples(prev => [...prev, newRipple]);
            }

            // Call the original onClick handler
            if (props.onClick) {
                props.onClick(event);
            }
        }, [isDisabled, props.onClick]);

        // Remove completed ripples
        const handleRippleComplete = useCallback((id: number) => {
            setRipples(prev => prev.filter(ripple => ripple.id !== id));
        }, []);

        // Build the complete className
        const buttonClasses = cn(
            // Base styles - always applied
            "tw-inline-flex tw-items-center tw-justify-center tw-gap-2",
            "tw-font-sans tw-font-medium tw-tracking-wider tw-uppercase",
            "tw-rounded tw-transition-all tw-duration-200",
            "tw-border-0 tw-outline-none",
            "focus:tw-ring-offset-background",
            // Add group class for space variant hover effects
            variant === "space" && "tw-group",

            // Width styles
            fullWidth ? "tw-w-full" : "tw-w-auto",

            // Add relative positioning for ripple effect (except space which already has it)
            variant !== "space" && "tw-relative tw-overflow-hidden",

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
                const spinnerSize = spinnerSizes[size];

                // Use Orbital Spinner for space-themed loading
                if (loadingIndicator === "orbital") {
                    return <OrbitalSpinner size={spinnerSize} />;
                }

                // Default circular spinner
                const spinnerVariant =
                    variant === "outline" || variant === "ghost" ? "current" :
                        variant === "secondary" ? "secondary" :
                            "white";

                return (
                    <CircularProgress
                        size={spinnerSize}
                        variant={spinnerVariant}
                        thickness={size === "sm" ? 2 : 3}
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
                onClick={handleClick}
                {...props}
            >
                {/* Space background for space variant */}
                {variant === "space" && (
                    <SpaceBackground
                        ripples={ripples}
                        onRippleComplete={handleRippleComplete}
                    />
                )}

                {/* Ripple effects for non-space variants */}
                {variant !== "space" && (
                    <RippleEffect
                        ripples={ripples}
                        onRippleComplete={handleRippleComplete}
                        color={rippleColors[variant]}
                    />
                )}

                {/* Button content with relative positioning */}
                <span className={cn(
                    "tw-inline-flex tw-items-center tw-justify-center",
                    variant === "space" ? "tw-relative tw-z-10" : ""
                )}>
                    {renderStartElement()}
                </span>
                {children && <span className={variant === "space" ? "tw-relative tw-z-10" : ""}>{children}</span>}
                {!isLoading && endIcon && (
                    <span className={cn(
                        "tw-inline-flex tw-items-center tw-justify-center",
                        variant === "space" ? "tw-relative tw-z-10" : ""
                    )}>
                        {endIcon}
                    </span>
                )}
            </button>
        );
    }
);

Button.displayName = "Button";

// Export a typed component factory for common use cases
export const ButtonFactory = {
    Primary: (props: Omit<ButtonProps, "variant">) => (
        <Button variant="primary" {...props} />
    ),
    Secondary: (props: Omit<ButtonProps, "variant">) => (
        <Button variant="secondary" {...props} />
    ),
    Outline: (props: Omit<ButtonProps, "variant">) => (
        <Button variant="outline" {...props} />
    ),
    Ghost: (props: Omit<ButtonProps, "variant">) => (
        <Button variant="ghost" {...props} />
    ),
    Danger: (props: Omit<ButtonProps, "variant">) => (
        <Button variant="danger" {...props} />
    ),
    Space: (props: Omit<ButtonProps, "variant">) => (
        <Button variant="space" {...props} />
    ),
};