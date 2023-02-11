// Menu for selecting 

import { Menu, MenuItem } from "@mui/material";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { SortMenuProps } from "../types";

export function SortMenu({
    sortOptions,
    anchorEl,
    lng,
    onClose,
}: SortMenuProps) {
    const { t } = useTranslation();
    const open = Boolean(anchorEl);

    const menuItems = useMemo(() => {
        let menuItems: JSX.Element[] = [];
        sortOptions.forEach(option => {
            const optionLabel = t(`common:${option.value}`, { lng })
            if (optionLabel) {
                menuItems.push(
                    <MenuItem
                        key={option.value}
                        value={option.value}
                        onClick={() => onClose(optionLabel, option.value)}
                    >
                        {optionLabel}
                    </MenuItem>
                );
            }
        });
        return menuItems;
    }, [sortOptions, t, lng, onClose])

    return (
        <Menu
            id="sort-results-menu"
            disableScrollLock={true}
            anchorEl={anchorEl}
            open={open}
            onClose={() => onClose()}
            MenuListProps={{ 'aria-labelledby': 'sort-results-menu-list' }}
        >
            {menuItems}
        </Menu>
    )
}