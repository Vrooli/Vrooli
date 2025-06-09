/* eslint-disable no-magic-numbers */
import { ApiKeyPermission, generatePK, type ApiKey, type Session, type SessionUser, type User } from "@vrooli/shared";
import { HttpResponse, http } from "msw";
import { API_URL, signedInNoPremiumNoCreditsSession, signedInNoPremiumWithCreditsSession, signedInPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../__test/storybookConsts.js";
import { PERMISSION_PRESETS, SettingsApiView } from "./SettingsApiView.js";

export default {
    title: "Views/Settings/SettingsApiView",
    component: SettingsApiView,
};

const session: Partial<Session> = {
    isLoggedIn: true,
    users: [{
        credits: "12345678912",
        hasPremium: true,
        id: generatePK().toString(),
    }] as SessionUser[],
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
            http.get(`${API_URL}/v2/profile`, () => {
                return HttpResponse.json({
                    data: noKeysOrIntegrationData,
                });
            }),
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
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/profile`, () => {
                return HttpResponse.json({
                    data: withKeysAndIntegrationsData,
                });
            }),
        ],
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
            http.get(`${API_URL}/v2/profile`, () => {
                return HttpResponse.json({
                    data: noKeysOrIntegrationData,
                });
            }),
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
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/profile`, () => {
                return HttpResponse.json({
                    data: withKeysAndIntegrationsData,
                });
            }),
        ],
    },
};

export function SignedInPremiumNoCredits() {
    return (
        <SettingsApiView display="Page" />
    );
}
SignedInPremiumNoCredits.parameters = {
    session: signedInPremiumNoCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/profile`, () => {
                return HttpResponse.json({
                    data: withKeysAndIntegrationsData,
                });
            }),
        ],
    },
};

export function SignedInPremiumWithCredits() {
    return (
        <SettingsApiView display="Page" />
    );
}
SignedInPremiumWithCredits.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/profile`, () => {
                return HttpResponse.json({
                    data: withKeysAndIntegrationsData,
                });
            }),
        ],
    },
};

// New comprehensive stories

export function WithNewlyCreatedKey() {
    const newKeyData = {
        ...withKeysAndIntegrationsData,
        _newApiKey: "sk-test-1234567890abcdef", // Simulates newly created key for one-time display
    };
    
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
            http.get(`${API_URL}/v2/profile`, () => {
                return HttpResponse.json({
                    data: newKeyData,
                });
            }),
        ],
    },
};

export function HighCreditUsage() {
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
            http.get(`${API_URL}/v2/profile`, () => {
                return HttpResponse.json({
                    data: highUsageData,
                });
            }),
        ],
    },
};

export function ExternalKeysOnly() {
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
            http.get(`${API_URL}/v2/profile`, () => {
                return HttpResponse.json({
                    data: externalOnlyData,
                });
            }),
        ],
    },
};

export function AdminUser() {
    const adminSession: Partial<Session> = {
        ...session,
        users: [{
            ...session.users![0],
            isAdmin: true,
        }] as SessionUser[],
    };

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
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/profile`, () => {
                return HttpResponse.json({
                    data: withKeysAndIntegrationsData,
                });
            }),
        ],
    },
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
            http.get(`${API_URL}/v2/profile`, () => {
                // Simulate slow loading
                return new Promise(resolve => {
                    setTimeout(() => {
                        resolve(HttpResponse.json({
                            data: withKeysAndIntegrationsData,
                        }));
                    }, 2000);
                });
            }),
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
            http.get(`${API_URL}/v2/profile`, () => {
                return HttpResponse.json(
                    { error: "Failed to fetch API keys" },
                    { status: 500 }
                );
            }),
        ],
    },
};

export function CustomPermissions() {
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
            http.get(`${API_URL}/v2/profile`, () => {
                return HttpResponse.json({
                    data: customPermissionsData,
                });
            }),
        ],
    },
};

export function MixedKeyStates() {
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
            http.get(`${API_URL}/v2/profile`, () => {
                return HttpResponse.json({
                    data: mixedStatesData,
                });
            }),
        ],
    },
};
