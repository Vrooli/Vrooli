import { jsx as _jsx } from "react/jsx-runtime";
import { RefreshIcon } from "@local/icons";
import { Box, IconButton, useTheme } from "@mui/material";
import { useEffect, useRef, useState } from "react";
import { getDeviceInfo } from "../../utils/display/device";
export const PullToRefresh = () => {
    const { palette } = useTheme();
    const [iconSize, setIconSize] = useState(20);
    const [willRefresh, setWillRefresh] = useState(false);
    const refs = useRef({
        willRefresh: false,
        startX: null,
    });
    const { isStandalone } = getDeviceInfo();
    useEffect(() => {
        const handleScroll = () => {
            const scrollY = window.scrollY;
            if (scrollY < -175 && !refs.current.willRefresh) {
                setWillRefresh(true);
                refs.current.willRefresh = true;
                setIconSize(32);
            }
            else if (scrollY < -50 && !refs.current.willRefresh) {
                setIconSize(Math.round(20 + (scrollY / -14.5)));
            }
            if (scrollY === 0 && refs.current.willRefresh) {
                refs.current.willRefresh = false;
                window.location.reload();
            }
        };
        window.addEventListener("scroll", handleScroll);
        return () => {
            window.removeEventListener("scroll", handleScroll);
        };
    }, []);
    if (!isStandalone)
        return null;
    return (_jsx(Box, { id: 'pull-to-refresh', sx: {
            position: "fixed",
            top: 0,
            left: 0,
            right: 0,
            paddingTop: { xs: "96px", md: "112px" },
            paddingBottom: 40,
            height: "56px",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            zIndex: -1,
            background: palette.background.default,
        }, children: _jsx(IconButton, { sx: {
                background: palette.background.paper,
                transform: willRefresh ? "rotate(360deg)" : "rotate(0deg)",
                transition: "transform 500ms cubic-bezier(0.4, 0, 0.2, 1)",
            }, children: _jsx(RefreshIcon, { fill: palette.background.textPrimary, width: iconSize, height: iconSize }) }) }));
};
//# sourceMappingURL=PullToRefresh.js.map