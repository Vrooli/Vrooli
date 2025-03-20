import { API_CREDITS_MULTIPLIER, ActionOption, HistoryPageTabOption, LINKS, PreActionOption, ProfileUpdateInput, Session, SessionUser, SwitchCurrentAccountInput, User, endpointsAuth, endpointsUser, profileValidation, shapeProfile } from "@local/shared";
import { Avatar, Box, Collapse, Divider, Link, List, ListItem, ListItemIcon, ListItemText, Palette, Typography, styled, useTheme } from "@mui/material";
import { Stack } from "@mui/system";
import { useFormik } from "formik";
import React, { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../../api/fetchWrapper.js";
import { SocketService } from "../../../api/socket.js";
import { SessionContext } from "../../../contexts.js";
import { useLazyFetch } from "../../../hooks/useLazyFetch.js";
import { useMenu } from "../../../hooks/useMenu.js";
import { useWindowSize } from "../../../hooks/useWindowSize.js";
import { AwardIcon, BookmarkFilledIcon, DisplaySettingsIcon, ExpandLessIcon, ExpandMoreIcon, HelpIcon, HistoryIcon, InfoIcon, LogInIcon, LogOutIcon, MonthIcon, PlusIcon, PremiumIcon, RoutineActiveIcon, SettingsIcon, UserIcon } from "../../../icons/common.js";
import { useLocation } from "../../../route/router.js";
import { noSelect } from "../../../styles.js";
import { SvgComponent } from "../../../types.js";
import { SessionService, checkIfLoggedIn, getCurrentUser, guestSession } from "../../../utils/authentication/session.js";
import { ELEMENT_IDS } from "../../../utils/consts.js";
import { extractImageUrl } from "../../../utils/display/imageTools.js";
import { openObject } from "../../../utils/navigation/openObject.js";
import { Actions, performAction } from "../../../utils/navigation/quickActions.js";
import { NAV_ACTION_TAGS, NavAction, getUserActions } from "../../../utils/navigation/userActions.js";
import { MenuPayloads, PubSub } from "../../../utils/pubsub.js";
import { FocusModeSelector } from "../../inputs/FocusModeSelector/FocusModeSelector.js";
import { LanguageSelector } from "../../inputs/LanguageSelector/LanguageSelector.js";
import { LeftHandedCheckbox } from "../../inputs/LeftHandedCheckbox/LeftHandedCheckbox.js";
import { TextSizeButtons } from "../../inputs/TextSizeButtons/TextSizeButtons.js";
import { ThemeSwitch } from "../../inputs/ThemeSwitch/ThemeSwitch.js";
import { ContactInfo } from "../../navigation/ContactInfo.js";
import { LargeDialog } from "../LargeDialog/LargeDialog.js";

/**
 * Maximum accounts to sign in with. 
 * Limited by cookie size (4kb)
 */
const MAX_ACCOUNTS = 10;
const AVATAR_SIZE_PX = 50;

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

const StyledAvatar = styled(Avatar)(({ theme }) => ({
    marginRight: theme.spacing(2),
}));
const UserMenuDisplaySettingsBox = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "column",
    gap: theme.spacing(2),
    minWidth: "fit-content",
    height: "fit-content",
    padding: theme.spacing(1),
}));

const dialogSxs = {
    paper: {
        overflowY: "auto",
    },
};
const seeAllLinkStyle = { textAlign: "right" } as const;
const seeAllLinkTextStyle = { marginRight: "12px", marginBottom: "8px" } as const;

/**
 * User menu dialog that displays user accounts, settings, and navigation options.
 * Appears as a dialog from the right side of the screen.
 */
