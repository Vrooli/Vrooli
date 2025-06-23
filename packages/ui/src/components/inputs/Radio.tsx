import React, { forwardRef, useCallback, useMemo } from "react";
import { type RadioProps } from "./types.js";
import { cn } from "../../utils/tailwind-theme.js";
import { getRadioStyles } from "./radioStyles.js";
import { useRippleEffect } from "../../hooks/index.js";

export const Radio = forwardRef<HTMLInputElement, RadioProps>(({
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
    ...props
}, ref) => {
    const { addRipple, ripples } = useRippleEffect();

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
});

Radio.displayName = "Radio";

// Factory functions for common variants
export const PrimaryRadio = forwardRef<HTMLInputElement, Omit<RadioProps, "color">>((props, ref) => (
    <Radio ref={ref} color="primary" {...props} />
));
PrimaryRadio.displayName = "PrimaryRadio";

export const SecondaryRadio = forwardRef<HTMLInputElement, Omit<RadioProps, "color">>((props, ref) => (
    <Radio ref={ref} color="secondary" {...props} />
));
SecondaryRadio.displayName = "SecondaryRadio";

export const DangerRadio = forwardRef<HTMLInputElement, Omit<RadioProps, "color">>((props, ref) => (
    <Radio ref={ref} color="danger" {...props} />
));
DangerRadio.displayName = "DangerRadio";

export const CustomRadio = forwardRef<HTMLInputElement, Omit<RadioProps, "color"> & { customColor: string }>((props, ref) => (
    <Radio ref={ref} color="custom" {...props} />
));
CustomRadio.displayName = "CustomRadio";
