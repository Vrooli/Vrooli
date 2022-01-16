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
import {
    Email as EmailIcon,
    GitHub as GitHubIcon,
    Twitter as TwitterIcon,
    SvgIconComponent,
} from '@mui/icons-material';
import { CopyrightBreadcrumbs } from 'components';
import { useLocation } from 'wouter';
import { openLink } from 'utils';

export const Footer = () => {
    const [, setLocation] = useLocation();

    const contactLinks: Array<[string, string, string, string, SvgIconComponent]> = [
        ['contact-twitter', 'Find us on Twitter', SOCIALS.Twitter, 'Twitter', TwitterIcon],
        ['contact-email', 'Have a question or feedback? Email us!', EMAIL.Link, 'Email Us', EmailIcon],
        ['contact-github', 'Check out the source code, or contribute :)', SOCIALS.GitHub, 'Source Code', GitHubIcon],
    ]

    return (
        <Box 
            overflow="hidden" 
            position="relative" 
            paddingBottom="7vh"
            sx={{
                backgroundColor: (t) => t.palette.primary.dark,
                color: (t) => t.palette.primary.contrastText,
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
                                        <Icon sx={{fill: (t) => t.palette.primary.contrastText}} ></Icon>
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
