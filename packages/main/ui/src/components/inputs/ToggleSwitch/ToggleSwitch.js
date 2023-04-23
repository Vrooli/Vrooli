import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Box, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { useCallback, useMemo } from "react";
import { noSelect } from "../../../styles";
import { ColorIconButton } from "../../buttons/ColorIconButton/ColorIconButton";
const grey = {
    400: "#BFC7CF",
    800: "#2F3A45",
};
export function ToggleSwitch({ checked, disabled = false, name, onChange, OffIcon, OnIcon, label, tooltip, sx, }) {
    const { palette } = useTheme();
    const handleChange = useCallback(() => {
        if (disabled)
            return;
        const customEvent = {
            target: {
                name,
                value: !checked,
                type: "checkbox",
            },
        };
        onChange(customEvent);
    }, [disabled, checked, onChange, name]);
    const Icon = useMemo(() => checked ? OnIcon : OffIcon, [checked, OffIcon, OnIcon]);
    return (_jsx(Tooltip, { title: tooltip, children: _jsxs(Stack, { direction: "row", spacing: 1, justifyContent: "center", sx: { ...(sx ?? {}) }, children: [_jsx(Box, { onClick: handleChange, children: _jsx(Typography, { variant: "h6", sx: { ...noSelect }, children: label }) }), _jsxs(Box, { component: "span", sx: {
                        display: "inline-block",
                        position: "relative",
                        width: "64px",
                        height: "36px",
                        padding: "8px",
                        filter: disabled ? "grayscale(0.8)" : "none",
                    }, children: [_jsx(Box, { component: "span", sx: {
                                backgroundColor: checked ? palette.primary.main : (palette.mode === "dark" ? grey[800] : grey[400]),
                                transition: "background-color 150ms ease-in-out",
                                borderRadius: "16px",
                                width: "50px",
                                height: "30px",
                                display: "block",
                                position: "absolute",
                                top: "50%",
                                transform: "translateY(-50%)",
                            }, children: _jsx(ColorIconButton, { background: (OnIcon || OffIcon) ? palette.secondary.main : "white", sx: {
                                    display: "inline-flex",
                                    width: "30px",
                                    height: "30px",
                                    padding: 0,
                                    position: "absolute",
                                    top: "50%",
                                    boxShadow: 2,
                                    transition: "transform 150ms ease-in-out",
                                    transform: `translateX(${checked ? "20" : "0"}px) translateY(-50%)`,
                                }, children: Icon && _jsx(Icon, { fill: "white", width: "80%", height: "80%" }) }) }), _jsx("input", { type: "checkbox", checked: checked, disabled: disabled, "aria-label": "toggle-switch", name: name, onChange: onChange, style: {
                                position: "absolute",
                                width: "100%",
                                height: "100%",
                                top: "0",
                                left: "0",
                                opacity: "0",
                                zIndex: "1",
                                margin: "0",
                                cursor: "pointer",
                            } })] })] }) }));
}
//# sourceMappingURL=ToggleSwitch.js.map