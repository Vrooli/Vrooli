/* eslint-disable no-magic-numbers */
import { Session, SessionUser, User, uuid } from "@local/shared";
import { HttpResponse, http } from "msw";
import { API_URL, loggedOutSession, signedInNoPremiumNoCreditsSession, signedInNoPremiumWithCreditsSession, signedInPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../__test/storybookConsts.js";
import { PageContainer } from "../../components/Page/Page.js";
import { SettingsApiView } from "./SettingsApiView.js";

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
        // Fresh key
        {
            __typename: "ApiKey" as const,
            id: uuid(),
            creditsUsed: BigInt(0).toString(),
            disabledAt: null,
            limitHard: BigInt(25000000000).toString(),
            limitSoft: null,
            name: "Key 1",
            stopAtLimit: true,
        },
        // Used but still active key
        {
            __typename: "ApiKey" as const,
            id: uuid(),
            creditsUsed: BigInt(1000000).toString(),
            disabledAt: null,
            limitHard: BigInt(25000000000).toString(),
            limitSoft: null,
            name: "Key 2",
            stopAtLimit: true,
        },
        // Inactive key
        {
            __typename: "ApiKey" as const,
            id: uuid(),
            creditsUsed: BigInt(1000000).toString(),
            disabledAt: new Date().toISOString(),
            limitHard: BigInt(25000000000).toString(),
            limitSoft: null,
            name: "Key 3",
            stopAtLimit: true,
        },
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
        <PageContainer>
            <SettingsApiView display="page" />
        </PageContainer>
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
        <PageContainer>
            <SettingsApiView display="page" />
        </PageContainer>
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

export function LoggedOut() {
    return (
        <PageContainer>
            <SettingsApiView display="page" />
        </PageContainer>
    );
}
LoggedOut.parameters = {
    session: loggedOutSession,
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

export function SignedInNoPremiumNoCredits() {
    return (
        <PageContainer>
            <SettingsApiView display="page" />
        </PageContainer>
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
        <PageContainer>
            <SettingsApiView display="page" />
        </PageContainer>
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
        <PageContainer>
            <SettingsApiView display="page" />
        </PageContainer>
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
        <PageContainer>
            <SettingsApiView display="page" />
        </PageContainer>
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
