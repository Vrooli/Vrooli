import { BUSINESS_NAME, LINKS } from "@local/shared";
import { AppBar, Box, IconButton, Stack, Typography, useTheme } from "@mui/material";
import { Title } from "components/text/Title/Title";
import { useDimensions } from "hooks/useDimensions";
import { useIsLeftHanded } from "hooks/useIsLeftHanded";
import { useWindowSize } from "hooks/useWindowSize";
import { VrooliIcon } from "icons";
import { forwardRef, useCallback, useEffect, useMemo } from "react";
import { useLocation } from "route";
import { noSelect } from "styles";
import { HideOnScroll } from "../HideOnScroll/HideOnScroll";
import { NavList } from "../NavList/NavList";
import { NavbarProps } from "../types";

const zIndex = 300;

const LogoComponent = ({
    isLeftHanded,
    onClick,
    state,
}: {
    isLeftHanded: boolean;
    onClick: () => unknown;
    state: "full" | "icon" | "none";
}) => {
    const { palette } = useTheme();
    // Logo isn't always shown
    if (state === "none") return null;
    if (state === "icon") return (
        <IconButton
            aria-label="Go to home page"
            onClick={onClick}
            sx={{
                display: "flex",
                padding: 0,
                margin: "5px",
                marginLeft: "max(-5px, -5vw)",
                width: "48px",
                height: "48px",
            }}>
            <VrooliIcon fill={palette.primary.contrastText} width="100%" height="100%" />
        </IconButton>
    );
    return (
        <Box
            onClick={onClick}
            sx={{
                padding: 0,
                paddingTop: "4px",
                display: "flex",
                alignItems: "center",
                marginRight: isLeftHanded ? 1 : "auto",
                marginLeft: isLeftHanded ? "auto" : 1,
            }}
        >
            <Box
                onClick={onClick}
                sx={{
                    padding: 0,
                    display: "flex",
                    alignItems: "center",
                }}
            >
                {/* Logo */}
                <IconButton
                    aria-label="Go to home page"
                    sx={{
                        display: "flex",
                        padding: 0,
                        margin: "5px",
                        marginLeft: "max(-5px, -5vw)",
                        width: "48px",
                        height: "48px",
                    }}>
                    <VrooliIcon fill={palette.primary.contrastText} width="100%" height="100%" />
                </IconButton>
                {/* Business name */}
                {state === "full" && <Typography
                    variant="h6"
                    noWrap
                    sx={{
                        position: "relative",
                        cursor: "pointer",
                        lineHeight: "1.3",
                        fontSize: "2.5em",
                        fontFamily: "SakBunderan",
                        color: palette.primary.contrastText,
                    }}
                >{BUSINESS_NAME}</Typography>}
                {/* Alpha indicator */}
                <Typography
                    variant="body2"
                    noWrap
                    sx={{
                        color: palette.error.main,
                        paddingLeft: state === "full" ? 1 : 0,
                    }}
                >Alpha</Typography>

            </Box>
        </Box>
    );
};

const TitleDisplay = ({ isMobile, title, titleComponent, help, options, shouldHideTitle, showOnMobile }) => {
    // Check if title should be displayed here, based on screen size
    if ((isMobile && !showOnMobile) || (!isMobile && showOnMobile)) return null;
    // Desktop title can be hidden
    if (!isMobile && shouldHideTitle) return null;
    // If no custom title component, use Title component
    if (title && !titleComponent) return <Title
        help={help}
        options={options}
        title={title}
        variant="header"
    />;
    // Otherwise, use custom title component
    if (titleComponent) return titleComponent;
    return null;
};

const NavListComponent = ({ isLeftHanded }) => {
    return <Box sx={{
        marginLeft: isLeftHanded ? 0 : "auto",
        marginRight: isLeftHanded ? "auto" : 0,
        maxHeight: "100%",
    }}>
        <NavList />
    </Box>;
};

/**
 * Navbar displayed at the top of the page. Has a few different 
 * looks depending on data passed to it.
 * 
 * If the screen is large, the navbar is always displayed the same. In 
 * this case, the title and other content are displayed below the navbar.
 * 
 * Otherwise, the default look is logo & business name on the left, and 
 * account menu profile icon on the right.
 * 
 * If title data is passed in, the business name is hidden. The 
 * title is displayed in the middle, with a help icon if specified.
 * 
 * Content to display below the title (but still in the navbar) can also 
 * be passed in. This is useful for displaying a search bar, page tabs, etc. This 
 * content is inside the navbar on small screens, and below the navbar on large screens.
 */
export const Navbar = forwardRef(({
    below,
    help,
    keepVisible,
    options,
    shouldHideTitle = false,
    startComponent,
    tabTitle,
    title,
    titleComponent,
}: NavbarProps, ref) => {
    const { breakpoints, palette } = useTheme();
    const [, setLocation] = useLocation();
    const { dimensions, ref: dimRef } = useDimensions();

    // Determine display texts and states
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const logoState = useMemo(() => {
        if (isMobile && startComponent) return (title || titleComponent) ? "none" : "icon";
        if (isMobile && (title || titleComponent)) return "none";
        return "full";
    }, [isMobile, startComponent, title, titleComponent]);
    const isLeftHanded = useIsLeftHanded();


    const toHome = useCallback(() => setLocation(LINKS.Home), [setLocation]);
    const scrollToTop = useCallback(() => window.scrollTo({ top: 0, behavior: "smooth" }), []);

    // Set tab to title
    useEffect(() => {
        document.title = tabTitle || title ? `${tabTitle ?? title} | ${BUSINESS_NAME}` : BUSINESS_NAME;
    }, [tabTitle, title]);

    return (
        <Box
            id='navbar'
            ref={ref}
            sx={{
                paddingTop: `${Math.max(dimensions.height, 64)}px`,
                "@media print": {
                    display: "none",
                },
            }}>
            <HideOnScroll forceVisible={keepVisible || !isMobile}>
                <AppBar
                    onClick={scrollToTop}
                    ref={dimRef}
                    sx={{
                        ...noSelect,
                        background: palette.primary.dark,
                        minHeight: "64px!important",
                        position: "fixed", // Allows items to be displayed below the navbar
                        justifyContent: "center",
                        zIndex,
                    }}>
                    <Stack direction="row" spacing={0} alignItems="center" sx={{
                        paddingLeft: 1,
                        paddingRight: 1,
                        // TODO Reverse order on left-handed mobile
                        flexDirection: isLeftHanded ? "row-reverse" : "row",
                    }}>
                        {startComponent ? <Box sx={isMobile ? {
                            marginRight: isLeftHanded ? 1 : "auto",
                            marginLeft: isLeftHanded ? "auto" : 1,
                        } : {}}>{startComponent}</Box> : null}
                        {/* Logo */}
                        <LogoComponent {...{ isLeftHanded, isMobile, state: logoState, onClick: toHome }} />
                        {/* Title displayed here on mobile */}
                        <TitleDisplay {...{ isMobile, title, titleComponent, help, options, shouldHideTitle, showOnMobile: true }} />
                        <NavListComponent {...{ isLeftHanded }} />
                    </Stack>
                    {/* "below" displayed inside AppBar on mobile */}
                    {isMobile && below}
                </AppBar>
            </HideOnScroll>
            {/* Title displayed here on desktop */}
            <TitleDisplay {...{ isMobile, title, titleComponent, help, options, shouldHideTitle, showOnMobile: false }} />
            {/* "below" and title displayered here on desktop */}
            {!isMobile && below}
        </Box>
    );
});
