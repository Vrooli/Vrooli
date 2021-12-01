import { useCallback, useEffect, useState } from 'react';
import Logo from 'assets/img/Logo.png';
import { LINKS } from 'utils';
import { AppBar, Toolbar, Typography, Slide, useScrollTrigger, Theme } from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { Hamburger } from './Hamburger';
import { NavList } from './NavList';
import { CommonProps } from 'types';
import { useHistory } from 'react-router';

const SHOW_HAMBURGER_AT = 1000;

const useStyles = makeStyles((theme: Theme) => ({
    root: {
        background: theme.palette.primary.dark,
        height: '10vh',
    },
    toRight: {
        marginLeft: 'auto',
    },
    navLogoContainer: {
        padding: 0,
        display: 'flex',
        alignItems: 'center',
    },
    navLogoDiv: {
        display: 'flex',
        padding: 0,
        cursor: 'pointer',
        margin: '5px',
        borderRadius: '500px',
    },
    navLogo: {
        verticalAlign: 'middle',
        fill: 'black',
        marginLeft: 'max(-5px, -5vw)',
        minWidth: '50px',
        minHeight: '50px',
        width: '6vh',
        height: '6vh',
    },
    navName: {
        position: 'relative',
        cursor: 'pointer',
        fontSize: '3.5em',
        fontFamily: `Lato`,
        color: theme.palette.primary.contrastText,
    },
    [theme.breakpoints.down(350)]: {
        navName: {
            display: 'none',
        }
    },
}));

interface HideOnScrollProps {
    children: JSX.Element;
}

const HideOnScroll = ({
    children,
}: HideOnScrollProps) => {
    const trigger = useScrollTrigger();
    return (
        <Slide appear={false} direction="down" in={!trigger}>
            {children}
        </Slide>
    );
}

export const Navbar = ({
    business,
    userRoles
}: Pick<CommonProps, 'business' | 'userRoles'>) => {
    const classes = useStyles();
    const history = useHistory();
    const [show_hamburger, setShowHamburger] = useState(false);

    let child_props = { 
        business: business,
        userRoles: userRoles,
    }

    useEffect(() => {
        updateWindowDimensions();
        window.addEventListener("resize", updateWindowDimensions);

        return () => window.removeEventListener("resize", updateWindowDimensions);
    }, []);

    const updateWindowDimensions = () => setShowHamburger(window.innerWidth <= SHOW_HAMBURGER_AT);

    const toLanding = useCallback(() => history.push(LINKS.Landing), [history]);

    return (
        <HideOnScroll>
            <AppBar className={classes.root}>
                <Toolbar>
                    <div className={classes.navLogoContainer} onClick={toLanding}>
                        <div className={classes.navLogoDiv}>
                            <img src={Logo} alt={`${business?.BUSINESS_NAME?.Short} Logo`} className={classes.navLogo} />
                        </div>
                        <Typography className={classes.navName} variant="h6" noWrap>{business?.BUSINESS_NAME?.Short}</Typography>
                    </div>
                    <div className={classes.toRight}>
                        {show_hamburger ? <Hamburger {...child_props} /> : <NavList {...child_props} />}
                    </div>
                </Toolbar>
            </AppBar>
        </HideOnScroll>
    );
}