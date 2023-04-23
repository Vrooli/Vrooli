import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { CloseIcon } from "@local/icons";
import { Box, DialogTitle as MuiDialogTitle, IconButton, useTheme } from "@mui/material";
import { forwardRef } from "react";
import { noSelect } from "../../../styles";
import { HelpButton } from "../../buttons/HelpButton/HelpButton";
export const DialogTitle = forwardRef(({ below, helpText, id, onClose, title, }, ref) => {
    const { palette } = useTheme();
    return (_jsxs(Box, { ref: ref, sx: {
            background: palette.primary.dark,
            color: palette.primary.contrastText,
        }, children: [_jsxs(MuiDialogTitle, { id: id, sx: {
                    ...noSelect,
                    display: "flex",
                    alignItems: "center",
                    padding: 2,
                    textAlign: "center",
                    fontSize: { xs: "1.5rem", sm: "2rem" },
                }, children: [_jsx(Box, { sx: { marginLeft: "auto" }, children: title }), helpText && _jsx(HelpButton, { markdown: helpText, sx: {
                            fill: palette.secondary.light,
                        }, sxRoot: {
                            display: "flex",
                            marginTop: "auto",
                            marginBottom: "auto",
                        } }), _jsx(IconButton, { "aria-label": "close", edge: "end", onClick: onClose, sx: { marginLeft: "auto" }, children: _jsx(CloseIcon, { fill: palette.primary.contrastText }) })] }), below] }));
});
//# sourceMappingURL=DialogTitle.js.map