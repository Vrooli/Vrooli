import { Box, CircularProgress, createTheme, CssBaseline, StyledEngineProvider, Theme, ThemeProvider } from '@mui/material';
import { makeStyles } from '@mui/styles';
import { Session, ValidateSessionInput } from '@shared/consts';
import { authValidateSession } from 'api/generated/endpoints/auth_validateSession';
import { useCustomMutation } from 'api/hooks';
import { hasErrorCode, mutationWrapper } from 'api/utils';
import { BannerChicken } from 'components/BannerChicken/BannerChicken';
import { AlertDialog } from 'components/dialogs/AlertDialog/AlertDialog';
import { SnackStack } from 'components/dialogs/snacks';
import { WelcomeDialog } from 'components/dialogs/WelcomeDialog/WelcomeDialog';
import { BottomNav } from 'components/navigation/BottomNav/BottomNav';
import { CommandPalette } from 'components/navigation/CommandPalette/CommandPalette';
import { FindInPage } from 'components/navigation/FindInPage/FindInPage';
import { Footer } from 'components/navigation/Footer/Footer';
import { PullToRefresh } from 'components/PullToRefresh/PullToRefresh';
import i18next from 'i18next';
import { useCallback, useEffect, useRef, useState } from 'react';
import Confetti from 'react-confetti';
import { Routes } from 'Routes';
import { getSiteLanguage, guestSession } from 'utils/authentication/session';
import { getCookieFontSize, getCookieIsLeftHanded, getCookiePreferences, getCookieTheme, setCookieFontSize, setCookieIsLeftHanded, setCookieLanguage, setCookieTheme } from 'utils/cookies';
import { getDeviceInfo } from 'utils/display/device';
import { themes } from 'utils/display/theme';
import { useReactHash } from 'utils/hooks/useReactHash';
import { PubSub } from 'utils/pubsub';
import SakBunderan from './assets/font/SakBunderan.woff';

/**
 * Adds font size to theme
 */
const withFontSize = (theme: Theme, fontSize: number): Theme => createTheme({
    ...theme,
    typography: {
        fontSize,
    },
})

/**
 * Sets "isLeftHanded" property on theme
 */
const withIsLeftHanded = (theme: Theme, isLeftHanded: boolean): Theme => createTheme({
    ...theme,
    isLeftHanded,
})

/**
 * Attempts to find theme without using session, defaulting to light
 */
const findThemeWithoutSession = (): Theme => {
    // Get font size from cookie
    const fontSize = getCookieFontSize() ?? 14;
    // Get isLeftHanded from cookie
    const isLefthanded = getCookieIsLeftHanded() ?? false;
    // Get theme. First check cookie, then window
    const windowPrefersDark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
    const theme = getCookieTheme() ?? (windowPrefersDark ? 'dark' : 'light');
    // Return theme object
    return withIsLeftHanded(withFontSize(themes[theme], fontSize), isLefthanded);
}

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
        '@font-face': {
            fontFamily: 'SakBunderan',
            src: `local('SakBunderan'), url(${SakBunderan}) format('truetype')`,
            fontDisplay: 'swap',
        },
        // Ensure popovers are displayed above everything else
        '.MuiPopover-root': {
            zIndex: 20000,
        }
    },
}));

