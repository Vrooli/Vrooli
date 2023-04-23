import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { LINKS, SOCIALS } from "@local/consts";
import { ArticleIcon, DiscordIcon, GitHubIcon, InfoIcon, StatsIcon, TwitterIcon } from "@local/icons";
import { BottomNavigation, BottomNavigationAction, Box, Tooltip, Typography, useTheme } from "@mui/material";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { noSelect } from "../../../styles";
import { openLink, useLocation } from "../../../utils/route";
import { CopyrightBreadcrumbs } from "../../breadcrumbs/CopyrightBreadcrumbs/CopyrightBreadcrumbs";
import { ColorIconButton } from "../../buttons/ColorIconButton/ColorIconButton";
export const ContactInfo = ({ sx, ...props }) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const { additionalInfo, contactInfo } = useMemo(() => {
        return {
            additionalInfo: [
                [t("AboutUs"), "About", LINKS.About, InfoIcon],
                [t("DocumentationShort"), "Docs", "https://docs.vrooli.com", ArticleIcon],
                [t("StatisticsShort"), "Stats", LINKS.Stats, StatsIcon],
            ],
            contactInfo: [
                [t("ContactHelpTwitter"), "Twitter", SOCIALS.Twitter, TwitterIcon],
                [t("ContactHelpDiscord"), "Discord", SOCIALS.Discord, DiscordIcon],
                [t("ContactHelpCode"), "Code", SOCIALS.GitHub, GitHubIcon],
            ],
        };
    }, [t]);
    const handleLink = (e, link) => {
        e.preventDefault();
        openLink(setLocation, link);
    };
    return (_jsxs(Box, { sx: {
            minWidth: "fit-content",
            height: "fit-content",
            background: palette.background.default,
            padding: 1,
            ...(sx ?? {}),
        }, ...props, children: [_jsx(Typography, { variant: "h6", textAlign: "center", color: palette.background.textPrimary, sx: { ...noSelect }, children: t("FindUsOn") }), _jsx(BottomNavigation, { showLabels: true, sx: {
                    alignItems: "baseline",
                    background: "transparent",
                    height: "fit-content",
                    padding: 1,
                    marginBottom: 2,
                }, children: contactInfo.map(([tooltip, label, link, Icon], index) => (_jsx(Tooltip, { title: tooltip, placement: "top", children: _jsx(BottomNavigationAction, { label: label, onClick: (e) => { e.preventDefault(); handleLink(e, link); }, href: link, icon: _jsx(ColorIconButton, { background: palette.secondary.main, children: _jsx(Icon, { fill: palette.secondary.contrastText }) }), sx: {
                            alignItems: "center",
                            color: palette.background.textPrimary,
                            overflowWrap: "anywhere",
                        } }) }, `contact-info-button-${index}`))) }), _jsx(Typography, { variant: "h6", textAlign: "center", color: palette.background.textPrimary, sx: { ...noSelect }, children: t("AdditionalResources") }), _jsx(BottomNavigation, { showLabels: true, sx: {
                    alignItems: "baseline",
                    background: "transparent",
                    height: "fit-content",
                    padding: 1,
                }, children: additionalInfo.map(([tooltip, label, link, Icon], index) => (_jsx(Tooltip, { title: tooltip, placement: "top", children: _jsx(BottomNavigationAction, { label: label, onClick: (e) => { e.preventDefault(); handleLink(e, link); }, href: link, icon: _jsx(ColorIconButton, { background: palette.secondary.main, children: _jsx(Icon, { fill: palette.secondary.contrastText }) }), sx: {
                            alignItems: "center",
                            color: palette.background.textPrimary,
                            overflowWrap: "anywhere",
                        } }) }, `additional-info-button-${index}`))) }), _jsx(CopyrightBreadcrumbs, { sx: { color: palette.background.textPrimary } })] }));
};
//# sourceMappingURL=ContactInfo.js.map