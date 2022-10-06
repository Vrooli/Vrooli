import { Box, Button, Grid, Stack, TextField, Typography, useTheme } from "@mui/material"
import { useMutation } from "@apollo/client";
import { useCallback } from "react";
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
import { PasswordTextField, SnackSeverity } from "components";
import { logOut } from "graphql/generated/logOut";
import { profileEmailUpdate, profileEmailUpdateVariables } from "graphql/generated/profileEmailUpdate";
import { EmailIcon, LogOutIcon, WalletIcon } from "@shared/icons";

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
}: SettingsAuthenticationProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const [logOut] = useMutation<logOut, any>(logOutMutation);
    const onLogOut = useCallback(() => {
        mutationWrapper({ mutation: logOut })
        PubSub.get().publishSession({});
        setLocation(APP_LINKS.Home);
    }, [logOut, setLocation]);

    const updateWallets = useCallback((updatedList: Wallet[]) => {
        if (!profile) {
            PubSub.get().publishSnack({ message: 'Profile not loaded.', severity: SnackSeverity.Error });
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
            PubSub.get().publishSnack({ message: 'Profile not loaded.', severity: SnackSeverity.Error });
            return;
        }
        onUpdated({
            ...profile,
            emails: updatedList,
        });
    }, [onUpdated, profile]);
    const numVerifiedWallets = profile?.wallets?.filter((wallet) => wallet.verified)?.length ?? 0;

    // Handle update
    const [mutation] = useMutation<profileEmailUpdate, profileEmailUpdateVariables>(profileEmailUpdateMutation);
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
                PubSub.get().publishSnack({ message: 'Could not find existing data.', severity: SnackSeverity.Error });
                return;
            }
            if (!formik.isValid) return;
            mutationWrapper({
                mutation,
                input: {
                    currentPassword: values.currentPassword,
                    newPassword: values.newPassword,
                },
                onSuccess: (data) => { onUpdated(data.profileEmailUpdate) },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    return (
        <Box style={{ overflow: 'hidden' }}>
            {/* Title */}
            <Box sx={{
                background: palette.primary.dark,
                color: palette.primary.contrastText,
                padding: 0.5,
                marginBottom: 2,
                display: 'flex',
                justifyContent: 'center',
                alignItems: 'center',
            }}>
                <Typography component="h1" variant="h4" textAlign="center">Authentication</Typography>
                <HelpButton markdown={helpText} />
            </Box>
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
        </Box>
    )
}