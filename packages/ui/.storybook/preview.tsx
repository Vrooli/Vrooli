import { CssBaseline, GlobalStyles, ThemeProvider } from '@mui/material';
import type { Preview } from '@storybook/react';
import { HttpResponse, http } from 'msw';
import { initialize, mswLoader } from 'msw-storybook-addon';
import React, { useCallback, useEffect, useState } from 'react';
import { I18nextProvider } from 'react-i18next';
import { ContentWrap, getGlobalStyles } from '../src/App.js';
import { ChatSideMenu } from "../src/components/dialogs/ChatSideMenu/ChatSideMenu.js";
import { SideMenu } from "../src/components/dialogs/SideMenu/SideMenu.js";
import { VideoPopup } from "../src/components/dialogs/media.js";
import { ActiveChatProvider, SessionContext, ZIndexProvider } from "../src/contexts.js";
import { useSideMenu } from "../src/hooks/useSideMenu.js";
import { useWindowSize } from "../src/hooks/useWindowSize.js";
import i18n from '../src/i18n';
import { DEFAULT_THEME, themes } from '../src/utils/display/theme';
import { CHAT_SIDE_MENU_ID, PubSub, SIDE_MENU_ID, type PopupVideoPub } from '../src/utils/pubsub.js';

// Initialize MSW
initialize();

// Mock session for Storybook
const mockSession = {
    __typename: "Session" as const,
    isLoggedIn: true,
    users: [{ theme: DEFAULT_THEME }], // Minimal session data; adjust as needed
    // Add other properties if your components require them
};

const preview: Preview = {
    loaders: [mswLoader],
    parameters: {
        controls: {
            matchers: {
                color: /(background|color)$/i,
                date: /Date$/i,
            },
        },
        msw: {
            // Put global request overrides here, such as requests made by the side menus
            handlers: [
                // focus modes
                http.get("http://localhost:5329/api/v2/rest/focusModes", () => {
                    return HttpResponse.json({
                        edges: [],
                        pageInfo: {
                            __typename: "PageInfo" as const,
                            endCursor: null,
                            hasNextPage: false,
                        }
                    })
                })
            ],
        },
    },
    globalTypes: {
        themeMode: {
            name: 'Theme Mode',
            description: 'Switch between available themes',
            defaultValue: DEFAULT_THEME,
            toolbar: {
                icon: 'circlehollow',
                items: Object.keys(themes),
            },
        },
        isLeftHanded: {
            name: 'Is Left Handed',
            description: 'Switch between left and right handed mode',
            defaultValue: false,
            toolbar: {
                icon: 'circlehollow',
                items: [true, false],
            },
        },
    },
    decorators: [
        (Story, context) => {
            const themeMode = context.globals.themeMode || DEFAULT_THEME;
            const mockSessionWithTheme = {
                ...mockSession,
                users: [{ ...mockSession.users[0], theme: themeMode }],
            };
            const theme = themes[themeMode];
            const isLeftHanded = context.globals.isLeftHanded || false;

            const isMobile = useWindowSize(({ width }) => width <= theme.breakpoints.values.md);
            const { isOpen: isLeftDrawerOpen } = useSideMenu({ id: CHAT_SIDE_MENU_ID, isMobile });
            const { isOpen: isRightDrawerOpen } = useSideMenu({ id: SIDE_MENU_ID, isMobile });

            const [openVideoData, setOpenVideoData] = useState<PopupVideoPub | null>(null);
            const closePopupVideo = useCallback(function closePopupVideoCallback() {
                setOpenVideoData(null);
            }, []);

            useEffect(function handleSessionAndSubscriptions() {
                const popupVideoSub = PubSub.get().subscribe("popupVideo", data => {
                    setOpenVideoData(data);
                });
                return (() => {
                    popupVideoSub();
                });
            }, []);

            return (
                <ThemeProvider theme={theme}>
                    <CssBaseline />
                    <GlobalStyles styles={getGlobalStyles(theme)} />
                    <I18nextProvider i18n={i18n}>
                        <SessionContext.Provider value={mockSessionWithTheme}>
                            <ZIndexProvider>
                                <ActiveChatProvider>
                                    <VideoPopup
                                        open={!!openVideoData}
                                        onClose={closePopupVideo}
                                        src={openVideoData?.src ?? ""}
                                        zIndex={999999999}
                                    />
                                    <ContentWrap
                                        id="content-wrap"
                                        isLeftDrawerOpen={isLeftDrawerOpen}
                                        isLeftHanded={isLeftHanded}
                                        isMobile={isMobile}
                                        isRightDrawerOpen={isRightDrawerOpen}
                                    >
                                        <ChatSideMenu />
                                        <Story />
                                        <SideMenu />
                                    </ContentWrap>
                                </ActiveChatProvider>
                            </ZIndexProvider>
                        </SessionContext.Provider>
                    </I18nextProvider>
                </ThemeProvider>
            );
        },
    ],
};

export default preview;