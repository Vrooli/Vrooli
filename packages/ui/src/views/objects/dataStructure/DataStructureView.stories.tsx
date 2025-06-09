/* eslint-disable no-magic-numbers */
import { CodeLanguage, DUMMY_ID, ResourceUsedFor, ResourceSubType, endpointsResource, generatePK, getObjectUrl, type Resource, type StandardVersion, type Tag, type User } from "@vrooli/shared";
import { HttpResponse, http } from "msw";
import { API_URL, loggedOutSession, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { DataStructureView } from "./DataStructureView.js";

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
    },
    "createdAt": {
      "type": "string",
      "format": "date-time"
    },
    "updatedAt": {
      "type": "string",
      "format": "date-time"
    },
    "tags": {
      "type": "array",
      "items": {
        "type": "string"
      }
    },
    "status": {
      "type": "string",
      "enum": ["active", "archived", "draft"]
    }
  },
  "required": ["id", "name", "createdAt", "updatedAt"]
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
            tag: ["Data Schema", "JSON Schema", "Database", "API", "Validation", "Data Modeling", "Other"][Math.floor(Math.random() * 7)],
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
        description: "This is a **detailed** description for the mock data structure using markdown.\nLorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.",
        name: `User Data Schema v${Math.floor(Math.random() * 100)}`,
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
    title: "Views/Objects/DataStructure/DataStructureView",
    component: DataStructureView,
};

export function NoResult() {
    return (
        <DataStructureView display="Page" />
    );
}
NoResult.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

export function Loading() {
    return (
        <DataStructureView display="Page" />
    );
}
Loading.parameters = {
    session: signedInNoPremiumNoCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2${endpointsResource.findDataStructureVersion.findOne.endpoint}`, async () => {
                // Delay the response to simulate loading
                await new Promise(resolve => setTimeout(resolve, 120000));
                return HttpResponse.json({ data: mockDataStructureVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2${getObjectUrl(mockDataStructureVersionData)}`,
    },
};

export function SignInWithResults() {
    return (
        <DataStructureView display="Page" />
    );
}
SignInWithResults.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2${endpointsResource.findDataStructureVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockDataStructureVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2${getObjectUrl(mockDataStructureVersionData)}`,
    },
};

export function LoggedOutWithResults() {
    return (
        <DataStructureView display="Page" />
    );
}
LoggedOutWithResults.parameters = {
    session: loggedOutSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2${endpointsResource.findDataStructureVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockDataStructureVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2${getObjectUrl(mockDataStructureVersionData)}`,
    },
}; 
