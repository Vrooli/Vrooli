// Menu for selecting the time contrainsts for a search

import { Menu, MenuItem } from "@mui/material";
import { useMemo, useState } from "react";
import { TimeMenuProps } from "../types";
import { DateRangeMenu } from 'components';

/**
 * Map time selections to time length in milliseconds
 */
const timeOptions = {
    'All Time': null,
    'Past Year': 31536000000,
    'Past Month': 2592000000,
    'Past Week': 604800000,
    'Past 24 Hours': 86400000,
    'Past Hour': 3600000,
}

export function TimeMenu({
    anchorEl,
    onClose,
}: TimeMenuProps) {
    const open = Boolean(anchorEl);

    const [customRangeAnchorEl, setCustomRangeAnchorEl] = useState<HTMLElement | null>(null);

    const handleTimeOpen = (event) => setCustomRangeAnchorEl(event.currentTarget);
    const handleTimeClose = () => {
        setCustomRangeAnchorEl(null)
    };

    const menuItems = useMemo(() => Object.keys(timeOptions).map((label: string) => (
        <MenuItem
            key={label}
            value={timeOptions[label]}
            onClick={() => onClose(label.replace('Past ', ''), (new Date(Date.now() - timeOptions[label])))}
        >
            {label}
        </MenuItem>
    )), [])

    return (
        <Menu
            id="results-time-menu"
            anchorEl={anchorEl}
            disableScrollLock={true}
            open={open}
            onClose={() => onClose()}
            MenuListProps={{ 'aria-labelledby': 'results-time-menu-list' }}
        >
            {menuItems}
            <MenuItem
                id='custom-range-menu-item'
                value='custom'
                onClick={handleTimeOpen}
            >
                Custom Range...
            </MenuItem>
            <DateRangeMenu
                anchorEl={customRangeAnchorEl}
                onClose={handleTimeClose}
                onSubmit={(after, before) => onClose('Custom', after, before)}
            />
        </Menu>
    )
}