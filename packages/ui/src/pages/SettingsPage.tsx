import { Box, Divider, IconButton, List, ListItem, ListItemIcon, ListItemText, SwipeableDrawer, useTheme } from '@mui/material';
import { useCallback, useMemo, useState } from 'react';
import { SettingsPageProps } from './types';
import {
    Lock as AuthenticationIcon,
    ChevronLeft as CloseIcon,
    LightMode as DisplayIcon,
    Notifications as NotificationsIcon,
    ChevronRight as OpenIcon,
    Shield as PrivacyIcon,
    AccountCircle as ProfileIcon,
    SvgIconComponent
} from '@mui/icons-material';
import { logOutMutation } from 'graphql/mutation';
import { useMutation } from '@apollo/client';
import PubSub from 'pubsub-js';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { Pubs } from 'utils';
import { APP_LINKS } from '@local/shared';
import { useLocation } from 'wouter';
import { ProfileUpdate } from 'components/views/ProfileUpdate/ProfileUpdate';

const settingPages: { [x: string]: [string, string, SvgIconComponent] } = {
    'profile': ['profile', 'Profile', ProfileIcon],
    'display': ['display', 'Display', DisplayIcon],
    'notifications': ['notifications', 'Notifications', NotificationsIcon],
    'privacy': ['privacy', 'Privacy', PrivacyIcon],
    'authentication': ['authentication', 'Authentication', AuthenticationIcon],
}

export function SettingsPage({
    session,
}: SettingsPageProps) {
    const theme = useTheme();
    const [location, setLocation] = useLocation();
    if (location === APP_LINKS.Settings) setLocation(`${APP_LINKS.Settings}?page=profile`, { replace: true });

    const selectedPage: [string, string, SvgIconComponent] = useMemo(() => {
        const split = location.split(`/${APP_LINKS.Settings}?page=`);
        if (split.length === 1) return settingPages['profile'];
        return settingPages[split[1]] ?? settingPages['profile'];
    }, [location]);

    const [drawerOpen, setDrawerOpen] = useState(false);
    const openDrawer = useCallback(() => setDrawerOpen(true), [setDrawerOpen]);
    const closeDrawer = useCallback(() => setDrawerOpen(false), [setDrawerOpen]);
    const toggleDrawer = useCallback(() => {console.log('in toggle drawer'); setDrawerOpen(false)}, [setDrawerOpen]);

    const [logOut] = useMutation<any>(logOutMutation);
    const onLogOut = useCallback(() => {
        mutationWrapper({
            mutation: logOut,
        })
        PubSub.publish(Pubs.LogOut);
    }, []);

    const listItems = useMemo(() => {
        return Object.values(settingPages).map(([link, label, Icon]: [string, string, SvgIconComponent], index) => {
            const selected = link === selectedPage[0];
            console.log('selected', selected, link, selectedPage);
            return (
                <ListItem
                    key={index}
                    button
                    onClick={() => { setLocation(`${APP_LINKS.Settings}?page=${link}`, { replace: true }) }}
                    sx={{
                        transition: 'brightness 0.2s ease-in-out',
                        background: selected ? '#5bb6ce6e' : 'inherit',
                    }}
                >
                    <ListItemIcon>
                        <Icon />
                    </ListItemIcon>
                    <ListItemText primary={label} />
                </ListItem>
            )
        });
    }, [selectedPage, setLocation]);

    const mainContent: JSX.Element = useMemo(() => {
        switch (selectedPage[0]) {
            case 'display':
                return <div>Display</div>;
            case 'notifications':
                return <div>Notifications</div>;
            case 'privacy':
                return <div>Privacy</div>;
            case 'authentication':
                return <div>Authentication</div>;
            default:
                return <ProfileUpdate session={session} onUpdated={() => { }} onCancel={() => { }} />;
        }
    }, [selectedPage]);

    return (
        <Box id="page">
            {/* Setting pages drawer */}
            <SwipeableDrawer
                variant='permanent'
                anchor="left"
                open={drawerOpen}
                onClose={closeDrawer}
                onOpen={openDrawer}
                sx={{
                    '& .MuiDrawer-paper': {
                        position: 'absolute',
                        top: 'calc(10vh)',
                        zIndex: 2,
                        backgroundColor: '#949ba56e',
                        backdropFilter: 'blur(5px)',
                        color: 'black',
                        borderRight: (t) => `1px solid ${t.palette.text.primary}`,
                    }
                }}
            >
                {/* Drawer header */}
                <Box sx={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'flex-end',
                }}>
                    <IconButton onClick={toggleDrawer}>
                        {theme.direction === 'rtl' ? <OpenIcon /> : <CloseIcon />}
                    </IconButton>
                </Box>
                <Divider />
                {/* Drawer content */}
                <List>
                    {listItems}
                </List>
            </SwipeableDrawer>
            {/* Selected page display */}
            <Box display="flex" justifyContent="center" alignItems="center" sx={{ maxWidth: 'min(100%, 700px)' }}>
                {mainContent}
            </Box>
        </Box >
    )
}
