import Box from "@mui/material/Box";
import List from "@mui/material/List";
import ListItem from "@mui/material/ListItem";
import ListItemIcon from "@mui/material/ListItemIcon";
import ListItemText from "@mui/material/ListItemText";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { styled, useTheme } from "@mui/material";
import { LINKS, SOCIALS, type TranslationKeyCommon } from "@vrooli/shared";
import { useTranslation } from "react-i18next";
import { Icon, IconCommon, type IconInfo } from "../../icons/Icons.js";
import { openLink } from "../../route/openLink.js";
import { useLocation } from "../../route/router.js";
import { pagePaddingBottom } from "../../styles.js";
import { getDeviceInfo } from "../../utils/display/device.js";
import { CopyrightBreadcrumbs } from "../breadcrumbs/CopyrightBreadcrumbs.js";

/** aria-label, tooltip, link, displayed text, icon */
const contactLinks: [string, TranslationKeyCommon, string, TranslationKeyCommon, IconInfo][] = [
    ["contact-x", "ContactHelpX", SOCIALS.X, "X", { name: "X", type: "Service" }],
    ["contact-github", "ContactHelpCode", SOCIALS.GitHub, "SourceCode", { name: "GitHub", type: "Service" }],
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

    function toAboutUs(event: React.MouseEvent<HTMLAnchorElement>) {
        event.preventDefault();
        openLink(setLocation, aboutUsLink);
    }
    function toViewStats(event: React.MouseEvent<HTMLAnchorElement>) {
        event.preventDefault();
        openLink(setLocation, viewStatsLink);
    }

    // Dont' display footer when app is running standalone
    const { isStandalone } = getDeviceInfo();
    if (isStandalone) return null;
    return (
        <StyledFooter id="footer" component="footer">
            <Box display="flex" justifyContent="space-around" width="100%">
                <Box sx={footerColumn}>
                    <Typography variant="body1" sx={{ textTransform: "uppercase" }}>{t("Resource", { count: 2 })}</Typography>
                    <List component="nav" dense>
                        <ListItem
                            aria-label={t("AboutUs")}
                            component="a"
                            href={aboutUsLink}
                            onClick={toAboutUs}
                            sx={listItemStyle}
                        >
                            <ListItemIcon>
                                <IconCommon
                                    decorative
                                    fill={palette.primary.contrastText}
                                    name="Info"
                                />
                            </ListItemIcon>
                            <ListItemText primary={t("AboutUs")} sx={{ color: palette.primary.contrastText }} />
                        </ListItem>
                        <ListItem
                            aria-label={t("StatisticsShort")}
                            component="a"
                            href={viewStatsLink}
                            onClick={toViewStats}
                            sx={listItemStyle}
                        >
                            <ListItemIcon>
                                <IconCommon
                                    decorative
                                    fill={palette.primary.contrastText}
                                    name="Stats"
                                />
                            </ListItemIcon>
                            <ListItemText primary={t("StatisticsShort")} sx={{ color: palette.primary.contrastText }} />
                        </ListItem>
                    </List>
                </Box>
                <Box sx={footerColumn}>
                    <Typography variant="body1" sx={{ textTransform: "uppercase" }}>{t("Contact")}</Typography>
                    <List component="nav" dense>
                        {contactLinks.map(([label, tooltip, src, text, iconInfo], key) => {
                            function toLink(event: React.MouseEvent<HTMLAnchorElement>) {
                                event.preventDefault();
                                openLink(setLocation, src);
                            }

                            return (
                                <Tooltip key={key} title={t(tooltip)} placement="left">
                                    <ListItem
                                        aria-label={label}
                                        component="a"
                                        href={src}
                                        onClick={toLink}
                                        sx={listItemStyle}
                                    >
                                        <ListItemIcon>
                                            <Icon
                                                decorative
                                                fill={palette.primary.contrastText}
                                                info={iconInfo}
                                            />
                                        </ListItemIcon>
                                        <ListItemText primary={t(text)} sx={{ color: palette.primary.contrastText }} />
                                    </ListItem>
                                </Tooltip>
                            );
                        })}
                    </List>
                </Box>
            </Box>
            <CopyrightBreadcrumbs textColor="primary.contrastText" />
            {/* Hidden div under the footer for bottom overscroll color */}
            <Box
                bgcolor={palette.primary.dark}
                height="50vh"
                position="absolute"
                left={0}
                right={0}
                bottom={0}
                zIndex={-2} // Below the main content, but above the page's hidden div
            />
        </StyledFooter>
    );
}
