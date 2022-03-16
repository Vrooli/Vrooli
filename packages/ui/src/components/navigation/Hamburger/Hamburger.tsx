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
    SxProps,
    Theme,
} from '@mui/material';
import { CopyrightBreadcrumbs } from 'components';
import { useLocation } from 'wouter';
import { HamburgerProps } from '../types';

const highlight: SxProps<Theme> = {
    transition: 'ease-in-out 0.2s',
    color: (t) => t.palette.primary.contrastText,
    '&:hover': {
        color: '(t) => t.palette.secondary.light',
    },
} as any

const menuItem: SxProps<Theme> = {
    borderBottom: `1px solid #373575`,
    color: (t) => t.palette.primary.contrastText,
    cursor: 'pointer',
} as const

export const Hamburger = ({
    session,
    sessionChecked,
}: HamburgerProps) => {
    const [, setLocation] = useLocation();
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

    const nav_actions = getUserActions({ roles: session.roles ?? [] });

    return (
        <>
            <IconButton edge="start" color="inherit" aria-label="menu" onClick={toggleOpen}>
                <MenuIcon />
            </IconButton>
            <SwipeableDrawer 
                anchor="right" 
                open={open} 
                onOpen={() => { }} 
                onClose={closeMenu}
                sx={{
                    '& .MuiDrawer-paper': {
                        background: (t) => t.palette.primary.light,
                        borderLeft: `1px solid ${(t) => t.palette.text.primary}`,
                    }
                }}
            >
                <IconButton onClick={closeMenu} sx={{
                    color: (t) => t.palette.primary.contrastText,
                    borderRadius: 0,
                    borderBottom: `1px solid ${(t) => t.palette.primary.dark}`,
                    justifyContent: 'end',
                    flexDirection: 'row-reverse',
                }}>
                    <CloseIcon fontSize="large" />
                </IconButton>
                <List>
                    {/* Collapsible contact information */}
                    <ListItem button onClick={handleContactClick} sx={{ ...menuItem, borderTop: '1px solid #373575' }}>
                        <ListItemIcon><ContactSupportIcon sx={{ ...highlight }} /></ListItemIcon>
                        <ListItemText primary="Contact Us" />
                        {contactOpen ? <ExpandLessIcon /> : <ExpandMoreIcon />}
                    </ListItem>
                    <Collapse in={contactOpen} timeout="auto" unmountOnExit sx={{ ...menuItem, ...highlight }}>
                        <ContactInfo />
                    </Collapse>
                    {actionsToList({
                        actions: nav_actions,
                        setLocation,
                        sxs: { listItem: { ...menuItem, ...highlight }, listItemIcon: { ...highlight } },
                        onAnyClick: closeMenu,
                    })}
                </List>
                <CopyrightBreadcrumbs
                    sx={{
                        color: (t) => t.palette.primary.contrastText,
                        padding: 5,
                        display: 'block',
                        marginLeft: 'auto',
                        marginRight: 'auto',
                        marginTop: 'auto',
                    }}
                />
            </SwipeableDrawer>
        </>
    );
}