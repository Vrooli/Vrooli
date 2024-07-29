import { Box, FormControl, FormHelperText, Input, InputLabel, Palette, Tooltip, useTheme } from "@mui/material";
import { useField } from "formik";
import { useMemo } from "react";
import { IntegerInputBaseProps, IntegerInputProps } from "../types";

export function getNumberInRange(
    updatedNumber: number,
    max: number,
    min: number,
) {
    let result = updatedNumber;
    if (result > max) result = max;
    if (result < min) result = min;
    return result;
}

export function calculateUpdatedNumber(
    updatedNumber: string,
    max: number,
    min: number,
    allowDecimal = false,
) {
    let asNumber = Number(updatedNumber);
    if (!Number.isFinite(asNumber)) asNumber = 0;
    let result = getNumberInRange(asNumber, max, min);
    if (!allowDecimal) result = Math.round(result);
    return result;
}

export function getColorForLabel(
    value: number | string,
    min: number,
    max: number,
    palette: Palette,
    zeroText: string | undefined,
) {
    let asNumber = zeroText && value === zeroText ? 0 : Number(value);
    if (!Number.isFinite(asNumber)) {
        asNumber = Number.MIN_SAFE_INTEGER;
    }
    if (asNumber < min || asNumber > max) {
        return palette.error.main;
    } else if (asNumber === min || asNumber === max) {
        return palette.warning.main;
    } else {
        return palette.background.textSecondary;
    }
}

export function IntegerInputBase({
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
    onBlur,
    onChange,
    step = 1,
    sx,
    tooltip = "",
    value,
    zeroText,
}: IntegerInputBaseProps) {
    const { palette } = useTheme();

    const offsetValue = (value ?? 0) + offset;
    const displayValue = offsetValue === 0 && zeroText ? zeroText : offsetValue;

    const inputProps = useMemo(function inputPropsMemo() {
        return {
            min,
            max,
            pattern: "[0-9]*",
        } as const;
    }, [min, max]);

    return (
        <Tooltip title={tooltip}>
            <Box
                key={key}
                onBlur={onBlur}
                sx={{
                    display: "flex",
                    justifyContent: "flex-start",
                    ...sx,
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
                            color: getColorForLabel(displayValue, min, max, palette, zeroText),
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
                        inputProps={inputProps}
                        value={displayValue}
                        onChange={(e) => onChange(calculateUpdatedNumber(e.target.value, max, min, allowDecimal))}
                        placeholder={zeroText}
                        sx={{
                            color: palette.background.textPrimary,
                            marginLeft: 1,
                        }}
                    />
                    {helperText && <FormHelperText id={`helper-text-${name}`}>{typeof helperText === "string" ? helperText : JSON.stringify(helperText)}</FormHelperText>}
                </FormControl>
            </Box>
        </Tooltip >
    );
}

export function IntegerInput({
    name,
    ...props
}: IntegerInputProps) {
    const [field, meta, helpers] = useField<number>(name);

    function handleChange(value) {
        helpers.setValue(value);
    }

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
}
