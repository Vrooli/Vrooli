import { Box, Button, Container, Grid, Stack, TextField, Typography } from "@mui/material"
import { useMutation } from "@apollo/client";
import { user } from "graphql/generated/user";
import { useCallback, useMemo } from "react";
import { mutationWrapper } from 'graphql/utils/wrappers';
import { APP_LINKS, profileUpdateSchema as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { profileUpdateMutation } from "graphql/mutation";
import { formatForUpdate, Pubs } from "utils";
import {
    Add as AddIcon,
    Restore as RevertIcon,
    Save as SaveIcon,
} from '@mui/icons-material';
import { DialogActionItem } from "components/containers/types";
import { DialogActionsContainer } from "components/containers/DialogActionsContainer/DialogActionsContainer";
import { SettingsAuthenticationProps } from "../types";
import { useLocation } from "wouter";
import { containerShadow } from "styles";
import { logOutMutation } from 'graphql/mutation';
import { HelpButton } from "components/buttons";
import { EmailList, WalletList } from "components/lists";
import { Email, Wallet } from "types";

const helpText =
    `This page allows you to manage your wallets, emails, and other authentication settings.`;

const walletHelpText =
    `This list contains all of your connected wallets. If a custom name has not been set, 
the wallet's reward address will be displayed.

You may add or remove as many wallets as you wish, but you must keep at least one *verified* authentication method (either a wallet or email address).`

const emailHelpText =
    `This list contains all of your connected email addresses.

You may add or remove as many email addresses as you wish, but you must keep at least one *verified* authentication method (either a wallet or email address).`

const TERTIARY_COLOR = '#95f3cd';

/**
 * Card styles
 */
const cardStyles = [
    { background: 'linear-gradient(38deg, rgba(65,96,233,1) 0%, rgba(235,54,106,1) 100%)', color: 'white' },
    { background: 'linear-gradient(38deg, rgba(65,233,177,1) 0%, rgba(54,131,235,1) 100%)', color: 'white' },
    { background: 'linear-gradient(38deg, rgba(233,186,65,1) 0%, rgba(172,54,235,1) 100%)', color: 'white' },
]

export const SettingsAuthentication = ({
    profile,
    onUpdated,
}: SettingsAuthenticationProps) => {
    const [, setLocation] = useLocation();

    const [logOut] = useMutation<any>(logOutMutation);
    const onLogOut = useCallback(() => {
        mutationWrapper({ mutation: logOut })
        PubSub.publish(Pubs.Session, {});
        setLocation(APP_LINKS.Home);
    }, []);

    const updateWallets = useCallback((updatedList: Wallet[]) => {
        onUpdated({
            ...profile,
            wallets: updatedList,
        });
    }, [onUpdated, profile]);
    const numVerifiedEmails = profile?.emails?.filter((email) => email.verified)?.length ?? 0;

    const updateEmails = useCallback((updatedList: Email[]) => {
        onUpdated({
            ...profile,
            emails: updatedList,
        });
    }, [onUpdated, profile]);
    const numVerifiedWallets = profile?.wallets?.filter((wallet) => wallet.verified)?.length ?? 0;

    // Handle update
    const [mutation] = useMutation<user>(profileUpdateMutation);
    const formik = useFormik({
        initialValues: {
            currentPassword: '',
            newPassword: '',
            newPasswordConfirmation: '',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            if (!formik.isValid) return;
            mutationWrapper({
                mutation,
                input: formatForUpdate(profile, { ...values }),
                onSuccess: (response) => { onUpdated(response.data.profileUpdate) },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });

    const actions: DialogActionItem[] = useMemo(() => [
        ['Save', SaveIcon, !formik.touched || formik.isSubmitting, true, () => { }],
        ['Cancel', RevertIcon, !formik.touched || formik.isSubmitting, false, () => { formik.resetForm() }],
    ], [formik, setLocation]);

    return (
        <Box sx={{ minHeight: '100vh', display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
            <Stack direction="column">
                <Box sx={{
                    ...containerShadow,
                    borderRadius: 2,
                    overflow: 'overlay',
                    marginTop: '-5vh',
                    background: (t) => t.palette.background.default,
                    width: 'min(100%, 700px)',
                }}>
                    <form onSubmit={formik.handleSubmit} style={{ overflow: 'hidden' }}>
                        {/* Title */}
                        <Box sx={{
                            background: (t) => t.palette.primary.dark,
                            color: (t) => t.palette.primary.contrastText,
                            padding: 0.5,
                            marginBottom: 2,
                            display: 'flex',
                            justifyContent: 'center',
                            alignItems: 'center',
                        }}>
                            <Typography component="h1" variant="h3" textAlign="center">Authentication</Typography>
                            <HelpButton markdown={helpText} sx={{ fill: TERTIARY_COLOR }} />
                        </Box>
                        <Stack direction="row" marginRight="auto" alignItems="center" justifyContent="center">
                            <Typography component="h2" variant="h5" textAlign="center">Connected Wallets</Typography>
                            <HelpButton markdown={walletHelpText} />
                        </Stack>
                        <WalletList
                            handleUpdate={updateWallets}
                            list={profile?.wallets ?? []}
                            numVerifiedEmails={numVerifiedEmails}
                        />
                        <Stack direction="row" marginRight="auto" alignItems="center" justifyContent="center">
                            <Typography component="h2" variant="h5" textAlign="center">Connected Emails</Typography>
                            <HelpButton markdown={emailHelpText} />
                        </Stack>
                        <EmailList
                            handleUpdate={updateEmails}
                            list={profile?.emails ?? []}
                            numVerifiedWallets={numVerifiedWallets}
                        />
                        <Container sx={{ paddingBottom: 2 }}>
                            <Grid container spacing={2}>
                                <Grid item xs={12}>
                                    <TextField
                                        fullWidth
                                        id="currentPassword"
                                        name="currentPassword"
                                        type="password"
                                        autoComplete="password"
                                        label="Current Password"
                                        value={formik.values.currentPassword}
                                        onBlur={formik.handleBlur}
                                        onChange={formik.handleChange}
                                        error={formik.touched.currentPassword && Boolean(formik.errors.currentPassword)}
                                        helperText={formik.touched.currentPassword && formik.errors.currentPassword}
                                    />
                                </Grid>
                                <Grid item xs={12}>
                                    <TextField
                                        fullWidth
                                        id="newPassword"
                                        name="newPassword"
                                        type="password"
                                        autoComplete="new-password"
                                        label="New Password"
                                        value={formik.values.newPassword}
                                        onBlur={formik.handleBlur}
                                        onChange={formik.handleChange}
                                        error={formik.touched.newPassword && Boolean(formik.errors.newPassword)}
                                        helperText={formik.touched.newPassword && formik.errors.newPassword}
                                    />
                                </Grid>
                                <Grid item xs={12}>
                                    <TextField
                                        fullWidth
                                        id="newPasswordConfirmation"
                                        name="newPasswordConfirmation"
                                        type="password"
                                        autoComplete="new-password"
                                        label="Confirm New Password"
                                        value={formik.values.newPasswordConfirmation}
                                        onBlur={formik.handleBlur}
                                        onChange={formik.handleChange}
                                        error={formik.touched.newPasswordConfirmation && Boolean(formik.errors.newPasswordConfirmation)}
                                        helperText={formik.touched.newPasswordConfirmation && formik.errors.newPasswordConfirmation}
                                    />
                                </Grid>
                            </Grid>
                        </Container>
                        <DialogActionsContainer fixed={false} actions={actions} />
                    </form>
                </Box>
                <Button color="secondary" onClick={onLogOut} sx={{
                    width: 'min(100%, 400px)',
                    margin: 'auto',
                    marginTop: 5,
                }}>Log Out</Button>
            </Stack>
        </Box>
    )
}