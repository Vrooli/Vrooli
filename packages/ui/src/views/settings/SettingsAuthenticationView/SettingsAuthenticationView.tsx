import { Box, Button, Grid, Stack, TextField, useTheme } from "@mui/material";
import { Email, LINKS, LogOutInput, ProfileEmailUpdateInput, Session, User, Wallet } from '@shared/consts';
import { DeleteIcon, EmailIcon, LogOutIcon, WalletIcon } from "@shared/icons";
import { useLocation } from '@shared/route';
import { userValidation } from "@shared/validation";
import { authLogOut } from "api/generated/endpoints/auth_logOut";
import { userProfileEmailUpdate } from "api/generated/endpoints/user_profileEmailUpdate";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { DeleteAccountDialog } from "components/dialogs/DeleteAccountDialog/DeleteAccountDialog";
import { PasswordTextField } from "components/inputs/PasswordTextField/PasswordTextField";
import { EmailList, WalletList } from "components/lists/devices";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { Subheader } from "components/text/Subheader/Subheader";
import { useFormik } from 'formik';
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser, guestSession } from "utils/authentication/session";
import { useProfileQuery } from "utils/hooks/useProfileQuery";
import { usePromptBeforeUnload } from "utils/hooks/usePromptBeforeUnload";
import { PubSub } from "utils/pubsub";
import { SettingsAuthenticationViewProps } from "../types";

