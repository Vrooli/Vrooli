import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Stack, Typography, useTheme } from "@mui/material";
import { noSelect } from "../../../styles";
import { HelpButton } from "../../buttons/HelpButton/HelpButton";
export const Subheader = ({ help, Icon, sxs, title, }) => {
    const { palette } = useTheme();
    const titleDisplay = _jsx(Typography, { component: "h2", variant: "h4", sx: {
            textAlign: "center",
            marginTop: 2,
            marginBottom: 2,
            fontSize: { xs: "1.5rem", sm: "1.75rem", md: "2rem" },
            ...noSelect,
            ...(sxs?.text || {}),
        }, children: title });
    if (!help && !Icon)
        return titleDisplay;
    return (_jsxs(Stack, { direction: "row", justifyContent: "center", alignItems: "center", sx: {
            marginTop: 2,
            marginBottom: 2,
            ...noSelect,
            ...(sxs?.stack || {}),
        }, children: [Icon && _jsx(Icon, { fill: palette.background.textPrimary, style: { width: "30px", height: "30px", marginRight: 8 } }), titleDisplay, help && _jsx(HelpButton, { markdown: help, sx: { width: "30px", height: "30px" } })] }));
};
//# sourceMappingURL=Subheader.js.map