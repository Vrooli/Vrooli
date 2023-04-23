import { jsx as _jsx } from "react/jsx-runtime";
import { Box } from "@mui/material";
export const CardGrid = ({ children, disableMargin, minWidth, sx, }) => {
    return (_jsx(Box, { sx: {
            display: "grid",
            gridTemplateColumns: `repeat(auto-fit, minmax(${minWidth}px, 1fr))`,
            alignItems: "stretch",
            gap: 2,
            margin: disableMargin ? 0 : 2,
            borderRadius: 2,
            ...(sx ?? {}),
        }, children: children }));
};
//# sourceMappingURL=CardGrid.js.map