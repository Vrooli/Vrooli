import { CommonKey, LINKS, SOCIALS, uuidValidate } from "@local/shared";
import { Box, Grid, List, ListItem, ListItemIcon, ListItemText, Tooltip, useTheme } from "@mui/material";
import { CopyrightBreadcrumbs } from "components/breadcrumbs/CopyrightBreadcrumbs/CopyrightBreadcrumbs";
import { SessionContext } from "contexts/SessionContext";
import { GitHubIcon, InfoIcon, StatsIcon, XIcon } from "icons";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { openLink, useLocation } from "route";
import { SvgComponent } from "types";
import { getCurrentUser } from "utils/authentication/session";
import { getDeviceInfo } from "utils/display/device";

/** aria-label, tooltip, link, displayed text, icon */
const contactLinks: [string, CommonKey, string, CommonKey, SvgComponent][] = [
    ["contact-x", "ContactHelpX", SOCIALS.X, "X", XIcon],
    // ["contact-discord", "ContactHelpDiscord", SOCIALS.Discord, "Discord", DiscordIcon],
    ["contact-github", "ContactHelpCode", SOCIALS.GitHub, "SourceCode", GitHubIcon],
];

const aboutUsLink = LINKS.About;
const viewStatsLink = LINKS.Stats;

const pagesWithFooter: string[] = [LINKS.About, LINKS.Pro, LINKS.Privacy, LINKS.Signup, LINKS.Login, LINKS.ForgotPassword, LINKS.ResetPassword, LINKS.Terms];

export const Footer = () => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const session = useContext(SessionContext);
    const { id } = useMemo(() => getCurrentUser(session), [session]);

    // Dont' display footer when app is running standalone, 
    // when the page isn't one of the pages with a footer,
    // or when the user is at the home page ("/") and logged in (since this becomes the dashboard page)
    const { isStandalone } = getDeviceInfo();
    if (isStandalone) return null;
    if (window.location.pathname !== LINKS.Home && !pagesWithFooter.some((page) => window.location.pathname.startsWith(page))) return null;
    if (window.location.pathname === LINKS.Home && uuidValidate(id)) return null;
    return (
        <>
            <Box
                display={"block"}
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
                        <List component="nav" dense>
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
                        <List component="nav" dense>
                            <ListItem component="h3" >
                                <ListItemText primary={t("Contact")} sx={{ textTransform: "uppercase" }} />
                            </ListItem>
                            {contactLinks.map(([label, tooltip, src, text, Icon], key) => (
                                <Tooltip key={key} title={t(tooltip)} placement="left">
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
                                        <ListItemText primary={t(text)} sx={{ color: palette.primary.contrastText }} />
                                    </ListItem>
                                </Tooltip>
                            ))}
                        </List>
                    </Grid>
                </Grid>
                <CopyrightBreadcrumbs sx={{ color: palette.primary.contrastText }} />
            </Box>
            {/* Hidden div under the footer for bottom overscroll color */}
            <Box sx={{
                backgroundColor: palette.primary.dark,
                height: "50vh",
                position: "fixed",
                bottom: "0",
                width: "100%",
                zIndex: -2, // Below the main content, but above the page's hidden div
            }} />
        </>
    );
};
