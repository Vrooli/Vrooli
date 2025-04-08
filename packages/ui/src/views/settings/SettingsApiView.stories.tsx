/* eslint-disable no-magic-numbers */
import { ApiKey, ApiKeyPermission, Session, SessionUser, User, uuid } from "@local/shared";
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
        id: uuid(),
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
            id: uuid(),
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
            id: uuid(),
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
            id: uuid(),
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
            id: uuid(),
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
            id: uuid(),
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
            id: uuid(),
            disabledAt: null,
            name: "External Key 1",
            service: "OpenAI",
        },
        {
            __typename: "ApiKeyExternal" as const,
            id: uuid(),
            disabledAt: new Date().toISOString(),
            name: "External Key 2",
            service: "OpenAI",
        },
        {
            __typename: "ApiKeyExternal" as const,
            id: uuid(),
            disabledAt: null,
            name: "External Key 3",
            service: "Microsoft",
        },
    ],
};

export function NoKeysOrIntegrations() {
    return (
        <SettingsApiView display="page" />
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
            http.get(`${API_URL}/v2/rest/profile`, () => {
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
        <SettingsApiView display="page" />
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
            http.get(`${API_URL}/v2/rest/profile`, () => {
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
        <SettingsApiView display="page" />
    );
}
SignedInNoPremiumNoCredits.parameters = {
    session: signedInNoPremiumNoCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest/profile`, () => {
                return HttpResponse.json({
                    data: noKeysOrIntegrationData,
                });
            }),
        ],
    },
};

export function SignedInNoPremiumWithCredits() {
    return (
        <SettingsApiView display="page" />
    );
}
SignedInNoPremiumWithCredits.parameters = {
    session: signedInNoPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest/profile`, () => {
                return HttpResponse.json({
                    data: withKeysAndIntegrationsData,
                });
            }),
        ],
    },
};

export function SignedInPremiumNoCredits() {
    return (
        <SettingsApiView display="page" />
    );
}
SignedInPremiumNoCredits.parameters = {
    session: signedInPremiumNoCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest/profile`, () => {
                return HttpResponse.json({
                    data: withKeysAndIntegrationsData,
                });
            }),
        ],
    },
};

export function SignedInPremiumWithCredits() {
    return (
        <SettingsApiView display="page" />
    );
}
SignedInPremiumWithCredits.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest/profile`, () => {
                return HttpResponse.json({
                    data: withKeysAndIntegrationsData,
                });
            }),
        ],
    },
};
