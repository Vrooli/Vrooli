/* eslint-disable @typescript-eslint/no-empty-function */
/* eslint-disable react-perf/jsx-no-new-function-as-prop */
/* eslint-disable no-magic-numbers */
import { CodeLanguage, DUMMY_ID, ResourceUsedFor, endpointsResource, generatePK, getObjectUrl, type ApiVersion, type Resource, type Tag, type User } from "@vrooli/shared";
import { HttpResponse, http } from "msw";
import { API_URL, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { getMockUrl, getMockEndpoint, getStoryRouteEditPath } from "../../../__test/helpers/storybookMocking.js";
import { type ViewDisplayType } from "../../../types.js";
import { ApiUpsert } from "./ApiUpsert.js";

// Create simplified mock data for API responses
const mockApiVersionData: ApiVersion = {
    __typename: "ApiVersion" as const,
    id: generatePK().toString(),
    calledByRoutineVersionsCount: Math.floor(Math.random() * 100),
    callLink: "https://api.example.com/v1",
    documentationLink: "https://docs.example.com/v1",
    isComplete: true,
    isPrivate: false,
    resourceList: {
        __typename: "ResourceList" as const,
        id: generatePK().toString(),
        listFor: {
            __typename: "ApiVersion" as const,
            id: generatePK().toString(),
        } as any,
        createdAt: new Date().toISOString(),
        resources: Array.from({ length: Math.floor(Math.random() * 5) + 3 }, () => ({
            __typename: "Resource" as const,
            id: generatePK().toString(),
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            usedFor: ResourceUsedFor.Context,
            link: `https://example.com/resource/${Math.floor(Math.random() * 1000)}`,
            list: {} as any, // This will be set by the circular reference below
            translations: [{
                __typename: "ResourceTranslation" as const,
                id: generatePK().toString(),
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString(),
                language: "en",
                name: `Resource ${Math.floor(Math.random() * 1000)}`,
                description: `Description for Resource ${Math.floor(Math.random() * 1000)}`,
            }],
        })) as unknown as Resource[], // Use unknown to bypass type checking until runtime
        translations: [],
        updatedAt: new Date().toISOString(),
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
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    root: {
        __typename: "Api" as const,
        id: generatePK().toString(),
        isPrivate: false,
        owner: { __typename: "User" as const, id: generatePK().toString() } as User,
        tags: Array.from({ length: Math.floor(Math.random() * 10) }, () => ({
            __typename: "Tag" as const,
            id: generatePK().toString(),
            tag: ["Automation", "AI Agents", "Software Development", "API", "Cloud Computing", "Integration"][Math.floor(Math.random() * 6)],
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
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
            http.get(getMockUrl(endpointsResource.findApiVersion), () => {
                return HttpResponse.json({ data: mockApiVersionData });
            }),
        ],
    },
    route: {
        path: getStoryRouteEditPath(mockApiVersionData),
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
            http.get(getMockUrl(endpointsResource.findApiVersion), async () => {
                // Delay the response to simulate loading
                await new Promise(resolve => setTimeout(resolve, 120_000));
                return HttpResponse.json({ data: mockApiVersionData });
            }),
        ],
    },
    route: {
        path: getStoryRouteEditPath(mockApiVersionData),
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
