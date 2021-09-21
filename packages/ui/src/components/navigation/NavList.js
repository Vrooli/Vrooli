import React from 'react';
import PropTypes from 'prop-types';
import {
    ContactInfo,
    PopupMenu
} from 'components';
import { getUserActions, LINKS } from 'utils';
import { Container, Button, List, ListItem, ListItemIcon, ListItemText } from '@material-ui/core';
import {
    Info as InfoIcon,
} from '@material-ui/icons';
import { makeStyles } from '@material-ui/styles';
import _ from 'lodash';

const useStyles = makeStyles((theme) => ({
    root: {
        display: 'flex',
        marginTop: '0px',
        marginBottom: '0px',
        right: '0px',
        padding: '0px',
    },
    navItem: {
        background: 'transparent',
        color: theme.palette.primary.contrastText,
        textTransform: 'none',
    },
    menuItem: {
        color: theme.palette.primary.contrastText,
    },
    menuIcon: {
        fill: theme.palette.primary.contrastText,
    },
    contact: {
        width: 'calc(min(100vw, 400px))',
        height: '300px',
    },
}));

function NavList({
    session,
    business,
    logout,
    roles,
    onRedirect
}) {
    const classes = useStyles();
    
    let nav_options = getUserActions(session, roles);
    // If someone is not logged in, display sign up/log in links
    if (!_.isObject(session) || Object.keys(session).length === 0) {
        nav_options.push(['Sign Up', 'signup', LINKS.Register]);
    } else {
        nav_options.push(['Log out', 'logout', LINKS.Home, logout])
    }

    let about_options = [
        ['About Us', 'about', LINKS.About, null, InfoIcon],
    ]

    const optionsToList = (options) => {
        return options.map(([label, value, link, onClick, Icon], index) => (
            <ListItem className={classes.menuItem} button key={index} onClick={() => { onRedirect(link); if (onClick) onClick() }}>
                {Icon ?
                    (<ListItemIcon>
                            <Icon className={classes.menuIcon} />
                    </ListItemIcon>) : null}
                <ListItemText primary={label} />
            </ListItem>
        ))
    }

    const optionsToMenu = (options) => {
        return options.map(([label, value, link, onClick], index) => (
            <Button
                key={index}
                variant="text"
                size="large"
                className={classes.navItem}
                onClick={() => { onRedirect(link); if(onClick) onClick()}}
            >
                {label}
            </Button>
        ));
    }

    return (
        <Container className={classes.root}>
            <PopupMenu 
                text="Contact"
                variant="text"
                size="large"
                className={classes.navItem}
            >
                <ContactInfo className={classes.contact} business={business} />
            </PopupMenu>
            <PopupMenu
                text="About"
                variant="text"
                size="large"
                className={classes.navItem}
            >
                <List>
                    {optionsToList(about_options)}
                </List>
            </PopupMenu>
            {optionsToMenu(nav_options)}
        </Container>
    );
}

NavList.propTypes = {
    session: PropTypes.object,
    logout: PropTypes.func.isRequired,
    roles: PropTypes.array,
}

export { NavList };