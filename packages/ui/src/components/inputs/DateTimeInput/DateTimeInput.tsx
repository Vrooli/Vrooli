import { CloseIcon } from "@local/shared";
import { IconButton, InputAdornment, TextField, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback } from "react";
import { DateTimeInputProps } from "../types";

function formatForDateTimeLocal(dateStr, type) {
    // Return empty string if no dateStr is provided
    if (!dateStr) {
        return "";
    }

    // Create a new Date object from the UTC string
    const date = new Date(dateStr);

    // Check if date is Invalid Date
    if (isNaN(date.getTime())) {
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

export const DateTimeInput = ({
    fullWidth = true,
    label,
    name,
    type = "datetime-local",
}: DateTimeInputProps) => {
    const { palette } = useTheme();

    const [field, , helpers] = useField(name);

    const clearDate = useCallback(() => {
        helpers.setValue("");
    }, [helpers]);

    return (
        <TextField
            fullWidth={fullWidth}
            label={label}
            type={type}
            InputProps={{
                endAdornment: (
                    <InputAdornment position="end" sx={{ display: "flex", alignItems: "center" }}>
                        <input type="hidden" />
                        <IconButton edge="end" size="small" onClick={clearDate}>
                            <CloseIcon fill={palette.background.textPrimary} />
                        </IconButton>
                    </InputAdornment>
                ),
            }}
            InputLabelProps={{
                shrink: true,
            }}
            {...field}
            value={formatForDateTimeLocal(field.value, type)}
        />
    );
};
