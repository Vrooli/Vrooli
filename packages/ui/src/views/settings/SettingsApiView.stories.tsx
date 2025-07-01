/* eslint-disable no-magic-numbers */
import { ApiKeyPermission, generatePK, type ApiKey, type Session, type SessionUser, type User, type Resource, type ResourceVersion, type ResourceTranslation, type ResourceSearchResult, type ApiKeyCreated } from "@vrooli/shared";
import { HttpResponse, http } from "msw";
import { API_URL, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession, signedInPremiumNoCreditsSession } from "../../__test/storybookConsts.js";
import { getMockApiUrl } from "../../__test/helpers/storybookMocking.js";
import { PERMISSION_PRESETS, SettingsApiView } from "./SettingsApiView.js";

// Session data
const session: Partial<Session> = {
    isLoggedIn: true,
    users: [{
        credits: "12345678912",
        hasPremium: true,
        id: generatePK().toString(),
    }] as SessionUser[],
};

// Mock API resources data
const mockApiResources: ResourceSearchResult = {
    __typename: "ResourceSearchResult",
    edges: [
        {
            __typename: "ResourceEdge",
            cursor: "1",
            node: {
                __typename: "Resource",
                id: generatePK().toString(),
                isDeleted: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                isInternal: false,
                usedFor: "Api",
                versions: [
                    {
                        __typename: "ResourceVersion",
                        id: generatePK().toString(),
                        isLatest: true,
                        versionLabel: "1.0.0",
                        callLink: "https://api.openai.com",
                        configCallData: JSON.stringify({
                            authentication: {
                                type: "oauth2",
                                settings: {
                                    clientId: "openai-client-id",
                                    authUrl: "https://openai.com/oauth/authorize",
                                    tokenUrl: "https://openai.com/oauth/token",
                                    scopes: ["read", "write"],
                                },
                            },
                        }),
                        translations: [
                            {
                                __typename: "ResourceVersionTranslation",
                                id: generatePK().toString(),
                                language: "en",
                                name: "OpenAI API",
                                description: "Official OpenAI API for GPT models",
                            },
                        ],
                    },
                ],
            } as Resource,
        },
        {
            __typename: "ResourceEdge", 
            cursor: "2",
            node: {
                __typename: "Resource",
                id: generatePK().toString(),
                isDeleted: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                isInternal: false,
                usedFor: "Api",
                versions: [
                    {
                        __typename: "ResourceVersion",
                        id: generatePK().toString(),
                        isLatest: true,
                        versionLabel: "1.0.0",
                        callLink: "https://api.anthropic.com",
                        configCallData: JSON.stringify({
                            authentication: {
                                type: "apikey",
                                settings: {
                                    headerName: "x-api-key",
                                },
                            },
                        }),
                        translations: [
                            {
                                __typename: "ResourceVersionTranslation",
                                id: generatePK().toString(),
                                language: "en",
                                name: "Anthropic API",
                                description: "Anthropic Claude API",
                            },
                        ],
                    },
                ],
            } as Resource,
        },
        {
            __typename: "ResourceEdge",
            cursor: "3", 
            node: {
                __typename: "Resource",
                id: generatePK().toString(),
                isDeleted: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                isInternal: false,
                usedFor: "Api",
                versions: [
                    {
                        __typename: "ResourceVersion",
                        id: generatePK().toString(),
                        isLatest: true,
                        versionLabel: "1.0.0",
                        callLink: "https://api.github.com",
                        configCallData: JSON.stringify({
                            authentication: {
                                type: "none",
                            },
                        }),
                        translations: [
                            {
                                __typename: "ResourceVersionTranslation",
                                id: generatePK().toString(),
                                language: "en",
                                name: "GitHub API",
                                description: "GitHub REST API v4",
                            },
                        ],
                    },
                ],
            } as Resource,
        },
    ],
    pageInfo: {
        __typename: "PageInfo",
        hasNextPage: false,
        hasPreviousPage: false,
        startCursor: "1",
        endCursor: "3",
    },
};

