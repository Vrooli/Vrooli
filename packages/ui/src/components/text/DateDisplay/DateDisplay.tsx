import { Box, Typography, useTheme } from "@mui/material";
import { TextLoading } from "components/lists/TextLoading/TextLoading";
import { ScheduleIcon } from "icons";
import { displayDate } from "utils/display/stringTools";
import { DateDisplayProps } from "../types";

/**
 * Displays a date in short format (e.g. "1 hour ago", "yesterday", "June 16", "Jan 1, 2020"). 
 * On hover or press, a popup displays the full date (e.g. "June 16, 2022 at 1:00pm").
 */
export const DateDisplay = ({
    loading = false,
    showDateAndTime = false,
    showIcon = false,
    textBeforeDate = "",
    timestamp,
    ...props
}: DateDisplayProps) => {
    const { palette } = useTheme();

    if (loading && !timestamp) return (
        <TextLoading size="body2" sx={{ width: "100px", ...props }} />
    );
    if (!timestamp) return null;
    return (
        <>
            <Box
                {...props}
                display="flex"
                justifyContent="center"
                alignItems="center"
                sx={{
                    color: palette.background.textSecondary,
                    ...(props.sx ?? {}),
                }}
            >
                {showIcon && <ScheduleIcon fill={palette.background.textSecondary} style={{ marginRight: "4px" }} />}
                <Typography variant="body2" color={palette.background.textSecondary}>
                    {`${textBeforeDate} ${displayDate(timestamp, showDateAndTime)}`}
                </Typography>
            </Box>
        </>
    );
};
