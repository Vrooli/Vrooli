/* eslint-disable @typescript-eslint/no-empty-function */
/* eslint-disable react-perf/jsx-no-new-function-as-prop */
/* eslint-disable no-magic-numbers */
import { CodeLanguage, CodeType, CodeVersion, DUMMY_ID, Resource, ResourceUsedFor, Tag, User, endpointsCodeVersion, getObjectUrl, uuid } from "@local/shared";
import { HttpResponse, http } from "msw";
import { API_URL, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { DataConverterUpsert } from "./DataConverterUpsert.js";

// Create simplified mock data for DataConverter responses
const mockDataConverterVersionData: CodeVersion = {
    __typename: "CodeVersion" as const,
    id: uuid(),
    calledByRoutineVersionsCount: Math.floor(Math.random() * 100),
    codeLanguage: CodeLanguage.Javascript,
    codeType: CodeType.DataConvert,
    comments: [],
    commentsCount: 0,
    content: `/**
 * Converts a comma-separated string of numbers into an array of numbers.
 * @param {string} input - A string containing numbers separated by commas.
 * @return {Array<number>} - An array of numbers extracted from the given string.
 */
function stringToNumberArray(input) {
  if (!input || typeof input !== 'string') {
    return [];
  }
  return input.split(',').map(item => Number(item.trim())).filter(num => !isNaN(num));
}`,
    created_at: new Date().toISOString(),
    data: null,
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
        isPrivate: false,
        owner: { __typename: "User" as const, id: uuid() } as User,
        tags: Array.from({ length: Math.floor(Math.random() * 10) }, () => ({
            __typename: "Tag" as const,
            id: uuid(),
            tag: ["Data Processing", "ETL", "Transformation", "CSV", "JSON", "XML", "Integration"][Math.floor(Math.random() * 7)],
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
        })) as Tag[],
        versions: [],
        views: Math.floor(Math.random() * 100_000),
    } as any,
    translations: [{
        __typename: "CodeVersionTranslation" as const,
        id: DUMMY_ID,
        language: "en",
        description: "This is a **detailed** description for the mock data converter using markdown.\nLorem ipsum dolor sit amet, consectetur adipiscing elit.",
        name: `String to Number Array Converter v${Math.floor(Math.random() * 100)}`,
        jsonVariable: null,
    }],
    translationsCount: 1,
    updated_at: new Date().toISOString(),
    versionIndex: 1,
    versionLabel: `${Math.floor(Math.random() * 10)}.${Math.floor(Math.random() * 10)}.${Math.floor(Math.random() * 10)}`,
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

export default {
    title: "Views/Objects/DataConverter/DataConverterUpsert",
    component: DataConverterUpsert,
};

// Create a new DataConverter
export function Create() {
    return (
        <DataConverterUpsert display="page" isCreate={true} />
    );
}
Create.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Create a new DataConverter in a dialog
export function CreateDialog() {
    return (
        <DataConverterUpsert
            display="dialog"
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

// Update an existing DataConverter
export function Update() {
    return (
        <DataConverterUpsert display="page" isCreate={false} />
    );
}
Update.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsCodeVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockDataConverterVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockDataConverterVersionData)}/edit`,
    },
};

// Update an existing DataConverter in a dialog
export function UpdateDialog() {
    return (
        <DataConverterUpsert
            display="dialog"
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
            http.get(`${API_URL}/v2/rest${endpointsCodeVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockDataConverterVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockDataConverterVersionData)}/edit`,
    },
};

// Loading state
export function Loading() {
    return (
        <DataConverterUpsert display="page" isCreate={false} />
    );
}
Loading.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsCodeVersion.findOne.endpoint}`, async () => {
                // Delay the response to simulate loading
                await new Promise(resolve => setTimeout(resolve, 120000));
                return HttpResponse.json({ data: mockDataConverterVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockDataConverterVersionData)}/edit`,
    },
};

// Non-premium user
export function NonPremiumUser() {
    return (
        <DataConverterUpsert display="page" isCreate={true} />
    );
}
NonPremiumUser.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

// With Override Object (using dialog display)
export function WithOverrideObject() {
    return (
        <DataConverterUpsert
            display="dialog"
            isCreate={true}
            isOpen={true}
            onClose={() => { }}
            onCancel={() => { }}
            onCompleted={() => { }}
            onDeleted={() => { }}
            overrideObject={mockDataConverterVersionData}
        />
    );
}
WithOverrideObject.parameters = {
    session: signedInPremiumWithCreditsSession,
}; 
