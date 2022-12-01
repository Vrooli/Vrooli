import { Palette, SxProps } from "@mui/material"

export const centeredDiv: SxProps = {
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center'
};

export const textShadow: SxProps = {
    textShadow:
        `-0.5px -0.5px 0 black,  
        0.5px -0.5px 0 black,
        -0.5px 0.5px 0 black,
        0.5px 0.5px 0 black`
};

/**
 * Lighthouse recommended size for clickable elements, to improve SEO
 */
export const clickSize: SxProps = {
    minHeight: '48px',
};

export const multiLineEllipsis = (lines: number): SxProps => ({
    display: '-webkit-box',
    WebkitLineClamp: lines,
    WebkitBoxOrient: 'vertical',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
});

/**
 * Disables text highlighting
 */
export const noSelect: SxProps = {
    WebkitTouchCallout: 'none', /* iOS Safari */
    WebkitUserSelect: 'none', /* Safari */
    MozUserSelect: 'none',
    msUserSelect: 'none', /* Internet Explorer/Edge */
    userSelect: 'none', /* Non-prefixed version, currently
    supported by Chrome, Edge, Opera and Firefox */
};

export const linkColors = (palette: Palette) => ({
    a: {
        color: palette.mode === 'light' ? '#001cd3' : '#dd86db',
        '&:visited': {
            color: palette.mode === 'light' ? '#001cd3' : '#f551ef',
        },
        '&:active': {
            color: palette.mode === 'light' ? '#001cd3' : '#f551ef',
        },
        '&:hover': {
            color: palette.mode === 'light' ? '#5a6ff6' : '#f3d4f2',
        },
        // Remove underline on links
        textDecoration: 'none',
    },
})