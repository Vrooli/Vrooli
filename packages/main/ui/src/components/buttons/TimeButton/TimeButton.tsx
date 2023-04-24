import { TimeFrame, HistoryIcon as TimeIcon } from "@local/shared";
import { Box, Tooltip, Typography, useTheme } from "@mui/material";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { TimeMenu } from "../../lists/TimeMenu/TimeMenu";
import { searchButtonStyle } from "../styles";
import { TimeButtonProps } from "../types";

export const TimeButton = ({
    setTimeFrame,
    timeFrame,
}: TimeButtonProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [timeAnchorEl, setTimeAnchorEl] = useState<HTMLElement | null>(null);
    const [timeFrameLabel, setTimeFrameLabel] = useState<string>("");

    const handleTimeOpen = (event: any) => setTimeAnchorEl(event.currentTarget);
    const handleTimeClose = (label?: string, frame?: TimeFrame) => {
        setTimeAnchorEl(null);
        setTimeFrame(frame);
        if (label) setTimeFrameLabel(label === "All Time" ? "" : label);
    };

    return (
        <>
            <TimeMenu
                anchorEl={timeAnchorEl}
                onClose={handleTimeClose}
            />
            <Tooltip title={t("TimeCreated")} placement="top">
                <Box
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
