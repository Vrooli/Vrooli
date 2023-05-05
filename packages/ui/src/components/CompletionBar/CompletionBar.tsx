import { Box, LinearProgress, Typography } from "@mui/material";

export const CompletionBar = (props) => {
    return (
        <Box sx={{ display: "flex", alignItems: "center", pointerEvents: "none" }}>
            <Box sx={{ width: "100%", mr: 1 }}>
                <LinearProgress variant={props.variant} {...props} sx={{ borderRadius: 1, height: 8 }} />
            </Box>
            <Box sx={{ minWidth: 35 }}>
                <Typography variant="body2" color="text.secondary">{`${Math.round(
                    props.value,
                )}%`}</Typography>
            </Box>
        </Box>
    );
};
