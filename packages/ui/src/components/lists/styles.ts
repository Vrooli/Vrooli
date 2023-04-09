// When there are too many tags, they should scroll horizontally 

import { Palette, SxProps } from "@mui/material";

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

export const cardRoot: SxProps = {
    boxShadow: 6,
    background: (t: any) => t.palette.primary.light,
    color: (t: any) => t.palette.primary.contrastText,
    borderRadius: '16px',
    margin: 0,
    cursor: 'pointer',
    maxWidth: '500px',
    '&:hover': {
        filter: `brightness(120%)`,
        transition: 'filter 0.2s',
    },
}