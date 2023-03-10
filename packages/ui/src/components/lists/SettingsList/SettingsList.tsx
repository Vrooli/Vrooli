import { Box, List, ListItem, ListItemButton, ListItemIcon, ListItemText, useTheme } from '@mui/material';
import { APP_LINKS } from '@shared/consts';
import { useLocation } from '@shared/route';
import { useCallback, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useWindowSize } from 'utils';
import { accountSettingsData, displaySettingsData } from 'views/settings';

export const SettingsList = () => {
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
        return Object.entries(accountSettingsData).map(([_, { title, link, Icon }], index) => (
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
                <ListItemText primary={t(title, { count: 2 })} />
            </ListItem>
        ))
    }, [onSelect, palette.background.textPrimary, palette.background.textSecondary, palette.primary.contrastText, palette.primary.main, t]);

    const [displayListOpen, setDisplayListOpen] = useState(false);
    const toggleDisplayList = useCallback(() => setDisplayListOpen(!displayListOpen), [displayListOpen]);
    const displayList = useMemo(() => {
        const isSelected = (link: string) => window.location.pathname.includes(link);
        return Object.entries(displaySettingsData).map(([_, { title, link, Icon }], index) => (
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
                <ListItemText primary={t(title, { count: 2 })} />
            </ListItem>
        ))
    }, [onSelect, palette.background.textPrimary, palette.background.textSecondary, palette.primary.contrastText, palette.primary.main, t]);


    if (isMobile) return null;
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