export const SettingsAuthenticationView = ({
    display = 'page',
    session,
}: SettingsAuthenticationViewProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery(session);

    const [logOut] = useCustomMutation<Session, LogOutInput>(authLogOut);
    const onLogOut = useCallback(() => {
        const { id } = getCurrentUser(session);
        mutationWrapper<Session, LogOutInput>({
            mutation: logOut,
            input: { id },
            onSuccess: (data) => { PubSub.get().publishSession(data) },
            // If error, log out anyway
            onError: () => { PubSub.get().publishSession(guestSession) },
        })
        PubSub.get().publishSession(guestSession);
        setLocation(LINKS.Home);
    }, [logOut, session, setLocation]);

    const updateWallets = useCallback((updatedList: Wallet[]) => {
        if (!profile) {
            PubSub.get().publishSnack({ messageKey: 'CouldNotReadProfile', severity: 'Error' });
            return;
        }
        onProfileUpdate({ ...profile, wallets: updatedList });
    }, [onProfileUpdate, profile]);
    const numVerifiedEmails = profile?.emails?.filter((email) => email.verified)?.length ?? 0;

    const updateEmails = useCallback((updatedList: Email[]) => {
        if (!profile) {
            PubSub.get().publishSnack({ messageKey: 'CouldNotReadProfile', severity: 'Error' });
            return;
        }
        onProfileUpdate({ ...profile, emails: updatedList });
    }, [onProfileUpdate, profile]);
    const numVerifiedWallets = profile?.wallets?.filter((wallet) => wallet.verified)?.length ?? 0;

    // Handle update
    const [mutation, { loading: isUpdating }] = useCustomMutation<User, ProfileEmailUpdateInput>(userProfileEmailUpdate);
    const formik = useFormik({
        initialValues: {
            currentPassword: '',
            newPassword: '',
            newPasswordConfirmation: '',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: userValidation.update({}),
        onSubmit: (values) => {
            if (!profile) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadProfile', severity: 'Error' });
                return;
            }
            if (!formik.isValid) return;
            mutationWrapper<User, ProfileEmailUpdateInput>({
                mutation,
                input: {
                    currentPassword: values.currentPassword,
                    newPassword: values.newPassword,
                },
                onSuccess: (data) => { onProfileUpdate(data) },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    const [deleteOpen, setDeleteOpen] = useState<boolean>(false);
    const openDelete = useCallback(() => setDeleteOpen(true), [setDeleteOpen]);
    const closeDelete = useCallback(() => setDeleteOpen(false), [setDeleteOpen]);

    return (
        <>
            {/* Delete account confirmation dialog */}
            <DeleteAccountDialog
                isOpen={deleteOpen}
                handleClose={closeDelete}
                session={session}
                zIndex={100}
            />
            <SettingsTopBar
                display={display}
                onClose={() => { }}
                session={session}
                titleData={{
                    titleKey: 'Authentication',
                }}
            />
            <Stack direction="row">
                <SettingsList />
                <Box style={{ overflow: 'hidden' }}>
                    <Subheader
                        help={t('WalletListHelp')}
                        Icon={WalletIcon}
                        title={t('Wallet', { count: 2 })} />
                    <WalletList
                        handleUpdate={updateWallets}
                        list={profile?.wallets ?? []}
                        numVerifiedEmails={numVerifiedEmails}
                    />
                    <Subheader
                        help={t('EmailListHelp')}
                        Icon={EmailIcon}
                        title={t('Email', { count: 2 })} />
                    <EmailList
                        handleUpdate={updateEmails}
                        list={profile?.emails ?? []}
                        numVerifiedWallets={numVerifiedWallets}
                    />
                    <Subheader
                        help={t('PasswordChangeHelp')}
                        title={t('ChangePassword')} />
                    <BaseForm
                        isLoading={isProfileLoading || isUpdating}
                        onSubmit={formik.handleSubmit}
                        style={{ margin: 8, paddingBottom: 16 }}
                    >
                        {/* Hidden username input because some password managers require it */}
                        <TextField
                            name="username"
                            autoComplete="username"
                            sx={{ display: 'none' }}
                        />
                        <Grid container spacing={1}>
                            <Grid item xs={12}>
                                <PasswordTextField
                                    fullWidth
                                    id="currentPassword"
                                    name="currentPassword"
                                    label="Current Password"
                                    autoComplete="current-password"
                                    value={formik.values.currentPassword}
                                    onBlur={formik.handleBlur}
                                    onChange={formik.handleChange}
                                    error={formik.touched.currentPassword && Boolean(formik.errors.currentPassword)}
                                    helperText={formik.touched.currentPassword ? formik.errors.currentPassword : null}
                                />
                            </Grid>
                            <Grid item xs={12}>
                                <PasswordTextField
                                    fullWidth
                                    id="newPassword"
                                    name="newPassword"
                                    label="New Password"
                                    autoComplete="new-password"
                                    value={formik.values.newPassword}
                                    onBlur={formik.handleBlur}
                                    onChange={formik.handleChange}
                                    error={formik.touched.newPassword && Boolean(formik.errors.newPassword)}
                                    helperText={formik.touched.newPassword ? formik.errors.newPassword : null}
                                />
                            </Grid>
                            <Grid item xs={12}>
                                <PasswordTextField
                                    fullWidth
                                    id="newPasswordConfirmation"
                                    name="newPasswordConfirmation"
                                    autoComplete="new-password"
                                    label="Confirm New Password"
                                    value={formik.values.newPasswordConfirmation}
                                    onBlur={formik.handleBlur}
                                    onChange={formik.handleChange}
                                    error={formik.touched.newPasswordConfirmation && Boolean(formik.errors.newPasswordConfirmation)}
                                    helperText={formik.touched.newPasswordConfirmation ? formik.errors.newPasswordConfirmation : null}
                                />
                            </Grid>
                        </Grid>
                        <GridSubmitButtons
                            disabledCancel={!formik.dirty}
                            disabledSubmit={!formik.dirty}
                            display={display}
                            errors={formik.errors}
                            isCreate={false}
                            loading={formik.isSubmitting}
                            onCancel={formik.resetForm}
                            onSetSubmitting={formik.setSubmitting}
                            onSubmit={formik.handleSubmit}
                        />
                    </BaseForm>
                    <Button
                        color="secondary"
                        onClick={onLogOut}
                        startIcon={<LogOutIcon />}
                        sx={{
                            display: 'flex',
                            width: 'min(100%, 400px)',
                            marginLeft: 'auto',
                            marginRight: 'auto',
                            marginTop: 5,
                            marginBottom: 2,
                            whiteSpace: 'nowrap',
                        }}
                    >{t('LogOut')}</Button>
                    <Button
                        onClick={openDelete}
                        startIcon={<DeleteIcon />}
                        sx={{
                            background: palette.error.main,
                            color: palette.error.contrastText,
                            display: 'flex',
                            width: 'min(100%, 400px)',
                            marginLeft: 'auto',
                            marginRight: 'auto',
                            marginBottom: 2,
                            whiteSpace: 'nowrap',
                        }}
                    >{t('DeleteAccount')}</Button>
                </Box>
            </Stack>
        </>
    )
}