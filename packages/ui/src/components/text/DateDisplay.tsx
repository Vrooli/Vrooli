import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import { useTheme } from "@mui/material/styles";
import { IconCommon } from "../../icons/Icons.js";
import { displayDate } from "../../utils/display/stringTools.js";
import { TextLoading } from "../lists/TextLoading/TextLoading.js";
import { type DateDisplayProps } from "./types.js";

const scheduleIconStyle = {
    marginRight: "4px",
} as const;

/**
 * Displays a date in short format (e.g. "1 hour ago", "yesterday", "June 16", "Jan 1, 2020"). 
 * On hover or press, a popup displays the full date (e.g. "June 16, 2022 at 1:00pm").
 */
export function DateDisplay({
    loading = false,
    showDateAndTime = false,
    showIcon = false,
    textBeforeDate = "",
    timestamp,
    ...props
}: DateDisplayProps) {
    const { palette } = useTheme();

    if (loading && !timestamp) return (
        <TextLoading size="body2" sx={{ width: "100px", ...props }} />
    );
    if (!timestamp) return null;
    return (
        <>
            <Box
                {...props}
                bgcolor={palette.background.textSecondary}
                display="flex"
                justifyContent="center"
                alignItems="center"
            >
                {showIcon && <IconCommon
                    decorative
                    fill={palette.background.textSecondary}
                    name="Schedule"
                    style={scheduleIconStyle}
                />}
                <Typography variant="body2" color={palette.background.textSecondary}>
                    {`${textBeforeDate} ${displayDate(timestamp, showDateAndTime)}`}
                </Typography>
            </Box>
        </>
    );
}
