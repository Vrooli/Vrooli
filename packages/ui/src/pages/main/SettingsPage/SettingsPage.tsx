import { Box, CircularProgress, Collapse, Divider, Grid, List, ListItem, ListItemIcon, ListItemText, Typography, useTheme } from '@mui/material';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { SettingsPageProps } from '../types';
import { useCustomLazyQuery } from 'api/hooks';
import { APP_LINKS, User } from '@shared/consts';
import { useLocation } from '@shared/route';
import { SettingsProfile } from 'components/views/SettingsProfile/SettingsProfile';
import { SettingsAuthentication } from 'components/views/SettingsAuthentication/SettingsAuthentication';
import { SettingsDisplay } from 'components/views/SettingsDisplay/SettingsDisplay';
import { SettingsNotifications } from 'components/views/SettingsNotifications/SettingsNotifications';
import { PreSearchItem, translateSearchItems, useReactSearch, useWindowSize } from 'utils';
import { ExpandLessIcon, ExpandMoreIcon, LightModeIcon, LockIcon, NotificationsCustomizedIcon, ProfileIcon, SvgComponent } from '@shared/icons';
import { PageContainer, SettingsSearchBar } from 'components';
import { getCurrentUser } from 'utils/authentication';
import { noSelect } from 'styles';
import { useTranslation } from 'react-i18next';
import { userProfile } from 'api/generated/endpoints/user_profile';
import { CommonKey } from '@shared/translations';

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

/**
 * Search bar options
 */
const searchItems: PreSearchItem[] = [
    {
        label: 'Profile',
        keywords: ['Bio', 'Handle', 'Name'],
        value: 'profile',
    },
    {
        label: 'Display',
        keywords: ['Theme', 'Light', 'Dark', 'Interests', 'Hidden', { key: 'Tag', count: 1 }, { key: 'Tag', count: 2 }, 'History'],
        value: 'display',
    },
    {
        label: 'Notification',
        labelArgs: { count: 2 },
        keywords: [{ key: 'Alert', count: 1 }, { key: 'Alert', count: 2 }, { key: 'PushNotification', count: 1 }, { key: 'PushNotification', count: 2 }],
        value: 'profile',
    },
    {
        label: 'Authentication',
        keywords: [{ key: 'Wallet', count: 1 }, { key: 'Wallet', count: 2 }, { key: 'Email', count: 1 }, { key: 'Email', count: 2 }, 'LogOut', 'Security'],
        value: 'authentication',
    },
]

const pageDisplayData: { [key in SettingsForm]: [CommonKey, SvgComponent] } = {
    'profile': ['Profile', ProfileIcon],
    'display': ['Display', LightModeIcon],
    'notifications': ['Notification', NotificationsCustomizedIcon],
    'authentication': ['Authentication', LockIcon],
}

export function SettingsPage({
    session,
}: SettingsPageProps) {
    const { t } = useTranslation();
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const [, setLocation] = useLocation();
    const [searchString, setSearchString] = useState<string>('');
    const [selectedPage, setSelectedPage] = useState<SettingsForm>('profile');
    const searchParams = useReactSearch();

    useEffect(() => {
        if (searchParams.page) setSelectedPage(searchParams.page as unknown as SettingsForm ?? 'profile');
        if (typeof searchParams.search === 'string') setSearchString(searchParams.search);
    }, [searchParams]);

    const updateSearch = useCallback((newValue: any) => { setSearchString(newValue) }, []);
    const onInputSelect = useCallback((newValue: any) => {
        if (!newValue) return;
        setSearchString(newValue.label);
        setSelectedPage(newValue.value as unknown as SettingsForm);
    }, []);

    const searchOptions = useMemo(() => translateSearchItems(searchItems, session), [session]);

    // Fetch profile data
    const [getData, { data, loading }] = useCustomLazyQuery<User, undefined>(userProfile, { errorPolicy: 'all' });
    useEffect(() => {
        if (getCurrentUser(session).id) getData();
    }, [getData, session])
    const [profile, setProfile] = useState<User | undefined>(undefined);
    useEffect(() => {
        if (data) setProfile(data);
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
                    <ListItemText primary={t(label, { count: 2 })} />
                </ListItem>
            )
        });
    }, [closeList, palette.background.textPrimary, palette.background.textSecondary, palette.primary.contrastText, palette.primary.main, selectedPage, setLocation, t]);

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
            {/* Search bar to find setting */}
            <Box sx={{
                width: 'min(100%, 700px)',
                margin: 'auto',
                marginTop: 2,
                marginBottom: 2,
                paddingLeft: 2,
                paddingRight: 2,
            }}>
                <SettingsSearchBar
                    value={searchString}
                    onChange={updateSearch}
                    onInputChange={onInputSelect}
                    options={searchOptions}
                    session={session}
                />
            </Box>
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
                        <Typography variant="h6" sx={{ marginLeft: 1 }}>{t(`Settings`)}</Typography>
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
