/* eslint-disable no-magic-numbers */
import { Avatar, AvatarProps, Box, BoxProps, IconButton, Palette, Stack, StackProps, styled, SxProps, Theme, Typography } from "@mui/material";

export const bottomNavHeight = "56px";
export const pagePaddingBottom = "var(--page-padding-bottom)";

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

export function multiLineEllipsis(lines: number) {
    return {
        display: "-webkit-box",
        lineHeight: "1.5", // Without this, WebkitLineClamp might accidentally include the top of the first line cut off
        WebkitLineClamp: lines,
        WebkitBoxOrient: "vertical",
        overflow: "hidden",
        textOverflow: "ellipsis",
    } as const;
}

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

export function linkColors(palette: Palette) {
    return {
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
    };
}

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

export function slideText(theme: Theme) {
    return {
        zIndex: 10,
        textAlign: "center",
        textWrap: "balance",
        [theme.breakpoints.up("md")]: {
            fontSize: "1.5rem",
        },
        [theme.breakpoints.down("md")]: {
            fontSize: "1.25rem",
        },
    } as const;
}
export const SlideText = styled("h3")(({ theme }) => ({
    ...slideText(theme),
}));

export function slideTitle(theme: Theme) {
    return {
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
    } as const;
}
export const SlideTitle = styled(Typography)(({ theme }) => ({
    ...slideTitle(theme),
}));

export function slideImageContainer(theme: Theme) {
    return {
        justifyContent: "center",
        height: "100%",
        display: "flex",
        "& > img": {
            maxWidth: "min(500px, 100%)",
            maxHeight: "100%",
            zIndex: "3",
        },
    } as const;
}
export const SlideImageContainer = styled(Box)(({ theme }) => ({
    ...slideImageContainer(theme),
}));

export function slideImage(theme: Theme) {
    return {
        borderRadius: theme.spacing(4),
        objectFit: "cover",
    } as const;
}
export const SlideImage = styled("img")(({ theme }) => ({
    ...slideImage(theme),
}));

export function slideContent(theme: Theme) {
    return {
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
    } as const;
}
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

export function slideBox(theme: Theme) {
    return {
        ...slideContent(theme),
        background: "#2c2d2fd1",
        borderRadius: theme.spacing(4),
        minHeight: "unset",
        zIndex: 2,
        gap: theme.spacing(6),
    } as const;
}
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

export function formSection(theme: Pick<Theme, "breakpoints" | "palette" | "spacing">, variant: "card" | "transparent" = "card") {
    let background: string | undefined;
    if (variant === "card") {
        background = theme.palette.mode === "dark" ?
            theme.palette.background.paper :
            theme.palette.background.default;
    }
    return {
        overflowX: "auto",
        gap: theme.spacing(2),
        padding: theme.spacing(2),
        background,
        borderRadius: theme.spacing(1),
        flexDirection: "column",
        "@media print": {
            border: `1px solid ${theme.palette.divider}`,
        },
        // Smaller padding on mobile
        [theme.breakpoints.down("sm")]: {
            padding: theme.spacing(1.5),
        },
        width: "100%",
    } as const;
}
interface FormSectionProps extends StackProps {
    variant?: "card" | "transparent";
}
export const FormSection = styled(Stack, {
    shouldForwardProp: (prop) => prop !== "variant",
})<FormSectionProps>(({ theme, variant }) => ({
    ...noSelect,
    ...formSection(theme, variant),
}));


export function formContainer(theme: Theme): SxProps {
    return {
        flexDirection: "column",
        gap: theme.spacing(3),
        width: "100%",
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
    backgroundColor: theme.palette.background.default,
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

interface ProfileAvatarProps extends AvatarProps {
    isBot: boolean;
    profileColors: [string, string];
}

export const ProfileAvatar = styled(Avatar, {
    shouldForwardProp: (prop) => prop !== "isBot" && prop !== "profileColors",
})<ProfileAvatarProps>(({ isBot, profileColors }) => ({
    backgroundColor: profileColors[0],
    color: profileColors[1],
    borderRadius: isBot ? "8px" : "50%",
}));


export const OverviewProfileAvatar = styled(ProfileAvatar)(() => ({
    width: "max(min(100px, 40vw), 75px)",
    height: "max(min(100px, 40vw), 75px)",
    top: "-100%",
    fontSize: "min(50px, 10vw)",
    marginRight: "auto",
}));

export const ProfilePictureInputAvatar = styled(ProfileAvatar)(({ theme }) => ({
    boxShadow: theme.shadows[4],
    width: "100px",
    height: "100px",
    cursor: "pointer",
}));

interface ObjectListProfileAvatarProps extends ProfileAvatarProps {
    isMobile: boolean;
}

export const ObjectListProfileAvatar = styled(ProfileAvatar, {
    shouldForwardProp: (prop) => prop !== "isMobile",
})<ObjectListProfileAvatarProps>(({ isMobile }) => ({
    width: isMobile ? "40px" : "50px",
    height: isMobile ? "40px" : "50px",
    pointerEvents: "none",
    marginTop: "auto",
    marginBottom: "auto",
}));

export function highlightStyle(background: string, disabled: boolean | undefined) {
    return {
        background,
        pointerEvents: disabled ? "none" : "auto",
        filter: disabled ? "grayscale(1) opacity(0.5)" : "none",
        transition: "filter 0.2s ease-in-out",
        "&:hover": {
            background,
            filter: disabled ? "grayscale(1) opacity(0.5)" : "brightness(1.2)",
        },
    } as const;
}

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

export const SideActionsButton = styled(IconButton)(({ theme }) => ({
    background: theme.palette.secondary.main,
    width: "54px",
    height: "54px",
    padding: 0,
    "& > *": {
        width: "36px",
        height: "36px",
    },
}));

export const ScrollBox = styled(Box)(() => ({
    height: "100%",
    overflowY: "auto",
}));

/**
 * Page container for centering content 
 * horizontally and vertically as the whole page 
 * (e.g. LoginView)
 */
export const CenteredContentPage = styled(Box)(({ theme }) => ({
    height: "100vh",
    display: "flex",
    flexDirection: "column",
    overflow: "hidden",
}));

/**
 * Used within CenteredContentPage to center content 
 * horizontally and vertically
 */
export const CenteredContentPageWrap = styled(Box)(({ theme }) => ({
    flex: 1,
    display: "flex",
    justifyContent: "center",
    alignItems: "center",
    overflow: "auto",
    padding: theme.spacing(2),
}));

interface CenteredContentPaperProps extends BoxProps {
    maxWidth: number;
}

/**
 * Used within CenteredContentPageWrap to display centered content
 */
export const CenteredContentPaper = styled(Box, {
    shouldForwardProp: (prop) => prop !== "maxWidth",
})<CenteredContentPaperProps>(({ maxWidth, theme }) => ({
    width: `min(${maxWidth}px, 100%)`,
    maxHeight: "100%",
    background: theme.palette.background.paper,
    overflow: "auto",
    borderRadius: theme.spacing(2),
    [theme.breakpoints.down("sm")]: {
        borderRadius: theme.spacing(1),
    },
}));

export function searchButtonStyle(palette: Palette) {
    return {
        minHeight: "34px",
        display: "flex",
        justifyContent: "center",
        alignItems: "center",
        borderRadius: "50px",
        border: `2px solid ${palette.secondary.main}`,
        margin: 1,
        padding: 0,
        paddingLeft: 1,
        paddingRight: 1,
        cursor: "pointer",
        "&:hover": {
            transform: "scale(1.1)",
        },
        transition: "transform 0.2s ease-in-out",
    } as const;
}
