import React, { forwardRef, useCallback, cloneElement, isValidElement } from "react";
import { FormControlLabelProps } from "./types.js";
import { cn } from "../../utils/tailwind-theme.js";
import { getFormControlLabelStyles } from "./formControlLabelStyles.js";

export const FormControlLabel = forwardRef<HTMLLabelElement, FormControlLabelProps>(({
    control,
    label,
    labelPlacement = "end",
    disabled = false,
    required = false,
    className,
    style,
    sx,
    value,
    onChange,
    ...props
}, ref) => {
    const styles = getFormControlLabelStyles({
        labelPlacement,
        disabled,
        required,
        sx,
    });

    // Handle change events from the control element
    const handleChange = useCallback((event: React.SyntheticEvent) => {
        if (onChange && !disabled) {
            const target = event.target as HTMLInputElement;
            onChange(event, target.checked);
        }
    }, [onChange, disabled]);

    // Clone the control element with additional props
    const controlElement = isValidElement(control) ? cloneElement(control, {
        ...control.props,
        disabled: control.props.disabled ?? disabled,
        required: control.props.required ?? required,
        value: control.props.value ?? value,
        onChange: control.props.onChange ?? handleChange,
    }) : control;

    // Render label with optional required asterisk
    const labelContent = (
        <span className={styles.labelText}>
            {label}
            {required && (
                <span className={styles.asterisk} aria-label="required">
                    {" *"}
                </span>
            )}
        </span>
    );

    return (
        <label
            ref={ref}
            className={cn(styles.root, className)}
            style={style}
            {...props}
        >
            {(labelPlacement === "start" || labelPlacement === "top") && labelContent}
            <span className={styles.controlWrapper}>
                {controlElement}
            </span>
            {(labelPlacement === "end" || labelPlacement === "bottom") && labelContent}
        </label>
    );
});

FormControlLabel.displayName = "FormControlLabel";