import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Box, LinearProgress, List, ListItem, ListItemText, Tooltip, Typography } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import usePress from "../../../utils/hooks/usePress";
import { addSearchParams, useLocation } from "../../../utils/route";
import { PopoverWithArrow } from "../../dialogs/PopoverWithArrow/PopoverWithArrow";
export const VersionDisplay = ({ confirmVersionChange, currentVersion, loading = false, prefix = "", versions = [], ...props }) => {
    const [, setLocation] = useLocation();
    const handleVersionChange = useCallback((version) => {
        addSearchParams(setLocation, { version });
        window.location.reload();
    }, [setLocation]);
    const openVersion = useCallback((version) => {
        if (typeof confirmVersionChange === "function") {
            confirmVersionChange(() => { handleVersionChange(version); });
        }
        else {
            handleVersionChange(version);
        }
    }, [confirmVersionChange, handleVersionChange]);
    const listItems = useMemo(() => versions?.map((version, index) => {
        return (_jsx(ListItem, { button: true, disabled: version.versionLabel === currentVersion?.versionLabel, onClick: () => { openVersion(version.versionLabel); }, sx: {
                padding: "0px 8px",
            }, children: _jsx(ListItemText, { primary: version.versionLabel }) }, index));
    }), [currentVersion, openVersion, versions]);
    const [anchorEl, setAnchorEl] = useState(null);
    const open = useCallback((target) => {
        if (versions.length > 1) {
            setAnchorEl(target);
        }
    }, [versions.length]);
    const close = useCallback(() => setAnchorEl(null), []);
    const pressEvents = usePress({
        onLongPress: open,
        onClick: open,
    });
    if (!currentVersion)
        return null;
    if (loading)
        return (_jsxs(Box, { sx: {
                ...(props.sx ?? {}),
                display: "flex",
                justifyContent: "center",
                alignItems: "center",
            }, children: [prefix && _jsx(Typography, { variant: "body1", sx: { marginRight: "4px" }, children: prefix }), _jsx(LinearProgress, { color: "inherit", sx: {
                        width: "40px",
                        height: "6px",
                        borderRadius: "12px",
                    } })] }));
    return (_jsxs(Box, { sx: { ...(props.sx ?? {}) }, children: [_jsx(PopoverWithArrow, { anchorEl: anchorEl, handleClose: close, sxs: {
                    content: {
                        maxHeight: "120px",
                    },
                }, children: _jsx(List, { children: listItems }) }), _jsx(Tooltip, { title: versions.length > 1 ? "Press to change version" : "", children: _jsx(Typography, { ...pressEvents, variant: "body1", sx: {
                        cursor: listItems.length > 1 ? "pointer" : "default",
                    }, children: `${prefix}${currentVersion.versionLabel}` }) })] }));
};
//# sourceMappingURL=VersionDisplay.js.map