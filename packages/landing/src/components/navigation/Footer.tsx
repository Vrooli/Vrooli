import { APP_LINKS, LANDING_LINKS, WEBSITE } from '@local/shared';
import { makeStyles } from '@material-ui/styles';
import { SvgIconTypeMap, useTheme } from '@material-ui/core';
import { List, ListItem, ListItemButton, ListItemIcon, ListItemText, Grid, Tooltip, Theme } from '@material-ui/core';
import {
    Email as EmailIcon,
    GitHub as GitHubIcon,
    Twitter as TwitterIcon,
} from '@material-ui/icons';
import { CopyrightBreadcrumbs } from 'components';
import { useHistory } from 'react-router';
import { EMAIL, SOCIALS } from '@local/shared';
import { OverridableComponent } from '@material-ui/core/OverridableComponent';
import { openLink } from 'utils';

const useStyles = makeStyles((theme: Theme) => ({
    root: {
        overflow: 'hidden',
        backgroundColor: theme.palette.primary.dark,
        color: theme.palette.primary.contrastText,
        position: 'relative',
        paddingBottom: '7vh',
    },
    upper: {
        textTransform: 'uppercase',
    },
    imageContainer: {
        maxWidth: '33vw',
        padding: 10,
    },
    image: {
        maxWidth: '100%',
        maxHeight: 200,
        background: theme.palette.primary.contrastText,
    },
    icon: {
        fill: theme.palette.primary.contrastText,
    },
    copyright: {
        color: theme.palette.primary.contrastText,
    },
}));

export const Footer = () => {
    const classes = useStyles();
    const history = useHistory();
    const theme = useTheme();

    const contactLinks: Array<[string, string, string, string, OverridableComponent<SvgIconTypeMap<{}, "svg">>]> = [
        ['contact-twitter', 'Find us on Twitter', SOCIALS.Twitter, 'Twitter', TwitterIcon],
        ['contact-email', 'Have a question or feedback? Email us!', EMAIL.Link, 'Email Us', EmailIcon],
        ['contact-github', 'Check out the source code, or contribute :)', SOCIALS.GitHub, 'Source Code', GitHubIcon],
    ]

    return (
        <div className={classes.root}>
            <Grid container justifyContent='center' spacing={1}>
                <Grid item xs={12} sm={6}>
                <List component="nav">
                        <ListItem component="h3" >
                            <ListItemText className={classes.upper} primary="Resources" />
                        </ListItem>
                        <ListItemButton component="a" onClick={() => openLink(history, LANDING_LINKS.About)} >
                            <ListItemText primary="About Us" />
                        </ListItemButton>
                        <ListItemButton component="a" onClick={() => openLink(history, `app.${WEBSITE}${APP_LINKS.Stats}`)} >
                            <ListItemText primary="View Stats" />
                        </ListItemButton>
                    </List>
                </Grid>
                <Grid item xs={12} sm={6}>
                    <List component="nav">
                        <ListItem component="h3" >
                            <ListItemText className={classes.upper} primary="Contact" />
                        </ListItem>
                        {contactLinks.map(([label, tooltip, src, text, Icon], key) => (
                            <Tooltip key={key} title={tooltip} placement="left">
                                <ListItemButton aria-label={label} onClick={() => window.open(src, '_blank', 'noopener,noreferrer')}>
                                    <ListItemIcon>
                                        <Icon className={classes.icon} ></Icon>
                                    </ListItemIcon>
                                    <ListItemText primary={text} />
                                </ListItemButton>
                            </Tooltip>
                        ))}
                    </List>
                </Grid>
            </Grid>
            <CopyrightBreadcrumbs className={classes.copyright} textColor={theme.palette.primary.contrastText} />
        </div>
    );
}
