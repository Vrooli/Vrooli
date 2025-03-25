import { LINKS, SOCIALS, TranslationKeyCommon } from "@local/shared";
import { Box, List, ListItem, ListItemIcon, ListItemText, Tooltip, Typography, styled, useTheme } from "@mui/material";
import { useTranslation } from "react-i18next";
import { GitHubIcon, InfoIcon, StatsIcon, XIcon } from "../../icons/common.js";
import { openLink } from "../../route/openLink.js";
import { useLocation } from "../../route/router.js";
import { pagePaddingBottom } from "../../styles.js";
import { SvgComponent } from "../../types.js";
import { getDeviceInfo } from "../../utils/display/device.js";
import { CopyrightBreadcrumbs } from "../breadcrumbs/CopyrightBreadcrumbs.js";

/** aria-label, tooltip, link, displayed text, icon */
const contactLinks: [string, TranslationKeyCommon, string, TranslationKeyCommon, SvgComponent][] = [
    ["contact-x", "ContactHelpX", SOCIALS.X, "X", XIcon],
    // ["contact-discord", "ContactHelpDiscord", SOCIALS.Discord, "Discord", DiscordIcon],
    ["contact-github", "ContactHelpCode", SOCIALS.GitHub, "SourceCode", GitHubIcon],
];

const aboutUsLink = LINKS.About;
const viewStatsLink = LINKS.Stats;

const StyledFooter = styled(Box)(({ theme }) => ({
    display: "block",
    position: "relative",
    overflow: "hidden",
    backgroundColor: theme.palette.primary.dark,
    color: theme.palette.primary.contrastText,
    paddingBottom: pagePaddingBottom,
    zIndex: 5,
    "@media print": {
        display: "none",
    },
}));

const listItemStyle = {
    padding: 1,
} as const;
const footerColumn = {
    display: "flex",
    flexDirection: "column",
    padding: 2,
};

export function Footer() {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    // Dont' display footer when app is running standalone
    const { isStandalone } = getDeviceInfo();
    if (isStandalone) return null;
    return (
        <StyledFooter id="footer" component="footer">
            <Box display="flex" justifyContent="space-around" sx={{ width: "100%" }}>
                <Box sx={footerColumn}>
                    <Typography variant="body1" sx={{ textTransform: "uppercase" }}>{t("Resource", { count: 2 })}</Typography>
                    <List component="nav" dense>
                        <ListItem
                            component="a"
                            href={aboutUsLink}
                            onClick={(e) => { e.preventDefault(); openLink(setLocation, aboutUsLink); }}
                            sx={listItemStyle}
                        >
                            <ListItemIcon>
                                <InfoIcon fill={palette.primary.contrastText} />
                            </ListItemIcon>
                            <ListItemText primary={t("AboutUs")} sx={{ color: palette.primary.contrastText }} />
                        </ListItem>
                        <ListItem
                            component="a"
                            href={viewStatsLink}
                            onClick={(e) => { e.preventDefault(); openLink(setLocation, viewStatsLink); }}
                            sx={listItemStyle}
                        >
                            <ListItemIcon>
                                <StatsIcon fill={palette.primary.contrastText} />
                            </ListItemIcon>
                            <ListItemText primary={t("StatisticsShort")} sx={{ color: palette.primary.contrastText }} />
                        </ListItem>
                    </List>
                </Box>
                <Box sx={footerColumn}>
                    <Typography variant="body1" sx={{ textTransform: "uppercase" }}>{t("Contact")}</Typography>
                    <List component="nav" dense>
                        {contactLinks.map(([label, tooltip, src, text, Icon], key) => (
                            <Tooltip key={key} title={t(tooltip)} placement="left">
                                <ListItem
                                    aria-label={label}
                                    component="a"
                                    href={src}
                                    onClick={(e) => { e.preventDefault(); openLink(setLocation, src); }}
                                    sx={listItemStyle}
                                >
                                    <ListItemIcon>
                                        <Icon fill={palette.primary.contrastText} />
                                    </ListItemIcon>
                                    <ListItemText primary={t(text)} sx={{ color: palette.primary.contrastText }} />
                                </ListItem>
                            </Tooltip>
                        ))}
                    </List>
                </Box>
            </Box>
            <CopyrightBreadcrumbs sx={{ color: palette.primary.contrastText }} />
            {/* Hidden div under the footer for bottom overscroll color */}
            <Box sx={{
                backgroundColor: palette.primary.dark,
                height: "50vh",
                position: "absolute",
                left: 0,
                right: 0,
                bottom: 0,
                zIndex: -2, // Below the main content, but above the page's hidden div
            }} />
        </StyledFooter>
    );
}
