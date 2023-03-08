import { Box, List, ListItem, ListItemButton, ListItemIcon, ListItemText, useTheme } from '@mui/material';
import { APP_LINKS } from '@shared/consts';
import { HistoryIcon, LightModeIcon, LockIcon, NotificationsCustomizedIcon, ProfileIcon, SvgComponent, VisibleIcon } from '@shared/icons';
import { useLocation } from '@shared/route';
import { CommonKey } from '@shared/translations';
import { useCallback, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useWindowSize } from 'utils';
import { SettingsPageType } from 'views/settings/types';
import { SettingsListProps } from '../types';


type ListData = { [key in SettingsPageType]?: [CommonKey, keyof typeof APP_LINKS, SvgComponent] };

const accountListData: ListData = {
    Profile: ['Profile', 'SettingsProfile', ProfileIcon],
    Privacy: ['Privacy', 'SettingsPrivacy', VisibleIcon],
    Authentication: ['Authentication', 'SettingsAuthentication', LockIcon],
};

const displayListData: ListData = {
    Display: ['Display', 'SettingsDisplay', LightModeIcon],
    Notification: ['Notification', 'SettingsNotifications', NotificationsCustomizedIcon],
    Schedule: ['Schedule', 'SettingsSchedules', HistoryIcon],
};

export const SettingsList = ({
    showOnMobile = false,
}: SettingsListProps) => {
    const { breakpoints, palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    const onSelect = useCallback((newValue: any) => {
        if (!newValue) return;
        setLocation(APP_LINKS[newValue]);
    }, [setLocation]);

    const [accountListOpen, setAccountListOpen] = useState(false);
    const toggleAccountList = useCallback(() => setAccountListOpen(!accountListOpen), [accountListOpen]);
    const accountList = useMemo(() => {
        const isSelected = (link: string) => window.location.pathname.includes(link);
        return Object.entries(accountListData).map(([_, [label, link, Icon]], index) => (
            <ListItem
                button
                key={index}
                onClick={() => onSelect(link)}
                sx={{
                    transition: 'brightness 0.2s ease-in-out',
                    background: isSelected(link) ? palette.primary.main : 'transparent',
                    color: isSelected(link) ? palette.primary.contrastText : palette.background.textPrimary,
                }}
            >
                <ListItemIcon>
                    <Icon fill={isSelected(link) ? palette.primary.contrastText : palette.background.textSecondary} />
                </ListItemIcon>
                <ListItemText primary={t(label, { count: 2 })} />
            </ListItem>
        ))
    }, [onSelect, palette.background.textPrimary, palette.background.textSecondary, palette.primary.contrastText, palette.primary.main, t]);

    const [displayListOpen, setDisplayListOpen] = useState(false);
    const toggleDisplayList = useCallback(() => setDisplayListOpen(!displayListOpen), [displayListOpen]);
    const displayList = useMemo(() => {
        const isSelected = (link: string) => window.location.pathname.includes(link);
        return Object.entries(displayListData).map(([_, [label, link, Icon]], index) => (
            <ListItem
                button
                key={index}
                onClick={() => onSelect(link)}
                sx={{
                    transition: 'brightness 0.2s ease-in-out',
                    background: isSelected(link) ? palette.primary.main : 'transparent',
                    color: isSelected(link) ? palette.primary.contrastText : palette.background.textPrimary,
                }}
            >
                <ListItemIcon>
                    <Icon fill={isSelected(link) ? palette.primary.contrastText : palette.background.textSecondary} />
                </ListItemIcon>
                <ListItemText primary={t(label, { count: 2 })} />
            </ListItem>
        ))
    }, [onSelect, palette.background.textPrimary, palette.background.textSecondary, palette.primary.contrastText, palette.primary.main, t]);


    if (!showOnMobile && isMobile) return null;
    return (
        // Full width on mobile, and 500px on desktop. 
        <Box sx={{
            width: 'min(100%, 300px)',
            marginLeft: 'auto',
            borderRight: { xs: 'none', md: '1px solid' },
        }}>
            <List>
                {/* Account-related items */}
                <ListItemButton onClick={toggleAccountList}>
                    <ListItemText primary={t(`Account`)} />
                </ListItemButton>
                <List component="div">
                    {accountList}
                </List>
                {/* Display-related items */}
                <ListItemButton onClick={toggleDisplayList}>
                    <ListItemText primary={t(`Display`)} />
                </ListItemButton>
                <List component="div">
                    {displayList}
                </List>
            </List>
        </Box>
    )
}