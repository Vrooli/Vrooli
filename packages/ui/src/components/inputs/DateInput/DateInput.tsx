import { IconButton } from "../../buttons/IconButton.js";
import { useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback, useMemo, useEffect } from "react";
import { IconCommon } from "../../../icons/Icons.js";
import { TextInput } from "../TextInput/TextInput.js";
import { type DateInputProps } from "../types.js";

const DATE_FORMAT_CHARACTERS = 2;

function formatForDateTimeLocal(dateStr, type) {
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

export function DateInput({
    isRequired,
    label,
    name,
    type = "datetime-local",
}: DateInputProps) {
    const { palette } = useTheme();

    const [field, , helpers] = useField(name);

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
        helpers.setValue("");
    }, [helpers]);

    const endAdornment = useMemo(function endAdornmentMemo() {
        if (typeof field.value === "string" && field.value.length > 0) {
            return (
                <IconButton 
                    edge="end" 
                    size="sm" 
                    variant="transparent" 
                    onClick={clearDate}
                    aria-label="Clear date"
                    data-testid="date-input-clear"
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
    }, [clearDate, field.value, palette.background.textPrimary]);

    return (
        <TextInput
            isRequired={isRequired}
            label={label}
            type={type}
            endAdornment={endAdornment}
            {...field}
            value={formatForDateTimeLocal(field.value, type)}
            className={dateInputClassName}
            data-testid="date-input"
            data-type={type}
        />
    );
}
