import { Email, endpointPostAuthLogout, endpointPutProfileEmail, LINKS, LogOutInput, profileEmailUpdateFormValidation, ProfileEmailUpdateInput, Session, User, Wallet } from "@local/shared";
import { Box, Button, Stack, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { DeleteAccountDialog } from "components/dialogs/DeleteAccountDialog/DeleteAccountDialog";
import { PasswordTextInput } from "components/inputs/PasswordTextInput/PasswordTextInput";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { EmailList, WalletList } from "components/lists/devices";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { Title } from "components/text/Title/Title";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useProfileQuery } from "hooks/useProfileQuery";
import { DeleteIcon, EmailIcon, LogOutIcon, WalletIcon } from "icons";
import { useCallback, useContext, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { FormSection, pagePaddingBottom } from "styles";
import { getCurrentUser, guestSession } from "utils/authentication/session";
import { Cookies } from "utils/cookies";
import { PubSub } from "utils/pubsub";
import { SettingsAuthenticationFormProps, SettingsAuthenticationViewProps } from "../types";

const SettingsAuthenticationForm = ({
    display,
    dirty,
    isLoading,
    onCancel,
    values,
    ...props
}: SettingsAuthenticationFormProps) => {
    const { t } = useTranslation();
    console.log("settingsauthenticationform render", props.errors, values);

    return (
        <>
            <BaseForm
                display={display}
                isLoading={isLoading}
            >
                {/* Hidden username input because some password managers require it */}
                <TextInput
                    name="username"
                    autoComplete="username"
                    sx={{ display: "none" }}
                />
                <FormSection>
                    <PasswordTextInput
                        fullWidth
                        name="currentPassword"
                        label={t("PasswordCurrent")}
                        autoComplete="current-password"
                    />
                    <PasswordTextInput
                        fullWidth
                        name="newPassword"
                        label={t("PasswordNew")}
                        autoComplete="new-password"
                    />
                    <PasswordTextInput
                        fullWidth
                        name="newPasswordConfirmation"
                        autoComplete="new-password"
                        label={t("PasswordNewConfirm")}
                    />
                </FormSection>
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={props.errors}
                isCreate={false}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </>
    );
};

export const SettingsAuthenticationView = ({
    display,
    onClose,
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
            onSuccess: (data) => {
                localStorage.removeItem(Cookies.FormData); // Clear old form data cache
                PubSub.get().publish("session", data);
                PubSub.get().publish("sideMenu", { id: "side-menu", isOpen: false });
            },
            // If error, log out anyway
            onError: () => { PubSub.get().publish("session", guestSession); },
        });
        PubSub.get().publish("session", guestSession);
        setLocation(LINKS.Home);
    }, [logOut, session, setLocation]);

    const updateWallets = useCallback((updatedList: Wallet[]) => {
        if (!profile) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadProfile", severity: "Error" });
            return;
        }
        onProfileUpdate({ ...profile, wallets: updatedList });
    }, [onProfileUpdate, profile]);
    const numVerifiedEmails = profile?.emails?.filter((email) => email.verified)?.length ?? 0;

    const updateEmails = useCallback((updatedList: Email[]) => {
        if (!profile) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadProfile", severity: "Error" });
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
            />
            <SettingsTopBar
                display={display}
                onClose={onClose}
                title={t("Authentication")}
            />
            <Stack direction="row" pt={2} sx={{ paddingBottom: pagePaddingBottom }}>
                <SettingsList />
                <Stack direction="column" spacing={8} m="auto" pl={2} pr={2} sx={{ maxWidth: "min(100%, 500px)" }}>
                    <Box>
                        <Title
                            help={t("WalletListHelp")}
                            Icon={WalletIcon}
                            title={t("Wallet", { count: 2 })}
                            variant="subheader"
                        />
                        <WalletList
                            handleUpdate={updateWallets}
                            list={profile?.wallets ?? []}
                            numVerifiedEmails={numVerifiedEmails}
                        />
                    </Box>
                    <Box>
                        <Title
                            help={t("EmailListHelp")}
                            Icon={EmailIcon}
                            title={t("Email", { count: 2 })}
                            variant="subheader"
                        />
                        <EmailList
                            handleUpdate={updateEmails}
                            list={profile?.emails ?? []}
                            numVerifiedWallets={numVerifiedWallets}
                        />
                    </Box>
                    <Box>
                        <Title
                            help={t("PasswordChangeHelp")}
                            title={t("ChangePassword")}
                            variant="subheader"
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
                                    PubSub.get().publish("snack", { messageKey: "CouldNotReadProfile", severity: "Error" });
                                    return;
                                }
                                if (typeof values.newPassword === "string" && values.newPassword.length > 0 && values.newPassword !== (values as any).newPasswordConfirmation) {
                                    PubSub.get().publish("snack", { messageKey: "PasswordsDontMatch", severity: "Error" });
                                    helpers.setSubmitting(false);
                                    return;
                                }
                                fetchLazyWrapper<ProfileEmailUpdateInput, User>({
                                    fetch: update,
                                    inputs: {
                                        currentPassword: values.currentPassword,
                                        newPassword: values.newPassword,
                                    },
                                    onSuccess: (data) => { onProfileUpdate(data); },
                                    onCompleted: () => { helpers.setSubmitting(false); },
                                    successMessage: () => ({ messageKey: "Success" }),
                                });
                            }}
                            validationSchema={profileEmailUpdateFormValidation}
                        >
                            {(formik) => <SettingsAuthenticationForm
                                display={display}
                                isLoading={isProfileLoading || isUpdating}
                                onCancel={formik.resetForm}
                                {...formik}
                            />}
                        </Formik>
                    </Box>
                    <Box>
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
            </Stack>
        </>
    );
};
