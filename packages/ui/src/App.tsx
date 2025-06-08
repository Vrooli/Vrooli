import Box from "@mui/material/Box";
import CssBaseline from "@mui/material/CssBaseline";
import GlobalStyles from "@mui/material/GlobalStyles";
import { StyledEngineProvider } from "@mui/material/styles";
import { ThemeProvider } from "@mui/material/styles";
import { createTheme } from "@mui/material/styles";
import type { Theme } from "@mui/material";
import { styled } from "@mui/material";
import { endpointsAuth, type Session, type ValidateSessionInput } from "@vrooli/shared";
import i18next from "i18next";
import { useCallback, useEffect, useState } from "react";
import { fetchAIConfig } from "./api/ai.js";
import { fetchLazyWrapper } from "./api/fetchWrapper.js";
import { ServerResponseParser } from "./api/responseParser.js";
import { SERVER_CONNECT_MESSAGE_ID } from "./api/socket.js";
import { AdaptiveLayout } from "./components/AdaptiveLayout.js";
import { Celebration } from "./components/Celebration/Celebration.js";
import { DebugComponent } from "./components/Debug/index.js";
import { PullToRefresh } from "./components/PullToRefresh/PullToRefresh.js";
import { FullPageSpinner } from "./components/Spinners.js";
import { AlertDialog } from "./components/dialogs/AlertDialog/AlertDialog.js";
import { ProDialog } from "./components/dialogs/ProDialog/ProDialog.js";
import { TutorialDialog } from "./components/dialogs/TutorialDialog.js";
import { UserMenu } from "./components/dialogs/UserMenu/UserMenu.js";
import { ImagePopup, VideoPopup } from "./components/dialogs/media.js";
import { BottomNav } from "./components/navigation/BottomNav.js";
import { CommandPalette } from "./components/navigation/CommandPalette.js";
import { FindInPage } from "./components/navigation/FindInPage.js";
import { SnackStack } from "./components/snacks/SnackStack/SnackStack.js";
import { SessionContext } from "./contexts/session.js";
import { useHashScroll } from "./hooks/hash.js";
import { useLazyFetch } from "./hooks/useFetch.js";
import { useHotkeys } from "./hooks/useHotkeys.js";
import { useSocketConnect } from "./hooks/useSocketConnect.js";
import { useSocketUser } from "./hooks/useSocketUser.js";
import { useWindowSize } from "./hooks/useWindowSize.js";
import { CI_MODE } from "./i18n.js";
import { bottomNavHeight } from "./styles.js";
import { SessionService, guestSession } from "./utils/authentication/session.js";
import { ELEMENT_IDS, Z_INDEX } from "./utils/consts.js";
import { getDeviceInfo } from "./utils/display/device.js";
import { NODE_HIGHLIGHT_ERROR, NODE_HIGHLIGHT_SELECTED, NODE_HIGHLIGHT_WARNING, SEARCH_HIGHLIGHT_CURRENT, SEARCH_HIGHLIGHT_WRAPPER, SNACK_HIGHLIGHT, TUTORIAL_HIGHLIGHT } from "./utils/display/documentTools.js";
import { BREAKPOINTS, DEFAULT_THEME, themes } from "./utils/display/theme.js";
import { getCookie, getStorageItem, setCookie, type ThemeType } from "./utils/localStorage.js";
import { PubSub, type PopupImagePub, type PopupVideoPub } from "./utils/pubsub.js";
import commonSpriteHref from "/sprites/common-sprite.svg";

