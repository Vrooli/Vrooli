import { useState, useEffect, useRef } from 'react';
import {
    AlertDialog,
    Footer,
    Navbar,
} from 'components';
import { PubSub, themes, useReactHash } from 'utils';
import { AllRoutes } from 'Routes';
import { Box, CssBaseline, CircularProgress, StyledEngineProvider, ThemeProvider } from '@mui/material';
import { makeStyles } from '@mui/styles';
import SakBunderan from './assets/font/SakBunderan.woff';

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
        },
        '@media (min-width:500px)': {
            '#page': {
                paddingLeft: 'max(1em, calc(15% - 75px))',
                paddingRight: 'max(1em, calc(15% - 75px))',
            }
        },
        '@font-face': {
            fontFamily: 'SakBunderan',
            src: `local('SakBunderan'), url(${SakBunderan}) format('truetype')`,
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
    const [theme, setTheme] = useState(themes.light);
    const [loading, setLoading] = useState(false);
    const timeoutRef = useRef<NodeJS.Timeout | null>(null);

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
        if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) setTheme(themes.dark);
        else setTheme(themes.light);
    }, [])

    useEffect(() => {
        // Handle loading spinner, which can have a delay
        let loadingSub = PubSub.get().subscribeLoading((data) => {
            if (timeoutRef.current) clearTimeout(timeoutRef.current);
            if (Number.isInteger(data)) {
                timeoutRef.current = setTimeout(() => setLoading(true), Math.abs(data as number));
            } else {
                setLoading(Boolean(data));
            }
        });
        let themeSub = PubSub.get().subscribeTheme((data) => setTheme(themes[data] ?? themes.light));
        return (() => {
            PubSub.get().unsubscribe(loadingSub);
            PubSub.get().unsubscribe(themeSub);
        })
    }, []);

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
                            <Navbar />
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
                            <AlertDialog />
                            <AllRoutes />
                        </Box>
                        <Footer />
                    </main>
                </Box>
            </ThemeProvider>
        </StyledEngineProvider>
    );
}
