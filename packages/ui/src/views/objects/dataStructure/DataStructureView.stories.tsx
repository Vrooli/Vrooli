/* eslint-disable no-magic-numbers */
import { CodeLanguage, DUMMY_ID, Resource, ResourceUsedFor, StandardType, StandardVersion, Tag, User, endpointsStandardVersion, getObjectUrl, uuid } from "@local/shared";
import { HttpResponse, http } from "msw";
import { API_URL, loggedOutSession, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { DataStructureView } from "./DataStructureView.js";

// Create simplified mock data for DataStructure responses
const mockDataStructureVersionData: StandardVersion = {
    __typename: "StandardVersion" as const,
    id: uuid(),
    comments: [],
    commentsCount: 0,
    codeLanguage: CodeLanguage.Json,
    created_at: new Date().toISOString(),
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
      "format": "uuid"
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
        id: uuid(),
        listFor: {
            __typename: "StandardVersion" as const,
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
        })) as Resource[],
        translations: [],
        updated_at: new Date().toISOString(),
    },
    root: {
        __typename: "Standard" as const,
        id: uuid(),
        isPrivate: false,
        isInternal: false,
        hasCompleteVersion: true,
        owner: { __typename: "User" as const, id: uuid() } as User,
        parent: null,
        permissions: "{}",
        tags: Array.from({ length: Math.floor(Math.random() * 10) }, () => ({
            __typename: "Tag" as const,
            id: uuid(),
            tag: ["Data Schema", "JSON Schema", "Database", "API", "Validation", "Data Modeling", "Other"][Math.floor(Math.random() * 7)],
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
        })) as Tag[],
        bookmarkedBy: [],
        bookmarks: 0,
        createdBy: null,
        forks: [],
        forksCount: 0,
        issues: [],
        issuesCount: 0,
        labels: [],
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
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
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
    updated_at: new Date().toISOString(),
    variant: StandardType.DataStructure,
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
        <DataStructureView display="page" />
    );
}
NoResult.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

export function Loading() {
    return (
        <DataStructureView display="page" />
    );
}
Loading.parameters = {
    session: signedInNoPremiumNoCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsStandardVersion.findOne.endpoint}`, async () => {
                // Delay the response to simulate loading
                await new Promise(resolve => setTimeout(resolve, 120000));
                return HttpResponse.json({ data: mockDataStructureVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockDataStructureVersionData)}`,
    },
};

export function SignInWithResults() {
    return (
        <DataStructureView display="page" />
    );
}
SignInWithResults.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsStandardVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockDataStructureVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockDataStructureVersionData)}`,
    },
};

export function LoggedOutWithResults() {
    return (
        <DataStructureView display="page" />
    );
}
LoggedOutWithResults.parameters = {
    session: loggedOutSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsStandardVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockDataStructureVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockDataStructureVersionData)}`,
    },
}; 
