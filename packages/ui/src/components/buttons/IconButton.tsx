import React, { type ButtonHTMLAttributes, type MouseEvent, type ReactNode } from "react";
import { forwardRef, useCallback, useMemo } from "react";
import { useTheme } from "@mui/material/styles";
import { cn } from "../../utils/tailwind-theme.js";
import { useRippleEffect } from "../../hooks/index.js";
import {
    ICON_BUTTON_COLORS,
    BASE_ICON_BUTTON_STYLES,
    ICON_VARIANT_STYLES,
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
const ButtonIcon = ({ children, size, variant }: { children: ReactNode; size: number; variant: IconButtonVariant }) => {
    const { palette } = useTheme();
    
    // Safety check for undefined children
    if (!children) {
        console.error("ButtonIcon received undefined children");
        return null;
    }
    
    // Calculate icon size based on button size
    // Transparent variant gets larger icons (65% of button size) for better visibility
    const sizeMultiplier = variant === "transparent" ? 0.65 : 0.5;
    const iconSize = Math.max(16, Math.min(32, Math.round(size * sizeMultiplier)));
    
    // Check if children is a React element that can be cloned
    const isReactElement = React.isValidElement(children);
    const iconElement = isReactElement ? children as React.ReactElement : null;
    const hasExplicitFill = iconElement?.props?.fill !== undefined;
    
    // Calculate dynamic icon color based on variant for proper contrast
    // Only use variant-based color if icon doesn't have explicit fill
    const iconColor = useMemo(() => {
        if (hasExplicitFill) {
            return "currentColor"; // Let the icon handle its own color
        }
        
        switch (variant) {
            case "solid":
                // For solid variant, use white text on the dark gradient background
                return "#ffffff";
            case "transparent":
                // For transparent variant, use theme text color
                return palette.text.primary;
            case "space":
                // For space variant, use light text on dark space background
                return "#ffffff";
            case "custom":
                // For custom variant, color is handled by parent component
                return "currentColor";
            case "neon":
                // For neon variant, use white text on green background
                return "#ffffff";
            default:
                return palette.text.primary;
        }
    }, [variant, palette.text.primary, hasExplicitFill]);
    
    return (
        <span 
            className={cn(
                "tw-inline-flex tw-items-center tw-justify-center",
                // Only force fill: currentColor if icon doesn't have explicit fill
                !hasExplicitFill && "[&>svg]:tw-fill-current",
            )}
            style={{
                width: iconSize,
                height: iconSize,
                color: iconColor,
            }}
        >
            {isReactElement ? (
                React.cloneElement(iconElement, {
                    size: iconSize,
                    // Preserve the original fill prop if it exists
                    ...(hasExplicitFill && { fill: iconElement.props.fill }),
                })
            ) : (
                // For non-React elements (like emoji strings), render them directly
                children
            )}
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
    color,
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
                    animation: "iconButtonRippleExpand 0.6s ease-out forwards",
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
    onRippleComplete,
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
                    willChange: "transform",
                    transform: "translateZ(0)",
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
                    willChange: "opacity, background-position",
                    transform: "translateZ(0)",
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
 * Neon Background Component
 * Special glowing effects for the neon variant with circular animation
 */
const NeonBackground = ({
    ripples,
    onRippleComplete,
}: {
    ripples: Array<{ id: number; x: number; y: number }>;
    onRippleComplete: (id: number) => void;
}) => {
    return (
        <>
            {/* Click ripple effects */}
            <RippleEffect
                ripples={ripples}
                onRippleComplete={onRippleComplete}
                color={ICON_BUTTON_COLORS.RIPPLE.neon}
            />
        </>
    );
};

/**
 * A performant, accessible icon button component with multiple variants.
 * 
 * Features:
 * - 5 variants: solid (3D physical), transparent, space-themed, custom, and neon (glowing green)
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
        ref,
    ) => {
        // Convert size to numeric value
        const numericSize = useMemo(() => getNumericSize(size), [size]);
        
        // Use the ripple effect hook for interactive feedback
        const { ripples, handleRippleClick, handleRippleComplete } = useRippleEffect();

        // Memoize button classes for performance
        // Note: We exclude padding since we set explicit dimensions
        const buttonClasses = useMemo(() => cn(
            // Base styles without padding
            BASE_ICON_BUTTON_STYLES,
            
            // Variant styles
            ICON_VARIANT_STYLES[variant],
            
            // State styles
            disabled && "tw-opacity-50 tw-cursor-not-allowed tw-pointer-events-none",
            
            // Custom overrides
            className,
        ), [variant, disabled, className]);

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
            <div style={{ 
                width: `${numericSize}px`, 
                height: `${numericSize}px`,
                flexShrink: 0,
                display: "inline-block",
            }}>
                <button
                    ref={ref}
                    type={type}
                    disabled={disabled}
                    className={buttonClasses}
                    onClick={handleClick}
                    aria-disabled={disabled}
                    aria-label={props["aria-label"] || "Icon button"}
                    {...props}
                    style={{
                        ...props.style,
                        width: "100%",
                        height: "100%",
                        boxSizing: "border-box",
                    }}
                >
                {/* Space background for space variant - simplified to just sweep animation */}
                {variant === "space" && (
                    <>
                        <div className="tw-icon-button-space-sweep" />
                        {/* Space-specific ripple effects */}
                        {ripples.map((ripple) => (
                            <div
                                key={ripple.id}
                                className="tw-button-ripple"
                                style={createIconRippleStyle(ripple, rippleColor)}
                                onAnimationEnd={() => handleRippleComplete(ripple.id)}
                            />
                        ))}
                    </>
                )}

                {/* Neon background for neon variant */}
                {variant === "neon" && (
                    <NeonBackground
                        ripples={ripples}
                        onRippleComplete={handleRippleComplete}
                    />
                )}

                {/* Ripple effects for solid, transparent, and custom variants */}
                {variant !== "space" && variant !== "neon" && (
                    <RippleEffect
                        ripples={ripples}
                        onRippleComplete={handleRippleComplete}
                        color={rippleColor}
                    />
                )}

                {/* Icon content with proper z-index for space and neon variants */}
                <span className={cn(
                    "tw-relative tw-flex tw-items-center tw-justify-center",
                    (variant === "space" || variant === "neon") && "tw-z-10",
                )}>
                    <ButtonIcon size={numericSize} variant={variant}>{children}</ButtonIcon>
                </span>
                
                {/* Screen reader support for disabled state */}
                {disabled && (
                    <span className="tw-sr-only" role="status">
                        Button disabled
                    </span>
                )}
                </button>
            </div>
        );
    },
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
    /** Neon glowing green icon button */
    Neon: (props: Omit<IconButtonProps, "variant">) => (
        <IconButton variant="neon" {...props} />
    ),
} as const;