const noKeysOrIntegrationData: Partial<User> = {
    apiKeys: [],
    apiKeysExternal: [],
};
const withKeysAndIntegrationsData: Partial<User> = {
    apiKeys: [
        // Fresh key with READ_ONLY permissions
        {
            __typename: "ApiKey" as const,
            id: generatePK().toString(),
            creditsUsed: BigInt(0).toString(),
            disabledAt: null,
            limitHard: BigInt(25000000000).toString(),
            limitSoft: null,
            name: "Read-Only Key",
            stopAtLimit: true,
            permissions: JSON.stringify(PERMISSION_PRESETS.READ_ONLY.permissions),
        } as ApiKey,
        // Used but still active key with STANDARD permissions
        {
            __typename: "ApiKey" as const,
            id: generatePK().toString(),
            creditsUsed: BigInt(1000000).toString(),
            disabledAt: null,
            limitHard: BigInt(25000000000).toString(),
            limitSoft: null,
            name: "Standard Key",
            stopAtLimit: true,
            permissions: JSON.stringify(PERMISSION_PRESETS.STANDARD.permissions),
        } as ApiKey,
        // Inactive key with DEVELOPER permissions
        {
            __typename: "ApiKey" as const,
            id: generatePK().toString(),
            creditsUsed: BigInt(1000000).toString(),
            disabledAt: new Date().toISOString(),
            limitHard: BigInt(25000000000).toString(),
            limitSoft: null,
            name: "Developer Key (Disabled)",
            stopAtLimit: true,
            permissions: JSON.stringify(PERMISSION_PRESETS.DEVELOPER.permissions),
        } as ApiKey,
        // Key with FULL_ACCESS permissions
        {
            __typename: "ApiKey" as const,
            id: generatePK().toString(),
            creditsUsed: BigInt(2000000).toString(),
            disabledAt: null,
            limitHard: BigInt(25000000000).toString(),
            limitSoft: BigInt(10000000000).toString(),
            name: "Full Access Key",
            stopAtLimit: true,
            permissions: JSON.stringify(PERMISSION_PRESETS.FULL_ACCESS.permissions),
        } as ApiKey,
        // Key with custom permissions
        {
            __typename: "ApiKey" as const,
            id: generatePK().toString(),
            creditsUsed: BigInt(500000).toString(),
            disabledAt: null,
            limitHard: BigInt(25000000000).toString(),
            limitSoft: null,
            name: "Custom Permissions Key",
            stopAtLimit: true,
            permissions: JSON.stringify([
                ApiKeyPermission.ReadPublic,
                ApiKeyPermission.ReadPrivate,
                ApiKeyPermission.ReadAuth,
            ]),
        } as ApiKey,
    ],
    apiKeysExternal: [
        {
            __typename: "ApiKeyExternal" as const,
            id: generatePK().toString(),
            disabledAt: null,
            name: "External Key 1",
            service: "OpenAI",
        },
        {
            __typename: "ApiKeyExternal" as const,
            id: generatePK().toString(),
            disabledAt: new Date().toISOString(),
            name: "External Key 2",
            service: "OpenAI",
        },
        {
            __typename: "ApiKeyExternal" as const,
            id: generatePK().toString(),
            disabledAt: null,
            name: "External Key 3",
            service: "Microsoft",
        },
    ],
};

