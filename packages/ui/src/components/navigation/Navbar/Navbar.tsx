import { BUSINESS_NAME, LINKS } from "@local/shared";
import { AppBar, Avatar, Box, BoxProps, Button, IconButton, Palette, Stack, Typography, styled, useTheme } from "@mui/material";
import { PopupMenu } from "components/buttons/PopupMenu/PopupMenu";
import { Title } from "components/text/Title/Title";
import { SessionContext } from "contexts";
import { useIsLeftHanded } from "hooks/subscriptions";
import { useDimensions } from "hooks/useDimensions";
import { useSideMenu } from "hooks/useSideMenu";
import { useWindowSize } from "hooks/useWindowSize";
import { ListIcon, LogInIcon, ProfileIcon, VrooliIcon } from "icons";
import { forwardRef, useCallback, useContext, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { openLink, useLocation } from "route";
import { noSelect } from "styles";
import { checkIfLoggedIn, getCurrentUser } from "utils/authentication/session";
import { extractImageUrl } from "utils/display/imageTools";
import { NAV_ACTION_TAGS, NavAction, actionsToMenu, getUserActions } from "utils/navigation/userActions";
import { CHAT_SIDE_MENU_ID, PubSub, SIDE_MENU_ID } from "utils/pubsub";
import { ContactInfo } from "../ContactInfo/ContactInfo";
import { HideOnScroll } from "../HideOnScroll/HideOnScroll";
import { NavbarProps } from "../types";

const zIndex = 300;
const MIN_APP_BAR_HEIGHT_PX = 64;

function navItemStyle(palette: Palette) {
    return {
        background: "transparent",
        color: palette.primary.contrastText,
        textTransform: "none",
        fontSize: "1.4em",
        "&:hover": {
            color: palette.secondary.light,
        },
    } as const;
}

interface OuterBoxProps extends BoxProps {
    isLeftHanded: boolean;
}

const OuterBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isLeftHanded",
})<OuterBoxProps>(({ isLeftHanded }) => ({
    display: "flex",
    height: "100%",
    paddingBottom: "0",
    paddingRight: "0 !important",
    right: "0",
    // Reverse order on left handed mode
    flexDirection: isLeftHanded ? "row-reverse" : "row",
}));

const EnterButton = styled(Button)(({ theme }) => ({
    whiteSpace: "nowrap",
    // Hide text on small screens, and remove start icon's padding
    fontSize: "1.4em",
    // eslint-disable-next-line no-magic-numbers
    borderRadius: theme.spacing(1.5),
    textTransform: "none",
    [theme.breakpoints.down("md")]: {
        fontSize: "1em",
    },
    [theme.breakpoints.down("sm")]: {
        fontSize: "0px",
        "& .MuiButton-startIcon": {
            marginLeft: "0px",
            marginRight: "0px",
        },
    },
}));

const ProfileAvatar = styled(Avatar)(({ theme }) => ({
    background: theme.palette.primary.contrastText,
    width: "40px",
    height: "40px",
    margin: "auto",
    marginRight: theme.spacing(1),
    cursor: "pointer",
}));

export function NavList() {
    const session = useContext(SessionContext);
    const user = useMemo(() => getCurrentUser(session), [session]);
    const isLeftHanded = useIsLeftHanded();
    const { t } = useTranslation();
    const { breakpoints, palette } = useTheme();
    const [, setLocation] = useLocation();

    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const navActions = useMemo<NavAction[]>(() => getUserActions({ session, exclude: [NAV_ACTION_TAGS.Home, NAV_ACTION_TAGS.LogIn] }), [session]);

    const { isOpen: isSideMenuOpen } = useSideMenu({ id: SIDE_MENU_ID, isMobile });
    const openSideMenu = useCallback(() => { PubSub.get().publish("sideMenu", { id: SIDE_MENU_ID, isOpen: true }); }, []);

    const toLoginPage = useCallback(function toLoginPageCallback(e: React.MouseEvent<HTMLElement>) {
        e.preventDefault();
        openLink(setLocation, LINKS.Login);
    }, [setLocation]);

    return (
        <OuterBox isLeftHanded={isLeftHanded}>
            {/* Contact menu */}
            {!isMobile && !getCurrentUser(session).id && !isSideMenuOpen && <PopupMenu
                text={t("Contact")}
                variant="text"
                size="large"
                sx={navItemStyle(palette)}
            >
                <ContactInfo />
            </PopupMenu>}
            {/* List items displayed when on wide screen */}
            <Box>
                {!isMobile && !isSideMenuOpen && actionsToMenu({
                    actions: navActions,
                    setLocation,
                    sx: navItemStyle(palette),
                })}
            </Box>
            {/* Enter button displayed when not logged in */}
            {!checkIfLoggedIn(session) && (
                <EnterButton
                    href={LINKS.Login}
                    onClick={toLoginPage}
                    startIcon={<LogInIcon />}
                    variant="contained"
                >
                    {t("LogIn")}
                </EnterButton>
            )}
            {checkIfLoggedIn(session) && (
                <ProfileAvatar
                    id="side-menu-profile-icon"
                    src={extractImageUrl(user.profileImage, user.updated_at, 50)}
                    onClick={openSideMenu}
                >
                    <ProfileIcon fill={palette.primary.dark} width="100%" height="100%" />
                </ProfileAvatar>
            )}
        </OuterBox>
    );
}

