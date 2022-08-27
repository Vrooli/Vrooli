import {
    ContactInfo,
    PopupMenu
} from 'components';
import {
    AccountCircle as ProfileIcon,
} from '@mui/icons-material';
import { Action, actionsToMenu, ACTION_TAGS, getUserActions, openLink } from 'utils';
import { Button, Container, IconButton, Theme, useTheme } from '@mui/material';
import { makeStyles } from '@mui/styles';
import { useLocation } from 'wouter';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { NavListProps } from '../types';
import { APP_LINKS } from '@local/shared';

const useStyles = makeStyles((theme: Theme) => ({
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
        fontSize: '1.5em',
        '&:hover': {
            color: theme.palette.secondary.light,
        },
    },
    button: {
        fontSize: '1.5em',
        borderRadius: '10px',
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

export const NavList = ({
    session,
    sessionChecked,
}: NavListProps) => {
    const classes = useStyles();
    const { breakpoints } = useTheme();
    const [, setLocation] = useLocation();

    const [isMobile, setIsMobile] = useState(false); // Not shown on mobile
    const updateWindowDimensions = useCallback(() => setIsMobile(window.innerWidth <= breakpoints.values.md), [breakpoints]);
    useEffect(() => {
        updateWindowDimensions();
        window.addEventListener("resize", updateWindowDimensions);
        return () => window.removeEventListener("resize", updateWindowDimensions);
    }, [updateWindowDimensions]);

    const nav_actions = useMemo<Action[]>(() => getUserActions({ session, exclude: [ACTION_TAGS.Home, ACTION_TAGS.LogIn] }), [session]);

    return (
        <Container className={classes.root}>
            {!isMobile && <PopupMenu
                text="Contact"
                variant="text"
                size="large"
                className={classes.navItem}
            >
                <ContactInfo className={classes.contact} />
            </PopupMenu>}
            {/* List items displayed when on wide screen */}
            {!isMobile && actionsToMenu({
                actions: nav_actions,
                setLocation,
                classes: { root: classes.navItem },
            })}
            {/* Enter button displayed when not logged in */}
            {sessionChecked && session?.isLoggedIn !== true && (
                <Button
                    className={classes.button}
                    onClick={() => openLink(setLocation, APP_LINKS.Start)}
                    sx={{ background: '#387e30' }}
                >
                    Log In
                </Button>
            )}
            {/* Profile icon for mobile */}
            {isMobile && session?.isLoggedIn === true && (
                <IconButton
                    color="inherit"
                    onClick={() => { setLocation(APP_LINKS.Profile) }}
                    aria-label="profile"
                >
                    <ProfileIcon sx={{ fill: 'white', width: '40px', height: '40px' }} />
                </IconButton>
            )}
        </Container>
    );
}