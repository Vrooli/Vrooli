import { Box, Button, IconButton, keyframes, Palette, Stack, styled, SxProps, Theme, Typography } from "@mui/material";

export const centeredDiv = {
    display: "flex",
    justifyContent: "center",
    alignItems: "center",
} as const;

export const textShadow = {
    textShadow:
        `-0.5px -0.5px 0 black,  
        0.5px -0.5px 0 black,
        -0.5px 0.5px 0 black,
        0.5px 0.5px 0 black`,
} as const;

/**
 * Lighthouse recommended size for clickable elements, to improve SEO
 */
export const clickSize = {
    minHeight: "48px",
} as const;

export const multiLineEllipsis = (lines: number) => ({
    display: "-webkit-box",
    WebkitLineClamp: lines,
    WebkitBoxOrient: "vertical",
    overflow: "hidden",
    textOverflow: "ellipsis",
}) as const;

/**
 * Disables text highlighting
 */
export const noSelect = {
    WebkitTouchCallout: "none", /* iOS Safari */
    WebkitUserSelect: "none", /* Safari */
    MozUserSelect: "none",
    msUserSelect: "none", /* Internet Explorer/Edge */
    userSelect: "none", /* Non-prefixed version, currently
    supported by Chrome, Edge, Opera and Firefox */
} as const;

export const linkColors = (palette: Palette) => ({
    a: {
        color: palette.mode === "light" ? "#001cd3" : "#dd86db",
        "&:visited": {
            color: palette.mode === "light" ? "#001cd3" : "#f551ef",
        },
        "&:active": {
            color: palette.mode === "light" ? "#001cd3" : "#f551ef",
        },
        "&:hover": {
            color: palette.mode === "light" ? "#5a6ff6" : "#f3d4f2",
        },
        // Remove underline on links
        textDecoration: "none",
    },
});

export const greenNeonText = {
    color: "#fff",
    filter: "drop-shadow(0 0 1px #0fa) drop-shadow(0 0 2px #0fa) drop-shadow(0 0 20px #0fa)",
} as const;

export const slideIconButton = {
    background: "transparent",
    border: "1px solid #0fa",
    "&:hover": {
        background: "transparent",
        filter: "brightness(1.2)",
        transform: "scale(1.2)",
    },
    transition: "all 0.2s ease",
} as const;
export const SlideIconButton = styled(IconButton)(({ theme }) => ({
    ...slideIconButton,
}));

export const slideText = (theme: Theme) => ({
    zIndex: 10,
    textAlign: "center",
    textWrap: "balance",
    [theme.breakpoints.up("md")]: {
        fontSize: "1.5rem",
    },
    [theme.breakpoints.down("md")]: {
        fontSize: "1.25rem",
    },
} as const);
export const SlideText = styled("h3")(({ theme }) => ({
    ...slideText(theme),
}));

export const slideTitle = (theme: Theme) => ({
    letterSpacing: "-0.05em",
    textAlign: "center",
    wordBreak: "break-word",
    zIndex: 10,
    [theme.breakpoints.up("md")]: {
        fontSize: "3.75rem",
    },
    [theme.breakpoints.up("sm")]: {
        fontSize: "3rem",
    },
    [theme.breakpoints.up("xs")]: {
        fontSize: "2.4rem",
    },
} as const);
export const SlideTitle = styled(Typography)(({ theme }) => ({
    ...slideTitle(theme),
}));

export const slideImageContainer = (theme: Theme) => ({
    justifyContent: "center",
    height: "100%",
    display: "flex",
    "& > img": {
        maxWidth: "min(500px, 100%)",
        maxHeight: "100%",
        zIndex: "3",
    },
} as const);
export const SlideImageContainer = styled(Box)(({ theme }) => ({
    ...slideImageContainer(theme),
}));

export const slideImage = (theme: Theme) => ({
    borderRadius: theme.spacing(4),
    objectFit: "cover",
} as const);
export const SlideImage = styled("img")(({ theme }) => ({
    ...slideImage(theme),
}));

