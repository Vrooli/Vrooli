/**
 * Prompts user to select which link the new node should be added on
 */
import { Grid, useTheme } from "@mui/material";
import { useMemo } from "react";
import { useKeyboardOpen } from "../../hooks/useKeyboardOpen.js";
import { pagePaddingBottom } from "../../styles.js";
import { BottomActionsGridProps } from "./types.js";

export function BottomActionsGrid({
    children,
    display,
    sx,
}: BottomActionsGridProps) {
    const { palette } = useTheme();
    const isKeyboardOpen = useKeyboardOpen();

    const gridStyle = useMemo(function gridStyleMemo() {
        return {
            padding: 0,
            marginLeft: display === "page" ? "auto" : 0,
            marginRight: display === "page" ? "auto" : 0,
            maxWidth: "min(700px, 100%)",
            zIndex: 10,
            position: "absolute",
            // Displayed directly above BottomNav (pages only), which is only visible on mobile
            bottom: { xs: (display === "page" && !isKeyboardOpen) ? pagePaddingBottom : 0, md: 0 },
            left: 0,
            right: 0,
            // Background has transparent blur gradient when used for a page, 
            // and a solid color when used for a dialog
            background: display === "page" ? "transparent" : palette.primary.dark,
            backdropFilter: display === "page" ? "blur(5px)" : undefined,
            "@media print": {
                display: "none",
            },
            "& > .MuiGrid-item": {
                paddingLeft: 1,
                paddingTop: 1,
                paddingRight: 1,
                paddingBottom: 1,
            },
            ...sx,
        } as const;
    }, [display, isKeyboardOpen, palette.primary.dark, sx]);

    return (
        <Grid container spacing={2} sx={gridStyle}>
            {children}
        </Grid>
    );
}
