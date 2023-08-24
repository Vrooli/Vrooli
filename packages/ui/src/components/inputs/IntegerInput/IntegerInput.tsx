import { Box, FormControl, FormHelperText, IconButton, Input, InputLabel, Tooltip, useTheme } from "@mui/material";
import { useField } from "formik";
import { MinusIcon, PlusIcon } from "icons";
import { useCallback, useEffect, useRef } from "react";
import { IntegerInputProps } from "../types";

const buttonProps = {
    minWidth: "30px",
    maxWidth: "48px",
    width: "20%",
};

// Time for a button press to become a hold
const HOLD_DELAY = 750;

type HoldRefs = {
    which: boolean | null; // False for minus button, true for plus button, and null for no button
    speed: number; // Speed that increases the longer you hold the button, with a maximum of 20 per second
    timeout: NodeJS.Timeout | null;
    value: number;
}

export const IntegerInput = ({
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

    const holdRefs = useRef<HoldRefs>({
        which: null,
        speed: 1,
        timeout: null,
        value: field.value,
    });
    useEffect(() => {
        holdRefs.current.value = field.value;
    }, [field.value]);

    const updateValue = useCallback((quantity) => {
        if (quantity > max) quantity = max;
        if (quantity < min) quantity = min;
        helpers.setValue(quantity);
    }, [max, min, helpers]);

    const startHold = useCallback(() => {
        // Check if hold is taking place
        if (holdRefs.current.which === null) return;
        // Increment for decrement value, depending on which button was pressed
        if (holdRefs.current.which === true) updateValue(holdRefs.current.value + 1);
        else updateValue(holdRefs.current.value - 1);
        // Calculate timeout for next tick. Speed increases with each tick, until max of 20 per second
        holdRefs.current.speed = Math.min(holdRefs.current.speed + 1, 20);
        // Set timeout for next tick
        holdRefs.current.timeout = setTimeout(startHold, 1000 / holdRefs.current.speed);
    }, [updateValue]);

    const handleMinusDown = useCallback(() => {
        updateValue(holdRefs.current.value * 1 - step);
        holdRefs.current.which = false;
        holdRefs.current.timeout = setTimeout(startHold, HOLD_DELAY);
    }, [startHold, step, updateValue]);

    const handlePlusDown = useCallback(() => {
        updateValue(holdRefs.current.value * 1 + step);
        holdRefs.current.which = true;
        holdRefs.current.timeout = setTimeout(startHold, HOLD_DELAY);
    }, [startHold, step, updateValue]);

    const stopTouch = () => {
        if (holdRefs.current.timeout) clearTimeout(holdRefs.current.timeout);
        holdRefs.current.which = null;
        holdRefs.current.speed = 1;
    };

    return (
        <Tooltip title={tooltip}>
            <Box key={key} {...props} sx={{
                display: "flex",
                justifyContent: "center",
                ...props?.sx ?? {},
            }}>
                <IconButton
                    aria-label='minus'
                    disabled={disabled}
                    onMouseDown={handleMinusDown}
                    onMouseUp={stopTouch}
                    onTouchStart={handleMinusDown}
                    onTouchEnd={stopTouch}
                    onContextMenu={(e) => e.preventDefault()}
                    sx={{
                        ...buttonProps,
                        background: palette.secondary.main,
                        borderRadius: "5px 0 0 5px",
                    }}>
                    <MinusIcon />
                </IconButton>
                <FormControl sx={{
                    background: palette.background.paper,
                    width: fullWidth ? "100%" : "60%",
                    maxWidth: fullWidth ? "100%" : "12ch",
                    height: "100%",
                    display: "grid",
                    "& input::-webkit-clear-button, & input::-webkit-outer-spin-button, & input::-webkit-inner-spin-button": {
                        display: "none",
                    },
                }} error={meta.touched && !!meta.error}>
                    <InputLabel
                        htmlFor={`quantity-box-${name}`}
                        sx={{
                            color: (field.value < min || field.value > max) ?
                                palette.error.main :
                                (field.value === min || field.value === max) ?
                                    palette.warning.main :
                                    palette.background.textSecondary,
                            paddingTop: "10px",
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
                <IconButton
                    aria-label='plus'
                    disabled={disabled}
                    onMouseDown={handlePlusDown}
                    onMouseUp={stopTouch}
                    onTouchStart={handlePlusDown}
                    onTouchEnd={stopTouch}
                    onContextMenu={(e) => e.preventDefault()}
                    sx={{
                        ...buttonProps,
                        background: palette.secondary.main,
                        borderRadius: "0 5px 5px 0",
                    }}>
                    <PlusIcon />
                </IconButton>
            </Box>
        </Tooltip >
    );
};
