import {
    Box,
    Collapse,
    Divider,
    IconButton,
    List,
    ListItem,
    ListItemIcon,
    ListItemText,
    SwipeableDrawer,
    Typography,
    useTheme,
} from '@mui/material';
import { Stack } from '@mui/system';
import { CloseIcon, ExpandLessIcon, ExpandMoreIcon, HelpIcon, LogOutIcon, PlusIcon, SettingsIcon, UserIcon } from '@shared/icons';
import { AccountMenuProps } from '../types';
import { noSelect } from 'styles';
import { ThemeSwitch } from 'components/inputs';
import React, { useCallback, useMemo, useState } from 'react';
import { profileUpdateSchema as validationSchema } from '@shared/validation';
import { profileUpdateVariables, profileUpdate_profileUpdate } from 'graphql/generated/profileUpdate';
import { useMutation } from '@apollo/client';
import { PubSub, shapeProfileUpdate } from 'utils';
import { mutationWrapper } from 'graphql/utils';
import { useFormik } from 'formik';
import { logOutMutation, profileUpdateMutation, switchCurrentAccountMutation } from 'graphql/mutation';
import { APP_LINKS } from '@shared/consts';
import { useLocation } from '@shared/route';
import { getCurrentUser, guestSession } from 'utils/authentication';
import { SessionUser } from 'types';
import { logOutVariables, logOut_logOut } from 'graphql/generated/logOut';
import { switchCurrentAccountVariables, switchCurrentAccount_switchCurrentAccount } from 'graphql/generated/switchCurrentAccount';
import { ContactInfo } from 'components/navigation';

// Maximum accounts to sign in with
const MAX_ACCOUNTS = 10;

