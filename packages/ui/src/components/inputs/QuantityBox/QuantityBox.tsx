import { useRef, useCallback, useEffect } from "react";
import { Box, FormControl, IconButton, Input, InputLabel, SxProps, Theme, Tooltip, useTheme } from '@mui/material';
import { QuantityBoxProps } from "../types";
import { MinusIcon, PlusIcon } from "@shared/icons";

const buttonProps: SxProps<Theme> = {
    minWidth: 30,
    width: '20%',
    background: (t) => t.palette.secondary.main,
    '&:hover': {
        background: (t) => t.palette.secondary.dark,
    }
} as const

// Time for a button press to become a hold
const HOLD_DELAY = 750;

type HoldRefs = {
    which: boolean | null; // False for minus button, true for plus button, and null for no button
    speed: number; // Speed that increases the longer you hold the button, with a maximum of 20 per second
    timeout: NodeJS.Timeout | null;
    value: number;
}

export const QuantityBox = ({
    autoFocus = false,
    disabled = false,
    error = false, // TODO use
    handleChange,
    helperText = '', //TODO use
    id,
    key,
    initial = 0,
    label = 'Quantity',
    max = 2097151,
    min = -2097151,
    step = 1,
    tooltip = '',
    value,
    ...props
}: QuantityBoxProps) => {
    const { palette } = useTheme();

    const holdRefs = useRef<HoldRefs>({
        which: null,
        speed: 1,
        timeout: null,
        value: value,
    })
    useEffect(() => {
        holdRefs.current.value = value
    }, [value])

    const updateValue = useCallback((quantity) => {
        if (quantity > max) quantity = max;
        if (quantity < min) quantity = min;
        handleChange(quantity);
    }, [max, min, handleChange]);

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
        updateValue(holdRefs.current.value * 1 - step)
        holdRefs.current.which = false;
        holdRefs.current.timeout = setTimeout(startHold, HOLD_DELAY);
    }, [startHold, step, updateValue]);

    const handlePlusDown = useCallback(() => {
        updateValue(holdRefs.current.value * 1 + step)
        holdRefs.current.which = true;
        holdRefs.current.timeout = setTimeout(startHold, HOLD_DELAY);
    }, [startHold, step, updateValue]);

    const stopTouch = () => {
        if (holdRefs.current.timeout) clearTimeout(holdRefs.current.timeout);
        holdRefs.current.which = null;
        holdRefs.current.speed = 1;
    }

    return (
        <Tooltip title={tooltip}>
            <Box key={key} {...props} sx={{
                ...props?.sx ?? {},
                display: 'flex',
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
                        borderRadius: '5px 0 0 5px',
                        maxWidth: '48px',
                    }}>
                    <MinusIcon />
                </IconButton>
                <FormControl sx={{
                    background: palette.background.paper,
                    width: '60%',
                    maxWidth: `12ch`,
                    height: '100%',
                    display: 'grid',
                    "& input::-webkit-clear-button, & input::-webkit-outer-spin-button, & input::-webkit-inner-spin-button": {
                        display: "none",
                    }
                }}>
                    <InputLabel
                        htmlFor={`quantity-box-${id}`}
                        sx={{
                            color: palette.background.textSecondary,
                            paddingTop: '10px'
                        }}
                    >{label}</InputLabel>
                    <Input
                        autoFocus={autoFocus}
                        disabled={disabled}
                        id={`quantity-box-${id}`}
                        aria-describedby={`helper-text-${id}`}
                        type="number"
                        inputMode="numeric"
                        inputProps={{ 
                            min, 
                            max,
                            pattern: "[0-9]*",
                        }}
                        value={value}
                        onChange={(e) => updateValue(e.target.value)}
                        sx={{
                            color: palette.background.textPrimary,
                        }}
                    />
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
                        borderRadius: '0 5px 5px 0',
                        maxWidth: '48px',
                    }}>
                    <PlusIcon />
                </IconButton>
            </Box>
        </Tooltip>
    );
}