// Common MSW handlers for API operations
const createCommonHandlers = (userData: Partial<User>) => [
    // Profile endpoint
    http.get(getMockApiUrl("/profile"), () => {
        console.log("Intercepted profile request, returning:", userData);
        return HttpResponse.json(userData);
    }),
    
    // Resources endpoint for loading external services
    http.post(getMockApiUrl("/resources"), () => {
        console.log("Intercepted resources request");
        return HttpResponse.json(mockApiResources);
    }),
    
    // Create internal API key
    http.post(getMockApiUrl("/apiKey"), async ({ request }) => {
        const body = await request.json() as any;
        const newKey: ApiKeyCreated = {
            __typename: "ApiKey",
            id: generatePK().toString(),
            creditsUsed: "0",
            disabledAt: null,
            limitHard: body.limitHard || "25000000000",
            limitSoft: body.limitSoft || null,
            name: body.name || "New API Key",
            stopAtLimit: body.stopAtLimit ?? true,
            permissions: body.permissions || JSON.stringify(PERMISSION_PRESETS.READ_ONLY.permissions),
            key: `sk-test-${generatePK().toString().slice(0, 16)}`, // Simulated API key
        };
        
        console.log("Created new API key:", newKey);
        return HttpResponse.json(newKey);
    }),
    
    // Update internal API key
    http.put(getMockApiUrl("/apiKey/:id"), async ({ request, params }) => {
        const body = await request.json() as any;
        const updatedKey: ApiKey = {
            __typename: "ApiKey",
            id: params.id as string,
            creditsUsed: "1000000",
            disabledAt: body.disabled ? new Date().toISOString() : null,
            limitHard: body.limitHard || "25000000000",
            limitSoft: body.limitSoft || null,
            name: body.name || "Updated API Key",
            stopAtLimit: body.stopAtLimit ?? true,
            permissions: body.permissions || JSON.stringify(PERMISSION_PRESETS.STANDARD.permissions),
        };
        
        console.log("Updated API key:", updatedKey);
        return HttpResponse.json(updatedKey);
    }),
    
    // Delete API key (internal or external)
    http.post(getMockApiUrl("/deleteOne"), () => {
        console.log("Deleting API key");
        return HttpResponse.json({ success: true });
    }),
    
    // Create external API key
    http.post(getMockApiUrl("/apiKeyExternal"), async ({ request }) => {
        const body = await request.json() as any;
        const newExternalKey = {
            __typename: "ApiKeyExternal",
            id: generatePK().toString(),
            disabledAt: null,
            name: body.name || "",
            service: body.service || "Custom Service",
            resourceId: body.resourceId,
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
        };
        
        console.log("Created external key:", newExternalKey);
        return HttpResponse.json(newExternalKey);
    }),
    
    // Update external API key
    http.put(getMockApiUrl("/apiKeyExternal/:id"), async ({ request, params }) => {
        const body = await request.json() as any;
        const updatedExternalKey = {
            __typename: "ApiKeyExternal", 
            id: params.id as string,
            disabledAt: null,
            name: body.name || "",
            service: body.service || "Updated Service",
            resourceId: body.resourceId,
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
        };
        
        console.log("Updated external key:", updatedExternalKey);
        return HttpResponse.json(updatedExternalKey);
    }),
    
    // OAuth initiate
    http.post(getMockApiUrl("/auth/oauth/initiate"), async ({ request }) => {
        const body = await request.json() as any;
        const result = {
            authUrl: `https://openai.com/oauth/authorize?client_id=test&redirect_uri=${encodeURIComponent(body.redirectUri)}&state=test-state-123`,
            state: "test-state-123",
        };
        console.log("OAuth initiate:", result);
        return HttpResponse.json(result);
    }),
    
    // OAuth callback
    http.post(getMockApiUrl("/auth/oauth/callback"), () => {
        const result = {
            success: true,
            provider: "openai",
        };
        console.log("OAuth callback:", result);
        return HttpResponse.json(result);
    }),
];

export function NoKeysOrIntegrations() {
    return (
        <SettingsApiView display="Page" />
    );
}
NoKeysOrIntegrations.parameters = {
    docs: {
        description: {
            story: "Displays the settings api view when the user has no keys or integrations.",
        },
    },
    msw: {
        handlers: [
            // Override the profile endpoint for this specific story
            http.get(getMockApiUrl("/profile"), () => {
                console.log("Returning no keys data");
                return HttpResponse.json(noKeysOrIntegrationData);
            }),
            // Include other handlers but filter out the default profile handler
            ...createCommonHandlers(noKeysOrIntegrationData).filter(handler => !handler.info.path.includes("/v2/profile")),
        ],
    },
    session,
};

export function WithKeysAndIntegrations() {
    return (
        <SettingsApiView display="Page" />
    );
}
WithKeysAndIntegrations.parameters = {
    docs: {
        description: {
            story: "Displays the settings api view when the user has keys and integrations.",
        },
    },
    session,
};

export function SignedInNoPremiumNoCredits() {
    return (
        <SettingsApiView display="Page" />
    );
}
SignedInNoPremiumNoCredits.parameters = {
    session: signedInNoPremiumNoCreditsSession,
    msw: {
        handlers: [
            http.get(getMockApiUrl("/profile"), () => {
                return HttpResponse.json(noKeysOrIntegrationData);
            }),
            ...createCommonHandlers(noKeysOrIntegrationData).filter(handler => !handler.info.path.includes("/v2/profile")),
        ],
    },
};

