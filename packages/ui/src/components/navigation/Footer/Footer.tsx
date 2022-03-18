import { APP_LINKS, EMAIL, LANDING_LINKS, LANDING_URL, SOCIALS } from '@local/shared';
import {
    Box, 
    List, 
    ListItem, 
    ListItemButton, 
    ListItemIcon, 
    ListItemText, 
    Grid, 
    Tooltip,
} from '@mui/material';
import { DiscordIcon, GitHubIcon, TwitterIcon } from 'assets/img';
import { CopyrightBreadcrumbs } from 'components';
import { useLocation } from 'wouter';
import { openLink } from 'utils';
import { useMemo } from 'react';

export const Footer = () => {
    const [pathname, setLocation] = useLocation();
    // Hides footer on certain pages (e.g. /build)
    const showFooter = useMemo(() => {
        const disableList = [APP_LINKS.Build];
        return !disableList.some(disable => pathname.startsWith(disable));
    }, [pathname]);

    const contactLinks: Array<[string, string, string, string, any]> = [
        ['contact-twitter', 'Find us on Twitter', SOCIALS.Twitter, 'Twitter', TwitterIcon],
        ['contact-discord', 'Have a question or feedback? Post it on our Discord!', SOCIALS.Discord, 'Join our Discord', DiscordIcon],
        ['contact-github', 'Check out the source code, or contribute :)', SOCIALS.GitHub, 'Source Code', GitHubIcon],
    ]

    return (
        <Box 
            display={showFooter ? 'block' : 'none'}
            overflow="hidden" 
            position="relative" 
            paddingBottom="7vh"
            sx={{
                backgroundColor: (t) => t.palette.primary.dark,
                color: (t) => t.palette.primary.contrastText,
                zIndex: 2,
            }}
        >
            <Grid container justifyContent='center' spacing={1}>
                <Grid item xs={12} sm={6}>
                    <List component="nav">
                        <ListItem component="h3" >
                            <ListItemText primary="Resources" sx={{textTransform: 'uppercase'}} />
                        </ListItem>
                        <ListItemButton component="a" onClick={() => openLink(setLocation, `${LANDING_URL}${LANDING_LINKS.About}`)} >
                            <ListItemText primary="About Us" />
                        </ListItemButton>
                        <ListItemButton component="a" onClick={() => openLink(setLocation, APP_LINKS.Stats)} >
                            <ListItemText primary="View Stats" />
                        </ListItemButton>
                    </List>
                </Grid>
                <Grid item xs={12} sm={6}>
                    <List component="nav">
                        <ListItem component="h3" >
                            <ListItemText primary="Contact" sx={{textTransform: 'uppercase'}} />
                        </ListItem>
                        {contactLinks.map(([label, tooltip, src, text, Icon], key) => (
                            <Tooltip key={key} title={tooltip} placement="left">
                                <ListItemButton aria-label={label} onClick={() => openLink(setLocation, src)}>
                                    <ListItemIcon>
                                        <Icon fill="white" />
                                    </ListItemIcon>
                                    <ListItemText primary={text} />
                                </ListItemButton>
                            </Tooltip>
                        ))}
                    </List>
                </Grid>
            </Grid>
            <CopyrightBreadcrumbs sx={{color: (t) => t.palette.primary.contrastText}} />
        </Box>
    );
}
