/**
 * Displays a date in short format (e.g. "1 hour ago", "yesterday", "June 16", "Jan 1, 2020"). 
 * On hover or press, a popup displays the full date (e.g. "June 16, 2022 at 1:00pm").
 */
import { Box, LinearProgress, Popover, Typography, useTheme } from "@mui/material";
import { DateDisplayProps } from "../types";
import { Today as CalendarIcon } from "@mui/icons-material";
import { displayDate, usePress } from "utils";
import { useCallback, useState } from "react";

export const DateDisplay = ({
    loading = false,
    showIcon = true,
    textBeforeDate = '',
    timestamp,
    ...props
}: DateDisplayProps) => {
    const { palette } = useTheme();
    const shadowColor = palette.mode === 'light' ? '0 0 0' : '255 255 255';

    // Full date popup
    const [anchorEl, setAnchorEl] = useState<any | null>(null);
    const isOpen = Boolean(anchorEl);
    const open = useCallback((target: React.MouseEvent['target']) => {
        setAnchorEl(target)
    }, []);
    const close = useCallback(() => setAnchorEl(null), []);

    const pressEvents = usePress({ 
        onHover: open,
        onHoverEnd: close,
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
            <Popover
                open={isOpen}
                anchorEl={anchorEl}
                onClose={close}
                anchorOrigin={{
                    vertical: 'top',
                    horizontal: 'center',
                }}
                transformOrigin={{
                    vertical: 'bottom',
                    horizontal: 'center',
                }}
                sx={{
                    '& .MuiPopover-paper': {
                        padding: 1,
                        overflow: 'unset',
                        background: palette.background.paper,
                        color: palette.background.textPrimary,
                        boxShadow: `0px 5px 5px -3px rgb(${shadowColor} / 20%), 
                        0px 8px 10px 1px rgb(${shadowColor} / 14%), 
                        0px 3px 14px 2px rgb(${shadowColor} / 12%)`
                    }
                }}
            >
                <Box>
                    <Typography variant="body2" color="textSecondary">
                        {displayDate(timestamp, true)}
                    </Typography>
                    {/* Triangle placed below popper */}
                    <Box sx={{
                        width: '0',
                        height: '0',
                        borderLeft: '10px solid transparent',
                        borderRight: '10px solid transparent',
                        borderTop: `10px solid ${palette.background.paper}`,
                        position: 'absolute',
                        bottom: '-10px',
                        left: '50%',
                        transform: 'translateX(-50%)',
                    }} />

                </Box>
            </Popover>
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