export const AccountMenu = ({
    anchorEl,
    onClose,
    session,
}: AccountMenuProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);
    const open = Boolean(anchorEl);

    // Additional resources collapse
    const [isAdditionalResourcesOpen, setIsAdditionalResourcesOpen] = useState(false);
    const toggleAdditionalResources = useCallback(() => { setIsAdditionalResourcesOpen(!isAdditionalResourcesOpen) }, [isAdditionalResourcesOpen]);
    const closeAdditionalResources = useCallback(() => { setIsAdditionalResourcesOpen(false) }, []);

    // Handle update. Only updates when menu closes, and account settings have changed.
    const [mutation] = useMutation(profileUpdateMutation);
    const formik = useFormik({
        initialValues: {
            theme: getCurrentUser(session).theme ?? 'light',
        },
        enableReinitialize: true,
        validationSchema,
        onSubmit: (values) => {
            // If not logged in, do nothing
            if (!userId) {
                return;
            }
            if (!formik.isValid) return;
            const input = shapeProfileUpdate({
                id: userId,
                theme: getCurrentUser(session).theme ?? 'light',
            }, {
                id: userId,
                theme: values.theme,
            })
            // If no changes, don't update
            if (!input || Object.keys(input).length === 0) {
                formik.setSubmitting(false);
                return;
            }
            mutationWrapper<profileUpdate_profileUpdate, profileUpdateVariables>({
                mutation,
                input,
                onSuccess: () => { formik.setSubmitting(false) },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });

    const handleClose = useCallback((event: React.MouseEvent<HTMLElement>) => {
        formik.handleSubmit();
        onClose(event);
        closeAdditionalResources();
    }, [closeAdditionalResources, formik, onClose]);

    const [switchCurrentAccount] = useMutation(switchCurrentAccountMutation);
    const handleUserClick = useCallback((event: React.MouseEvent<HTMLElement>, user: SessionUser) => {
        // Close menu
        handleClose(event);
        // If already selected, go to profile page
        if (userId === user.id) {
            setLocation(APP_LINKS.Profile);
        }
        // Otherwise, switch to selected account
        else {
            mutationWrapper<switchCurrentAccount_switchCurrentAccount, switchCurrentAccountVariables>({
                mutation: switchCurrentAccount,
                input: { id: user.id },
                successMessage: () => ({ key: 'LoggedInAs', variables: { name: user.name ?? user.handle ?? '' } }),
                onSuccess: (data) => { PubSub.get().publishSession(data) },
            })
        }
    }, [handleClose, userId, setLocation, switchCurrentAccount]);

    const handleAddAccount = useCallback((event: React.MouseEvent<HTMLElement>) => {
        setLocation(APP_LINKS.Start);
        handleClose(event);
    }, [handleClose, setLocation]);

    const [logOut] = useMutation(logOutMutation);
    const handleLogOut = useCallback((event: React.MouseEvent<HTMLElement>) => {
        handleClose(event);
        const user = getCurrentUser(session);
        mutationWrapper<logOut_logOut, logOutVariables>({
            mutation: logOut,
            input: { id: user.id },
            successMessage: () => ({ key: 'LoggedOutOf', variables: { name: user.name ?? user.handle ?? '' } }),
            onSuccess: (data) => { PubSub.get().publishSession(data) },
            // If error, log out anyway
            onError: () => { PubSub.get().publishSession(guestSession) },
        })
        setLocation(APP_LINKS.Home);
    }, [handleClose, session, logOut, setLocation]);

    const handleOpenSettings = useCallback((event: React.MouseEvent<HTMLElement>) => {
        setLocation(APP_LINKS.Settings);
        handleClose(event);
    }, [handleClose, setLocation]);


    const accounts = useMemo(() => session.users ?? [], [session.users]);
    const profileListItems = accounts.map((account) => (
        <ListItem
            button
            key={account.id}
            onClick={(event) => handleUserClick(event, account)}
            sx={{
                background: account.id === userId ? palette.secondary.light : palette.background.default,
            }}
        >
            <ListItemIcon>
                <UserIcon fill={palette.background.textPrimary} />
            </ListItemIcon>
            <ListItemText primary={account.name ?? account.handle} />
        </ListItem>
    ), [accounts, handleUserClick]);

    return (
        <SwipeableDrawer
            anchor="right"
            open={open}
            onOpen={() => { }}
            onClose={handleClose}
            sx={{
                zIndex: 20000,
                '& .MuiDrawer-paper': {
                    background: palette.background.default,
                    overflowY: 'auto',
                }
            }}
        >
            {/* Custom menu title with theme switch, preferred language selector, text size quantity box, and close icon */}
            <Stack
                direction='row'
                spacing={1}
                sx={{
                    ...noSelect,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: 1,
                    background: palette.primary.dark,
                    color: palette.primary.contrastText,
                    textAlign: 'center',
                    fontSize: { xs: '1.5rem', sm: '2rem' },
                    height: { xs: '64px', md: '80px' }, // Matches navbar height
                    paddingRight: 3, // Matches navbar padding
                }}
            >
                {/* Theme switch */}
                <ThemeSwitch
                    showText={false}
                    theme={formik.values.theme as 'light' | 'dark'}
                    onChange={(t) => formik.setFieldValue('theme', t)}
                />
                {/* Preferred language selector */}
                {/* TODO */}
                {/* Text size quantity box */}
                {/* TODO */}
                {/* Close icon */}
                <IconButton
                    aria-label="close"
                    edge="end"
                    onClick={handleClose}
                    sx={{ marginLeft: 'auto' }}
                >
                    <CloseIcon fill={palette.primary.contrastText} width="40px" height="40px" />
                </IconButton>
            </Stack>
            {/* List of logged/in accounts, authentication-related actions, and additional quick links */}
            <List sx={{ paddingTop: 0, paddingBottom: 0 }}>
                {profileListItems}
                <Divider sx={{ background: palette.background.textSecondary }} />
                {/* Buttons to add account and log out */}
                {accounts.length < MAX_ACCOUNTS && <ListItem button onClick={handleAddAccount}>
                    <ListItemIcon>
                        <PlusIcon fill={palette.background.textPrimary} />
                    </ListItemIcon>
                    <ListItemText primary={'Add account'} />
                </ListItem>}
                {accounts.length > 0 && <ListItem button onClick={handleLogOut}>
                    <ListItemIcon>
                        <LogOutIcon fill={palette.background.textPrimary} />
                    </ListItemIcon>
                    <ListItemText primary={'Log out'} />
                </ListItem>}
                <Divider sx={{ background: palette.background.textSecondary }} />
                {/* Settings page */}
                <ListItem button onClick={handleOpenSettings}>
                    <ListItemIcon>
                        <SettingsIcon fill={palette.background.textPrimary} />
                    </ListItemIcon>
                    <ListItemText primary={'Settings'} />
                </ListItem>
            </List>
            {/* Additional Resources */}
            <Stack direction="row" spacing={1} onClick={toggleAdditionalResources} sx={{
                display: 'flex',
                alignItems: 'center',
                textAlign: 'left',
                paddingLeft: 2,
                paddingRight: 2,
                paddingTop: 1,
                paddingBottom: 1,
            }}>
                <Box sx={{ minWidth: '56px', display: 'flex', alignItems: 'center' }}>
                    <HelpIcon fill={palette.background.textPrimary} />
                </Box>
                <Typography variant="body1" sx={{ color: palette.background.textPrimary, ...noSelect, margin: '0 !important' }}>Additional Resources</Typography>
                {isAdditionalResourcesOpen ? <ExpandMoreIcon fill={palette.background.textPrimary} /> : <ExpandLessIcon fill={palette.background.textPrimary} />}
            </Stack>
            <Collapse in={isAdditionalResourcesOpen} sx={{ display: 'inline-block' }}>
                <ContactInfo />
            </Collapse>
        </SwipeableDrawer>
    )
}