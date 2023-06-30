import { DiscordIcon, GitHubIcon, InfoIcon, LINKS, openLink, SOCIALS, StatsIcon, SvgComponent, TwitterIcon, useLocation } from "@local/shared";
import { Box, Grid, List, ListItem, ListItemIcon, ListItemText, Tooltip, useTheme } from "@mui/material";
import { CopyrightBreadcrumbs } from "components/breadcrumbs/CopyrightBreadcrumbs/CopyrightBreadcrumbs";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { getDeviceInfo } from "utils/display/device";

const contactLinks: [string, string, string, string, SvgComponent][] = [
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
    // Hides footer on certain pages (e.g. /routine)
    const showFooter = useMemo(() => {
        const disableList = [LINKS.Routine];
        return !disableList.some(disable => pathname.startsWith(disable));
    }, [pathname]);

    // Dont' display footer when app is running standalone
    const { isStandalone } = getDeviceInfo();
    if (isStandalone) return null;
    return (
        <Box
            display={showFooter ? "block" : "none"}
            overflow="hidden"
            position="relative"
            sx={{
                backgroundColor: palette.primary.dark,
                color: palette.primary.contrastText,
                paddingBottom: {
                    xs: "calc(64px + env(safe-area-inset-bottom))",
                    md: "env(safe-area-inset-bottom)",
                },
                zIndex: 5,
                "@media print": {
                    display: "none",
                },
            }}
        >
            <Grid container justifyContent='center' spacing={1}>
                <Grid item xs={12} sm={6}>
                    <List component="nav">
                        <ListItem component="h3" >
                            <ListItemText primary={t("Resource", { count: 2 })} sx={{ textTransform: "uppercase" }} />
                        </ListItem>
                        <ListItem
                            component="a"
                            href={aboutUsLink}
                            onClick={(e) => { e.preventDefault(); openLink(setLocation, aboutUsLink); }}
                            sx={{ padding: 2 }}
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
                            sx={{ padding: 2 }}
                        >
                            <ListItemIcon>
                                <StatsIcon fill={palette.primary.contrastText} />
                            </ListItemIcon>
                            <ListItemText primary={t("StatisticsShort")} sx={{ color: palette.primary.contrastText }} />
                        </ListItem>
                    </List>
                </Grid>
                <Grid item xs={12} sm={6}>
                    <List component="nav">
                        <ListItem component="h3" >
                            <ListItemText primary={t("Contact")} sx={{ textTransform: "uppercase" }} />
                        </ListItem>
                        {contactLinks.map(([label, tooltip, src, text, Icon], key) => (
                            <Tooltip key={key} title={tooltip} placement="left">
                                <ListItem
                                    aria-label={label}
                                    component="a"
                                    href={src}
                                    onClick={(e) => { e.preventDefault(); openLink(setLocation, src); }}
                                    sx={{ padding: 2 }}
                                >
                                    <ListItemIcon>
                                        <Icon fill={palette.primary.contrastText} />
                                    </ListItemIcon>
                                    <ListItemText primary={text} sx={{ color: palette.primary.contrastText }} />
                                </ListItem>
                            </Tooltip>
                        ))}
                    </List>
                </Grid>
            </Grid>
            <CopyrightBreadcrumbs sx={{ color: palette.primary.contrastText }} />
        </Box>
    );
};
