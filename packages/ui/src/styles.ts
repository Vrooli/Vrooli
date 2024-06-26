/* eslint-disable no-magic-numbers */
import { Avatar, Box, IconButton, Palette, Stack, styled, SxProps, Theme, Typography } from "@mui/material";

export const pagePaddingBottom = "calc(56px + env(safe-area-inset-bottom))";

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
    lineHeight: "1.5", // Without this, WebkitLineClamp might accidentally include the top of the first line cut off
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
    gap: theme.spacing(4),
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
    gap: theme.spacing(6),
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

export const baseSection = (theme: Theme) => ({
    background: theme.palette.mode === "dark" ?
        theme.palette.background.paper :
        theme.palette.background.default,
    borderRadius: theme.spacing(1),
    flexDirection: "column",
    padding: theme.spacing(2),
    "@media print": {
        border: `1px solid ${theme.palette.divider}`,
    },
} as const);
export const BaseSection = styled(Box)(({ theme }) => ({
    ...baseSection(theme),
}));

export function formSection(theme: Theme) {
    return {
        ...baseSection(theme),
        overflowX: "auto",
        gap: theme.spacing(2),
        padding: theme.spacing(2),
        // Smaller padding on mobile
        [theme.breakpoints.down("sm")]: {
            padding: theme.spacing(1.5),
        },
    } as const;
}
export const FormSection = styled(Stack)(({ theme }) => ({
    ...noSelect,
    ...formSection(theme),
}));


export function formContainer(theme: Theme): SxProps {
    return {
        flexDirection: "column",
        gap: theme.spacing(3),
        margin: theme.spacing(2),
        // Smaller margin on mobile
        [theme.breakpoints.down("sm")]: {
            margin: theme.spacing(1),
        },
    };
}
export const FormContainer = styled(Stack)(({ theme }) => ({
    ...formContainer(theme),
}));

export const BannerImageContainer = styled(Box)(({ theme }) => ({
    margin: "auto",
    backgroundColor: theme.palette.mode === "light" ? "#c2cadd" : theme.palette.background.default,
    backgroundSize: "cover",
    backgroundPosition: "center",
    position: "relative",
    width: `min(${theme.breakpoints.values.sm}px, 100%)`,
    // Height should be 1/3 of width
    height: `min(calc(100vw / 3), ${theme.breakpoints.values.sm / 3}px)`,
}));

export const OverviewContainer = styled(Box)(({ theme }) => ({
    position: "relative",
    marginLeft: "auto",
    marginRight: "auto",
    background: theme.palette.background.paper,
    borderRadius: "0",
    width: `min(${theme.breakpoints.values.sm}px, 100%)`,
}));

export const OverviewProfileStack = styled(Stack)(({ theme }) => ({
    height: "48px",
    marginLeft: theme.spacing(2),
    marginRight: theme.spacing(2),
    alignItems: "flex-start",
    flexDirection: "row",
    // Apply auto margin to the second element to push the first one to the left
    "& > :nth-child(2)": {
        marginLeft: "auto",
    },
}));

export const OverviewProfileAvatar = styled(Avatar)(({ theme }) => ({
    width: "max(min(100px, 40vw), 75px)",
    height: "max(min(100px, 40vw), 75px)",
    top: "-100%",
    fontSize: "min(50px, 10vw)",
    marginRight: "auto",
}));

export const highlightStyle = (background: string, disabled: boolean | undefined) => ({
    background,
    pointerEvents: disabled ? "none" : "auto",
    filter: disabled ? "grayscale(1) opacity(0.5)" : "none",
    transition: "filter 0.2s ease-in-out",
    "&:hover": {
        background,
        filter: disabled ? "grayscale(1) opacity(0.5)" : "brightness(1.2)",
    },
} as const);

export const CardBox = styled(Box)(({ theme }) => ({
    ...noSelect,
    display: "block",
    boxShadow: theme.shadows[4],
    background: theme.palette.primary.light,
    color: theme.palette.secondary.contrastText,
    borderRadius: "8px",
    padding: theme.spacing(0.5),
    cursor: "pointer",
    width: "200px",
    minWidth: "200px",
    height: `calc(${theme.typography.h3.fontSize} * 2 + ${theme.spacing(1)})`,
    position: "relative",
    "&:hover": {
        filter: "brightness(120%)",
        transition: "filter 0.2s",
    },
})) as any;// TODO: Fix any - https://github.com/mui/material-ui/issues/38274
