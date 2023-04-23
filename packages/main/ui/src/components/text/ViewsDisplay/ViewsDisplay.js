import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { VisibleIcon } from "@local/icons";
import { Box, Typography, useTheme } from "@mui/material";
export const ViewsDisplay = ({ views, }) => {
    const { palette } = useTheme();
    return (_jsxs(Box, { display: "flex", alignItems: "center", children: [_jsx(VisibleIcon, { fill: palette.background.textPrimary }), _jsx(Typography, { variant: "body2", color: palette.background.textPrimary, children: views ?? 1 })] }));
};
//# sourceMappingURL=ViewsDisplay.js.map