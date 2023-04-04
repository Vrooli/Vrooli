import { Box, Button, Stack, useTheme } from "@mui/material";
import { Email, LINKS, LogOutInput, ProfileEmailUpdateInput, Session, User, Wallet } from '@shared/consts';
import { DeleteIcon, EmailIcon, LogOutIcon, WalletIcon } from "@shared/icons";
import { useLocation } from '@shared/route';
import { profileEmailUpdateValidation } from "@shared/validation";
import { authLogOut } from "api/generated/endpoints/auth_logOut";
import { userProfileEmailUpdate } from "api/generated/endpoints/user_profileEmailUpdate";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { DeleteAccountDialog } from "components/dialogs/DeleteAccountDialog/DeleteAccountDialog";
import { EmailList, WalletList } from "components/lists/devices";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { Subheader } from "components/text/Subheader/Subheader";
import { Formik } from 'formik';
import { SettingsAuthenticationForm } from "forms/settings";
import { useCallback, useContext, useState } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser, guestSession } from "utils/authentication/session";
import { useProfileQuery } from "utils/hooks/useProfileQuery";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { SettingsAuthenticationViewProps } from "../types";

export const SettingsAuthenticationView = ({
    display = 'page',
}: SettingsAuthenticationViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery();

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

    const [deleteOpen, setDeleteOpen] = useState<boolean>(false);
    const openDelete = useCallback(() => setDeleteOpen(true), [setDeleteOpen]);
    const closeDelete = useCallback(() => setDeleteOpen(false), [setDeleteOpen]);

    return (
        <>
            {/* Delete account confirmation dialog */}
            <DeleteAccountDialog
                isOpen={deleteOpen}
                handleClose={closeDelete}
                zIndex={100}
            />
            <SettingsTopBar
                display={display}
                onClose={() => { }}
                titleData={{
                    titleKey: 'Authentication',
                }}
            />
            <Stack direction="row">
                <SettingsList />
                <Box>
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
                    <Formik
                        enableReinitialize={true}
                        initialValues={{
                            currentPassword: '',
                            newPassword: '',
                            newPasswordConfirmation: '',
                        } as ProfileEmailUpdateInput}
                        onSubmit={(values, helpers) => {
                            if (!profile) {
                                PubSub.get().publishSnack({ messageKey: 'CouldNotReadProfile', severity: 'Error' });
                                return;
                            }
                            mutationWrapper<User, ProfileEmailUpdateInput>({
                                mutation,
                                input: {
                                    currentPassword: values.currentPassword,
                                    newPassword: values.newPassword,
                                },
                                onSuccess: (data) => { onProfileUpdate(data) },
                                onError: () => { helpers.setSubmitting(false) },
                            })
                        }}
                        validationSchema={profileEmailUpdateValidation.update({})}
                    >
                        {(formik) => <SettingsAuthenticationForm
                            display={display}
                            isLoading={isProfileLoading || isUpdating}
                            onCancel={formik.resetForm}
                            {...formik}
                        />}
                    </Formik>
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