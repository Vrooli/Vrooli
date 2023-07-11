// Menu for selecting the time contrainsts for a search

import { CommonKey } from "@local/shared";
import { Menu, MenuItem } from "@mui/material";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { DateRangeMenu } from "../DateRangeMenu/DateRangeMenu";
import { TimeMenuProps } from "../types";

/**
 * Map time selections to time length in milliseconds
 */
const timeOptions = {
    "TimeAll": undefined,
    "TimeYear": 31536000000,
    "TimeMonth": 2592000000,
    "TimeWeek": 604800000,
    "TimeDay": 86400000,
    "TimeHour": 3600000,
} as const;

export function TimeMenu({
    anchorEl,
    onClose,
    zIndex,
}: TimeMenuProps) {
    const { t } = useTranslation();

    const open = Boolean(anchorEl);

    const [customRangeAnchorEl, setCustomRangeAnchorEl] = useState<HTMLElement | null>(null);
    const handleTimeOpen = (event) => setCustomRangeAnchorEl(event.currentTarget);
    const handleTimeClose = () => {
        setCustomRangeAnchorEl(null);
    };

    const menuItems = useMemo(() => Object.keys(timeOptions).map((labelKey) => (
        <MenuItem
            key={labelKey}
            value={timeOptions[labelKey]}
            onClick={() => {
                // If All is selected, pass undefined to onClose
                if (!timeOptions[labelKey]) onClose(labelKey as CommonKey);
                // Otherwise, pass the time as object with "after"
                else onClose(labelKey as CommonKey, { after: new Date(Date.now() - timeOptions[labelKey]) });
            }}
        >
            {t(labelKey as CommonKey)}
        </MenuItem>
    )), [onClose, t]);

    return (
        <Menu
            id="results-time-menu"
            anchorEl={anchorEl}
            disableScrollLock={true}
            open={open}
            onClose={() => onClose()}
            MenuListProps={{ "aria-labelledby": "results-time-menu-list" }}
        >
            {menuItems}
            <MenuItem
                id='custom-range-menu-item'
                value='custom'
                onClick={handleTimeOpen}
            >
                {t("CustomRange")}
            </MenuItem>
            <DateRangeMenu
                anchorEl={customRangeAnchorEl}
                onClose={handleTimeClose}
                onSubmit={(after, before) => onClose("Custom", { after, before })}
                zIndex={zIndex + 1}
            />
        </Menu>
    );
}
