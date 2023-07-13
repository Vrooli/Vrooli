import { LINKS, useLocation } from "@local/shared";
import { Box, List, ListItem, ListItemIcon, ListItemText, useTheme } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useWindowSize } from "utils/hooks/useWindowSize";
import { accountSettingsData, displaySettingsData } from "views/settings";

export const SettingsList = () => {
    const { breakpoints, palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    const onSelect = useCallback((link: LINKS) => {
        if (!link) return;
        setLocation(link);
    }, [setLocation]);

    const [accountListOpen, setAccountListOpen] = useState(false);
    const toggleAccountList = useCallback(() => setAccountListOpen(!accountListOpen), [accountListOpen]);
    const accountList = useMemo(() => {
        const isSelected = (link: LINKS) => window.location.pathname.includes(link);
        return Object.entries(accountSettingsData).map(([_, { title, link, Icon }], index) => (
            <ListItem
                button
                key={index}
                onClick={() => onSelect(link)}
                sx={{
                    transition: "brightness 0.2s ease-in-out",
                    background: isSelected(link) ? palette.primary.main : "transparent",
                    color: isSelected(link) ? palette.primary.contrastText : palette.background.textPrimary,
                }}
            >
                <ListItemIcon>
                    <Icon fill={isSelected(link) ? palette.primary.contrastText : palette.background.textSecondary} />
                </ListItemIcon>
                <ListItemText primary={t(title, { count: 2 })} />
            </ListItem>
        ));
    }, [onSelect, palette.background.textPrimary, palette.background.textSecondary, palette.primary.contrastText, palette.primary.main, t]);

    const [displayListOpen, setDisplayListOpen] = useState(false);
    const toggleDisplayList = useCallback(() => setDisplayListOpen(!displayListOpen), [displayListOpen]);
    const displayList = useMemo(() => {
        const isSelected = (link: LINKS) => window.location.pathname.includes(link);
        return Object.entries(displaySettingsData).map(([_, { title, link, Icon }], index) => (
            <ListItem
                button
                key={index}
                onClick={() => onSelect(link)}
                sx={{
                    transition: "brightness 0.2s ease-in-out",
                    background: isSelected(link) ? palette.primary.main : "transparent",
                    color: isSelected(link) ? palette.primary.contrastText : palette.background.textPrimary,
                }}
            >
                <ListItemIcon>
                    <Icon fill={isSelected(link) ? palette.primary.contrastText : palette.background.textSecondary} />
                </ListItemIcon>
                <ListItemText primary={t(title, { count: 2 })} />
            </ListItem>
        ));
    }, [onSelect, palette.background.textPrimary, palette.background.textSecondary, palette.primary.contrastText, palette.primary.main, t]);


    if (isMobile) return null;
    return (
        <Box sx={{
            width: "min(100%, 300px)",
            height: "fit-content",
            marginLeft: 0,
            marginTop: 4,
            background: palette.background.paper,
            boxShadow: 2,
            borderRadius: 2,
        }}>
            <List>
                {/* Account-related items */}
                <ListItem onClick={toggleAccountList}>
                    <ListItemText primary={t("Account")} />
                </ListItem>
                <List component="div">
                    {accountList}
                </List>
                {/* Display-related items */}
                <ListItem onClick={toggleDisplayList}>
                    <ListItemText primary={t("Display")} />
                </ListItem>
                <List component="div">
                    {displayList}
                </List>
            </List>
        </Box>
    );
};
