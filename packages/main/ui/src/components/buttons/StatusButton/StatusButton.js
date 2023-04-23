import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { CloseIcon, RoutineIncompleteIcon, RoutineInvalidIcon, RoutineValidIcon } from "@local/icons";
import { Box, IconButton, Menu, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import Markdown from "markdown-to-jsx";
import { useCallback, useMemo, useState } from "react";
import { noSelect } from "../../../styles";
import { Status } from "../../../utils/consts";
const STATUS_COLOR = {
    [Status.Incomplete]: "#a0b121",
    [Status.Invalid]: "#ff6a6a",
    [Status.Valid]: "#00d51e",
};
const STATUS_LABEL = {
    [Status.Incomplete]: "Incomplete",
    [Status.Invalid]: "Invalid",
    [Status.Valid]: "Valid",
};
const STATUS_ICON = {
    [Status.Incomplete]: RoutineIncompleteIcon,
    [Status.Invalid]: RoutineInvalidIcon,
    [Status.Valid]: RoutineValidIcon,
};
export const StatusButton = ({ status, messages, sx, }) => {
    const { palette } = useTheme();
    const statusMarkdown = useMemo(() => {
        if (messages.length === 0)
            return "Routine is valid.";
        if (messages.length === 1) {
            return messages[0];
        }
        return messages.map((s) => {
            return `* ${s}`;
        }).join("\n");
    }, [messages]);
    const StatusIcon = useMemo(() => STATUS_ICON[status], [status]);
    const [anchorEl, setAnchorEl] = useState(null);
    const open = Boolean(anchorEl);
    const openMenu = useCallback((event) => {
        if (!anchorEl)
            setAnchorEl(event.currentTarget);
    }, [anchorEl]);
    const closeMenu = () => {
        setAnchorEl(null);
    };
    const menu = useMemo(() => (_jsxs(Box, { children: [_jsx(Box, { sx: { background: palette.primary.dark }, children: _jsx(IconButton, { edge: "end", color: "inherit", onClick: closeMenu, "aria-label": "close", children: _jsx(CloseIcon, { fill: palette.primary.contrastText }) }) }), _jsx(Box, { sx: { padding: 1 }, children: _jsx(Markdown, { children: statusMarkdown }) })] })), [statusMarkdown, palette.primary.dark, palette.primary.contrastText]);
    return (_jsxs(_Fragment, { children: [_jsx(Tooltip, { title: 'Press for details', children: _jsxs(Stack, { direction: "row", spacing: 1, onClick: openMenu, sx: {
                        ...noSelect,
                        display: "flex",
                        justifyContent: "center",
                        alignItems: "center",
                        cursor: "pointer",
                        background: STATUS_COLOR[status],
                        color: "white",
                        padding: "4px",
                        borderRadius: "16px",
                        ...(sx ?? {}),
                    }, children: [_jsx(StatusIcon, { fill: 'white' }), _jsx(Typography, { variant: 'body2', sx: {
                                display: { xs: "none", sm: "inline" },
                                paddingRight: "4px",
                            }, children: STATUS_LABEL[status] })] }) }), _jsx(Menu, { id: 'status-menu', open: open, disableScrollLock: true, anchorEl: anchorEl, onClose: closeMenu, anchorOrigin: {
                    vertical: "bottom",
                    horizontal: "center",
                }, transformOrigin: {
                    vertical: "top",
                    horizontal: "center",
                }, sx: {
                    "& .MuiPopover-paper": {
                        background: palette.background.default,
                        maxWidth: "min(100vw, 400px)",
                    },
                    "& .MuiMenu-list": {
                        padding: 0,
                    },
                }, children: menu })] }));
};
//# sourceMappingURL=StatusButton.js.map