/* eslint-disable @typescript-eslint/no-empty-function */
/* eslint-disable react-perf/jsx-no-new-function-as-prop */
/* eslint-disable no-magic-numbers */
import { CodeLanguage, CodeType, DUMMY_ID, ResourceUsedFor, endpointsCodeVersion, generatePK, getObjectUrl, type CodeVersion, type Resource, type Tag, type User } from "@vrooli/shared";
import { HttpResponse, http } from "msw";
import { API_URL, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { DataConverterUpsert } from "./DataConverterUpsert.js";

// Create simplified mock data for DataConverter responses
const mockDataConverterVersionData: CodeVersion = {
    __typename: "CodeVersion" as const,
    id: generatePK().toString(),
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
    createdAt: new Date().toISOString(),
    data: null,
    default: null,
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
        id: generatePK().toString(),
        listFor: {
            __typename: "CodeVersion" as const,
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
        __typename: "Code" as const,
        id: generatePK().toString(),
        isPrivate: false,
        owner: { __typename: "User" as const, id: generatePK().toString() } as User,
        tags: Array.from({ length: Math.floor(Math.random() * 10) }, () => ({
            __typename: "Tag" as const,
            id: generatePK().toString(),
            tag: ["Data Processing", "ETL", "Transformation", "CSV", "JSON", "XML", "Integration"][Math.floor(Math.random() * 7)],
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
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
    updatedAt: new Date().toISOString(),
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
        <DataConverterUpsert display="Page" isCreate={true} />
    );
}
Create.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Create a new DataConverter in a dialog
export function CreateDialog() {
    return (
        <DataConverterUpsert
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

// Update an existing DataConverter
export function Update() {
    return (
        <DataConverterUpsert display="Page" isCreate={false} />
    );
}
Update.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2${endpointsCodeVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockDataConverterVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2${getObjectUrl(mockDataConverterVersionData)}/edit`,
    },
};

// Update an existing DataConverter in a dialog
export function UpdateDialog() {
    return (
        <DataConverterUpsert
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
            http.get(`${API_URL}/v2${endpointsCodeVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockDataConverterVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2${getObjectUrl(mockDataConverterVersionData)}/edit`,
    },
};

// Loading state
export function Loading() {
    return (
        <DataConverterUpsert display="Page" isCreate={false} />
    );
}
Loading.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2${endpointsCodeVersion.findOne.endpoint}`, async () => {
                // Delay the response to simulate loading
                await new Promise(resolve => setTimeout(resolve, 120000));
                return HttpResponse.json({ data: mockDataConverterVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2${getObjectUrl(mockDataConverterVersionData)}/edit`,
    },
};

// Non-premium user
export function NonPremiumUser() {
    return (
        <DataConverterUpsert display="Page" isCreate={true} />
    );
}
NonPremiumUser.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

// With Override Object (using dialog display)
export function WithOverrideObject() {
    return (
        <DataConverterUpsert
            display="Dialog"
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
