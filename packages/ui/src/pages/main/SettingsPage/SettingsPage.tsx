import { Box, CircularProgress, Collapse, Divider, Grid, List, ListItem, ListItemIcon, ListItemText, Typography, useTheme } from '@mui/material';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { SettingsPageProps } from '../types';
import { useLazyQuery } from 'api/hooks';
import { APP_LINKS, User } from '@shared/consts';
import { useLocation } from '@shared/route';
import { SettingsProfile } from 'components/views/SettingsProfile/SettingsProfile';
import { SettingsAuthentication } from 'components/views/SettingsAuthentication/SettingsAuthentication';
import { SettingsDisplay } from 'components/views/SettingsDisplay/SettingsDisplay';
import { SettingsNotifications } from 'components/views/SettingsNotifications/SettingsNotifications';
import { getUserLanguages, useReactSearch, useWindowSize } from 'utils';
import { ExpandLessIcon, ExpandMoreIcon, LightModeIcon, LockIcon, NotificationsCustomizedIcon, ProfileIcon, SvgComponent } from '@shared/icons';
import { PageContainer, SearchBar } from 'components';
import { getCurrentUser } from 'utils/authentication';
import { noSelect } from 'styles';
import { CommonKey } from 'types';
import { useTranslation } from 'react-i18next';
import { userEndpoint } from 'api/endpoints';

/**
 * Describes a settings page button
 */
export interface SettingsFormItemData {
    id: string;
    labels: string[];
}

/**
 * Describes a settings page form for the search bar
 */
export interface SettingsFormData {
    labels: string[];
    items: SettingsFormItemData[];
}

/**
 * All settings forms. Same as their route names.
 */
type SettingsForm = 'profile' | 'display' | 'notifications' | 'authentication';

const pageDisplayData: { [key in SettingsForm]: [CommonKey, SvgComponent] } = {
    'profile': ['Profile', ProfileIcon],
    'display': ['Display', LightModeIcon],
    'notifications': ['Notifications', NotificationsCustomizedIcon],
    'authentication': ['Authentication', LockIcon],
}

export function SettingsPage({
    session,
}: SettingsPageProps) {
    const { t } = useTranslation();
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const lng = useMemo(() => getUserLanguages(session)[0], [session]);
    const [, setLocation] = useLocation();
    const searchParams = useReactSearch();
    const { selectedPage } = useMemo(() => ({
        selectedPage: searchParams.page as unknown as SettingsForm ?? 'profile',
    }), [searchParams]);

    // Fetch profile data
    const [getData, { data, loading }] = useLazyQuery<User, null, 'profile'>(...userEndpoint.profile, { errorPolicy: 'all' });
    useEffect(() => {
        if (getCurrentUser(session).id) getData();
    }, [getData, session])
    const [profile, setProfile] = useState<User | undefined>(undefined);
    useEffect(() => {
        if (data?.profile) setProfile(data.profile);
    }, [data]);
    const onUpdated = useCallback((updatedProfile: User | undefined) => {
        if (updatedProfile) setProfile(updatedProfile);
    }, []);

    const [isListOpen, setIsListOpen] = useState(isMobile);
    const toggleList = useCallback(() => { setIsListOpen(!isListOpen) }, [isListOpen, setIsListOpen]);
    const closeList = useCallback(() => { setIsListOpen(false) }, [setIsListOpen]);

    const listItems = useMemo(() => {
        return Object.entries(pageDisplayData).map(([link, [label, Icon]], index) => {
            const selected = link === selectedPage;
            return (
                <ListItem
                    key={index}
                    button
                    onClick={() => {
                        setLocation(`${APP_LINKS.Settings}?page="${link}"`, { replace: true });
                        closeList();
                    }}
                    sx={{
                        transition: 'brightness 0.2s ease-in-out',
                        background: selected ? palette.primary.main : 'transparent',
                        color: selected ? palette.primary.contrastText : palette.background.textPrimary,
                        padding: '8px 12px'
                    }}
                >
                    <ListItemIcon>
                        <Icon fill={selected ? palette.primary.contrastText : palette.background.textSecondary} />
                    </ListItemIcon>
                    <ListItemText primary={t(`common:${label}`, { lng })} />
                </ListItem>
            )
        });
    }, [closeList, lng, palette.background.textPrimary, palette.background.textSecondary, palette.primary.contrastText, palette.primary.main, selectedPage, setLocation, t]);

    const mainContent: JSX.Element = useMemo(() => {
        switch (selectedPage) {
            case 'profile':
                return <SettingsProfile session={session} profile={profile} onUpdated={onUpdated} zIndex={200} />
            case 'display':
                return <SettingsDisplay session={session} profile={profile} onUpdated={onUpdated} zIndex={200} />
            case 'notifications':
                return <SettingsNotifications session={session} profile={profile} onUpdated={onUpdated} zIndex={200} />
            default:
                return <SettingsAuthentication session={session} profile={profile} onUpdated={onUpdated} zIndex={200} />
        }
    }, [selectedPage, session, profile, onUpdated]);

    return (
        <PageContainer>
            {/* Search bar to find settings TODO */}
            {/* <Box sx={{
                width: 'min(100%, 700px)',
                margin: 'auto',
                marginTop: 2,
                marginBottom: 2,
                paddingLeft: 2,
                paddingRight: 2,
            }}>
                <SearchBar onChange={() => { }} value={''} />
            </Box> */}
            {/* Forms list and currnet form */}
            <Grid container spacing={0}>
                {/* List of settings forms, which is collapsible only on mobile */}
                <Grid item xs={12} md={6}>
                    {isMobile && <Box onClick={toggleList} sx={{
                        display: 'flex',
                        justifyContent: 'center',
                        alignItems: 'center',
                        marginBottom: 1,
                        cursor: 'pointer',
                        ...noSelect,
                    }}>
                        {isListOpen ? <ExpandLessIcon fill={palette.background.textPrimary} /> : <ExpandMoreIcon fill={palette.background.textPrimary} />}
                        <Typography variant="h6" sx={{ marginLeft: 1 }}>Settings</Typography>
                    </Box>}
                    <Collapse in={!isMobile || isListOpen}>
                        <List>
                            {listItems}
                        </List>
                    </Collapse>
                </Grid>
                {/* Current form */}
                <Grid item xs={12} md={6}>
                    {isMobile && <Divider />}
                    {loading ? <CircularProgress color="secondary" /> : mainContent}
                </Grid>
            </Grid>
        </PageContainer>
    )
}
