/**
 * Prompts user to select which link the new node should be added on
 */
import Grid from "@mui/material/Grid";
import { useTheme } from "@mui/material/styles";
import { useMemo } from "react";
import { useKeyboardOpen } from "../../hooks/useKeyboardOpen.js";
import { pagePaddingBottom } from "../../styles.js";
import { type BottomActionsGridProps } from "./types.js";

export function BottomActionsGrid({
    children,
    display,
    sx,
    ...props
}: BottomActionsGridProps) {
    const { palette } = useTheme();
    const isKeyboardOpen = useKeyboardOpen();

    const gridStyle = useMemo(function gridStyleMemo() {
        return {
            padding: 0,
            marginLeft: display === "Page" ? "auto" : 0,
            marginRight: display === "Page" ? "auto" : 0,
            maxWidth: display === "Page" ? "min(700px, 100%)" : "100%",
            zIndex: 10,
            position: "sticky", //"absolute",
            // Displayed directly above BottomNav (pages only), which is only visible on mobile
            bottom: { xs: (display === "Page" && !isKeyboardOpen) ? pagePaddingBottom : 0, md: 0 },
            left: 0,
            right: 0,
            // Background has transparent blur gradient when used for a page, 
            // and a solid color when used for a dialog
            background: display === "Page" ? "transparent" : palette.primary.dark,
            backdropFilter: display === "Page" ? "blur(5px)" : undefined,
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
        <Grid container spacing={2} sx={gridStyle} {...props}>
            {children}
        </Grid>
    );
}
