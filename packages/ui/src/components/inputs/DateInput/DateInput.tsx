import { IconButton, InputAdornment, styled, useTheme } from "@mui/material";
import { useField } from "formik";
import { CloseIcon } from "icons";
import { useCallback, useMemo } from "react";
import { TextInput } from "../TextInput/TextInput";
import { DateInputProps } from "../types";

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

const StyledTextInput = styled(TextInput)(({ theme }) => ({
    display: "block",
    "& ::-webkit-calendar-picker-indicator": {
        filter: theme.palette.mode === "dark" ? "invert(1)" : "invert(0)",
    },
}));

const inputAdornmentStyle = { display: "flex", alignItems: "center" } as const;
const textInputLabelProps = { shrink: true } as const;

export function DateInput({
    isRequired,
    label,
    name,
    type = "datetime-local",
}: DateInputProps) {
    const { palette } = useTheme();

    const [field, , helpers] = useField(name);

    const clearDate = useCallback(function clearDateCallback() {
        helpers.setValue("");
    }, [helpers]);

    const inputProps = useMemo(function inputPropsMemo() {
        return {
            endAdornment: (
                <InputAdornment position="end" sx={inputAdornmentStyle}>
                    <input type="hidden" />
                    {typeof field.value === "string" && field.value.length > 0 && <IconButton edge="end" size="small" onClick={clearDate}>
                        <CloseIcon fill={palette.background.textPrimary} width="20px" height="20px" />
                    </IconButton>}
                </InputAdornment>
            ),
        };
    }, [clearDate, field.value, palette.background.textPrimary]);

    return (
        <StyledTextInput
            isRequired={isRequired}
            label={label}
            type={type}
            InputProps={inputProps}
            InputLabelProps={textInputLabelProps}
            {...field}
            value={formatForDateTimeLocal(field.value, type)}
        />
    );
}
