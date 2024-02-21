import { IconButton, InputAdornment, useTheme } from "@mui/material";
import { useField } from "formik";
import { CloseIcon } from "icons";
import { useCallback } from "react";
import { TextInput } from "../TextInput/TextInput";
import { DateInputProps } from "../types";

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
    const month = ("0" + (date.getMonth() + 1)).slice(-2); // months are 0-indexed in JS
    const day = ("0" + date.getDate()).slice(-2);

    if (type === "date") {
        // Return the date string in the format 'YYYY-MM-DD'
        return `${year}-${month}-${day}`;
    } else {
        // Get local time components
        const hours = ("0" + date.getHours()).slice(-2);
        const minutes = ("0" + date.getMinutes()).slice(-2);
        const seconds = ("0" + date.getSeconds()).slice(-2);

        // Return the date string in the format 'YYYY-MM-DDTHH:mm:ss'
        return `${year}-${month}-${day}T${hours}:${minutes}:${seconds}`;
    }
}

export const DateInput = ({
    isOptional,
    label,
    name,
    type = "datetime-local",
}: DateInputProps) => {
    const { palette } = useTheme();

    const [field, , helpers] = useField(name);

    const clearDate = useCallback(() => {
        helpers.setValue("");
    }, [helpers]);

    return (
        <TextInput
            isOptional={isOptional}
            label={label}
            type={type}
            InputProps={{
                endAdornment: (
                    <InputAdornment position="end" sx={{ display: "flex", alignItems: "center" }}>
                        <input type="hidden" />
                        {typeof field.value === "string" && field.value.length > 0 && <IconButton edge="end" size="small" onClick={clearDate}>
                            <CloseIcon fill={palette.background.textPrimary} width="20px" height="20px" />
                        </IconButton>}
                    </InputAdornment>
                ),
            }}
            InputLabelProps={{ shrink: true }}
            {...field}
            value={formatForDateTimeLocal(field.value, type)}
            sx={{
                display: "block",
                "& ::-webkit-calendar-picker-indicator": {
                    filter: palette.mode === "dark" ? "invert(1)" : "invert(0)",
                },
            }}
        />
    );
};
