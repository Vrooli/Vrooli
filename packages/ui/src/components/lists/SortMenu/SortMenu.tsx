// Menu for selecting 

import { Menu, MenuItem } from "@mui/material";
import { useMemo } from "react";
import { SortMenuProps } from "../types";

const optionMap = {
    'AlphabeticalAsc': 'Z-A',
    'AlphabeticalDesc': 'A-Z',
    'CommentsDesc': 'Most Comments',
    'StarsDesc': 'Most Stars',
    'ForksDesc': 'Most Forks',
}

export function SortMenu({
    sortOptions,
    anchorEl,
    onClose,
}: SortMenuProps) {
    const open = Boolean(anchorEl);

    const menuItems = useMemo(() => {
        let menuItems: JSX.Element[] = [];
        sortOptions.forEach(option => {
            if (optionMap[option.value]) {
                menuItems.push(
                    <MenuItem
                        key={option.value}
                        value={option.value}
                        onClick={() => onClose(optionMap[option.value], option.value)}
                    >
                        {optionMap[option.value]}
                    </MenuItem>
                );
            }
        });
        return menuItems;
    }, [sortOptions])

    return (
        <Menu
            id="sort-results-menu"
            anchorEl={anchorEl}
            open={open}
            onClose={() => onClose()}
            MenuListProps={{ 'aria-labelledby': 'sort-results-menu-list' }}
        >
            {menuItems}
        </Menu>
    )
}