import {
    ContactInfo,
    PopupMenu
} from 'components';
import { Container, Theme, useTheme } from '@mui/material';
import { makeStyles } from '@mui/styles';
import { useLocation } from '@local/shared';
import { useCallback, useEffect, useState } from 'react';

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

export const NavList = () => {
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
        </Container>
    );
}