export function SignedInNoPremiumWithCredits() {
    return (
        <SettingsApiView display="Page" />
    );
}
SignedInNoPremiumWithCredits.parameters = {
    session: signedInNoPremiumWithCreditsSession,
};

export function SignedInPremiumNoCredits() {
    return (
        <SettingsApiView display="Page" />
    );
}
SignedInPremiumNoCredits.parameters = {
    session: signedInPremiumNoCreditsSession,
};

export function SignedInPremiumWithCredits() {
    return (
        <SettingsApiView display="Page" />
    );
}
SignedInPremiumWithCredits.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// New comprehensive stories

const newKeyData = {
    ...withKeysAndIntegrationsData,
    _newApiKey: "sk-test-1234567890abcdef", // Simulates newly created key for one-time display
};

export function WithNewlyCreatedKey() {
    return (
        <SettingsApiView display="Page" />
    );
}
WithNewlyCreatedKey.parameters = {
    docs: {
        description: {
            story: "Shows the one-time key display dialog after creating a new API key. This is critical for testing the key copy functionality.",
        },
    },
    session,
    msw: {
        handlers: [
            http.get(getMockApiUrl("/profile"), () => {
                return HttpResponse.json(newKeyData);
            }),
            ...createCommonHandlers(newKeyData).filter(handler => !handler.info.path.includes("/v2/profile")),
        ],
    },
};

const highUsageData: Partial<User> = {
    apiKeys: [
        {
            __typename: "ApiKey" as const,
            id: generatePK().toString(),
            creditsUsed: BigInt(24000000000).toString(), // 96% of limit
            disabledAt: null,
            limitHard: BigInt(25000000000).toString(),
            limitSoft: BigInt(20000000000).toString(),
            name: "High Usage Key",
            stopAtLimit: true,
            permissions: JSON.stringify(PERMISSION_PRESETS.STANDARD.permissions),
        } as ApiKey,
        {
            __typename: "ApiKey" as const,
            id: generatePK().toString(),
            creditsUsed: BigInt(26000000000).toString(), // Exceeds hard limit
            disabledAt: null,
            limitHard: BigInt(25000000000).toString(),
            limitSoft: null,
            name: "Exceeded Limit Key",
            stopAtLimit: true,
            permissions: JSON.stringify(PERMISSION_PRESETS.DEVELOPER.permissions),
        } as ApiKey,
    ],
    apiKeysExternal: [],
};

export function HighCreditUsage() {
    return (
        <SettingsApiView display="Page" />
    );
}
HighCreditUsage.parameters = {
    docs: {
        description: {
            story: "Shows API keys with high credit usage and exceeded limits to test warning indicators and limit enforcement UI.",
        },
    },
    session,
    msw: {
        handlers: [
            http.get(getMockApiUrl("/profile"), () => {
                return HttpResponse.json(highUsageData);
            }),
            ...createCommonHandlers(highUsageData).filter(handler => !handler.info.path.includes("/v2/profile")),
        ],
    },
};

const externalOnlyData: Partial<User> = {
    apiKeys: [],
    apiKeysExternal: [
        {
            __typename: "ApiKeyExternal" as const,
            id: generatePK().toString(),
            disabledAt: null,
            name: "OpenAI GPT-4",
            service: "openai",
        },
        {
            __typename: "ApiKeyExternal" as const,
            id: generatePK().toString(),
            disabledAt: null,
            name: "Anthropic Claude",
            service: "anthropic",
        },
        {
            __typename: "ApiKeyExternal" as const,
            id: generatePK().toString(),
            disabledAt: null,
            name: "GitHub Personal Token",
            service: "github",
        },
        {
            __typename: "ApiKeyExternal" as const,
            id: generatePK().toString(),
            disabledAt: new Date().toISOString(),
            name: "Disabled Key",
            service: "openai",
        },
    ],
};

export function ExternalKeysOnly() {
    return (
        <SettingsApiView display="Page" />
    );
}
ExternalKeysOnly.parameters = {
    docs: {
        description: {
            story: "Shows only external API keys without internal Vrooli keys. Useful for testing external key management in isolation.",
        },
    },
    session,
    msw: {
        handlers: [
            http.get(getMockApiUrl("/profile"), () => {
                return HttpResponse.json(externalOnlyData);
            }),
            ...createCommonHandlers(externalOnlyData).filter(handler => !handler.info.path.includes("/v2/profile")),
        ],
    },
};

