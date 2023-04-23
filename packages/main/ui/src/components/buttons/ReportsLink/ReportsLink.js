import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { ReportIcon } from "@local/icons";
import { IconButton, Tooltip, Typography, useTheme } from "@mui/material";
import { useCallback, useMemo } from "react";
import { getObjectSlug, getObjectUrlBase } from "../../../utils/navigation/openObject";
import { useLocation } from "../../../utils/route";
export const ReportsLink = ({ object, }) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const link = useMemo(() => object ? `${getObjectUrlBase(object)}/reports/${getObjectSlug(object)}` : "", [object]);
    const onClick = useCallback((e) => {
        setLocation(link);
        e.preventDefault();
    }, [link, setLocation]);
    if (!object?.reportsCount || object.reportsCount <= 0)
        return null;
    return (_jsx(Tooltip, { title: "Press to view and repond to reports.", children: _jsxs(IconButton, { href: link, onClick: onClick, children: [_jsx(ReportIcon, { fill: palette.background.textPrimary }), _jsxs(Typography, { variant: "body1", sx: { ml: 0.5, color: palette.background.textPrimary }, children: ["(", object.reportsCount, ")"] })] }) }));
};
//# sourceMappingURL=ReportsLink.js.map