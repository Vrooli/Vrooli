/* eslint-disable @typescript-eslint/no-empty-function */
/* eslint-disable react-perf/jsx-no-new-function-as-prop */
/* eslint-disable no-magic-numbers */
import { CodeLanguage, DUMMY_ID, ResourceUsedFor, ResourceSubType, endpointsResource, generatePK, getObjectUrl, type Resource, type StandardVersion, type Tag, type User } from "@vrooli/shared";
import { HttpResponse, http } from "msw";
import { API_URL, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { getMockEndpoint, getStoryRouteEditPath } from "../../../__test/helpers/storybookMocking.js";
import { DataStructureUpsert } from "./DataStructureUpsert.js";

// Create simplified mock data for DataStructure responses
const mockDataStructureVersionData: StandardVersion = {
    __typename: "StandardVersion" as const,
    id: generatePK().toString(),
    comments: [],
    commentsCount: 0,
    codeLanguage: CodeLanguage.Json,
    createdAt: new Date().toISOString(),
    default: null,
    forks: [],
    forksCount: 0,
    isComplete: true,
    isDeleted: false,
    isFile: false,
    isLatest: true,
    isPrivate: false,
    props: `{
  "type": "object",
  "properties": {
    "id": {
      "type": "string",
      "format": "snowflake"
    },
    "name": {
      "type": "string"
    },
    "description": {
      "type": "string"
    }
  },
  "required": ["id", "name"]
}`,
    pullRequest: null,
    reports: [],
    reportsCount: 0,
    resourceList: {
        __typename: "ResourceList" as const,
        id: generatePK().toString(),
        listFor: {
            __typename: "StandardVersion" as const,
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
        })) as Resource[],
        translations: [],
        updatedAt: new Date().toISOString(),
    },
    root: {
        __typename: "Standard" as const,
        id: generatePK().toString(),
        isPrivate: false,
        isInternal: false,
        hasCompleteVersion: true,
        owner: { __typename: "User" as const, id: generatePK().toString() } as User,
        parent: null,
        permissions: "{}",
        tags: Array.from({ length: Math.floor(Math.random() * 10) }, () => ({
            __typename: "Tag" as const,
            id: generatePK().toString(),
            tag: ["Data Schema", "JSON Schema", "Database", "API", "Validation", "Data Modeling"][Math.floor(Math.random() * 6)],
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
        })) as Tag[],
        bookmarkedBy: [],
        bookmarks: 0,
        createdBy: null,
        forks: [],
        forksCount: 0,
        issues: [],
        issuesCount: 0,
        pullRequests: [],
        pullRequestsCount: 0,
        score: 0,
        stats: [],
        transfers: [],
        transfersCount: 0,
        translatedName: "Sample Data Structure",
        versions: [],
        versionsCount: 1,
        views: Math.floor(Math.random() * 100_000),
        you: {
            __typename: "StandardYou",
            canBookmark: true,
            canComment: true,
            canCopy: true,
            canDelete: true,
            canReact: true,
            canRead: true,
            canTransfer: true,
            canUpdate: true,
            isBookmarked: false,
        },
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        isDeleted: false,
    } as any,
    translations: [{
        __typename: "StandardVersionTranslation" as const,
        id: DUMMY_ID,
        language: "en",
        description: "This is a **detailed** description for the mock data structure using markdown.\nLorem ipsum dolor sit amet, consectetur adipiscing elit.",
        name: `Simple Data Schema v${Math.floor(Math.random() * 100)}`,
        jsonVariable: null,
    }],
    translationsCount: 1,
    updatedAt: new Date().toISOString(),
    variant: ResourceSubType.StandardDataStructure,
    versionIndex: 1,
    versionLabel: `${Math.floor(Math.random() * 10)}.${Math.floor(Math.random() * 10)}.${Math.floor(Math.random() * 10)}`,
    versionNotes: "Initial version",
    yup: null,
    you: {
        __typename: "VersionYou",
        canComment: true,
        canCopy: true,
        canDelete: true,
        canRead: true,
        canReport: true,
        canUpdate: true,
        canUse: true,
    },
};

export default {
    title: "Views/Objects/DataStructure/DataStructureUpsert",
    component: DataStructureUpsert,
};

// Create a new DataStructure
export function Create() {
    return (
        <DataStructureUpsert display="Page" isCreate={true} />
    );
}
Create.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Create a new DataStructure in a dialog
export function CreateDialog() {
    return (
        <DataStructureUpsert
            display="Dialog"
            isCreate={true}
            isOpen={true}
            onClose={() => { }}
            onCancel={() => { }}
            onCompleted={() => { }}
            onDeleted={() => { }}
        />
    );
}
CreateDialog.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Update an existing DataStructure
export function Update() {
    return (
        <DataStructureUpsert display="Page" isCreate={false} />
    );
}
Update.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(getMockEndpoint(endpointsResource.findDataStructureVersion), () => {
                return HttpResponse.json({ data: mockDataStructureVersionData });
            }),
        ],
    },
    route: {
        path: getStoryRouteEditPath(mockDataStructureVersionData),
    },
};

// Update an existing DataStructure in a dialog
export function UpdateDialog() {
    return (
        <DataStructureUpsert
            display="Dialog"
            isCreate={false}
            isOpen={true}
            onClose={() => { }}
            onCancel={() => { }}
            onCompleted={() => { }}
            onDeleted={() => { }}
        />
    );
}
UpdateDialog.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(getMockEndpoint(endpointsResource.findDataStructureVersion), () => {
                return HttpResponse.json({ data: mockDataStructureVersionData });
            }),
        ],
    },
    route: {
        path: getStoryRouteEditPath(mockDataStructureVersionData),
    },
};

// Loading state
export function Loading() {
    return (
        <DataStructureUpsert display="Page" isCreate={false} />
    );
}
Loading.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(getMockEndpoint(endpointsResource.findDataStructureVersion), async () => {
                // Delay the response to simulate loading
                await new Promise(resolve => setTimeout(resolve, 120000));
                return HttpResponse.json({ data: mockDataStructureVersionData });
            }),
        ],
    },
    route: {
        path: getStoryRouteEditPath(mockDataStructureVersionData),
    },
};

// Non-premium user
export function NonPremiumUser() {
    return (
        <DataStructureUpsert display="Page" isCreate={true} />
    );
}
NonPremiumUser.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

// With Override Object (using dialog display)
export function WithOverrideObject() {
    return (
        <DataStructureUpsert
            display="Dialog"
            isCreate={true}
            isOpen={true}
            onClose={() => { }}
            onCancel={() => { }}
            onCompleted={() => { }}
            onDeleted={() => { }}
            overrideObject={mockDataStructureVersionData}
        />
    );
}
WithOverrideObject.parameters = {
    session: signedInPremiumWithCreditsSession,
}; 
