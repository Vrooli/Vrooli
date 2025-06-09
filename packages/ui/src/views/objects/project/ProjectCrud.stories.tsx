/* eslint-disable @typescript-eslint/no-empty-function */
/* eslint-disable react-perf/jsx-no-new-function-as-prop */
/* eslint-disable no-magic-numbers */
import { DUMMY_ID, ResourceUsedFor, endpointsResource, generatePK, getObjectUrl, type ProjectVersion, type ProjectVersionDirectory, type Resource, type Tag, type User } from "@vrooli/shared";
import { HttpResponse, http } from "msw";
import { API_URL, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { ProjectCrud } from "./ProjectCrud.js";

// Create simplified mock data for Project responses
const mockProjectVersionData = {
    __typename: "ProjectVersion" as const,
    id: generatePK().toString(),
    comments: [],
    commentsCount: 0,
    createdAt: new Date().toISOString(),
    forks: [],
    forksCount: 0,
    isComplete: true,
    isLatest: true,
    isPrivate: false,
    pullRequest: null,
    reports: [],
    reportsCount: 0,
    resourceList: {
        __typename: "ResourceList" as const,
        id: generatePK().toString(),
        listFor: {
            __typename: "ProjectVersion" as const,
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
            list: {} as any,
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
        __typename: "Project" as const,
        id: generatePK().toString(),
        isPrivate: false,
        owner: { __typename: "User" as const, id: generatePK().toString() } as User,
        parent: null,
        tags: Array.from({ length: Math.floor(Math.random() * 10) }, () => ({
            __typename: "Tag" as const,
            id: generatePK().toString(),
            tag: ["Development", "Design", "Research", "Planning", "Documentation", "Testing"][Math.floor(Math.random() * 6)],
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
        })) as Tag[],
        versions: [{
            __typename: "ProjectVersion" as const,
            id: generatePK().toString(),
            versionLabel: "1.0.0",
        }],
    } as any,
    translations: [{
        __typename: "ProjectVersionTranslation" as const,
        id: DUMMY_ID,
        language: "en",
        description: "This is a **detailed** description for the mock project using markdown.\nLorem ipsum dolor sit amet, consectetur adipiscing elit.",
        name: `Sample Project v${Math.floor(Math.random() * 100)}`,
    }],
    translationsCount: 1,
    updatedAt: new Date().toISOString(),
    versionIndex: 1,
    versionLabel: "1.0.0",
    versionNotes: "Initial version",
    directories: [] as ProjectVersionDirectory[],
    you: {
        __typename: "ProjectVersionYou" as const,
        canComment: true,
        canCopy: true,
        canDelete: true,
        canRead: true,
        canReport: true,
        canUpdate: true,
        canUse: true,
        runs: [],
    },
} as unknown as ProjectVersion;

export default {
    title: "Views/Objects/Project/ProjectCrud",
    component: ProjectCrud,
};

// Create a new Project
export function Create() {
    return (
        <ProjectCrud display="Page" isCreate={true} />
    );
}
Create.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Create a new Project in a dialog
export function CreateDialog() {
    return (
        <ProjectCrud
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

// Update an existing Project
export function Update() {
    return (
        <ProjectCrud display="Page" isCreate={false} />
    );
}
Update.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2${endpointsResource.findProjectVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockProjectVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2${getObjectUrl(mockProjectVersionData)}/edit`,
    },
};

// Update an existing Project in a dialog
export function UpdateDialog() {
    return (
        <ProjectCrud
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
            http.get(`${API_URL}/v2${endpointsResource.findProjectVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockProjectVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2${getObjectUrl(mockProjectVersionData)}/edit`,
    },
};

// Loading state
export function Loading() {
    return (
        <ProjectCrud display="Page" isCreate={false} />
    );
}
Loading.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2${endpointsResource.findProjectVersion.findOne.endpoint}`, async () => {
                // Delay the response to simulate loading
                await new Promise(resolve => setTimeout(resolve, 120000));
                return HttpResponse.json({ data: mockProjectVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2${getObjectUrl(mockProjectVersionData)}/edit`,
    },
};

// Non-premium user
export function NonPremiumUser() {
    return (
        <ProjectCrud display="Page" isCreate={true} />
    );
}
NonPremiumUser.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

// With Override Object (using dialog display)
export function WithOverrideObject() {
    return (
        <ProjectCrud
            display="Dialog"
            isCreate={true}
            isOpen={true}
            onClose={() => { }}
            onCancel={() => { }}
            onCompleted={() => { }}
            onDeleted={() => { }}
            overrideObject={mockProjectVersionData}
        />
    );
}
WithOverrideObject.parameters = {
    session: signedInPremiumWithCreditsSession,
}; 
