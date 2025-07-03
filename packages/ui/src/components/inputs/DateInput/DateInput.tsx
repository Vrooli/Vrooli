import { IconButton } from "../../buttons/IconButton.js";
import { useTheme } from "@mui/material";
import { useField } from "formik";
import React, { forwardRef, useCallback, useMemo, useEffect } from "react";
import { IconCommon } from "../../../icons/Icons.js";
import { TextInput, TextInputBase } from "../TextInput/TextInput.js";
import { type DateInputProps, type DateInputBaseProps, type DateInputFormikProps } from "../types.js";

const DATE_FORMAT_CHARACTERS = 2;

function formatForDateTimeLocal(dateStr: string, type: "date" | "datetime-local") {
    // Return empty string if no dateStr is provided
    if (!dateStr) {
        return "";
    }

    // Create a new Date object from the UTC string
    const date = new Date(dateStr);

    // Check if date is Invalid Date
    if (!isFinite(date.getTime())) {
        throw new Error("Invalid date format");
    }

    // Get local date components
    const year = date.getFullYear();
    const month = ("0" + (date.getMonth() + 1)).slice(-DATE_FORMAT_CHARACTERS); // months are 0-indexed in JS
    const day = ("0" + date.getDate()).slice(-DATE_FORMAT_CHARACTERS);

    if (type === "date") {
        // Return the date string in the format 'YYYY-MM-DD'
        return `${year}-${month}-${day}`;
    } else {
        // Get local time components
        const hours = ("0" + date.getHours()).slice(-DATE_FORMAT_CHARACTERS);
        const minutes = ("0" + date.getMinutes()).slice(-DATE_FORMAT_CHARACTERS);
        const seconds = ("0" + date.getSeconds()).slice(-DATE_FORMAT_CHARACTERS);

        // Return the date string in the format 'YYYY-MM-DDTHH:mm:ss'
        return `${year}-${month}-${day}T${hours}:${minutes}:${seconds}`;
    }
}

const dateInputClassName = "tw-date-input";

/**
 * Base date input component without Formik integration.
 * This is the pure visual component that handles all styling and interaction logic.
 */
export const DateInputBase = forwardRef<HTMLInputElement, DateInputBaseProps>(({
    isRequired,
    label,
    name,
    type = "datetime-local",
    value,
    onChange,
    onBlur,
    error,
    helperText,
    className,
    sx,
    disabled,
    "data-testid": dataTestId,
    ...props
}, ref) => {
    const { palette } = useTheme();

    // Apply dark mode styles for calendar picker indicator
    useEffect(() => {
        const styleId = "date-input-styles";
        let styleElement = document.getElementById(styleId) as HTMLStyleElement;
        
        if (!styleElement) {
            styleElement = document.createElement("style");
            styleElement.id = styleId;
            document.head.appendChild(styleElement);
        }
        
        styleElement.textContent = `
            .${dateInputClassName} input::-webkit-calendar-picker-indicator {
                filter: ${palette.mode === "dark" ? "invert(1)" : "invert(0)"};
            }
        `;
        
        return () => {
            // Clean up is handled by not removing the style since it may be used by multiple instances
        };
    }, [palette.mode]);

    const clearDate = useCallback(function clearDateCallback() {
        onChange("");
    }, [onChange]);

    const handleChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        onChange(e.target.value);
    }, [onChange]);

    const endAdornment = useMemo(function endAdornmentMemo() {
        if (typeof value === "string" && value.length > 0) {
            return (
                <IconButton 
                    edge="end" 
                    size="sm" 
                    variant="transparent" 
                    onClick={clearDate}
                    aria-label="Clear date"
                    data-testid="date-input-clear"
                    disabled={disabled}
                >
                    <IconCommon
                        decorative
                        fill={palette.background.textPrimary}
                        name="Close"
                        size={20}
                    />
                </IconButton>
            );
        }
        return null;
    }, [clearDate, value, palette.background.textPrimary, disabled]);

    return (
        <TextInputBase
            ref={ref}
            isRequired={isRequired}
            label={label}
            name={name}
            type={type}
            endAdornment={endAdornment}
            value={formatForDateTimeLocal(value, type)}
            onChange={handleChange}
            onBlur={onBlur}
            error={error}
            helperText={helperText}
            className={`${dateInputClassName} ${className || ""}`}
            sx={sx}
            disabled={disabled}
            data-testid={dataTestId || "date-input"}
            data-type={type}
            {...props}
        />
    );
});

DateInputBase.displayName = "DateInputBase";

/**
 * Formik-integrated date input component.
 * Automatically connects to Formik context using the field name.
 * 
 * @example
 * ```tsx
 * // Inside a Formik form
 * <DateInput name="birthDate" label="Birth Date" type="date" />
 * 
 * // With validation
 * <DateInput 
 *   name="meetingTime" 
 *   label="Meeting Time"
 *   type="datetime-local"
 *   validate={(value) => !value ? "Required" : undefined}
 * />
 * ```
 */
export const DateInput = forwardRef<HTMLInputElement, DateInputFormikProps>(({
    name,
    validate,
    ...props
}, ref) => {
    const [field, meta, helpers] = useField({ name, validate });

    const handleChange = useCallback((value: string) => {
        helpers.setValue(value);
    }, [helpers]);

    return (
        <DateInputBase
            {...props}
            ref={ref}
            name={name}
            value={field.value || ""}
            onChange={handleChange}
            onBlur={field.onBlur}
            error={meta.touched && Boolean(meta.error)}
            helperText={meta.touched && meta.error}
        />
    );
});

DateInput.displayName = "DateInput";
