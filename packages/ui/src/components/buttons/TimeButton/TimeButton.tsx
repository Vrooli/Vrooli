import { TimeFrame, TranslationKeyCommon } from "@local/shared";
import { Box, Menu, MenuItem, Tooltip, Typography, useTheme } from "@mui/material";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { DateRangeMenu } from "../../../components/lists/DateRangeMenu/DateRangeMenu.js";
import { usePopover } from "../../../hooks/usePopover.js";
import { HistoryIcon as TimeIcon } from "../../../icons/common.js";
import { searchButtonStyle } from "../../../styles.js";
import { TimeButtonProps } from "../types.js";

/** Map time selections to time length in milliseconds */
const timeOptions = {
    "TimeAll": undefined,
    "TimeYear": 31536000000,
    "TimeMonth": 2592000000,
    "TimeWeek": 604800000,
    "TimeDay": 86400000,
    "TimeHour": 3600000,
} as const;

interface TimeMenuProps {
    anchorEl: Element | null;
    onClose: (labelKey?: TranslationKeyCommon, timeFrame?: { after?: Date, before?: Date }) => unknown;
}

function TimeMenu({
    anchorEl,
    onClose,
}: TimeMenuProps) {
    const { t } = useTranslation();

    const open = Boolean(anchorEl);

    const [customRangeAnchorEl, openCustomRange, closeCustomRange] = usePopover();

    const menuItems = useMemo(() => Object.keys(timeOptions).map((labelKey) => (
        <MenuItem
            key={labelKey}
            value={timeOptions[labelKey]}
            onClick={() => {
                // If All is selected, pass undefined to onClose
                if (!timeOptions[labelKey]) onClose(labelKey as TranslationKeyCommon);
                // Otherwise, pass the time as object with "after"
                else onClose(labelKey as TranslationKeyCommon, { after: new Date(Date.now() - timeOptions[labelKey]) });
            }}
        >
            {t(labelKey as TranslationKeyCommon)}
        </MenuItem>
    )), [onClose, t]);

    return (
        <Menu
            id="results-time-menu"
            anchorEl={anchorEl}
            disableScrollLock={true}
            open={open}
            onClose={() => onClose()}
            MenuListProps={{
                "aria-label": "Filters search results by time created or updated",
                "aria-describedby": "time-filter-button",
            }}
        >
            {menuItems}
            <MenuItem
                id='custom-range-menu-item'
                value='custom'
                onClick={openCustomRange}
            >
                {t("CustomRange")}
            </MenuItem>
            <DateRangeMenu
                anchorEl={customRangeAnchorEl}
                onClose={closeCustomRange}
                onSubmit={(after, before) => onClose("Custom", { after, before })}
            />
        </Menu>
    );
}


export function TimeButton({
    setTimeFrame,
    timeFrame,
}: TimeButtonProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [timeFrameLabel, setTimeFrameLabel] = useState<string>("");

    const [timeAnchorEl, openTime, closeTime] = usePopover();
    function handleTimeClose(labelKey?: TranslationKeyCommon, frame?: TimeFrame) {
        closeTime();
        setTimeFrame(frame);
        if (labelKey) setTimeFrameLabel(t(labelKey));
    }

    return (
        <>
            <TimeMenu
                anchorEl={timeAnchorEl}
                onClose={handleTimeClose}
            />
            <Tooltip title={t("TimeCreated")} placement="top">
                <Box
                    id="time-filter-button"
                    onClick={openTime}
                    sx={searchButtonStyle(palette)}
                >
                    <TimeIcon fill={palette.secondary.main} />
                    <Typography variant="body2" sx={{ marginLeft: 0.5 }}>{timeFrameLabel}</Typography>
                </Box>
            </Tooltip>
        </>
    );
}
