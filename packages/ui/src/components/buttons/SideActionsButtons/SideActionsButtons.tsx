import { Box, useTheme } from "@mui/material";
import { useWindowSize } from "hooks/useWindowSize";
import { useZIndex } from "hooks/useZIndex";
import { useEffect, useRef, useState } from "react";
import { SideActionsButtonsProps } from "../types";

export const SideActionsButtons = ({
    children,
    display,
    sx,
}: SideActionsButtonsProps) => {
    const zIndex = useZIndex();
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    const [paddingTop, setPaddingTop] = useState(window.innerHeight);
    const containerRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        // Get top padding needed to be at the bottom of the screen
        let parentContainer: HTMLElement | null = null;
        if (display === "page") {
            parentContainer = containerRef.current?.closest("#content-wrap") ?? null;
        } else if (display === "dialog") {
            parentContainer = containerRef.current?.closest(".MuiDialog-paper") ?? null;
        }
        console.log("parentContainer", parentContainer, containerRef);
        let containerHeight = parentContainer?.offsetHeight ?? window.innerHeight;
        if (containerHeight > window.innerHeight) containerHeight = window.innerHeight;
        if (containerHeight <= 0) containerHeight = window.innerHeight;
        setPaddingTop(containerHeight);
    }, [display]);

    return (
        <Box ref={containerRef} sx={{
            position: "sticky",
            paddingTop: `${paddingTop}px`,
            bottom: 0,
            justifyContent: "flex-end",
            display: "flex",
            flexDirection: "row",
            gap: "16px",
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
        }}>
            {children}
        </Box>
    );
};