export function UserMenu() {
    const session = useContext(SessionContext);
    const { breakpoints, palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const navActions = useMemo<NavAction[]>(() => getUserActions({ session, exclude: [NAV_ACTION_TAGS.LogIn] }), [session]);

    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);
    const isLoggedIn = checkIfLoggedIn(session);

    // Display settings collapse
    const [isDisplaySettingsOpen, setIsDisplaySettingsOpen] = useState(false);
    const toggleDisplaySettings = useCallback(() => { setIsDisplaySettingsOpen(!isDisplaySettingsOpen); }, [isDisplaySettingsOpen]);
    const closeDisplaySettings = useCallback(() => { setIsDisplaySettingsOpen(false); }, []);

    // Additional resources collapse
    const [isAdditionalResourcesOpen, setIsAdditionalResourcesOpen] = useState(false);
    const toggleAdditionalResources = useCallback(() => { setIsAdditionalResourcesOpen(!isAdditionalResourcesOpen); }, [isAdditionalResourcesOpen]);
    const closeAdditionalResources = useCallback(() => { setIsAdditionalResourcesOpen(false); }, []);

    // Handle opening and closing
    const onEvent = useCallback(function onEventCallback({ data }: MenuPayloads[typeof ELEMENT_IDS.UserMenu]) {
        if (!data) return;
        if (typeof data.isAdditionalResourcesCollapsed === "boolean") {
            setIsAdditionalResourcesOpen(!data.isAdditionalResourcesCollapsed);
        }
        if (typeof data.isDisplaySettingsCollapsed === "boolean") {
            setIsDisplaySettingsOpen(!data.isDisplaySettingsCollapsed);
        }
    }, []);
    const { isOpen, close } = useMenu({
        id: ELEMENT_IDS.UserMenu,
        isMobile,
        onEvent,
    });
    // When moving between mobile/desktop, publish current state
    useEffect(() => {
        PubSub.get().publish("menu", { id: ELEMENT_IDS.UserMenu, isOpen });
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

    const handleClose = useCallback((_event: unknown, _reason?: "backdropClick" | "escapeKeyDown") => {
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

    const handleLoginSignup = useCallback((event: React.MouseEvent<HTMLElement>) => {
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
                SessionService.logOut();
            },
        });
        setLocation(LINKS.Home);
    }, [handleClose, isMobile, session, logOut, setLocation]);

    const handleOpen = useCallback(function handleOpenCallback(event: React.MouseEvent<HTMLElement>, link: string) {
        setLocation(link);
        if (isMobile) handleClose(event);
    }, [handleClose, isMobile, setLocation]);

    const handleAction = useCallback(function handleActionCallback(event: React.MouseEvent<HTMLElement>, action: PreActionOption | ActionOption) {
        if (isMobile) handleClose(event);
        performAction(action, session);
    }, [handleClose, isMobile, session]);

    const accounts = useMemo(() => session?.users ?? [], [session?.users]);
    const profileListItems = accounts.map((account) => {
        function handleClick(event: React.MouseEvent<HTMLElement>) {
            handleUserClick(event, account);
        }

        const listItemStyle = {
            background: account.id === userId ? palette.secondary.light : palette.background.default,
        };

        return (
            <ListItem
                button
                key={account.id}
                onClick={handleClick}
                sx={listItemStyle}
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
        // Only include Pro link when not logged in
        if (!isLoggedIn) {
            return [
                { label: t("Pro"), Icon: PremiumIcon, link: LINKS.Pro, action: null },
                { label: t("Tutorial"), Icon: HelpIcon, action: Actions.tutorial },
            ] as const;
        }

        // Include all links when logged in
        return [
            { label: t("Bookmark", { count: 2 }), Icon: BookmarkFilledIcon, link: `${LINKS.History}?type="${HistoryPageTabOption.Bookmarked}"`, action: null },
            { label: t("Calendar", { count: 2 }), Icon: MonthIcon, link: LINKS.Calendar, action: null },
            { label: t("View", { count: 2 }), Icon: HistoryIcon, link: `${LINKS.History}?type="${HistoryPageTabOption.Viewed}"`, action: null },
            { label: t("Run", { count: 2 }), Icon: RoutineActiveIcon, link: `${LINKS.History}?type="${HistoryPageTabOption.RunsActive}"`, action: null },
            { label: t("Award", { count: 2 }), Icon: AwardIcon, link: LINKS.Awards, action: null },
            { label: t("Pro"), Icon: PremiumIcon, link: LINKS.Pro, action: null },
            { label: t("Settings"), Icon: SettingsIcon, link: LINKS.Settings, action: null },
            { label: t("Tutorial"), Icon: HelpIcon, action: Actions.tutorial },
        ] as const;
    }, [t, isLoggedIn]);

    const handleNavItemClick = useCallback(function handleNavItemClickCallback(event: React.MouseEvent<HTMLElement>, item: typeof navItems[number]) {
        if (item.action) {
            handleAction(event, item.action);
        } else if (item.link) {
            handleOpen(event, item.link);
        }
    }, [handleAction, handleOpen]);

    const accountListStyle = { paddingTop: 0, paddingBottom: 0 };
    const dividerStyle = { background: palette.background.textSecondary };

    const displayHeaderStyle = {
        display: "flex",
        alignItems: "center",
        textAlign: "left",
        paddingLeft: 2,
        paddingRight: 2,
        paddingTop: 1,
        paddingBottom: 1,
    };

    const boxIconStyle = { minWidth: "56px", display: "flex", alignItems: "center" };
    const typographyStyle = { color: palette.background.textPrimary, ...noSelect, margin: "0 !important" };
    const expandIconStyle = { marginLeft: "auto" };
    const collapseStyle = { display: "inline-block", minHeight: "auto!important" };

    const themeSwitchStyle = { justifyContent: "flex-start" };
    const leftHandedCheckboxStyle = { justifyContent: "flex-start" };

    const additionalResourcesStyle = {
        display: "flex",
        alignItems: "center",
        textAlign: "left",
        paddingLeft: 2,
        paddingRight: 2,
        paddingTop: 1,
        paddingBottom: 1,
    };

    const additionalResourcesCollapseStyle = { display: "inline-block" };

    return (
        <LargeDialog
            id={ELEMENT_IDS.UserMenu}
            isOpen={isOpen}
            onClose={handleClose}
            sxs={dialogSxs}
        >
            {/* List of logged/in accounts and authentication-related actions */}
            <List
                id={ELEMENT_IDS.UserMenuAccountList}
                sx={accountListStyle}
            >
                {profileListItems}
                <Divider sx={dividerStyle} />
                {/* Show login/signup button when not logged in */}
                {!isLoggedIn && (
                    <ListItem button onClick={handleLoginSignup}>
                        <ListItemIcon>
                            <LogInIcon fill={palette.background.textPrimary} />
                        </ListItemIcon>
                        <ListItemText primary={"Log In/Sign Up"} />
                    </ListItem>
                )}
                {/* Show add account button when logged in and under max accounts */}
                {isLoggedIn && accounts.length < MAX_ACCOUNTS && (
                    <ListItem button onClick={handleLoginSignup}>
                        <ListItemIcon>
                            <PlusIcon fill={palette.background.textPrimary} />
                        </ListItemIcon>
                        <ListItemText primary={t("AddAccount")} />
                    </ListItem>
                )}
                {/* Show logout button when logged in */}
                {isLoggedIn && accounts.length > 0 && (
                    <ListItem button onClick={handleLogOut}>
                        <ListItemIcon>
                            <LogOutIcon fill={palette.background.textPrimary} />
                        </ListItemIcon>
                        <ListItemText primary={t("LogOut")} />
                    </ListItem>
                )}
            </List>
            <Divider sx={dividerStyle} />
            {/* Display Settings */}
            <Stack id="side-menu-display-header" direction="row" spacing={1} onClick={toggleDisplaySettings} sx={displayHeaderStyle}>
                <Box sx={boxIconStyle}>
                    <DisplaySettingsIcon fill={palette.background.textPrimary} />
                </Box>
                <Typography variant="body1" sx={typographyStyle}>{t("Display")}</Typography>
                {isDisplaySettingsOpen ?
                    <ExpandMoreIcon fill={palette.background.textPrimary} style={expandIconStyle} /> :
                    <ExpandLessIcon fill={palette.background.textPrimary} style={expandIconStyle} />
                }
            </Stack>
            <Collapse in={isDisplaySettingsOpen} sx={collapseStyle}>
                <UserMenuDisplaySettingsBox id={ELEMENT_IDS.UserMenuDisplaySettings}>
                    <ThemeSwitch updateServer sx={themeSwitchStyle} />
                    <TextSizeButtons />
                    <LeftHandedCheckbox sx={leftHandedCheckboxStyle} />
                    <LanguageSelector />
                    <FocusModeSelector />
                </UserMenuDisplaySettingsBox>
                <Link
                    href={LINKS.SettingsDisplay}
                    sx={seeAllLinkStyle}
                >
                    <Typography variant="body2" sx={seeAllLinkTextStyle}>{t("SeeAll")}</Typography>
                </Link>
            </Collapse>
            <Divider sx={dividerStyle} />
            {/* List of quick links */}
            <List id={ELEMENT_IDS.UserMenuQuickLinks}>
                {/* Main navigation links, if not mobile and logged in */}
                {!isMobile && isLoggedIn && navActions.map((action) => {
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
            <Divider sx={dividerStyle} />
            {/* Additional Resources */}
            <Stack direction="row" spacing={1} onClick={toggleAdditionalResources} sx={additionalResourcesStyle}>
                <Box sx={boxIconStyle}>
                    <InfoIcon fill={palette.background.textPrimary} />
                </Box>
                <Typography variant="body1" sx={typographyStyle}>{t("AdditionalResources")}</Typography>
                {isAdditionalResourcesOpen ?
                    <ExpandMoreIcon fill={palette.background.textPrimary} style={expandIconStyle} /> :
                    <ExpandLessIcon fill={palette.background.textPrimary} style={expandIconStyle} />
                }
            </Stack>
            <Collapse in={isAdditionalResourcesOpen} sx={additionalResourcesCollapseStyle}>
                <ContactInfo />
            </Collapse>
        </LargeDialog>
    );
}

