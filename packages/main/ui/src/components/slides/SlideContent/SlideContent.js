import { jsx as _jsx } from "react/jsx-runtime";
import { Stack } from "@mui/material";
export const SlideContent = ({ children, id, sx, }) => {
    return (_jsx(Stack, { id: id, direction: "column", spacing: 4, p: 2, textAlign: "center", zIndex: 5, sx: {
            maxWidth: { xs: "100vw", sm: "90vw", md: "min(80vw, 1000px)" },
            minHeight: "100vh",
            justifyContent: "center",
            margin: "auto",
            ...(sx || {}),
        }, children: children }));
};
//# sourceMappingURL=SlideContent.js.map