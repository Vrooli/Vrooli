import { Box, LinearProgress, Typography } from "@mui/material";
import { DateDisplayProps } from "../types";
import { Today as CalendarIcon } from "@mui/icons-material";
import { displayDate, usePress } from "utils";
import { useCallback, useState } from "react";
import { PopoverWithArrow } from "components/dialogs";

/**
 * Displays a date in short format (e.g. "1 hour ago", "yesterday", "June 16", "Jan 1, 2020"). 
 * On hover or press, a popup displays the full date (e.g. "June 16, 2022 at 1:00pm").
 */
export const DateDisplay = ({
    loading = false,
    showIcon = true,
    textBeforeDate = '',
    timestamp,
    ...props
}: DateDisplayProps) => {

    // Full date popup
    const [anchorEl, setAnchorEl] = useState<any | null>(null);
    const open = useCallback((target: React.MouseEvent['target']) => {
        setAnchorEl(target)
    }, []);
    const close = useCallback(() => setAnchorEl(null), []);

    const pressEvents = usePress({
        onHover: open,
        onLongPress: open,
        onClick: open,
    });

    if (loading) return (
        <Box {...props}>
            <LinearProgress color="inherit" sx={{ height: '6px', borderRadius: '12px' }} />
        </Box>
    );
    if (!timestamp) return null;
    return (
        <>
            {/* Full date popup */}
            <PopoverWithArrow
                anchorEl={anchorEl}
                handleClose={close}
            >
                <Typography variant="body2" color="textSecondary">
                    {displayDate(timestamp, true)}
                </Typography>
            </PopoverWithArrow>
            {/* Displayed date */}
            <Box
                {...props}
                {...pressEvents}
                display="flex"
                justifyContent="center"
                sx={{
                    ...(props.sx ?? {}),
                    cursor: 'pointer',
                }}
            >
                {showIcon && <CalendarIcon />}
                {`${textBeforeDate} ${displayDate(timestamp, false)}`}
            </Box>
        </>
    )
}