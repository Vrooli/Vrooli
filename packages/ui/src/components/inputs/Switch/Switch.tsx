import type { ReactNode } from "react";
import { forwardRef, useCallback, useMemo, useId } from "react";
import { cn } from "../../../utils/tailwind-theme.js";
import {
    SWITCH_TRACK_STYLES,
    SWITCH_THUMB_STYLES,
    SWITCH_LABEL_STYLES,
    buildSwitchClasses,
    getTrackDimensions,
    getThumbDimensions,
    getThumbPosition,
    getCustomSwitchStyle,
} from "./switchStyles.js";
import type { SwitchProps, SwitchVariant, SwitchSize, LabelPosition } from "./types.js";

// Re-export types for backward compatibility
export type { SwitchProps, SwitchVariant, SwitchSize, LabelPosition } from "./types.js";

/**
 * Label component that handles positioning and styling
 */
const SwitchLabel = ({ 
    children, 
    htmlFor, 
    position, 
    disabled, 
}: { 
    children: ReactNode; 
    htmlFor: string; 
    position: LabelPosition;
    disabled: boolean;
}) => {
    if (position === "none" || !children) return null;
    
    return (
        <label 
            htmlFor={htmlFor}
            className={cn(
                SWITCH_LABEL_STYLES,
                disabled && "tw-opacity-50 tw-cursor-not-allowed",
            )}
        >
            {children}
        </label>
    );
};

/**
 * A performant, accessible Tailwind CSS switch component with multiple variants and sizes.
 * 
 * Features:
 * - 7 variants including custom color support, space-themed, and neon glow
 * - 3 size options with consistent proportions
 * - Flexible label positioning (left, right, or none)
 * - Smooth animations and transitions
 * - Full accessibility support with ARIA attributes
 * - Keyboard navigation (Space/Enter to toggle)
 * - Optimized performance with memoization
 * 
 * @example
 * ```tsx
 * // Basic switch
 * <Switch checked={isEnabled} onChange={setIsEnabled} label="Enable feature" />
 * 
 * // Custom color variant
 * <Switch variant="custom" color="#9333EA" label="Custom purple switch" />
 * 
 * // Space-themed switch
 * <Switch variant="space" size="lg" label="Activate space mode" />
 * 
 * // Left-positioned label
 * <Switch labelPosition="left" label="Show notifications" />
 * ```
 */
