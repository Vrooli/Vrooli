import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { IconButton, List, ListItem, ListItemIcon, ListItemText, Menu, useTheme } from "@mui/material";
import { useMemo } from "react";
import { HelpButton } from "../../buttons/HelpButton/HelpButton";
import { MenuTitle } from "../MenuTitle/MenuTitle";
const titleId = "list-menu-title";
export function ListMenu({ id, anchorEl, onSelect, onClose, title, data, zIndex, }) {
    const { palette } = useTheme();
    const open = Boolean(anchorEl);
    const items = useMemo(() => data?.map(({ label, value, Icon, iconColor, preview, helpData }, index) => {
        const itemText = _jsx(ListItemText, { primary: label, secondary: preview ? "Coming Soon" : null, sx: {
                "& .MuiListItemText-secondary": {
                    color: "red",
                },
            } });
        const fill = !iconColor || ["default", "unset"].includes(iconColor) ? palette.background.textSecondary : iconColor;
        const itemIcon = Icon ? (_jsx(ListItemIcon, { children: _jsx(Icon, { fill: fill }) })) : null;
        const helpIcon = helpData ? (_jsx(IconButton, { edge: "end", onClick: (e) => e.stopPropagation(), children: _jsx(HelpButton, { ...helpData }) })) : null;
        return (_jsxs(ListItem, { disabled: preview, button: true, onClick: () => { onSelect(value); onClose(); }, children: [itemIcon, itemText, helpIcon] }, index));
    }), [data, onClose, onSelect, palette.background.textSecondary]);
    return (_jsxs(Menu, { id: id, disableScrollLock: true, autoFocus: true, open: open, anchorEl: anchorEl, anchorOrigin: {
            vertical: "bottom",
            horizontal: "center",
        }, transformOrigin: {
            vertical: "top",
            horizontal: "center",
        }, onClose: (e) => { onClose(); }, sx: {
            zIndex,
            "& .MuiMenu-paper": {
                background: palette.background.default,
            },
            "& .MuiMenu-list": {
                paddingTop: "0",
            },
        }, children: [title && _jsx(MenuTitle, { ariaLabel: titleId, title: title, onClose: () => { onClose(); } }), _jsx(List, { children: items })] }));
}
//# sourceMappingURL=ListMenu.js.map