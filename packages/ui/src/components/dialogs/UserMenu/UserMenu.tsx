import { API_CREDITS_MULTIPLIER, ActionOption, HistoryPageTabOption, LINKS, PreActionOption, ProfileUpdateInput, Session, SessionUser, SwitchCurrentAccountInput, User, endpointsAuth, endpointsUser, profileValidation, shapeProfile } from "@local/shared";
import { Box, Collapse, Divider, Link, List, ListItemButton, ListItemIcon, ListItemText, Palette, Stack, Typography, styled, useTheme } from "@mui/material";
import { useFormik } from "formik";
import React, { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../../api/fetchWrapper.js";
import { SocketService } from "../../../api/socket.js";
import { SessionContext } from "../../../contexts/session.js";
import { useLazyFetch } from "../../../hooks/useLazyFetch.js";
import { useMenu } from "../../../hooks/useMenu.js";
import { useWindowSize } from "../../../hooks/useWindowSize.js";
import { Icon, IconCommon, IconInfo } from "../../../icons/Icons.js";
import { useLocation } from "../../../route/router.js";
import { ProfileAvatar, noSelect } from "../../../styles.js";
import { SessionService, checkIfLoggedIn, getCurrentUser, guestSession } from "../../../utils/authentication/session.js";
import { ELEMENT_IDS } from "../../../utils/consts.js";
import { extractImageUrl } from "../../../utils/display/imageTools.js";
import { placeholderColor } from "../../../utils/display/listTools.js";
import { openObject } from "../../../utils/navigation/openObject.js";
import { Actions, performAction } from "../../../utils/navigation/quickActions.js";
import { MenuPayloads, PubSub } from "../../../utils/pubsub.js";
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
const SPACING_MULTIPLIER = 1.5;
const AVATAR_SIZE_PX = 50;
const helpIconInfo = { name: "Help", type: "Common" } as const;

const StyledListItemButton = styled(ListItemButton)(({ theme }) => ({
    paddingTop: theme.spacing(0.5),
    paddingBottom: theme.spacing(0.5),
}));

function NavListItem({ iconInfo, label, onClick, palette }: {
    iconInfo: IconInfo;
    label: string;
    onClick: (event: React.MouseEvent<HTMLElement>) => unknown;
    palette: Palette;
}) {
    return (
        <StyledListItemButton
            aria-label={label}
            onClick={onClick}
        >
            <ListItemIcon>
                <Icon
                    decorative
                    fill={palette.background.textPrimary}
                    info={iconInfo}
                />
            </ListItemIcon>
            <ListItemText primary={label} />
        </StyledListItemButton>
    );
}

const UserMenuDisplaySettingsBox = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "column",
    gap: theme.spacing(2),
    alignItems: "flex-start",
    minWidth: "fit-content",
    height: "fit-content",
    padding: theme.spacing(1),
}));
const SeeAllLink = styled(Link)({
    textAlign: "right",
});
const SeeAllLinkText = styled(Typography)(({ theme }) => ({
    marginRight: theme.spacing(SPACING_MULTIPLIER),
    marginBottom: theme.spacing(1),
}));
const CollapseBox = styled(Box)({
    display: "block",
    width: "100%",
});
const AccountList = styled(List)({
    paddingTop: 0,
    paddingBottom: 0,
});
const DividerStyled = styled(Divider)(({ theme }) => ({
    background: theme.palette.background.textSecondary,
}));
const DisplayHeader = styled(Stack)(({ theme }) => ({
    display: "flex",
    alignItems: "center",
    textAlign: "left",
    paddingLeft: theme.spacing(2),
    paddingRight: theme.spacing(2),
    paddingTop: theme.spacing(1),
    paddingBottom: theme.spacing(1),
}));
const BoxIcon = styled(Box)({
    minWidth: "56px",
    display: "flex",
    alignItems: "center",
});
const HeaderTypography = styled(Typography)(({ theme }) => ({
    color: theme.palette.background.textPrimary,
    ...noSelect,
    margin: "0 !important",
}));
const ExpandIcon = styled(Box)({
    marginLeft: "auto",
});

const leftHandedCheckboxStyle = { justifyContent: "flex-start" } as const;

const dialogSxs = {
    paper: {
        overflowY: "auto",
    },
} as const;

const anchorOrigin = {
    vertical: "bottom" as const,
    horizontal: "right" as const,
} as const;
const transformOrigin = {
    vertical: "top" as const,
    horizontal: "right" as const,
} as const;
const profileAvatarStyle = {
    marginRight: 2,
};

