import { endpointPostAuthLogout, endpointPostAuthSwitchCurrentAccount, endpointPutProfile, LINKS, LogOutInput, ProfileUpdateInput, Session, SessionUser, SwitchCurrentAccountInput, User, userValidation } from "@local/shared";
import { Avatar, Box, Collapse, Divider, IconButton, List, ListItem, ListItemIcon, ListItemText, Palette, SwipeableDrawer, Typography, useTheme } from "@mui/material";
import { Stack } from "@mui/system";
import { fetchLazyWrapper } from "api";
import { FocusModeSelector } from "components/inputs/FocusModeSelector/FocusModeSelector";
import { LanguageSelector } from "components/inputs/LanguageSelector/LanguageSelector";
import { LeftHandedCheckbox } from "components/inputs/LeftHandedCheckbox/LeftHandedCheckbox";
import { TextSizeButtons } from "components/inputs/TextSizeButtons/TextSizeButtons";
import { ThemeSwitch } from "components/inputs/ThemeSwitch/ThemeSwitch";
import { ContactInfo } from "components/navigation/ContactInfo/ContactInfo";
import { SessionContext } from "contexts/SessionContext";
import { useFormik } from "formik";
import { useIsLeftHanded } from "hooks/useIsLeftHanded";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useSideMenu } from "hooks/useSideMenu";
import { useWindowSize } from "hooks/useWindowSize";
import { AwardIcon, BookmarkFilledIcon, CloseIcon, DisplaySettingsIcon, ExpandLessIcon, ExpandMoreIcon, HelpIcon, HistoryIcon, LogOutIcon, PlusIcon, PremiumIcon, RoutineActiveIcon, SettingsIcon, UserIcon } from "icons";
import React, { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { noSelect } from "styles";
import { SvgComponent } from "types";
import { getCurrentUser, guestSession } from "utils/authentication/session";
import { extractImageUrl } from "utils/display/imageTools";
import { getUserActions, NavAction, NAV_ACTION_TAGS } from "utils/navigation/userActions";
import { PubSub } from "utils/pubsub";
import { HistoryPageTabOption } from "utils/search/objectToSearch";
import { shapeProfile } from "utils/shape/models/profile";

export const sideMenuDisplayData = {
    persistentOnDesktop: true,
    sideForRightHanded: "right",
} as const;

// Maximum accounts to sign in with. 
// Limited by cookie size (4kb)
const MAX_ACCOUNTS = 10;

const zIndex = 1300;
const id = "side-menu";

const NavListItem = ({ label, Icon, onClick, palette }: {
    label: string;
    Icon: SvgComponent;
    onClick: (event: React.MouseEvent<HTMLElement>) => unknown;
    palette: Palette;
}) => (
    <ListItem button onClick={onClick}>
        <ListItemIcon>
            <Icon fill={palette.background.textPrimary} />
        </ListItemIcon>
        <ListItemText primary={label} />
    </ListItem>
);

export const SideMenu = () => {
    const session = useContext(SessionContext);
    const { breakpoints, palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const isLeftHanded = useIsLeftHanded();
    const navActions = useMemo<NavAction[]>(() => getUserActions({ session, exclude: [NAV_ACTION_TAGS.LogIn] }), [session]);

    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    // Handle opening and closing
    const { isOpen, close } = useSideMenu(id, isMobile);
    // When moving between mobile/desktop, publish current state
    useEffect(() => {
        PubSub.get().publishSideMenu({ id, isOpen });
    }, [breakpoints, isOpen]);
    console.log("is side menu open", isOpen, isMobile, isLeftHanded);

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
        validationSchema: userValidation.update({}),
        onSubmit: (values) => {
            // If not logged in, do nothing
            if (!userId) {
                return;
            }
            if (!formik.isValid) return;
            const input = shapeProfile.update({
                id: userId,
                theme: getCurrentUser(session).theme ?? "light",
            }, {
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
            setLocation(LINKS.Profile);
        }
        // Otherwise, switch to selected account
        else {
            fetchLazyWrapper<SwitchCurrentAccountInput, Session>({
                fetch: switchCurrentAccount,
                inputs: { id: user.id },
                successMessage: () => ({ messageKey: "LoggedInAs", messageVariables: { name: user.name ?? user.handle ?? "" } }),
                onSuccess: (data) => { PubSub.get().publishSession(data); },
            });
        }
    }, [handleClose, isMobile, userId, setLocation, switchCurrentAccount]);

    const handleAddAccount = useCallback((event: React.MouseEvent<HTMLElement>) => {
        setLocation(LINKS.Start);
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
            onSuccess: (data) => { PubSub.get().publishSession(data); },
            // If error, log out anyway
            onError: () => { PubSub.get().publishSession(guestSession); },
        });
        setLocation(LINKS.Home);
    }, [handleClose, isMobile, session, logOut, setLocation]);

    const handleOpen = (event: React.MouseEvent<HTMLElement>, link: string) => {
        setLocation(link);
        if (isMobile) handleClose(event);
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
            <ListItemText primary={account.name ?? account.handle} />
        </ListItem>
    ), [accounts, handleUserClick]);

    return (
        <SwipeableDrawer
            anchor={isLeftHanded ? "left" : "right"}
            open={isOpen}
            // eslint-disable-next-line @typescript-eslint/no-empty-function
            onOpen={() => { }}
            onClose={handleClose}
            PaperProps={{ id }}
            variant={isMobile ? "temporary" : "persistent"}
            sx={{
                zIndex,
                "& .MuiDrawer-paper": {
                    background: palette.background.default,
                    overflowY: "auto",
                    borderLeft: palette.mode === "light" ? "none" : `1px solid ${palette.divider}`,
                },
                "& > .MuiDialog-container": {
                    "& > .MuiPaper-root": {
                        zIndex,
                    },
                },
            }}
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
                    <Stack id="side-menu-display-settings" direction="column" spacing={2} sx={{
                        minWidth: "fit-content",
                        height: "fit-content",
                        padding: 1,
                    }}>
                        <ThemeSwitch />
                        <TextSizeButtons />
                        <LeftHandedCheckbox />
                        <LanguageSelector />
                        <FocusModeSelector />
                    </Stack>
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
                    <NavListItem label={t("Settings")} Icon={SettingsIcon} onClick={(event) => handleOpen(event, LINKS.Settings)} palette={palette} />
                    <NavListItem label={t("Bookmark", { count: 2 })} Icon={BookmarkFilledIcon} onClick={(event) => handleOpen(event, `${LINKS.History}?type=${HistoryPageTabOption.Bookmarked}`)} palette={palette} />
                    <NavListItem label={t("History")} Icon={HistoryIcon} onClick={(event) => handleOpen(event, LINKS.History)} palette={palette} />
                    <NavListItem label={t("Run", { count: 2 })} Icon={RoutineActiveIcon} onClick={(event) => handleOpen(event, `${LINKS.History}?type=${HistoryPageTabOption.RunsActive}`)} palette={palette} />
                    <NavListItem label={t("Award", { count: 2 })} Icon={AwardIcon} onClick={(event) => handleOpen(event, LINKS.Awards)} palette={palette} />
                    <NavListItem label={t("Premium")} Icon={PremiumIcon} onClick={(event) => handleOpen(event, LINKS.Premium)} palette={palette} />
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
                        <HelpIcon fill={palette.background.textPrimary} />
                    </Box>
                    <Typography variant="body1" sx={{ color: palette.background.textPrimary, ...noSelect, margin: "0 !important" }}>{t("AdditionalResources")}</Typography>
                    {isAdditionalResourcesOpen ? <ExpandMoreIcon fill={palette.background.textPrimary} style={{ marginLeft: "auto" }} /> : <ExpandLessIcon fill={palette.background.textPrimary} style={{ marginLeft: "auto" }} />}
                </Stack>
                <Collapse in={isAdditionalResourcesOpen} sx={{ display: "inline-block" }}>
                    <ContactInfo />
                </Collapse>
            </Box>
        </SwipeableDrawer>
    );
};
