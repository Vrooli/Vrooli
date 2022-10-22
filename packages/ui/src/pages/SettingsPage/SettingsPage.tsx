import { Box, CircularProgress, List, ListItem, ListItemIcon, useTheme } from '@mui/material';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { SettingsPageProps } from '../types';
import { useLazyQuery } from '@apollo/client';
import { APP_LINKS } from '@shared/consts';
import { useLocation } from '@shared/route';
import { SettingsProfile } from 'components/views/SettingsProfile/SettingsProfile';
import { profile, profile_profile } from 'graphql/generated/profile';
import { profileQuery } from 'graphql/query';
import { SettingsAuthentication } from 'components/views/SettingsAuthentication/SettingsAuthentication';
import { SettingsDisplay } from 'components/views/SettingsDisplay/SettingsDisplay';
import { SettingsNotifications } from 'components/views/SettingsNotifications/SettingsNotifications';
import { useReactSearch } from 'utils';
import { LightModeIcon, LockIcon, NotificationsCustomizedIcon, ProfileIcon, SvgComponent } from '@shared/icons';
import { PageContainer } from 'components';
import { getCurrentUser } from 'utils/authentication';

/**
 * All settings forms. Same as their route names.
 */
enum SettingsForm {
    Profile = 'profile',
    Display = 'display',
    Notifications = 'notifications',
    Authentication = 'authentication',
}

const settingPages: { [x: string]: [SettingsForm, string, SvgComponent] } = {
    [SettingsForm.Profile]: [SettingsForm.Profile, 'Profile', ProfileIcon],
    [SettingsForm.Display]: [SettingsForm.Display, 'Display', LightModeIcon],
    [SettingsForm.Notifications]: [SettingsForm.Notifications, 'Notifications', NotificationsCustomizedIcon],
    [SettingsForm.Authentication]: [SettingsForm.Authentication, 'Authentication', LockIcon],
}

export function SettingsPage({
    session,
}: SettingsPageProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const searchParams = useReactSearch();
    const { selectedPage } = useMemo(() => ({
        selectedPage: searchParams.page as unknown as SettingsForm ?? SettingsForm.Profile,
    }), [searchParams]);

    // Fetch profile data
    const [getData, { data, loading }] = useLazyQuery<profile>(profileQuery, { errorPolicy: 'all' });
    useEffect(() => {
        if (getCurrentUser(session).id) getData();
    }, [getData, session])
    const [profile, setProfile] = useState<profile_profile | undefined>(undefined);
    useEffect(() => {
        if (data?.profile) setProfile(data.profile);
    }, [data]);
    const onUpdated = useCallback((updatedProfile: profile_profile | undefined) => {
        if (updatedProfile) setProfile(updatedProfile);
    }, []);

    const listItems = useMemo(() => {
        return Object.values(settingPages).map(([link, label, Icon]: [SettingsForm, string, SvgComponent], index) => {
            const selected = link === selectedPage;
            return (
                <ListItem
                    key={index}
                    button
                    onClick={() => { setLocation(`${APP_LINKS.Settings}?page="${link}"`, { replace: true }) }}
                    sx={{
                        transition: 'brightness 0.2s ease-in-out',
                        background: selected ? '#5bb6ce6e' : 'inherit',
                        padding: '8px 12px'
                    }}
                >
                    <ListItemIcon>
                        <Icon fill='rgba(0, 0, 0, 0.54)' />
                    </ListItemIcon>
                </ListItem>
            )
        });
    }, [selectedPage, setLocation]);

    const mainContent: JSX.Element = useMemo(() => {
        switch (selectedPage) {
            case SettingsForm.Profile:
                return <SettingsProfile session={session} profile={profile} onUpdated={onUpdated} zIndex={200} />
            case SettingsForm.Display:
                return <SettingsDisplay session={session} profile={profile} onUpdated={onUpdated} zIndex={200} />
            case SettingsForm.Notifications:
                return <SettingsNotifications session={session} profile={profile} onUpdated={onUpdated} zIndex={200} />
            case SettingsForm.Authentication:
                return <SettingsAuthentication session={session} profile={profile} onUpdated={onUpdated} zIndex={200} />
        }
    }, [selectedPage, session, profile, onUpdated]);

    return (
        <PageContainer sx={{
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
        }}>
            <Box sx={{
                boxShadow: { xs: 'none', sm: 12 },
                borderRadius: { xs: 0, sm: 2 },
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
                    backgroundColor: palette.mode === 'light' ? '#e4efff' : palette.primary.light,
                    borderRight: `1px solid ${palette.mode === 'light' ? palette.text.primary : palette.background.default}`,

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
                    {loading ? <CircularProgress color="secondary" /> : mainContent}
                </Box>
            </Box>
        </PageContainer>
    )
}
