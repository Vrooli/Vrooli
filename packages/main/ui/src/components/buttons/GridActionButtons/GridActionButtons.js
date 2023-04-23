import { jsx as _jsx } from "react/jsx-runtime";
import { Grid, useTheme } from "@mui/material";
export const GridActionButtons = ({ children, display, }) => {
    const { palette } = useTheme();
    return (_jsx(Grid, { container: true, spacing: 2, sx: {
            padding: 2,
            paddingTop: 0,
            marginLeft: "auto",
            marginRight: "auto",
            maxWidth: "min(700px, 100%)",
            left: display === "page" ? undefined : 0,
            zIndex: 1,
            position: { xs: display === "page" ? "sticky" : "fixed", sm: "sticky" },
            bottom: { xs: display === "page" ? "calc(56px + env(safe-area-inset-bottom))" : 0, md: 0 },
            paddingBottom: display === "page" ? undefined : "calc(12px + env(safe-area-inset-bottom))",
            background: display === "page" ? "transparent" : palette.primary.dark,
            backdropFilter: display === "page" ? "blur(5px)" : undefined,
        }, children: children }));
};
//# sourceMappingURL=GridActionButtons.js.map