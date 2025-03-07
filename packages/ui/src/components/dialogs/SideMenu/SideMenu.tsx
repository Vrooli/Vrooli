import { API_CREDITS_MULTIPLIER, ActionOption, HistoryPageTabOption, LINKS, ProfileUpdateInput, Session, SessionUser, SwitchCurrentAccountInput, User, endpointsAuth, endpointsUser, noop, profileValidation, shapeProfile } from "@local/shared";
import { Avatar, Box, Collapse, Divider, IconButton, Link, List, ListItem, ListItemIcon, ListItemText, Palette, SwipeableDrawer, Typography, styled, useTheme } from "@mui/material";
import { Stack } from "@mui/system";
import { fetchLazyWrapper } from "api/fetchWrapper.js";
import { SocketService } from "api/socket.js";
import { FocusModeSelector } from "components/inputs/FocusModeSelector/FocusModeSelector.js";
import { LanguageSelector } from "components/inputs/LanguageSelector/LanguageSelector.js";
import { LeftHandedCheckbox } from "components/inputs/LeftHandedCheckbox/LeftHandedCheckbox.js";
import { TextSizeButtons } from "components/inputs/TextSizeButtons/TextSizeButtons.js";
import { ThemeSwitch } from "components/inputs/ThemeSwitch/ThemeSwitch.js";
import { ContactInfo } from "components/navigation/ContactInfo/ContactInfo.js";
import { useFormik } from "formik";
import { useIsLeftHanded } from "hooks/subscriptions.js";
import { useLazyFetch } from "hooks/useLazyFetch.js";
import { useSideMenu } from "hooks/useSideMenu.js";
import { useWindowSize } from "hooks/useWindowSize.js";
import { AwardIcon, BookmarkFilledIcon, CloseIcon, DisplaySettingsIcon, ExpandLessIcon, ExpandMoreIcon, HelpIcon, HistoryIcon, InfoIcon, LogOutIcon, MonthIcon, PlusIcon, PremiumIcon, RoutineActiveIcon, SettingsIcon, UserIcon } from "icons/common.js";
import React, { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { getCurrentUser, guestSession } from "utils/authentication/session.js";
import { ELEMENT_IDS, RIGHT_DRAWER_WIDTH } from "utils/consts.js";
import { extractImageUrl } from "utils/display/imageTools.js";
import { removeCookie } from "utils/localStorage.js";
import { openObject } from "utils/navigation/openObject.js";
import { Actions, performAction, toActionOption } from "utils/navigation/quickActions.js";
import { NAV_ACTION_TAGS, NavAction, getUserActions } from "utils/navigation/userActions.js";
import { CHAT_SIDE_MENU_ID, PubSub, SIDE_MENU_ID, SideMenuPayloads } from "utils/pubsub.js";
import { SessionContext } from "../../../contexts.js";
import { noSelect } from "../../../styles.js";
import { SvgComponent } from "../../../types.js";

/**
 * Maximum accounts to sign in with. 
 * Limited by cookie size (4kb)
 */
const MAX_ACCOUNTS = 10;
const AVATAR_SIZE_PX = 50;

const drawerPaperProps = { id: SIDE_MENU_ID } as const;

function NavListItem({ label, Icon, onClick, palette }: {
    label: string;
    Icon: SvgComponent;
    onClick: (event: React.MouseEvent<HTMLElement>) => unknown;
    palette: Palette;
}) {
    return (
        <ListItem button onClick={onClick}>
            <ListItemIcon>
                <Icon fill={palette.background.textPrimary} />
            </ListItemIcon>
            <ListItemText primary={label} />
        </ListItem>
    );
}

const SizedDrawer = styled(SwipeableDrawer)(() => ({
    width: RIGHT_DRAWER_WIDTH,
    flexShrink: 0,
    "& .MuiDrawer-paper": {
        width: RIGHT_DRAWER_WIDTH,
        boxSizing: "border-box",
    },
    "& > .MuiDrawer-root": {
        "& > .MuiPaper-root": {
        },
    },
}));
const SideTopBar = styled(Box)(({ theme }) => ({
    ...noSelect,
    cursor: "pointer",
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    padding: theme.spacing(1),
    background: theme.palette.primary.dark,
    color: theme.palette.primary.contrastText,
    textAlign: "center",
    height: "64px", // Matches Navbar height
    // eslint-disable-next-line no-magic-numbers
    paddingRight: theme.spacing(3), // Matches navbar padding
}));
const StyledAvatar = styled(Avatar)(({ theme }) => ({
    marginRight: theme.spacing(2),
}));
const SideMenuDisplaySettingsBox = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "column",
    gap: theme.spacing(2),
    minWidth: "fit-content",
    height: "fit-content",
    padding: theme.spacing(1),
}));

