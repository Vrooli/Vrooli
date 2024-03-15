import { ActiveFocusMode, endpointPostAuthValidateSession, endpointPutFocusModeActive, getActiveFocusMode, Session, SetActiveFocusModeInput, ValidateSessionInput } from "@local/shared";
import { Box, createTheme, CssBaseline, GlobalStyles, StyledEngineProvider, Theme, ThemeProvider } from "@mui/material";
import { fetchLazyWrapper, hasErrorCode } from "api";
import { BannerChicken } from "components/BannerChicken/BannerChicken";
import { Celebration } from "components/Celebration/Celebration";
import { DiagonalWaveLoader } from "components/DiagonalWaveLoader/DiagonalWaveLoader";
import { AlertDialog } from "components/dialogs/AlertDialog/AlertDialog";
import { chatSideMenuDisplayData } from "components/dialogs/ChatSideMenu/ChatSideMenu";
import { SideMenu, sideMenuDisplayData } from "components/dialogs/SideMenu/SideMenu";
import { TutorialDialog } from "components/dialogs/TutorialDialog/TutorialDialog";
import { BottomNav } from "components/navigation/BottomNav/BottomNav";
import { CommandPalette } from "components/navigation/CommandPalette/CommandPalette";
import { FindInPage } from "components/navigation/FindInPage/FindInPage";
import { Footer } from "components/navigation/Footer/Footer";
import { PullToRefresh } from "components/PullToRefresh/PullToRefresh";
import { SnackStack } from "components/snacks";
import { SessionContext } from "contexts/SessionContext";
import { ZIndexProvider } from "contexts/ZIndexContext";
import { useHotkeys } from "hooks/useHotkeys";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useReactHash } from "hooks/useReactHash";
import { useSocketConnect } from "hooks/useSocketConnect";
import { useSocketUser } from "hooks/useSocketUser";
import { useWindowSize } from "hooks/useWindowSize";
import i18next from "i18next";
import { useCallback, useEffect, useRef, useState } from "react";
import { Routes } from "Routes";
import { getCurrentUser, getSiteLanguage, guestSession } from "utils/authentication/session";
import { getCookieFontSize, getCookieIsLeftHanded, getCookiePreferences, getCookieTheme, setCookieActiveFocusMode, setCookieAllFocusModes, setCookieFontSize, setCookieIsLeftHanded, setCookieLanguage, setCookieTheme } from "utils/cookies";
import { DEFAULT_THEME, themes } from "utils/display/theme";
import { PubSub, SideMenuPub } from "utils/pubsub";
import { CI_MODE } from "./i18n";

/** Adds font size to theme */
export const withFontSize = (theme: Theme, fontSize: number): Theme => createTheme({
    ...theme,
    typography: {
        fontSize,
    },
});

/** Sets "isLeftHanded" property on theme */
export const withIsLeftHanded = (theme: Theme, isLeftHanded: boolean): Theme => createTheme({
    ...theme,
    isLeftHanded,
});

/** Attempts to find theme without using session */
const findThemeWithoutSession = (): Theme => {
    // Get font size from cookie
    const fontSize = getCookieFontSize(14);
    // Get isLeftHanded from cookie
    const isLefthanded = getCookieIsLeftHanded(false);
    // Get theme. First check cookie, then window
    const windowPrefersLight = window.matchMedia && window.matchMedia("(prefers-color-scheme: light)").matches;
    const theme = getCookieTheme(windowPrefersLight ? "light" : "dark");
    // Return theme object
    return withIsLeftHanded(withFontSize(themes[theme], fontSize), isLefthanded);
};

const menusDisplayData: { [key in SideMenuPub["id"]]: { persistentOnDesktop: boolean, sideForRightHanded: "left" | "right" } } = {
    "side-menu": sideMenuDisplayData,
    "chat-side-menu": chatSideMenuDisplayData,
};

