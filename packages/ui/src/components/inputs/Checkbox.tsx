import React, { forwardRef, useCallback, useEffect, useMemo, useRef } from "react";
import { useField } from "formik";
import { type CheckboxProps, type CheckboxBaseProps, type CheckboxFormikProps } from "./types.js";
import { cn } from "../../utils/tailwind-theme.js";
import { getCheckboxStyles } from "./checkboxStyles.js";
import { useRippleEffect } from "../../hooks/index.js";

/**
 * Base checkbox component without Formik integration.
 * This is the pure visual component that handles all styling and interaction logic.
 */
export const CheckboxBase = forwardRef<HTMLInputElement, CheckboxBaseProps>(({
    className,
    color = "primary",
    customColor,
    size = "md",
    disabled = false,
    checked,
    defaultChecked,
    indeterminate = false,
    required,
    onChange,
    onClick,
    onFocus,
    onBlur,
    "aria-label": ariaLabel,
    "aria-labelledby": ariaLabelledBy,
    "aria-describedby": ariaDescribedBy,
    sx,
    style,
    error = false,
    helperText,
    label,
    ...props
}, ref) => {
    const { addRipple, ripples } = useRippleEffect();
    const internalRef = useRef<HTMLInputElement>(null);
    const checkboxRef = (ref as React.MutableRefObject<HTMLInputElement>) || internalRef;

    // Handle indeterminate state
    useEffect(() => {
        if (checkboxRef.current) {
            checkboxRef.current.indeterminate = indeterminate;
        }
    }, [indeterminate, checkboxRef]);

    const styles = useMemo(() => getCheckboxStyles({
        color,
        customColor,
        size,
        disabled,
        checked: checked ?? defaultChecked ?? false,
        indeterminate,
        sx,
    }), [color, customColor, size, disabled, checked, defaultChecked, indeterminate, sx]);

    const handleClick = useCallback((e: React.MouseEvent<HTMLSpanElement>) => {
        if (!disabled) {
            const rect = e.currentTarget.getBoundingClientRect();
            addRipple(e.clientX - rect.left, e.clientY - rect.top);
        }
    }, [disabled, addRipple]);

    const handleChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        if (!disabled && onChange) {
            onChange(e);
        }
    }, [disabled, onChange]);

    const handleInputClick = useCallback((e: React.MouseEvent<HTMLInputElement>) => {
        if (!disabled && onClick) {
            onClick(e);
        }
    }, [disabled, onClick]);

    const checkboxElement = (
        <label 
            className={cn(styles.container, className)} 
            style={style}
            data-testid="checkbox-container"
            data-disabled={disabled}
            data-checked={checked ?? defaultChecked ?? false}
            data-indeterminate={indeterminate}
        >
            <input
                ref={checkboxRef}
                type="checkbox"
                className={cn(styles.input, "peer")}
                checked={checked}
                defaultChecked={defaultChecked}
                disabled={disabled}
                required={required}
                onChange={handleChange}
                onClick={handleInputClick}
                onFocus={onFocus}
                onBlur={onBlur}
                aria-label={ariaLabel}
                aria-labelledby={ariaLabelledBy}
                aria-describedby={ariaDescribedBy}
                aria-checked={indeterminate ? "mixed" : checked}
                data-testid="checkbox-input"
                {...props}
            />
            <span 
                className={styles.checkboxWrapper}
                onClick={handleClick}
                onMouseEnter={(e) => {
                    if (color === "custom" && customColor && !disabled && styles.customStyles?.hover) {
                        e.currentTarget.style.backgroundColor = styles.customStyles.hover.backgroundColor;
                    }
                }}
                onMouseLeave={(e) => {
                    if (color === "custom" && customColor && !disabled) {
                        e.currentTarget.style.backgroundColor = "";
                    }
                }}
                style={color === "custom" && customColor && !disabled && styles.customStyles?.focus ? {
                    "--tw-ring-color": styles.customStyles.focus.ringColor,
                } as React.CSSProperties : undefined}
                data-testid="checkbox-visual"
            >
                <span 
                    className={styles.checkboxBox}
                    style={color === "custom" && customColor ? styles.customStyles?.box : undefined}
                    data-testid="checkbox-box"
                >
                    {/* Checkmark icon */}
                    <svg 
                        className={styles.checkmark}
                        viewBox="0 0 24 24"
                        aria-hidden="true"
                        data-testid="checkbox-icon"
                    >
                        <path
                            fill="currentColor"
                            d={indeterminate 
                                ? "M19 13H5v-2h14v2z" // Horizontal line for indeterminate
                                : "M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" // Checkmark
                            }
                        />
                    </svg>
                </span>
                {ripples.map((ripple) => (
                    <span
                        key={ripple.id}
                        className={styles.ripple}
                        style={{
                            left: ripple.x - ripple.size / 2,
                            top: ripple.y - ripple.size / 2,
                            width: ripple.size,
                            height: ripple.size,
                        }}
                        onAnimationEnd={() => ripple.onAnimationEnd()}
                        data-testid={`checkbox-ripple-${ripple.id}`}
                    />
                ))}
            </span>
        </label>
    );

    // Return checkbox with optional label and helper text
    return (
        <div className={cn("tw-flex tw-flex-col tw-gap-1", error && "tw-text-red-500")}>
            <div className="tw-flex tw-items-center tw-gap-2">
                {checkboxElement}
                {label && (
                    <span className={cn(
                        "tw-text-sm",
                        disabled && "tw-opacity-50 tw-cursor-not-allowed",
                    )}>
                        {label}
                    </span>
                )}
            </div>
            {helperText && (
                <div className={cn(
                    "tw-text-sm tw-ml-1",
                    error ? "tw-text-red-500" : "tw-text-gray-600",
                )}>
                    {helperText}
                </div>
            )}
        </div>
    );
});