export function getGlobalStyles(theme: Theme) {
    return {
        html: {
            backgroundColor: theme.palette.background.default,
            overflow: "hidden", //Force children to handle scrolling. This makes it easier to support persistent sidebars
        },
        // Custom scrollbar
        "*": {
            "&::-webkit-scrollbar": {
                width: theme.spacing(1),
                height: theme.spacing(1),
            },
            "&::-webkit-scrollbar-track": {
                backgroundColor: "transparent",
            },
            "&::-webkit-scrollbar-thumb": {
                borderRadius: "100px",
                backgroundColor: theme.palette.mode === "light" ? "#b5c5ce" : "#45484a",
            },
            "&:hover::-webkit-scrollbar-thumb": {
                backgroundColor: "#98acc0",
            },
        },
        body: {
            fontFamily: "Roboto",
            fontWeight: 400,
            overflowX: "hidden",
            overflowY: "auto",
            // Disable padding for storybook
            padding: "0!important",
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
        [`.${SEARCH_HIGHLIGHT_WRAPPER}`]: {
            backgroundColor: "#ff0",
            color: "#000",
        },
        [`.${SEARCH_HIGHLIGHT_CURRENT}`]: {
            backgroundColor: "#3f0",
            color: "#000",
        },
        [`.${TUTORIAL_HIGHLIGHT}`]: {
            boxShadow: theme.palette.mode === "light"
                ? "inset 0 0 5px #339358, 0 0 10px #009e15"
                : "inset 0 0 5px #00cbff, 0 0 10px #00cbff",
            transition: "box-shadow 0.3s ease-in-out",
        },
        [`.${SNACK_HIGHLIGHT}`]: {
            boxShadow: theme.palette.mode === "light"
                ? "inset 0 0 3px #339358, 0 0 6px #009e15"
                : "inset 0 0 3px #00cbff, 0 0 6px #00cbff",
            transition: "box-shadow 0.2s ease-in-out",
        },
        [`.${NODE_HIGHLIGHT_ERROR}`]: {
            boxShadow: "0 0 6px 2px #f00",
        },
        [`.${NODE_HIGHLIGHT_WARNING}`]: {
            boxShadow: "0 0 6px 2px #ff0",
        },
        [`.${NODE_HIGHLIGHT_SELECTED}`]: {
            boxShadow: `0 0 6px 2px ${theme.palette.primary.main}`,
        },
        // Add custom fonts
        "@font-face": [
            // Logo
            {
                fontFamily: "sakbunderan",
                src: "url('/sakbunderan-logo-only-webfont.woff2') format('woff2')",
                fontWeight: "normal",
                fontStyle: "normal",
                fontDisplay: "swap",
            },
        ],
        // Ensure popovers are displayed above everything else
        ".MuiPopover-root": {
            zIndex: Z_INDEX.Popup,
        },
    };
}

/**
 * Sets up CSS variables that can be shared across components.
 */
export function useCssVariables() {
    const isMobile = useWindowSize(({ width }) => width <= BREAKPOINTS.md);

    useEffect(function pagPaddingBottomEffect() {
        // Page bottom padding depends on the existence of the BottomNav component, 
        // which only appears on mobile sizes.
        const paddingBottom = isMobile
            ? `calc(${bottomNavHeight} + env(safe-area-inset-bottom))`
            : "env(safe-area-inset-bottom)";
        document.documentElement.style.setProperty("--page-padding-bottom", paddingBottom);
    }, [isMobile]);
}

/** Adds font size to theme */
export function withFontSize(theme: Theme, fontSize: number): Theme {
    return createTheme({
        ...theme,
        typography: {
            fontSize,
        },
    });
}

/** Sets "isLeftHanded" property on theme */
export function withIsLeftHanded(theme: Theme, isLeftHanded: boolean): Theme {
    return createTheme({
        ...theme,
        isLeftHanded,
    });
}

/** Attempts to find theme without using session */
function findThemeWithoutSession(): Theme {
    const fontSize = getCookie("FontSize");
    const isLefthanded = getCookie("IsLeftHanded");
    const theme = getCookie("Theme");
    // Return theme object
    return withIsLeftHanded(withFontSize(themes[theme], fontSize), isLefthanded);
}

export const MainBox = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "row",
    minHeight: "100vh",
    minWidth: "100vw",
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
}));

