import {
    IconButton,
    List,
    ListItem,
    ListItemIcon,
    ListItemText,
    Menu,
    useTheme,
} from '@mui/material';
import { Stack } from '@mui/system';
import { CloseIcon, UserIcon } from '@shared/icons';
import { AccountMenuProps } from '../types';
import { noSelect } from 'styles';
import { ThemeSwitch } from 'components/inputs';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { profileUpdateSchema as validationSchema } from '@shared/validation';
import { profileUpdateVariables, profileUpdate_profileUpdate } from 'graphql/generated/profileUpdate';
import { useMutation } from '@apollo/client';
import { shapeProfileUpdate } from 'utils';
import { mutationWrapper } from 'graphql/utils';
import { useFormik } from 'formik';
import { profileUpdateMutation } from 'graphql/mutation';
import { APP_LINKS } from '@shared/consts';
import { useLocation } from '@shared/route';

export const AccountMenu = ({
    anchorEl,
    onClose,
    session,
}: AccountMenuProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const open = Boolean(anchorEl);

    // Handle theme
    const [theme, setTheme] = useState<string>('light');
    useEffect(() => {
        setTheme(palette.mode);
    }, [palette.mode]);

    // Handle update. Only updates when menu closes, and account settings have changed.
    const [mutation] = useMutation(profileUpdateMutation);
    const formik = useFormik({
        initialValues: {
            theme,
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            // If not logged in, do nothing
            if (!session?.id) {
                return;
            }
            if (!formik.isValid) return;
            const input = shapeProfileUpdate({
                id: session.id,
                theme: palette.mode,
            }, {
                id: session.id,
                theme: values.theme,
            })
            // If no changes, don't update
            if (!input || Object.keys(input).length === 0) {
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

    /**
     * Handle account click.
     */
    const handleUserClick = useCallback((id: string) => {
        // If already selected, go to profile page
        if (session?.id === id) {
            setLocation(APP_LINKS.Profile);
        }
        // Otherwise, switch to selected account
        else {
            console.log('TODO')
        }
        // Close menu
        onClose();
    }, [onClose, session?.id, setLocation]);

    // TODO temporarily one profile, since we don't support multiple sessions yet
    const profileListItems = useMemo(() => (<ListItem
        button
        key={session?.id ?? ''}
        onClick={() => handleUserClick(session.id ?? '')}
    >
        <ListItemIcon>
            <UserIcon />
        </ListItemIcon>
        <ListItemText primary={'me'} />
    </ListItem>), [handleUserClick, session.id])

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
            onClose={(e) => { onClose() }}
            sx={{
                zIndex: 20000,
                '& .MuiMenu-paper': {
                    background: palette.background.default
                },
                '& .MuiMenu-list': {
                    paddingTop: '0',
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
                    padding: 2,
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
                    onClick={onClose}
                    sx={{ marginLeft: 'auto' }}
                >
                    <CloseIcon fill={palette.primary.contrastText} />
                </IconButton>
            </Stack>
            {/* List of logged/in accounts */}
            <List>
                {profileListItems}
            </List>
            {/* Buttons to add account and log out */}
            {/* TODO */}
        </Menu>
    )
}