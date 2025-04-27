import { CodeLanguage, StripeEndpoint, User, uuid } from '@local/shared';
import { CssBaseline, GlobalStyles, ThemeProvider, createTheme } from '@mui/material';
import type { Preview } from '@storybook/react';
import { HttpResponse, http } from 'msw';
import { initialize, mswLoader } from 'msw-storybook-addon';
import React, { useCallback, useEffect, useState } from 'react';
import { I18nextProvider } from 'react-i18next';
import { MainBox, getGlobalStyles, useCssVariables } from '../src/App.js';
import { API_URL, baseSession } from '../src/__test/storybookConsts.js';
import { AdaptiveLayout } from '../src/components/AdaptiveLayout.js';
import { Celebration } from '../src/components/Celebration/Celebration.js';
import { DebugComponent } from '../src/components/Debug/index.js';
import { PullToRefresh } from '../src/components/PullToRefresh/PullToRefresh.js';
import { FullPageSpinner } from '../src/components/Spinners.js';
import { AlertDialog } from '../src/components/dialogs/AlertDialog/AlertDialog.js';
import { TutorialDialog } from '../src/components/dialogs/TutorialDialog.js';
import { UserMenu } from '../src/components/dialogs/UserMenu/UserMenu.js';
import { VideoPopup } from "../src/components/dialogs/media.js";
import { BottomNav } from '../src/components/navigation/BottomNav.js';
import { CommandPalette } from '../src/components/navigation/CommandPalette.js';
import { FindInPage } from '../src/components/navigation/FindInPage.js';
import { SnackStack } from '../src/components/snacks/SnackStack/SnackStack.js';
import { SessionContext } from "../src/contexts/session.js";
import { useHotkeys } from '../src/hooks/useHotkeys.js';
import i18n from '../src/i18n';
import { clearMockedLocationForStorybook, mockLocationForStorybook } from '../src/route/useLocation.js';
import { ELEMENT_IDS, Z_INDEX } from '../src/utils/consts.js';
import { DEFAULT_THEME, themes } from '../src/utils/display/theme';
import { PubSub, type PopupVideoPub } from '../src/utils/pubsub.js';

// Initialize MSW
initialize({
    onUnhandledRequest: "bypass",
});

// Mock profile for settings and other views
let mockProfile: Partial<User> = {
    id: uuid(),
    name: 'John Doe',
    apiKeys: [
        { __typename: "ApiKey" as const, id: '1', name: 'Key 1' },
        { __typename: "ApiKey" as const, id: '2', name: 'Key 2' },
    ],
    externalServiceKeys: {
        openAI: '',
        googleMaps: '',
    },
    // Add other necessary profile fields as required
};

function generateDummyStandards() {
    const response = {
        data: {
            edges: Array.from({ length: 20 }, (_, index) => {
                const hasCompleteVersion = Math.random() > 0.5;
                const standardName = "Standard " + Math.floor(Math.random() * 100);
                return {
                    cursor: String(index + 1),
                    node: {
                        __typename: "Standard" as const,
                        id: uuid(),
                        bookmarks: Math.floor(Math.random() * 100),
                        hasCompleteVersion,
                        isDeleted: false,
                        score: Math.floor(Math.random() * 100),
                        translatedName: standardName,
                        versions: [
                            {
                                __typename: "StandardVersion" as const,
                                id: uuid(),
                                codeLanguage: CodeLanguage.Javascript,
                                isComplete: hasCompleteVersion,
                                translations: [
                                    {
                                        __typename: "StandardVersionTranslation" as const,
                                        id: uuid(),
                                        language: "en",
                                        name: standardName,
                                    },
                                ],
                            },
                        ],
                        you: {
                            __typename: "StandardYou" as const,
                            canBookmark: true,
                            canDelete: false,
                            canReact: true,
                            canRead: true,
                            isViewed: false,
                        },
                    },
                };
            }),
            pageInfo: {
                __typename: "PageInfo" as const,
                endCursor: null,
                hasNextPage: false,
            },
        },
    };
    return response;
}

const isLeftHandedMapping = {
    "Right-handed (default)": false,
    "Left-handed": true,
};

