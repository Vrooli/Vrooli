import { BUSINESS_NAME, LINKS } from "@local/shared";
import { AppBar, Badge, Box, BoxProps, Button, IconButton, Stack, Typography, styled, useTheme } from "@mui/material";
import { forwardRef, useCallback, useContext, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { SessionContext } from "../../contexts/session.js";
import { useIsLeftHanded } from "../../hooks/subscriptions.js";
import { useDimensions } from "../../hooks/useDimensions.js";
import { useMenu } from "../../hooks/useMenu.js";
import { useWindowSize } from "../../hooks/useWindowSize.js";
import { IconCommon, IconText } from "../../icons/Icons.js";
import { openLink } from "../../route/openLink.js";
import { useLocation } from "../../route/router.js";
import { ProfileAvatar, noSelect } from "../../styles.js";
import { checkIfLoggedIn, getCurrentUser } from "../../utils/authentication/session.js";
import { ELEMENT_IDS, Z_INDEX } from "../../utils/consts.js";
import { extractImageUrl } from "../../utils/display/imageTools.js";
import { NAV_ACTION_TAGS, NavAction, actionsToMenu, getUserActions } from "../../utils/navigation/userActions.js";
import { PubSub } from "../../utils/pubsub.js";
import { PopupMenu } from "../buttons/PopupMenu/PopupMenu.js";
import { Title } from "../text/Title.js";
import { ContactInfo } from "./ContactInfo.js";
import { HideOnScroll } from "./HideOnScroll.js";
import { NavbarProps } from "./types.js";

const MIN_APP_BAR_HEIGHT_PX = 64;
const AVATAR_SIZE_PX = 50;

const navItemStyle = {
    background: "transparent",
    color: "text.secondary",
    textTransform: "none",
    fontSize: "1.4em",
    "&:hover": {
        filter: "brightness(1.05)",
    },
} as const;

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

const iconButtonStyle = {
    marginRight: "8px",
} as const;

export function NavList() {
    const session = useContext(SessionContext);
    const user = useMemo(() => getCurrentUser(session), [session]);
    const isLeftHanded = useIsLeftHanded();
    const { t } = useTranslation();
    const { breakpoints } = useTheme();
    const [, setLocation] = useLocation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const navActions = useMemo<NavAction[]>(() => {
        if (session?.isLoggedIn) {
            return [];
        }
        return getUserActions({ session, exclude: [NAV_ACTION_TAGS.Home, NAV_ACTION_TAGS.LogIn] });
    }, [session]);
    const numNotifications = 3; //TODO

    const { isOpen: isUserMenuOpen } = useMenu({ id: ELEMENT_IDS.UserMenu, isMobile });
    const openUserMenu = useCallback((event: React.MouseEvent<HTMLElement>) => {
        PubSub.get().publish("menu", { id: ELEMENT_IDS.UserMenu, isOpen: true, data: { anchorEl: event.currentTarget } });
    }, []);

    const toInbox = useCallback(function toInboxCallback() {
        openLink(setLocation, LINKS.Inbox);
    }, [setLocation]);

    const toLoginPage = useCallback(function toLoginPageCallback(e: React.MouseEvent<HTMLElement>) {
        e.preventDefault();
        openLink(setLocation, LINKS.Login);
    }, [setLocation]);

    return (
        <OuterBox isLeftHanded={isLeftHanded}>
            {/* Contact menu */}
            {!isMobile && !getCurrentUser(session).id && !isUserMenuOpen && <PopupMenu
                text={t("Contact")}
                variant="text"
                size="large"
                sx={navItemStyle}
            >
                <ContactInfo />
            </PopupMenu>}
            {/* List items displayed when on wide screen */}
            <Box>
                {!isMobile && !isUserMenuOpen && actionsToMenu({
                    actions: navActions,
                    setLocation,
                    sx: navItemStyle,
                })}
            </Box>
            {/* Enter button displayed when not logged in */}
            {!checkIfLoggedIn(session) && (
                <EnterButton
                    href={LINKS.Login}
                    onClick={toLoginPage}
                    startIcon={<IconCommon
                        decorative
                        name="LogIn"
                    />}
                    variant="contained"
                >
                    {t("LogIn")}
                </EnterButton>
            )}
            {/* Inbox and profile icons if logged in and not on mobile */}
            {checkIfLoggedIn(session) && !isMobile && (
                <>
                    <IconButton
                        aria-label="Open inbox"
                        onClick={toInbox}
                        sx={iconButtonStyle}
                    >
                        <Badge badgeContent={numNotifications} color="error">
                            <IconCommon
                                decorative
                                name={numNotifications > 0 ? "NotificationsAll" : "NotificationsOff"}
                                size={32}
                            />
                        </Badge>
                    </IconButton>
                    <ProfileAvatar
                        id={ELEMENT_IDS.UserMenuProfileIcon}
                        isBot={false}
                        src={extractImageUrl(user.profileImage, user.updated_at, AVATAR_SIZE_PX)}
                        onClick={openUserMenu}
                    >
                        <IconCommon
                            decorative
                            fill="text.secondary"
                            name="Profile"
                        />
                    </ProfileAvatar>
                </>
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
    color: theme.palette.background.textSecondary,
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
    const { isOpen: isLeftDrawerOpen } = useMenu({ id: ELEMENT_IDS.LeftDrawer, isMobile });

    let TitleComponent: JSX.Element | null = null;
    let StartComponent: JSX.Element | null = null;

    // Check if title should be displayed here
    const behavior = isMobile ? titleBehaviorMobile : titleBehaviorDesktop;
    const showOnDesktop = behavior
        ? behavior === "Hide"
            ? false
            : (location === "In" && behavior === "ShowIn") || (location === "Below" && behavior === "ShowBelow") || (location === "Below" && isLeftDrawerOpen)
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
            <IconCommon
                decorative
                fill={"text.secondary"}
                name="Vrooli"
                size={48}
            />
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

const appBarStyle = {
    ...noSelect,
    background: "transparent",
    boxShadow: "none",
    minHeight: `${MIN_APP_BAR_HEIGHT_PX}px!important`,
    position: "fixed", // Allows items to be displayed below the navbar
    justifyContent: "center",
    zIndex: Z_INDEX.TopBar,
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
    const { breakpoints } = useTheme();
    const [, setLocation] = useLocation();
    const { dimensions, ref: dimRef } = useDimensions();
    const session = useContext(SessionContext);
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const isLeftHanded = useIsLeftHanded();


    const toHome = useCallback(() => setLocation(LINKS.Home), [setLocation]);
    const scrollToTop = useCallback(() => window.scrollTo({ top: 0, behavior: "smooth" }), []);

    useEffect(function setTabTitleEffect() {
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

    const appBarStackStyle = useMemo(function appBarStackStyleMemo() {
        return {
            paddingLeft: 1,
            paddingRight: 1,
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

    const openSiteNavigatorMenu = useCallback(() => { PubSub.get().publish("menu", { id: ELEMENT_IDS.LeftDrawer, isOpen: true }); }, []);
    const displayedStartComponent = useMemo(function displayedStartComponentMemo() {
        // If startComponent provided, display it
        if (startComponent) {
            return startComponent;
        }
        // If not, display icon to open chat menu if logged in
        if (session?.isLoggedIn) {
            return (
                <IconButton
                    aria-label="Open site navigator menu"
                    id={ELEMENT_IDS.SiteNavigatorMenuIcon}
                    onClick={openSiteNavigatorMenu}
                >
                    <IconText
                        decorative
                        fill="text.secondary"
                        name="List"
                    />
                </IconButton>
            );
        }
        // Otherwise, display nothing
        return null;
    }, [openSiteNavigatorMenu, session?.isLoggedIn, startComponent]);

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
