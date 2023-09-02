/**
 * Prompts user to select which link the new node should be added on
 */
import { Grid, useTheme } from "@mui/material";
import { pagePaddingBottom } from "styles";
import { BottomActionsGridProps } from "../types";

export const BottomActionsGrid = ({
    children,
    display,
    sx,
}: BottomActionsGridProps) => {
    const { palette } = useTheme();

    return (
        <Grid container spacing={2} sx={{
            padding: 2,
            paddingTop: 0,
            marginLeft: display === "page" ? "auto" : 0,
            marginRight: display === "page" ? "auto" : 0,
            maxWidth: display === "page" ? "min(700px, 100%)" : "100%",
            zIndex: 1,
            // Position is sticky when used for a page or for large screens, and static when used for a dialog
            position: { xs: display === "page" ? "sticky" : "fixed", sm: "sticky" },
            // Displayed directly above BottomNav (pages only), which is only visible on mobile
            bottom: { xs: display === "page" ? pagePaddingBottom : 0, md: 0 },
            paddingBottom: display === "page" ? undefined : "calc(12px + env(safe-area-inset-bottom))",
            // Background has transparent blur gradient when used for a page, 
            // and a solid color when used for a dialog
            background: display === "page" ? "transparent" : palette.primary.dark,
            backdropFilter: display === "page" ? "blur(5px)" : undefined,
            "@media print": {
                display: "none",
            },
            ...sx,
        }}
        >
            {children}
        </Grid>
    );
};
