import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { LinearProgress, Stack, Typography } from "@mui/material";
import { SelectLanguageMenu } from "../../dialogs/SelectLanguageMenu/SelectLanguageMenu";
export const ObjectTitle = ({ language, languages, loading, setLanguage, title, zIndex, }) => {
    const titleComponent = loading ? _jsx(LinearProgress, { color: "inherit", sx: {
            borderRadius: 1,
            width: "50vw",
            height: 8,
            marginTop: "12px !important",
            marginBottom: "12px !important",
            maxWidth: "300px",
        } }) : _jsx(Typography, { component: "h1", variant: "h3", sx: {
            textAlign: "center",
            sx: { marginTop: 2, marginBottom: 2 },
            fontSize: { xs: "1.5rem", sm: "2rem", md: "2.5rem" },
        }, children: title });
    return (_jsxs(Stack, { direction: "row", justifyContent: "center", alignItems: "center", spacing: 2, sx: {
            marginTop: 2,
            marginBottom: 2,
            marginLeft: "auto",
            marginRight: "auto",
            maxWidth: "700px",
        }, children: [titleComponent, _jsx(SelectLanguageMenu, { currentLanguage: language, handleCurrent: setLanguage, languages: languages, zIndex: zIndex })] }));
};
//# sourceMappingURL=ObjectTitle.js.map