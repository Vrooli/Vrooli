import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { MinusIcon, PlusIcon } from "@local/icons";
import { Box, FormControl, FormHelperText, Input, InputLabel, Tooltip, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback, useEffect, useRef } from "react";
import { ColorIconButton } from "../../buttons/ColorIconButton/ColorIconButton";
const buttonProps = {
    minWidth: 30,
    width: "20%",
};
const HOLD_DELAY = 750;
export const IntegerInput = ({ autoFocus = false, disabled = false, key, initial = 0, label = "Number", max = 2097151, min = -2097151, name, step = 1, tooltip = "", ...props }) => {
    const { palette } = useTheme();
    const [field, meta, helpers] = useField(name);
    const holdRefs = useRef({
        which: null,
        speed: 1,
        timeout: null,
        value: field.value,
    });
    useEffect(() => {
        holdRefs.current.value = field.value;
    }, [field.value]);
    const updateValue = useCallback((quantity) => {
        if (quantity > max)
            quantity = max;
        if (quantity < min)
            quantity = min;
        helpers.setValue(quantity);
    }, [max, min, helpers]);
    const startHold = useCallback(() => {
        if (holdRefs.current.which === null)
            return;
        if (holdRefs.current.which === true)
            updateValue(holdRefs.current.value + 1);
        else
            updateValue(holdRefs.current.value - 1);
        holdRefs.current.speed = Math.min(holdRefs.current.speed + 1, 20);
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
        if (holdRefs.current.timeout)
            clearTimeout(holdRefs.current.timeout);
        holdRefs.current.which = null;
        holdRefs.current.speed = 1;
    };
    return (_jsx(Tooltip, { title: tooltip, children: _jsxs(Box, { ...props, sx: {
                display: "flex",
                justifyContent: "center",
                ...props?.sx ?? {},
            }, children: [_jsx(ColorIconButton, { "aria-label": 'minus', background: palette.secondary.main, disabled: disabled, onMouseDown: handleMinusDown, onMouseUp: stopTouch, onTouchStart: handleMinusDown, onTouchEnd: stopTouch, onContextMenu: (e) => e.preventDefault(), sx: {
                        ...buttonProps,
                        borderRadius: "5px 0 0 5px",
                        maxWidth: "48px",
                    }, children: _jsx(MinusIcon, {}) }), _jsxs(FormControl, { sx: {
                        background: palette.background.paper,
                        width: "60%",
                        maxWidth: "12ch",
                        height: "100%",
                        display: "grid",
                        "& input::-webkit-clear-button, & input::-webkit-outer-spin-button, & input::-webkit-inner-spin-button": {
                            display: "none",
                        },
                    }, error: meta.touched && !!meta.error, children: [_jsx(InputLabel, { htmlFor: `quantity-box-${name}`, sx: {
                                color: palette.background.textSecondary,
                                paddingTop: "10px",
                            }, children: label }), _jsx(Input, { autoFocus: autoFocus, disabled: disabled, id: `quantity-box-${name}`, name: name, "aria-describedby": `helper-text-${name}`, type: "number", inputMode: "numeric", inputProps: {
                                min,
                                max,
                                pattern: "[0-9]*",
                            }, value: field.value ?? 0, onChange: (e) => updateValue(e.target.value), sx: {
                                color: palette.background.textPrimary,
                            } }), meta.touched && meta.error && _jsx(FormHelperText, { id: `helper-text-${name}`, children: meta.error })] }), _jsx(ColorIconButton, { "aria-label": 'plus', background: palette.secondary.main, disabled: disabled, onMouseDown: handlePlusDown, onMouseUp: stopTouch, onTouchStart: handlePlusDown, onTouchEnd: stopTouch, onContextMenu: (e) => e.preventDefault(), sx: {
                        ...buttonProps,
                        borderRadius: "0 5px 5px 0",
                        maxWidth: "48px",
                    }, children: _jsx(PlusIcon, {}) })] }, key) }));
};
//# sourceMappingURL=IntegerInput.js.map