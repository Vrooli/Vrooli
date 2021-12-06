import { useEffect, useState } from 'react';
import { ContactInfo } from 'components';
import { actionsToList, ACTION_TAGS, getUserActions, PUBS } from 'utils';
import PubSub from 'pubsub-js';
import {
    Close as CloseIcon,
    ContactSupport as ContactSupportIcon,
    ExpandLess as ExpandLessIcon,
    ExpandMore as ExpandMoreIcon,
    Menu as MenuIcon,
} from '@material-ui/icons';
import { IconButton, SwipeableDrawer, List, ListItem, ListItemIcon, Collapse, ListItemText, Theme } from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { useTheme } from '@material-ui/core';
import { CopyrightBreadcrumbs } from 'components';
import { useHistory } from 'react-router';

const useStyles = makeStyles((theme: Theme) => ({
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
    facebook: {
        fill: '#ffffff', //'#43609C', // UCLA blue
    },
    instagram: {
        fill: '#ffffff', // '#F77737',
    },
    copyright: {
        color: theme.palette.primary.contrastText,
        padding: 5,
        display: 'block',
        marginLeft: 'auto',
        marginRight: 'auto',
    },
}));

export const Hamburger = () => {
    const classes = useStyles();
    const history = useHistory();
    const theme = useTheme();
    const [contactOpen, setContactOpen] = useState(true);
    const [open, setOpen] = useState(false);

    useEffect(() => {
        let openSub = PubSub.subscribe(PUBS.BurgerMenuOpen, (_, b) => {
            setOpen(open => b === 'toggle' ? !open : b);
        });
        return () => { PubSub.unsubscribe(openSub) };
    }, [])

    const closeMenu = () => PubSub.publish(PUBS.BurgerMenuOpen, false);
    const toggleOpen = () => PubSub.publish(PUBS.BurgerMenuOpen, 'toggle');

    const handleContactClick = () => {
        setContactOpen(!contactOpen);
    };

    const nav_actions = getUserActions({ exclude: [ACTION_TAGS.Home] });

    return (
        <>
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
                        <ContactInfo />
                    </Collapse>
                    {actionsToList({
                        actions: nav_actions,
                        history,
                        classes: { listItem: classes.menuItem, listItemIcon: classes.menuIcon },
                        onAnyClick: closeMenu,
                    })}
                </List>
                <CopyrightBreadcrumbs className={classes.copyright} textColor={theme.palette.primary.contrastText} />
            </SwipeableDrawer>
        </>
    );
}