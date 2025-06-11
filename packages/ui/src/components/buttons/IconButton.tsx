import React, { type ButtonHTMLAttributes, type MouseEvent, type ReactNode, forwardRef, useCallback, useState } from "react";
import { cn } from "../../utils/tailwind-theme.js";

// Define variant types
export type IconButtonVariant = "solid" | "transparent" | "space";
export type IconButtonSize = "sm" | "md" | "lg" | number;

export interface IconButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
    variant?: IconButtonVariant;
    size?: IconButtonSize;
    children: ReactNode;
    className?: string;
}

// Size mappings for predefined sizes
const sizeMap: Record<string, number> = {
    sm: 32,
    md: 48,
    lg: 64,
};

// Helper function to get numeric size
function getNumericSize(size: IconButtonSize): number {
    if (typeof size === "number") return size;
    return sizeMap[size] || sizeMap.md;
}

// Helper function to calculate padding based on size
function calculatePadding(size: number): string {
    if (size <= 16) return "tw-p-0";
    if (size <= 32) return "tw-p-1";
    if (size <= 48) return "tw-p-2";
    return "tw-p-3";
}

// Ripple effect interface
interface Ripple {
    id: number;
    x: number;
    y: number;
}

// Ripple colors for each variant - using simple rgba values
const rippleColors: Record<IconButtonVariant, string> = {
    solid: "rgba(255, 255, 255, 0.5)",
    transparent: "rgba(100, 100, 100, 0.3)",
    space: "rgba(15, 170, 170, 0.8)",
};

// Space background component
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
                className="tw-absolute tw-inset-0 tw-rounded-full"
                style={{
                    background: `linear-gradient(135deg, #1a3a4a 0%, #2a4a6a 50%, #1a3a4a 100%)`,
                }}
            />

            {/* Clean gradient background with subtle space feel */}
            <div
                className="tw-absolute tw-inset-0.5 tw-rounded-full"
                style={{
                    background: `linear-gradient(135deg, #0a1a2a 0%, #16213a 50%, #0a1a2a 100%)`,
                }}
            />

            {/* Subtle glow effect */}
            <div
                className="tw-absolute tw-inset-0.5 tw-rounded-full tw-opacity-60"
                style={{
                    background: `radial-gradient(ellipse at center, rgba(22, 163, 97, 0.15) 0%, transparent 70%)`,
                }}
            />

            {/* Animated gradient sweep on hover */}
            <div
                className="tw-absolute tw-inset-0.5 tw-rounded-full tw-opacity-0 group-hover:tw-opacity-100 tw-transition-opacity tw-duration-500 tw-pointer-events-none"
                style={{
                    background: `linear-gradient(110deg, transparent 25%, rgba(15, 170, 170, 0.4) 45%, rgba(22, 163, 97, 0.4) 55%, transparent 75%)`,
                    backgroundSize: '200% 100%',
                    animation: 'iconButtonGradientSweep 3s ease-in-out infinite',
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
                        animation: 'iconButtonRippleExpand 0.6s ease-out forwards',
                    }}
                    onAnimationEnd={() => onRippleComplete(ripple.id)}
                />
            ))}
        </>
    );
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
    // Create a faded version of the color - simplified approach
    const fadeColor = color.includes('rgba') 
        ? color.replace(/[\d.]+(?=\))/, '0.1')
        : 'rgba(100, 100, 100, 0.1)'; // Safe fallback
    
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
                        background: `radial-gradient(circle, ${color} 0%, ${fadeColor} 50%, transparent 70%)`,
                        animation: 'iconButtonRippleExpand 0.6s ease-out forwards',
                    }}
                    onAnimationEnd={() => onRippleComplete(ripple.id)}
                />
            ))}
        </>
    );
};

/**
 * A versatile icon button component with multiple variants
 * Supports solid (3D physical), transparent, and space-themed styles
 */
export const IconButton = forwardRef<HTMLButtonElement, IconButtonProps>(
    (
        {
            variant = "transparent",
            size = "md",
            disabled = false,
            className,
            children,
            type = "button",
            onClick,
            ...props
        },
        ref
    ) => {
        const numericSize = getNumericSize(size);
        const [ripples, setRipples] = useState<Ripple[]>([]);

        // Handle click for ripple effect
        const handleClick = useCallback((event: MouseEvent<HTMLButtonElement>) => {
            // Add ripple effect
            if (!disabled) {
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
            if (onClick) {
                onClick(event);
            }
        }, [disabled, onClick]);

        // Remove completed ripples
        const handleRippleComplete = useCallback((id: number) => {
            setRipples(prev => prev.filter(ripple => ripple.id !== id));
        }, []);

        // Build the complete className
        const buttonClasses = cn(
            // Base classes for all icon buttons
            "tw-relative tw-rounded-full tw-transition-all tw-duration-150",
            "tw-inline-flex tw-items-center tw-justify-center",
            "tw-border-0 tw-outline-none tw-overflow-hidden",
            calculatePadding(numericSize),
            
            // Variant-specific classes
            variant === "solid" && "tw-icon-button-solid",
            variant === "transparent" && "tw-icon-button-transparent",
            variant === "space" && "tw-icon-button-space tw-group",
            
            // Disabled state
            disabled && "tw-opacity-50 tw-cursor-not-allowed tw-pointer-events-none",
            
            // Custom className (allows overrides and additions like listening state)
            className
        );

        return (
            <button
                ref={ref}
                type={type}
                disabled={disabled}
                className={buttonClasses}
                style={{
                    width: numericSize,
                    height: numericSize,
                }}
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

                {/* Ripple effects for solid and transparent variants */}
                {variant !== "space" && (
                    <RippleEffect
                        ripples={ripples}
                        onRippleComplete={handleRippleComplete}
                        color={rippleColors[variant]}
                    />
                )}

                {/* Icon content with relative positioning for space variant */}
                <span className={cn(
                    "tw-relative",
                    variant === "space" && "tw-z-10"
                )}>
                    {children}
                </span>
            </button>
        );
    }
);

IconButton.displayName = "IconButton";