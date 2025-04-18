import { Email, endpointsAuth, endpointsUser, Phone, profileEmailUpdateFormValidation, ProfileEmailUpdateInput, Session, User, Wallet } from "@local/shared";
import { Box, Button, Divider, Stack, useTheme } from "@mui/material";
import { Formik, FormikHelpers } from "formik";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { SocketService } from "../../api/socket.js";
import { LazyRequestWithResult } from "../../api/types.js";
import { BottomActionsButtons } from "../../components/buttons/BottomActionsButtons.js";
import { DeleteAccountDialog } from "../../components/dialogs/DeleteAccountDialog/DeleteAccountDialog.js";
import { PasswordTextInput } from "../../components/inputs/PasswordTextInput/PasswordTextInput.js";
import { TextInput } from "../../components/inputs/TextInput/TextInput.js";
import { EmailList } from "../../components/lists/devices/EmailList.js";
import { PhoneList } from "../../components/lists/devices/PhoneList.js";
import { WalletList } from "../../components/lists/devices/WalletList.js";
import { SettingsList } from "../../components/lists/SettingsList/SettingsList.js";
import { Navbar } from "../../components/navigation/Navbar.js";
import { SettingsContent } from "../../components/navigation/SettingsTopBar.js";
import { Title } from "../../components/text/Title.js";
import { BaseForm } from "../../forms/BaseForm/BaseForm.js";
import { useLazyFetch } from "../../hooks/useLazyFetch.js";
import { useProfileQuery } from "../../hooks/useProfileQuery.js";
import { IconCommon } from "../../icons/Icons.js";
import { useLocation } from "../../route/router.js";
import { FormSection, ScrollBox } from "../../styles.js";
import { guestSession, SessionService } from "../../utils/authentication/session.js";
import { PubSub } from "../../utils/pubsub.js";
import { SettingsAuthenticationFormProps, SettingsAuthenticationFormValues, SettingsAuthenticationViewProps } from "./types.js";

const initialValues: SettingsAuthenticationFormValues = {
    currentPassword: "",
    newPassword: "",
    newPasswordConfirmation: "",
};

const hiddenInputStyle = { display: "none" } as const;
const phoneIconInfo = { name: "Phone", type: "Common" } as const;
const emailIconInfo = { name: "Email", type: "Common" } as const;
const walletIconInfo = { name: "Wallet", type: "Common" } as const;

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

    const [logOut] = useLazyFetch<undefined, Session>(endpointsAuth.logout);
    const [logOutAllDevices] = useLazyFetch<undefined, Session>(endpointsAuth.logoutAll);
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
                SessionService.logOut();
            },
        });
    }, [setLocation]);
    const onLogOut = useCallback(() => handleLogout(logOut), [handleLogout, logOut]);
    const onLogOutAllDevices = useCallback(() => handleLogout(logOutAllDevices), [handleLogout, logOutAllDevices]);

    const updateProfileField = useCallback((field: "phones" | "emails" | "wallets", updatedList: unknown[]) => {
        if (!profile) {
            PubSub.get().publish("snack", { message: t("CouldNotReadProfile", { ns: "error" }), severity: "Error" });
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
    const [update, { loading: isUpdating }] = useLazyFetch<ProfileEmailUpdateInput, User>(endpointsUser.profileEmailUpdate);

    const [deleteOpen, setDeleteOpen] = useState<boolean>(false);
    const openDelete = useCallback(() => setDeleteOpen(true), [setDeleteOpen]);
    const closeDelete = useCallback(() => setDeleteOpen(false), [setDeleteOpen]);


    const onSubmit = useCallback(function onSubmitCallback(values: SettingsAuthenticationFormValues, helpers: FormikHelpers<SettingsAuthenticationFormValues>) {
        if (!profile) {
            PubSub.get().publish("snack", { message: t("CouldNotReadProfile", { ns: "error" }), severity: "Error" });
            return;
        }
        if (typeof values.newPassword === "string" && values.newPassword.length > 0 && values.newPassword !== values.newPasswordConfirmation) {
            PubSub.get().publish("snack", { message: t("PasswordsDontMatch", { ns: "error" }), severity: "Error" });
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
            successMessage: () => ({ message: t("Success") }),
        });
    }, [onProfileUpdate, profile, update, t]);

    return (
        <ScrollBox>
            {/* Delete account confirmation dialog */}
            <DeleteAccountDialog
                isOpen={deleteOpen}
                handleClose={closeDelete}
            />
            <Navbar title={t("Authentication")} />
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
                            iconInfo={phoneIconInfo}
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
                            iconInfo={emailIconInfo}
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
                            iconInfo={walletIconInfo}
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
                            startIcon={<IconCommon name="LogOut" />}
                            variant="outlined"
                            sx={buttonStyles}
                        >{t("LogOut")}</Button>
                        <Button
                            color="secondary"
                            onClick={onLogOutAllDevices}
                            startIcon={<IconCommon name="LogOut" />}
                            variant="outlined"
                            sx={buttonStyles}
                        >
                            {t("LogOutAllDevices")}
                        </Button>
                        <Button
                            onClick={openDelete}
                            startIcon={<IconCommon name="Delete" />}
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
