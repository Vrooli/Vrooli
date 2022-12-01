import { Box, Button, Grid, Stack, TextField, Typography, useTheme } from "@mui/material"
import { useMutation } from "@apollo/client";
import { useCallback, useState } from "react";
import { mutationWrapper } from 'graphql/utils/graphqlWrapper';
import { profileUpdateSchema as validationSchema } from '@shared/validation';
import { APP_LINKS } from '@shared/consts';
import { useFormik } from 'formik';
import { profileEmailUpdateMutation } from "graphql/mutation";
import { PubSub, usePromptBeforeUnload } from "utils";
import { SettingsAuthenticationProps } from "../types";
import { useLocation } from '@shared/route';
import { logOutMutation } from 'graphql/mutation';
import { GridSubmitButtons, HelpButton } from "components/buttons";
import { EmailList, WalletList } from "components/lists";
import { Email, Wallet } from "types";
import { DeleteAccountDialog, PageTitle, PasswordTextField, SnackSeverity } from "components";
import { profileEmailUpdateVariables, profileEmailUpdate_profileEmailUpdate } from "graphql/generated/profileEmailUpdate";
import { DeleteIcon, EmailIcon, LogOutIcon, WalletIcon } from "@shared/icons";
import { getCurrentUser, guestSession } from "utils/authentication";
import { logOutVariables, logOut_logOut } from "graphql/generated/logOut";
import { SettingsFormData } from "pages";

const helpText =
    `This page allows you to manage your wallets, emails, and other authentication settings.`;

const walletHelpText =
    `This list contains all of your connected wallets. If a custom name has not been set, 
the wallet's reward address will be displayed.

You may add or remove as many wallets as you wish, but you must keep at least one *verified* authentication method (either a wallet or email address).`

const emailHelpText =
    `This list contains all of your connected email addresses.

You may add or remove as many email addresses as you wish, but you must keep at least one *verified* authentication method (either a wallet or email address).`

const passwordHelpText =
    `Change the password you use for email log ins. Wallet log ins use your wallet's extension, so no need to set a password here.`

export const SettingsAuthentication = ({
    profile,
    onUpdated,
    session,
}: SettingsAuthenticationProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const [logOut] = useMutation(logOutMutation);
    const onLogOut = useCallback(() => {
        const { id } = getCurrentUser(session);
        mutationWrapper<logOut_logOut, logOutVariables>({ 
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
            PubSub.get().publishSnack({ messageKey: 'CouldNotReadProfile', severity: SnackSeverity.Error });
            return;
        }
        onUpdated({
            ...profile,
            wallets: updatedList,
        });
    }, [onUpdated, profile]);
    const numVerifiedEmails = profile?.emails?.filter((email) => email.verified)?.length ?? 0;

    const updateEmails = useCallback((updatedList: Email[]) => {
        if (!profile) {
            PubSub.get().publishSnack({ messageKey: 'CouldNotReadProfile', severity: SnackSeverity.Error });
            return;
        }
        onUpdated({
            ...profile,
            emails: updatedList,
        });
    }, [onUpdated, profile]);
    const numVerifiedWallets = profile?.wallets?.filter((wallet) => wallet.verified)?.length ?? 0;

    // Handle update
    const [mutation] = useMutation(profileEmailUpdateMutation);
    const formik = useFormik({
        initialValues: {
            currentPassword: '',
            newPassword: '',
            newPasswordConfirmation: '',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            if (!profile) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadProfile', severity: SnackSeverity.Error });
                return;
            }
            if (!formik.isValid) return;
            mutationWrapper<profileEmailUpdate_profileEmailUpdate, profileEmailUpdateVariables>({
                mutation,
                input: {
                    currentPassword: values.currentPassword,
                    newPassword: values.newPassword,
                },
                onSuccess: (data) => { onUpdated(data) },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    const [deleteOpen, setDeleteOpen] = useState<boolean>(false);
    const openDelete = useCallback(() => setDeleteOpen(true), [setDeleteOpen]);
    const closeDelete = useCallback(() => setDeleteOpen(false), [setDeleteOpen]);

    return (
        <Box style={{ overflow: 'hidden' }}>
            {/* Delete account confirmation dialog */}
            <DeleteAccountDialog
                isOpen={deleteOpen}
                handleClose={closeDelete}
                session={session}
                zIndex={100}
            />
            <PageTitle title="Authentication" helpText={helpText} />
            <Stack direction="row" marginRight="auto" alignItems="center" justifyContent="center">
                <WalletIcon fill={palette.background.textPrimary} />
                <Typography component="h2" variant="h5" textAlign="center" ml={1}>Connected Wallets</Typography>
                <HelpButton markdown={walletHelpText} />
            </Stack>
            <WalletList
                handleUpdate={updateWallets}
                list={profile?.wallets ?? []}
                numVerifiedEmails={numVerifiedEmails}
            />
            <Stack direction="row" marginRight="auto" alignItems="center" justifyContent="center">
                <EmailIcon fill={palette.background.textPrimary} />
                <Typography component="h2" variant="h5" textAlign="center" ml={1}>Connected Emails</Typography>
                <HelpButton markdown={emailHelpText} />
            </Stack>
            <EmailList
                handleUpdate={updateEmails}
                list={profile?.emails ?? []}
                numVerifiedWallets={numVerifiedWallets}
            />
            <Stack direction="row" marginRight="auto" alignItems="center" justifyContent="center">
                <Typography component="h2" variant="h5" textAlign="center">Change Password</Typography>
                <HelpButton markdown={passwordHelpText} />
            </Stack>
            <form onSubmit={formik.handleSubmit} style={{ margin: 8, paddingBottom: 16 }}>
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
                    <GridSubmitButtons
                        disabledCancel={!formik.dirty}
                        disabledSubmit={!formik.dirty}
                        errors={formik.errors}
                        isCreate={false}
                        loading={formik.isSubmitting}
                        onCancel={formik.resetForm}
                        onSetSubmitting={formik.setSubmitting}
                        onSubmit={formik.handleSubmit}
                    />
                </Grid>
            </form>
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
            >Log Out</Button>
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
            >Delete Account</Button>
        </Box>
    )
}

export const settingsAuthenticationFormData: SettingsFormData = {
    labels: ['Authentication', 'Authentication Update', 'Update Authentication'],
    items: [
        { id: 'wallet-list', labels: ['Connected Wallets', 'Wallet List', 'Wallets List'] },
        { id: 'add-wallet-button', labels: ['Add Wallet', 'Wallet Add', 'Connect Wallet', 'Wallet Connect', 'New Wallet', 'Wallet New'] },
        { id: 'email-list', labels: ['Connected Emails', 'Email List', 'Emails List'] },
        { id: 'emailAddress', labels: ['Add Email', 'Email Add', 'Connect Email', 'Email Connect', 'New Email', 'Email New'] },
    ],
}