const adminSession: Partial<Session> = {
    ...session,
    users: [{
        ...session.users![0],
        isAdmin: true,
    }] as SessionUser[],
};

// Default export with MSW handlers at the top level
export default {
    title: "Views/Settings/SettingsApiView",
    component: SettingsApiView,
    parameters: {
        msw: {
            handlers: [
                // Default handlers that apply to all stories unless overridden
                ...createCommonHandlers(withKeysAndIntegrationsData),
            ],
        },
    },
    decorators: [
        (Story) => {
            // Add debug helper to window for testing
            if (typeof window !== "undefined") {
                window.debugApiKeys = {
                    mockData: {
                        noKeys: noKeysOrIntegrationData,
                        withKeys: withKeysAndIntegrationsData,
                        newKey: newKeyData,
                        highUsage: highUsageData,
                        externalOnly: externalOnlyData,
                        customPermissions: customPermissionsData,
                        mixedStates: mixedStatesData,
                    },
                    resources: mockApiResources,
                };
                console.log("SettingsApiView debug data available at window.debugApiKeys");
            }

            return <Story />;  
        },
    ],
};

export function AdminUser() {
    return (
        <SettingsApiView display="Page" />
    );
}
AdminUser.parameters = {
    docs: {
        description: {
            story: "Shows the API view for an admin user. May display additional admin-specific options or information.",
        },
    },
    session: adminSession,
};

export function LoadingState() {
    return (
        <SettingsApiView display="Page" />
    );
}
LoadingState.parameters = {
    docs: {
        description: {
            story: "Shows the loading state while API keys are being fetched.",
        },
    },
    session,
    msw: {
        handlers: [
            http.get(getMockApiUrl("/profile"), () => {
                // Simulate slow loading
                return new Promise(resolve => {
                    setTimeout(() => {
                        resolve(HttpResponse.json(withKeysAndIntegrationsData));
                    }, 2000);
                });
            }),
            // Include other common handlers but override profile
            ...createCommonHandlers(withKeysAndIntegrationsData).filter(handler => !handler.info.path.includes("/v2/profile")),
        ],
    },
};

export function ErrorState() {
    return (
        <SettingsApiView display="Page" />
    );
}
ErrorState.parameters = {
    docs: {
        description: {
            story: "Shows error handling when API key data fails to load.",
        },
    },
    session,
    msw: {
        handlers: [
            http.get(getMockApiUrl("/profile"), () => {
                return HttpResponse.json(
                    { error: "Failed to fetch API keys" },
                    { status: 500 },
                );
            }),
            // Include other common handlers but override profile
            ...createCommonHandlers(withKeysAndIntegrationsData).filter(handler => !handler.info.path.includes("/v2/profile")),
        ],
    },
};

const customPermissionsData: Partial<User> = {
    apiKeys: [
        {
            __typename: "ApiKey" as const,
            id: generatePK().toString(),
            creditsUsed: BigInt(0).toString(),
            disabledAt: null,
            limitHard: BigInt(25000000000).toString(),
            limitSoft: null,
            name: "Low Security Custom",
            stopAtLimit: true,
            permissions: JSON.stringify([ApiKeyPermission.ReadPublic]),
        } as ApiKey,
        {
            __typename: "ApiKey" as const,
            id: generatePK().toString(),
            creditsUsed: BigInt(1000000).toString(),
            disabledAt: null,
            limitHard: BigInt(25000000000).toString(),
            limitSoft: null,
            name: "Medium Security Custom",
            stopAtLimit: true,
            permissions: JSON.stringify([
                ApiKeyPermission.ReadPublic,
                ApiKeyPermission.ReadPrivate,
                ApiKeyPermission.WritePrivate,
            ]),
        } as ApiKey,
        {
            __typename: "ApiKey" as const,
            id: generatePK().toString(),
            creditsUsed: BigInt(500000).toString(),
            disabledAt: null,
            limitHard: BigInt(25000000000).toString(),
            limitSoft: null,
            name: "High Security Custom",
            stopAtLimit: true,
            permissions: JSON.stringify([
                ApiKeyPermission.ReadPublic,
                ApiKeyPermission.ReadPrivate,
                ApiKeyPermission.WritePrivate,
                ApiKeyPermission.ReadAuth,
                ApiKeyPermission.WriteAuth,
            ]),
        } as ApiKey,
    ],
    apiKeysExternal: [],
};

