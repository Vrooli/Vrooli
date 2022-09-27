import {
    ContactInfo,
    PopupMenu
} from 'components';
import { Action, actionsToMenu, ACTION_TAGS, getUserActions, openLink } from 'utils';
import { Button, Container, IconButton, Palette, useTheme } from '@mui/material';
import { useLocation } from '@shared/route';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { NavListProps } from '../types';
import { APP_LINKS } from '@shared/consts';
import { LogInIcon, ProfileIcon } from '@shared/icons';

const navItemStyle = (palette: Palette) => ({
    background: 'transparent',
    color: palette.primary.contrastText,
    textTransform: 'none',
    fontSize: '1.4em',
    '&:hover': {
        color: palette.secondary.light,
    },
})

export const NavList = ({
    session,
    sessionChecked,
}: NavListProps) => {
    const { breakpoints, palette } = useTheme();
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
        <Container sx={{
            display: 'flex',
            height: '100%',
            paddingBottom: '0',
            paddingRight: '0 !important',
            right: '0',
        }}>
            {!isMobile && <PopupMenu
                text="Contact"
                variant="text"
                size="large"
                sx={navItemStyle(palette)}
            >
                <ContactInfo />
            </PopupMenu>}
            {/* List items displayed when on wide screen */}
            {!isMobile && actionsToMenu({
                actions: nav_actions,
                setLocation,
                sx: navItemStyle(palette),
            })}
            {/* Enter button displayed when not logged in */}
            {sessionChecked && session?.isLoggedIn !== true && (
                <Button
                    onClick={() => openLink(setLocation, APP_LINKS.Start)}
                    startIcon={<LogInIcon />}
                    sx={{
                        background: '#387e30',
                        borderRadius: '10px',
                        whiteSpace: 'nowrap',
                        // Hide text on small screens, and remove start icon's padding
                        fontSize: { xs: '0px', sm: '1em', md: '1.4em' },
                        '& .MuiButton-startIcon': {
                            marginLeft: { xs: '0px', sm: '-4px' },
                            marginRight: { xs: '0px', sm: '8px' },
                        },
                    }}
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
                    <ProfileIcon fill='white' width='40px' height='40px' />
                </IconButton>
            )}
        </Container>
    );
}