type TitleDisplayProps = Pick<NavbarProps, "help" | "options" | "startComponent" | "title" | "titleComponent" | "titleBehaviorDesktop" | "titleBehaviorMobile"> & {
    isLeftHanded: boolean;
    isMobile: boolean;
    location: "In" | "Below";
    onLogoClick: () => unknown;
}

const NameTypography = styled(Typography)(({ theme }) => ({
    position: "relative",
    cursor: "pointer",
    lineHeight: "1.3",
    fontSize: "2.5em",
    fontFamily: "sakbunderan",
    color: theme.palette.primary.contrastText,
}));

interface StartAndTitleBoxProps extends BoxProps {
    isLeftHanded: boolean;
}

const StartAndTitleBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isLeftHanded",
})<StartAndTitleBoxProps>(({ isLeftHanded, theme }) => ({
    padding: 0,
    paddingTop: "4px",
    display: "flex",
    alignItems: "center",
    marginRight: isLeftHanded ? theme.spacing(1) : "auto",
    marginLeft: isLeftHanded ? "auto" : theme.spacing(1),
}));


const logoIconStyle = {
    display: "flex",
    padding: 0,
    margin: "5px",
    marginLeft: "max(-5px, -5vw)",
    width: "48px",
    height: "48px",
} as const;
const pageTitleStyle = {
    stack: { padding: 0, paddingLeft: 1 },
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
    const { isOpen: isChatSideMenuOpen } = useSideMenu({ id: CHAT_SIDE_MENU_ID, isMobile });

    let TitleComponent: JSX.Element | null = null;
    let StartComponent: JSX.Element | null = null;

    // Check if title should be displayed here
    const behavior = isMobile ? titleBehaviorMobile : titleBehaviorDesktop;
    const showOnDesktop = behavior
        ? behavior === "Hide"
            ? false
            : (location === "In" && behavior === "ShowIn") || (location === "Below" && behavior === "ShowBelow") || (location === "Below" && isChatSideMenuOpen)
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
            sxs={pageTitleStyle}
            title={title}
            variant="header"
        />;
    } else if (location === "In") {
        isBusinessName = true;
        TitleComponent = <NameTypography
            variant="h6"
            noWrap
            onClick={onLogoClick}
        >{BUSINESS_NAME}</NameTypography>;
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

    if (TitleComponent && StartComponent) {
        return (
            <StartAndTitleBox
                isLeftHanded={isLeftHanded}
            >
                {StartComponent}
                {TitleComponent}
            </StartAndTitleBox>
        );
    }
    if (TitleComponent) return TitleComponent;
    if (StartComponent) return StartComponent;
    return null;
}

interface NavListBoxProps extends BoxProps {
    isLeftHanded: boolean;
}

const NavListBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isLeftHanded",
})<NavListBoxProps>(({ isLeftHanded }) => ({
    marginLeft: isLeftHanded ? 0 : "auto",
    marginRight: isLeftHanded ? "auto" : 0,
    maxHeight: "100%",
}));

const sideMenuStyle = {
    width: "48px",
    height: "48px",
    marginLeft: 1,
    marginRight: 1,
    cursor: "pointer",
} as const;

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
    const session = useContext(SessionContext);
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const isLeftHanded = useIsLeftHanded();


    const toHome = useCallback(() => setLocation(LINKS.Home), [setLocation]);
    const scrollToTop = useCallback(() => window.scrollTo({ top: 0, behavior: "smooth" }), []);

    // Set tab to title
    useEffect(() => {
        document.title = tabTitle || title ? `${tabTitle ?? title} | ${BUSINESS_NAME}` : BUSINESS_NAME;
    }, [tabTitle, title]);

    const outerBoxStyle = useMemo(function outerBoxStyleMemo() {
        return {
            paddingTop: `${Math.max(dimensions.height, MIN_APP_BAR_HEIGHT_PX)}px`,
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
            minHeight: `${MIN_APP_BAR_HEIGHT_PX}px!important`,
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

    const openSideMenu = useCallback(() => { PubSub.get().publish("sideMenu", { id: "chat-side-menu", isOpen: true }); }, []);
    const displayedStartComponent = useMemo(function displayedStartComponentMemo() {
        // If startComponent provided, display it
        if (startComponent) {
            return startComponent;
        }
        // If not, display icon to open chat menu if logged in
        if (session?.isLoggedIn) {
            return (
                <IconButton
                    aria-label="Open chat menu"
                    onClick={openSideMenu}
                    sx={sideMenuStyle}
                >
                    <ListIcon fill={palette.primary.contrastText} width="100%" height="100%" />
                </IconButton>
            );
        }
        // Otherwise, display nothing
        return null;
    }, [openSideMenu, palette.primary.contrastText, session?.isLoggedIn, startComponent]);

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
                        {displayedStartComponent}
                        <TitleDisplay {...titleInProps} />
                        <NavListBox isLeftHanded={isLeftHanded}>
                            <NavList />
                        </NavListBox>
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
