import { Box, BoxProps, IconButton, Tooltip, Typography, styled, useTheme } from "@mui/material";
import { useCallback, useEffect, useRef, useState } from "react";
import { SxType } from "types";
import { useWindowSize } from "../../hooks/useWindowSize.js";
import { multiLineEllipsis } from "../../styles.js";
import { fontSizeToPixels } from "../../utils/display/stringTools.js";
import { HelpButton } from "../buttons/HelpButton/HelpButton.js";
import { TitleProps } from "./types.js";

interface OuterBoxProps extends BoxProps {
    addSidePadding: boolean;
    numberOfLines: number;
    stackSx?: SxType;
}
const OuterBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "addSidePadding" && prop !== "numberOfLines" && prop !== "stackSx",
})<OuterBoxProps>(({ addSidePadding, numberOfLines, stackSx, theme }) => ({
    display: "flex",
    flexDirection: numberOfLines >= 2 ? "column" : "row",
    justifyContent: "flex-start",
    alignItems: "center",
    paddingTop: theme.spacing(1),
    paddingBottom: theme.spacing(1),
    paddingLeft: addSidePadding ? theme.spacing(2) : 0,
    paddingRight: addSidePadding ? theme.spacing(2) : 0,
    wordBreak: "break-word",
    ...(stackSx as object),
}));

export function Title({
    addSidePadding = true,
    adornments,
    help,
    Icon,
    options,
    sxs,
    title,
    titleComponent,
    variant = "header",
}: TitleProps) {
    const { breakpoints, palette, typography } = useTheme();

    const textRef = useRef<HTMLDivElement>(null);
    const [numberOfLines, setNumberOfLines] = useState(1);
    const findLineCount = useCallback(() => {
        const titleElement = textRef.current;
        if (titleElement) {
            const lineHeight = parseInt(window.getComputedStyle(titleElement).getPropertyValue("line-height"), 10);
            if (lineHeight > 0) {
                setNumberOfLines(Math.round(titleElement.clientHeight / lineHeight));
            }
        }
    }, []);
    useEffect(() => {
        findLineCount();
        window.addEventListener("resize", findLineCount);
        return () => window.removeEventListener("resize", findLineCount);
    }, [findLineCount, title]);

    // Determine title size based on variant and screen size
    const matchesXS = useWindowSize(({ width }) => width <= breakpoints.values.xs);
    const matchesSM = useWindowSize(({ width }) => width <= breakpoints.values.sm);
    let fontSize: string;
    if (matchesXS) {
        fontSize = variant === "header" ? "1.75rem" : variant === "subheader" ? "1.5rem" : "1.25rem";
    } else if (matchesSM) {
        fontSize = variant === "header" ? "2rem" : variant === "subheader" ? "1.75rem" : "1.5rem";
    } else {
        fontSize = variant === "header" ? "2.5rem" : variant === "subheader" ? "2rem" : "1.75rem";
    }

    return (
        <OuterBox addSidePadding={addSidePadding} numberOfLines={numberOfLines} stackSx={sxs?.stack}>
            <Box display="flex" alignItems="center">
                {/* Icon */}
                {Icon && <Icon fill={palette.background.textPrimary} style={{ width: "30px", height: "30px", marginRight: 8 }} />}
                {/* Title */}
                {titleComponent ?? <Typography
                    ref={textRef}
                    component={variant === "header" ? "h1" : variant === "subheader" ? "h2" : "h3"}
                    variant={variant === "header" ? "h3" : variant === "subheader" ? "h4" : "h5"}
                    textAlign="center"
                    sx={{
                        fontSize,
                        ...multiLineEllipsis(3),
                        ...sxs?.text,
                    }}
                >{title}</Typography>}
                {/* Adornments */}
                {adornments && adornments.map(({ Adornment, key }) => (
                    <Box key={key} sx={{
                        height: fontSizeToPixels(fontSize ?? "1rem") * Number(typography[variant === "header" ? "h3" : "h4"].lineHeight ?? "1.5"),
                    }}>
                        {Adornment}
                    </Box>
                ))}
            </Box>
            <Box display="flex">
                {/* Help button */}
                {help && help.length > 0 ? <HelpButton
                    markdown={help}
                    sx={{
                        width: variant === "header" ? "40px" : variant === "subheader" ? "30px" : "25px",
                        height: variant === "header" ? "40px" : variant === "subheader" ? "30px" : "25px",
                    }}
                /> : null}
                {/* Additional options */}
                {options?.filter(({ Icon }) => Icon).map(({ Icon, label, onClick }, index) => (
                    <Tooltip key={`title-option-${index}`} title={label}>
                        <IconButton onClick={onClick}>
                            {Icon && <Icon fill={palette.secondary.main} style={{ width: "30px", height: "30px" }} />}
                        </IconButton>
                    </Tooltip>
                ))}
            </Box>
        </OuterBox>
    );
}
