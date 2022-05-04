import { useState, useEffect, useCallback, useRef } from 'react';
import {
    AlertDialog,
    BottomNav,
    Footer,
    Navbar,
    Snack
} from 'components';
import PubSub from 'pubsub-js';
import { Pubs, themes, useReactHash } from 'utils';
import { AllRoutes } from 'Routes';
import { Box, CssBaseline, CircularProgress, StyledEngineProvider, ThemeProvider } from '@mui/material';
import { makeStyles } from '@mui/styles';
import { useMutation } from '@apollo/client';
import { validateSessionMutation } from 'graphql/mutation';
import SakBunderan from './assets/font/SakBunderan.woff';
import { Session } from 'types';
import hotkeys from 'hotkeys-js';
import Confetti from 'react-confetti';

const useStyles = makeStyles(() => ({
    "@global": {
        body: {
            backgroundColor: 'black',
            overflowX: 'hidden',
            overflowY: 'auto',
            "&::-webkit-scrollbar": {
                width: 10,
            },
            "&::-webkit-scrollbar-track": {
                backgroundColor: '#dae5f0',
            },
            "&::-webkit-scrollbar-thumb": {
                borderRadius: '100px',
                backgroundColor: "#409590",
            },
        },
        '#page': {
            minWidth: '100vw',
            minHeight: '100vh',
            padding: '0.5em',
            paddingTop: '10vh'
        },
        '@media (min-width:500px)': {
            '#page': {
                paddingLeft: 'max(1em, calc(15% - 75px))',
                paddingRight: 'max(1em, calc(15% - 75px))',
            }
        },
        '@font-face': {
            fontFamily: 'Lato',
            src: `local('Lato'), url(${SakBunderan}) format('truetype')`,
            fontDisplay: 'swap',
        },
        '@keyframes gradient': {
            '0%': {
                backgroundPosition: '0% 50%',
            },
            '50%': {
                backgroundPosition: '100% 50%',
            },
            '100%': {
                backgroundPosition: '0% 50%',
            },
        }

    },
}));

export function App() {
    useStyles();
    // Session cookie should automatically expire in time determined by server,
    // so no need to validate session on first load
    const [session, setSession] = useState<Session | undefined>(undefined);
    const [theme, setTheme] = useState(themes.light);
    const [loading, setLoading] = useState(false);
    const [celebrating, setCelebrating] = useState(false);
    const timeoutRef = useRef<NodeJS.Timeout | null>(null);
    const [validateSession] = useMutation<any>(validateSessionMutation);

    // If anchor tag in url, scroll to element
    const hash = useReactHash();
    useEffect(() => {
        // if not a hash link, scroll to top
        if (window.location.hash === '') {
            window.scrollTo(0, 0);
        }
        // else scroll to id
        else {
            setTimeout(() => {
                const id = window.location.hash.replace('#', '');
                const element = document.getElementById(id);
                if (element) {
                    element.scrollIntoView();
                }
            }, 0);
        }
    }, [hash]);

    useEffect(() => {
        if (timeoutRef.current) clearTimeout(timeoutRef.current);
        setLoading(false);

        const handlers = {
            OPEN_MENU: () => PubSub.publish(Pubs.BurgerMenuOpen, true),
            TOGGLE_MENU: () => PubSub.publish(Pubs.BurgerMenuOpen, 'toggle'),
            CLOSE_MENU: () => PubSub.publish(Pubs.BurgerMenuOpen, false),
            CLOSE_MENU_OR_POPUP: () => {
                handlers.CLOSE_MENU();
            }
        }
        hotkeys('left', handlers.OPEN_MENU);
        hotkeys('right', handlers.CLOSE_MENU);
        hotkeys('m', handlers.TOGGLE_MENU);
        hotkeys('escape,backspace', handlers.CLOSE_MENU_OR_POPUP);
    }, []);

    useEffect(() => {
        // Determine theme
        if (session?.theme) setTheme(themes[session?.theme])
        //else if (session && window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) setTheme(themes.dark);
        else setTheme(themes.light);
    }, [session])

    const checkSession = useCallback((session?: any) => {
        if (session) {
            setSession(session);
            return;
        }
        // Check if previous log in exists
        validateSession().then(({ data }) => {
            setSession(data?.validateSession as Session);
        }).catch((response) => {
            if (process.env.NODE_ENV === 'development') console.error('Error: failed to verify session', response);
            // If not logged in as guest and failed to log in as user, set empty object
            if (!session) setSession({})
        })
    }, [validateSession])

    useEffect(() => {
        checkSession();
        // Handle loading spinner, which can have a delay
        let loadingSub = PubSub.subscribe(Pubs.Loading, (_, data) => {
            if (timeoutRef.current) clearTimeout(timeoutRef.current);
            if (Number.isInteger(data)) {
                timeoutRef.current = setTimeout(() => setLoading(true), Math.abs(data));
            } else {
                setLoading(Boolean(data));
            }
        });
        // Handle celebration (confetti). Defaults to 5 seconds long, but duration 
        // can be passed in as a number
        let celebrationSub = PubSub.subscribe(Pubs.Celebration, (_, data) => {
            // Start confetti immediately
            setCelebrating(true);
            // Determine duration
            let duration = 5000;
            if (Number.isInteger(data)) duration = data;
            // Stop confetti after duration
            setTimeout(() => setCelebrating(false), duration);
        });
        let sessionSub = PubSub.subscribe(Pubs.Session, (_, session) => {
            setSession(s => (session === undefined ? undefined : { ...s, ...session }));
        });
        let themeSub = PubSub.subscribe(Pubs.Theme, (_, data) => setTheme(themes[data] ?? themes.light));
        return (() => {
            PubSub.unsubscribe(loadingSub);
            PubSub.unsubscribe(celebrationSub);
            PubSub.unsubscribe(sessionSub);
            PubSub.unsubscribe(themeSub);
        })
    }, [checkSession]);

    return (
        <StyledEngineProvider injectFirst>
            <CssBaseline />
            <ThemeProvider theme={theme}>
                <div id="App">
                    <main
                        id="page-container"
                        style={{
                            background: theme.palette.background.default,
                            color: theme.palette.background.textPrimary,
                        }}
                    >
                        <Box id="content-wrap" sx={{
                            background: 'fixed radial-gradient(circle, rgba(208,213,226,1) 7%, rgba(179,191,217,1) 66%, rgba(160,188,249,1) 94%)',
                            minHeight: '100vh',
                        }}>
                            <Navbar session={session ?? {}} sessionChecked={session !== undefined} />
                            {/* Progress bar */}
                            {
                                loading && <Box sx={{
                                    position: 'absolute',
                                    top: '50%',
                                    left: '50%',
                                    transform: 'translate(-50%, -50%)',
                                    zIndex: 100000,
                                }}>
                                    <CircularProgress size={100} />
                                </Box>
                            }
                            {/* Celebratory confetti. To be used sparingly */}
                            {
                                celebrating && <Confetti />
                            }
                            <AlertDialog />
                            <Snack />
                            <AllRoutes
                                session={session ?? {}}
                                sessionChecked={session !== undefined}
                            />
                        </Box>
                        <BottomNav session={session ?? {}} />
                        <Footer />
                    </main>
                </div>
            </ThemeProvider>
        </StyledEngineProvider>
    );
}
