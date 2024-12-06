import { Email, endpointPostAuthLogout, endpointPostAuthLogoutAll, endpointPutProfileEmail, LINKS, Phone, profileEmailUpdateFormValidation, ProfileEmailUpdateInput, Session, User, Wallet } from "@local/shared";
import { Box, Button, Divider, Stack, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api/fetchWrapper";
import { SocketService } from "api/socket";
import { LazyRequestWithResult } from "api/types";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { DeleteAccountDialog } from "components/dialogs/DeleteAccountDialog/DeleteAccountDialog";
import { PasswordTextInput } from "components/inputs/PasswordTextInput/PasswordTextInput";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { EmailList } from "components/lists/devices/EmailList/EmailList";
import { PhoneList } from "components/lists/devices/PhoneList/PhoneList";
import { WalletList } from "components/lists/devices/WalletList/WalletList";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsContent, SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { Title } from "components/text/Title/Title";
import { Formik, FormikHelpers } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useProfileQuery } from "hooks/useProfileQuery";
import { DeleteIcon, EmailIcon, LogOutIcon, PhoneIcon, WalletIcon } from "icons";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { FormSection, ScrollBox } from "styles";
import { guestSession } from "utils/authentication/session";
import { removeCookie } from "utils/localStorage";
import { PubSub, SIDE_MENU_ID } from "utils/pubsub";
import { SettingsAuthenticationFormProps, SettingsAuthenticationFormValues, SettingsAuthenticationViewProps } from "../types";

const initialValues: SettingsAuthenticationFormValues = {
    currentPassword: "",
    newPassword: "",
    newPasswordConfirmation: "",
};

const hiddenInputStyle = { display: "none" } as const;

function SettingsAuthenticationForm({
    display,
    dirty,
    isLoading,
    onCancel,
    values,
    ...props
}: SettingsAuthenticationFormProps) {
    const { t } = useTranslation();
    console.log("settingsauthenticationform render", props.errors, values);

    return (
        <>
            <BaseForm
                display={display}
                isLoading={isLoading}
                style={{ paddingBottom: 0 }}
            >
                {/* Hidden username input because some password managers require it */}
                <TextInput
                    name="username"
                    autoComplete="username"
                    sx={hiddenInputStyle}
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
}

const buttonStyles = {
    display: "flex",
    width: "min(100%, 400px)",
    margin: "8px auto",
    whiteSpace: "nowrap",
};

export function SettingsAuthenticationView({
    display,
    onClose,
}: SettingsAuthenticationViewProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery();

    const [logOut] = useLazyFetch<undefined, Session>(endpointPostAuthLogout);
    const [logOutAllDevices] = useLazyFetch<undefined, Session>(endpointPostAuthLogoutAll);
    const handleLogout = useCallback((logoutEndpoint: LazyRequestWithResult<undefined, Session>) => {
        SocketService.get().disconnect();
        fetchLazyWrapper<undefined, Session>({
            fetch: logoutEndpoint,
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
                setLocation(LINKS.Home);
            },
        });
    }, [setLocation]);
    const onLogOut = useCallback(() => handleLogout(logOut), [handleLogout, logOut]);
    const onLogOutAllDevices = useCallback(() => handleLogout(logOutAllDevices), [handleLogout, logOutAllDevices]);

    const updateProfileField = useCallback((field: "phones" | "emails" | "wallets", updatedList: unknown[]) => {
        if (!profile) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadProfile", severity: "Error" });
            return;
        }
        onProfileUpdate({ ...profile, [field]: updatedList });
    }, [onProfileUpdate, profile]);
    const updatePhones = useCallback((updatedList: Phone[]) => updateProfileField("phones", updatedList), [updateProfileField]);
    const updateEmails = useCallback((updatedList: Email[]) => updateProfileField("emails", updatedList), [updateProfileField]);
    const updateWallets = useCallback((updatedList: Wallet[]) => updateProfileField("wallets", updatedList), [updateProfileField]);

    const numVerifiedPhones = profile?.phones?.filter((phone) => phone.verified)?.length ?? 0;
    const numVerifiedEmails = profile?.emails?.filter((email) => email.verified)?.length ?? 0;
    const numVerifiedWallets = profile?.wallets?.filter((wallet) => wallet.verified)?.length ?? 0;

    // Handle update
    const [update, { loading: isUpdating }] = useLazyFetch<ProfileEmailUpdateInput, User>(endpointPutProfileEmail);

    const [deleteOpen, setDeleteOpen] = useState<boolean>(false);
    const openDelete = useCallback(() => setDeleteOpen(true), [setDeleteOpen]);
    const closeDelete = useCallback(() => setDeleteOpen(false), [setDeleteOpen]);


    const onSubmit = useCallback(function onSubmitCallback(values: SettingsAuthenticationFormValues, helpers: FormikHelpers<SettingsAuthenticationFormValues>) {
        if (!profile) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadProfile", severity: "Error" });
            return;
        }
        if (typeof values.newPassword === "string" && values.newPassword.length > 0 && values.newPassword !== values.newPasswordConfirmation) {
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
    }, [onProfileUpdate, profile, update]);

    return (
        <ScrollBox>
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
            <SettingsContent>
                <SettingsList />
                <Stack
                    direction="column"
                    spacing={6}
                    m="auto"
                    pl={2}
                    pr={2}
                    sx={{
                        maxWidth: "min(100%, 700px)",
                        width: "100%",
                    }}
                >
                    <Box>
                        <Title
                            help={t("PhoneListHelp")}
                            Icon={PhoneIcon}
                            title={t("Phone", { count: 2 })}
                            variant="subheader"
                        />
                        <PhoneList
                            handleUpdate={updatePhones}
                            list={profile?.phones ?? []}
                            numOtherVerified={numVerifiedEmails + numVerifiedWallets}
                        />
                    </Box>
                    <Divider />
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
                            numOtherVerified={numVerifiedPhones + numVerifiedWallets}
                        />
                    </Box>
                    <Divider />
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
                            numOtherVerified={numVerifiedPhones + numVerifiedEmails}
                        />
                    </Box>
                    <Divider />
                    <Box>
                        <Title
                            help={t("PasswordChangeHelp")}
                            title={t("ChangePassword")}
                            variant="subheader"
                        />
                        <Formik
                            enableReinitialize={true}
                            initialValues={initialValues}
                            onSubmit={onSubmit}
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
                    <Divider />
                    <Box>
                        <Button
                            color="secondary"
                            onClick={onLogOut}
                            startIcon={<LogOutIcon />}
                            variant="outlined"
                            sx={buttonStyles}
                        >{t("LogOut")}</Button>
                        <Button
                            color="secondary"
                            onClick={onLogOutAllDevices}
                            startIcon={<LogOutIcon />}
                            variant="outlined"
                            sx={buttonStyles}
                        >
                            {t("LogOutAllDevices")}
                        </Button>
                        <Button
                            onClick={openDelete}
                            startIcon={<DeleteIcon />}
                            variant="text"
                            sx={{
                                ...buttonStyles,
                                background: palette.error.main,
                                color: palette.error.contrastText,
                            }}
                        >{t("DeleteAccount")}</Button>
                    </Box>
                </Stack>
            </SettingsContent>
        </ScrollBox>
    );
}
