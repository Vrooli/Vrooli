import { Box, Typography, useTheme } from "@mui/material";
import { PopoverWithArrow } from "components/dialogs/PopoverWithArrow/PopoverWithArrow";
import { TextLoading } from "components/lists/TextLoading/TextLoading";
import usePress from "hooks/usePress";
import { ScheduleIcon } from "icons";
import { useCallback, useState } from "react";
import { displayDate } from "utils/display/stringTools";
import { DateDisplayProps } from "../types";

/**
 * Displays a date in short format (e.g. "1 hour ago", "yesterday", "June 16", "Jan 1, 2020"). 
 * On hover or press, a popup displays the full date (e.g. "June 16, 2022 at 1:00pm").
 */
export const DateDisplay = ({
    loading = false,
    showIcon = true,
    textBeforeDate = "",
    timestamp,
    ...props
}: DateDisplayProps) => {
    const { palette } = useTheme();

    // Full date popup
    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
    const open = useCallback((target: EventTarget) => {
        setAnchorEl(target as HTMLElement);
    }, []);
    const close = useCallback(() => setAnchorEl(null), []);

    const pressEvents = usePress({
        onHover: open,
        onLongPress: open,
        onClick: open,
    });

    if (loading && !timestamp) return (
        <TextLoading size="body2" sx={{ width: "100px", ...props }} />
    );
    if (!timestamp) return null;
    return (
        <>
            {/* Full date popup */}
            <PopoverWithArrow
                anchorEl={anchorEl}
                handleClose={close}
            >
                <Typography variant="body2" color={palette.background.textPrimary}>
                    {displayDate(timestamp, true)}
                </Typography>
            </PopoverWithArrow>
            {/* Displayed date */}
            <Box
                {...props}
                {...pressEvents}
                display="flex"
                justifyContent="center"
                alignItems="center"
                sx={{
                    color: palette.background.textSecondary,
                    cursor: "pointer",
                    ...(props.sx ?? {}),
                }}
            >
                {showIcon && <ScheduleIcon fill={palette.background.textSecondary} style={{ marginRight: "4px" }} />}
                <Typography variant="body2" color={palette.background.textSecondary}>
                    {`${textBeforeDate} ${displayDate(timestamp, false)}`}
                </Typography>
            </Box>
        </>
    );
};