const fontSizeMapping = {
    small: 12,
    medium: 16,
    large: 20,
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
                // api keys
                http.post(`${API_URL}/v2/rest/apikeys`, async ({ request }) => {
                    const input = await request.json();
                    const newKey = {
                        id: `key-${Date.now()}`,
                        name: input.name || 'New Key',
                        createdAt: new Date().toISOString(),
                        key: `mock-key-${Math.random().toString(36).substring(2)}`, // Simulate server-generated key
                    };
                    mockProfile.apiKeys?.push(newKey as ApiKey);
                    return HttpResponse.json({
                        data: newKey,
                    });
                }),
                http.put(`${API_URL}/v2/rest/apikeys/:id`, async ({ params, request }) => {
                    const id = params.id;
                    const input = await request.json();
                    const keyIndex = mockProfile.apiKeys?.findIndex(key => key.id === id);
                    if (keyIndex !== -1) {
                        mockProfile.apiKeys[keyIndex] = {
                            ...mockProfile.apiKeys[keyIndex],
                            ...input,
                        };
                        return HttpResponse.json({
                            data: mockProfile.apiKeys[keyIndex],
                        });
                    }
                    return HttpResponse.json({ error: 'Key not found' }, { status: 404 });
                }),
                http.delete(`${API_URL}/v2/rest/apikeys/:id`, ({ params }) => {
                    const id = params.id;
                    const keyIndex = mockProfile.apiKeys.findIndex(key => key.id === id);
                    if (keyIndex !== -1) {
                        mockProfile.apiKeys.splice(keyIndex, 1);
                        return HttpResponse.json({ success: true });
                    }
                    return HttpResponse.json({ error: 'Key not found' }, { status: 404 });
                }),
                // pricing
                http.get(`${API_URL}${StripeEndpoint.SubscriptionPrices}`, () => {
                    return HttpResponse.json({
                        data: {
                            monthly: 2500,
                            yearly: 2000 * 12,
                        },
                    });
                }),
                // Search lists
                http.get(`${API_URL}/v2/rest/feed/popular*`, () => {
                    return HttpResponse.json(generateDummyStandards());
                }),
                http.get(`${API_URL}/v2/rest/standards*`, () => {
                    return HttpResponse.json(generateDummyStandards());
                }),
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
            defaultValue: "Right-handed (default)",
            toolbar: {
                icon: 'circlehollow',
                items: ["Right-handed (default)", "Left-handed"],
            },
        },
        fontSize: {
            name: 'Font Size',
            description: 'Switch between font sizes',
            defaultValue: 'medium',
            toolbar: {
                icon: 'circlehollow',
                items: ['small', 'medium', 'large'],
            },
        },
    },
    decorators: [
        (Story, context) => {
            const themeMode = context.globals.themeMode || DEFAULT_THEME;
            const sessionOverrides = context.parameters.session || {};
            const finalSession = {
                ...baseSession,
                ...sessionOverrides,
                users: sessionOverrides.users
                    ? sessionOverrides.users.map(user => ({ ...user, theme: themeMode }))
                    : baseSession.users.map(user => ({ ...user, theme: themeMode })),
            };
            const theme = createTheme({
                ...themes[themeMode],
                typography: {
                    fontSize: fontSizeMapping[context.globals.fontSize] || fontSizeMapping.medium,
                },
                isLeftHanded: isLeftHandedMapping[context.globals.isLeftHanded] || false,
            });
            const routeOverrides = context.parameters.route || {};
            if (routeOverrides.path) {
                mockLocationForStorybook(routeOverrides.path);
            } else {
                clearMockedLocationForStorybook();
            }

            useCssVariables();

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

            // Handle site-wide keyboard shortcuts
            useHotkeys([
                { keys: ["p"], ctrlKey: true, callback: () => { PubSub.get().publish("menu", { id: ELEMENT_IDS.CommandPalette, isOpen: true }); } },
                { keys: ["f"], ctrlKey: true, callback: () => { PubSub.get().publish("menu", { id: ELEMENT_IDS.FindInPage, isOpen: true }); } },
            ]);

            return (
                <ThemeProvider theme={theme}>
                    <CssBaseline />
                    <GlobalStyles styles={getGlobalStyles(theme)} />
                    <I18nextProvider i18n={i18n}>
                        <SessionContext.Provider value={finalSession}>
                            <MainBox id="App" component="main">
                                <PullToRefresh />
                                <CommandPalette />
                                <FindInPage />
                                <Celebration />
                                <AlertDialog />
                                <SnackStack />
                                <TutorialDialog />
                                <VideoPopup
                                    open={!!openVideoData}
                                    onClose={closePopupVideo}
                                    src={openVideoData?.src ?? ""}
                                    zIndex={Z_INDEX.Popup}
                                />
                                <UserMenu />
                                <FullPageSpinner />
                                <AdaptiveLayout Story={Story} />
                                <BottomNav />
                                <DebugComponent />
                            </MainBox>
                        </SessionContext.Provider>
                    </I18nextProvider>
                </ThemeProvider>
            );
        },
    ],
};

export default preview;