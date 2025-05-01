/* eslint-disable @typescript-eslint/no-empty-function */
/* eslint-disable react-perf/jsx-no-new-function-as-prop */
/* eslint-disable no-magic-numbers */
import { DUMMY_ID, InputType, ResourceUsedFor, RoutineType, RoutineVersion, RoutineVersionYou, endpointsRoutineVersion, generatePKString, getObjectUrl } from "@local/shared";
import { HttpResponse, http } from "msw";
import { API_URL, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { RoutineMultiStepCrud } from "./RoutineMultiStepCrud.js";

// Create node IDs in advance to reference in the links
const nodeId1 = generatePKString();
const nodeId2 = generatePKString();
const nodeId3 = generatePKString();

// Create simplified mock data for Routine responses with nodes and nodeLinks
const mockRoutineVersionData: RoutineVersion = {
    __typename: "RoutineVersion" as const,
    id: generatePKString(),
    apiVersion: null,
    calledByRoutineVersionsCount: Math.floor(Math.random() * 100),
    callLink: "",
    codeVersion: null,
    comments: [],
    commentsCount: 0,
    complexity: 4,
    config: JSON.stringify({
        __version: "1.0.0",
        formInput: {
            __version: "1.0.0",
            schema: {
                elements: [
                    {
                        id: "userInput",
                        type: InputType.Text,
                        label: "User Input",
                        fieldName: "userInput",
                        description: "Enter your request",
                        props: {
                            defaultValue: "",
                            placeholder: "What would you like me to do?",
                            minRows: 2,
                            maxRows: 4,
                        },
                    },
                ],
                containers: [
                    {
                        title: "Workflow Input",
                        description: "Input for the workflow",
                        totalItems: 1,
                    },
                ],
            },
        },
        formOutput: {
            __version: "1.0.0",
            schema: {
                elements: [
                    {
                        id: "result",
                        type: InputType.Text,
                        label: "Result",
                        fieldName: "result",
                        description: "Workflow result",
                        props: {
                            defaultValue: "",
                            isMarkdown: true,
                            minRows: 4,
                            maxRows: 20,
                        },
                    },
                ],
                containers: [
                    {
                        title: "Workflow Output",
                        description: "Output from the workflow",
                        totalItems: 1,
                    },
                ],
            },
        },
    }),
    createdAt: new Date().toISOString(),
    forks: [],
    forksCount: 0,
    inputs: [],
    isAutomatable: true,
    isComplete: true,
    isDeleted: false,
    isLatest: true,
    isPrivate: false,
    nodes: [
        {
            __typename: "Node" as const,
            id: nodeId1,
            nodeType: "RoutineList",
            rowIndex: 0,
            columnIndex: 0,
            routineList: {
                __typename: "NodeRoutineList" as const,
                id: generatePKString(),
                isOrdered: true,
                isOptional: false,
                items: [
                    {
                        __typename: "NodeRoutineListItem" as const,
                        id: generatePKString(),
                        index: 0,
                        isOptional: false,
                        routineVersion: null,
                        translations: [
                            {
                                __typename: "NodeRoutineListItemTranslation" as const,
                                id: generatePKString(),
                                language: "en",
                                description: "Process user input",
                                name: "Process Input",
                            },
                        ],
                    },
                    {
                        __typename: "NodeRoutineListItem" as const,
                        id: generatePKString(),
                        index: 1,
                        isOptional: false,
                        routineVersion: null,
                        translations: [
                            {
                                __typename: "NodeRoutineListItemTranslation" as const,
                                id: generatePKString(),
                                language: "en",
                                description: "Generate response based on processed input",
                                name: "Generate Response",
                            },
                        ],
                    },
                ],
            },
            translations: [
                {
                    __typename: "NodeTranslation" as const,
                    id: generatePKString(),
                    language: "en",
                    name: "Process and Generate",
                    description: "First node in the workflow",
                },
            ],
        },
        {
            __typename: "Node" as const,
            id: nodeId2,
            nodeType: "RoutineList",
            rowIndex: 1,
            columnIndex: 0,
            routineList: {
                __typename: "NodeRoutineList" as const,
                id: generatePKString(),
                isOrdered: true,
                isOptional: false,
                items: [
                    {
                        __typename: "NodeRoutineListItem" as const,
                        id: generatePKString(),
                        index: 0,
                        isOptional: false,
                        routineVersion: null,
                        translations: [
                            {
                                __typename: "NodeRoutineListItemTranslation" as const,
                                id: generatePKString(),
                                language: "en",
                                description: "Format the generated response",
                                name: "Format Response",
                            },
                        ],
                    },
                ],
            },
            translations: [
                {
                    __typename: "NodeTranslation" as const,
                    id: generatePKString(),
                    language: "en",
                    name: "Format Output",
                    description: "Format the final output",
                },
            ],
        },
        {
            __typename: "Node" as const,
            id: nodeId3,
            nodeType: "End",
            rowIndex: 2,
            columnIndex: 0,
            end: {
                __typename: "NodeEnd" as const,
                id: generatePKString(),
                wasSuccessful: true,
            },
            translations: [
                {
                    __typename: "NodeTranslation" as const,
                    id: generatePKString(),
                    language: "en",
                    name: "End",
                    description: "Workflow completed successfully",
                },
            ],
        },
    ],
    nodeLinks: [
        {
            __typename: "NodeLink" as const,
            id: generatePKString(),
            from: {
                __typename: "Node" as const,
                id: nodeId1,
            },
            to: {
                __typename: "Node" as const,
                id: nodeId2,
            },
        },
        {
            __typename: "NodeLink" as const,
            id: generatePKString(),
            from: {
                __typename: "Node" as const,
                id: nodeId2,
            },
            to: {
                __typename: "Node" as const,
                id: nodeId3,
            },
        },
    ],
    outputs: [],
    pullRequest: null,
    reports: [],
    reportsCount: 0,
    resourceList: {
        __typename: "ResourceList" as const,
        id: generatePKString(),
        createdAt: new Date().toISOString(),
        listFor: {
            __typename: "RoutineVersion" as const,
            id: generatePKString(),
        },
        resources: Array.from({ length: Math.floor(Math.random() * 3) + 1 }, () => ({
            __typename: "Resource" as const,
            id: generatePKString(),
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            usedFor: ResourceUsedFor.Context,
            link: `https://example.com/resource/${Math.floor(Math.random() * 1000)}`,
            list: {} as any, // This will be set by the circular reference below
            translations: [{
                __typename: "ResourceTranslation" as const,
                id: generatePKString(),
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString(),
                language: "en",
                name: `Resource ${Math.floor(Math.random() * 1000)}`,
                description: `Description for Resource ${Math.floor(Math.random() * 1000)}`,
            }],
        })),
        translations: [],
        updatedAt: new Date().toISOString(),
    },
    root: {
        __typename: "Routine" as const,
        id: generatePKString(),
        isPrivate: false,
        owner: { __typename: "User" as const, id: generatePKString() },
        tags: Array.from({ length: Math.floor(Math.random() * 5) + 2 }, () => ({
            __typename: "Tag" as const,
            id: generatePKString(),
            tag: ["Workflow", "Multi-step", "Process", "Automation", "AI", "Business Logic", "ETL", "Data Processing"][Math.floor(Math.random() * 8)],
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
        })),
        versions: [],
        views: Math.floor(Math.random() * 10000),
    },
    routineType: RoutineType.Standard,
    simplicity: 3,
    subroutineLinks: [],
    timesCompleted: Math.floor(Math.random() * 500),
    timesStarted: Math.floor(Math.random() * 1000),
    translations: [{
        __typename: "RoutineVersionTranslation" as const,
        id: DUMMY_ID,
        language: "en",
        description: "A multi-step workflow that processes input, generates a response, and formats the output.",
        instructions: "Enter your request in the input field and execute the workflow to see the formatted result.",
        name: "Multi-Step Processing Workflow",
    }],
    updatedAt: new Date().toISOString(),
    versionIndex: 0,
    versionLabel: "1.0.0",
    versionNotes: "Initial version",
    you: {
        canBookmark: true,
        canDelete: false,
        canRead: true,
        canRun: true,
        canUpdate: false,
    } as RoutineVersionYou,
};

export default {
    title: "Views/Objects/Routine/RoutineMultiStepCrud",
    component: RoutineMultiStepCrud,
};

// Create a new Routine
export function Create() {
    return (
        <RoutineMultiStepCrud display="Page" isCreate={true} />
    );
}
Create.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Create a new Routine in a dialog
export function CreateDialog() {
    return (
        <RoutineMultiStepCrud
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

// Update an existing Routine
export function Update() {
    return (
        <RoutineMultiStepCrud display="Page" isCreate={false} />
    );
}
Update.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsRoutineVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockRoutineVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockRoutineVersionData)}/edit`,
    },
};

// Update an existing Routine in a dialog
export function UpdateDialog() {
    return (
        <RoutineMultiStepCrud
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
            http.get(`${API_URL}/v2/rest${endpointsRoutineVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockRoutineVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockRoutineVersionData)}/edit`,
    },
};

// Loading state
export function Loading() {
    return (
        <RoutineMultiStepCrud display="Page" isCreate={false} />
    );
}
Loading.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsRoutineVersion.findOne.endpoint}`, async () => {
                // Delay the response to simulate loading
                await new Promise(resolve => setTimeout(resolve, 120_000));
                return HttpResponse.json({ data: mockRoutineVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockRoutineVersionData)}/edit`,
    },
};

// Non-premium user
export function NonPremiumUser() {
    return (
        <RoutineMultiStepCrud display="Page" isCreate={true} />
    );
}
NonPremiumUser.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

// With Override Object
export function WithOverrideObject() {
    return (
        <RoutineMultiStepCrud
            display="Dialog"
            isCreate={true}
            isOpen={true}
            onClose={() => { }}
            onCancel={() => { }}
            onCompleted={() => { }}
            onDeleted={() => { }}
            overrideObject={mockRoutineVersionData}
        />
    );
}
WithOverrideObject.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// With Owner Permissions
export function Own() {
    return (
        <RoutineMultiStepCrud display="Page" isCreate={false} />
    );
}
Own.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsRoutineVersion.findOne.endpoint}`, () => {
                // Create a modified version of the mock data with owner permissions
                const mockWithOwnerPermissions = {
                    ...mockRoutineVersionData,
                    root: {
                        ...mockRoutineVersionData.root,
                        owner: {
                            ...mockRoutineVersionData.root.owner,
                            id: signedInPremiumWithCreditsSession.users?.[0]?.id,
                        },
                        you: {
                            // Full permissions as the owner
                            canBookmark: true,
                            canDelete: true,
                            canUpdate: true,
                            canRead: true,
                            isBookmarked: false,
                            isOwner: true,
                            isStarred: false,
                        },
                    },
                    you: {
                        canBookmark: true,
                        canDelete: true,
                        canRead: true,
                        canRun: true,
                        canUpdate: true,
                    } as RoutineVersionYou,
                };

                return HttpResponse.json({ data: mockWithOwnerPermissions });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockRoutineVersionData)}/edit`,
    },
}; 
