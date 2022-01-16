// Menu for selecting 

import { Menu, MenuItem } from "@mui/material";
import { useMemo } from "react";
import { SortValueToLabelMap } from "utils";
import { SortMenuProps } from "../types";

export function SortMenu({
    sortOptions,
    anchorEl,
    onClose,
}: SortMenuProps) {
    const open = Boolean(anchorEl);

    const menuItems = useMemo(() => {
        let menuItems: JSX.Element[] = [];
        sortOptions.forEach(option => {
            const optionLabel = SortValueToLabelMap[option.value];
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
    }, [sortOptions, onClose])

    return (
        <Menu
            id="sort-results-menu"
            disableScrollLock
            anchorEl={anchorEl}
            open={open}
            onClose={() => onClose()}
            MenuListProps={{ 'aria-labelledby': 'sort-results-menu-list' }}
        >
            {menuItems}
        </Menu>
    )
}