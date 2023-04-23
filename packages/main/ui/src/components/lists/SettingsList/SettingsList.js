import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { LINKS } from "@local/consts";
import { Box, List, ListItem, ListItemIcon, ListItemText, useTheme } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useWindowSize } from "../../../utils/hooks/useWindowSize";
import { useLocation } from "../../../utils/route";
import { accountSettingsData, displaySettingsData } from "../../../views/settings";
export const SettingsList = () => {
    const { breakpoints, palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const onSelect = useCallback((newValue) => {
        if (!newValue)
            return;
        setLocation(LINKS[newValue]);
    }, [setLocation]);
    const [accountListOpen, setAccountListOpen] = useState(false);
    const toggleAccountList = useCallback(() => setAccountListOpen(!accountListOpen), [accountListOpen]);
    const accountList = useMemo(() => {
        const isSelected = (link) => window.location.pathname.includes(link);
        return Object.entries(accountSettingsData).map(([_, { title, link, Icon }], index) => (_jsxs(ListItem, { button: true, onClick: () => onSelect(link), sx: {
                transition: "brightness 0.2s ease-in-out",
                background: isSelected(link) ? palette.primary.main : "transparent",
                color: isSelected(link) ? palette.primary.contrastText : palette.background.textPrimary,
            }, children: [_jsx(ListItemIcon, { children: _jsx(Icon, { fill: isSelected(link) ? palette.primary.contrastText : palette.background.textSecondary }) }), _jsx(ListItemText, { primary: t(title, { count: 2 }) })] }, index)));
    }, [onSelect, palette.background.textPrimary, palette.background.textSecondary, palette.primary.contrastText, palette.primary.main, t]);
    const [displayListOpen, setDisplayListOpen] = useState(false);
    const toggleDisplayList = useCallback(() => setDisplayListOpen(!displayListOpen), [displayListOpen]);
    const displayList = useMemo(() => {
        const isSelected = (link) => window.location.pathname.includes(link);
        return Object.entries(displaySettingsData).map(([_, { title, link, Icon }], index) => (_jsxs(ListItem, { button: true, onClick: () => onSelect(link), sx: {
                transition: "brightness 0.2s ease-in-out",
                background: isSelected(link) ? palette.primary.main : "transparent",
                color: isSelected(link) ? palette.primary.contrastText : palette.background.textPrimary,
            }, children: [_jsx(ListItemIcon, { children: _jsx(Icon, { fill: isSelected(link) ? palette.primary.contrastText : palette.background.textSecondary }) }), _jsx(ListItemText, { primary: t(title, { count: 2 }) })] }, index)));
    }, [onSelect, palette.background.textPrimary, palette.background.textSecondary, palette.primary.contrastText, palette.primary.main, t]);
    if (isMobile)
        return null;
    return (_jsx(Box, { sx: {
            width: "min(100%, 300px)",
            marginLeft: "auto",
            borderRight: { xs: "none", md: "1px solid" },
        }, children: _jsxs(List, { children: [_jsx(ListItem, { onClick: toggleAccountList, children: _jsx(ListItemText, { primary: t("Account") }) }), _jsx(List, { component: "div", children: accountList }), _jsx(ListItem, { onClick: toggleDisplayList, children: _jsx(ListItemText, { primary: t("Display") }) }), _jsx(List, { component: "div", children: displayList })] }) }));
};
//# sourceMappingURL=SettingsList.js.map