import { Box, FormControl, FormHelperText, Input, InputLabel, Tooltip, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback } from "react";
import { IntegerInputProps } from "../types";

export const IntegerInput = ({
    allowDecimal = false,
    autoFocus = false,
    disabled = false,
    fullWidth = false,
    key,
    initial = 0,
    label = "Number",
    max = Number.MAX_SAFE_INTEGER,
    min = Number.MIN_SAFE_INTEGER,
    name,
    offset = 0,
    step = 1,
    tooltip = "",
    ...props
}: IntegerInputProps) => {
    const { palette } = useTheme();
    const [field, meta, helpers] = useField<number>(name);

    const updateValue = useCallback((quantity) => {
        if (quantity > max) quantity = max;
        if (quantity < min) quantity = min;
        helpers.setValue(quantity);
    }, [max, min, helpers]);

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
                }} error={meta.touched && !!meta.error}>
                    <InputLabel
                        htmlFor={`quantity-box-${name}`}
                        sx={{
                            color: (field.value < min || field.value > max) ?
                                palette.error.main :
                                (field.value === min || field.value === max) ?
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
                        value={(field.value ?? 0) + offset}
                        onChange={(e) => updateValue(Number(e.target.value) - offset)}
                        sx={{
                            color: palette.background.textPrimary,
                            marginLeft: 1,
                        }}
                    />
                    {meta.touched && meta.error && <FormHelperText id={`helper-text-${name}`}>{meta.error}</FormHelperText>}
                </FormControl>
            </Box>
        </Tooltip >
    );
};
