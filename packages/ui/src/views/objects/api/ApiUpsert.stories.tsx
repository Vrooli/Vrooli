/* eslint-disable @typescript-eslint/no-empty-function */
/* eslint-disable react-perf/jsx-no-new-function-as-prop */
/* eslint-disable no-magic-numbers */
import { ApiVersion, CodeLanguage, DUMMY_ID, Resource, ResourceUsedFor, Tag, User, endpointsApiVersion, getObjectUrl, uuid } from "@local/shared";
import { HttpResponse, http } from "msw";
import { API_URL, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { ViewDisplayType } from "../../../types.js";
import { ApiUpsert } from "./ApiUpsert.js";

// Create simplified mock data for API responses
const mockApiVersionData: ApiVersion = {
    __typename: "ApiVersion" as const,
    id: uuid(),
    calledByRoutineVersionsCount: Math.floor(Math.random() * 100),
    callLink: "https://api.example.com/v1",
    documentationLink: "https://docs.example.com/v1",
    directoryListings: [],
    isComplete: true,
    isPrivate: false,
    resourceList: {
        __typename: "ResourceList" as const,
        id: uuid(),
        listFor: {
            __typename: "ApiVersion" as const,
            id: uuid(),
        } as any,
        created_at: new Date().toISOString(),
        resources: Array.from({ length: Math.floor(Math.random() * 5) + 3 }, () => ({
            __typename: "Resource" as const,
            id: uuid(),
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            usedFor: ResourceUsedFor.Context,
            link: `https://example.com/resource/${Math.floor(Math.random() * 1000)}`,
            list: {} as any, // This will be set by the circular reference below
            translations: [{
                __typename: "ResourceTranslation" as const,
                id: uuid(),
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                language: "en",
                name: `Resource ${Math.floor(Math.random() * 1000)}`,
                description: `Description for Resource ${Math.floor(Math.random() * 1000)}`,
            }],
        })) as unknown as Resource[], // Use unknown to bypass type checking until runtime
        translations: [],
        updated_at: new Date().toISOString(),
    },
    schemaLanguage: CodeLanguage.Yaml,
    schemaText: `openapi: 3.0.0
info:
  title: Mock API
  version: ${Math.floor(Math.random() * 1000)}
paths:
  /comments:
    get:
      summary: List all comments
      description: Returns a list of comments
`,
    versionLabel: `${Math.floor(Math.random() * 10)}.${Math.floor(Math.random() * 10)}.${Math.floor(Math.random() * 10)}`,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    root: {
        __typename: "Api" as const,
        id: uuid(),
        isPrivate: false,
        owner: { __typename: "User" as const, id: uuid() } as User,
        tags: Array.from({ length: Math.floor(Math.random() * 10) }, () => ({
            __typename: "Tag" as const,
            id: uuid(),
            tag: ["Automation", "AI Agents", "Software Development", "API", "Cloud Computing", "Integration"][Math.floor(Math.random() * 6)],
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
        })) as Tag[],
        versions: [],
        views: Math.floor(Math.random() * 100_000),
    } as any,
    translations: [{
        __typename: "ApiVersionTranslation" as const,
        id: DUMMY_ID,
        language: "en",
        details: "This is a **detailed** description for the mock API using markdown.\nLorem ipsum dolor sit amet, consectetur adipiscing elit.",
        name: `Mock API v${Math.floor(Math.random() * 1000)}`,
        summary: "A simple mock API for demonstration purposes.",
    }],
};

export default {
    title: "Views/Objects/Api/ApiUpsert",
    component: ApiUpsert,
};

// Create a new API
export function Create({ display }: { display: ViewDisplayType }) {
    return (
        <ApiUpsert display={display} isCreate={true} />
    );
}
Create.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Update an existing API
export function Update({ display }: { display: ViewDisplayType }) {
    return (
        <ApiUpsert display={display} isCreate={false} />
    );
}
Update.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsApiVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockApiVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockApiVersionData)}/edit`,
    },
};

// Loading state
export function Loading({ display }: { display: ViewDisplayType }) {
    return (
        <ApiUpsert display={display} isCreate={false} />
    );
}
Loading.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsApiVersion.findOne.endpoint}`, async () => {
                // Delay the response to simulate loading
                await new Promise(resolve => setTimeout(resolve, 120_000));
                return HttpResponse.json({ data: mockApiVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockApiVersionData)}/edit`,
    },
};

// Non-premium user
export function NonPremiumUser({ display }: { display: ViewDisplayType }) {
    return (
        <ApiUpsert display={display} isCreate={true} />
    );
}
NonPremiumUser.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

// With Override Object (using dialog display)
export function WithOverrideObject({ display }: { display: ViewDisplayType }) {
    return (
        <ApiUpsert
            display={display}
            isCreate={true}
            isOpen={true}
            onClose={() => { }}
            onCancel={() => { }}
            onCompleted={() => { }}
            onDeleted={() => { }}
            overrideObject={mockApiVersionData}
        />
    );
}
WithOverrideObject.parameters = {
    session: signedInPremiumWithCreditsSession,
};
