import { useCallback, useEffect, useState } from 'react';
import Logo from 'assets/img/Logo.png';
import { LANDING_LINKS } from '@local/shared';
import { AppBar, Toolbar, Typography, Slide, useScrollTrigger, Theme } from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { Hamburger } from './Hamburger';
import { NavList } from './NavList';
import { useHistory } from 'react-router';
import { BUSINESS_NAME } from '@local/shared';

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

export const Navbar = () => {
    const classes = useStyles();
    const history = useHistory();
    const [show_hamburger, setShowHamburger] = useState(false);

    useEffect(() => {
        updateWindowDimensions();
        window.addEventListener("resize", updateWindowDimensions);

        return () => window.removeEventListener("resize", updateWindowDimensions);
    }, []);

    const updateWindowDimensions = () => setShowHamburger(window.innerWidth <= SHOW_HAMBURGER_AT);

    const toHome = useCallback(() => history.push(LANDING_LINKS.Home), [history]);

    return (
        <HideOnScroll>
            <AppBar className={classes.root}>
                <Toolbar>
                    <div className={classes.navLogoContainer} onClick={toHome}>
                        <div className={classes.navLogoDiv}>
                            <img src={Logo} alt={`${BUSINESS_NAME} Logo`} className={classes.navLogo} />
                        </div>
                        <Typography className={classes.navName} variant="h6" noWrap>{BUSINESS_NAME}</Typography>
                    </div>
                    <div className={classes.toRight}>
                        {show_hamburger ? <Hamburger /> : <NavList />}
                    </div>
                </Toolbar>
            </AppBar>
        </HideOnScroll>
    );
}