export function App() {
    useStyles();
    // Session cookie should automatically expire in time determined by server,
    // so no need to validate session on first load
    const [session, setSession] = useState<Session | undefined>(undefined);
    const [theme, setTheme] = useState<Theme>(findThemeWithoutSession());
    const [fontSize, setFontSize] = useState(getCookieFontSize() ?? 14);
    const [language, setLanguage] = useState(getSiteLanguage(undefined));
    const [isLeftHanded, setIsLeftHanded] = useState(getCookieIsLeftHanded() ?? false);
    const [isLoading, setIsLoading] = useState(false);
    const [isCelebrating, setIsCelebrating] = useState(false);
    const [isWelcomeDialogOpen, setIsWelcomeDialogOpen] = useState(false);
    const timeoutRef = useRef<NodeJS.Timeout | null>(null);
    const [validateSession] = useCustomMutation<Session, ValidateSessionInput>(authValidateSession);

    // Applies language change
    useEffect(() => {
        console.log('language change 1', language)
        i18next.changeLanguage(language);
    }, [language]);
    useEffect(() => {
        console.log('language session thing', session, getSiteLanguage(session), getSiteLanguage(undefined))
        if (!session) return;
        if (getSiteLanguage(session) !== getSiteLanguage(undefined)) {
            setLanguage(getSiteLanguage(session));
        }
    }, [session]);
    console.log('themeeeeeeeee!!!!!!!!!!!!!', theme)

    // Applies font size change
    useEffect(() => {
        console.log('Applying font size change', fontSize, withFontSize(themes[theme.palette.mode], fontSize));
        setTheme(withFontSize(themes[theme.palette.mode], fontSize));
    }, [fontSize, theme.palette.mode]);

    // Applies isLeftHanded change
    useEffect(() => {
        console.log('Applying isLeftHanded change', isLeftHanded);
        setTheme(withIsLeftHanded(themes[theme.palette.mode], isLeftHanded));
    }, [isLeftHanded, theme.palette.mode]);

    /**
     * Sets theme state and meta tags. Meta tags allow standalone apps to
     * use the theme color as the status bar color.
     */
    const setThemeAndMeta = useCallback((theme: Theme) => {
        // Update state
        setTheme(withIsLeftHanded(withFontSize(theme, fontSize), isLeftHanded));
        // Update meta tags, for theme-color and apple-mobile-web-app-status-bar-style
        document.querySelector('meta[name="theme-color"]')?.setAttribute('content', theme.palette.primary.dark);
        document.querySelector('meta[name="apple-mobile-web-app-status-bar-style"]')?.setAttribute('content', theme.palette.primary.dark);
        // Also store in local storage
        setCookieTheme(theme.palette.mode);
    }, [fontSize, isLeftHanded]);

    /**
     * Sets up google adsense
     */
    useEffect(() => {
        ((window as any).adsbygoogle = (window as any).adsbygoogle || []).push({});
    }, []);

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
        setIsLoading(false);
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
        let theme: Theme | null | undefined;
        // Try getting theme from session
        if (Array.isArray(session?.users) && session?.users[0]?.theme) theme = themes[session?.users[0]?.theme as 'light' | 'dark'];
        // If not found, try alternative methods
        if (!theme) theme = findThemeWithoutSession();
        // Update theme state, meta tags, and local storage
        setThemeAndMeta(theme);
    }, [session, setThemeAndMeta])

    // Detect online/offline status, as well as "This site uses cookies" banner
    useEffect(() => {
        window.addEventListener('online', () => {
            PubSub.get().publishSnack({ id: 'online-status', messageKey: 'NowOnline', severity: 'Success' });
        });
        window.addEventListener('offline', () => {
            // ID is the same so there is ever only one online/offline snack displayed at a time
            PubSub.get().publishSnack({ autoHideDuration: 'persist', id: 'online-status', messageKey: 'NoInternet', severity: 'Error' });
        });
        // Check if cookie banner should be shown. This is only a requirement for websites, not standalone apps.
        const cookiePreferences = getCookiePreferences();
        if (!getDeviceInfo().isStandalone && !cookiePreferences) {
            PubSub.get().publishCookies();
        }
    }, []);

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

    const checkSession = useCallback((session?: Session) => {
        if (session) {
            setSession(session);
            return;
        }
        // Check if previous log in exists
        mutationWrapper<Session, ValidateSessionInput>({
            mutation: validateSession,
            input: { timeZone: Intl.DateTimeFormat().resolvedOptions().timeZone },
            onSuccess: (data) => { setSession(data) },
            onError: (error: any) => {
                let isInvalidSession = false;
                // Check if error is expired/invalid session
                if (hasErrorCode(error, 'SessionExpired')) {
                    isInvalidSession = true;
                    // Log in development mode
                    if (process.env.NODE_ENV === 'development') console.error('Error: failed to verify session', error);
                }
                // If error is something else, notify user
                if (!isInvalidSession) {
                    PubSub.get().publishSnack({
                        messageKey: 'CannotConnectToServer',
                        severity: 'Error',
                        buttonKey: 'Reload',
                        buttonClicked: () => window.location.reload(),
                    });
                }
                // If not logged in as guest and failed to log in as user, set guest session
                if (!session) {
                    setSession(guestSession)
                }
            },
        })
    }, [validateSession])

    useEffect(() => {
        checkSession();
        // Handle loading spinner, which can have a delay
        let loadingSub = PubSub.get().subscribeLoading((data) => {
            if (timeoutRef.current) clearTimeout(timeoutRef.current);
            if (Number.isInteger(data)) {
                timeoutRef.current = setTimeout(() => setIsLoading(true), Math.abs(data as number));
            } else {
                setIsLoading(Boolean(data));
            }
        });
        // Handle celebration (confetti). Defaults to 5 seconds long, but duration 
        // can be passed in as a number
        let celebrationSub = PubSub.get().subscribeCelebration((data) => {
            // Start confetti immediately
            setIsCelebrating(true);
            // Determine duration
            let duration = 5000;
            if (Number.isInteger(data)) duration = data as number;
            // Stop confetti after duration
            setTimeout(() => setIsCelebrating(false), duration);
        });
        // Handle session updates
        let sessionSub = PubSub.get().subscribeSession((session) => {
            console.log('action setting session from sub', session)
            // If undefined or empty, set session to published data
            if (session === undefined || Object.keys(session).length === 0) {
                setSession(session);
            }
            // Otherwise, combine existing session data with published data
            else {
                setSession(s => ({ ...s, ...session }));
            }
        });
        // Handle theme updates
        let themeSub = PubSub.get().subscribeTheme((data) => {
            const newTheme = themes[data] ?? themes.light
            setThemeAndMeta(newTheme);
        });
        // Handle font size updates
        let fontSizeSub = PubSub.get().subscribeFontSize((data) => {
            setFontSize(data);
            setCookieFontSize(data);
        });
        // Handle language updates
        let languageSub = PubSub.get().subscribeLanguage((data) => {
            setLanguage(data);
            setCookieLanguage(data);
        });
        // Handle isLeftHanded updates
        let isLeftHandedSub = PubSub.get().subscribeIsLeftHanded((data) => {
            setIsLeftHanded(data);
            setCookieIsLeftHanded(data);
        });
        // Handle welcome message
        let welcomeSub = PubSub.get().subscribeWelcome(() => {
            setIsWelcomeDialogOpen(true);
        });
        // On unmount, unsubscribe from all PubSub topics
        return (() => {
            PubSub.get().unsubscribe(loadingSub);
            PubSub.get().unsubscribe(celebrationSub);
            PubSub.get().unsubscribe(sessionSub);
            PubSub.get().unsubscribe(themeSub);
            PubSub.get().unsubscribe(fontSizeSub);
            PubSub.get().unsubscribe(languageSub);
            PubSub.get().unsubscribe(isLeftHandedSub);
            PubSub.get().unsubscribe(welcomeSub);
        })
    }, [checkSession, setThemeAndMeta]);

    return (
        <StyledEngineProvider injectFirst>
            <CssBaseline />
            <ThemeProvider theme={theme}>
                <Box id="App" component="main" sx={{
                    background: theme.palette.background.default,
                    color: theme.palette.background.textPrimary,
                    // Style visited, active, and hovered links differently
                    // a: {
                    //     color: theme.palette.mode === 'light' ? '#001cd3' : '#dd86db',
                    //     '&:visited': {
                    //         color: theme.palette.mode === 'light' ? '#001cd3' : '#f551ef',
                    //     },
                    //     '&:active': {
                    //         color: theme.palette.mode === 'light' ? '#001cd3' : '#f551ef',
                    //     },
                    //     '&:hover': {
                    //         color: theme.palette.mode === 'light' ? '#5a6ff6' : '#f3d4f2',
                    //     },
                    //     // Remove underline on links
                    //     textDecoration: 'none',
                    // },
                }}>
                    {/* Pull-to-refresh for PWAs */}
                    <PullToRefresh />
                    {/* Command palette */}
                    <CommandPalette session={session} />
                    {/* Find in page */}
                    <FindInPage session={session} />
                    {/* WelcomeDialog */}
                    <WelcomeDialog
                        isOpen={isWelcomeDialogOpen}
                        onClose={() => setIsWelcomeDialogOpen(false)}
                        session={session}
                    />
                    {/* Celebratory confetti. To be used sparingly */}
                    {
                        isCelebrating && <Confetti
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
                    <SnackStack />
                    <Box id="content-wrap" sx={{
                        background: theme.palette.mode === 'light' ? '#c2cadd' : theme.palette.background.default,
                        minHeight: { xs: 'calc(100vh - 56px - env(safe-area-inset-bottom))', md: '100vh' },
                    }}>

                        {/* Progress bar */}
                        {
                            isLoading && <Box sx={{
                                position: 'absolute',
                                top: '50%',
                                left: '50%',
                                transform: 'translate(-50%, -50%)',
                                zIndex: 100000,
                            }}>
                                <CircularProgress size={100} />
                            </Box>
                        }
                        <Routes
                            session={session}
                            sessionChecked={session !== undefined}
                        />
                    </Box>
                    <BottomNav session={session} />
                    <BannerChicken session={session} />
                    <Footer />
                </Box>
            </ThemeProvider>
        </StyledEngineProvider>
    );
}
