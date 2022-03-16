import { Box, Divider, IconButton, List, ListItem, ListItemIcon, ListItemText, SwipeableDrawer, useTheme } from '@mui/material';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { SettingsPageProps } from '../types';
import {
    Lock as AuthenticationIcon,
    ChevronLeft as CloseIcon,
    LightMode as DisplayIcon,
    Notifications as NotificationsIcon,
    ChevronRight as OpenIcon,
    AccountCircle as ProfileIcon,
    SvgIconComponent
} from '@mui/icons-material';
import { useQuery } from '@apollo/client';
import PubSub from 'pubsub-js';
import { Pubs } from 'utils';
import { APP_LINKS } from '@local/shared';
import { useLocation } from 'wouter';
import { SettingsProfile } from 'components/views/SettingsProfile/SettingsProfile';
import { parseSearchParams } from 'utils/urlTools';
import { profile, profile_profile } from 'graphql/generated/profile';
import { profileQuery } from 'graphql/query';
import { UserView } from 'components';
import { SettingsAuthentication } from 'components/views/SettingsAuthentication/SettingsAuthentication';
import { SettingsDisplay } from 'components/views/SettingsDisplay/SettingsDisplay';
import { SettingsNotifications } from 'components/views/SettingsNotifications/SettingsNotifications';

const settingPages: { [x: string]: [string, string, SvgIconComponent] } = {
    'profile': ['profile', 'Profile', ProfileIcon],
    'display': ['display', 'Display', DisplayIcon],
    'notifications': ['notifications', 'Notifications', NotificationsIcon],
    'authentication': ['authentication', 'Authentication', AuthenticationIcon],
}

export function SettingsPage({
    session,
}: SettingsPageProps) {
    const [location, setLocation] = useLocation();
    const selectedPage = useMemo<string>(() => { console.log('PARAMS', parseSearchParams(window.location.search)); return parseSearchParams(window.location.search).page ?? 'profile' }, [window.location.search]);
    const editing = useMemo<boolean>(() => Boolean(parseSearchParams(window.location.search).editing), [window.location.search]);

    // Fetch profile data
    const { data, loading } = useQuery<profile>(profileQuery, { variables: { input: { id: session?.id ?? '' } } });
    const [profile, setProfile] = useState<profile_profile | undefined>(undefined);
    useEffect(() => {
        if (data?.profile) setProfile(data.profile);
    }, [data]);
    const onUpdated = useCallback((updatedProfile: profile_profile | undefined) => {
        console.log('onUpdated', updatedProfile);
        if (updatedProfile) setProfile(updatedProfile);
        PubSub.publish(Pubs.Snack, { message: 'Update successful.' });
    }, []);

    const [drawerOpen, setDrawerOpen] = useState(false);
    const toggleDrawer = useCallback(() => { console.log('in toggle drawer'); setDrawerOpen(o => !o) }, [setDrawerOpen]);

    const listItems = useMemo(() => {
        console.log('in list items', selectedPage);
        return Object.values(settingPages).map(([link, label, Icon]: [string, string, SvgIconComponent], index) => {
            const selected = link === selectedPage;
            console.log('selected', selectedPage);
            return (
                <ListItem
                    key={index}
                    button
                    onClick={() => { console.log('setting location', `${APP_LINKS.Settings}?page=${link}`); setLocation(`${APP_LINKS.Settings}?page=${link}`, { replace: true }) }}
                    sx={{
                        transition: 'brightness 0.2s ease-in-out',
                        background: selected ? '#5bb6ce6e' : 'inherit',
                        padding: drawerOpen ? '8px 16px' : '8px 12px'
                    }}
                >
                    <ListItemIcon>
                        <Icon />
                    </ListItemIcon>
                    <ListItemText primary={label} />
                </ListItem>
            )
        });
    }, [selectedPage, drawerOpen]);

    const mainContent: JSX.Element = useMemo(() => {
        switch (selectedPage) {
            case 'display':
                return <SettingsDisplay session={session} profile={profile} onUpdated={onUpdated} />
            case 'notifications':
                return <SettingsNotifications session={session} profile={profile} onUpdated={onUpdated} />
            case 'authentication':
                return <SettingsAuthentication session={session} profile={profile} onUpdated={onUpdated} />
            default:
                return editing ? <SettingsProfile profile={profile} onUpdated={onUpdated} /> : <UserView partialData={profile as any} session={session} />
        }
    }, [selectedPage, editing, profile, onUpdated]);

    console.log('drawer open', drawerOpen);

    return (
        <Box id="page">
            {/* Setting pages drawer */}
            <SwipeableDrawer
                variant='permanent'
                anchor="left"
                open={drawerOpen}
                onClose={() => {}}
                onOpen={() => {}}
                sx={{
                    '& .MuiDrawer-paper': {
                        position: 'absolute',
                        top: 'calc(10vh)',
                        zIndex: 2,
                        backgroundColor: '#e4efff',
                        color: 'black',
                        borderRight: (t) => `1px solid ${t.palette.text.primary}`,
                        width: drawerOpen ? 'fit-content' : '50px', 
                        transition: 'width 0.5s ease-in-out',
                    }
                }}
            >
                {/* Drawer header */}
                <Box sx={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'flex-end',
                }}>
                    <IconButton onClick={toggleDrawer} sx={{padding: drawerOpen ? '8px 16px' : '8px 12px'}}>
                        {drawerOpen ? <CloseIcon /> : <OpenIcon />}
                    </IconButton>
                </Box>
                <Divider />
                {/* Drawer content */}
                <List>
                    {listItems}
                </List>
            </SwipeableDrawer>
            {/* Selected page display */}
            <Box sx={{
                width: `calc(100vw - 58px)`,
                position: 'absolute',
                right: 0,
            }}>
                {mainContent}
            </Box>
        </Box >
    )
}
