import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { ReportIcon } from "@local/icons";
import { Box, ListItemText, Stack, useTheme } from "@mui/material";
import { useCallback, useMemo } from "react";
import { multiLineEllipsis } from "../../../styles";
import { getObjectReportUrl } from "../../../utils/navigation/openObject";
import { useLocation } from "../../../utils/route";
export const ReportsButton = ({ reportsCount = 0, object, }) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const link = useMemo(() => object ? getObjectReportUrl(object) : "", [object]);
    const handleClick = useCallback((event) => {
        event.stopPropagation();
        event.preventDefault();
        if (link.length === 0)
            return;
        setLocation(link);
    }, [link, setLocation]);
    return (_jsxs(Stack, { direction: "row", spacing: 0.5, sx: {
            marginRight: 0,
            pointerEvents: "none",
        }, children: [_jsx(Box, { component: "a", href: link, onClick: handleClick, sx: {
                    display: "contents",
                    cursor: "pointer",
                    pointerEvents: "all",
                }, children: _jsx(ReportIcon, { fill: palette.secondary.main }) }), _jsx(ListItemText, { primary: reportsCount, sx: { ...multiLineEllipsis(1), pointerEvents: "none" } })] }));
};
//# sourceMappingURL=ReportsButton.js.map