import React from 'react';
import PropTypes from 'prop-types';
import { LINKS } from 'utils';
import { makeStyles } from '@material-ui/styles';
import { List, ListItem, ListItemIcon, ListItemText, Grid, Tooltip } from '@material-ui/core';
import {
    Email as EmailIcon,
    GitHub as GitHubIcon,
    Twitter as TwitterIcon,
} from '@material-ui/icons';
import { CopyrightBreadcrumbs } from 'components';
import { useTheme } from '@emotion/react';
import { useHistory } from 'react-router';

const useStyles = makeStyles((theme) => ({
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

function Footer({
    business
}) {
    const classes = useStyles();
    const history = useHistory();
    const theme = useTheme();

    const contactLinks = [
        ['contact-twitter', 'Find us on Twitter', business?.SOCIAL?.Twitter, 'Twitter', TwitterIcon],
        ['contact-email', 'Have a question or feedback? Email us!', business?.EMAIL?.Link, 'Email Us', EmailIcon],
        ['contact-github', 'Check out the source code, or contribute :)', business?.SOCIAL?.GitHub, 'Source Code', GitHubIcon],
    ]

    return (
        <div className={classes.root}>
            <Grid container justifyContent='center' spacing={1}>
                <Grid item xs={12} sm={6}>
                    <List component="nav">
                        <ListItem variant="h5" component="h3" >
                            <ListItemText className={classes.upper} primary="Resources" />
                        </ListItem>
                        <ListItem button component="a" onClick={() => history.push(LINKS.About)} >
                            <ListItemText primary="About Us" />
                        </ListItem>
                        <ListItem button component="a" onClick={() => history.push(LINKS.Waitlist)} >
                            <ListItemText primary="Join Waitlist" />
                        </ListItem>
                    </List>
                </Grid>
                <Grid item xs={12} sm={6}>
                    <List component="nav">
                        <ListItem variant="h5" component="h3" >
                            <ListItemText className={classes.upper} primary="Contact" />
                        </ListItem>
                        {contactLinks.map(([label, tooltip, src, text, Icon], key) => (
                            <Tooltip key={key} title={tooltip} placement="left">
                                <ListItem button component="a" aria-label={label} href={src}>
                                    <ListItemIcon>
                                        <Icon className={classes.icon} ></Icon>
                                    </ListItemIcon>
                                    <ListItemText primary={text} />
                                </ListItem>
                            </Tooltip>
                        ))}
                    </List>
                </Grid>
            </Grid>
            <CopyrightBreadcrumbs className={classes.copyright} business={business} textColor={theme.palette.primary.contrastText} />
        </div>
    );
}

Footer.propTypes = {
    session: PropTypes.object,
}

export { Footer };