CheckboxBase.displayName = "CheckboxBase";

/**
 * Formik-integrated checkbox component.
 * Automatically connects to Formik context using the field name.
 * 
 * @example
 * ```tsx
 * // Inside a Formik form
 * <Checkbox name="acceptTerms" label="I accept the terms and conditions" />
 * 
 * // With validation
 * <Checkbox 
 *   name="newsletter" 
 *   label="Send me newsletters"
 *   validate={(value) => !value ? "Required" : undefined}
 * />
 * ```
 */
export const Checkbox = forwardRef<HTMLInputElement, CheckboxFormikProps>(({
    name,
    validate,
    id,
    ...props
}, ref) => {
    const [field, meta] = useField({ name, validate });
    
    // Use provided id or fall back to the field name
    const inputId = id || name;
    
    return (
        <CheckboxBase
            {...field}
            {...props}
            id={inputId}
            checked={field.value || false}
            error={meta.touched && Boolean(meta.error)}
            helperText={meta.touched && meta.error}
            data-testid={props["data-testid"] || `checkbox-${name}`}
            ref={ref}
        />
    );
});

Checkbox.displayName = "Checkbox";

// Factory functions for Formik-integrated variants
export const CheckboxFactory = {
    Primary: (props: Omit<CheckboxFormikProps, "color">) => (
        <Checkbox color="primary" {...props} />
    ),
    Secondary: (props: Omit<CheckboxFormikProps, "color">) => (
        <Checkbox color="secondary" {...props} />
    ),
    Danger: (props: Omit<CheckboxFormikProps, "color">) => (
        <Checkbox color="danger" {...props} />
    ),
    Success: (props: Omit<CheckboxFormikProps, "color">) => (
        <Checkbox color="success" {...props} />
    ),
    Warning: (props: Omit<CheckboxFormikProps, "color">) => (
        <Checkbox color="warning" {...props} />
    ),
    Info: (props: Omit<CheckboxFormikProps, "color">) => (
        <Checkbox color="info" {...props} />
    ),
    Custom: (props: Omit<CheckboxFormikProps, "color"> & { customColor: string }) => (
        <Checkbox color="custom" {...props} />
    ),
} as const;

// Factory functions for base variants (no Formik)
export const CheckboxFactoryBase = {
    Primary: (props: Omit<CheckboxBaseProps, "color">) => (
        <CheckboxBase color="primary" {...props} />
    ),
    Secondary: (props: Omit<CheckboxBaseProps, "color">) => (
        <CheckboxBase color="secondary" {...props} />
    ),
    Danger: (props: Omit<CheckboxBaseProps, "color">) => (
        <CheckboxBase color="danger" {...props} />
    ),
    Success: (props: Omit<CheckboxBaseProps, "color">) => (
        <CheckboxBase color="success" {...props} />
    ),
    Warning: (props: Omit<CheckboxBaseProps, "color">) => (
        <CheckboxBase color="warning" {...props} />
    ),
    Info: (props: Omit<CheckboxBaseProps, "color">) => (
        <CheckboxBase color="info" {...props} />
    ),
    Custom: (props: Omit<CheckboxBaseProps, "color"> & { customColor: string }) => (
        <CheckboxBase color="custom" {...props} />
    ),
} as const;

// Legacy factory exports for backward compatibility
export const PrimaryCheckbox = CheckboxFactory.Primary;
export const SecondaryCheckbox = CheckboxFactory.Secondary;
export const DangerCheckbox = CheckboxFactory.Danger;
export const SuccessCheckbox = CheckboxFactory.Success;
export const CustomCheckbox = CheckboxFactory.Custom;
