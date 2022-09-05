// When there are too many tags, they should scroll horizontally 

import { Palette } from "@mui/material";

// with a nonintrusive scrollbar
export const smallHorizontalScrollbar = (palette: Palette) => ({
    overflowX: 'auto',
    "&::-webkit-scrollbar": {
        height: 5,
    },
    "&::-webkit-scrollbar-track": {
        backgroundColor: "transparent",
    },
    "&::-webkit-scrollbar-thumb": {
        borderRadius: '100px',
        backgroundColor: palette.background.textSecondary,
    },
})