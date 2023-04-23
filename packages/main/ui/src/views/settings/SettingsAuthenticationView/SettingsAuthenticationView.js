import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { LINKS } from "@local/consts";
import { DeleteIcon, EmailIcon, LogOutIcon, WalletIcon } from "@local/icons";
import { profileEmailUpdateValidation } from "@local/validation";
import { Box, Button, Stack, useTheme } from "@mui/material";
import { Formik } from "formik";
import { useCallback, useContext, useState } from "react";
import { useTranslation } from "react-i18next";
import { authLogOut } from "../../../api/generated/endpoints/auth_logOut";
import { userProfileEmailUpdate } from "../../../api/generated/endpoints/user_profileEmailUpdate";
import { useCustomMutation } from "../../../api/hooks";
import { mutationWrapper } from "../../../api/utils";
import { DeleteAccountDialog } from "../../../components/dialogs/DeleteAccountDialog/DeleteAccountDialog";
import { EmailList, WalletList } from "../../../components/lists/devices";
import { SettingsList } from "../../../components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "../../../components/navigation/SettingsTopBar/SettingsTopBar";
import { Subheader } from "../../../components/text/Subheader/Subheader";
import { SettingsAuthenticationForm } from "../../../forms/settings";
import { getCurrentUser, guestSession } from "../../../utils/authentication/session";
import { useProfileQuery } from "../../../utils/hooks/useProfileQuery";
import { PubSub } from "../../../utils/pubsub";
import { useLocation } from "../../../utils/route";
import { SessionContext } from "../../../utils/SessionContext";
export const SettingsAuthenticationView = ({ display = "page", }) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery();
    const [logOut] = useCustomMutation(authLogOut);
    const onLogOut = useCallback(() => {
        const { id } = getCurrentUser(session);
        mutationWrapper({
            mutation: logOut,
            input: { id },
            onSuccess: (data) => { PubSub.get().publishSession(data); },
            onError: () => { PubSub.get().publishSession(guestSession); },
        });
        PubSub.get().publishSession(guestSession);
        setLocation(LINKS.Home);
    }, [logOut, session, setLocation]);
    const updateWallets = useCallback((updatedList) => {
        if (!profile) {
            PubSub.get().publishSnack({ messageKey: "CouldNotReadProfile", severity: "Error" });
            return;
        }
        onProfileUpdate({ ...profile, wallets: updatedList });
    }, [onProfileUpdate, profile]);
    const numVerifiedEmails = profile?.emails?.filter((email) => email.verified)?.length ?? 0;
    const updateEmails = useCallback((updatedList) => {
        if (!profile) {
            PubSub.get().publishSnack({ messageKey: "CouldNotReadProfile", severity: "Error" });
            return;
        }
        onProfileUpdate({ ...profile, emails: updatedList });
    }, [onProfileUpdate, profile]);
    const numVerifiedWallets = profile?.wallets?.filter((wallet) => wallet.verified)?.length ?? 0;
    const [mutation, { loading: isUpdating }] = useCustomMutation(userProfileEmailUpdate);
    const [deleteOpen, setDeleteOpen] = useState(false);
    const openDelete = useCallback(() => setDeleteOpen(true), [setDeleteOpen]);
    const closeDelete = useCallback(() => setDeleteOpen(false), [setDeleteOpen]);
    return (_jsxs(_Fragment, { children: [_jsx(DeleteAccountDialog, { isOpen: deleteOpen, handleClose: closeDelete, zIndex: 100 }), _jsx(SettingsTopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "Authentication",
                } }), _jsxs(Stack, { direction: "row", children: [_jsx(SettingsList, {}), _jsxs(Box, { children: [_jsx(Subheader, { help: t("WalletListHelp"), Icon: WalletIcon, title: t("Wallet", { count: 2 }) }), _jsx(WalletList, { handleUpdate: updateWallets, list: profile?.wallets ?? [], numVerifiedEmails: numVerifiedEmails }), _jsx(Subheader, { help: t("EmailListHelp"), Icon: EmailIcon, title: t("Email", { count: 2 }) }), _jsx(EmailList, { handleUpdate: updateEmails, list: profile?.emails ?? [], numVerifiedWallets: numVerifiedWallets }), _jsx(Subheader, { help: t("PasswordChangeHelp"), title: t("ChangePassword") }), _jsx(Formik, { enableReinitialize: true, initialValues: {
                                    currentPassword: "",
                                    newPassword: "",
                                    newPasswordConfirmation: "",
                                }, onSubmit: (values, helpers) => {
                                    if (!profile) {
                                        PubSub.get().publishSnack({ messageKey: "CouldNotReadProfile", severity: "Error" });
                                        return;
                                    }
                                    mutationWrapper({
                                        mutation,
                                        input: {
                                            currentPassword: values.currentPassword,
                                            newPassword: values.newPassword,
                                        },
                                        onSuccess: (data) => { onProfileUpdate(data); },
                                        onError: () => { helpers.setSubmitting(false); },
                                    });
                                }, validationSchema: profileEmailUpdateValidation.update({}), children: (formik) => _jsx(SettingsAuthenticationForm, { display: display, isLoading: isProfileLoading || isUpdating, onCancel: formik.resetForm, ...formik }) }), _jsx(Button, { color: "secondary", onClick: onLogOut, startIcon: _jsx(LogOutIcon, {}), sx: {
                                    display: "flex",
                                    width: "min(100%, 400px)",
                                    marginLeft: "auto",
                                    marginRight: "auto",
                                    marginTop: 5,
                                    marginBottom: 2,
                                    whiteSpace: "nowrap",
                                }, children: t("LogOut") }), _jsx(Button, { onClick: openDelete, startIcon: _jsx(DeleteIcon, {}), sx: {
                                    background: palette.error.main,
                                    color: palette.error.contrastText,
                                    display: "flex",
                                    width: "min(100%, 400px)",
                                    marginLeft: "auto",
                                    marginRight: "auto",
                                    marginBottom: 2,
                                    whiteSpace: "nowrap",
                                }, children: t("DeleteAccount") })] })] })] }));
};
//# sourceMappingURL=SettingsAuthenticationView.js.map