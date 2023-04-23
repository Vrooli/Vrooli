import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { CloseIcon } from "@local/icons";
import { Box, IconButton, useTheme } from "@mui/material";
import { noSelect } from "../../../styles";
import { HelpButton } from "../../buttons/HelpButton/HelpButton";
export const MenuTitle = ({ ariaLabel, helpText, onClose, title, }) => {
    const { palette } = useTheme();
    return (_jsxs(Box, { id: ariaLabel, sx: {
            ...noSelect,
            display: "flex",
            alignItems: "center",
            paddingTop: 1,
            paddingBottom: 1,
            paddingLeft: 2,
            paddingRight: 2,
            background: palette.primary.dark,
            color: palette.primary.contrastText,
            textAlign: "center",
            fontSize: { xs: "1.5rem", sm: "2rem" },
        }, children: [_jsx(Box, { sx: { marginLeft: "auto" }, children: title }), helpText && _jsx(HelpButton, { markdown: helpText, sx: {
                    fill: palette.secondary.light,
                }, sxRoot: {
                    display: "flex",
                    marginTop: "auto",
                    marginBottom: "auto",
                } }), _jsx(IconButton, { "aria-label": "close", edge: "end", onClick: onClose, sx: { marginLeft: "auto" }, children: _jsx(CloseIcon, { fill: palette.primary.contrastText }) })] }));
};
//# sourceMappingURL=MenuTitle.js.map