import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { LINKS, SOCIALS } from "@local/consts";
import { DiscordIcon, GitHubIcon, InfoIcon, StatsIcon, TwitterIcon } from "@local/icons";
import { Box, Grid, List, ListItem, ListItemIcon, ListItemText, Tooltip, useTheme } from "@mui/material";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { getDeviceInfo } from "../../../utils/display/device";
import { openLink, useLocation } from "../../../utils/route";
import { CopyrightBreadcrumbs } from "../../breadcrumbs/CopyrightBreadcrumbs/CopyrightBreadcrumbs";
const contactLinks = [
    ["contact-twitter", "Find us on Twitter", SOCIALS.Twitter, "Twitter", TwitterIcon],
    ["contact-discord", "Have a question or feedback? Post it on our Discord!", SOCIALS.Discord, "Join our Discord", DiscordIcon],
    ["contact-github", "Check out the source code, or contribute :)", SOCIALS.GitHub, "Source Code", GitHubIcon],
];
const aboutUsLink = LINKS.About;
const viewStatsLink = LINKS.Stats;
export const Footer = () => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [pathname, setLocation] = useLocation();
    const showFooter = useMemo(() => {
        const disableList = [LINKS.Routine];
        return !disableList.some(disable => pathname.startsWith(disable));
    }, [pathname]);
    const { isStandalone } = getDeviceInfo();
    if (isStandalone)
        return null;
    return (_jsxs(Box, { display: showFooter ? "block" : "none", overflow: "hidden", position: "relative", paddingBottom: 'calc(64px + env(safe-area-inset-bottom))', sx: {
            backgroundColor: palette.primary.dark,
            color: palette.primary.contrastText,
            zIndex: 5,
        }, children: [_jsxs(Grid, { container: true, justifyContent: 'center', spacing: 1, children: [_jsx(Grid, { item: true, xs: 12, sm: 6, children: _jsxs(List, { component: "nav", children: [_jsx(ListItem, { component: "h3", children: _jsx(ListItemText, { primary: t("Resource", { count: 2 }), sx: { textTransform: "uppercase" } }) }), _jsxs(ListItem, { component: "a", href: aboutUsLink, onClick: (e) => { e.preventDefault(); openLink(setLocation, aboutUsLink); }, sx: { padding: 2 }, children: [_jsx(ListItemIcon, { children: _jsx(InfoIcon, { fill: palette.primary.contrastText }) }), _jsx(ListItemText, { primary: t("AboutUs"), sx: { color: palette.primary.contrastText } })] }), _jsxs(ListItem, { component: "a", href: viewStatsLink, onClick: (e) => { e.preventDefault(); openLink(setLocation, viewStatsLink); }, sx: { padding: 2 }, children: [_jsx(ListItemIcon, { children: _jsx(StatsIcon, { fill: palette.primary.contrastText }) }), _jsx(ListItemText, { primary: t("StatisticsShort"), sx: { color: palette.primary.contrastText } })] })] }) }), _jsx(Grid, { item: true, xs: 12, sm: 6, children: _jsxs(List, { component: "nav", children: [_jsx(ListItem, { component: "h3", children: _jsx(ListItemText, { primary: t("Contact"), sx: { textTransform: "uppercase" } }) }), contactLinks.map(([label, tooltip, src, text, Icon], key) => (_jsx(Tooltip, { title: tooltip, placement: "left", children: _jsxs(ListItem, { "aria-label": label, component: "a", href: src, onClick: (e) => { e.preventDefault(); openLink(setLocation, src); }, sx: { padding: 2 }, children: [_jsx(ListItemIcon, { children: _jsx(Icon, { fill: palette.primary.contrastText }) }), _jsx(ListItemText, { primary: text, sx: { color: palette.primary.contrastText } })] }) }, key)))] }) })] }), _jsx(CopyrightBreadcrumbs, { sx: { color: palette.primary.contrastText } })] }));
};
//# sourceMappingURL=Footer.js.map