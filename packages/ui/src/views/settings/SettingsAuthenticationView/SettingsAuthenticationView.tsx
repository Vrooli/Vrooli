import { DeleteIcon, Email, EmailIcon, endpointPostAuthLogout, endpointPutProfileEmail, LINKS, LogOutIcon, LogOutInput, ProfileEmailUpdateInput, profileEmailUpdateValidation, Session, User, Wallet, WalletIcon } from "@local/shared";
import { Box, Button, Stack, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { DeleteAccountDialog } from "components/dialogs/DeleteAccountDialog/DeleteAccountDialog";
import { EmailList, WalletList } from "components/lists/devices";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { Title } from "components/text/Title/Title";
import { Formik } from "formik";
import { SettingsAuthenticationForm } from "forms/settings";
import { useCallback, useContext, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { getCurrentUser, guestSession } from "utils/authentication/session";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { useProfileQuery } from "utils/hooks/useProfileQuery";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { SettingsAuthenticationViewProps } from "../types";

export const SettingsAuthenticationView = ({
    display = "page",
    onClose,
    zIndex,
}: SettingsAuthenticationViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery();

    const [logOut] = useLazyFetch<LogOutInput, Session>(endpointPostAuthLogout);
    const onLogOut = useCallback(() => {
        const { id } = getCurrentUser(session);
        fetchLazyWrapper<LogOutInput, Session>({
            fetch: logOut,
            inputs: { id },
            onSuccess: (data) => { PubSub.get().publishSession(data); },
            // If error, log out anyway
            onError: () => { PubSub.get().publishSession(guestSession); },
        });
        PubSub.get().publishSession(guestSession);
        setLocation(LINKS.Home);
    }, [logOut, session, setLocation]);

    const updateWallets = useCallback((updatedList: Wallet[]) => {
        if (!profile) {
            PubSub.get().publishSnack({ messageKey: "CouldNotReadProfile", severity: "Error" });
            return;
        }
        onProfileUpdate({ ...profile, wallets: updatedList });
    }, [onProfileUpdate, profile]);
    const numVerifiedEmails = profile?.emails?.filter((email) => email.verified)?.length ?? 0;

    const updateEmails = useCallback((updatedList: Email[]) => {
        if (!profile) {
            PubSub.get().publishSnack({ messageKey: "CouldNotReadProfile", severity: "Error" });
            return;
        }
        onProfileUpdate({ ...profile, emails: updatedList });
    }, [onProfileUpdate, profile]);
    const numVerifiedWallets = profile?.wallets?.filter((wallet) => wallet.verified)?.length ?? 0;

    // Handle update
    const [update, { loading: isUpdating }] = useLazyFetch<ProfileEmailUpdateInput, User>(endpointPutProfileEmail);

    const [deleteOpen, setDeleteOpen] = useState<boolean>(false);
    const openDelete = useCallback(() => setDeleteOpen(true), [setDeleteOpen]);
    const closeDelete = useCallback(() => setDeleteOpen(false), [setDeleteOpen]);

    return (
        <>
            {/* Delete account confirmation dialog */}
            <DeleteAccountDialog
                isOpen={deleteOpen}
                handleClose={closeDelete}
                zIndex={zIndex + 1}
            />
            <SettingsTopBar
                display={display}
                onClose={onClose}
                title={t("Authentication")}
                zIndex={zIndex}
            />
            <Stack direction="row" mt={2}>
                <SettingsList />
                <Box m="auto">
                    <Title
                        help={t("WalletListHelp")}
                        Icon={WalletIcon}
                        title={t("Wallet", { count: 2 })}
                        variant="subheader"
                        zIndex={zIndex}
                    />
                    <WalletList
                        handleUpdate={updateWallets}
                        list={profile?.wallets ?? []}
                        numVerifiedEmails={numVerifiedEmails}
                        zIndex={zIndex}
                    />
                    <Title
                        help={t("EmailListHelp")}
                        Icon={EmailIcon}
                        title={t("Email", { count: 2 })}
                        variant="subheader"
                        zIndex={zIndex}
                    />
                    <EmailList
                        handleUpdate={updateEmails}
                        list={profile?.emails ?? []}
                        numVerifiedWallets={numVerifiedWallets}
                    />
                    <Title
                        help={t("PasswordChangeHelp")}
                        title={t("ChangePassword")}
                        variant="subheader"
                        zIndex={zIndex}
                    />
                    <Formik
                        enableReinitialize={true}
                        initialValues={{
                            currentPassword: "",
                            newPassword: "",
                            newPasswordConfirmation: "",
                        } as ProfileEmailUpdateInput}
                        onSubmit={(values, helpers) => {
                            if (!profile) {
                                PubSub.get().publishSnack({ messageKey: "CouldNotReadProfile", severity: "Error" });
                                return;
                            }
                            fetchLazyWrapper<ProfileEmailUpdateInput, User>({
                                fetch: update,
                                inputs: {
                                    currentPassword: values.currentPassword,
                                    newPassword: values.newPassword,
                                },
                                onSuccess: (data) => { onProfileUpdate(data); },
                                onError: () => { helpers.setSubmitting(false); },
                            });
                        }}
                        validationSchema={profileEmailUpdateValidation.update({})}
                    >
                        {(formik) => <SettingsAuthenticationForm
                            display={display}
                            isLoading={isProfileLoading || isUpdating}
                            onCancel={formik.resetForm}
                            zIndex={zIndex}
                            {...formik}
                        />}
                    </Formik>
                    <Button
                        color="secondary"
                        onClick={onLogOut}
                        startIcon={<LogOutIcon />}
                        variant="outlined"
                        sx={{
                            display: "flex",
                            width: "min(100%, 400px)",
                            marginLeft: "auto",
                            marginRight: "auto",
                            marginTop: 5,
                            marginBottom: 2,
                            whiteSpace: "nowrap",
                        }}
                    >{t("LogOut")}</Button>
                    <Button
                        onClick={openDelete}
                        startIcon={<DeleteIcon />}
                        variant="text"
                        sx={{
                            background: palette.error.main,
                            color: palette.error.contrastText,
                            display: "flex",
                            width: "min(100%, 400px)",
                            marginLeft: "auto",
                            marginRight: "auto",
                            marginBottom: 2,
                            whiteSpace: "nowrap",
                        }}
                    >{t("DeleteAccount")}</Button>
                </Box>
            </Stack>
        </>
    );
};
