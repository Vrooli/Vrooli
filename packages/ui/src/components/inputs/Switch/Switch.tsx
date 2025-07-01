import type { ReactNode } from "react";
import { forwardRef, useCallback, useMemo, useId } from "react";
import { useField } from "formik";
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
import type { SwitchProps, SwitchVariant, SwitchSize, LabelPosition, SwitchBaseProps, SwitchFormikProps } from "./types.js";
import { Icon } from "../../../icons/Icons.js";
import { Tooltip } from "../../Tooltip/Tooltip.js";

// Re-export types for backward compatibility
export type { SwitchProps, SwitchVariant, SwitchSize, LabelPosition, SwitchBaseProps, SwitchFormikProps } from "./types.js";

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
 * Base switch component without Formik integration.
 * This is the pure visual component that handles all styling and interaction logic.
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
 * <SwitchBase checked={isEnabled} onChange={setIsEnabled} label="Enable feature" />
 * 
 * // Custom color variant
 * <SwitchBase variant="custom" color="#9333EA" label="Custom purple switch" />
 * 
 * // Space-themed switch
 * <SwitchBase variant="space" size="lg" label="Activate space mode" />
 * 
 * // Left-positioned label
 * <SwitchBase labelPosition="left" label="Show notifications" />
 * ```
 */
export const SwitchBase = forwardRef<HTMLInputElement, SwitchBaseProps>(
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
            offIcon,
            onIcon,
            tooltip,
            id: providedId,
            "aria-label": ariaLabel,
            "aria-labelledby": ariaLabelledBy,
            "aria-describedby": ariaDescribedBy,
            error = false,
            helperText,
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
        
        // Get current icon based on checked state
        const currentIcon = checked ? onIcon : offIcon;
        
        // Wrap in tooltip if provided
        const wrapInTooltip = (content: ReactNode) => {
            if (!tooltip) return content;
            return <Tooltip title={tooltip}>{content}</Tooltip>;
        };
        
        const switchElement = (
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
                            
                            {/* Custom icons inside thumb */}
                            {currentIcon && variant !== "theme" && (
                                <div className="tw-absolute tw-inset-0 tw-flex tw-items-center tw-justify-center">
                                    <Icon
                                        info={currentIcon}
                                        fill="white"
                                        sizeOverride={thumbDimensions.size * 0.6}
                                    />
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
        
        // Return switch with optional helper text
        return wrapInTooltip(
            <div className="tw-flex tw-flex-col tw-gap-1">
                {switchElement}
                {helperText && (
                    <div className={cn(
                        "tw-text-sm tw-ml-1",
                        error ? "tw-text-red-500" : "tw-text-gray-600",
                    )}>
                        {helperText}
                    </div>
                )}
            </div>,
        );
    },
);

SwitchBase.displayName = "SwitchBase";

/**
 * Formik-integrated switch component.
 * Automatically connects to Formik context using the field name.
 * 
 * @example
 * ```tsx
 * // Inside a Formik form
 * <Switch name="notifications" label="Enable notifications" />
 * 
 * // With validation
 * <Switch 
 *   name="acceptTerms" 
 *   label="I accept the terms and conditions"
 *   validate={(value) => !value ? "You must accept the terms" : undefined}
 * />
 * ```
 */
export const Switch = forwardRef<HTMLInputElement, SwitchFormikProps>(({
    name,
    validate,
    id,
    ...props
}, ref) => {
    const [field, meta] = useField({ name, validate });
    
    // Use provided id or fall back to the field name
    const inputId = id || name;
    
    // Convert Formik's onChange to our Switch onChange signature
    const handleChange = useCallback((checked: boolean, event: React.ChangeEvent<HTMLInputElement>) => {
        field.onChange(event);
    }, [field]);
    
    return (
        <SwitchBase
            {...field}
            {...props}
            id={inputId}
            checked={field.value || false}
            onChange={handleChange}
            error={meta.touched && Boolean(meta.error)}
            helperText={meta.touched && meta.error}
            data-testid={props["data-testid"] || `switch-${name}`}
            ref={ref}
        />
    );
});

Switch.displayName = "Switch";

/**
 * Pre-configured switch components for common use cases
 * Provides convenience components with locked variants
 */
export const SwitchFactory = {
    /** Default blue switch */
    Default: (props: Omit<SwitchFormikProps, "variant">) => (
        <Switch variant="default" {...props} />
    ),
    /** Success/green switch */
    Success: (props: Omit<SwitchFormikProps, "variant">) => (
        <Switch variant="success" {...props} />
    ),
    /** Warning/orange switch */
    Warning: (props: Omit<SwitchFormikProps, "variant">) => (
        <Switch variant="warning" {...props} />
    ),
    /** Danger/red switch */
    Danger: (props: Omit<SwitchFormikProps, "variant">) => (
        <Switch variant="danger" {...props} />
    ),
    /** Space-themed switch */
    Space: (props: Omit<SwitchFormikProps, "variant">) => (
        <Switch variant="space" {...props} />
    ),
    /** Neon glowing green switch */
    Neon: (props: Omit<SwitchFormikProps, "variant">) => (
        <Switch variant="neon" {...props} />
    ),
    /** Theme switch for light/dark mode */
    Theme: (props: Omit<SwitchFormikProps, "variant">) => (
        <Switch variant="theme" {...props} />
    ),
} as const;

/**
 * Pre-configured base switch components for use outside Formik
 * These are pure visual components without form integration
 */
export const SwitchFactoryBase = {
    /** Default blue switch */
    Default: (props: Omit<SwitchBaseProps, "variant">) => (
        <SwitchBase variant="default" {...props} />
    ),
    /** Success/green switch */
    Success: (props: Omit<SwitchBaseProps, "variant">) => (
        <SwitchBase variant="success" {...props} />
    ),
    /** Warning/orange switch */
    Warning: (props: Omit<SwitchBaseProps, "variant">) => (
        <SwitchBase variant="warning" {...props} />
    ),
    /** Danger/red switch */
    Danger: (props: Omit<SwitchBaseProps, "variant">) => (
        <SwitchBase variant="danger" {...props} />
    ),
    /** Space-themed switch */
    Space: (props: Omit<SwitchBaseProps, "variant">) => (
        <SwitchBase variant="space" {...props} />
    ),
    /** Neon glowing green switch */
    Neon: (props: Omit<SwitchBaseProps, "variant">) => (
        <SwitchBase variant="neon" {...props} />
    ),
    /** Theme switch for light/dark mode */
    Theme: (props: Omit<SwitchBaseProps, "variant">) => (
        <SwitchBase variant="theme" {...props} />
    ),
} as const;
