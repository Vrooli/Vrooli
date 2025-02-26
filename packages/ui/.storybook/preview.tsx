import { CssBaseline, GlobalStyles, ThemeProvider } from '@mui/material';
import type { Preview } from '@storybook/react';
import React from 'react';
import { I18nextProvider } from 'react-i18next';
import { getGlobalStyles } from '../src/App';
import { ActiveChatProvider, SessionContext, ZIndexProvider } from "../src/contexts.js";
import i18n from '../src/i18n';
import { DEFAULT_THEME, themes } from '../src/utils/display/theme';

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
    decorators: [
        (Story) => (
            <ThemeProvider theme={themes[DEFAULT_THEME]}>
                <CssBaseline />
                <GlobalStyles styles={getGlobalStyles(themes[DEFAULT_THEME])} />
                <I18nextProvider i18n={i18n}>
                    <SessionContext.Provider value={mockSession}>
                        <ZIndexProvider>
                            <ActiveChatProvider>
                                <Story />
                            </ActiveChatProvider>
                        </ZIndexProvider>
                    </SessionContext.Provider>
                </I18nextProvider>
            </ThemeProvider>
        ),
    ],
};

export default preview;