export const Switch = forwardRef<HTMLInputElement, SwitchProps>(
    (
        {
            variant = "default",
            size = "md",
            labelPosition = "right",
            label,
            color = "#9333EA",
            checked = false,
            disabled = false,
            className,
            onChange,
            id: providedId,
            "aria-label": ariaLabel,
            "aria-labelledby": ariaLabelledBy,
            "aria-describedby": ariaDescribedBy,
            ...props
        },
        ref,
    ) => {
        // Generate unique ID for accessibility
        const generatedId = useId();
        const id = providedId || generatedId;
        
        // Get dimensions for the current size
        const trackDimensions = useMemo(() => getTrackDimensions(size), [size]);
        const thumbDimensions = useMemo(() => getThumbDimensions(size), [size]);
        const thumbPosition = useMemo(() => getThumbPosition(checked, size), [checked, size]);
        
        // Memoize switch classes for performance
        const switchClasses = useMemo(() => 
            buildSwitchClasses({
                variant,
                size,
                disabled,
                className,
            }),
            [variant, size, disabled, className],
        );
        
        // Memoize track classes
        const trackClasses = useMemo(() => cn(
            "tw-rounded-full tw-relative tw-flex-shrink-0",
            SWITCH_TRACK_STYLES[variant],
        ), [variant]);
        
        // Memoize thumb classes
        const thumbClasses = useMemo(() => cn(
            "tw-absolute tw-top-1/2 tw-transform -tw-translate-y-1/2",
            SWITCH_THUMB_STYLES[variant],
        ), [variant]);
        
        // Handle change events
        const handleChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
            if (disabled) return;
            
            const newChecked = event.target.checked;
            onChange?.(newChecked, event);
        }, [disabled, onChange]);
        
        // Handle keyboard events for accessibility
        const handleKeyDown = useCallback((event: React.KeyboardEvent<HTMLInputElement>) => {
            if (disabled) return;
            
            // Toggle on Space or Enter
            if (event.key === " " || event.key === "Enter") {
                event.preventDefault();
                const syntheticEvent = {
                    ...event,
                    target: { ...event.target, checked: !checked },
                } as React.ChangeEvent<HTMLInputElement>;
                
                onChange?.(!checked, syntheticEvent);
            }
        }, [disabled, checked, onChange]);
        
        // Custom style for custom variant
        const customStyle = useMemo(() => 
            variant === "custom" ? getCustomSwitchStyle(color) : {},
            [variant, color],
        );
        
        // Determine aria-label
        const finalAriaLabel = ariaLabel || (typeof label === "string" ? label : "Toggle switch");
        
        return (
            <div 
                className={cn(
                    "tw-inline-flex tw-items-center tw-gap-3",
                    labelPosition === "right" && "tw-flex-row-reverse",
                )}
                style={customStyle}
            >
                {/* Label - positioned based on labelPosition */}
                <SwitchLabel 
                    htmlFor={id} 
                    position={labelPosition} 
                    disabled={disabled}
                >
                    {label}
                </SwitchLabel>
                
                {/* Hidden input for form integration and accessibility */}
                <input
                    ref={ref}
                    type="checkbox"
                    id={id}
                    checked={checked}
                    disabled={disabled}
                    onChange={handleChange}
                    onKeyDown={handleKeyDown}
                    className="tw-sr-only"
                    aria-label={finalAriaLabel}
                    aria-labelledby={ariaLabelledBy}
                    aria-describedby={ariaDescribedBy}
                    role="switch"
                    aria-checked={checked}
                    {...props}
                />
                
                {/* Visual switch container */}
                <div 
                    className={switchClasses}
                    data-testid="switch-visual-container"
                    onClick={() => {
                        if (!disabled) {
                            const syntheticEvent = {
                                target: { checked: !checked },
                            } as React.ChangeEvent<HTMLInputElement>;
                            onChange?.(!checked, syntheticEvent);
                        }
                    }}
                >
                    {/* Track */}
                    <div
                        className={trackClasses}
                        style={{
                            width: `${trackDimensions.width}px`,
                            height: `${trackDimensions.height}px`,
                            ...(variant === "custom" && checked ? {
                                backgroundColor: color,
                            } : {}),
                        }}
                        data-checked={checked}
                    >
                        {/* Thumb */}
                        <div
                            className={thumbClasses}
                            style={{
                                width: `${thumbDimensions.size}px`,
                                height: `${thumbDimensions.size}px`,
                                left: `${thumbPosition}px`,
                                ...(variant === "custom" && checked ? {
                                    boxShadow: `0 0 0 2px ${color}20, 0 4px 8px -2px ${color}40`,
                                } : {}),
                            }}
                            data-checked={checked}
                        >
                            {/* Theme variant sun/moon inside thumb */}
                            {variant === "theme" && (
                                <div className="tw-absolute tw-inset-0 tw-flex tw-items-center tw-justify-center">
                                    {/* Sun (visible when off/light mode) */}
                                    <svg 
                                        className={cn(
                                            "tw-absolute tw-fill-current",
                                            "tw-transition-all tw-duration-300",
                                            checked ? "tw-opacity-0 tw-scale-50 tw-rotate-180" : "tw-opacity-100 tw-scale-100 tw-rotate-0",
                                        )}
                                        width={thumbDimensions.size}
                                        height={thumbDimensions.size}
                                        viewBox="0 0 24 24"
                                    >
                                        {/* Simple yellow circle for sun */}
                                        <circle cx="12" cy="12" r="10" fill="#FFD700"/>
                                    </svg>
                                    {/* Moon (visible when on/dark mode) */}
                                    <svg 
                                        className={cn(
                                            "tw-absolute tw-fill-current",
                                            "tw-transition-all tw-duration-300",
                                            checked ? "tw-opacity-100 tw-scale-100 tw-rotate-0" : "tw-opacity-0 tw-scale-50 tw-rotate-180",
                                        )}
                                        width={thumbDimensions.size}
                                        height={thumbDimensions.size}
                                        viewBox="0 0 24 24"
                                    >
                                        {/* Moon circle - bigger */}
                                        <circle cx="12" cy="12" r="10.5" fill="#D3D3D3"/>
                                        {/* Moon craters */}
                                        <circle cx="8" cy="7" r="2.5" fill="#B8B8B8" opacity="0.8"/>
                                        <circle cx="15.5" cy="10" r="2" fill="#B8B8B8" opacity="0.6"/>
                                        <circle cx="10" cy="15" r="1.5" fill="#B8B8B8" opacity="0.7"/>
                                        <circle cx="16" cy="16.5" r="1" fill="#B8B8B8" opacity="0.5"/>
                                        <circle cx="13" cy="18" r="0.8" fill="#B8B8B8" opacity="0.4"/>
                                    </svg>
                                </div>
                            )}
                        </div>
                        
                        {/* Special effects for space variant */}
                        {variant === "space" && checked && (
                            <div 
                                className="tw-absolute tw-inset-0 tw-rounded-full tw-bg-gradient-to-r tw-from-cyan-400/20 tw-to-blue-500/20 tw-animate-pulse"
                                style={{ animationDuration: "2s" }}
                            />
                        )}
                        
                        {/* Special effects for neon variant */}
                        {variant === "neon" && checked && (
                            <div className="tw-absolute tw-inset-0 tw-rounded-full tw-bg-gradient-to-r tw-from-green-300/20 tw-to-green-500/20" />
                        )}
                        
                        
                        {/* Custom variant glow effect */}
                        {variant === "custom" && checked && (
                            <div 
                                className="tw-absolute tw-inset-0 tw-rounded-full tw-opacity-30"
                                style={{
                                    background: `radial-gradient(circle, ${color}40 0%, ${color}10 70%, transparent 100%)`,
                                }}
                            />
                        )}
                    </div>
                </div>
                
                {/* Screen reader feedback for state changes */}
                <span className="tw-sr-only" role="status" aria-live="polite">
                    {checked ? "On" : "Off"}
                </span>
            </div>
        );
    },
);

Switch.displayName = "Switch";

/**
 * Pre-configured switch components for common use cases
 * Provides convenience components with locked variants
 */
export const SwitchFactory = {
    /** Default blue switch */
    Default: (props: Omit<SwitchProps, "variant">) => (
        <Switch variant="default" {...props} />
    ),
    /** Success/green switch */
    Success: (props: Omit<SwitchProps, "variant">) => (
        <Switch variant="success" {...props} />
    ),
    /** Warning/orange switch */
    Warning: (props: Omit<SwitchProps, "variant">) => (
        <Switch variant="warning" {...props} />
    ),
    /** Danger/red switch */
    Danger: (props: Omit<SwitchProps, "variant">) => (
        <Switch variant="danger" {...props} />
    ),
    /** Space-themed switch */
    Space: (props: Omit<SwitchProps, "variant">) => (
        <Switch variant="space" {...props} />
    ),
    /** Neon glowing green switch */
    Neon: (props: Omit<SwitchProps, "variant">) => (
        <Switch variant="neon" {...props} />
    ),
    /** Theme switch for light/dark mode */
    Theme: (props: Omit<SwitchProps, "variant">) => (
        <Switch variant="theme" {...props} />
    ),
} as const;
