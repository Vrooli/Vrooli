import { ActiveFocusMode, endpointPostAuthValidateSession, endpointPutFocusModeActive, getActiveFocusMode, Session, SetActiveFocusModeInput, ValidateSessionInput } from "@local/shared";
import { Box, BoxProps, createTheme, CssBaseline, GlobalStyles, styled, StyledEngineProvider, Theme, ThemeProvider } from "@mui/material";
import { fetchAIConfig } from "api/ai";
import { hasErrorCode } from "api/errorParser";
import { fetchLazyWrapper } from "api/fetchWrapper";
import { BannerChicken } from "components/BannerChicken/BannerChicken";
import { Celebration } from "components/Celebration/Celebration";
import { DiagonalWaveLoader } from "components/DiagonalWaveLoader/DiagonalWaveLoader";
import { AlertDialog } from "components/dialogs/AlertDialog/AlertDialog";
import { ChatSideMenu } from "components/dialogs/ChatSideMenu/ChatSideMenu";
import { ImagePopup } from "components/dialogs/ImagePopup/ImagePopup";
import { SideMenu } from "components/dialogs/SideMenu/SideMenu";
import { TutorialDialog } from "components/dialogs/TutorialDialog/TutorialDialog";
import { VideoPopup } from "components/dialogs/VideoPopup/VideoPopup";
import { BottomNav } from "components/navigation/BottomNav/BottomNav";
import { CommandPalette } from "components/navigation/CommandPalette/CommandPalette";
import { FindInPage } from "components/navigation/FindInPage/FindInPage";
import { PullToRefresh } from "components/PullToRefresh/PullToRefresh";
import { SnackStack } from "components/snacks/SnackStack/SnackStack";
import { ActiveChatProvider, SessionContext, ZIndexProvider } from "contexts";
import { useHotkeys } from "hooks/useHotkeys";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useSideMenu } from "hooks/useSideMenu";
import { useSocketConnect } from "hooks/useSocketConnect";
import { useSocketUser } from "hooks/useSocketUser";
import { useWindowSize } from "hooks/useWindowSize";
import i18next from "i18next";
import { vrooliIconPath } from "icons/common";
import { useCallback, useEffect, useRef, useState } from "react";
import { Routes } from "Routes";
import { getCurrentUser, getSiteLanguage, guestSession } from "utils/authentication/session";
import { LEFT_DRAWER_WIDTH, RIGHT_DRAWER_WIDTH } from "utils/consts";
import { getDeviceInfo } from "utils/display/device";
import { DEFAULT_THEME, themes } from "utils/display/theme";
import { getCookie, getStorageItem, setCookie, ThemeType } from "utils/localStorage";
import { CHAT_SIDE_MENU_ID, PopupImagePub, PopupVideoPub, PubSub, SIDE_MENU_ID } from "utils/pubsub";
import { CI_MODE } from "./i18n";

function getGlobalStyles(theme: Theme) {
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
        ".tutorial-highlight": {
            boxShadow: theme.palette.mode === "light"
                ? "inset 0 0 5px #339358, 0 0 10px #009e15"
                : "inset 0 0 5px #00cbff, 0 0 10px #00cbff",
            transition: "box-shadow 0.3s ease-in-out",
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
            zIndex: 20000,
        },
    };
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

