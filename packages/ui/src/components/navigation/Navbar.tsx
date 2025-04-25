import { BUSINESS_NAME, LINKS } from "@local/shared";
import { AppBar, Badge, Box, BoxProps, Button, IconButton, Slide, Typography, styled, useScrollTrigger, useTheme } from "@mui/material";
import { useCallback, useContext, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { SessionContext } from "../../contexts/session.js";
import { useIsLeftHanded } from "../../hooks/subscriptions.js";
import { useDimensions } from "../../hooks/useDimensions.js";
import { useMenu } from "../../hooks/useMenu.js";
import { useWindowSize } from "../../hooks/useWindowSize.js";
import { IconCommon, IconText } from "../../icons/Icons.js";
import { openLink } from "../../route/openLink.js";
import { useLocation } from "../../route/router.js";
import { useNotifications, useNotificationsStore } from "../../stores/notificationsStore.js";
import { ProfileAvatar, noSelect } from "../../styles.js";
import { checkIfLoggedIn, getCurrentUser } from "../../utils/authentication/session.js";
import { ELEMENT_CLASSES, ELEMENT_IDS, Z_INDEX } from "../../utils/consts.js";
import { extractImageUrl } from "../../utils/display/imageTools.js";
import { placeholderColor } from "../../utils/display/listTools.js";
import { NAV_ACTION_TAGS, getUserActions } from "../../utils/navigation/userActions.js";
import { PubSub } from "../../utils/pubsub.js";
import { PopupMenu } from "../buttons/PopupMenu.js";
import { Title } from "../text/Title.js";
import { ContactInfo } from "./ContactInfo.js";
import { NavbarProps } from "./types.js";

export const APP_BAR_HEIGHT_PX = 64;
const AVATAR_SIZE_PX = 50;
const HIDE_THRESHOLD_PX = 100;

export interface HideOnScrollProps {
    children: JSX.Element;
    forceVisible?: boolean;
}

export function HideOnScroll({
    children,
    forceVisible = false,
}: HideOnScrollProps) {
    const triggerElement = document.getElementsByClassName(ELEMENT_CLASSES.ScrollBox)[0];

    // First try to use the provided ref, then try to get element by ID, then use default window behavior
    const trigger = useScrollTrigger({
        threshold: HIDE_THRESHOLD_PX,
        target: triggerElement || undefined,
    });

    const shouldDisplay = forceVisible ? true : !trigger;

    return (
        <Slide appear={false} direction="down" in={shouldDisplay}>
            {children}
        </Slide>
    );
}

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
const navItemStyle = {
    background: "transparent",
    textTransform: "none",
    "&:hover": {
        filter: "brightness(1.05)",
    },
};

export function NavListLoggedOutButtons() {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const { breakpoints } = useTheme();
    const [, setLocation] = useLocation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const { isOpen: isSiteNavigatorOpen } = useMenu({ id: ELEMENT_IDS.LeftDrawer });

    // Don't display if:
    // 1. User is logged in
    // 2. On mobile (mobile uses BottomNav)
    // 3. SiteNavigator drawer is open
    if (checkIfLoggedIn(session) || isMobile || isSiteNavigatorOpen) {
        return null;
    }
    const navActions = getUserActions({ session, exclude: [NAV_ACTION_TAGS.Home, NAV_ACTION_TAGS.LogIn] });
    return (
        <>
            <PopupMenu
                text={t("Contact")}
                variant="text"
                size="medium"
                sx={navItemStyle}
            >
                <ContactInfo />
            </PopupMenu>
            {navActions.map(({ label, value, link }) => {
                function handleClick(event: React.MouseEvent) {
                    event.preventDefault();
                    openLink(setLocation, link);
                }

                return (
                    <Button
                        key={value}
                        variant="text"
                        size="medium"
                        href={link}
                        onClick={handleClick}
                        sx={navItemStyle}
                    >
                        {t(label, { count: 2 })}
                    </Button>
                );
            })}
        </>
    );
}

/**
 * Button to create a new chat
 */
export function NavListNewChatButton({
    handleNewChat,
}: {
    handleNewChat: () => unknown;
}) {
    const { t } = useTranslation();

    return (
        <IconButton
            aria-label={t("NewChat")}
            onClick={handleNewChat}
            title={t("NewChat")}
        >
            <IconCommon
                decorative
                name="ChatNew"
                size={32}
            />
        </IconButton>
    );
}

export function NavListInboxButton() {
    const session = useContext(SessionContext);
    const { breakpoints } = useTheme();
    const [, setLocation] = useLocation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    useNotifications();
    const notifications = useNotificationsStore(state => state.notifications);
    const numNotifications = useMemo(() => notifications.filter(({ isRead }) => !isRead).length, [notifications]);

    const toInbox = useCallback(function toInboxCallback() {
        openLink(setLocation, LINKS.Inbox);
    }, [setLocation]);

    if (!checkIfLoggedIn(session) || isMobile) {
        return null;
    }
    return (
        <IconButton
            aria-label="Open inbox"
            onClick={toInbox}
        >
            <Badge badgeContent={numNotifications} color="error">
                <IconCommon
                    decorative
                    name={numNotifications > 0 ? "NotificationsAll" : "NotificationsOff"}
                    size={32}
                />
            </Badge>
        </IconButton>
    );
}

export function NavListProfileButton() {
    const session = useContext(SessionContext);
    const user = useMemo(() => getCurrentUser(session), [session]);
    const { t } = useTranslation();
    const { breakpoints } = useTheme();
    const [, setLocation] = useLocation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    const openUserMenu = useCallback((event: React.MouseEvent<HTMLElement>) => {
        PubSub.get().publish("menu", { id: ELEMENT_IDS.UserMenu, isOpen: true, data: { anchorEl: event.currentTarget } });
    }, []);

    const toSignUpPage = useCallback(function toSignUpPageCallback(e: React.MouseEvent<HTMLElement>) {
        e.preventDefault();
        openLink(setLocation, LINKS.Signup);
    }, [setLocation]);

    if (checkIfLoggedIn(session) && !isMobile) {
        return (
            <ProfileAvatar
                id={ELEMENT_IDS.UserMenuProfileIcon}
                isBot={false}
                src={extractImageUrl(user.profileImage, user.updated_at, AVATAR_SIZE_PX)}
                onClick={openUserMenu}
                profileColors={placeholderColor(user.id)}
            >
                <IconCommon
                    decorative
                    fill="text.secondary"
                    name="Profile"
                />
            </ProfileAvatar>
        );
    }
    if (!checkIfLoggedIn(session)) {
        return (
            <EnterButton
                href={LINKS.Login}
                onClick={toSignUpPage}
                startIcon={<IconCommon
                    decorative
                    name="LogIn"
                />}
                variant="contained"
            >
                {t("LogIn")}
            </EnterButton>
        );
    }
    return null;
}

export function NavListDefault() {
    const isLeftHanded = useIsLeftHanded();

    return (
        <NavListBox isLeftHanded={isLeftHanded}>
            <NavListLoggedOutButtons />
            <NavListInboxButton />
            <NavListProfileButton />
        </NavListBox>
    );
}

const NameTypography = styled(Typography)(() => ({
    position: "relative",
    cursor: "pointer",
    lineHeight: "1.3",
    fontSize: "2.5em",
    fontFamily: "sakbunderan",
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

type TitleDisplayProps = Pick<NavbarProps, "help" | "options" | "title" | "titleBehavior">;

export function TitleDisplay({
    help,
    options,
    title,
    titleBehavior,
}: TitleDisplayProps) {
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const [, setLocation] = useLocation();
    const isLeftHanded = useIsLeftHanded();
    const { isOpen: isSiteNavigatorOpen } = useMenu({ id: ELEMENT_IDS.LeftDrawer });

    function handleLogoClick() {
        setLocation(LINKS.Home);
    }

    let TitleComponent: JSX.Element | null = null;
    if (titleBehavior !== "Hide") {
        if (title) {
            TitleComponent = <Title
                help={help}
                options={options}
                sxs={pageTitleStyle}
                title={title}
                variant="header"
            />;
        } else {
            TitleComponent = <NameTypography
                variant="h6"
                noWrap
                onClick={handleLogoClick}
            >{BUSINESS_NAME}</NameTypography>;
        }
    }

    let LogoComponent: JSX.Element | null = null;
    if (!isMobile && !isSiteNavigatorOpen) {
        LogoComponent = <IconButton
            aria-label="Go to home page"
            onClick={handleLogoClick}
            sx={logoIconStyle}>
            <IconCommon
                decorative
                name="Vrooli"
                size={48}
            />
        </IconButton>;
    }

    if (LogoComponent) {
        return (
            <StartAndTitleBox isLeftHanded={isLeftHanded}>
                {LogoComponent}
                {TitleComponent}
            </StartAndTitleBox>
        );
    }
    return TitleComponent;
}

interface NavListBoxProps extends BoxProps {
    isLeftHanded: boolean;
}
export const NavListBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isLeftHanded",
})<NavListBoxProps>(({ isLeftHanded, theme }) => ({
    alignItems: "center",
    display: "flex",
    flexDirection: isLeftHanded ? "row-reverse" : "row",
    marginLeft: isLeftHanded ? 0 : "auto",
    marginRight: isLeftHanded ? "auto" : 0,
    maxHeight: "100%",
    "& .MuiIconButton-root": {
        marginRight: theme.spacing(1),
    },
}));


type StyledAppBarProps = { textColor?: string };
const StyledAppBar = styled(AppBar, {
    shouldForwardProp: (prop) => prop !== "textColor",
})<StyledAppBarProps>(({ textColor, theme }) => ({
    ...noSelect,
    background: "transparent",
    boxShadow: "none",
    height: `${APP_BAR_HEIGHT_PX}px`,
    justifyContent: "center",
    position: "sticky",
    top: 0,
    zIndex: Z_INDEX.TopBar,
    "@media print": {
        display: "none",
    },
    // Svgs inside icon buttons
    "& .MuiIconButton-root svg": {
        fill: textColor || theme.palette.background.textSecondary,
    },
    // Text buttons
    "& .MuiButton-text": {
        color: textColor || theme.palette.background.textSecondary,
    },
}));

export interface NavbarInnerProps {
    color?: string | undefined;
    children?: React.ReactNode;
    keepVisible?: boolean;
    onClick?: () => void;
}

export function NavbarInner({
    color,
    children,
    keepVisible,
    onClick,
}: NavbarInnerProps) {
    const isLeftHanded = useIsLeftHanded();
    const { ref: dimRef } = useDimensions();
    const scrollToTop = useCallback(() => window.scrollTo({ top: 0, behavior: "smooth" }), []);

    return (
        <HideOnScroll forceVisible={keepVisible}>
            <StyledAppBar
                onClick={onClick || scrollToTop}
                ref={dimRef}
                textColor={color}
            >
                <Box
                    alignItems="center"
                    display="flex"
                    flexDirection={isLeftHanded ? "row-reverse" : "row"}
                    paddingLeft={1}
                    paddingRight={1}
                >
                    {children}
                </Box>
            </StyledAppBar>
        </HideOnScroll>
    );
}

export function SiteNavigatorButton() {
    const { isOpen: isSiteNavigatorOpen } = useMenu({ id: ELEMENT_IDS.LeftDrawer });

    const openSiteNavigatorMenu = useCallback(() => {
        PubSub.get().publish("menu", { id: ELEMENT_IDS.LeftDrawer, isOpen: true });
    }, []);

    if (isSiteNavigatorOpen) {
        return null;
    }
    return (
        <IconButton
            aria-label="Open site navigator menu"
            id={ELEMENT_IDS.SiteNavigatorMenuIcon}
            onClick={openSiteNavigatorMenu}
        >
            <IconText name="List" />
        </IconButton>
    );
}

export function useTabTitle(tabTitle: string | undefined, title: string | undefined) {
    useEffect(function setTabTitleEffect() {
        document.title = tabTitle || title ? `${tabTitle ?? title} | ${BUSINESS_NAME}` : BUSINESS_NAME;
    }, [tabTitle, title]);
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
export function Navbar({
    below,
    color,
    help,
    keepVisible,
    options,
    tabTitle,
    title,
    titleBehavior,
}: NavbarProps) {
    useTabTitle(tabTitle, title);

    return (
        <>
            <NavbarInner
                color={color}
                keepVisible={keepVisible}
            >
                <SiteNavigatorButton />
                <TitleDisplay
                    help={help}
                    title={title}
                    options={options}
                    titleBehavior={titleBehavior}
                />
                <NavListDefault />
                {below}
            </NavbarInner>
        </>
    );
}
