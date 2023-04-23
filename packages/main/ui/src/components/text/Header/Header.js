import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Stack, Typography } from "@mui/material";
import { noSelect } from "../../../styles";
import { HelpButton } from "../../buttons/HelpButton/HelpButton";
export const Header = ({ help, sxs, title, }) => {
    const titleDisplay = _jsx(Typography, { component: "h1", variant: "h3", sx: {
            textAlign: "center",
            marginTop: 2,
            marginBottom: 2,
            fontSize: { xs: "1.75rem", sm: "2rem", md: "2.5rem" },
            ...noSelect,
            ...(sxs?.text || {}),
        }, children: title });
    if (!help)
        return titleDisplay;
    return (_jsxs(Stack, { direction: "row", justifyContent: "center", alignItems: "center", sx: {
            marginTop: 2,
            marginBottom: 2,
            ...noSelect,
            ...(sxs?.stack || {}),
        }, children: [titleDisplay, _jsx(HelpButton, { markdown: help, sx: { width: "40px", height: "40px" } })] }));
};
//# sourceMappingURL=Header.js.map