const MainBox = styled(Box)(({ theme }) => ({
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

interface ContentWrapProps extends BoxProps {
    isLeftDrawerOpen: boolean;
    isLeftHanded: boolean;
    isMobile: boolean;
    isRightDrawerOpen: boolean;
}

const ContentWrap = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isLeftDrawerOpen" && prop !== "isLeftHanded" && prop !== "isMobile" && prop !== "isRightDrawerOpen",
})<ContentWrapProps>(({ isLeftDrawerOpen, isLeftHanded, isMobile, isRightDrawerOpen, theme }) => {
    const leftDrawerWidth = isLeftHanded ? isRightDrawerOpen ? RIGHT_DRAWER_WIDTH : 0 : isLeftDrawerOpen ? LEFT_DRAWER_WIDTH : 0;
    const rightDrawerWidth = isLeftHanded ? isLeftDrawerOpen ? LEFT_DRAWER_WIDTH : 0 : isRightDrawerOpen ? RIGHT_DRAWER_WIDTH : 0;
    return {
        position: "relative",
        background: theme.palette.background.default,
        minHeight: "100vh",
        width: isMobile ? "100vw" : `calc(100vw - ${leftDrawerWidth}px - ${rightDrawerWidth}px)`,
        marginLeft: isMobile ? 0 : leftDrawerWidth,
        marginRight: isMobile ? 0 : rightDrawerWidth,
        transition: theme.transitions.create(["margin", "width"], {
            easing: theme.transitions.easing.easeOut,
            duration: theme.transitions.duration.enteringScreen,
        }),
        [theme.breakpoints.down("md")]: {
            minHeight: "calc(100vh - 56px - env(safe-area-inset-bottom))",
        },
    } as const;
});
const LoaderBox = styled(Box)(() => ({
    position: "absolute",
    top: "50%",
    left: "50%",
    transform: "translate(-50%, -50%)",
    zIndex: 100000,
}));

export function App() {
    // Session cookie should automatically expire in time determined by server,
    // so no need to validate session on first load
    const [session, setSession] = useState<Session | undefined>(undefined);
    const [theme, setTheme] = useState<Theme>(findThemeWithoutSession());
    const [fontSize, setFontSize] = useState<number>(getCookie("FontSize"));
    const [language, setLanguage] = useState<string>(getSiteLanguage(undefined));
    const [isLeftHanded, setIsLeftHanded] = useState<boolean>(getCookie("IsLeftHanded"));
    const [isLoading, setIsLoading] = useState(false);
    const [isTutorialOpen, setIsTutorialOpen] = useState(false);
    const [openImageData, setOpenImageData] = useState<PopupImagePub | null>(null);
    const [openVideoData, setOpenVideoData] = useState<PopupVideoPub | null>(null);
    const timeoutRef = useRef<NodeJS.Timeout | null>(null);
    const [validateSession] = useLazyFetch<ValidateSessionInput, Session>(endpointPostAuthValidateSession);
    const [setActiveFocusMode] = useLazyFetch<SetActiveFocusModeInput, ActiveFocusMode>(endpointPutFocusModeActive);
    const isSettingActiveFocusMode = useRef<boolean>(false);
    const isMobile = useWindowSize(({ width }) => width <= theme.breakpoints.values.md);
    const { isOpen: isLeftDrawerOpen } = useSideMenu({ id: CHAT_SIDE_MENU_ID, isMobile });
    const { isOpen: isRightDrawerOpen } = useSideMenu({ id: SIDE_MENU_ID, isMobile });

    const openTutorial = useCallback(function openTutorialCallback() {
        setIsTutorialOpen(true);
    }, []);
    const closeTutorial = useCallback(function closeTutorialCallback() {
        setIsTutorialOpen(false);
    }, []);
    const closePopupImage = useCallback(function closePopupImageCallback() {
        setOpenImageData(null);
    }, []);
    const closePopupVideo = useCallback(function closePopupVideoCallback() {
        setOpenVideoData(null);
    }, []);

    // Applies language change
    useEffect(() => {
        // Ignore if cimode (for testing) is enabled
        if (!CI_MODE) i18next.changeLanguage(language);
        // Refetch LLM config data, which is language-dependent
        fetchAIConfig(language);
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
        // Clear loading state
        if (timeoutRef.current) clearTimeout(timeoutRef.current);
        setIsLoading(false);
        // Add help wanted to console logs
        const svgCode = `
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
  <path d="${vrooliIconPath}" fill="#ccc"/>
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
            PubSub.get().publish("snack", { id: onlineStatusId, messageKey: "NowOnline", severity: "Success" });
        }
        function handleOffline() {
            PubSub.get().publish("snack", { autoHideDuration: "persist", id: onlineStatusId, messageKey: "NoInternet", severity: "Error" });
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
        { keys: ["p"], ctrlKey: true, callback: () => { PubSub.get().publish("commandPalette"); } },
        { keys: ["f"], ctrlKey: true, callback: () => { PubSub.get().publish("findInPage"); } },
    ]);

    const checkSession = useCallback(function checkSessionCallback(data?: Session) {
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
                localStorage.setItem("isLoggedIn", "true");
                setSession(data);
            },
            showDefaultErrorSnack: false,
            onError: (error) => {
                let isInvalidSession = false;
                localStorage.removeItem("isLoggedIn");
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

    useEffect(function handleSessionAndSubscriptions() {
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
            const activeFocusMode = await getActiveFocusMode(currentlyActiveFocusMode, focusModes);
            setCookie("FocusModeActive", activeFocusMode);
            setCookie("FocusModeAll", focusModes);
        });
        // Handle theme updates
        const themeSub = PubSub.get().subscribe("theme", (data) => {
            const newTheme = themes[data] ?? themes[DEFAULT_THEME];
            setThemeAndMeta(newTheme);
        });
        // Handle focus mode updates
        const focusModeSub = PubSub.get().subscribe("focusMode", (data) => {
            setCookie("FocusModeActive", data);
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
        // Handle tutorial popup
        const tutorialSub = PubSub.get().subscribe("tutorial", () => {
            setIsTutorialOpen(true);
        });
        const popupImageSub = PubSub.get().subscribe("popupImage", (data) => {
            setOpenImageData(data);
        });
        const popupVideoSub = PubSub.get().subscribe("popupVideo", data => {
            setOpenVideoData(data);
        });
        return (() => {
            loadingSub();
            sessionSub();
            themeSub();
            focusModeSub();
            fontSizeSub();
            languageSub();
            isLeftHandedSub();
            tutorialSub();
            popupImageSub();
            popupVideoSub();
        });
    }, [checkSession, isLeftHanded, isMobile, setActiveFocusMode, setThemeAndMeta]);

    useSocketConnect();
    useSocketUser(session, setSession);

    return (
        <>
            <StyledEngineProvider injectFirst>
                <CssBaseline />
                <ThemeProvider theme={theme}>
                    <GlobalStyles styles={getGlobalStyles} />
                    <SessionContext.Provider value={session}>
                        <ZIndexProvider>
                            <ActiveChatProvider>
                                <MainBox id="App" component="main">
                                    {/* Popups and other components that don't effect layout */}
                                    <PullToRefresh />
                                    <CommandPalette />
                                    <FindInPage />
                                    <Celebration />
                                    <AlertDialog />
                                    <SnackStack />
                                    <TutorialDialog
                                        isOpen={isTutorialOpen}
                                        onClose={closeTutorial}
                                        onOpen={openTutorial}
                                    />
                                    <ImagePopup
                                        alt="Tutorial content"
                                        open={!!openImageData}
                                        onClose={closePopupImage}
                                        src={openImageData?.src ?? ""}
                                        zIndex={999999999}
                                    />
                                    <VideoPopup
                                        open={!!openVideoData}
                                        onClose={closePopupVideo}
                                        src={openVideoData?.src ?? ""}
                                        zIndex={999999999}
                                    />
                                    {/* Main content*/}
                                    <ContentWrap
                                        id="content-wrap"
                                        isLeftDrawerOpen={isLeftDrawerOpen}
                                        isLeftHanded={isLeftHanded}
                                        isMobile={isMobile}
                                        isRightDrawerOpen={isRightDrawerOpen}
                                    >
                                        <ChatSideMenu />
                                        {
                                            isLoading && <LoaderBox>
                                                <DiagonalWaveLoader size={100} />
                                            </LoaderBox>
                                        }
                                        <Routes sessionChecked={session !== undefined} />
                                        <SideMenu />
                                    </ContentWrap>
                                    {/* Below main content */}
                                    <BannerChicken
                                        backgroundColor={theme.palette.background.default}
                                        isMobile={isMobile}
                                    />
                                    <BottomNav />
                                </MainBox>
                            </ActiveChatProvider>
                        </ZIndexProvider>
                    </SessionContext.Provider>
                </ThemeProvider>
            </StyledEngineProvider>
        </>
    );
}
