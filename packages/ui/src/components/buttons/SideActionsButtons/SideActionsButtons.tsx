import { Box, BoxProps, styled, useTheme } from "@mui/material";
import { useRef } from "react";
import { useWindowSize } from "../../../hooks/useWindowSize.js";
import { SxType, ViewDisplayType } from "../../../types.js";
import { Z_INDEX } from "../../../utils/consts.js";
import { SideActionsButtonsProps } from "../types.js";

interface OuterBoxProps extends BoxProps {
    display: ViewDisplayType;
    isMobile: boolean;
    sx?: SxType;
    zIndex: number;
}

const OuterBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "display" && prop !== "isMobile" && prop !== "sx" && prop !== "zIndex",
})<OuterBoxProps>(({ display, isMobile, sx, theme, zIndex }) => ({
    position: "sticky",
    bottom: 0,
    justifyContent: "flex-end",
    display: "flex",
    flexDirection: "row",
    gap: theme.spacing(2),
    alignItems: "end",
    zIndex,
    paddingLeft: "calc(16px + env(safe-area-inset-left))",
    paddingRight: "calc(16px + env(safe-area-inset-right))",
    height: "calc(64px + env(safe-area-inset-bottom))",
    width: "100%",
    pointerEvents: "none",
    "& > *": {
        marginBottom: `calc(16px + ${isMobile && display === "page" ? "56px" : "0px"} + env(safe-area-inset-bottom))!important`,
        pointerEvents: "auto",
    },
    ...(sx ?? {}),
} as any));

export function SideActionsButtons({
    children,
    display,
    sx,
}: SideActionsButtonsProps) {
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    const containerRef = useRef<HTMLDivElement>(null);

    return (
        <OuterBox
            ref={containerRef}
            display={display}
            isMobile={isMobile}
            sx={sx}
            zIndex={Z_INDEX.ActionButton}
        >
            {children}
        </OuterBox>
    );
}
