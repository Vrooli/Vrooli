import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { HelpIcon } from "@local/icons";
import { Box, IconButton, Menu, Tooltip, useTheme } from "@mui/material";
import Markdown from "markdown-to-jsx";
import { useCallback, useState } from "react";
import { linkColors, noSelect } from "../../../styles";
import { MenuTitle } from "../../dialogs/MenuTitle/MenuTitle";
export const HelpButton = ({ id = "help-details-menu", markdown, onClick, sxRoot, sx, }) => {
    const { palette } = useTheme();
    const [anchorEl, setAnchorEl] = useState(null);
    const open = Boolean(anchorEl);
    const openMenu = useCallback((event) => {
        if (onClick)
            onClick(event);
        if (!anchorEl)
            setAnchorEl(event.currentTarget);
    }, [anchorEl, onClick]);
    const closeMenu = () => {
        setAnchorEl(null);
    };
    return (_jsx(Box, { sx: {
            display: "inline",
            ...sxRoot,
        }, children: _jsx(Tooltip, { placement: 'top', title: !open ? "Open Help Menu" : "", children: _jsxs(IconButton, { onClick: openMenu, sx: {
                    display: "inline-flex",
                    bottom: "0",
                    verticalAlign: "top",
                }, children: [_jsx(HelpIcon, { fill: palette.secondary.main, ...sx }), _jsxs(Menu, { id: id, open: open, disableScrollLock: true, anchorEl: anchorEl, onClose: closeMenu, anchorOrigin: {
                            vertical: "bottom",
                            horizontal: "right",
                        }, transformOrigin: {
                            vertical: "top",
                            horizontal: "left",
                        }, sx: {
                            "& .MuiPopover-paper": {
                                background: palette.background.default,
                                maxWidth: "min(90vw, 500px)",
                            },
                            "& .MuiMenu-list": {
                                padding: 0,
                            },
                        }, children: [_jsx(MenuTitle, { onClose: closeMenu }), _jsx(Box, { sx: { padding: 2, ...linkColors(palette), ...noSelect }, children: _jsx(Markdown, { children: markdown }) })] })] }) }) }));
};
//# sourceMappingURL=HelpButton.js.map