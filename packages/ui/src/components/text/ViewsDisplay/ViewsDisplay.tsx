import { VisibleIcon } from "@local/shared";
import { Box, Typography, useTheme } from "@mui/material";
import { ViewsDisplayProps } from "../types";

/**
 * Displays views count for an object.
 */
export const ViewsDisplay = ({
    views,
}: ViewsDisplayProps) => {
    const { palette } = useTheme();

    return (
        <Box
            display="flex"
            alignItems="center"
        >
            <VisibleIcon fill={palette.background.textPrimary} />
            <Typography variant="body2" color={palette.background.textPrimary}>
                {views ?? 1}
            </Typography>
        </Box>
    );
};
