import {
    AccountMenu,
    ContactInfo,
    PopupMenu
} from 'components';
import { Action, actionsToMenu, ACTION_TAGS, getUserActions, openLink, useWindowSize } from 'utils';
import { Button, Container, IconButton, Palette, useTheme } from '@mui/material';
import { useLocation } from '@shared/route';
import { useCallback, useMemo, useState } from 'react';
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

    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    const nav_actions = useMemo<Action[]>(() => getUserActions({ session, exclude: [ACTION_TAGS.Home, ACTION_TAGS.LogIn] }), [session]);

    // Handle account menu
    const [accountMenuAnchor, setAccountMenuAnchor] = useState<any>(null);
    const openAccountMenu = useCallback((ev: React.MouseEvent<any>) => {
        setAccountMenuAnchor(ev.currentTarget)
    }, [setAccountMenuAnchor]);
    const closeAccountMenu = useCallback(() => setAccountMenuAnchor(null), []);

    return (
        <Container sx={{
            display: 'flex',
            height: '100%',
            paddingBottom: '0',
            paddingRight: '0 !important',
            right: '0',
        }}>
            {/* Contact menu */}
            {!isMobile && <PopupMenu
                text="Contact"
                variant="text"
                size="large"
                sx={navItemStyle(palette)}
            >
                <ContactInfo />
            </PopupMenu>}
            {/* Account menu */}
            <AccountMenu
                anchorEl={accountMenuAnchor}
                onClose={closeAccountMenu}
                session={session}
            />
            {/* List items displayed when on wide screen */}
            {!isMobile && actionsToMenu({
                actions: nav_actions,
                setLocation,
                sx: navItemStyle(palette),
            })}
            {/* Enter button displayed when not logged in */}
            {sessionChecked && session?.isLoggedIn !== true && (
                <Button
                    href={APP_LINKS.Start}
                    onClick={(e) => { e.preventDefault(); openLink(setLocation, APP_LINKS.Start) }}
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
            {/* Profile icon */}
            {session?.isLoggedIn === true && (
                <IconButton
                    color="inherit"
                    onClick={openAccountMenu}
                    aria-label="profile"
                >
                    <ProfileIcon fill='white' width='40px' height='40px' />
                </IconButton>
            )}
        </Container>
    );
}