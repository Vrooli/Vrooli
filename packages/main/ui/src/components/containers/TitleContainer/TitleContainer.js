import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Box, CircularProgress, Link, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { useTranslation } from "react-i18next";
import { clickSize } from "../../../styles";
import { HelpButton } from "../../buttons/HelpButton/HelpButton";
export function TitleContainer({ id, helpKey, helpVariables, titleKey, titleVariables, onClick, loading = false, tooltip = "", options = [], sx, children, }) {
    const { palette } = useTheme();
    const { t } = useTranslation();
    return (_jsx(Tooltip, { placement: "bottom", title: tooltip, children: _jsx(Box, { id: id, display: "flex", justifyContent: "center", children: _jsxs(Box, { sx: {
                    boxShadow: 4,
                    borderRadius: { xs: 0, sm: 2 },
                    overflow: "overlay",
                    background: palette.background.paper,
                    width: "min(100%, 700px)",
                    cursor: onClick ? "pointer" : "default",
                    "&:hover": {
                        filter: `brightness(onClick ? ${102} : ${100}%)`,
                    },
                    ...sx,
                }, children: [_jsx(Box, { onClick: (e) => { onClick && onClick(e); }, sx: {
                            background: palette.primary.dark,
                            color: palette.primary.contrastText,
                            padding: 0.5,
                        }, children: _jsxs(Stack, { direction: "row", justifyContent: "center", alignItems: "center", children: [_jsx(Typography, { component: "h2", variant: "h4", textAlign: "center", children: t(titleKey, { ...titleVariables, defaultValue: titleKey }) }), Boolean(helpKey) ? _jsx(HelpButton, { markdown: t(helpKey, { ...helpVariables, defaultValue: helpKey }) }) : null] }) }), _jsxs(Stack, { direction: "column", children: [_jsx(Box, { sx: {
                                    ...(loading ? {
                                        minHeight: "min(300px, 25vh)",
                                        display: "flex",
                                        justifyContent: "center",
                                        alignItems: "center",
                                    } : {
                                        display: "block",
                                    }),
                                }, children: loading ? _jsx(CircularProgress, { color: "secondary" }) : children }), options.length > 0 && (_jsx(Stack, { direction: "row", sx: {
                                    ...clickSize,
                                    justifyContent: "end",
                                }, children: options.map(([labelData, onClick], index) => (_jsx(Link, { onClick: onClick, sx: {
                                        marginTop: "auto",
                                        marginBottom: "auto",
                                        marginRight: 2,
                                    }, children: _jsx(Typography, { sx: {
                                            color: palette.mode === "light" ? palette.secondary.dark : palette.secondary.light,
                                        }, children: typeof labelData === "string" ? t(labelData) : t(labelData.key, { ...labelData.variables, defaultValue: labelData.key }) }) }, index))) }))] })] }) }) }));
}
//# sourceMappingURL=TitleContainer.js.map