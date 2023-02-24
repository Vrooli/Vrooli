import { Box, Tooltip, Typography, useTheme } from "@mui/material";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getUserLanguages } from "utils";
import { searchButtonStyle } from "../styles";
import { TimeButtonProps } from "../types";
import { HistoryIcon as TimeIcon } from '@shared/icons';
import { TimeFrame } from "@shared/consts";
import { TimeMenu } from "components/lists";

export const TimeButton = ({
    setTimeFrame,
    session,
    timeFrame,
}: TimeButtonProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const lng = useMemo(() => getUserLanguages(session)[0], [session]);

    const [timeAnchorEl, setTimeAnchorEl] = useState<HTMLElement | null>(null);
    const [timeFrameLabel, setTimeFrameLabel] = useState<string>('');

    const handleTimeOpen = (event) => setTimeAnchorEl(event.currentTarget);
    const handleTimeClose = (label?: string, frame?: TimeFrame) => {
        setTimeAnchorEl(null);
        setTimeFrame(frame);
        if (label) setTimeFrameLabel(label === 'All Time' ? '' : label);
    };

    return (
        <>
            <TimeMenu
                anchorEl={timeAnchorEl}
                onClose={handleTimeClose}
                session={session}
            />
            <Tooltip title={t(`common:TimeCreated`, { lng })} placement="top">
                <Box
                    onClick={handleTimeOpen}
                    sx={searchButtonStyle(palette)}
                >
                    <TimeIcon fill={palette.secondary.main} />
                    <Typography variant="body2" sx={{ marginLeft: 0.5 }}>{timeFrameLabel}</Typography>
                </Box>
            </Tooltip>
        </>
    )
}