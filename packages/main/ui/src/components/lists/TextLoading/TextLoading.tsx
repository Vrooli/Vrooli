import { LinearProgress } from "@mui/material";

/**
 * Placeholder for text of list items, while data is loading.
 */
export function TextLoading(props) {
    return (
        <LinearProgress
            variant={props.variant}
            {...props}
            sx={{
                ...props.sx,
                borderRadius: 1,
                height: 8,
                marginTop: "12px !important",
                marginBottom: "12px !important",
                maxWidth: "300px",
                background: (t) => t.palette.mode === "light" ? "#bbc1c5" : "#57595a",
                "& .MuiLinearProgress-bar": {
                    background: (t) => t.palette.mode === "light" ? "#0000002e" : "#ffffff2e",
                },
            }}
        />
    );
}
