import { useEffect, useState } from 'react';
import { ContactInfo } from 'components';
import { actionsToList, getUserActions, Pubs } from 'utils';
import PubSub from 'pubsub-js';
import {
    Close as CloseIcon,
    ContactSupport as ContactSupportIcon,
    ExpandLess as ExpandLessIcon,
    ExpandMore as ExpandMoreIcon,
    Menu as MenuIcon,
} from '@mui/icons-material';
import {
    Collapse,
    IconButton,
    List,
    ListItem,
    ListItemIcon,
    ListItemText,
    SwipeableDrawer,
    Theme,
    useTheme
} from '@mui/material';
import { makeStyles } from '@mui/styles';
import { CopyrightBreadcrumbs } from 'components';
import { useLocation } from 'wouter';
import { HamburgerProps } from '../types';

const useStyles = makeStyles((theme: Theme) => ({
    drawerPaper: {
        background: theme.palette.primary.light,
        borderLeft: `1px solid ${theme.palette.text.primary}`,
    },
    highlight: {
        transition: 'ease-in-out 0.2s',
        color: theme.palette.primary.contrastText,
        '&:hover': {
            color: theme.palette.secondary.light,
        },
    },
    menuItem: {
        borderBottom: `1px solid ${theme.palette.primary.dark}`,
        color: theme.palette.primary.contrastText,
    },
    close: {
        color: theme.palette.primary.contrastText,
        borderRadius: 0,
        borderBottom: `1px solid ${theme.palette.primary.dark}`,
        justifyContent: 'end',
        direction: 'rtl',
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

export const Hamburger = ({
    userRoles,
}: HamburgerProps) => {
    const classes = useStyles();
    const [, setLocation] = useLocation();
    const theme = useTheme();
    const [contactOpen, setContactOpen] = useState(true);
    const [open, setOpen] = useState(false);

    useEffect(() => {
        let openSub = PubSub.subscribe(Pubs.BurgerMenuOpen, (_, b) => {
            setOpen(open => b === 'toggle' ? !open : b);
        });
        return () => { PubSub.unsubscribe(openSub) };
    }, [])

    const closeMenu = () => PubSub.publish(Pubs.BurgerMenuOpen, false);
    const toggleOpen = () => PubSub.publish(Pubs.BurgerMenuOpen, 'toggle');

    const handleContactClick = () => {
        setContactOpen(!contactOpen);
    };

    const nav_actions = getUserActions({ userRoles });

    return (
        <>
            <IconButton edge="start" color="inherit" aria-label="menu" onClick={toggleOpen}>
                <MenuIcon />
            </IconButton>
            <SwipeableDrawer classes={{ paper: classes.drawerPaper }} anchor="right" open={open} onOpen={() => { }} onClose={closeMenu}>
                <IconButton className={classes.close} onClick={closeMenu}>
                    <CloseIcon fontSize="large" />
                </IconButton>
                <List>
                    {/* Collapsible contact information */}
                    <ListItem className={classes.menuItem} button onClick={handleContactClick}>
                        <ListItemIcon><ContactSupportIcon className={classes.highlight} /></ListItemIcon>
                        <ListItemText primary="Contact Us" />
                        {contactOpen ? <ExpandLessIcon /> : <ExpandMoreIcon />}
                    </ListItem>
                    <Collapse className={classes.highlight} in={contactOpen} timeout="auto" unmountOnExit>
                        <ContactInfo />
                    </Collapse>
                    {actionsToList({
                        actions: nav_actions,
                        setLocation,
                        classes: { listItem: `${classes.menuItem} ${classes.highlight}`, listItemIcon: classes.highlight },
                        onAnyClick: closeMenu,
                    })}
                </List>
                <CopyrightBreadcrumbs className={classes.copyright} textColor={theme.palette.primary.contrastText} />
            </SwipeableDrawer>
        </>
    );
}