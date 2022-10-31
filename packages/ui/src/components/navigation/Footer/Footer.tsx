import { APP_LINKS, LANDING_LINKS, LANDING_URL, SOCIALS } from '@shared/consts';
import {
    Box,
    List,
    ListItem,
    ListItemIcon,
    ListItemText,
    Grid,
    Tooltip,
    useTheme,
} from '@mui/material';
import { InfoIcon, DiscordIcon, GitHubIcon, StatsIcon, SvgComponent, TwitterIcon } from '@shared/icons';
import { CopyrightBreadcrumbs } from 'components';
import { useLocation } from '@shared/route';
import { openLink } from 'utils';
import { useMemo } from 'react';

const contactLinks: [string, string, string, string, SvgComponent][] = [
    ['contact-twitter', 'Find us on Twitter', SOCIALS.Twitter, 'Twitter', TwitterIcon],
    ['contact-discord', 'Have a question or feedback? Post it on our Discord!', SOCIALS.Discord, 'Join our Discord', DiscordIcon],
    ['contact-github', 'Check out the source code, or contribute :)', SOCIALS.GitHub, 'Source Code', GitHubIcon],
]

const aboutUsLink = `${LANDING_URL}${LANDING_LINKS.AboutUs}`
const viewStatsLink = APP_LINKS.Stats

export const Footer = () => {
    const { palette } = useTheme();
    const [pathname, setLocation] = useLocation();
    // Hides footer on certain pages (e.g. /routine)
    const showFooter = useMemo(() => {
        const disableList = [APP_LINKS.Routine];
        return !disableList.some(disable => pathname.startsWith(disable));
    }, [pathname]);

    return (
        <Box
            display={showFooter ? 'block' : 'none'}
            overflow="hidden"
            position="relative"
            // safe-area-inset-bottom is the iOS navigation bar
            paddingBottom='calc(64px + env(safe-area-inset-bottom))'
            sx={{
                backgroundColor: palette.primary.dark,
                color: palette.primary.contrastText,
                zIndex: 2,
            }}
        >
            <Grid container justifyContent='center' spacing={1}>
                <Grid item xs={12} sm={6}>
                    <List component="nav">
                        <ListItem component="h3" >
                            <ListItemText primary="Resources" sx={{ textTransform: 'uppercase' }} />
                        </ListItem>
                        <ListItem
                            component="a"
                            href={aboutUsLink}
                            onClick={(e) => { e.preventDefault(); openLink(setLocation, aboutUsLink) }}
                            sx={{ padding: 2 }}
                        >
                            <ListItemIcon>
                                <InfoIcon fill={palette.primary.contrastText} />
                            </ListItemIcon>
                            <ListItemText primary="About Us" sx={{ color: palette.primary.contrastText }} />
                        </ListItem>
                        <ListItem
                            component="a"
                            href={viewStatsLink}
                            onClick={(e) => { e.preventDefault(); openLink(setLocation, viewStatsLink) }}
                            sx={{ padding: 2 }}
                        >
                            <ListItemIcon>
                                <StatsIcon fill={palette.primary.contrastText} />
                            </ListItemIcon>
                            <ListItemText primary="View Stats" sx={{ color: palette.primary.contrastText }} />
                        </ListItem>
                    </List>
                </Grid>
                <Grid item xs={12} sm={6}>
                    <List component="nav">
                        <ListItem component="h3" >
                            <ListItemText primary="Contact" sx={{ textTransform: 'uppercase' }} />
                        </ListItem>
                        {contactLinks.map(([label, tooltip, src, text, Icon], key) => (
                            <Tooltip key={key} title={tooltip} placement="left">
                                <ListItem
                                    aria-label={label}
                                    component="a"
                                    href={src}
                                    onClick={(e) => { e.preventDefault(); openLink(setLocation, src) }}
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
}
