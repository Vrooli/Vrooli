import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { ScheduleIcon } from "@local/icons";
import { Box, LinearProgress, Typography, useTheme } from "@mui/material";
import { useCallback, useState } from "react";
import { displayDate } from "../../../utils/display/stringTools";
import usePress from "../../../utils/hooks/usePress";
import { PopoverWithArrow } from "../../dialogs/PopoverWithArrow/PopoverWithArrow";
export const DateDisplay = ({ loading = false, showIcon = true, textBeforeDate = "", timestamp, ...props }) => {
    const { palette } = useTheme();
    const [anchorEl, setAnchorEl] = useState(null);
    const open = useCallback((target) => {
        setAnchorEl(target);
    }, []);
    const close = useCallback(() => setAnchorEl(null), []);
    const pressEvents = usePress({
        onHover: open,
        onLongPress: open,
        onClick: open,
    });
    if (loading)
        return (_jsx(Box, { ...props, children: _jsx(LinearProgress, { color: "inherit", sx: { height: "6px", borderRadius: "12px" } }) }));
    if (!timestamp)
        return null;
    return (_jsxs(_Fragment, { children: [_jsx(PopoverWithArrow, { anchorEl: anchorEl, handleClose: close, children: _jsx(Typography, { variant: "body2", color: palette.background.textPrimary, children: displayDate(timestamp, true) }) }), _jsxs(Box, { ...props, ...pressEvents, display: "flex", justifyContent: "center", sx: {
                    ...(props.sx ?? {}),
                    cursor: "pointer",
                }, children: [showIcon && _jsx(ScheduleIcon, { fill: palette.background.textPrimary }), `${textBeforeDate} ${displayDate(timestamp, false)}`] })] }));
};
//# sourceMappingURL=DateDisplay.js.map