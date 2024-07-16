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

type TitleDisplayProps = Pick<NavbarProps, "help" | "options" | "startComponent" | "title" | "titleComponent" | "titleBehaviorDesktop" | "titleBehaviorMobile"> & {
    isLeftHanded: boolean;
    isMobile: boolean;
    location: "In" | "Below";
    onLogoClick: () => unknown;
}

const logoIconStyle = {
    display: "flex",
    padding: 0,
    margin: "5px",
    marginLeft: "max(-5px, -5vw)",
    width: "48px",
    height: "48px",
} as const;

function TitleDisplay({
    help,
    isLeftHanded,
    isMobile,
    location,
    onLogoClick,
    options,
    startComponent,
    title,
    titleComponent,
    titleBehaviorDesktop,
    titleBehaviorMobile,
}: TitleDisplayProps) {
    const { palette } = useTheme();

    let TitleComponent: JSX.Element | null = null;
    let StartComponent: JSX.Element | null = null;

    // Check if title should be displayed here
    const behavior = isMobile ? titleBehaviorMobile : titleBehaviorDesktop;
    const showOnDesktop = behavior
        ? behavior === "Hide"
            ? false
            : (location === "In" && behavior === "ShowIn") || (location === "Below" && behavior === "ShowBelow")
        : location === "Below";
    const showOnMobile = behavior
        ? behavior === "Hide"
            ? false
            : location === "In" && behavior === "ShowIn" :
        location === "In";
    const showTitle = (isMobile ? showOnMobile : showOnDesktop) && (title || titleComponent);
    let isBusinessName = false;

    // Create title, as provided text/component or as business name
    if (showTitle && titleComponent) {
        TitleComponent = titleComponent;
    } else if (showTitle && title) {
        TitleComponent = <Title
            help={help}
            options={options}
            sxs={{ stack: { padding: 0, paddingLeft: 1 } }}
            title={title}
            variant="header"
        />;
    } else if (location === "In") {
        isBusinessName = true;
        TitleComponent = <Typography
            variant="h6"
            noWrap
            onClick={onLogoClick}
            sx={{
                position: "relative",
                cursor: "pointer",
                lineHeight: "1.3",
                fontSize: "2.5em",
                fontFamily: "SakBunderan",
                color: palette.primary.contrastText,
            }}
        >{BUSINESS_NAME}</Typography>;
    }

    // Create start component
    if (startComponent) {
        StartComponent = startComponent;
    } else if (location === "In" && !startComponent && (isBusinessName || !isMobile)) {
        StartComponent = <IconButton
            aria-label="Go to home page"
            onClick={onLogoClick}
            sx={logoIconStyle}>
            <VrooliIcon fill={palette.primary.contrastText} width="100%" height="100%" />
        </IconButton>;
    }

    // Render title and start component
    if (TitleComponent && StartComponent) {
        return (
            <Box
                sx={{
                    padding: 0,
                    paddingTop: "4px",
                    display: "flex",
                    alignItems: "center",
                    marginRight: isLeftHanded ? 1 : "auto",
                    marginLeft: isLeftHanded ? "auto" : 1,
                }}
            >
                {StartComponent}
                {TitleComponent}
            </Box>
        );
    }
    if (TitleComponent) return TitleComponent;
    if (StartComponent) return StartComponent;
    return null;
}

function NavListComponent({ isLeftHanded }) {
    return <Box sx={{
        marginLeft: isLeftHanded ? 0 : "auto",
        marginRight: isLeftHanded ? "auto" : 0,
        maxHeight: "100%",
    }}>
        <NavList />
    </Box>;
}

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
    startComponent,
    sxs,
    tabTitle,
    title,
    titleBehaviorDesktop,
    titleBehaviorMobile,
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

    const outerBoxStyle = useMemo(function outerBoxStyleMemo() {
        return {
            paddingTop: `${Math.max(dimensions.height, 64)}px`,
            "@media print": {
                display: "none",
            },
            ...sxs?.root,
        } as const;
    }, [dimensions.height, sxs]);

    const appBarStyle = useMemo(function appBarStyleMemo() {
        return {
            ...noSelect,
            background: palette.primary.dark,
            minHeight: "64px!important",
            position: "fixed", // Allows items to be displayed below the navbar
            justifyContent: "center",
            zIndex,
            ...sxs?.appBar,
        } as const;
    }, [palette.primary.dark, sxs?.appBar]);

    const appBarStackStyle = useMemo(function appBarStackStyleMemo() {
        return {
            paddingLeft: 1,
            paddingRight: 1,
            // TODO Reverse order on left-handed mobile
            flexDirection: isLeftHanded ? "row-reverse" : "row",
        } as const;
    }, [isLeftHanded]);

    const { titleInProps, titleBelowProps } = useMemo(function titlePropsMemo() {
        const common = {
            help,
            isLeftHanded,
            isMobile,
            title,
            titleComponent,
            onLogoClick: toHome,
            options,
            titleBehaviorDesktop,
            titleBehaviorMobile,
        } as const;
        return {
            titleInProps: { ...common, location: "In" },
            titleBelowProps: { ...common, location: "Below" },
        } as const;
    }, [help, isLeftHanded, isMobile, options, title, titleBehaviorDesktop, titleBehaviorMobile, titleComponent, toHome]);

    return (
        <Box
            id='navbar'
            ref={ref}
            sx={outerBoxStyle}>
            <HideOnScroll forceVisible={keepVisible || !isMobile}>
                <AppBar
                    onClick={scrollToTop}
                    ref={dimRef}
                    sx={appBarStyle}>
                    <Stack direction="row" spacing={0} alignItems="center" sx={appBarStackStyle}>
                        {startComponent ? <Box sx={isMobile ? {
                            marginRight: isLeftHanded ? 1 : "auto",
                            marginLeft: isLeftHanded ? "auto" : 1,
                        } : {}}>{startComponent}</Box> : null}
                        <TitleDisplay {...titleInProps} />
                        <NavListComponent {...{ isLeftHanded }} />
                    </Stack>
                    {/* "below" displayed inside AppBar on mobile */}
                    {isMobile && below}
                </AppBar>
            </HideOnScroll>
            <TitleDisplay {...titleBelowProps} />
            {/* "below" and title displayered here on desktop */}
            {!isMobile && below}
        </Box>
    );
});
Navbar.displayName = "Navbar";
