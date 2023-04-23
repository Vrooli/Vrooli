import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Box, List, Typography, useTheme } from "@mui/material";
import { useTranslation } from "react-i18next";
export const ListContainer = ({ children, emptyText, isEmpty = false, sx, }) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    return (_jsxs(Box, { sx: {
            marginTop: 2,
            maxWidth: "1000px",
            marginLeft: "auto",
            marginRight: "auto",
            ...(isEmpty ? {} : {
                boxShadow: 12,
                background: palette.background.paper,
                borderRadius: "8px",
                overflow: "overlay",
                display: "block",
            }),
            ...(sx ?? {}),
        }, children: [isEmpty && (_jsx(Typography, { variant: "h6", textAlign: "center", children: emptyText ?? t("NoResults", { ns: "error" }) })), !isEmpty && (_jsx(List, { sx: { padding: 0 }, children: children }))] }));
};
//# sourceMappingURL=ListContainer.js.map