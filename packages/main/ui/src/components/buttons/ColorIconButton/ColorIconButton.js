import { jsx as _jsx } from "react/jsx-runtime";
import { IconButton } from "@mui/material";
import { forwardRef } from "react";
const buttonSx = (background, disabled) => ({
    background,
    pointerEvents: disabled ? "none" : "auto",
    filter: disabled ? "grayscale(1) opacity(0.5)" : "none",
    transition: "filter 0.2s ease-in-out",
    "&:hover": {
        background,
        filter: disabled ? "grayscale(1) opacity(0.5)" : "brightness(1.2)",
    },
});
export const ColorIconButton = forwardRef(({ background, children, disabled, href, onClick, sx, ...props }, ref) => {
    if (href)
        return (_jsx(IconButton, { component: "a", href: href, onClick: onClick, sx: {
                ...buttonSx(background, disabled),
                ...(sx ?? {}),
            }, children: children }));
    return (_jsx(IconButton, { ...props, onClick: onClick, sx: {
            ...buttonSx(background, disabled),
            ...(sx ?? {}),
        }, children: children }));
});
//# sourceMappingURL=ColorIconButton.js.map