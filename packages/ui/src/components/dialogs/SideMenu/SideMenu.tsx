import { API_CREDITS_MULTIPLIER, ActionOption, HistoryPageTabOption, LINKS, LogOutInput, ProfileUpdateInput, Session, SessionUser, SwitchCurrentAccountInput, User, endpointPostAuthLogout, endpointPostAuthSwitchCurrentAccount, endpointPutProfile, noop, profileValidation, shapeProfile } from "@local/shared";
import { Avatar, Box, Collapse, Divider, IconButton, Link, List, ListItem, ListItemIcon, ListItemText, Palette, SwipeableDrawer, Typography, styled, useTheme } from "@mui/material";
import { Stack } from "@mui/system";
import { fetchLazyWrapper } from "api/fetchWrapper";
import { FocusModeSelector } from "components/inputs/FocusModeSelector/FocusModeSelector";
import { LanguageSelector } from "components/inputs/LanguageSelector/LanguageSelector";
import { LeftHandedCheckbox } from "components/inputs/LeftHandedCheckbox/LeftHandedCheckbox";
import { TextSizeButtons } from "components/inputs/TextSizeButtons/TextSizeButtons";
import { ThemeSwitch } from "components/inputs/ThemeSwitch/ThemeSwitch";
import { ContactInfo } from "components/navigation/ContactInfo/ContactInfo";
import { SessionContext } from "contexts";
import { useFormik } from "formik";
import { useIsLeftHanded } from "hooks/subscriptions";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useSideMenu } from "hooks/useSideMenu";
import { useWindowSize } from "hooks/useWindowSize";
import { AwardIcon, BookmarkFilledIcon, CloseIcon, DisplaySettingsIcon, ExpandLessIcon, ExpandMoreIcon, HelpIcon, HistoryIcon, InfoIcon, LogOutIcon, MonthIcon, PlusIcon, PremiumIcon, RoutineActiveIcon, SettingsIcon, UserIcon } from "icons";
import React, { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { noSelect } from "styles";
import { SvgComponent } from "types";
import { getCurrentUser, guestSession } from "utils/authentication/session";
import { RIGHT_DRAWER_WIDTH } from "utils/consts";
import { extractImageUrl } from "utils/display/imageTools";
import { removeCookie } from "utils/localStorage";
import { openObject } from "utils/navigation/openObject";
import { Actions, performAction, toActionOption } from "utils/navigation/quickActions";
import { NAV_ACTION_TAGS, NavAction, getUserActions } from "utils/navigation/userActions";
import { CHAT_SIDE_MENU_ID, PubSub, SIDE_MENU_ID } from "utils/pubsub";

// Maximum accounts to sign in with. 
// Limited by cookie size (4kb)
const MAX_ACCOUNTS = 10;

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

export function SideMenu() {
    const session = useContext(SessionContext);
    const { breakpoints, palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const isLeftHanded = useIsLeftHanded();
    const navActions = useMemo<NavAction[]>(() => getUserActions({ session, exclude: [NAV_ACTION_TAGS.LogIn] }), [session]);

    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    // Handle opening and closing
    const { isOpen, close } = useSideMenu({ id: SIDE_MENU_ID, isMobile });
    // When moving between mobile/desktop, publish current state
    useEffect(() => {
        PubSub.get().publish("sideMenu", { id: SIDE_MENU_ID, isOpen });
    }, [breakpoints, isOpen]);

    // Display settings collapse
    const [isDisplaySettingsOpen, setIsDisplaySettingsOpen] = useState(false);
    const toggleDisplaySettings = useCallback(() => { setIsDisplaySettingsOpen(!isDisplaySettingsOpen); }, [isDisplaySettingsOpen]);
    const closeDisplaySettings = useCallback(() => { setIsDisplaySettingsOpen(false); }, []);

    // Additional resources collapse
    const [isAdditionalResourcesOpen, setIsAdditionalResourcesOpen] = useState(false);
    const toggleAdditionalResources = useCallback(() => { setIsAdditionalResourcesOpen(!isAdditionalResourcesOpen); }, [isAdditionalResourcesOpen]);
    const closeAdditionalResources = useCallback(() => { setIsAdditionalResourcesOpen(false); }, []);

    // Handle update. Only updates when menu closes, and account settings have changed.
    const [fetch] = useLazyFetch<ProfileUpdateInput, User>(endpointPutProfile);
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

    const handleClose = useCallback((event: React.MouseEvent<HTMLElement>) => {
        formik.handleSubmit();
        close();
        closeAdditionalResources();
        closeDisplaySettings();
    }, [close, closeAdditionalResources, closeDisplaySettings, formik]);

    const [switchCurrentAccount] = useLazyFetch<SwitchCurrentAccountInput, Session>(endpointPostAuthSwitchCurrentAccount);
    const handleUserClick = useCallback((event: React.MouseEvent<HTMLElement>, user: SessionUser) => {
        // Close menu if not persistent
        if (isMobile) handleClose(event);
        // If already selected, go to profile page
        if (userId === user.id) {
            openObject(user, setLocation);
        }
        // Otherwise, switch to selected account
        else {
            fetchLazyWrapper<SwitchCurrentAccountInput, Session>({
                fetch: switchCurrentAccount,
                inputs: { id: user.id },
                successMessage: () => ({ messageKey: "LoggedInAs", messageVariables: { name: user.name ?? user.handle ?? "" } }),
                onSuccess: (data) => {
                    PubSub.get().publish("session", data);
                    window.location.reload();
                },
            });
        }
    }, [handleClose, isMobile, userId, setLocation, switchCurrentAccount]);

    const handleAddAccount = useCallback((event: React.MouseEvent<HTMLElement>) => {
        setLocation(LINKS.Login);
        if (isMobile) handleClose(event);
    }, [handleClose, isMobile, setLocation]);

    const [logOut] = useLazyFetch<LogOutInput, Session>(endpointPostAuthLogout);
    const handleLogOut = useCallback((event: React.MouseEvent<HTMLElement>) => {
        if (isMobile) handleClose(event);
        const user = getCurrentUser(session);
        fetchLazyWrapper<LogOutInput, Session>({
            fetch: logOut,
            inputs: { id: user.id },
            successMessage: () => ({ messageKey: "LoggedOutOf", messageVariables: { name: user.name ?? user.handle ?? "" } }),
            onSuccess: (data) => {
                removeCookie("FormData"); // Clear old form data cache
                localStorage.removeItem("isLoggedIn");
                PubSub.get().publish("session", data);
                PubSub.get().publish("sideMenu", { id: SIDE_MENU_ID, isOpen: false });
                PubSub.get().publish("sideMenu", { id: CHAT_SIDE_MENU_ID, isOpen: false });
            },
            // If error, log out anyway
            onError: () => { PubSub.get().publish("session", guestSession); },
        });
        setLocation(LINKS.Home);
    }, [handleClose, isMobile, session, logOut, setLocation]);

    const handleOpen = (event: React.MouseEvent<HTMLElement>, link: string) => {
        setLocation(link);
        if (isMobile) handleClose(event);
    };

    const handleAction = (event: React.MouseEvent<HTMLElement>, action: ActionOption) => {
        if (isMobile) handleClose(event);
        performAction(action, session);
    };

    const accounts = useMemo(() => session?.users ?? [], [session?.users]);
    const profileListItems = accounts.map((account) => (
        <ListItem
            button
            key={account.id}
            onClick={(event) => handleUserClick(event, account)}
            sx={{
                background: account.id === userId ? palette.secondary.light : palette.background.default,
            }}
        >
            <Avatar
                src={extractImageUrl(account.profileImage, account.updated_at, 50)}
                sx={{
                    marginRight: 2,
                }}
            >
                <UserIcon
                    width="75%"
                    height="75%"
                />
            </Avatar>
            <ListItemText
                primary={account.name ?? account.handle}
                // Credits are calculated in cents * 1 million. 
                // We'll convert it to dollars
                secondary={`${t("Credit", { count: 2 })}: $${(Number(BigInt(account.credits ?? "0") / BigInt(API_CREDITS_MULTIPLIER)) / 100).toFixed(2)}`}
            />
        </ListItem>
    ), [accounts, handleUserClick]);

    return (
        <SizedDrawer
            anchor={isLeftHanded ? "left" : "right"}
            open={isOpen}
            onOpen={noop}
            onClose={handleClose}
            PaperProps={drawerPaperProps}
            variant={isMobile ? "temporary" : "persistent"}
        >
            {/* Menu title with close icon */}
            <Box
                onClick={handleClose}
                sx={{
                    ...noSelect,
                    cursor: "pointer",
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "space-between",
                    padding: 1,
                    background: palette.primary.dark,
                    color: palette.primary.contrastText,
                    textAlign: "center",
                    height: "64px", // Matches Navbar height
                    paddingRight: 3, // Matches navbar padding
                }}
            >
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
            </Box>
            <Box sx={{
                overflow: "auto",
                display: "grid",
                overflowX: "hidden",
            }}>
                {/* List of logged/in accounts and authentication-related actions */}
                <List id="side-menu-account-list" sx={{ paddingTop: 0, paddingBottom: 0 }}>
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
                <Stack direction="row" spacing={1} onClick={toggleDisplaySettings} sx={{
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
                    <Box
                        id="side-menu-display-settings"
                        sx={{
                            display: "flex",
                            flexDirection: "column",
                            gap: 2,
                            minWidth: "fit-content",
                            height: "fit-content",
                            padding: 1,
                        }}>
                        <ThemeSwitch updateServer sx={{ justifyContent: "flex-start" }} />
                        <TextSizeButtons />
                        <LeftHandedCheckbox sx={{ justifyContent: "flex-start" }} />
                        <LanguageSelector />
                        <FocusModeSelector />
                    </Box>
                    <Link
                        href={LINKS.SettingsDisplay}
                        sx={{ textAlign: "right" }}
                    >
                        <Typography variant="body2" sx={{ marginRight: "12px", marginBottom: 1 }}>{t("SeeAll")}</Typography>
                    </Link>
                </Collapse>
                <Divider sx={{ background: palette.background.textSecondary }} />
                {/* List of quick links */}
                <List id="side-menu-quick-links">
                    {/* Main navigation links, if not mobile */}
                    {!isMobile && navActions.map((action) => (
                        <NavListItem
                            key={action.value}
                            label={t(action.label, { count: action.numNotifications })}
                            Icon={action.Icon}
                            onClick={(event) => handleOpen(event, action.link)}
                            palette={palette}
                        />
                    ))}
                    <NavListItem label={t("Bookmark", { count: 2 })} Icon={BookmarkFilledIcon} onClick={(event) => handleOpen(event, `${LINKS.History}?type="${HistoryPageTabOption.Bookmarked}"`)} palette={palette} />
                    <NavListItem label={t("Calendar", { count: 2 })} Icon={MonthIcon} onClick={(event) => handleOpen(event, LINKS.Calendar)} palette={palette} />
                    <NavListItem label={t("View", { count: 2 })} Icon={HistoryIcon} onClick={(event) => handleOpen(event, `${LINKS.History}?type="${HistoryPageTabOption.Viewed}"`)} palette={palette} />
                    <NavListItem label={t("Run", { count: 2 })} Icon={RoutineActiveIcon} onClick={(event) => handleOpen(event, `${LINKS.History}?type="${HistoryPageTabOption.RunsActive}"`)} palette={palette} />
                    <NavListItem label={t("Award", { count: 2 })} Icon={AwardIcon} onClick={(event) => handleOpen(event, LINKS.Awards)} palette={palette} />
                    <NavListItem label={t("Pro")} Icon={PremiumIcon} onClick={(event) => handleOpen(event, LINKS.Pro)} palette={palette} />
                    <NavListItem label={t("Settings")} Icon={SettingsIcon} onClick={(event) => handleOpen(event, LINKS.Settings)} palette={palette} />
                    <NavListItem label={t("Tutorial")} Icon={HelpIcon} onClick={(event) => handleAction(event, toActionOption(Actions.tutorial))} palette={palette} />
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