export function App() {
    // Session cookie should automatically expire in time determined by server,
    // so no need to validate session on first load
    const [session, setSession] = useState<Session | undefined>(undefined);
    const [theme, setTheme] = useState<Theme>(findThemeWithoutSession());
    const [fontSize, setFontSize] = useState<number>(getCookie("FontSize"));
    const [language, setLanguage] = useState<string>(SessionService.getSiteLanguage(undefined));
    const [isLeftHanded, setIsLeftHanded] = useState<boolean>(getCookie("IsLeftHanded"));
    const [openImageData, setOpenImageData] = useState<PopupImagePub | null>(null);
    const [openVideoData, setOpenVideoData] = useState<PopupVideoPub | null>(null);
    const [isProDialogOpen, setIsProDialogOpen] = useState(false);
    const [validateSession] = useLazyFetch<ValidateSessionInput, Session>(endpointsAuth.validateSession);
    const isMobile = useWindowSize(({ width }) => width <= theme.breakpoints.values.md);
    useCssVariables();

    const closePopupImage = useCallback(function closePopupImageCallback() {
        setOpenImageData(null);
    }, []);
    const closePopupVideo = useCallback(function closePopupVideoCallback() {
        setOpenVideoData(null);
    }, []);
    const closeProDialog = useCallback(function closeProDialogCallback() {
        setIsProDialogOpen(false);
    }, []);

    useHashScroll();

    // Applies language change
    useEffect(() => {
        // Ignore if cimode (for testing) is enabled
        if (!CI_MODE) i18next.changeLanguage(language);
        // Refetch LLM config data, which is language-dependent
        fetchAIConfig(language);
    }, [language]);
    useEffect(() => {
        if (!session) return;
        if (SessionService.getSiteLanguage(session) !== SessionService.getSiteLanguage(undefined)) {
            setLanguage(SessionService.getSiteLanguage(session));
        }
    }, [session]);

    useEffect(function applyFontSizeEffect() {
        setTheme(withFontSize(themes[theme.palette.mode], fontSize));
    }, [fontSize, theme.palette.mode]);

    useEffect(function applyIsLeftHandedEffect() {
        setTheme(withIsLeftHanded(themes[theme.palette.mode], isLeftHanded));
    }, [isLeftHanded, theme.palette.mode]);

    /**
     * Sets theme state and meta tags. Meta tags allow standalone apps to
     * use the theme color as the status bar color.
     */
    const setThemeAndMeta = useCallback(function setThemeAndMetaCallback(theme: Theme) {
        // Update state
        setTheme(withIsLeftHanded(withFontSize(theme, fontSize), isLeftHanded));
        // Update meta tags, for theme-color and apple-mobile-web-app-status-bar-style
        document.querySelector("meta[name=\"theme-color\"]")?.setAttribute("content", theme.palette.primary.dark);
        document.querySelector("meta[name=\"apple-mobile-web-app-status-bar-style\"]")?.setAttribute("content", theme.palette.primary.dark);
        // Also store in local storage
        setCookie("Theme", theme.palette.mode);
    }, [fontSize, isLeftHanded]);

    // Handle component mount
    useEffect(() => {
        // Set up Google Adsense
        ((window as { adsbygoogle?: object[] }).adsbygoogle = (window as { adsbygoogle?: object[] }).adsbygoogle || []).push({});

        // Add help wanted to console logs
        const svgCode = `
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
    <use href="${commonSpriteHref}#vrooli-logo" fill="#ccc"/>
</svg>
        `;
        const svgDataUrl = `data:image/svg+xml,${encodeURIComponent(svgCode)}`;
        console.info(
            "%c ", `
            background-image: url(${svgDataUrl});
            font-size: 0;
            padding-left: 200px;
            background-size: contain;
            background-position: center center;
            background-repeat: no-repeat;
            line-height: 200px;
            `);
        console.info(`                                                                              
  Consider developing with us!                                         
https://github.com/Vrooli/Vrooli                            
`);
        // Detect online/offline status
        const onlineStatusId = "online-status"; // Use same ID for both so both can't be displayed at the same time
        function handleOnline() {
            PubSub.get().publish("snack", { id: onlineStatusId, message: i18next.t("NowOnline"), severity: "Success" });
        }
        function handleOffline() {
            PubSub.get().publish("snack", { autoHideDuration: "persist", id: onlineStatusId, message: i18next.t("NoInternet"), severity: "Error" });
        }
        window.addEventListener("online", handleOnline);
        window.addEventListener("offline", handleOffline);
        // Check if cookie banner should be shown. This is only a requirement for websites, not standalone apps.
        const cookiePreferences = getStorageItem("Preferences", () => true);
        const { isStandalone } = getDeviceInfo();
        if (!cookiePreferences && !isStandalone) {
            PubSub.get().publish("cookies");
        }
        return () => {
            window.removeEventListener("online", handleOnline);
            window.removeEventListener("offline", handleOffline);
        };
        // Make sure the dependency array here stays empty
    }, []);

    useEffect(() => {
        // Determine theme
        let theme: Theme | null | undefined;
        // Try getting theme from session
        if (Array.isArray(session?.users) && session?.users[0]?.theme) theme = themes[session?.users[0]?.theme as ThemeType];
        // If not found, try alternative methods
        if (!theme) theme = findThemeWithoutSession();
        // Update theme state, meta tags, and local storage
        setThemeAndMeta(theme);
    }, [session, setThemeAndMeta]);


    // Handle site-wide keyboard shortcuts
    useHotkeys([
        { keys: ["p"], ctrlKey: true, callback: () => { PubSub.get().publish("menu", { id: ELEMENT_IDS.CommandPalette, isOpen: true }); } },
        { keys: ["f"], ctrlKey: true, callback: () => { PubSub.get().publish("menu", { id: ELEMENT_IDS.FindInPage, isOpen: true }); } },
    ]);

    useEffect(function checkSessionEffect() {
        // Check if previous log in exists
        fetchLazyWrapper<ValidateSessionInput, Session>({
            fetch: validateSession,
            inputs: { timeZone: Intl.DateTimeFormat().resolvedOptions().timeZone },
            onSuccess: (data) => {
                localStorage.setItem("isLoggedIn", "true");
                setSession(data);
            },
            showDefaultErrorSnack: false,
            onError: (error) => {
                let isInvalidSession = false;
                localStorage.removeItem("isLoggedIn");
                // Check if error is expired/invalid session
                if (ServerResponseParser.hasErrorCode(error, "SessionExpired")) {
                    isInvalidSession = true;
                    // Log in development mode
                    if (process.env.DEV) console.error("Error: failed to verify session", error);
                }
                // If error is something else, notify user
                if (!isInvalidSession) {
                    PubSub.get().publish("snack", {
                        id: SERVER_CONNECT_MESSAGE_ID,
                        message: i18next.t("CannotConnectToServer"),
                        autoHideDuration: "persist",
                        severity: "Error",
                        buttonKey: "Reload",
                        buttonClicked: () => window.location.reload(),
                    });
                }
                // If not logged in as guest and failed to log in as user, set guest session
                setSession(guestSession);
            },
        });
    }, [validateSession]);

    useEffect(function handleSessionAndSubscriptions() {
        const sessionSub = PubSub.get().subscribe("session", async (session) => {
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
        const themeSub = PubSub.get().subscribe("theme", (data) => {
            const newTheme = themes[data] ?? themes[DEFAULT_THEME];
            setThemeAndMeta(newTheme);
        });
        // Handle font size updates
        const fontSizeSub = PubSub.get().subscribe("fontSize", (data) => {
            setFontSize(data);
            setCookie("FontSize", data);
        });
        // Handle language updates
        const languageSub = PubSub.get().subscribe("language", (data) => {
            setLanguage(data);
            setCookie("Language", data);
        });
        // Handle isLeftHanded updates
        const isLeftHandedSub = PubSub.get().subscribe("isLeftHanded", (data) => {
            setIsLeftHanded(data);
            setCookie("IsLeftHanded", data);
        });
        const popupImageSub = PubSub.get().subscribe("popupImage", (data) => {
            setOpenImageData(data);
        });
        const popupVideoSub = PubSub.get().subscribe("popupVideo", data => {
            setOpenVideoData(data);
        });
        const proDialogSub = PubSub.get().subscribe("proDialog", () => {
            setIsProDialogOpen(true);
        });
        return (() => {
            sessionSub();
            themeSub();
            fontSizeSub();
            languageSub();
            isLeftHandedSub();
            popupImageSub();
            popupVideoSub();
            proDialogSub();
        });
    }, [isLeftHanded, isMobile, setThemeAndMeta]);

    useSocketConnect();
    useSocketUser(session, setSession);

    return (
        <ThemeProvider theme={theme}>
            <StyledEngineProvider injectFirst>
                <GlobalStyles styles={getGlobalStyles(theme)} />
                <CssBaseline />
                <SessionContext.Provider value={session}>
                    <MainBox>
                        {/* Popups and other components that don't effect layout */}
                        <PullToRefresh />
                        <CommandPalette />
                        <FindInPage />
                        <Celebration />
                        <AlertDialog />
                        <SnackStack />
                        <ProDialog
                            isOpen={isProDialogOpen}
                            onClose={closeProDialog}
                        />
                        <TutorialDialog />
                        <ImagePopup
                            alt="Tutorial content"
                            open={!!openImageData}
                            onClose={closePopupImage}
                            src={openImageData?.src ?? ""}
                            zIndex={Z_INDEX.Popup}
                        />
                        <VideoPopup
                            open={!!openVideoData}
                            onClose={closePopupVideo}
                            src={openVideoData?.src ?? ""}
                            zIndex={Z_INDEX.Popup}
                        />
                        <UserMenu />
                        <FullPageSpinner />
                        <AdaptiveLayout />
                        <BottomNav />
                        {process.env.NODE_ENV === "development" && <DebugComponent />}
                    </MainBox>
                </SessionContext.Provider>
            </StyledEngineProvider>
        </ThemeProvider>
    );
}
