import { Box, Button, Grid, Stack, TextField, Typography, useTheme } from "@mui/material"
import { useCustomMutation } from "api/hooks";
import { useCallback, useState } from "react";
import { mutationWrapper } from 'api/utils';
import { APP_LINKS, Email, LogOutInput, ProfileEmailUpdateInput, Session, User, Wallet } from '@shared/consts';
import { useFormik } from 'formik';
import { PubSub, useProfileQuery, usePromptBeforeUnload } from "utils";
import { SettingsAuthenticationViewProps } from "../types";
import { useLocation } from '@shared/route';
import { GridSubmitButtons, HelpButton } from "components/buttons";
import { EmailList, SettingsList, WalletList } from "components/lists";
import { DeleteAccountDialog, PasswordTextField, SettingsTopBar } from "components";
import { DeleteIcon, EmailIcon, LogOutIcon, WalletIcon } from "@shared/icons";
import { getCurrentUser, guestSession } from "utils/authentication";
import { userValidation } from "@shared/validation";
import { authLogOut } from "api/generated/endpoints/auth_logOut";
import { userProfileEmailUpdate } from "api/generated/endpoints/user_profileEmailUpdate";
import { BaseForm } from "forms";
import { useTranslation } from "react-i18next";

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
        setLocation(APP_LINKS.Home);
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
                    <Stack direction="row" marginRight="auto" alignItems="center" justifyContent="center">
                        <WalletIcon fill={palette.background.textPrimary} />
                        <Typography component="h2" variant="h5" textAlign="center" ml={1}>{t('Wallet', { count: 2 })}</Typography>
                        <HelpButton markdown={t('WalletListHelp')} />
                    </Stack>
                    <WalletList
                        handleUpdate={updateWallets}
                        list={profile?.wallets ?? []}
                        numVerifiedEmails={numVerifiedEmails}
                    />
                    <Stack direction="row" marginRight="auto" alignItems="center" justifyContent="center">
                        <EmailIcon fill={palette.background.textPrimary} />
                        <Typography component="h2" variant="h5" textAlign="center" ml={1}>{t('Email', { count: 2 })}</Typography>
                        <HelpButton markdown={t('EmailListHelp')} />
                    </Stack>
                    <EmailList
                        handleUpdate={updateEmails}
                        list={profile?.emails ?? []}
                        numVerifiedWallets={numVerifiedWallets}
                    />
                    <Stack direction="row" marginRight="auto" alignItems="center" justifyContent="center">
                        <Typography component="h2" variant="h5" textAlign="center">{t('ChangePassword')}</Typography>
                        <HelpButton markdown={t('PasswordChangeHelp')} />
                    </Stack>
                    <BaseForm
                        isLoading={isProfileLoading || isUpdating}
                        onSubmit={formik.handleSubmit}
                        sx={{ margin: 8, paddingBottom: 16 }}
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