import {
    Divider,
    IconButton,
    List,
    ListItem,
    ListItemIcon,
    ListItemText,
    Menu,
    useTheme,
} from '@mui/material';
import { Stack } from '@mui/system';
import { CloseIcon, LogOutIcon, PlusIcon, UserIcon } from '@shared/icons';
import { AccountMenuProps } from '../types';
import { noSelect } from 'styles';
import { ThemeSwitch } from 'components/inputs';
import { useCallback, useMemo } from 'react';
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

    const handleClose = useCallback(() => {
        formik.handleSubmit();
        onClose();
    }, [formik, onClose]);

    const [switchCurrentAccount] = useMutation(switchCurrentAccountMutation);
    const handleUserClick = useCallback((user: SessionUser) => {
        // Close menu
        handleClose();
        // If already selected, go to profile page
        if (userId === user.id) {
            setLocation(APP_LINKS.Profile);
        }
        // Otherwise, switch to selected account
        else {
            mutationWrapper<switchCurrentAccount_switchCurrentAccount, switchCurrentAccountVariables>({
                mutation: switchCurrentAccount,
                input: { id: user.id },
                successMessage: () => `Logged in as ${user.name ?? user.handle}`,
                onSuccess: (data) => { PubSub.get().publishSession(data) },
            })
        }
    }, [handleClose, userId, setLocation, switchCurrentAccount]);

    const handleAddAccount = useCallback(() => {
        setLocation(APP_LINKS.Start);
        handleClose();
    }, [handleClose, setLocation]);

    const [logOut] = useMutation(logOutMutation);
    const handleLogOut = useCallback(() => {
        handleClose();
        const user = getCurrentUser(session);
        mutationWrapper<logOut_logOut, logOutVariables>({
            mutation: logOut,
            input: { id: user.id },
            successMessage: () => `Logged out of ${user.name ?? user.handle}`,
            onSuccess: (data) => { PubSub.get().publishSession(data) },
            // If error, log out anyway
            onError: () => { PubSub.get().publishSession(guestSession) },
        })
        setLocation(APP_LINKS.Home);
    }, [handleClose, session, logOut, setLocation]);


    const accounts = useMemo(() => session.users ?? [], [session.users]);
    const profileListItems = accounts.map((account) => (
        <ListItem
            button
            key={account.id}
            onClick={() => handleUserClick(account)}
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
        <Menu
            id='account-menu-id'
            disableScrollLock={true}
            autoFocus={true}
            open={open}
            anchorEl={anchorEl}
            anchorOrigin={{
                vertical: 'bottom',
                horizontal: 'right',
            }}
            transformOrigin={{
                vertical: 'bottom',
                horizontal: 'right',
            }}
            onClose={(e) => { handleClose() }}
            sx={{
                zIndex: 20000,
                '& .MuiMenu-paper': {
                    background: palette.background.default,
                    minWidth: 'min(100%, 250px)',
                    boxShadow: 12,
                },
                '& .MuiMenu-list': {
                    paddingTop: '0',
                },
                '& .MuiList-root': {
                    padding: '0',
                },
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
                    <CloseIcon fill={palette.primary.contrastText} />
                </IconButton>
            </Stack>
            {/* List of logged/in accounts */}
            <List>
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
            </List>
        </Menu>
    )
}