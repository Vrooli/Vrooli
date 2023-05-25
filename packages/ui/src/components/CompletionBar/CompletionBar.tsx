import { Box, LinearProgress, Typography } from "@mui/material";
import { CompletionBarProps } from "components/types";

export const CompletionBar = ({
    isLoading = false,
    showLabel = true,
    value,
    ...props
}: CompletionBarProps) => {
    return (
        <Box sx={{ display: "flex", alignItems: "center", pointerEvents: "none" }}>
            <Box sx={{ width: "100%", mr: 1, maxWidth: "300px" }}>
                <LinearProgress
                    value={value}
                    variant={isLoading ? "indeterminate" : "determinate"}
                    {...props}
                    sx={{ borderRadius: 1, height: 12 }}
                />
            </Box>
            {showLabel && <Box sx={{ minWidth: 35 }}>
                <Typography variant="body2">{`${Math.round(value)}%`}</Typography>
            </Box>}
        </Box>
    );
};
