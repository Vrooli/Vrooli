import type { ButtonHTMLAttributes, MouseEvent, ReactNode } from "react";
import { forwardRef, useCallback, useMemo } from "react";
import { cn } from "../../utils/tailwind-theme.js";
import { CircularProgress, OrbitalSpinner } from "../indicators/CircularProgress.js";
import { useRippleEffect, type Ripple } from "../../hooks/index.js";
import {
    BUTTON_CONFIG,
    BUTTON_COLORS,
    buildButtonClasses,
    createRippleStyle,
    getSpinnerConfig,
    getSpinnerVariant,
} from "./buttonStyles.js";

// Export types for external use
export type ButtonVariant = "primary" | "secondary" | "outline" | "ghost" | "danger" | "space" | "custom";
export type ButtonSize = "sm" | "md" | "lg" | "icon";

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
    /** Visual style variant of the button */
    variant?: ButtonVariant;
    /** Size of the button */
    size?: ButtonSize;
    /** Whether the button is in a loading state */
    isLoading?: boolean;
    /** Type of loading indicator to display */
    loadingIndicator?: "circular" | "orbital";
    /** Icon to display at the start of the button */
    startIcon?: ReactNode;
    /** Icon to display at the end of the button */
    endIcon?: ReactNode;
    /** Whether the button should take full width of its container */
    fullWidth?: boolean;
    /** Button content */
    children?: ReactNode;
}

/**
 * Icon wrapper component to ensure icons inherit button text color
 * Wraps icon elements with proper styling for color inheritance
 */
const ButtonIcon = ({ children }: { children: ReactNode }) => (
    <span className="tw-inline-flex [&>svg]:tw-fill-current">
        {children}
    </span>
);

/**
 * Generic Ripple Effect Component
 * Reusable ripple effect for all button variants except space
 */
const RippleEffect = ({
    ripples,
    onRippleComplete,
    color
}: {
    ripples: Ripple[];
    onRippleComplete: (id: number) => void;
    color: string;
}) => (
    <>
        {ripples.map((ripple) => (
            <div
                key={ripple.id}
                className="tw-button-ripple"
                style={createRippleStyle(ripple, color)}
                onAnimationEnd={() => onRippleComplete(ripple.id)}
            />
        ))}
    </>
);

/**
 * Space Background Component
 * Special background effects for the space variant
 */
const SpaceBackground = ({
    ripples,
    onRippleComplete
}: {
    ripples: Ripple[];
    onRippleComplete: (id: number) => void;
}) => (
    <>
        {/* Background layers */}
        <div className="tw-button-space-border" />
        <div className="tw-button-space-background" />
        <div className="tw-button-space-glow" />
        <div className="tw-button-space-sweep group-active:tw-animation-pause" />
        
        {/* Space-specific ripple effects */}
        {ripples.map((ripple) => (
            <div
                key={ripple.id}
                className="tw-button-ripple"
                style={createRippleStyle(ripple, BUTTON_COLORS.RIPPLE.space)}
                onAnimationEnd={() => onRippleComplete(ripple.id)}
            />
        ))}
    </>
);

/**
 * A performant, accessible Tailwind CSS button component with multiple variants and sizes.
 * 
 * Features:
 * - 7 variants including custom color support
 * - 4 size options with consistent spacing
 * - Loading states with circular/orbital spinners
 * - Ripple effects for visual feedback
 * - Full accessibility support
 * - Optimized performance with memoization
 * 
 * @example
 * ```tsx
 * <Button variant="primary" size="md" startIcon={<Icon />}>
 *   Click me
 * </Button>
 * ```
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
            onClick,
            ...props
        },
        ref
    ) => {
        // Determine if button should be disabled (either explicitly or when loading)
        const isDisabled = disabled || isLoading;

        // Use the ripple effect hook for interactive feedback
        const { ripples, handleRippleClick, handleRippleComplete } = useRippleEffect();

        // Memoize button classes for performance - only recalculate when dependencies change
        const buttonClasses = useMemo(() => 
            buildButtonClasses({
                variant,
                size,
                fullWidth,
                disabled: isDisabled,
                className,
            }),
            [variant, size, fullWidth, isDisabled, className]
        );

        // Memoize start element (loading spinner or start icon) for performance
        const startElement = useMemo(() => {
            if (isLoading) {
                const { size: spinnerSize, thickness } = getSpinnerConfig(size);

                // Use Orbital Spinner for space-themed loading
                if (loadingIndicator === "orbital") {
                    return <OrbitalSpinner size={spinnerSize} />;
                }

                // Use appropriate spinner variant based on button variant
                const spinnerVariant = getSpinnerVariant(variant);

                return (
                    <CircularProgress
                        size={spinnerSize}
                        variant={spinnerVariant}
                        thickness={thickness}
                    />
                );
            }
            return startIcon;
        }, [isLoading, size, loadingIndicator, variant, startIcon]);

        // Handle click events with ripple effect and user's onClick
        const handleClick = useCallback((event: MouseEvent<HTMLButtonElement>) => {
            // Add ripple effect if not disabled
            handleRippleClick(event, isDisabled);
            
            // Call the user's onClick handler
            onClick?.(event);
        }, [isDisabled, onClick, handleRippleClick]);

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
                        color={BUTTON_COLORS.RIPPLE[variant]}
                    />
                )}

                {/* Button content with proper spacing and z-index for space variant */}
                <span className={cn(
                    "tw-inline-flex tw-items-center tw-justify-center tw-gap-2",
                    variant === "space" && "tw-relative tw-z-10"
                )}>
                    {startElement && <ButtonIcon>{startElement}</ButtonIcon>}
                    {children}
                    {!isLoading && endIcon && <ButtonIcon>{endIcon}</ButtonIcon>}
                </span>
                
                {/* Screen reader text for loading state */}
                {isLoading && (
                    <span className="tw-sr-only" role="status">
                        Loading, please wait
                    </span>
                )}
            </button>
        );
    }
);

Button.displayName = "Button";

/**
 * Pre-configured button components for common use cases
 * Provides convenience components with locked variants
 */
export const ButtonFactory = {
    /** Primary action button */
    Primary: (props: Omit<ButtonProps, "variant">) => (
        <Button variant="primary" {...props} />
    ),
    /** Secondary action button */
    Secondary: (props: Omit<ButtonProps, "variant">) => (
        <Button variant="secondary" {...props} />
    ),
    /** Outline style button */
    Outline: (props: Omit<ButtonProps, "variant">) => (
        <Button variant="outline" {...props} />
    ),
    /** Ghost style button */
    Ghost: (props: Omit<ButtonProps, "variant">) => (
        <Button variant="ghost" {...props} />
    ),
    /** Danger/destructive action button */
    Danger: (props: Omit<ButtonProps, "variant">) => (
        <Button variant="danger" {...props} />
    ),
    /** Space-themed button */
    Space: (props: Omit<ButtonProps, "variant">) => (
        <Button variant="space" {...props} />
    ),
} as const;