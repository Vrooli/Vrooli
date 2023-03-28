import { Palette, SxProps } from "@mui/material";
import { CSSProperties } from "@mui/styles";

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

export const greenNeonText = {
    color: '#fff',
    textShadow: '0 0 7px #fff, 0 0 10px #fff, 0 0 21px #0fa, 0 0 42px #0fa, 0 0 82px #0fa, 0 0 92px #0fa, 0 0 102px #0fa',
}

export const iconButtonProps = {
    background: 'transparent',
    border: '1px solid #0fa',
    '&:hover': {
        background: 'transparent',
        filter: 'brightness(1.2)',
        transform: 'scale(1.2)',
    },
    transition: 'all 0.2s ease',
}

export const slideText: SxProps = {
    margin: 'auto',
    textAlign: { xs: 'left', md: 'justify' }, fontSize: { xs: '1.25rem', md: '1.5rem' },
    zIndex: 10,
};

export const slideTitle: SxProps = {
    textAlign: 'center',
    fontSize: { xs: '2.4em', sm: '3rem', md: '3.75rem' },
    zIndex: 10,
};

export const slideImageContainer: SxProps = {
    justifyContent: 'center',
    height: '100%',
    display: 'flex',
    '& > img': {
        maxWidth: 'min(300px, 90%)',
        maxHeight: '100%',
        objectFit: 'contain',
        zIndex: '3',
    }
} as CSSProperties;

export const textPop: SxProps = {
    padding: '0',
    color: 'white',
    textAlign: 'center',
    fontWeight: 600,
    textShadow:
        `-1px -1px 0 black,  
                1px -1px 0 black,
                -1px 1px 0 black,
                1px 1px 0 black`
} as CSSProperties;