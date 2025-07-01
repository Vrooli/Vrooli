import Box from "@mui/material/Box";
import { Tooltip } from "../../Tooltip/Tooltip.js";
import type { Palette } from "@mui/material";
import { useTheme } from "@mui/material";
import { useField } from "formik";
import { useMemo } from "react";
import { InputContainer } from "../InputContainer/InputContainer.js";
import { type IntegerInputBaseProps, type IntegerInputProps } from "../types.js";

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
    variant = "filled",
    size = "md",
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
                data-testid="integer-input-container"
                sx={{
                    display: "flex",
                    justifyContent: "flex-start",
                    ...sx,
                }}>
                <InputContainer
                    variant={variant}
                    size={size}
                    error={error}
                    disabled={disabled}
                    fullWidth={fullWidth}
                    label={label}
                    helperText={helperText}
                    htmlFor={`quantity-box-${name}`}
                    className={fullWidth ? undefined : "tw-max-w-[12ch]"}
                >
                    <input
                        id={`quantity-box-${name}`}
                        name={name}
                        type="number"
                        inputMode="numeric"
                        autoFocus={autoFocus}
                        disabled={disabled}
                        value={displayValue}
                        onChange={(e) => onChange(calculateUpdatedNumber(e.target.value, max, min, allowDecimal))}
                        placeholder={zeroText}
                        data-testid="integer-input"
                        aria-invalid={error ? "true" : undefined}
                        aria-describedby={helperText ? `helper-text-${name}` : undefined}
                        {...inputProps}
                        className="tw-w-full tw-bg-transparent tw-border-0 tw-outline-none focus:tw-outline-none focus:tw-ring-0 tw-text-text-primary placeholder:tw-text-text-secondary"
                    />
                </InputContainer>
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
