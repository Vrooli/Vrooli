import { jsx as _jsx } from "react/jsx-runtime";
import { Box } from "@mui/material";
export const PageContainer = ({ children, sx, }) => {
    return (_jsx(Box, { id: "page", sx: {
            minWidth: "100vw",
            minHeight: "100vh",
            width: "min(100%, 700px)",
            margin: "auto",
            paddingBottom: "calc(56px + env(safe-area-inset-bottom))",
            paddingLeft: { xs: 0, sm: "max(1em, calc(15% - 75px))" },
            paddingRight: { xs: 0, sm: "max(1em, calc(15% - 75px))" },
            ...(sx ?? {}),
        }, children: children }));
};
//# sourceMappingURL=PageContainer.js.map