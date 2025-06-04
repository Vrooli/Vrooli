import { LinearProgress, type LinearProgressProps, type TypographyProps } from "@mui/material";

interface TextLoadingProps extends LinearProgressProps {
    /** Number of lines (progress bars) to be rendered */
    lines?: number;
    /** Pass in text size so we can determine height */
    size?: TypographyProps["variant"] | "header" | "subheader";
}

/** Converts text size to height */
const sizeToHeight = (size: TypographyProps["variant"] | "header" | "subheader") => {
    switch (size) {
        case "h1":
            return "1.5rem";
        case "h2":
            return "1.3rem";
        case "h3":
        case "header":
            return "1.2rem";
        case "h4":
        case "subheader":
            return "1.1rem";
        case "h5":
            return "1rem";
        case "h6":
            return "1rem";
        case "subtitle1":
            return "0.8rem";
        case "subtitle2":
            return "0.7rem";
        case "body1":
            return "0.8rem";
        case "body2":
            return "0.7rem";
        case "caption":
            return "0.7rem";
        case "button":
            return "0.8rem";
        case "overline":
            return "0.7rem";
        default:
            return "1rem";
    }
};

/**
 * Placeholder for text of list items, while data is loading.
 */
export function TextLoading({
    lines = 1,
    size = "body1",
    ...props
}: TextLoadingProps) {
    return (
        <>
            {Array.from({ length: lines }, (_, i) => (
                <LinearProgress
                    key={`text-loading-${i}`}
                    {...props}
                    sx={{
                        borderRadius: 2,
                        color: "grey",
                        height: sizeToHeight(size),
                        marginTop: "12px !important",
                        marginBottom: "12px !important",
                        maxWidth: "300px",
                        background: (t) => t.palette.mode === "light" ? "#bbc1c5" : "#57595a",
                        "& .MuiLinearProgress-bar": {
                            background: (t) => t.palette.mode === "light" ? "#0000002e" : "#ffffff2e",
                        },
                        ...props.sx,
                    }}
                />
            ))}
        </>
    );
}
