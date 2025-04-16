/* eslint-disable no-magic-numbers */
import { Avatar, AvatarProps, Box, BoxProps, IconButton, Palette, Stack, StackProps, styled, Theme } from "@mui/material";
import { forwardRef } from "react";
import { ELEMENT_CLASSES } from "./utils/consts.js";

/**
 * Lighthouse recommended size for clickable elements, to improve SEO
 */
export const clickSize = {
    minHeight: "48px",
} as const;

export const bottomNavHeight = clickSize.minHeight;
export const pagePaddingBottom = "var(--page-padding-bottom)";

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
export const SlideIconButton = styled(IconButton)(() => ({
    ...slideIconButton,
}));

export const SlideImageContainer = styled(Box)(() => ({
    position: "relative",
    justifyContent: "center",
    height: "100%",
    display: "flex",
    "& > img": {
        maxWidth: "min(500px, 100%)",
        maxHeight: "100%",
        zIndex: "3",
    },
}));

export const SlideImage = styled("img")(({ theme }) => ({
    borderRadius: theme.spacing(4),
    objectFit: "cover",
}));

export function formSection(theme: Pick<Theme, "breakpoints" | "palette" | "spacing">, variant: "card" | "transparent" = "card") {
    return {
        overflowX: "auto",
        gap: theme.spacing(2),
        padding: theme.spacing(2),
        background: variant === "card" ? theme.palette.background.paper : "transparent",
        border: variant === "card" ? "none" : `1px solid ${theme.palette.divider}`,
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

export const FormContainer = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "column",
    gap: theme.spacing(3),
    width: "100%",
    margin: theme.spacing(2),
    // Smaller margin on mobile
    [theme.breakpoints.down("sm")]: {
        margin: theme.spacing(1),
    },
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
    profileColors?: [string, string];
}

export const ProfileAvatar = styled(Avatar, {
    shouldForwardProp: (prop) => prop !== "isBot" && prop !== "profileColors",
})<ProfileAvatarProps>(({ isBot, profileColors, theme }) => ({
    backgroundColor: profileColors?.[0] ?? theme.palette.background.paper,
    color: profileColors?.[1] ?? theme.palette.background.textSecondary,
    cursor: "pointer",
    borderRadius: isBot ? "8px" : "50%",
    "& svg": {
        width: "32px",
        height: "32px",
    },
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
    width: "125px",
    height: "125px",
    cursor: "pointer",
    "& svg": {
        width: "100px",
        height: "100px",
    },
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

const scrollBoxStyle = {
    height: "100%",
    width: "100%",
    overflowY: "auto",
    position: "relative",
} as const;
export const ScrollBox = forwardRef<HTMLDivElement, BoxProps>((props, ref) => {
    return <Box
        className={ELEMENT_CLASSES.ScrollBox}
        sx={scrollBoxStyle}
        {...props}
        ref={ref}
    />;
});
ScrollBox.displayName = "ScrollBox";

/**
 * Page container for centering content 
 * horizontally and vertically as the whole page 
 * (e.g. LoginView)
 */
export const CenteredContentPage = styled(Box)(() => ({
    height: "100vh",
    display: "flex",
    flexDirection: "column",
    overflow: "hidden",
}));
