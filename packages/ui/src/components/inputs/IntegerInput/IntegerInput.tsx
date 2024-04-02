import { Box, FormControl, FormHelperText, Input, InputLabel, Tooltip, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback } from "react";
import { IntegerInputBaseProps, IntegerInputProps } from "../types";

export const IntegerInputBase = ({
    allowDecimal = false,
    autoFocus = false,
    disabled = false,
    error = false,
    fullWidth = false,
    helperText,
    key,
    initial = 0,
    label = "Number",
    max = Number.MAX_SAFE_INTEGER,
    min = Number.MIN_SAFE_INTEGER,
    name,
    offset = 0,
    onChange,
    step = 1,
    tooltip = "",
    value,
    ...props
}: IntegerInputBaseProps) => {
    const { palette } = useTheme();

    const updateValue = useCallback((quantity) => {
        if (quantity > max) quantity = max;
        if (quantity < min) quantity = min;
        onChange(quantity);
    }, [max, min, onChange]);

    return (
        <Tooltip title={tooltip}>
            <Box key={key} {...props} sx={{
                display: "flex",
                justifyContent: "center",
                ...props?.sx ?? {},
            }}>
                <FormControl sx={{
                    background: palette.background.paper,
                    width: "100%",
                    maxWidth: fullWidth ? "100%" : "12ch",
                    height: "100%",
                    borderRadius: "4px",
                    border: `1px solid ${palette.divider}`,
                }} error={!!error}>
                    <InputLabel
                        htmlFor={`quantity-box-${name}`}
                        sx={{
                            color: (value < min || value > max) ?
                                palette.error.main :
                                (value === min || value === max) ?
                                    palette.warning.main :
                                    palette.background.textSecondary,
                            paddingTop: "12px",
                        }}
                    >{label}</InputLabel>
                    <Input
                        autoFocus={autoFocus}
                        disabled={disabled}
                        id={`quantity-box-${name}`}
                        name={name}
                        aria-describedby={`helper-text-${name}`}
                        type="number"
                        inputMode="numeric"
                        inputProps={{
                            min,
                            max,
                            pattern: "[0-9]*",
                        }}
                        value={(value ?? 0) + offset}
                        onChange={(e) => updateValue(Number(e.target.value) - offset)}
                        sx={{
                            color: palette.background.textPrimary,
                            marginLeft: 1,
                        }}
                    />
                    {helperText && <FormHelperText id={`helper-text-${name}`}>{helperText}</FormHelperText>}
                </FormControl>
            </Box>
        </Tooltip >
    );
};

export const IntegerInput = ({
    name,
    ...props
}: IntegerInputProps) => {
    const [field, meta, helpers] = useField<number>(name);

    const handleChange = (value) => {
        helpers.setValue(value);
    };

    return (
        <IntegerInputBase
            {...props}
            name={name}
            value={field.value}
            error={meta.touched && !!meta.error}
            helperText={meta.touched && meta.error}
            onBlur={field.onBlur}
            onChange={handleChange}
        />
    );
};
