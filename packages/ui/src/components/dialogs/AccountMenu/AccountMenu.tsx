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
    useTheme
} from '@mui/material';
import { Stack } from '@mui/system';
import { LINKS, LogOutInput, ProfileUpdateInput, Session, SessionUser, SwitchCurrentAccountInput, User } from '@shared/consts';
import { AwardIcon, BookmarkFilledIcon, CloseIcon, DisplaySettingsIcon, ExpandLessIcon, ExpandMoreIcon, HelpIcon, HistoryIcon, LogOutIcon, PlusIcon, PremiumIcon, SettingsIcon, UserIcon } from '@shared/icons';
import { useLocation } from '@shared/route';
import { userValidation } from '@shared/validation';
import { authLogOut } from 'api/generated/endpoints/auth_logOut';
import { authSwitchCurrentAccount } from 'api/generated/endpoints/auth_switchCurrentAccount';
import { userProfileUpdate } from 'api/generated/endpoints/user_profileUpdate';
import { useCustomMutation } from 'api/hooks';
import { mutationWrapper } from 'api/utils';
import { FocusModeSelector } from 'components/inputs/FocusModeSelector/FocusModeSelector';
import { LanguageSelector } from 'components/inputs/LanguageSelector/LanguageSelector';
import { LeftHandedCheckbox } from 'components/inputs/LeftHandedCheckbox/LeftHandedCheckbox';
import { TextSizeButtons } from 'components/inputs/TextSizeButtons/TextSizeButtons';
import { ThemeSwitch } from 'components/inputs/ThemeSwitch/ThemeSwitch';
import { ContactInfo } from 'components/navigation/ContactInfo/ContactInfo';
import { useFormik } from 'formik';
import React, { useCallback, useContext, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { noSelect } from 'styles';
import { getCurrentUser, guestSession } from 'utils/authentication/session';
import { useIsLeftHanded } from 'utils/hooks/useIsLeftHanded';
import { useWindowSize } from 'utils/hooks/useWindowSize';
import { PubSub } from 'utils/pubsub';
import { HistoryPageTabOption } from 'utils/search/objectToSearch';
import { SessionContext } from 'utils/SessionContext';
import { shapeProfile } from 'utils/shape/models/profile';
import { AccountMenuProps } from '../types';

// Maximum accounts to sign in with
const MAX_ACCOUNTS = 10;

export const AccountMenu = ({
    anchorEl,
    onClose,
}: AccountMenuProps) => {
    const session = useContext(SessionContext);
    const { breakpoints, palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const isLeftHanded = useIsLeftHanded();

    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);
    const open = Boolean(anchorEl);

    // Display settings collapse
    const [isDisplaySettingsOpen, setIsDisplaySettingsOpen] = useState(false);
    const toggleDisplaySettings = useCallback(() => { setIsDisplaySettingsOpen(!isDisplaySettingsOpen) }, [isDisplaySettingsOpen]);
    const closeDisplaySettings = useCallback(() => { setIsDisplaySettingsOpen(false) }, []);

    // Additional resources collapse
    const [isAdditionalResourcesOpen, setIsAdditionalResourcesOpen] = useState(false);
    const toggleAdditionalResources = useCallback(() => { setIsAdditionalResourcesOpen(!isAdditionalResourcesOpen) }, [isAdditionalResourcesOpen]);
    const closeAdditionalResources = useCallback(() => { setIsAdditionalResourcesOpen(false) }, []);

    // Handle update. Only updates when menu closes, and account settings have changed.
    const [mutation] = useCustomMutation<User, ProfileUpdateInput>(userProfileUpdate);
    const formik = useFormik({
        initialValues: {
            theme: getCurrentUser(session).theme ?? 'light',
        },
        enableReinitialize: true,
        validationSchema: userValidation.update({}),
        onSubmit: (values) => {
            console.log('formik submit')
            // If not logged in, do nothing
            if (!userId) {
                return;
            }
            if (!formik.isValid) return;
            const input = shapeProfile.update({
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
            mutationWrapper<User, ProfileUpdateInput>({
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
        closeDisplaySettings();
    }, [closeAdditionalResources, closeDisplaySettings, formik, onClose]);

    const [switchCurrentAccount] = useCustomMutation<Session, SwitchCurrentAccountInput>(authSwitchCurrentAccount);
    const handleUserClick = useCallback((event: React.MouseEvent<HTMLElement>, user: SessionUser) => {
        // Close menu
        handleClose(event);
        // If already selected, go to profile page
        if (userId === user.id) {
            setLocation(LINKS.Profile);
        }
        // Otherwise, switch to selected account
        else {
            mutationWrapper<Session, SwitchCurrentAccountInput>({
                mutation: switchCurrentAccount,
                input: { id: user.id },
                successMessage: () => ({ key: 'LoggedInAs', variables: { name: user.name ?? user.handle ?? '' } }),
                onSuccess: (data) => { PubSub.get().publishSession(data) },
            })
        }
    }, [handleClose, userId, setLocation, switchCurrentAccount]);

    const handleAddAccount = useCallback((event: React.MouseEvent<HTMLElement>) => {
        setLocation(LINKS.Start);
        handleClose(event);
    }, [handleClose, setLocation]);

    const [logOut] = useCustomMutation<Session, LogOutInput>(authLogOut);
    const handleLogOut = useCallback((event: React.MouseEvent<HTMLElement>) => {
        handleClose(event);
        const user = getCurrentUser(session);
        mutationWrapper<Session, LogOutInput>({
            mutation: logOut,
            input: { id: user.id },
            successMessage: () => ({ key: 'LoggedOutOf', variables: { name: user.name ?? user.handle ?? '' } }),
            onSuccess: (data) => { PubSub.get().publishSession(data) },
            // If error, log out anyway
            onError: () => { PubSub.get().publishSession(guestSession) },
        })
        setLocation(LINKS.Home);
    }, [handleClose, session, logOut, setLocation]);

    const handleOpen = useCallback((event: React.MouseEvent<HTMLElement>, link: string) => {
        setLocation(link);
        handleClose(event);
    }, [handleClose, setLocation]);
    const handleOpenSettings = useCallback((event: React.MouseEvent<HTMLElement>) => {
        handleOpen(event, LINKS.Settings);
    }, [handleOpen]);
    const handleOpenBookmarks = useCallback((event: React.MouseEvent<HTMLElement>) => {
        handleOpen(event, `${LINKS.History}?type=${HistoryPageTabOption.Bookmarked}`);
    }, [handleOpen]);
    const handleOpenHistory = useCallback((event: React.MouseEvent<HTMLElement>) => {
        handleOpen(event, LINKS.History);
    }, [handleOpen]);
    const handleOpenAwards = useCallback((event: React.MouseEvent<HTMLElement>) => {
        handleOpen(event, LINKS.Awards);
    }, [handleOpen]);
    const handleOpenPremium = useCallback((event: React.MouseEvent<HTMLElement>) => {
        handleOpen(event, LINKS.Premium);
    }, [handleOpen]);


    const accounts = useMemo(() => session?.users ?? [], [session?.users]);
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
            anchor={(isMobile && isLeftHanded) ? 'left' : 'right'}
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
            {/* Menu title with close icon */}
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
                    height: '64px', // Matches Navbar height
                    paddingRight: 3, // Matches navbar padding
                }}
            >
                {/* Close icon */}
                <IconButton
                    aria-label="close"
                    edge="end"
                    onClick={handleClose}
                    sx={{
                        marginLeft: (isMobile && isLeftHanded) ? 'unset' : 'auto',
                        marginRight: (isMobile && isLeftHanded) ? 'auto' : 'unset',
                    }}
                >
                    <CloseIcon fill={palette.primary.contrastText} width="40px" height="40px" />
                </IconButton>
            </Stack>
            {/* List of logged/in accounts and authentication-related actions */}
            <List sx={{ paddingTop: 0, paddingBottom: 0 }}>
                {profileListItems}
                <Divider sx={{ background: palette.background.textSecondary }} />
                {/* Buttons to add account and log out */}
                {accounts.length < MAX_ACCOUNTS && <ListItem button onClick={handleAddAccount}>
                    <ListItemIcon>
                        <PlusIcon fill={palette.background.textPrimary} />
                    </ListItemIcon>
                    <ListItemText primary={t(`AddAccount`)} />
                </ListItem>}
                {accounts.length > 0 && <ListItem button onClick={handleLogOut}>
                    <ListItemIcon>
                        <LogOutIcon fill={palette.background.textPrimary} />
                    </ListItemIcon>
                    <ListItemText primary={t(`LogOut`)} />
                </ListItem>}
            </List>
            <Divider sx={{ background: palette.background.textSecondary }} />
            {/* Display Settings */}
            <Stack direction="row" spacing={1} onClick={toggleDisplaySettings} sx={{
                display: 'flex',
                alignItems: 'center',
                textAlign: 'left',
                paddingLeft: 2,
                paddingRight: 2,
                paddingTop: 1,
                paddingBottom: 1,
            }}>
                <Box sx={{ minWidth: '56px', display: 'flex', alignItems: 'center' }}>
                    <DisplaySettingsIcon fill={palette.background.textPrimary} />
                </Box>
                <Typography variant="body1" sx={{ color: palette.background.textPrimary, ...noSelect, margin: '0 !important' }}>{t(`Display`)}</Typography>
                {isDisplaySettingsOpen ? <ExpandMoreIcon fill={palette.background.textPrimary} style={{ marginLeft: "auto" }} /> : <ExpandLessIcon fill={palette.background.textPrimary} style={{ marginLeft: "auto" }} />}
            </Stack>
            <Collapse in={isDisplaySettingsOpen} sx={{ display: 'inline-block', minHeight: 'auto!important' }}>
                <Stack direction="column" spacing={2} sx={{
                    minWidth: 'fit-content',
                    height: 'fit-content',
                    padding: 1,
                }}>
                    <ThemeSwitch />
                    <TextSizeButtons />
                    <LeftHandedCheckbox />
                    <LanguageSelector />
                    <FocusModeSelector />
                </Stack>
            </Collapse>
            <Divider sx={{ background: palette.background.textSecondary }} />
            {/* List of quick links */}
            <List>
                {/* Settings page */}
                <ListItem button onClick={handleOpenSettings}>
                    <ListItemIcon>
                        <SettingsIcon fill={palette.background.textPrimary} />
                    </ListItemIcon>
                    <ListItemText primary={t(`Settings`)} />
                </ListItem>
                {/* Bookmarked */}
                <ListItem button onClick={handleOpenBookmarks}>
                    <ListItemIcon>
                        <BookmarkFilledIcon fill={palette.background.textPrimary} />
                    </ListItemIcon>
                    <ListItemText primary={t(`Bookmark`, { count: 2 })} />
                </ListItem>
                {/* History */}
                <ListItem button onClick={handleOpenHistory}>
                    <ListItemIcon>
                        <HistoryIcon fill={palette.background.textPrimary} />
                    </ListItemIcon>
                    <ListItemText primary={t(`History`)} />
                </ListItem>
                {/* Awards */}
                <ListItem button onClick={handleOpenAwards}>
                    <ListItemIcon>
                        <AwardIcon fill={palette.background.textPrimary} />
                    </ListItemIcon>
                    <ListItemText primary={t(`Award`, { count: 2 })} />
                </ListItem>
                {/* Premium */}
                <ListItem button onClick={handleOpenPremium}>
                    <ListItemIcon>
                        <PremiumIcon fill={palette.background.textPrimary} />
                    </ListItemIcon>
                    <ListItemText primary={t(`Premium`)} />
                </ListItem>
            </List>
            <Divider sx={{ background: palette.background.textSecondary }} />
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
                <Typography variant="body1" sx={{ color: palette.background.textPrimary, ...noSelect, margin: '0 !important' }}>{t(`AdditionalResources`)}</Typography>
                {isAdditionalResourcesOpen ? <ExpandMoreIcon fill={palette.background.textPrimary} style={{ marginLeft: "auto" }} /> : <ExpandLessIcon fill={palette.background.textPrimary} style={{ marginLeft: "auto" }} />}
            </Stack>
            <Collapse in={isAdditionalResourcesOpen} sx={{ display: 'inline-block' }}>
                <ContactInfo />
            </Collapse>
        </SwipeableDrawer>
    )
}