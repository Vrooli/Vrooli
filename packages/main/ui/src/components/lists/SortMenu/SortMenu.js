import { jsx as _jsx } from "react/jsx-runtime";
import { Menu, MenuItem } from "@mui/material";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
export function SortMenu({ sortOptions, anchorEl, onClose, }) {
    const { t } = useTranslation();
    const open = Boolean(anchorEl);
    const menuItems = useMemo(() => {
        const menuItems = [];
        sortOptions.forEach(option => {
            const optionLabel = t(`${option.value}`);
            if (optionLabel) {
                menuItems.push(_jsx(MenuItem, { value: option.value, onClick: () => onClose(optionLabel, option.value), children: optionLabel }, option.value));
            }
        });
        return menuItems;
    }, [sortOptions, t, onClose]);
    return (_jsx(Menu, { id: "sort-results-menu", disableScrollLock: true, anchorEl: anchorEl, open: open, onClose: () => onClose(), MenuListProps: { "aria-labelledby": "sort-results-menu-list" }, children: menuItems }));
}
//# sourceMappingURL=SortMenu.js.map