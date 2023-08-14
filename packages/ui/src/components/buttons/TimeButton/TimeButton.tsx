import { CommonKey, TimeFrame } from "@local/shared";
import { Box, Menu, MenuItem, Tooltip, Typography, useTheme } from "@mui/material";
import { DateRangeMenu } from "components/lists/DateRangeMenu/DateRangeMenu";
import { HistoryIcon as TimeIcon } from "icons";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { searchButtonStyle } from "../styles";
import { TimeButtonProps } from "../types";

/** Map time selections to time length in milliseconds */
const timeOptions = {
    "TimeAll": undefined,
    "TimeYear": 31536000000,
    "TimeMonth": 2592000000,
    "TimeWeek": 604800000,
    "TimeDay": 86400000,
    "TimeHour": 3600000,
} as const;

const TimeMenu = ({
    anchorEl,
    onClose,
    zIndex,
}: {
    anchorEl: HTMLElement | null;
    onClose: (labelKey?: CommonKey, timeFrame?: { after?: Date, before?: Date }) => void;
    zIndex: number;
}) => {
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
            MenuListProps={{
                "aria-label": "Filters search results by time created or updated",
                "aria-describedby": "time-filter-button",
            }}
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
};


export const TimeButton = ({
    setTimeFrame,
    timeFrame,
    zIndex,
}: TimeButtonProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [timeAnchorEl, setTimeAnchorEl] = useState<HTMLElement | null>(null);
    const [timeFrameLabel, setTimeFrameLabel] = useState<string>("");

    const handleTimeOpen = (event: { currentTarget: HTMLElement }) => setTimeAnchorEl(event.currentTarget);
    const handleTimeClose = (labelKey?: CommonKey, frame?: TimeFrame) => {
        setTimeAnchorEl(null);
        setTimeFrame(frame);
        if (labelKey) setTimeFrameLabel(t(labelKey));
    };

    return (
        <>
            <TimeMenu
                anchorEl={timeAnchorEl}
                onClose={handleTimeClose}
                zIndex={zIndex + 1}
            />
            <Tooltip title={t("TimeCreated")} placement="top">
                <Box
                    id="time-filter-button"
                    onClick={handleTimeOpen}
                    sx={searchButtonStyle(palette)}
                >
                    <TimeIcon fill={palette.secondary.main} />
                    <Typography variant="body2" sx={{ marginLeft: 0.5 }}>{timeFrameLabel}</Typography>
                </Box>
            </Tooltip>
        </>
    );
};
