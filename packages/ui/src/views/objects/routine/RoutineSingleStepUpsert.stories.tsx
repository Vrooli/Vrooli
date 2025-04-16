/* eslint-disable @typescript-eslint/no-empty-function */
/* eslint-disable react-perf/jsx-no-new-function-as-prop */
/* eslint-disable no-magic-numbers */
import { DUMMY_ID, InputType, ResourceUsedFor, RoutineType, RoutineVersion, RoutineVersionYou, endpointsRoutineVersion, getObjectUrl, uuid } from "@local/shared";
import { HttpResponse, http } from "msw";
import { API_URL, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { RoutineSingleStepUpsert } from "./RoutineSingleStepUpsert.js";

// Create simplified mock data for Routine responses
const mockRoutineVersionData: RoutineVersion = {
    __typename: "RoutineVersion" as const,
    id: uuid(),
    apiVersion: null,
    calledByRoutineVersionsCount: Math.floor(Math.random() * 100),
    callLink: "",
    codeVersion: null,
    comments: [],
    commentsCount: 0,
    complexity: 2,
    config: JSON.stringify({
        __version: "1.0.0",
        formInput: {
            __version: "1.0.0",
            schema: {
                elements: [
                    {
                        id: "prompt",
                        type: InputType.Text,
                        label: "Prompt",
                        fieldName: "prompt",
                        description: "Enter your prompt for the AI",
                        props: {
                            defaultValue: "",
                            placeholder: "Generate a blog post about...",
                            minRows: 2,
                            maxRows: 4,
                        },
                    },
                ],
                containers: [
                    {
                        title: "Generator Input",
                        description: "Input for the AI generator",
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
                        id: "generatedText",
                        type: InputType.Text,
                        label: "Generated Output",
                        fieldName: "generatedText",
                        description: "AI generated output",
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
                        title: "Generator Output",
                        description: "Output from the AI generator",
                        totalItems: 1,
                    },
                ],
            },
        },
        callDataGenerate: {
            __version: "1.0.0",
            schema: {
                prompt: "{{input.prompt}}",
                model: null,
                botStyle: "default",
                maxTokens: 1000,
            },
        },
    }),
    created_at: new Date().toISOString(),
    directoryListings: [],
    directoryListingsCount: 0,
    forks: [],
    forksCount: 0,
    inputs: [],
    isAutomatable: true,
    isComplete: true,
    isDeleted: false,
    isLatest: true,
    isPrivate: false,
    nodes: [],
    nodeLinks: [],
    outputs: [],
    pullRequest: null,
    reports: [],
    reportsCount: 0,
    resourceList: {
        __typename: "ResourceList" as const,
        id: uuid(),
        created_at: new Date().toISOString(),
        listFor: {
            __typename: "RoutineVersion" as const,
            id: uuid(),
        },
        resources: Array.from({ length: Math.floor(Math.random() * 3) + 1 }, () => ({
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
        })),
        translations: [],
        updated_at: new Date().toISOString(),
    },
    root: {
        __typename: "Routine" as const,
        id: uuid(),
        isPrivate: false,
        owner: { __typename: "User" as const, id: uuid() },
        tags: Array.from({ length: Math.floor(Math.random() * 5) + 2 }, () => ({
            __typename: "Tag" as const,
            id: uuid(),
            tag: ["AI", "Generate", "Content", "Automation", "Workflow", "Productivity", "Tools", "Development"][Math.floor(Math.random() * 8)],
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
        })),
        versions: [],
        views: Math.floor(Math.random() * 10000),
    },
    routineType: RoutineType.Generate,
    simplicity: 5,
    subroutineLinks: [],
    suggestedNextByRoutineVersion: [],
    timesCompleted: Math.floor(Math.random() * 500),
    timesStarted: Math.floor(Math.random() * 1000),
    translations: [{
        __typename: "RoutineVersionTranslation" as const,
        id: DUMMY_ID,
        language: "en",
        description: "This is a sample AI generator routine that creates content based on your prompt. Use it to quickly generate blog posts, stories, or other text content.",
        instructions: "Enter your prompt in the input field. Be as specific as possible for better results. Then click 'Run' to generate content.",
        name: "AI Content Generator",
    }],
    updated_at: new Date().toISOString(),
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
    title: "Views/Objects/Routine/RoutineSingleStepUpsert",
    component: RoutineSingleStepUpsert,
};

// Create a new Routine
export function Create() {
    return (
        <RoutineSingleStepUpsert display="page" isCreate={true} />
    );
}
Create.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Create a new Routine in a dialog
export function CreateDialog() {
    return (
        <RoutineSingleStepUpsert
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

// Update an existing Routine
export function Update() {
    return (
        <RoutineSingleStepUpsert display="page" isCreate={false} />
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
        <RoutineSingleStepUpsert
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
        <RoutineSingleStepUpsert display="page" isCreate={false} />
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
        <RoutineSingleStepUpsert display="page" isCreate={true} />
    );
}
NonPremiumUser.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

// With Override Object (using dialog display)
export function WithOverrideObject() {
    return (
        <RoutineSingleStepUpsert
            display="dialog"
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

// With subroutine flag
export function WithSubroutineFlag() {
    return (
        <RoutineSingleStepUpsert
            display="dialog"
            isCreate={true}
            isOpen={true}
            isSubroutine={true}
            onClose={() => { }}
            onCancel={() => { }}
            onCompleted={() => { }}
            onDeleted={() => { }}
        />
    );
}
WithSubroutineFlag.parameters = {
    session: signedInPremiumWithCreditsSession,
}; 