export const slideContent = (theme: Theme) => ({
    display: "flex",
    flexDirection: "column",
    justifyContent: "center",
    marginLeft: "auto",
    marginRight: "auto",
    maxWidth: "700px",
    minHeight: "100vh",
    padding: theme.spacing(2),
    textAlign: "center",
    zIndex: 5,
    "& > *:not(:last-child)": {
        marginBottom: theme.spacing(4),
    },
} as const);
export const SlideContent = styled(Stack)(({ theme }) => ({
    ...slideContent(theme),
}));

export const slideContainer = {
    overflow: "hidden",
    position: "relative",
} as const;
export const SlideContainer = styled(Box)(() => ({
    ...slideContainer,
}));

export const slidePage = {
} as const;
export const SlidePage = styled(Box)(() => ({
    ...slidePage,
}));

export const slideBox = (theme: Theme) => ({
    ...slideContent(theme),
    background: "#2c2d2fd1",
    borderRadius: theme.spacing(4),
    boxShadow: theme.shadows[2],
    minHeight: "unset",
    zIndex: 2,
    "& > *:not(:last-child)": {
        marginBottom: theme.spacing(6),
    },
} as const);
export const SlideBox = styled(Stack)(({ theme }) => ({
    ...slideBox(theme),
}));

export const textPop = {
    padding: "0",
    color: "white",
    textAlign: "center",
    fontWeight: 600,
    textShadow:
        `-1px -1px 0 black,  
                1px -1px 0 black,
                -1px 1px 0 black,
                1px 1px 0 black`,
} as const;

export const formSection = (theme: Theme) => ({
    background: theme.palette.mode === "dark" ?
        theme.palette.background.paper :
        theme.palette.background.default,
    borderRadius: theme.spacing(1),
    flexDirection: "column",
    overflowX: "auto",
    padding: theme.spacing(2),
    "& > *:not(:last-child)": {
        marginBottom: theme.spacing(2),
    },
    "@media print": {
        border: `1px solid ${theme.palette.divider}`,
    },
} as const);
export const FormSection = styled(Stack)(({ theme }) => ({
    ...noSelect,
    ...formSection(theme),
}));


export const formContainer = (theme: Theme): SxProps => ({
    flexDirection: "column",
    margin: theme.spacing(2),
    "& > *:not(:last-child)": {
        marginBottom: theme.spacing(4),
    },
});
export const FormContainer = styled(Stack)(({ theme }) => ({
    ...formContainer(theme),
}));

export const OverviewContainer = styled(Box)(({ theme }) => ({
    position: "relative",
    marginLeft: "auto",
    marginRight: "auto",
    marginTop: theme.spacing(3),
    background: theme.palette.background.paper,
    borderRadius: "0",
    width: `min(${theme.breakpoints.values.sm}px, 100%)`,
    [theme.breakpoints.up("sm")]: {
        borderRadius: theme.spacing(2),
        boxShadow: theme.shadows[2],
    },
}));

const pulse = keyframes`
    0% {
        box-shadow: 0 0 0 0 rgba(0, 255, 170, 0.7);
    }
    70% {
        box-shadow: 0 0 0 10px rgba(0, 255, 170, 0);
    }
    100% {
        box-shadow: 0 0 0 0 rgba(0, 255, 170, 0);
    }
`;

export const PulseButton = styled(Button)(({ theme }) => ({
    fontSize: "1.8rem",
    // Button border has neon green glow animation
    animation: `${pulse} 3s infinite ease`,
    borderColor: "#0fa",
    borderWidth: "2px",
    color: "#0fa",
    fontWeight: 550,
    width: "fit-content",
    // On hover, brighten and grow
    "&:hover": {
        borderColor: "#0fa",
        color: "#0fa",
        background: "transparent",
        filter: "brightness(1.2)",
        transform: "scale(1.05)",
    },
    transition: "all 0.2s ease",
}));
