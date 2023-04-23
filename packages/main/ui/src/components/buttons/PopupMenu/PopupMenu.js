import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { Button, Popover, useTheme } from "@mui/material";
import { useState } from "react";
export function PopupMenu({ text = "Menu", children, ...props }) {
    const { palette } = useTheme();
    const [anchorEl, setAnchorEl] = useState(null);
    const handleClick = (event) => {
        setAnchorEl(event.currentTarget);
    };
    const handleClose = () => {
        setAnchorEl(null);
    };
    const open = Boolean(anchorEl);
    const id = open ? "simple-popover" : undefined;
    return (_jsxs(_Fragment, { children: [_jsx(Button, { "aria-describedby": id, ...props, onClick: handleClick, children: text }), _jsx(Popover, { id: id, open: open, anchorEl: anchorEl, onClose: handleClose, disableScrollLock: true, anchorOrigin: {
                    vertical: "bottom",
                    horizontal: "center",
                }, transformOrigin: {
                    vertical: "top",
                    horizontal: "center",
                }, sx: {
                    "& .MuiPopover-paper": {
                        background: palette.primary.light,
                        borderRadius: "24px",
                    },
                }, children: children })] }));
}
//# sourceMappingURL=PopupMenu.js.map