import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { LINKS } from "@local/consts";
import { AwardIcon, BookmarkFilledIcon, CloseIcon, DisplaySettingsIcon, ExpandLessIcon, ExpandMoreIcon, HelpIcon, HistoryIcon, LogOutIcon, PlusIcon, PremiumIcon, SettingsIcon, UserIcon } from "@local/icons";
import { userValidation } from "@local/validation";
import { Box, Collapse, Divider, IconButton, List, ListItem, ListItemIcon, ListItemText, SwipeableDrawer, Typography, useTheme } from "@mui/material";
import { Stack } from "@mui/system";
import { useFormik } from "formik";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { authLogOut } from "../../../api/generated/endpoints/auth_logOut";
import { authSwitchCurrentAccount } from "../../../api/generated/endpoints/auth_switchCurrentAccount";
import { userProfileUpdate } from "../../../api/generated/endpoints/user_profileUpdate";
import { useCustomMutation } from "../../../api/hooks";
import { mutationWrapper } from "../../../api/utils";
import { noSelect } from "../../../styles";
import { getCurrentUser, guestSession } from "../../../utils/authentication/session";
import { useIsLeftHanded } from "../../../utils/hooks/useIsLeftHanded";
import { useWindowSize } from "../../../utils/hooks/useWindowSize";
import { PubSub } from "../../../utils/pubsub";
import { useLocation } from "../../../utils/route";
import { HistoryPageTabOption } from "../../../utils/search/objectToSearch";
import { SessionContext } from "../../../utils/SessionContext";
import { shapeProfile } from "../../../utils/shape/models/profile";
import { FocusModeSelector } from "../../inputs/FocusModeSelector/FocusModeSelector";
import { LanguageSelector } from "../../inputs/LanguageSelector/LanguageSelector";
import { LeftHandedCheckbox } from "../../inputs/LeftHandedCheckbox/LeftHandedCheckbox";
import { TextSizeButtons } from "../../inputs/TextSizeButtons/TextSizeButtons";
import { ThemeSwitch } from "../../inputs/ThemeSwitch/ThemeSwitch";
import { ContactInfo } from "../../navigation/ContactInfo/ContactInfo";
const MAX_ACCOUNTS = 10;
export const AccountMenu = ({ anchorEl, onClose, }) => {
    const session = useContext(SessionContext);
    const { breakpoints, palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const isLeftHanded = useIsLeftHanded();
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);
    const open = Boolean(anchorEl);
    const [isDisplaySettingsOpen, setIsDisplaySettingsOpen] = useState(false);
    const toggleDisplaySettings = useCallback(() => { setIsDisplaySettingsOpen(!isDisplaySettingsOpen); }, [isDisplaySettingsOpen]);
    const closeDisplaySettings = useCallback(() => { setIsDisplaySettingsOpen(false); }, []);
    const [isAdditionalResourcesOpen, setIsAdditionalResourcesOpen] = useState(false);
    const toggleAdditionalResources = useCallback(() => { setIsAdditionalResourcesOpen(!isAdditionalResourcesOpen); }, [isAdditionalResourcesOpen]);
    const closeAdditionalResources = useCallback(() => { setIsAdditionalResourcesOpen(false); }, []);
    const [mutation] = useCustomMutation(userProfileUpdate);
    const formik = useFormik({
        initialValues: {
            theme: getCurrentUser(session).theme ?? "light",
        },
        enableReinitialize: true,
        validationSchema: userValidation.update({}),
        onSubmit: (values) => {
            if (!userId) {
                return;
            }
            if (!formik.isValid)
                return;
            const input = shapeProfile.update({
                id: userId,
                theme: getCurrentUser(session).theme ?? "light",
            }, {
                id: userId,
                theme: values.theme,
            });
            if (!input || Object.keys(input).length === 0) {
                formik.setSubmitting(false);
                return;
            }
            mutationWrapper({
                mutation,
                input,
                onSuccess: () => { formik.setSubmitting(false); },
                onError: () => { formik.setSubmitting(false); },
            });
        },
    });
    const handleClose = useCallback((event) => {
        formik.handleSubmit();
        onClose(event);
        closeAdditionalResources();
        closeDisplaySettings();
    }, [closeAdditionalResources, closeDisplaySettings, formik, onClose]);
    const [switchCurrentAccount] = useCustomMutation(authSwitchCurrentAccount);
    const handleUserClick = useCallback((event, user) => {
        handleClose(event);
        if (userId === user.id) {
            setLocation(LINKS.Profile);
        }
        else {
            mutationWrapper({
                mutation: switchCurrentAccount,
                input: { id: user.id },
                successMessage: () => ({ key: "LoggedInAs", variables: { name: user.name ?? user.handle ?? "" } }),
                onSuccess: (data) => { PubSub.get().publishSession(data); },
            });
        }
    }, [handleClose, userId, setLocation, switchCurrentAccount]);
    const handleAddAccount = useCallback((event) => {
        setLocation(LINKS.Start);
        handleClose(event);
    }, [handleClose, setLocation]);
    const [logOut] = useCustomMutation(authLogOut);
    const handleLogOut = useCallback((event) => {
        handleClose(event);
        const user = getCurrentUser(session);
        mutationWrapper({
            mutation: logOut,
            input: { id: user.id },
            successMessage: () => ({ key: "LoggedOutOf", variables: { name: user.name ?? user.handle ?? "" } }),
            onSuccess: (data) => { PubSub.get().publishSession(data); },
            onError: () => { PubSub.get().publishSession(guestSession); },
        });
        setLocation(LINKS.Home);
    }, [handleClose, session, logOut, setLocation]);
    const handleOpen = useCallback((event, link) => {
        setLocation(link);
        handleClose(event);
    }, [handleClose, setLocation]);
    const handleOpenSettings = useCallback((event) => {
        handleOpen(event, LINKS.Settings);
    }, [handleOpen]);
    const handleOpenBookmarks = useCallback((event) => {
        handleOpen(event, `${LINKS.History}?type=${HistoryPageTabOption.Bookmarked}`);
    }, [handleOpen]);
    const handleOpenHistory = useCallback((event) => {
        handleOpen(event, LINKS.History);
    }, [handleOpen]);
    const handleOpenAwards = useCallback((event) => {
        handleOpen(event, LINKS.Awards);
    }, [handleOpen]);
    const handleOpenPremium = useCallback((event) => {
        handleOpen(event, LINKS.Premium);
    }, [handleOpen]);
    const accounts = useMemo(() => session?.users ?? [], [session?.users]);
    const profileListItems = accounts.map((account) => (_jsxs(ListItem, { button: true, onClick: (event) => handleUserClick(event, account), sx: {
            background: account.id === userId ? palette.secondary.light : palette.background.default,
        }, children: [_jsx(ListItemIcon, { children: _jsx(UserIcon, { fill: palette.background.textPrimary }) }), _jsx(ListItemText, { primary: account.name ?? account.handle })] }, account.id)), [accounts, handleUserClick]);
    return (_jsxs(SwipeableDrawer, { anchor: (isMobile && isLeftHanded) ? "left" : "right", open: open, onOpen: () => { }, onClose: handleClose, sx: {
            zIndex: 20000,
            "& .MuiDrawer-paper": {
                background: palette.background.default,
                overflowY: "auto",
            },
        }, children: [_jsx(Stack, { direction: 'row', spacing: 1, sx: {
                    ...noSelect,
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "space-between",
                    padding: 1,
                    background: palette.primary.dark,
                    color: palette.primary.contrastText,
                    textAlign: "center",
                    fontSize: { xs: "1.5rem", sm: "2rem" },
                    height: "64px",
                    paddingRight: 3,
                }, children: _jsx(IconButton, { "aria-label": "close", edge: "end", onClick: handleClose, sx: {
                        marginLeft: (isMobile && isLeftHanded) ? "unset" : "auto",
                        marginRight: (isMobile && isLeftHanded) ? "auto" : "unset",
                    }, children: _jsx(CloseIcon, { fill: palette.primary.contrastText, width: "40px", height: "40px" }) }) }), _jsxs(List, { sx: { paddingTop: 0, paddingBottom: 0 }, children: [profileListItems, _jsx(Divider, { sx: { background: palette.background.textSecondary } }), accounts.length < MAX_ACCOUNTS && _jsxs(ListItem, { button: true, onClick: handleAddAccount, children: [_jsx(ListItemIcon, { children: _jsx(PlusIcon, { fill: palette.background.textPrimary }) }), _jsx(ListItemText, { primary: t("AddAccount") })] }), accounts.length > 0 && _jsxs(ListItem, { button: true, onClick: handleLogOut, children: [_jsx(ListItemIcon, { children: _jsx(LogOutIcon, { fill: palette.background.textPrimary }) }), _jsx(ListItemText, { primary: t("LogOut") })] })] }), _jsx(Divider, { sx: { background: palette.background.textSecondary } }), _jsxs(Stack, { direction: "row", spacing: 1, onClick: toggleDisplaySettings, sx: {
                    display: "flex",
                    alignItems: "center",
                    textAlign: "left",
                    paddingLeft: 2,
                    paddingRight: 2,
                    paddingTop: 1,
                    paddingBottom: 1,
                }, children: [_jsx(Box, { sx: { minWidth: "56px", display: "flex", alignItems: "center" }, children: _jsx(DisplaySettingsIcon, { fill: palette.background.textPrimary }) }), _jsx(Typography, { variant: "body1", sx: { color: palette.background.textPrimary, ...noSelect, margin: "0 !important" }, children: t("Display") }), isDisplaySettingsOpen ? _jsx(ExpandMoreIcon, { fill: palette.background.textPrimary, style: { marginLeft: "auto" } }) : _jsx(ExpandLessIcon, { fill: palette.background.textPrimary, style: { marginLeft: "auto" } })] }), _jsx(Collapse, { in: isDisplaySettingsOpen, sx: { display: "inline-block", minHeight: "auto!important" }, children: _jsxs(Stack, { direction: "column", spacing: 2, sx: {
                        minWidth: "fit-content",
                        height: "fit-content",
                        padding: 1,
                    }, children: [_jsx(ThemeSwitch, {}), _jsx(TextSizeButtons, {}), _jsx(LeftHandedCheckbox, {}), _jsx(LanguageSelector, {}), _jsx(FocusModeSelector, {})] }) }), _jsx(Divider, { sx: { background: palette.background.textSecondary } }), _jsxs(List, { children: [_jsxs(ListItem, { button: true, onClick: handleOpenSettings, children: [_jsx(ListItemIcon, { children: _jsx(SettingsIcon, { fill: palette.background.textPrimary }) }), _jsx(ListItemText, { primary: t("Settings") })] }), _jsxs(ListItem, { button: true, onClick: handleOpenBookmarks, children: [_jsx(ListItemIcon, { children: _jsx(BookmarkFilledIcon, { fill: palette.background.textPrimary }) }), _jsx(ListItemText, { primary: t("Bookmark", { count: 2 }) })] }), _jsxs(ListItem, { button: true, onClick: handleOpenHistory, children: [_jsx(ListItemIcon, { children: _jsx(HistoryIcon, { fill: palette.background.textPrimary }) }), _jsx(ListItemText, { primary: t("History") })] }), _jsxs(ListItem, { button: true, onClick: handleOpenAwards, children: [_jsx(ListItemIcon, { children: _jsx(AwardIcon, { fill: palette.background.textPrimary }) }), _jsx(ListItemText, { primary: t("Award", { count: 2 }) })] }), _jsxs(ListItem, { button: true, onClick: handleOpenPremium, children: [_jsx(ListItemIcon, { children: _jsx(PremiumIcon, { fill: palette.background.textPrimary }) }), _jsx(ListItemText, { primary: t("Premium") })] })] }), _jsx(Divider, { sx: { background: palette.background.textSecondary } }), _jsxs(Stack, { direction: "row", spacing: 1, onClick: toggleAdditionalResources, sx: {
                    display: "flex",
                    alignItems: "center",
                    textAlign: "left",
                    paddingLeft: 2,
                    paddingRight: 2,
                    paddingTop: 1,
                    paddingBottom: 1,
                }, children: [_jsx(Box, { sx: { minWidth: "56px", display: "flex", alignItems: "center" }, children: _jsx(HelpIcon, { fill: palette.background.textPrimary }) }), _jsx(Typography, { variant: "body1", sx: { color: palette.background.textPrimary, ...noSelect, margin: "0 !important" }, children: t("AdditionalResources") }), isAdditionalResourcesOpen ? _jsx(ExpandMoreIcon, { fill: palette.background.textPrimary, style: { marginLeft: "auto" } }) : _jsx(ExpandLessIcon, { fill: palette.background.textPrimary, style: { marginLeft: "auto" } })] }), _jsx(Collapse, { in: isAdditionalResourcesOpen, sx: { display: "inline-block" }, children: _jsx(ContactInfo, {}) })] }));
};
//# sourceMappingURL=AccountMenu.js.map