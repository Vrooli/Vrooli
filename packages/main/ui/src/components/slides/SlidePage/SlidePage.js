import { jsx as _jsx } from "react/jsx-runtime";
import { Box } from "@mui/material";
export const SlidePage = ({ children, id, sx, }) => {
    return (_jsx(Box, { id: id, sx: {
            scrollBehavior: "smooth",
            ...(sx || {}),
        }, children: children }));
};
//# sourceMappingURL=SlidePage.js.map