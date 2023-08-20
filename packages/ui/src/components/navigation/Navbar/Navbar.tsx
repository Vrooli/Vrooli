import { BUSINESS_NAME, LINKS } from "@local/shared";
import { AppBar, Box, Stack, useTheme } from "@mui/material";
import { Title } from "components/text/Title/Title";
import { useDimensions } from "hooks/useDimensions";
import { useIsLeftHanded } from "hooks/useIsLeftHanded";
import { useWindowSize } from "hooks/useWindowSize";
import { forwardRef, useCallback, useEffect, useMemo } from "react";
import { useLocation } from "route";
import { noSelect } from "styles";
import { HideOnScroll } from "../HideOnScroll/HideOnScroll";
import { NavbarLogo } from "../NavbarLogo/NavbarLogo";
import { NavList } from "../NavList/NavList";
import { NavbarLogoState, NavbarProps } from "../types";

const zIndex = 300;

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
    options,
    shouldHideTitle = false,
    tabTitle,
    title,
    titleComponent,
}: NavbarProps, ref) => {
    const { breakpoints, palette } = useTheme();
    const [, setLocation] = useLocation();

    const { dimensions, ref: dimRef } = useDimensions();

    const toHome = useCallback(() => setLocation(LINKS.Home), [setLocation]);
    const scrollToTop = useCallback(() => window.scrollTo({ top: 0, behavior: "smooth" }), []);

    // Determine display texts and states
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const { logoState } = useMemo(() => {
        const logoState: NavbarLogoState = (isMobile && (title || titleComponent)) ? "icon" : "full";
        return { logoState };
    }, [isMobile, title, titleComponent]);

    const isLeftHanded = useIsLeftHanded();

    // Set tab to title
    useEffect(() => {
        document.title = tabTitle || title ? `${tabTitle ?? title} | ${BUSINESS_NAME}` : BUSINESS_NAME;
    }, [tabTitle, title]);

    const logo = useMemo(() => (
        <Box
            onClick={toHome}
            sx={{
                padding: 0,
                paddingTop: "4px",
                display: "flex",
                alignItems: "center",
                marginRight: isMobile && isLeftHanded ? 1 : "auto",
                marginLeft: isMobile && isLeftHanded ? "auto" : 1,
            }}
        >
            <NavbarLogo
                onClick={toHome}
                state={logoState}
            />
        </Box>
    ), [isLeftHanded, isMobile, logoState, toHome]);

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
            <HideOnScroll forceVisible={!isMobile}>
                <AppBar
                    onClick={scrollToTop}
                    ref={dimRef}
                    sx={{
                        ...noSelect,
                        background: palette.primary.dark,
                        minHeight: "64px!important",
                        position: "fixed", // Allows items to be displayed below the navbar
                        zIndex,
                    }}>
                    <Stack direction="row" spacing={0} alignItems="center" sx={{
                        paddingLeft: 1,
                        paddingRight: 1,
                    }}>
                        {/* Logo displayed on left for desktop and right-handed mobile users.
                        Account menu displayed otherwise */}
                        {!(isMobile && isLeftHanded) ? logo : <Box sx={{
                            marginRight: "auto",
                            maxHeight: "100%",
                        }}>
                            <NavList />
                        </Box>}
                        {/* Title displayed here on mobile */}
                        {isMobile && title && !titleComponent && <Title
                            help={help}
                            options={options}
                            title={title}
                            variant="header"
                        />}
                        {isMobile && titleComponent}
                        {(isMobile && isLeftHanded) ? logo : <Box sx={{
                            marginLeft: "auto",
                            maxHeight: "100%",
                        }}>
                            <NavList />
                        </Box>}
                    </Stack>
                    {/* "below" displayed inside AppBar on mobile */}
                    {isMobile && below}
                </AppBar>
            </HideOnScroll>
            {/* Title displayed here on desktop */}
            {!isMobile && title && !titleComponent && !shouldHideTitle && <Title
                help={help}
                options={options}
                title={title}
                variant="header"
            />}
            {!isMobile && !shouldHideTitle && titleComponent}
            {/* "below" and title displayered here on desktop */}
            {!isMobile && below}
        </Box>
    );
});
