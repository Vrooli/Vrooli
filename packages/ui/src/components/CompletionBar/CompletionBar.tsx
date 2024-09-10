import { Box, LinearProgress, Typography } from "@mui/material";
import { CompletionBarProps } from "components/types";

export function CompletionBar({
    color,
    isLoading = false,
    showLabel = true,
    value,
    sxs,
    ...props
}: CompletionBarProps) {
    return (
        <Box sx={{ display: "flex", alignItems: "center", pointerEvents: "none", ...sxs?.root }}>
            <Box sx={{ width: "100%", mr: 1, maxWidth: "300px", ...sxs?.barBox }}>
                <LinearProgress
                    color={color}
                    value={value}
                    variant={isLoading ? "indeterminate" : "determinate"}
                    {...props}
                    sx={{ borderRadius: 1, height: 12, ...sxs?.bar }}
                />
            </Box>
            {showLabel && <Box sx={{ minWidth: 35 }}>
                <Typography variant="body2" sx={{ ...sxs?.label }}>{`${Math.round(value)}%`}</Typography>
            </Box>}
        </Box>
    );
}