export const App = () => {
    // Session cookie should automatically expire in time determined by server,
    // so no need to validate session on first load
    const [session, setSession] = useState<Session | undefined>(undefined);
    const [theme, setTheme] = useState<Theme>(findThemeWithoutSession());
    const [fontSize, setFontSize] = useState<number>(getCookieFontSize(14));
    const [language, setLanguage] = useState<string>(getSiteLanguage(undefined));
    const [isLeftHanded, setIsLeftHanded] = useState<boolean>(getCookieIsLeftHanded(false));
    const [isLoading, setIsLoading] = useState(false);
    const [isTutorialOpen, setIsTutorialOpen] = useState(false);
    const timeoutRef = useRef<NodeJS.Timeout | null>(null);
    const [validateSession] = useLazyFetch<ValidateSessionInput, Session>(endpointPostAuthValidateSession);
    const [setActiveFocusMode] = useLazyFetch<SetActiveFocusModeInput, ActiveFocusMode>(endpointPutFocusModeActive);
    const isSettingActiveFocusMode = useRef<boolean>(false);
    const [contentMargins, setContentMargins] = useState<{ marginLeft?: string, marginRight?: string }>({}); // Adds margins to content when a persistent drawer is open
    const isMobile = useWindowSize(({ width }) => width <= theme.breakpoints.values.md);

    // Applies language change
    useEffect(() => {
        // Ignore if cimode (for testing) is enabled
        if (!CI_MODE) i18next.changeLanguage(language);
    }, [language]);
    useEffect(() => {
        if (!session) return;
        if (getSiteLanguage(session) !== getSiteLanguage(undefined)) {
            setLanguage(getSiteLanguage(session));
        }
    }, [session]);

    // Applies font size change
    useEffect(() => {
        setTheme(withFontSize(themes[theme.palette.mode], fontSize));
    }, [fontSize, theme.palette.mode]);

    // Applies isLeftHanded change
    useEffect(() => {
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
        document.querySelector("meta[name=\"theme-color\"]")?.setAttribute("content", theme.palette.primary.dark);
        document.querySelector("meta[name=\"apple-mobile-web-app-status-bar-style\"]")?.setAttribute("content", theme.palette.primary.dark);
        // Also store in local storage
        setCookieTheme(theme.palette.mode);
    }, [fontSize, isLeftHanded]);

    /** Sets up google adsense */
    useEffect(() => {
        ((window as { adsbygoogle?: object[] }).adsbygoogle = (window as { adsbygoogle?: object[] }).adsbygoogle || []).push({});
    }, []);

    // If anchor tag in url, scroll to element
    const hash = useReactHash();
    useEffect(() => {
        // if not a hash link, scroll to top
        if (window.location.hash === "") {
            window.scrollTo(0, 0);
        }
        // else scroll to id
        else {
            setTimeout(() => {
                const id = window.location.hash.replace("#", "");
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
        console.info(`                                               
                               !G!              
                               #@@!   :?J.      
                              .&@@#.:5@@&.      
                               B@@@?#@@@?       
                               J@@@Y#@@Y        
                               .B@@JB#!         
                                J@@GGPJ?7^.     
                             .J#@@@@@@@@###Y:   
 :!!:                       ~G@@@@@@@@@  ^&@&~  
?@@@&?#@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@##@@@5  
!#@@#?&@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@#~  
 .^^ :@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@&BJ:   
     :&@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@GY?~:      
     :&@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@&#GY^      
     :@@@@@&#####################&&&@@@@@@^     
     :@@@@@J                       :^!YBBY.     
      5&@@#^                                    
       :^^.                                     
                                                
         Consider developing with us!                                         
       https://github.com/Vrooli/Vrooli                            
`);
    }, []);

    useEffect(() => {
        // Determine theme
        let theme: Theme | null | undefined;
        // Try getting theme from session
        if (Array.isArray(session?.users) && session?.users[0]?.theme) theme = themes[session?.users[0]?.theme as "light" | "dark"];
        // If not found, try alternative methods
        if (!theme) theme = findThemeWithoutSession();
        // Update theme state, meta tags, and local storage
        setThemeAndMeta(theme);
    }, [session, setThemeAndMeta]);

    // Detect online/offline status, as well as "This site uses cookies" banner
    useEffect(() => {
        window.addEventListener("online", () => {
            PubSub.get().publish("snack", { id: "online-status", messageKey: "NowOnline", severity: "Success" });
        });
        window.addEventListener("offline", () => {
            // ID is the same so there is ever only one online/offline snack displayed at a time
            PubSub.get().publish("snack", { autoHideDuration: "persist", id: "online-status", messageKey: "NoInternet", severity: "Error" });
        });
        // Check if cookie banner should be shown. This is only a requirement for websites, not standalone apps.
        const cookiePreferences = getCookiePreferences();
        if (!cookiePreferences) {
            PubSub.get().publish("cookies");
        }
    }, []);

    // Handle site-wide keyboard shortcuts
    useHotkeys([
        { keys: ["p"], ctrlKey: true, callback: () => { PubSub.get().publish("commandPalette"); } },
        { keys: ["f"], ctrlKey: true, callback: () => { PubSub.get().publish("findInPage"); } },
    ]);

    const checkSession = useCallback((data?: Session) => {
        if (data) {
            setSession(data);
            return;
        }
        // Check if previous log in exists
        fetchLazyWrapper<ValidateSessionInput, Session>({
            fetch: validateSession,
            inputs: { timeZone: Intl.DateTimeFormat().resolvedOptions().timeZone },
            onSuccess: (data) => {
                //TODO store in local storage. validateSession will only return full data for the current user.
                //Other logged in users will not have their full data returned (will be in shape SessionUserToken
                //instead of SessionUser). Not sure if this is a problem yet.
                setSession(data);
            },
            showDefaultErrorSnack: false,
            onError: (error) => {
                let isInvalidSession = false;
                // Check if error is expired/invalid session
                if (hasErrorCode(error, "SessionExpired")) {
                    isInvalidSession = true;
                    // Log in development mode
                    if (process.env.DEV) console.error("Error: failed to verify session", error);
                }
                // If error is something else, notify user
                if (!isInvalidSession) {
                    PubSub.get().publish("snack", {
                        messageKey: "CannotConnectToServer",
                        autoHideDuration: "persist",
                        severity: "Error",
                        buttonKey: "Reload",
                        buttonClicked: () => window.location.reload(),
                    });
                }
                // If not logged in as guest and failed to log in as user, set guest session
                if (!data) {
                    setSession(guestSession);
                }
            },
        });
    }, [validateSession]);

    useEffect(() => {
        checkSession();
        // Handle session updates
        const loadingSub = PubSub.get().subscribe("loading", (data) => {
            if (timeoutRef.current) clearTimeout(timeoutRef.current);
            if (Number.isInteger(data)) {
                timeoutRef.current = setTimeout(() => setIsLoading(true), Math.abs(data as number));
            } else {
                setIsLoading(Boolean(data));
            }
        });
        const sessionSub = PubSub.get().subscribe("session", async (session) => {
            // If undefined or empty, set session to published data
            if (session === undefined || Object.keys(session).length === 0) {
                setSession(session);
            }
            // Otherwise, combine existing session data with published data
            else {
                setSession(s => ({ ...s, ...session }));
            }
            // Store user's focus modes in local storage
            const currentlyActiveFocusMode = getCurrentUser(session)?.activeFocusMode ?? null;
            const focusModes = getCurrentUser(session)?.focusModes ?? [];
            setCookieActiveFocusMode(await getActiveFocusMode(currentlyActiveFocusMode, focusModes));
            setCookieAllFocusModes(focusModes);
        });
        // Handle theme updates
        const themeSub = PubSub.get().subscribe("theme", (data) => {
            const newTheme = themes[data] ?? themes[DEFAULT_THEME];
            setThemeAndMeta(newTheme);
        });
        // Handle focus mode updates
        const focusModeSub = PubSub.get().subscribe("focusMode", (data) => {
            setCookieActiveFocusMode(data);
            setSession((prevState) => {
                if (!prevState) return prevState;
                const updatedUsers = prevState?.users?.map((user, index) => {
                    if (index === 0) {
                        return {
                            ...user,
                            activeFocusMode: data,
                        };
                    }
                    return user;
                });
                return {
                    ...prevState,
                    users: updatedUsers ?? [],
                };
            });
            if (!isSettingActiveFocusMode.current) {
                isSettingActiveFocusMode.current = true;
                const { mode, ...rest } = data;
                fetchLazyWrapper<SetActiveFocusModeInput, ActiveFocusMode>({
                    fetch: setActiveFocusMode,
                    inputs: { ...rest, id: data.mode.id },
                    successCondition: (data) => data !== null,
                    onSuccess: () => { isSettingActiveFocusMode.current = false; },
                    onError: (error) => {
                        isSettingActiveFocusMode.current = false;
                        console.error("Failed to set active focus mode", error);
                    },
                });
            }
        });
        // Handle font size updates
        const fontSizeSub = PubSub.get().subscribe("fontSize", (data) => {
            setFontSize(data);
            setCookieFontSize(data);
        });
        // Handle language updates
        const languageSub = PubSub.get().subscribe("language", (data) => {
            setLanguage(data);
            setCookieLanguage(data);
        });
        // Handle isLeftHanded updates
        const isLeftHandedSub = PubSub.get().subscribe("isLeftHanded", (data) => {
            setIsLeftHanded(data);
            setCookieIsLeftHanded(data);
        });
        // Handle tutorial popup
        const tutorialSub = PubSub.get().subscribe("tutorial", () => {
            setIsTutorialOpen(true);
        });
        // Handle content margins when drawer(s) open/close
        const sideMenuPub = PubSub.get().subscribe("sideMenu", (data) => {
            const { persistentOnDesktop, sideForRightHanded } = menusDisplayData[data.id];
            // Ignore if dialog is not persistent on desktop
            if (!persistentOnDesktop) return;
            // For now, ignore if "idPrefix" is present. This is currently only used for menus associated with dialogs
            if (data.idPrefix) return;
            // Flip side when in left-handed mode
            const side = isLeftHanded ? (sideForRightHanded === "left" ? "right" : "left") : sideForRightHanded;
            const menuElement = document.getElementById(data.id);
            const margin = data.isOpen && !isMobile ? `${menuElement?.clientWidth ?? 0}px` : "0px";
            // Only set on desktop
            if (side === "left") {
                setContentMargins(existing => ({ ...existing, marginLeft: margin }));
            } else if (side === "right") {
                setContentMargins(existing => ({ ...existing, marginRight: margin }));
            }
        });
        // On unmount, unsubscribe from all PubSub topics
        return (() => {
            loadingSub();
            sessionSub();
            themeSub();
            focusModeSub();
            fontSizeSub();
            languageSub();
            isLeftHandedSub();
            tutorialSub();
            sideMenuPub();
        });
    }, [checkSession, isLeftHanded, isMobile, setActiveFocusMode, setThemeAndMeta]);

    useSocketConnect();
    useSocketUser(session, setSession);

    return (
        <>
            <StyledEngineProvider injectFirst>
                <CssBaseline />
                <ThemeProvider theme={theme}>
                    <GlobalStyles
                        styles={(theme) => ({
                            html: {
                                backgroundColor: theme.palette.background.default,
                            },
                            // Custom scrollbar
                            "*": {
                                "&::-webkit-scrollbar": {
                                    width: 10,
                                },
                                "&::-webkit-scrollbar-track": {
                                    backgroundColor: "transparent",
                                },
                                "&::-webkit-scrollbar-thumb": {
                                    borderRadius: "100px",
                                    backgroundColor: "transparent", // Set initial color as transparent
                                },
                                "&:hover::-webkit-scrollbar-thumb": {
                                    backgroundColor: "#18587d85",  // Change color on hover
                                },
                            },
                            body: {
                                fontFamily: "Roboto",
                                fontWeight: 400,
                                overflowX: "hidden",
                                overflowY: "auto",
                                // Scrollbar should always be visible for the body
                                "&::-webkit-scrollbar-thumb": {
                                    backgroundColor: "#18587d85",
                                },
                            },
                            // Custom IconButton hover highlighting, which doesn't hide background color
                            ".MuiIconButton-root": {
                                "&:hover": {
                                    filter: "brightness(1.3)",
                                },
                                "&.Mui-disabled": {
                                    pointerEvents: "none",
                                    filter: "grayscale(1) opacity(0.5)",
                                },
                                transition: "filter 0.2s ease-in-out",
                            },
                            // Style bullet points in unordered lists
                            ul: {
                                listStyle: "circle",
                            },
                            // Search highlight classes
                            ".search-highlight": {
                                backgroundColor: "#ff0",
                                color: "#000",
                            },
                            ".search-highlight-current": {
                                backgroundColor: "#3f0",
                                color: "#000",
                            },
                            // Add custom fonts
                            "@font-face": [
                                // Logo
                                {
                                    fontFamily: "SakBunderan",
                                    src: "local('SakBunderan'), url(data:font/woff;charset=utf-8;base64,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) format('truetype')",
                                    fontDisplay: "swap",
                                },
                            ],
                            // Ensure popovers are displayed above everything else
                            ".MuiPopover-root": {
                                zIndex: 20000,
                            },
                        })}
                    />
                    <SessionContext.Provider value={session}>
                        <ZIndexProvider>
                            <Box id="App" component="main" sx={{
                                background: theme.palette.background.default,
                                color: theme.palette.background.textPrimary,
                                // Style visited, active, and hovered links
                                "& span, p": {
                                    "& a": {
                                        color: theme.palette.mode === "light" ? "#001cd3" : "#dd86db",
                                        "&:visited": {
                                            color: theme.palette.mode === "light" ? "#001cd3" : "#f551ef",
                                        },
                                        "&:active": {
                                            color: theme.palette.mode === "light" ? "#001cd3" : "#f551ef",
                                        },
                                        "&:hover": {
                                            color: theme.palette.mode === "light" ? "#5a6ff6" : "#f3d4f2",
                                        },
                                        // Remove underline on links
                                        textDecoration: "none",
                                    },
                                },
                            }}>
                                {/* Pull-to-refresh for PWAs */}
                                <PullToRefresh />
                                <CommandPalette />
                                <FindInPage />
                                <Celebration />
                                <AlertDialog />
                                <SnackStack />
                                <TutorialDialog isOpen={isTutorialOpen} onClose={() => setIsTutorialOpen(false)} />
                                <SideMenu />
                                <Box id="content-wrap" sx={{
                                    position: "relative",
                                    background: theme.palette.mode === "light" ? "#c2cadd" : theme.palette.background.default,
                                    minHeight: { xs: "calc(100vh - 56px - env(safe-area-inset-bottom))", md: "100vh" },
                                    ...(contentMargins),
                                    transition: "margin 0.225s cubic-bezier(0, 0, 0.2, 1) 0s",
                                }}>

                                    {/* Progress bar */}
                                    {
                                        isLoading && <Box sx={{
                                            position: "absolute",
                                            top: "50%",
                                            left: "50%",
                                            transform: "translate(-50%, -50%)",
                                            zIndex: 100000,
                                        }}>
                                            <DiagonalWaveLoader size={100} />
                                        </Box>
                                    }
                                    <Routes sessionChecked={session !== undefined} />
                                </Box>
                                <BottomNav />
                                <BannerChicken />
                                <Footer />
                            </Box>
                        </ZIndexProvider>
                    </SessionContext.Provider>
                </ThemeProvider>
            </StyledEngineProvider>
        </>
    );
};
