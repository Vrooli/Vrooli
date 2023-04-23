import { jsx as _jsx } from "react/jsx-runtime";
import { LinearProgress } from "@mui/material";
export function TextLoading(props) {
    return (_jsx(LinearProgress, { variant: props.variant, ...props, sx: {
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
        } }));
}
//# sourceMappingURL=TextLoading.js.map