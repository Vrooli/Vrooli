/* eslint-disable no-magic-numbers */
import { CodeLanguage, CodeType, CodeVersion, Resource, ResourceUsedFor, Tag, User, endpointsCodeVersion, getObjectUrl, uuid } from "@local/shared";
import { Meta } from "@storybook/react";
import { HttpResponse, http } from "msw";
import { API_URL, loggedOutSession, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { DataConverterView } from "./DataConverterView.js";

const mockTestCases = [
    {
        description: "Basic string conversion test",
        input: { text: "1,2,3,4,5" },
        expectedOutput: [1, 2, 3, 4, 5],
    },
    {
        description: "Handle empty input",
        input: { text: "" },
        expectedOutput: [],
    },
    {
        description: "Handle whitespace",
        input: { text: " 1, 2,  3,4 , 5 " },
        expectedOutput: [1, 2, 3, 4, 5],
    },
];

// Serialize the test cases to JSON for the data field
const serializedData = JSON.stringify({
    __version: "1.0.0",
    inputConfig: {
        inputSchema: {
            type: "object",
            properties: {
                text: { type: "string" },
            },
        },
        shouldSpread: false,
    },
    outputConfig: {
        type: "array",
        items: { type: "number" },
    },
    testCases: mockTestCases,
});

// Create simplified mock data for DataConverter responses
const mockDataConverterVersionData: CodeVersion = {
    __typename: "CodeVersion" as const,
    id: uuid(),
    calledByRoutineVersionsCount: 3,
    codeLanguage: CodeLanguage.Javascript,
    codeType: CodeType.DataConvert,
    comments: [],
    commentsCount: 0,
    content: `
function convertStringToNumberArray(input) {
    if (!input.text || input.text.trim() === '') {
        return [];
    }
    
    return input.text
        .split(',')
        .map(item => item.trim())
        .filter(item => item.length > 0)
        .map(item => parseInt(item, 10));
}

// This is the main function that will be called by the platform
function main(input) {
    return convertStringToNumberArray(input);
}
    `,
    created_at: new Date().toISOString(),
    data: serializedData,
    default: null,
    directoryListings: [],
    directoryListingsCount: 0,
    forks: [],
    forksCount: 0,
    isComplete: true,
    isDeleted: false,
    isLatest: true,
    isPrivate: false,
    pullRequest: null,
    reports: [],
    reportsCount: 0,
    resourceList: {
        __typename: "ResourceList" as const,
        id: uuid(),
        listFor: {
            __typename: "CodeVersion" as const,
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
        __typename: "Code" as const,
        id: uuid(),
        bookmarks: 5,
        likes: 10,
        views: 25,
        isPrivate: false,
        owner: { __typename: "User" as const, id: uuid() } as User,
        tags: [
            {
                __typename: "Tag" as const,
                id: "tag1",
                tag: "JavaScript",
            },
            {
                __typename: "Tag" as const,
                id: "tag2",
                tag: "Converter",
            },
            {
                __typename: "Tag" as const,
                id: "tag3",
                tag: "String",
            },
        ] as Tag[],
        you: {
            __typename: "CodeYou" as const,
            isBookmarked: false,
            isViewed: true,
        },
    },
    translations: [{
        __typename: "CodeVersionTranslation" as const,
        id: uuid(),
        language: "en",
        description: `# String to Number Array Converter

This data converter takes a comma-separated string and converts it to an array of numbers.

## Usage
Input should be a string where numbers are separated by commas. The converter will:
1. Split the string by commas
2. Trim whitespace from each item
3. Convert each item to a number
4. Return the resulting array

## Examples
- "1,2,3" → [1,2,3]
- "10, 20, 30" → [10,20,30]
- "" → []

## Implementation Notes
- Empty strings return an empty array
- Whitespace is handled automatically
- Non-numeric values may cause unexpected results`,
        name: "String to Number Array Converter",
        jsonVariable: null,
    }],
    translationsCount: 1,
    updated_at: new Date().toISOString(),
    versionIndex: 1,
    versionLabel: "1.0.0",
    versionNotes: "Initial version",
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

const meta: Meta<typeof DataConverterView> = {
    title: "Views/Objects/DataConverter/DataConverterView",
    component: DataConverterView,
    parameters: {
        layout: "fullscreen",
    },
};

export default {
    title: "Views/Objects/DataConverter/DataConverterView",
    component: DataConverterView,
};

export function NoResult() {
    return (
        <DataConverterView display="page" />
    );
}
NoResult.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

export function Loading() {
    return (
        <DataConverterView display="page" />
    );
}
Loading.parameters = {
    session: signedInNoPremiumNoCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsCodeVersion.findOne.endpoint}`, async () => {
                // Delay the response to simulate loading
                await new Promise(resolve => setTimeout(resolve, 120_000));
                return HttpResponse.json({ data: mockDataConverterVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockDataConverterVersionData)}`,
    },
};

export function SignInWithResults() {
    return (
        <DataConverterView display="page" />
    );
}
SignInWithResults.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsCodeVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockDataConverterVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockDataConverterVersionData)}`,
    },
};

export function LoggedOutWithResults() {
    return (
        <DataConverterView display="page" />
    );
}
LoggedOutWithResults.parameters = {
    session: loggedOutSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsCodeVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockDataConverterVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockDataConverterVersionData)}`,
    },
};
console.log('yeet mocked url', `${API_URL}/v2/rest${getObjectUrl(mockDataConverterVersionData)}`);

export function Own() {
    return (
        <DataConverterView display="page" />
    );
}
Own.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsCodeVersion.findOne.endpoint}`, () => {
                // Create a modified version of the mock data with owner permissions
                const mockWithOwnerPermissions = {
                    ...mockDataConverterVersionData,
                    root: {
                        ...mockDataConverterVersionData.root,
                        you: {
                            // Full permissions as the owner
                            canBookmark: true,
                            canDelete: true,
                            canUpdate: true,
                            canRead: true,
                            isBookmarked: false,
                            isOwner: true,
                            isStarred: false,
                        }
                    },
                    permissions: {
                        canDelete: true,
                        canUpdate: true,
                        canRead: true,
                    }
                };

                return HttpResponse.json({ data: mockWithOwnerPermissions });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockDataConverterVersionData)}`,
    },
};
