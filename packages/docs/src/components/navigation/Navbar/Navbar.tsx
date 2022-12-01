import { useCallback } from 'react';
import Logo from 'assets/img/Logo-128x128.png';
import { BUSINESS_NAME, APP_LINKS } from '@shared/consts';
import { AppBar, Toolbar, Typography, Box, useTheme } from '@mui/material';
import { NavList } from '../NavList/NavList';
import { useLocation } from '@shared/route';
import { HideOnScroll } from '..';
import { noSelect } from 'styles';

export const Navbar = () => {
    const { breakpoints, palette } = useTheme();
    const [, setLocation] = useLocation();

    const toHome = useCallback(() => setLocation(APP_LINKS.Home), [setLocation]);

    return (
        <HideOnScroll>
            <AppBar sx={{
                ...noSelect,
                background: palette.primary.dark,
                height: { xs: '64px', md: '80px' },
                zIndex: 100,
            }}>
                <Toolbar>
                    <Box
                        onClick={toHome}
                        sx={{
                            padding: 0,
                            display: 'flex',
                            alignItems: 'center',
                        }}
                    >
                        <Box sx={{
                            display: 'flex',
                            padding: 0,
                            cursor: 'pointer',
                            margin: '5px',
                            borderRadius: '500px',
                        }}>
                            <Box
                                component="img"
                                src={Logo}
                                alt={`${BUSINESS_NAME} Logo`}
                                sx={{
                                    verticalAlign: 'middle',
                                    fill: 'black',
                                    marginLeft: 'max(-5px, -5vw)',
                                    width: '48px',
                                    height: '48px',
                                    [breakpoints.up('md')]: {
                                        width: '6vh',
                                        height: '6vh',
                                    },
                                }}
                            />
                        </Box>
                        <Typography
                            variant="h6"
                            noWrap
                            sx={{
                                position: 'relative',
                                cursor: 'pointer',
                                fontSize: '3.5em',
                                fontFamily: `SakBunderan`,
                                color: palette.primary.contrastText,
                                [breakpoints.down(400)]: {
                                    fontSize: '3em',
                                },
                            }}
                        >{BUSINESS_NAME}</Typography>
                    </Box>
                    <Box sx={{
                        marginLeft: 'auto',
                        maxHeight: '100%',
                    }}>
                        <NavList />
                    </Box>
                </Toolbar>
            </AppBar>
        </HideOnScroll>
    );
}