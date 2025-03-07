import { CssBaseline, GlobalStyles, ThemeProvider } from '@mui/material';
import type { Preview } from '@storybook/react';
import React, { useCallback, useEffect, useState } from 'react';
import { I18nextProvider } from 'react-i18next';
import { getGlobalStyles } from '../src/App';
import { VideoPopup } from "../src/components/dialogs/media.js";
import { ActiveChatProvider, SessionContext, ZIndexProvider } from "../src/contexts.js";
import i18n from '../src/i18n';
import { DEFAULT_THEME, themes } from '../src/utils/display/theme';
import { PubSub, type PopupVideoPub } from '../src/utils/pubsub.js';

// Mock session for Storybook
const mockSession = {
    __typename: "Session" as const,
    isLoggedIn: true,
    users: [{ theme: DEFAULT_THEME }], // Minimal session data; adjust as needed
    // Add other properties if your components require them
};

const preview: Preview = {
    parameters: {
        controls: {
            matchers: {
                color: /(background|color)$/i,
                date: /Date$/i,
            },
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
    },
    decorators: [
        (Story, context) => {
            const themeMode = context.globals.themeMode || DEFAULT_THEME;
            const mockSessionWithTheme = {
                ...mockSession,
                users: [{ ...mockSession.users[0], theme: themeMode }],
            };
            const theme = themes[themeMode];

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
                                    <Story />
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