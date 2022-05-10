import { Box, Divider, IconButton, List, ListItem, ListItemIcon, ListItemText, Stack, SwipeableDrawer, useTheme } from '@mui/material';
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
import { useLazyQuery } from '@apollo/client';
import { APP_LINKS } from '@local/shared';
import { useLocation } from 'wouter';
import { SettingsProfile } from 'components/views/SettingsProfile/SettingsProfile';
import { parseSearchParams } from 'utils/navigation/urlTools';
import { profile, profile_profile } from 'graphql/generated/profile';
import { profileQuery } from 'graphql/query';
import { SettingsAuthentication } from 'components/views/SettingsAuthentication/SettingsAuthentication';
import { SettingsDisplay } from 'components/views/SettingsDisplay/SettingsDisplay';
import { SettingsNotifications } from 'components/views/SettingsNotifications/SettingsNotifications';
import { containerShadow } from 'styles';

/**
 * All settings forms. Same as their route names.
 */
enum SettingsForm {
    Profile = 'profile',
    Display = 'display',
    Notifications = 'notifications',
    Authentication = 'authentication',
}

const settingPages: { [x: string]: [SettingsForm, string, SvgIconComponent] } = {
    [SettingsForm.Profile]: [SettingsForm.Profile, 'Profile', ProfileIcon],
    [SettingsForm.Display]: [SettingsForm.Display, 'Display', DisplayIcon],
    [SettingsForm.Notifications]: [SettingsForm.Notifications, 'Notifications', NotificationsIcon],
    [SettingsForm.Authentication]: [SettingsForm.Authentication, 'Authentication', AuthenticationIcon],
}

export function SettingsPage({
    session,
}: SettingsPageProps) {
    const { breakpoints, palette } = useTheme();
    const [location, setLocation] = useLocation();
    const selectedPage = useMemo<SettingsForm>(() => { return parseSearchParams(window.location.search).page as unknown as SettingsForm ?? SettingsForm.Profile }, [window.location.search]);
    const editing = useMemo<boolean>(() => Boolean(parseSearchParams(window.location.search).editing), [window.location.search]);

    // Fetch profile data
    const [getData, { data, loading }] = useLazyQuery<profile>(profileQuery);
    useEffect(() => {
        if (session?.id) getData();
    }, [session])
    const [profile, setProfile] = useState<profile_profile | undefined>(undefined);
    useEffect(() => {
        if (data?.profile) setProfile(data.profile);
    }, [data]);
    const onUpdated = useCallback((updatedProfile: profile_profile | undefined) => {
        if (updatedProfile) setProfile(updatedProfile);
    }, []);

    const [drawerOpen, setDrawerOpen] = useState(false);
    const toggleDrawer = useCallback(() => { setDrawerOpen(o => !o) }, [setDrawerOpen]);

    const listItems = useMemo(() => {
        return Object.values(settingPages).map(([link, label, Icon]: [SettingsForm, string, SvgIconComponent], index) => {
            const selected = link === selectedPage;
            return (
                <ListItem
                    key={index}
                    button
                    onClick={() => { setLocation(`${APP_LINKS.Settings}?page=${link}`, { replace: true }) }}
                    sx={{
                        transition: 'brightness 0.2s ease-in-out',
                        background: selected ? '#5bb6ce6e' : 'inherit',
                        padding: drawerOpen ? '8px 16px' : '8px 12px'
                    }}
                >
                    <ListItemIcon>
                        <Icon />
                    </ListItemIcon>
                </ListItem>
            )
        });
    }, [selectedPage, drawerOpen]);

    const mainContent: JSX.Element = useMemo(() => {
        switch (selectedPage) {
            case SettingsForm.Profile:
                return <SettingsProfile session={session} profile={profile} onUpdated={onUpdated} />
            case SettingsForm.Display:
                return <SettingsDisplay session={session} profile={profile} onUpdated={onUpdated} />
            case SettingsForm.Notifications:
                return <SettingsNotifications session={session} profile={profile} onUpdated={onUpdated} />
            case SettingsForm.Authentication:
                return <SettingsAuthentication session={session} profile={profile} onUpdated={onUpdated} />
        }
    }, [selectedPage, editing, profile, onUpdated]);

    return (
        <Box id='page' sx={{ 
            display: 'flex', 
            justifyContent: 'center', 
            alignItems: 'center',
            [breakpoints.up('md')]: {
                paddingTop: '10vh',
            },
        }}>
            <Box sx={{
                ...containerShadow,
                borderRadius: 2,
                overflow: 'overlay',
                background: palette.background.default,
                width: 'min(100%, 700px)',
                minHeight: '300px',
                position: 'relative',
                margin: 'auto',
            }}>
                {/* Left side drawer */}
                <Box sx={{
                    position: 'absolute',
                    width: '50px',
                    left: 0,
                    top: 0,
                    bottom: 0,
                    backgroundColor: '#e4efff',
                    borderRight: `1px solid ${palette.text.primary}`,

                }}>
                    <List>
                        {listItems}
                    </List>
                </Box>
                {/* Settings form */}
                <Box sx={{
                    marginLeft: '50px',
                    width: 'calc(100% - 50px)',
                }}>
                    {mainContent}
                </Box>
            </Box>
        </Box>
    )
}
