import React, { forwardRef, useCallback, useMemo } from "react";
import { useField } from "formik";
import { type RadioProps, type RadioBaseProps, type RadioFormikProps } from "./types.js";
import { cn } from "../../utils/tailwind-theme.js";
import { getRadioStyles } from "./radioStyles.js";
import { useRippleEffect } from "../../hooks/index.js";

/**
 * Base radio button component without Formik integration.
 * This is the pure visual component that handles all styling and interaction logic.
 */
export const RadioBase = forwardRef<HTMLInputElement, RadioBaseProps>(({
    className,
    color = "primary",
    customColor,
    size = "md",
    disabled = false,
    checked,
    defaultChecked,
    value,
    name,
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
    // Calculate ripple size based on radio size
    const rippleSize = useMemo(() => {
        switch (size) {
            case "sm": return 24; // 6 * 4 = 24px
            case "lg": return 40; // 10 * 4 = 40px  
            case "md":
            default: return 32; // 8 * 4 = 32px
        }
    }, [size]);
    
    const { addRipple, ripples } = useRippleEffect(rippleSize);

    const styles = useMemo(() => getRadioStyles({
        color,
        customColor,
        size,
        disabled,
        checked: checked ?? defaultChecked ?? false,
        sx,
    }), [color, customColor, size, disabled, checked, defaultChecked, sx]);

    const handleClick = useCallback((e: React.MouseEvent<HTMLSpanElement>) => {
        if (!disabled) {
            const rect = e.currentTarget.getBoundingClientRect();
            // Center the ripple in the radio wrapper instead of using cursor position
            const centerX = rect.width / 2;
            const centerY = rect.height / 2;
            addRipple(centerX, centerY);
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

    const radioElement = (
        <label 
            className={cn(styles.container, className)} 
            style={style}
            data-testid="radio-label"
            data-disabled={disabled}
            data-checked={checked ?? defaultChecked ?? false}
        >
            <input
                ref={ref}
                type="radio"
                className={styles.input}
                checked={checked}
                defaultChecked={defaultChecked}
                value={value}
                name={name}
                required={required}
                disabled={disabled}
                onChange={handleChange}
                onClick={handleInputClick}
                onFocus={onFocus}
                onBlur={onBlur}
                aria-label={ariaLabel}
                aria-labelledby={ariaLabelledBy}
                aria-describedby={ariaDescribedBy}
                data-testid="radio-input"
                {...props}
            />
            <span 
                className={styles.radioWrapper}
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
                data-testid="radio-wrapper"
            >
                <span 
                    className={styles.radioOuter}
                    style={color === "custom" && customColor ? styles.customStyles?.outer : undefined}
                    data-testid="radio-outer"
                >
                    <span 
                        className={styles.radioInner}
                        style={color === "custom" && customColor ? styles.customStyles?.inner : undefined}
                        data-testid="radio-inner"
                    />
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
                        data-testid="radio-ripple"
                    />
                ))}
            </span>
        </label>
    );

    // Return radio with optional label and helper text
    return (
        <div className={cn("tw-flex tw-flex-col tw-gap-1", error && "tw-text-red-500")}>
            <div className="tw-flex tw-items-center tw-gap-2">
                {radioElement}
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

RadioBase.displayName = "RadioBase";

/**
 * Formik-integrated radio button component.
 * Automatically connects to Formik context using the field name.
 * 
 * @example
 * ```tsx
 * // Inside a Formik form
 * <Radio name="plan" value="basic" label="Basic Plan" />
 * <Radio name="plan" value="pro" label="Pro Plan" />
 * <Radio name="plan" value="enterprise" label="Enterprise Plan" />
 * ```
 */
export const Radio = forwardRef<HTMLInputElement, RadioFormikProps>(({
    name,
    validate,
    id,
    ...props
}, ref) => {
    const [field, meta] = useField({ name, validate });
    
    // Use provided id or fall back to name + value
    const inputId = id || `${name}-${props.value}`;
    
    return (
        <RadioBase
            {...field}
            {...props}
            id={inputId}
            error={meta.touched && Boolean(meta.error)}
            helperText={meta.touched && meta.error}
            data-testid={props["data-testid"] || `radio-${name}`}
            ref={ref}
        />
    );
});

Radio.displayName = "Radio";

// Factory functions for Formik-integrated variants
export const RadioFactory = {
    Primary: (props: Omit<RadioFormikProps, "color">) => (
        <Radio color="primary" {...props} />
    ),
    Secondary: (props: Omit<RadioFormikProps, "color">) => (
        <Radio color="secondary" {...props} />
    ),
    Danger: (props: Omit<RadioFormikProps, "color">) => (
        <Radio color="danger" {...props} />
    ),
    Success: (props: Omit<RadioFormikProps, "color">) => (
        <Radio color="success" {...props} />
    ),
    Warning: (props: Omit<RadioFormikProps, "color">) => (
        <Radio color="warning" {...props} />
    ),
    Info: (props: Omit<RadioFormikProps, "color">) => (
        <Radio color="info" {...props} />
    ),
    Custom: (props: Omit<RadioFormikProps, "color"> & { customColor: string }) => (
        <Radio color="custom" {...props} />
    ),
} as const;

// Factory functions for base variants (no Formik)
export const RadioFactoryBase = {
    Primary: (props: Omit<RadioBaseProps, "color">) => (
        <RadioBase color="primary" {...props} />
    ),
    Secondary: (props: Omit<RadioBaseProps, "color">) => (
        <RadioBase color="secondary" {...props} />
    ),
    Danger: (props: Omit<RadioBaseProps, "color">) => (
        <RadioBase color="danger" {...props} />
    ),
    Success: (props: Omit<RadioBaseProps, "color">) => (
        <RadioBase color="success" {...props} />
    ),
    Warning: (props: Omit<RadioBaseProps, "color">) => (
        <RadioBase color="warning" {...props} />
    ),
    Info: (props: Omit<RadioBaseProps, "color">) => (
        <RadioBase color="info" {...props} />
    ),
    Custom: (props: Omit<RadioBaseProps, "color"> & { customColor: string }) => (
        <RadioBase color="custom" {...props} />
    ),
} as const;

// Legacy factory exports for backward compatibility
export const PrimaryRadio = RadioFactory.Primary;
export const SecondaryRadio = RadioFactory.Secondary;
export const DangerRadio = RadioFactory.Danger;
export const CustomRadio = RadioFactory.Custom;
