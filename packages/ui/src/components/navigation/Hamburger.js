import React, { useEffect, useState } from 'react';
import PropTypes from 'prop-types';
import { 
    ContactInfo,
} from 'components';
import { getUserActions, LINKS, PUBS, PubSub } from 'utils';
import {
    Close as CloseIcon,
    ContactSupport as ContactSupportIcon,
    ExitToApp as ExitToAppIcon,
    ExpandLess as ExpandLessIcon,
    ExpandMore as ExpandMoreIcon,
    Home as HomeIcon,
    Menu as MenuIcon,
    Info as InfoIcon,
    PhotoLibrary as PhotoLibraryIcon,
    Share as ShareIcon,
    Twitter as TwitterIcon,
} from '@material-ui/icons';
import { IconButton, SwipeableDrawer, List, ListItem, ListItemIcon, Badge, Collapse, Divider, ListItemText } from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { CopyrightBreadcrumbs } from 'components';
import { useTheme } from '@emotion/react';
import _ from 'lodash';

const useStyles = makeStyles((theme) => ({
    drawerPaper: {
        background: theme.palette.primary.light,
        borderLeft: `1px solid ${theme.palette.text.primary}`,
    },
    menuItem: {
        color: theme.palette.primary.contrastText,
        borderBottom: `1px solid ${theme.palette.primary.dark}`,
    },
    close: {
        color: theme.palette.primary.contrastText,
        borderRadius: 0,
        borderBottom: `1px solid ${theme.palette.primary.dark}`,
        justifyContent: 'end',
        direction: 'rtl',
    },
    menuIcon: {
        color: theme.palette.primary.contrastText,
    },
    twitter: {
        fill: theme.palette.primary.contrastText,
    },
    copyright: {
        color: theme.palette.primary.contrastText,
        padding: 5,
        display: 'block',
        marginLeft: 'auto',
        marginRight: 'auto',
    },
}));

function Hamburger({
    session,
    business,
    logout,
    roles,
    cart,
    onRedirect
}) {
    const classes = useStyles();
    const theme = useTheme();
    const [contactOpen, setContactOpen] = useState(true);
    const [socialOpen, setSocialOpen] = useState(false);
    const [open, setOpen] = useState(false);

    useEffect(() => {
        let openSub = PubSub.subscribe(PUBS.BurgerMenuOpen, (_, b) => {
            setOpen(open => b === 'toggle' ? !open : b);
        });
        return (() => {
            PubSub.unsubscribe(openSub);
        })
    }, [])

    const closeMenu = () => PubSub.publish(PUBS.BurgerMenuOpen, false);
    const toggleOpen = () => PubSub.publish(PUBS.BurgerMenuOpen, 'toggle');

    const handleContactClick = () => {
        setContactOpen(!contactOpen);
    };

    const handleSocialClick = () => {
        setSocialOpen(!socialOpen);
    }

    const newTab = (link) => {
        window.open(link, "_blank");
    }

    const optionsToList = (options) => {
        return options.map(([label, value, link, onClick, Icon, badgeNum], index) => (
            <ListItem 
                key={index}
                className={classes.menuItem}
                button
                onClick={() => { 
                    onRedirect(link); 
                    if (onClick) onClick();
                    closeMenu();
                }}>
                {Icon ?
                    (<ListItemIcon>
                        <Badge badgeContent={badgeNum ?? 0} color="error">
                            <Icon className={classes.menuIcon} />
                        </Badge>
                    </ListItemIcon>) : null}
                <ListItemText primary={label} />
            </ListItem>
        ))
    }

    let nav_options = [
        ['Home', 'home', LINKS.Home, null, HomeIcon],
        ['About Us', 'about', LINKS.About, null, InfoIcon],
        ['Gallery', 'gallery', LINKS.Gallery, null, PhotoLibraryIcon]
    ]

    let customer_actions = getUserActions(session, roles, cart);
    if (_.isObject(session) && Object.entries(session).length > 0) {
        customer_actions.push(['Log Out', 'logout', LINKS.Home, logout, ExitToAppIcon]);
    }

    return (
        <React.Fragment>
            <IconButton edge="start" color="inherit" aria-label="menu" onClick={toggleOpen}>
                <MenuIcon />
            </IconButton>
            <SwipeableDrawer classes={{ paper: classes.drawerPaper }} anchor="right" open={open} onOpen={()=>{}} onClose={closeMenu}>
                <IconButton className={classes.close} onClick={closeMenu}>
                    <CloseIcon fontSize="large" />
                </IconButton>
                <List>
                    {/* Collapsible contact information */}
                    <ListItem className={classes.menuItem} button onClick={handleContactClick}>
                        <ListItemIcon><ContactSupportIcon className={classes.menuIcon} /></ListItemIcon>
                        <ListItemText primary="Contact Us" />
                        {contactOpen ? <ExpandLessIcon /> : <ExpandMoreIcon />}
                    </ListItem>
                    <Collapse className={classes.menuItem} in={contactOpen} timeout="auto" unmountOnExit>
                        <ContactInfo business={business} />
                    </Collapse>
                    {/* Collapsible social media links */}
                    <ListItem className={classes.menuItem} button onClick={handleSocialClick}>
                        <ListItemIcon><ShareIcon className={classes.menuIcon} /></ListItemIcon>
                        <ListItemText primary="Socials" />
                        {socialOpen ? <ExpandLessIcon /> : <ExpandMoreIcon />}
                    </ListItem>
                    <Collapse in={socialOpen} timeout="auto" unmountOnExit>
                        <ListItem className={classes.menuItem} button onClick={() => newTab(business?.SOCIAL?.Twitter)}>
                            <ListItemIcon>
                                <TwitterIcon className={classes.twitter} />
                            </ListItemIcon>
                            <ListItemText primary="Twitter" />
                        </ListItem>
                    </Collapse>
                    {optionsToList(nav_options)}
                    <Divider />
                    {optionsToList(customer_actions)}
                </List>
                <CopyrightBreadcrumbs className={classes.copyright} business={business} textColor={theme.palette.primary.contrastText} />
            </SwipeableDrawer>
        </React.Fragment>
    );
}

Hamburger.propTypes = {
    session: PropTypes.object,
    logout: PropTypes.func.isRequired,
    roles: PropTypes.array,
    cart: PropTypes.object,
}

export { Hamburger };