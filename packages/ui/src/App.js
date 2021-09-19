import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
    AlertDialog,
    BottomNav,
    Footer,
    Navbar,
    Snack
} from 'components';
import { PUBS, PubSub, themes } from 'utils';
import { GlobalHotKeys } from "react-hotkeys";
import { Routes } from 'Routes';
import { CssBaseline, CircularProgress } from '@material-ui/core';
import { ThemeProvider } from '@material-ui/core/styles';
import { makeStyles } from '@material-ui/styles';
import StyledEngineProvider from '@material-ui/core/StyledEngineProvider';
import { useHistory } from 'react-router';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { DndProvider } from 'react-dnd';
import { useMutation, useQuery } from '@apollo/client';
import { readAssetsQuery } from 'graphql/query/readAssets';
import { loginMutation } from 'graphql/mutation';

const useStyles = makeStyles(() => ({
        "@global": {
            body: {
                backgroundColor: 'black',
            },
            '#page': {
                minWidth: '100%',
                minHeight: '100%',
                padding: '1em',
                paddingTop: 'calc(14vh + 20px)'
            },
            '@media (min-width:500px)': {
                '#page': {
                    paddingLeft: 'max(1em, calc(15% - 75px))',
                    paddingRight: 'max(1em, calc(15% - 75px))',
                }
            },
        },
        contentWrap: {
            minHeight: '100vh',
        },
        spinner: {
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: 'translate(-50%, -50%)',
            zIndex: '100000',
        },
}));

const keyMap = {
    OPEN_MENU: "left",
    TOGGLE_MENU: "m",
    CLOSE_MENU: "right",
    CLOSE_MENU_OR_POPUP: ["escape", "backspace"]
};

export function App() {
    const classes = useStyles();
    // Session cookie should automatically expire in time determined by server,
    // so no need to validate session on first load
    const [session, setSession] = useState(null);
    const [theme, setTheme] = useState(themes.light);
    const [cart, setCart] = useState(null);
    const [loading, setLoading] = useState(false);
    const timerRef = useRef();
    const [business, setBusiness] =  useState(null)
    const { data: businessData } = useQuery(readAssetsQuery, { variables: { files: ['hours.md', 'business.json'] } });
    const [login] = useMutation(loginMutation);
    let history = useHistory();

    useEffect(() => () => {
        clearTimeout(timerRef.current);
        setLoading(false);
    }, []);

    useEffect(() => {
        if (businessData === undefined) return;
        let data = businessData.readAssets[1] ? JSON.parse(businessData.readAssets[1]) : {};
        let hoursRaw = businessData.readAssets[0];
        data.hours = hoursRaw;
        setBusiness(data);
    }, [businessData])

    useEffect(() => {
        // Determine theme
        if (session?.theme) setTheme(themes[session?.theme])
        else if (session && window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) setTheme(themes.dark);
        else setTheme(themes.light);
        setCart(session?.cart ?? null);
    }, [session])

    const handlers = {
        OPEN_MENU: () => PubSub.publish(PUBS.BurgerMenuOpen, true),
        TOGGLE_MENU: () => PubSub.publish(PUBS.BurgerMenuOpen, 'toggle'),
        CLOSE_MENU: () => PubSub.publish(PUBS.BurgerMenuOpen, false),
        CLOSE_MENU_OR_POPUP: () => {
            handlers.CLOSE_MENU();
        },
    };

    const checkLogin = useCallback((session) => {
        if (session) {
            setSession(session);
            return;
        }
        login().then((response) => {
            setSession(response.data.login);
        }).catch((response) => { 
            if (process.env.NODE_ENV === 'development') console.error('Error: cannot login', response);
            setSession({})
        })
    }, [login])

    useEffect(() => {
        checkLogin();
        // Handle loading spinner, which can have a delay
        let loadingSub = PubSub.subscribe(PUBS.Loading, (_, data) => {
            clearTimeout(timerRef.current);
            if (Number.isInteger(data)) {
                timerRef.current = window.setTimeout(() => setLoading(true), Math.abs(data));
            } else {
                setLoading(Boolean(data));
            }
        });
        let businessSub = PubSub.subscribe(PUBS.Business, (_, data) => setBusiness(data));
        let themeSub = PubSub.subscribe(PUBS.Theme, (_, data) => setTheme(themes[data] ?? themes.light));
        return (() => {
            PubSub.unsubscribe(loadingSub);
            PubSub.unsubscribe(businessSub);
            PubSub.unsubscribe(themeSub);
        })
    }, [checkLogin])

    const redirect = (link) => history.push(link);

    return (
        <StyledEngineProvider injectFirst>
            <CssBaseline />
            <ThemeProvider theme={theme}>
                <DndProvider backend={HTML5Backend}>
                    <div id="App">
                        <GlobalHotKeys keyMap={keyMap} handlers={handlers} root={true} />
                        <main 
                            id="page-container" 
                            style={{ 
                                background: theme.palette.background.default, 
                                color: theme.palette.background.textPrimary,
                            }}
                        >
                            <div id="content-wrap" className={classes.contentWrap}>
                                <Navbar
                                    session={session}
                                    business={business}
                                    onSessionUpdate={checkLogin}
                                    roles={session?.roles}
                                    cart={cart}
                                    onRedirect={redirect}
                                />
                                {loading ?
                                    <div className={classes.spinner}>
                                        <CircularProgress size={100} />
                                    </div>
                                    : null}
                                <AlertDialog />
                                <Snack />
                                <Routes
                                    session={session}
                                    onSessionUpdate={checkLogin}
                                    business={business}
                                    userRoles={session?.roles}
                                    cart={cart}
                                    onRedirect={redirect}
                                />
                            </div>
                            <BottomNav session={session} userRoles={session?.roles} cart={cart} />
                            <Footer session={session} business={business} />
                        </main>
                    </div>
                </DndProvider>
            </ThemeProvider>
        </StyledEngineProvider>
    );
}
