import { CircularProgress, Typography } from "@mui/material";
import { Box } from "@mui/system";
import { CharLimitIndicatorProps } from "../../components/inputs/types.js";

/**
 * Displays a label inside of a CircularProgress, indicating 
 * the number of characters left in a text field before 
 * reaching the maximum character limit.
 */
export function CharLimitIndicator({
    chars,
    minCharsToShow,
    maxChars,
    size = 34,
}: CharLimitIndicatorProps) {
    // Calculate remaining characters
    const charsRemaining = maxChars - chars;

    // Calculate percentage of characters used
    const progress = Math.min(100, Math.ceil((chars / maxChars) * 100));

    // Determine the color of CircularProgress based on charsRemaining
    let color;
    if (charsRemaining < 0) {
        color = "error.main"; // red color
    } else if (charsRemaining <= maxChars * 0.2) {
        color = "warning.main"; // yellow color
    } else {
        color = "success.main"; // green color
    }

    if (minCharsToShow !== undefined && chars < minCharsToShow) {
        return null;
    }
    return (
        <Box position="relative" display="inline-flex" sx={{ verticalAlign: "middle" }}>
            <CircularProgress
                variant="determinate"
                size={size}
                value={progress}
                sx={{
                    color,
                }}
            />
            <Box
                top={0}
                left={0}
                bottom={0}
                right={0}
                position="absolute"
                display="flex"
                alignItems="center"
                justifyContent="center"
            >
                <Typography variant="caption" component="div">
                    {charsRemaining}
                </Typography>
            </Box>
        </Box>
    );
}
