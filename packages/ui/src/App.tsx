import { useState, useEffect, useCallback, useRef } from 'react';
import {
    AlertDialog,
    BottomNav,
    CommandPalette,
    FindInPage,
    Footer,
    Navbar,
    Snack
} from 'components';
import { PubSub, themes, useReactHash } from 'utils';
import { AllRoutes } from 'Routes';
import { Box, CssBaseline, CircularProgress, StyledEngineProvider, ThemeProvider } from '@mui/material';
import { makeStyles } from '@mui/styles';
import { ApolloError, useMutation } from '@apollo/client';
import { validateSessionMutation } from 'graphql/mutation';
import SakBunderan from './assets/font/SakBunderan.woff';
import { Session } from 'types';
import Confetti from 'react-confetti';
import { CODE } from '@shared/consts';

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
        // Style bullet points in unordered lists
        ul: {
            listStyle: 'circle',
        },
        // Search highlight classes
        '.search-highlight': {
            backgroundColor: '#ff0',
            color: '#000',
        },
        '.search-highlight-current': {
            backgroundColor: '#3f0',
            color: '#000',
        },
        '#page': {
            minWidth: '100vw',
            minHeight: '100vh',
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
        // Add help wanted to console logs
        console.log(`
               @@@                 @@@                  
            @@     @@           @@     @@               
           @@       @@         @@       @@              
            @@     @@           @@     @@               
               @@@   @@        @   @@@                  
                        @   @@                   @@@      
                         @@                   @@     @@
                         @@             @@@@@@@       @@
            @@           @@          @@       @@     @@
    @@@  @@    @@    @@@    @@@   @@             @@@ 
 @@     @@         @@          @@                     
@@       @@       @@            @@                                  
 @@     @@        @             @@@@@@@@@@                Consider developing with us!  
    @@@           @@            @@        @@@@@           https://github.com/Vrooli/Vrooli
                   @@          @@                @      
                     @@@    @@@                  @      
                         @@                     @@@   
       @@@              @@@@                 @@     @@  
    @@     @@@@@@@@@@@@      @@             @@       @@ 
   @@       @@      @@        @@             @@     @@ 
    @@     @@        @@      @@                 @@@           
       @@@              @@@@                             
                         @@ 
                       @@@@@@                        
                     @@      @@                        
                     @@      @@                        
                       @@@@@@  
        `)
    }, []);

    useEffect(() => {
        // Determine theme
        if (session?.theme) setTheme(themes[session?.theme])
        else if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) setTheme(themes.dark);
        else setTheme(themes.light);
    }, [session])

    // Handle site-wide keyboard shortcuts
    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            // CTRL + P - Opens Command Palette
            if (e.ctrlKey && e.key === 'p') {
                e.preventDefault();
                PubSub.get().publishCommandPalette();
            }
            // CTRL + F - Opens Find in Page
            else if (e.ctrlKey && e.key === 'f') {
                e.preventDefault();
                PubSub.get().publishFindInPage();
            }
        };

        // attach the event listener
        document.addEventListener('keydown', handleKeyDown);
        // remove the event listener
        return () => {
            document.removeEventListener('keydown', handleKeyDown);
        };
    }, []);

    const checkSession = useCallback((session?: any) => {
        if (session) {
            setSession(session);
            return;
        }
        // Check if previous log in exists
        validateSession().then(({ data }) => {
            setSession(data?.validateSession as Session);
        }).catch((response: ApolloError) => {
            // Check if error is expired/invalid session
            let isInvalidSession = false;
            if (response.graphQLErrors && response.graphQLErrors.length > 0) {
                const error = response.graphQLErrors[0];
                if (error.extensions.code === CODE.SessionExpired.code) {
                    isInvalidSession = true;
                    // Log in development mode
                    if (process.env.NODE_ENV === 'development') console.error('Error: failed to verify session', response);
                }
            }
            // If error is something else, notify user
            if (!isInvalidSession) {
                PubSub.get().publishSnack({ message: 'Failed to connect to server.', severity: 'error' });
            }
            // If not logged in as guest and failed to log in as user, set empty object
            if (!session) setSession({})
        })
    }, [validateSession])

    useEffect(() => {
        checkSession();
        // Handle loading spinner, which can have a delay
        let loadingSub = PubSub.get().subscribeLoading((data) => {
            if (timeoutRef.current) clearTimeout(timeoutRef.current);
            if (Number.isInteger(data)) {
                timeoutRef.current = setTimeout(() => setLoading(true), Math.abs(data as number));
            } else {
                setLoading(Boolean(data));
            }
        });
        // Handle celebration (confetti). Defaults to 5 seconds long, but duration 
        // can be passed in as a number
        let celebrationSub = PubSub.get().subscribeCelebration((data) => {
            // Start confetti immediately
            setCelebrating(true);
            // Determine duration
            let duration = 5000;
            if (Number.isInteger(data)) duration = data as number;
            // Stop confetti after duration
            setTimeout(() => setCelebrating(false), duration);
        });
        let sessionSub = PubSub.get().subscribeSession((session) => {
            // If undefined or empty, set session to published data
            if (session === undefined || Object.keys(session).length === 0) {
                setSession(session);
            }
            // Otherwise, combine existing session data with published data
            else {
                setSession(s => ({ ...s, ...session }));
            }
        });
        let themeSub = PubSub.get().subscribeTheme((data) => setTheme(themes[data] ?? themes.light));
        return (() => {
            PubSub.get().unsubscribe(loadingSub);
            PubSub.get().unsubscribe(celebrationSub);
            PubSub.get().unsubscribe(sessionSub);
            PubSub.get().unsubscribe(themeSub);
        })
    }, [checkSession]);

    return (
        <StyledEngineProvider injectFirst>
            <CssBaseline />
            <ThemeProvider theme={theme}>
                <Box id="App" sx={{
                    // Style visited, active, and hovered links differently
                    a: {
                        color: theme.palette.mode === 'light' ? '#001cd3' : '#dd86db',
                        '&:visited': {
                            color: theme.palette.mode === 'light' ? '#001cd3' : '#f551ef',
                        },
                        '&:active': {
                            color: theme.palette.mode === 'light' ? '#001cd3' : '#f551ef',
                        },
                        '&:hover': {
                            color: theme.palette.mode === 'light' ? '#5a6ff6' : '#f3d4f2',
                        },
                        // Remove underline on links
                        textDecoration: 'none',
                    },
                }}>
                    <main
                        id="page-container"
                        style={{
                            background: theme.palette.background.default,
                            color: theme.palette.background.textPrimary,
                        }}
                    >
                        <Box id="content-wrap" sx={{
                            background: theme.palette.mode === 'light' ? '#c2cadd' : theme.palette.background.default,
                            // xs: 100vh - bottom nav (56px) - iOS nav bar
                            // md: 100vh
                            minHeight: { xs: 'calc(100vh - 56px - env(safe-area-inset-bottom))', md: '100vh' },
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
                            {/* Command palette */}
                            <CommandPalette session={session ?? {}} />
                            {/* Find in page */}
                            <FindInPage />
                            {/* Celebratory confetti. To be used sparingly */}
                            {
                                celebrating && <Confetti
                                    initialVelocityY={-10}
                                    recycle={false}
                                    confettiSource={{
                                        x: 0,
                                        y: 40,
                                        w: window.innerWidth,
                                        h: 0
                                    }}
                                />
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
                </Box>
            </ThemeProvider>
        </StyledEngineProvider>
    );
}
