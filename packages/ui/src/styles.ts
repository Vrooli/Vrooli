import { Box, Palette, Stack, styled, SxProps, Theme } from "@mui/material";

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

export const iconButtonProps = {
    background: "transparent",
    border: "1px solid #0fa",
    "&:hover": {
        background: "transparent",
        filter: "brightness(1.2)",
        transform: "scale(1.2)",
    },
    transition: "all 0.2s ease",
} as const;

export const slideText = {
    margin: "auto",
    textAlign: { xs: "left", md: "justify" }, fontSize: { xs: "1.25rem", md: "1.5rem" },
    zIndex: 10,
} as const;

export const slideTitle = {
    fontSize: { xs: "2.4em", sm: "3rem", md: "3.75rem" },
    letterSpacing: "-0.05em",
    textAlign: "center",
    zIndex: 10,
} as const;

export const slideImageContainer = {
    justifyContent: "center",
    height: "100%",
    display: "flex",
    "& > img": {
        maxWidth: {
            xs: "min(225px, 90%)",
            sm: "min(300px, 90%)",
        },
        maxHeight: "100%",
        objectFit: "contain",
        zIndex: "3",
    },
} as const;

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
    [theme.breakpoints.down("sm")]: {
        borderRadius: "0",
        boxShadow: "none",
        width: "100%",
    },
    [theme.breakpoints.up("sm")]: {
        borderRadius: theme.spacing(2),
        boxShadow: theme.shadows[2],
        width: "min(500px, 100vw)",
    },
}));