export function CustomPermissions() {
    return (
        <SettingsApiView display="Page" />
    );
}
CustomPermissions.parameters = {
    docs: {
        description: {
            story: "Shows API keys with various custom permission combinations and their security level indicators.",
        },
    },
    session,
    msw: {
        handlers: [
            http.get(getMockApiUrl("/profile"), () => {
                return HttpResponse.json(customPermissionsData);
            }),
            ...createCommonHandlers(customPermissionsData).filter(handler => !handler.info.path.includes("/v2/profile")),
        ],
    },
};

const mixedStatesData: Partial<User> = {
    apiKeys: [
        {
            __typename: "ApiKey" as const,
            id: generatePK().toString(),
            creditsUsed: BigInt(0).toString(),
            disabledAt: null,
            limitHard: BigInt(25000000000).toString(),
            limitSoft: null,
            name: "Fresh Unused Key",
            stopAtLimit: true,
            permissions: JSON.stringify(PERMISSION_PRESETS.READ_ONLY.permissions),
        } as ApiKey,
        {
            __typename: "ApiKey" as const,
            id: generatePK().toString(),
            creditsUsed: BigInt(15000000000).toString(),
            disabledAt: null,
            limitHard: BigInt(25000000000).toString(),
            limitSoft: BigInt(20000000000).toString(),
            name: "Approaching Soft Limit",
            stopAtLimit: false,
            permissions: JSON.stringify(PERMISSION_PRESETS.STANDARD.permissions),
        } as ApiKey,
        {
            __typename: "ApiKey" as const,
            id: generatePK().toString(),
            creditsUsed: BigInt(22000000000).toString(),
            disabledAt: null,
            limitHard: BigInt(25000000000).toString(),
            limitSoft: BigInt(20000000000).toString(),
            name: "Exceeded Soft Limit",
            stopAtLimit: false,
            permissions: JSON.stringify(PERMISSION_PRESETS.DEVELOPER.permissions),
        } as ApiKey,
        {
            __typename: "ApiKey" as const,
            id: generatePK().toString(),
            creditsUsed: BigInt(25000000000).toString(),
            disabledAt: null,
            limitHard: BigInt(25000000000).toString(),
            limitSoft: null,
            name: "At Hard Limit",
            stopAtLimit: true,
            permissions: JSON.stringify(PERMISSION_PRESETS.FULL_ACCESS.permissions),
        } as ApiKey,
        {
            __typename: "ApiKey" as const,
            id: generatePK().toString(),
            creditsUsed: BigInt(5000000).toString(),
            disabledAt: new Date().toISOString(),
            limitHard: BigInt(25000000000).toString(),
            limitSoft: null,
            name: "Disabled Key",
            stopAtLimit: true,
            permissions: JSON.stringify(PERMISSION_PRESETS.STANDARD.permissions),
        } as ApiKey,
    ],
    apiKeysExternal: [
        {
            __typename: "ApiKeyExternal" as const,
            id: generatePK().toString(),
            disabledAt: null,
            name: "Active External Key",
            service: "openai",
        },
        {
            __typename: "ApiKeyExternal" as const,
            id: generatePK().toString(),
            disabledAt: new Date().toISOString(),
            name: "Disabled External Key",
            service: "anthropic",
        },
    ],
};

export function MixedKeyStates() {
    return (
        <SettingsApiView display="Page" />
    );
}
MixedKeyStates.parameters = {
    docs: {
        description: {
            story: "Comprehensive test showing various key states: fresh, approaching limits, exceeded limits, disabled, etc.",
        },
    },
    session,
    msw: {
        handlers: [
            http.get(getMockApiUrl("/profile"), () => {
                return HttpResponse.json(mixedStatesData);
            }),
            ...createCommonHandlers(mixedStatesData).filter(handler => !handler.info.path.includes("/v2/profile")),
        ],
    },
};
