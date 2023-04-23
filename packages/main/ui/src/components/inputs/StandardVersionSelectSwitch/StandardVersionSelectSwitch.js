import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { EditIcon as CustomIcon, LinkIcon } from "@local/icons";
import { Box, Stack, Typography, useTheme } from "@mui/material";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { noSelect } from "../../../styles";
import { getTranslation, getUserLanguages } from "../../../utils/display/translationTools";
import { SessionContext } from "../../../utils/SessionContext";
import { ColorIconButton } from "../../buttons/ColorIconButton/ColorIconButton";
import { FindObjectDialog } from "../../dialogs/FindObjectDialog/FindObjectDialog";
const grey = {
    400: "#BFC7CF",
    800: "#2F3A45",
};
export function StandardVersionSelectSwitch({ selected, onChange, disabled, zIndex, ...props }) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
    const openCreateDialog = useCallback(() => { setIsCreateDialogOpen(true); }, [setIsCreateDialogOpen]);
    const closeCreateDialog = useCallback(() => { setIsCreateDialogOpen(false); }, [setIsCreateDialogOpen]);
    const handleClick = useCallback((ev) => {
        if (disabled)
            return;
        if (Boolean(selected)) {
            onChange(null);
        }
        else {
            openCreateDialog();
            ev.preventDefault();
        }
    }, [disabled, onChange, openCreateDialog, selected]);
    const Icon = useMemo(() => Boolean(selected) ? LinkIcon : CustomIcon, [selected]);
    return (_jsxs(_Fragment, { children: [_jsx(FindObjectDialog, { find: "Full", isOpen: isCreateDialogOpen, handleComplete: onChange, handleCancel: closeCreateDialog, limitTo: ["StandardVersion"], zIndex: zIndex + 1 }), _jsxs(Stack, { direction: "row", spacing: 1, children: [_jsxs(Typography, { variant: "h6", sx: { ...noSelect }, children: [t("Standard", { count: 1 }), ":"] }), _jsxs(Box, { component: "span", sx: {
                            display: "inline-block",
                            position: "relative",
                            width: "64px",
                            height: "36px",
                            padding: "8px",
                        }, children: [_jsx(Box, { component: "span", sx: {
                                    backgroundColor: palette.mode === "dark" ? grey[800] : grey[400],
                                    borderRadius: "16px",
                                    width: "100%",
                                    height: "65%",
                                    display: "block",
                                }, children: _jsx(ColorIconButton, { background: palette.secondary.main, sx: {
                                        display: "inline-flex",
                                        width: "30px",
                                        height: "30px",
                                        position: "absolute",
                                        top: 0,
                                        padding: "4px",
                                        transition: "transform 150ms cubic-bezier(0.4, 0, 0.2, 1)",
                                        transform: `translateX(${Boolean(selected) ? "24" : "0"}px)`,
                                    }, children: _jsx(Icon, { width: '30px', height: '30px', fill: "white" }) }) }), _jsx("input", { type: "checkbox", checked: Boolean(selected), readOnly: true, disabled: disabled, "aria-label": "custom-standard-toggle", onClick: handleClick, style: {
                                    position: "absolute",
                                    width: "100%",
                                    height: "100%",
                                    top: "0",
                                    left: "0",
                                    opacity: "0",
                                    zIndex: "1",
                                    margin: "0",
                                    cursor: "pointer",
                                } })] }), _jsx(Typography, { variant: "h6", sx: { ...noSelect }, children: selected ? getTranslation(selected, getUserLanguages(session)).name : t("Custom") })] })] }));
}
//# sourceMappingURL=StandardVersionSelectSwitch.js.map