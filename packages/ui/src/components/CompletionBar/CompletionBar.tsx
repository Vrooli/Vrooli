import { Box, BoxProps, LinearProgress, LinearProgressProps, TypographyProps, styled } from "@mui/material";
import { CompletionBarProps } from "components/types";
import { CSSProperties } from "react";
import { SxType } from "types";

const MAX_WIDTH = "300px";

interface OuterBoxProps extends BoxProps {
    customStyle?: SxType;
}
const OuterBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "customStyle",
})<OuterBoxProps>(({ customStyle }) => ({
    display: "flex",
    flexDirection: "row",
    alignItems: "center",
    pointerEvents: "none",
    maxWidth: MAX_WIDTH,
    ...(customStyle as CSSProperties),
}));

interface ProgressBarBoxProps extends BoxProps {
    customStyle?: SxType;
}
const ProgressBarBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "customStyle",
})<ProgressBarBoxProps>(({ customStyle, theme }) => ({
    width: "100%",
    marginRight: theme.spacing(1),
    maxWidth: MAX_WIDTH,
    ...(customStyle as CSSProperties),
}));

interface ProgressBarProps extends LinearProgressProps {
    customStyle?: SxType;
}
const ProgressBar = styled(LinearProgress, {
    shouldForwardProp: (prop) => prop !== "customStyle",
})<ProgressBarProps>(({ customStyle, theme }) => ({
    borderRadius: theme.spacing(1),
    height: theme.spacing(1.5),
    ...(customStyle as CSSProperties),
}));

interface PercentageLabelProps extends TypographyProps {
    customStyle?: SxType;
}
const PercentageLabel = styled(Box, {
    shouldForwardProp: (prop) => prop !== "customStyle",
})<PercentageLabelProps>(({ customStyle, theme }) => ({
    color: theme.palette.background.textSecondary,
    ...(customStyle as CSSProperties),
}));

export function CompletionBar({
    color,
    isLoading = false,
    showLabel = true,
    value,
    sxs,
    ...props
}: CompletionBarProps) {

    return (
        <OuterBox customStyle={sxs?.root}>
            <ProgressBarBox customStyle={sxs?.barBox}>
                <ProgressBar
                    color={color}
                    customStyle={sxs?.bar}
                    value={value}
                    variant={isLoading ? "indeterminate" : "determinate"}
                    {...props}
                />
            </ProgressBarBox>
            {showLabel && <Box sx={{ minWidth: 35 }}>
                <PercentageLabel
                    customStyle={sxs?.label}
                    variant="body2"
                >
                    {`${Math.round(value)}%`}
                </PercentageLabel>
            </Box>}
        </OuterBox>
    );
}