const MonospaceTypography = styled(Typography)({
    fontFamily: "monospace",
});

const CreditStack = styled(Stack)({
    marginTop: 1,
});

const PremiumBox = styled(Box)(({ theme }) => ({
    backgroundColor: `${theme.palette.secondary.main}15`,
    color: theme.palette.secondary.main,
    borderRadius: 1,
    paddingLeft: 1,
    paddingRight: 1,
    paddingTop: 0.25,
    paddingBottom: 0.25,
    display: "flex",
    alignItems: "center",
    gap: 0.5,
    boxShadow: `0 0 0 1px ${theme.palette.secondary.main}40`,
}));

const PremiumTypography = styled(Typography)({
    fontWeight: 500,
    fontSize: "0.75rem",
    lineHeight: 1,
});

const CreditTypography = styled(Typography)(({ theme }) => ({
    color: theme.palette.background.textSecondary,
    fontSize: "0.75rem",
}));

const MonospaceHandleTypography = styled(MonospaceTypography)(({ theme }) => ({
    color: theme.palette.background.textSecondary,
}));

/**
 * User menu dialog that displays user accounts, settings, and navigation options.
 * Appears as a dialog from the right side of the screen.
 */
export function UserMenu() {
    const session = useContext(SessionContext);
    const { breakpoints, palette } = useTheme();
    const [location, setLocation] = useLocation();
    const { t } = useTranslation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);
    const isLoggedIn = checkIfLoggedIn(session);

    // Display settings collapse
    const [isDisplaySettingsOpen, setIsDisplaySettingsOpen] = useState(false);
    const toggleDisplaySettings = useCallback(() => { setIsDisplaySettingsOpen(!isDisplaySettingsOpen); }, [isDisplaySettingsOpen]);
    const closeDisplaySettings = useCallback(() => { setIsDisplaySettingsOpen(false); }, []);

    // Handle opening and closing
    const onEvent = useCallback(function onEventCallback({ data }: MenuPayloads[typeof ELEMENT_IDS.UserMenu]) {
        if (!data) return;
        if (typeof data.isDisplaySettingsCollapsed === "boolean") {
            setIsDisplaySettingsOpen(!data.isDisplaySettingsCollapsed);
        }
    }, []);
    const { isOpen, close } = useMenu({
        id: ELEMENT_IDS.UserMenu,
        isMobile,
        onEvent,
    });

    // Get anchor element from menu data
    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
    useEffect(() => {
        const subscription = PubSub.get().subscribe("menu", (data) => {
            console.log("menu subscription", data);
            if (data.id === ELEMENT_IDS.UserMenu && data.data?.anchorEl) {
                setAnchorEl(data.data.anchorEl);
            }
        });
        return subscription;
    }, []);

    // When moving between mobile/desktop, publish current state
    useEffect(() => {
        PubSub.get().publish("menu", { id: ELEMENT_IDS.UserMenu, isOpen });
    }, [breakpoints, isOpen]);

    // Add this effect to close the menu on location change
    useEffect(() => {
        close();
    }, [location, close]);

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
        closeDisplaySettings();
    }, [close, closeDisplaySettings, formik]);

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

    const navItems = useMemo(function navItemsMemo() {
        // Only include Pro link when not logged in
        if (!isLoggedIn) {
            return [
                { label: t("Pro"), iconInfo: { name: "Premium" as const, type: "Common" as const }, link: LINKS.Pro, action: null },
                { label: t("Tutorial"), iconInfo: { name: "Help" as const, type: "Common" as const }, action: Actions.tutorial },
            ] as const;
        }

        // Include all links when logged in
        return [
            { label: t("Bookmark", { count: 2 }), iconInfo: { name: "BookmarkFilled" as const, type: "Common" as const }, link: `${LINKS.History}?type="${HistoryPageTabOption.Bookmarked}"`, action: null },
            { label: t("Calendar", { count: 2 }), iconInfo: { name: "Month" as const, type: "Common" as const }, link: LINKS.Calendar, action: null },
            { label: t("History", { count: 2 }), iconInfo: { name: "History" as const, type: "Common" as const }, link: LINKS.History, action: null },
            { label: t("Award", { count: 2 }), iconInfo: { name: "Award" as const, type: "Common" as const }, link: LINKS.Awards, action: null },
            { label: t("Pro"), iconInfo: { name: "Premium" as const, type: "Common" as const }, link: LINKS.Pro, action: null },
            { label: t("Settings"), iconInfo: { name: "Settings" as const, type: "Common" as const }, link: LINKS.Settings, action: null },
            { label: t("Tutorial"), iconInfo: { name: "Help" as const, type: "Common" as const }, action: Actions.tutorial },
        ] as const;
    }, [t, isLoggedIn]);

    const handleNavItemClick = useCallback(function handleNavItemClickCallback(event: React.MouseEvent<HTMLElement>, item: typeof navItems[number]) {
        if (item.action) {
            handleAction(event, item.action);
        } else if (item.link) {
            handleOpen(event, item.link);
        }
    }, [handleAction, handleOpen]);
    const handleTutorialClick = useCallback(function handleTutorialClickCallback(event: React.MouseEvent<HTMLElement>) {
        handleNavItemClick(event, { label: t("Tutorial"), iconInfo: { name: "Help" as const, type: "Common" as const }, action: Actions.tutorial });
    }, [handleNavItemClick, t]);

    return (
        <LargeDialog
            id={ELEMENT_IDS.UserMenu}
            isOpen={isOpen}
            onClose={handleClose}
            sxs={dialogSxs}
            anchorEl={anchorEl}
            anchorOrigin={anchorOrigin}
            transformOrigin={transformOrigin}
        >
            <AccountList>
                {accounts
                    .filter(account => account.id === userId)
                    .map((account) => {
                        const profileColors = placeholderColor(account.id);
                        function handleClick(event: React.MouseEvent<HTMLElement>) {
                            handleUserClick(event, account);
                        }
                        return (
                            <StyledListItemButton
                                key={account.id}
                                onClick={handleClick}
                            >
                                <ProfileAvatar
                                    src={extractImageUrl(account.profileImage, account.updated_at, AVATAR_SIZE_PX)}
                                    isBot={false}
                                    profileColors={profileColors}
                                    sx={profileAvatarStyle}
                                >
                                    <IconCommon
                                        decorative
                                        name="Profile"
                                    />
                                </ProfileAvatar>
                                <ListItemText
                                    primary={
                                        <Stack direction="column" spacing={0}>
                                            <Typography variant="body1">{account.name}</Typography>
                                            {account.handle && (
                                                <MonospaceHandleTypography variant="body2">
                                                    @{account.handle}
                                                </MonospaceHandleTypography>
                                            )}
                                        </Stack>
                                    }
                                    secondary={
                                        <CreditStack direction="row" spacing={1} alignItems="center">
                                            {account.hasPremium && (
                                                <PremiumBox>
                                                    <IconCommon
                                                        decorative
                                                        fill={palette.secondary.main}
                                                        name="Premium"
                                                        size={14}
                                                    />
                                                    <PremiumTypography variant="body2">
                                                        {t("Pro")}
                                                    </PremiumTypography>
                                                </PremiumBox>
                                            )}
                                            <CreditTypography variant="body2">
                                                {t("Credit", { count: 2 })}: ${(Number(BigInt(account.credits ?? "0") / BigInt(API_CREDITS_MULTIPLIER)) / 100).toFixed(2)}
                                            </CreditTypography>
                                        </CreditStack>
                                    }
                                />
                            </StyledListItemButton>
                        );
                    })}
            </AccountList>

            {/* User-specific navigation links and display settings */}
            {isLoggedIn && (
                <>
                    <List>
                        {navItems
                            .filter(item => item.label !== t("Tutorial"))
                            .map((item, index) => {
                                function handleClick(event: React.MouseEvent<HTMLElement>) {
                                    handleNavItemClick(event, item);
                                }
                                return (
                                    <NavListItem
                                        key={index}
                                        label={item.label}
                                        iconInfo={item.iconInfo}
                                        onClick={handleClick}
                                        palette={palette}
                                    />
                                );
                            })}
                    </List>

                    {/* Display Settings */}
                    <DisplayHeader direction="row" spacing={1} onClick={toggleDisplaySettings}>
                        <BoxIcon>
                            <IconCommon
                                decorative
                                name="DisplaySettings"
                                fill={palette.background.textPrimary}
                            />
                        </BoxIcon>
                        <HeaderTypography variant="body1">{t("Display")}</HeaderTypography>
                        <ExpandIcon aria-label={isDisplaySettingsOpen ? t("Shrink") : t("Expand")}>
                            {isDisplaySettingsOpen ?
                                <IconCommon
                                    decorative
                                    name="ExpandMore"
                                    fill={palette.background.textPrimary}
                                /> :
                                <IconCommon
                                    decorative
                                    name="ExpandLess"
                                    fill={palette.background.textPrimary}
                                />
                            }
                        </ExpandIcon>
                    </DisplayHeader>
                    <CollapseBox>
                        <Collapse in={isDisplaySettingsOpen}>
                            <UserMenuDisplaySettingsBox id={ELEMENT_IDS.UserMenuDisplaySettings}>
                                <ThemeSwitch updateServer />
                                <TextSizeButtons />
                                <LeftHandedCheckbox sx={leftHandedCheckboxStyle} />
                                <LanguageSelector />
                                {/* <FocusModeSelector /> */}
                            </UserMenuDisplaySettingsBox>
                            <SeeAllLink href={LINKS.SettingsDisplay}>
                                <SeeAllLinkText variant="body2">{t("SeeAll")}</SeeAllLinkText>
                            </SeeAllLink>
                        </Collapse>
                    </CollapseBox>
                </>
            )}

            <DividerStyled />

            {/* Other accounts and session actions */}
            <List>
                {accounts
                    .filter(account => account.id !== userId)
                    .map((account) => {
                        const profileColors = placeholderColor(account.id);
                        function handleClick(event: React.MouseEvent<HTMLElement>) {
                            handleUserClick(event, account);
                        }
                        return (
                            <StyledListItemButton
                                key={account.id}
                                onClick={handleClick}
                            >
                                <ProfileAvatar
                                    src={extractImageUrl(account.profileImage, account.updated_at, AVATAR_SIZE_PX)}
                                    isBot={false}
                                    profileColors={profileColors}
                                    sx={profileAvatarStyle}
                                >
                                    <IconCommon
                                        decorative
                                        name="Profile"
                                    />
                                </ProfileAvatar>
                                <ListItemText
                                    primary={
                                        <Stack direction="column" spacing={0}>
                                            <Typography variant="body1">{account.name}</Typography>
                                            <MonospaceHandleTypography variant="body2">
                                                @{account.handle}
                                            </MonospaceHandleTypography>
                                        </Stack>
                                    }
                                    secondary={
                                        <CreditStack direction="row" spacing={1} alignItems="center">
                                            {account.hasPremium && (
                                                <PremiumBox>
                                                    <IconCommon
                                                        decorative
                                                        fill={palette.secondary.main}
                                                        name="Premium"
                                                        size={14}
                                                    />
                                                    <PremiumTypography variant="body2">
                                                        {t("Pro")}
                                                    </PremiumTypography>
                                                </PremiumBox>
                                            )}
                                            <CreditTypography variant="body2">
                                                {t("Credit", { count: 2 })}: ${(Number(BigInt(account.credits ?? "0") / BigInt(API_CREDITS_MULTIPLIER)) / 100).toFixed(2)}
                                            </CreditTypography>
                                        </CreditStack>
                                    }
                                />
                            </StyledListItemButton>
                        );
                    })}

                {/* Show login/signup button when not logged in */}
                {!isLoggedIn && (
                    <StyledListItemButton
                        aria-label={t("LogInSignUp")}
                        onClick={handleLoginSignup}
                    >
                        <ListItemIcon>
                            <IconCommon
                                decorative
                                name="LogIn"
                                fill={palette.background.textPrimary}
                            />
                        </ListItemIcon>
                        <ListItemText primary={t("LogInSignUp")} />
                    </StyledListItemButton>
                )}

                {/* Show add account button when logged in and under max accounts */}
                {isLoggedIn && accounts.length < MAX_ACCOUNTS && (
                    <StyledListItemButton
                        aria-label={t("AddAccount")}
                        onClick={handleLoginSignup}
                    >
                        <ListItemIcon>
                            <IconCommon
                                decorative
                                name="Plus"
                                fill={palette.background.textPrimary}
                            />
                        </ListItemIcon>
                        <ListItemText primary={t("AddAccount")} />
                    </StyledListItemButton>
                )}

                {/* Show logout button when logged in */}
                {isLoggedIn && accounts.length > 0 && (
                    <StyledListItemButton
                        aria-label={t("LogOut")}
                        onClick={handleLogOut}
                    >
                        <ListItemIcon>
                            <IconCommon
                                decorative
                                name="LogOut"
                                fill={palette.background.textPrimary}
                            />
                        </ListItemIcon>
                        <ListItemText primary={t("LogOut")} />
                    </StyledListItemButton>
                )}
            </List>
            <DividerStyled />
            {/* Help and support content */}
            <List>
                <NavListItem
                    label={t("Tutorial")}
                    iconInfo={helpIconInfo}
                    onClick={handleTutorialClick}
                    palette={palette}
                />
            </List>
            <DividerStyled />
            <ContactInfo />
        </LargeDialog>
    );
}

