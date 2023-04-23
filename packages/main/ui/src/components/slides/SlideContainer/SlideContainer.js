import { jsx as _jsx } from "react/jsx-runtime";
import { Box } from "@mui/material";
export const SlideContainer = ({ id, children, sx, }) => {
    return (_jsx(Box, { id: id, sx: {
            position: "relative",
            overflow: "hidden",
            scrollSnapAlign: "start",
            ...sx,
        }, children: children }, id));
};
//# sourceMappingURL=SlideContainer.js.map