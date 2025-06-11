import React, { type ButtonHTMLAttributes, type MouseEvent, type ReactNode } from "react";
import { forwardRef, useCallback, useMemo } from "react";
import { cn } from "../../utils/tailwind-theme.js";
import { useRippleEffect } from "../../hooks/index.js";
import {
    ICON_BUTTON_COLORS,
    buildIconButtonClasses,
    createIconRippleStyle,
    getNumericSize,
    getRippleColor,
    SPACE_BACKGROUND_STYLES,
    type IconButtonVariant,
    type IconButtonSize,
} from "./iconButtonStyles.js";

// Export types from styles file to maintain public API
export type { IconButtonVariant, IconButtonSize } from "./iconButtonStyles.js";

export interface IconButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
    /** Visual style variant of the icon button */
    variant?: IconButtonVariant;
    /** Size of the icon button (predefined or custom number) */
    size?: IconButtonSize;
    /** Icon button content (typically an icon component) */
    children: ReactNode;
    /** Additional CSS classes */
    className?: string;
}

/**
 * Icon wrapper component to ensure icons inherit button state colors and scale properly
 * Provides consistent icon styling across all variants with size-responsive scaling
 */
const ButtonIcon = ({ children, size }: { children: ReactNode; size: number }) => {
    // Calculate icon size based on button size - roughly 50% of button size
    const iconSize = Math.max(16, Math.min(32, Math.round(size * 0.5)));
    
    return (
        <span 
            className="tw-inline-flex [&>svg]:tw-fill-current"
            style={{
                fontSize: iconSize,
                width: iconSize,
                height: iconSize,
            }}
        >
            {React.cloneElement(children as React.ReactElement, {
                size: iconSize,
            })}
        </span>
    );
};

/**
 * Generic Ripple Effect Component
 * Reusable ripple effect for all icon button variants except space
 */
const RippleEffect = ({
    ripples,
    onRippleComplete,
    color
}: {
    ripples: Array<{ id: number; x: number; y: number }>;
    onRippleComplete: (id: number) => void;
    color: string;
}) => (
    <>
        {ripples.map((ripple) => (
            <div
                key={ripple.id}
                className="tw-absolute tw-pointer-events-none tw-rounded-full"
                style={{
                    ...createIconRippleStyle(ripple, color),
                    animation: 'iconButtonRippleExpand 0.6s ease-out forwards',
                }}
                onAnimationEnd={() => onRippleComplete(ripple.id)}
            />
        ))}
    </>
);

/**
 * Space Background Component
 * Special background effects for the space variant with anti-aliasing optimizations
 */
const SpaceBackground = ({
    ripples,
    onRippleComplete
}: {
    ripples: Array<{ id: number; x: number; y: number }>;
    onRippleComplete: (id: number) => void;
}) => {
    // Memoize static styles for performance
    const styles = useMemo(() => SPACE_BACKGROUND_STYLES, []);
    
    return (
        <>
            {/* Border gradient background - using pseudo-element approach */}
            <div
                className="tw-absolute tw-inset-0 tw-rounded-full -tw-z-10"
                style={styles.borderGradient}
            />

            {/* Main background with combined gradients and anti-aliasing */}
            <div
                className="tw-absolute tw-inset-0.5 tw-rounded-full tw-antialiased"
                style={{
                    ...styles.main,
                    // Force hardware acceleration for smoother rendering
                    willChange: 'transform',
                    transform: 'translateZ(0)',
                }}
            />

            {/* Animated gradient sweep on hover - pauses on active */}
            <div
                className="tw-absolute tw-inset-0.5 tw-rounded-full tw-opacity-0 
                         group-hover:tw-opacity-100 tw-transition-opacity tw-duration-500 
                         tw-pointer-events-none group-active:tw-animation-pause tw-antialiased"
                style={{
                    ...styles.sweep,
                    // Ensure smooth animation
                    willChange: 'opacity, background-position',
                    transform: 'translateZ(0)',
                }}
            />

            {/* Click ripple effects */}
            <RippleEffect
                ripples={ripples}
                onRippleComplete={onRippleComplete}
                color={ICON_BUTTON_COLORS.RIPPLE.space}
            />
        </>
    );
};

/**
 * A performant, accessible icon button component with multiple variants.
 * 
 * Features:
 * - 3 variants: solid (3D physical), transparent, and space-themed
 * - Flexible sizing with predefined and custom sizes
 * - Ripple effects for visual feedback
 * - Full accessibility support with ARIA attributes
 * - Optimized performance with memoization
 * - Consistent with Button component patterns
 * 
 * @example
 * ```tsx
 * // Predefined size
 * <IconButton variant="solid" size="md">
 *   <Icon />
 * </IconButton>
 * 
 * // Custom size
 * <IconButton variant="space" size={72}>
 *   <SpaceIcon />
 * </IconButton>
 * 
 * // With listening state for microphone
 * <IconButton variant="solid" size="lg" className="tw-microphone-listening">
 *   <MicrophoneIcon />
 * </IconButton>
 * ```
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
        // Convert size to numeric value
        const numericSize = useMemo(() => getNumericSize(size), [size]);
        
        // Use the ripple effect hook for interactive feedback
        const { ripples, handleRippleClick, handleRippleComplete } = useRippleEffect();

        // Memoize button classes for performance
        const buttonClasses = useMemo(() => 
            buildIconButtonClasses({
                variant,
                size: numericSize,
                disabled,
                className,
            }),
            [variant, numericSize, disabled, className]
        );

        // Memoize ripple color based on variant
        const rippleColor = useMemo(() => getRippleColor(variant), [variant]);

        // Handle click events with ripple effect
        const handleClick = useCallback((event: MouseEvent<HTMLButtonElement>) => {
            // Add ripple effect if not disabled
            handleRippleClick(event, disabled);
            
            // Call the user's onClick handler
            onClick?.(event);
        }, [disabled, onClick, handleRippleClick]);

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
                aria-disabled={disabled}
                aria-label={props['aria-label'] || 'Icon button'}
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
                        color={rippleColor}
                    />
                )}

                {/* Icon content with proper z-index for space variant */}
                <span className={cn(
                    "tw-relative",
                    variant === "space" && "tw-z-10"
                )}>
                    <ButtonIcon size={numericSize}>{children}</ButtonIcon>
                </span>
                
                {/* Screen reader support for disabled state */}
                {disabled && (
                    <span className="tw-sr-only" role="status">
                        Button disabled
                    </span>
                )}
            </button>
        );
    }
);

IconButton.displayName = "IconButton";

/**
 * Pre-configured icon button components for common use cases
 * Provides convenience components with locked variants
 */
export const IconButtonFactory = {
    /** Solid 3D-style icon button */
    Solid: (props: Omit<IconButtonProps, "variant">) => (
        <IconButton variant="solid" {...props} />
    ),
    /** Transparent icon button */
    Transparent: (props: Omit<IconButtonProps, "variant">) => (
        <IconButton variant="transparent" {...props} />
    ),
    /** Space-themed icon button */
    Space: (props: Omit<IconButtonProps, "variant">) => (
        <IconButton variant="space" {...props} />
    ),
} as const;