const profileListContainerStyle = {
    overflow: "auto",
    display: "grid",
    overflowX: "hidden",
} as const;
const seeAllLinkStyle = { textAlign: "right" } as const;
const seeAllLinkTextStyle = { marginRight: "12px", marginBottom: "8px" } as const;

export function SideMenu() {
    const session = useContext(SessionContext);
    const { breakpoints, palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const isLeftHanded = useIsLeftHanded();
    const navActions = useMemo<NavAction[]>(() => getUserActions({ session, exclude: [NAV_ACTION_TAGS.LogIn] }), [session]);

    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    // Display settings collapse
    const [isDisplaySettingsOpen, setIsDisplaySettingsOpen] = useState(false);
    const toggleDisplaySettings = useCallback(() => { setIsDisplaySettingsOpen(!isDisplaySettingsOpen); }, [isDisplaySettingsOpen]);
    const closeDisplaySettings = useCallback(() => { setIsDisplaySettingsOpen(false); }, []);

    // Additional resources collapse
    const [isAdditionalResourcesOpen, setIsAdditionalResourcesOpen] = useState(false);
    const toggleAdditionalResources = useCallback(() => { setIsAdditionalResourcesOpen(!isAdditionalResourcesOpen); }, [isAdditionalResourcesOpen]);
    const closeAdditionalResources = useCallback(() => { setIsAdditionalResourcesOpen(false); }, []);

    // Handle opening and closing
    const onEvent = useCallback(function onEventCallback({ data }: SideMenuPayloads["side-menu"]) {
        if (!data) return;
        if (typeof data.isAdditionalResourcesCollapsed === "boolean") {
            setIsAdditionalResourcesOpen(!data.isAdditionalResourcesCollapsed);
        }
        if (typeof data.isDisplaySettingsCollapsed === "boolean") {
            setIsDisplaySettingsOpen(!data.isDisplaySettingsCollapsed);
        }
    }, []);
    const { isOpen, close } = useSideMenu({
        id: SIDE_MENU_ID,
        isMobile,
        onEvent,
    });
    // When moving between mobile/desktop, publish current state
    useEffect(() => {
        PubSub.get().publish("sideMenu", { id: SIDE_MENU_ID, isOpen });
    }, [breakpoints, isOpen]);

    // Handle update. Only updates when menu closes, and account settings have changed.
    const [fetch] = useLazyFetch<ProfileUpdateInput, User>(endpointsUser.profileUpdate);
    const formik = useFormik({
        initialValues: {
            theme: getCurrentUser(session).theme ?? "light",
        },
        enableReinitialize: true,
        validationSchema: profileValidation.update({ env: process.env.NODE_ENV }),
        onSubmit: (values) => {
            // If not logged in, do nothing
            if (!userId) {
                return;
            }
            if (!formik.isValid) return;
            const input = shapeProfile.update({
                __typename: "User",
                id: userId,
                theme: getCurrentUser(session).theme ?? "light",
            }, {
                __typename: "User",
                id: userId,
                theme: values.theme,
            });
            // If no changes, don't update
            if (!input || Object.keys(input).length === 0) {
                formik.setSubmitting(false);
                return;
            }
            fetchLazyWrapper<ProfileUpdateInput, User>({
                fetch,
                inputs: input,
                onSuccess: () => { formik.setSubmitting(false); },
                onError: () => { formik.setSubmitting(false); },
            });
        },
    });

    const handleClose = useCallback((_event: React.MouseEvent<HTMLElement>) => {
        formik.handleSubmit();
        close();
        closeAdditionalResources();
        closeDisplaySettings();
    }, [close, closeAdditionalResources, closeDisplaySettings, formik]);

    const [switchCurrentAccount] = useLazyFetch<SwitchCurrentAccountInput, Session>(endpointsAuth.switchCurrentAccount);
    const handleUserClick = useCallback((event: React.MouseEvent<HTMLElement>, user: SessionUser) => {
        // Close menu if not persistent
        if (isMobile) handleClose(event);
        // If already selected, go to profile page
        if (userId === user.id) {
            openObject(user, setLocation);
        }
        // Otherwise, switch to selected account
        else {
            SocketService.get().disconnect();
            fetchLazyWrapper<SwitchCurrentAccountInput, Session>({
                fetch: switchCurrentAccount,
                inputs: { id: user.id },
                successMessage: () => ({ messageKey: "LoggedInAs", messageVariables: { name: user.name ?? user.handle ?? "" } }),
                onSuccess: (data) => {
                    PubSub.get().publish("session", data);
                    window.location.reload();
                },
                onCompleted: () => {
                    SocketService.get().connect();
                },
            });
        }
    }, [handleClose, isMobile, userId, setLocation, switchCurrentAccount]);

    const handleAddAccount = useCallback((event: React.MouseEvent<HTMLElement>) => {
        setLocation(LINKS.Login);
        if (isMobile) handleClose(event);
    }, [handleClose, isMobile, setLocation]);

    const [logOut] = useLazyFetch<undefined, Session>(endpointsAuth.logout);
    const handleLogOut = useCallback((event: React.MouseEvent<HTMLElement>) => {
        if (isMobile) handleClose(event);
        const user = getCurrentUser(session);
        SocketService.get().disconnect();
        fetchLazyWrapper<undefined, Session>({
            fetch: logOut,
            successMessage: () => ({ messageKey: "LoggedOutOf", messageVariables: { name: user.name ?? user.handle ?? "" } }),
            onSuccess: (data) => {
                PubSub.get().publish("session", data);
            },
            onError: () => {
                PubSub.get().publish("session", guestSession);
            },
            onCompleted: () => {
                SocketService.get().connect();
                removeCookie("FormData"); // Clear old form data cache
                localStorage.removeItem("isLoggedIn");
                PubSub.get().publish("sideMenu", { id: SIDE_MENU_ID, isOpen: false });
                PubSub.get().publish("sideMenu", { id: CHAT_SIDE_MENU_ID, isOpen: false });
            },
        });
        setLocation(LINKS.Home);
    }, [handleClose, isMobile, session, logOut, setLocation]);

    const handleOpen = useCallback(function handleOpenCallback(event: React.MouseEvent<HTMLElement>, link: string) {
        setLocation(link);
        if (isMobile) handleClose(event);
    }, [handleClose, isMobile, setLocation]);

    const handleAction = useCallback(function handleActionCallback(event: React.MouseEvent<HTMLElement>, action: ActionOption) {
        if (isMobile) handleClose(event);
        performAction(action, session);
    }, [handleClose, isMobile, session]);

    const accounts = useMemo(() => session?.users ?? [], [session?.users]);
    const profileListItems = accounts.map((account) => {
        function handleClick(event: React.MouseEvent<HTMLElement>) {
            handleUserClick(event, account);
        }

        return (
            <ListItem
                button
                key={account.id}
                onClick={handleClick}
                sx={{
                    background: account.id === userId ? palette.secondary.light : palette.background.default,
                }}
            >
                <StyledAvatar
                    src={extractImageUrl(account.profileImage, account.updated_at, AVATAR_SIZE_PX)}
                >
                    <UserIcon
                        width="75%"
                        height="75%"
                    />
                </StyledAvatar>
                <ListItemText
                    primary={account.name ?? account.handle}
                    // Credits are calculated in cents * 1 million. 
                    // We'll convert it to dollars
                    secondary={`${t("Credit", { count: 2 })}: $${(Number(BigInt(account.credits ?? "0") / BigInt(API_CREDITS_MULTIPLIER)) / 100).toFixed(2)}`}
                />
            </ListItem>
        );
    }, [accounts, handleUserClick]);

    const navItems = useMemo(function navItemsMemo() {
        return [
            { label: t("Bookmark", { count: 2 }), Icon: BookmarkFilledIcon, link: `${LINKS.History}?type="${HistoryPageTabOption.Bookmarked}"`, action: null },
            { label: t("Calendar", { count: 2 }), Icon: MonthIcon, link: LINKS.Calendar, action: null },
            { label: t("View", { count: 2 }), Icon: HistoryIcon, link: `${LINKS.History}?type="${HistoryPageTabOption.Viewed}"`, action: null },
            { label: t("Run", { count: 2 }), Icon: RoutineActiveIcon, link: `${LINKS.History}?type="${HistoryPageTabOption.RunsActive}"`, action: null },
            { label: t("Award", { count: 2 }), Icon: AwardIcon, link: LINKS.Awards, action: null },
            { label: t("Pro"), Icon: PremiumIcon, link: LINKS.Pro, action: null },
            { label: t("Settings"), Icon: SettingsIcon, link: LINKS.Settings, action: null },
            { label: t("Tutorial"), Icon: HelpIcon, action: toActionOption(Actions.tutorial) },
        ] as const;
    }, [t]);

    const handleNavItemClick = useCallback(function handleNavItemClickCallback(event: React.MouseEvent<HTMLElement>, item: typeof navItems[number]) {
        if (item.action) {
            handleAction(event, item.action);
        } else if (item.link) {
            handleOpen(event, item.link);
        }
    }, [handleAction, handleOpen]);

    return (
        <SizedDrawer
            anchor={isLeftHanded ? "left" : "right"}
            open={isOpen}
            onOpen={noop}
            onClose={handleClose}
            PaperProps={drawerPaperProps}
            variant={isMobile ? "temporary" : "persistent"}
        >
            <SideTopBar onClick={handleClose}>
                {/* Close icon */}
                <IconButton
                    aria-label="close"
                    edge="end"
                    onClick={handleClose}
                    sx={{
                        marginLeft: isLeftHanded ? "unset" : "auto",
                        marginRight: isLeftHanded ? "auto" : "unset",
                    }}
                >
                    <CloseIcon fill={palette.primary.contrastText} width="40px" height="40px" />
                </IconButton>
            </SideTopBar>
            <Box sx={profileListContainerStyle}>
                {/* List of logged/in accounts and authentication-related actions */}
                <List
                    id={ELEMENT_IDS.SideMenuAccountList}
                    sx={{ paddingTop: 0, paddingBottom: 0 }}
                >
                    {profileListItems}
                    <Divider sx={{ background: palette.background.textSecondary }} />
                    {/* Buttons to add account and log out */}
                    {accounts.length < MAX_ACCOUNTS && <ListItem button onClick={handleAddAccount}>
                        <ListItemIcon>
                            <PlusIcon fill={palette.background.textPrimary} />
                        </ListItemIcon>
                        <ListItemText primary={t("AddAccount")} />
                    </ListItem>}
                    {accounts.length > 0 && <ListItem button onClick={handleLogOut}>
                        <ListItemIcon>
                            <LogOutIcon fill={palette.background.textPrimary} />
                        </ListItemIcon>
                        <ListItemText primary={t("LogOut")} />
                    </ListItem>}
                </List>
                <Divider sx={{ background: palette.background.textSecondary }} />
                {/* Display Settings */}
                <Stack id="side-menu-display-header" direction="row" spacing={1} onClick={toggleDisplaySettings} sx={{
                    display: "flex",
                    alignItems: "center",
                    textAlign: "left",
                    paddingLeft: 2,
                    paddingRight: 2,
                    paddingTop: 1,
                    paddingBottom: 1,
                }}>
                    <Box sx={{ minWidth: "56px", display: "flex", alignItems: "center" }}>
                        <DisplaySettingsIcon fill={palette.background.textPrimary} />
                    </Box>
                    <Typography variant="body1" sx={{ color: palette.background.textPrimary, ...noSelect, margin: "0 !important" }}>{t("Display")}</Typography>
                    {isDisplaySettingsOpen ? <ExpandMoreIcon fill={palette.background.textPrimary} style={{ marginLeft: "auto" }} /> : <ExpandLessIcon fill={palette.background.textPrimary} style={{ marginLeft: "auto" }} />}
                </Stack>
                <Collapse in={isDisplaySettingsOpen} sx={{ display: "inline-block", minHeight: "auto!important" }}>
                    <SideMenuDisplaySettingsBox id={ELEMENT_IDS.SideMenuDisplaySettings}>
                        <ThemeSwitch updateServer sx={{ justifyContent: "flex-start" }} />
                        <TextSizeButtons />
                        <LeftHandedCheckbox sx={{ justifyContent: "flex-start" }} />
                        <LanguageSelector />
                        <FocusModeSelector />
                    </SideMenuDisplaySettingsBox>
                    <Link
                        href={LINKS.SettingsDisplay}
                        sx={seeAllLinkStyle}
                    >
                        <Typography variant="body2" sx={seeAllLinkTextStyle}>{t("SeeAll")}</Typography>
                    </Link>
                </Collapse>
                <Divider sx={{ background: palette.background.textSecondary }} />
                {/* List of quick links */}
                <List id={ELEMENT_IDS.SideMenuQuickLinks}>
                    {/* Main navigation links, if not mobile */}
                    {!isMobile && navActions.map((action) => {
                        function handleClick(event: React.MouseEvent<HTMLElement>) {
                            handleOpen(event, action.link);
                        }

                        return (
                            <NavListItem
                                key={action.value}
                                label={t(action.label, { count: action.numNotifications })}
                                Icon={action.Icon}
                                onClick={handleClick}
                                palette={palette}
                            />
                        );
                    })}
                    {/* Other navigation links */}
                    {navItems.map((item, index) => {
                        function handleClick(event: React.MouseEvent<HTMLElement>) {
                            handleNavItemClick(event, item);
                        }

                        return (
                            <NavListItem
                                key={index}
                                label={item.label}
                                Icon={item.Icon}
                                onClick={handleClick}
                                palette={palette}
                            />
                        );
                    })}
                </List>
                <Divider sx={{ background: palette.background.textSecondary }} />
                {/* Additional Resources */}
                <Stack direction="row" spacing={1} onClick={toggleAdditionalResources} sx={{
                    display: "flex",
                    alignItems: "center",
                    textAlign: "left",
                    paddingLeft: 2,
                    paddingRight: 2,
                    paddingTop: 1,
                    paddingBottom: 1,
                }}>
                    <Box sx={{ minWidth: "56px", display: "flex", alignItems: "center" }}>
                        <InfoIcon fill={palette.background.textPrimary} />
                    </Box>
                    <Typography variant="body1" sx={{ color: palette.background.textPrimary, ...noSelect, margin: "0 !important" }}>{t("AdditionalResources")}</Typography>
                    {isAdditionalResourcesOpen ? <ExpandMoreIcon fill={palette.background.textPrimary} style={{ marginLeft: "auto" }} /> : <ExpandLessIcon fill={palette.background.textPrimary} style={{ marginLeft: "auto" }} />}
                </Stack>
                <Collapse in={isAdditionalResourcesOpen} sx={{ display: "inline-block" }}>
                    <ContactInfo />
                </Collapse>
            </Box>
        </SizedDrawer>
    );
}
