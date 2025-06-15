import React, { forwardRef, useCallback, useEffect, useMemo, useRef } from "react";
import { CheckboxProps } from "./types.js";
import { cn } from "../../utils/tailwind-theme.js";
import { getCheckboxStyles } from "./checkboxStyles.js";
import { useRippleEffect } from "../../hooks/index.js";

export const Checkbox = forwardRef<HTMLInputElement, CheckboxProps>(({
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

    return (
        <label className={cn(styles.container, className)} style={style}>
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
            >
                <span 
                    className={styles.checkboxBox}
                    style={color === "custom" && customColor ? styles.customStyles?.box : undefined}
                >
                    {/* Checkmark icon */}
                    <svg 
                        className={styles.checkmark}
                        viewBox="0 0 24 24"
                        aria-hidden="true"
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
                    />
                ))}
            </span>
        </label>
    );
});

Checkbox.displayName = "Checkbox";

// Factory functions for common variants
export const PrimaryCheckbox = forwardRef<HTMLInputElement, Omit<CheckboxProps, "color">>((props, ref) => (
    <Checkbox ref={ref} color="primary" {...props} />
));
PrimaryCheckbox.displayName = "PrimaryCheckbox";

export const SecondaryCheckbox = forwardRef<HTMLInputElement, Omit<CheckboxProps, "color">>((props, ref) => (
    <Checkbox ref={ref} color="secondary" {...props} />
));
SecondaryCheckbox.displayName = "SecondaryCheckbox";

export const DangerCheckbox = forwardRef<HTMLInputElement, Omit<CheckboxProps, "color">>((props, ref) => (
    <Checkbox ref={ref} color="danger" {...props} />
));
DangerCheckbox.displayName = "DangerCheckbox";

export const SuccessCheckbox = forwardRef<HTMLInputElement, Omit<CheckboxProps, "color">>((props, ref) => (
    <Checkbox ref={ref} color="success" {...props} />
));
SuccessCheckbox.displayName = "SuccessCheckbox";

export const CustomCheckbox = forwardRef<HTMLInputElement, Omit<CheckboxProps, "color"> & { customColor: string }>((props, ref) => (
    <Checkbox ref={ref} color="custom" {...props} />
));
CustomCheckbox.displayName = "CustomCheckbox";