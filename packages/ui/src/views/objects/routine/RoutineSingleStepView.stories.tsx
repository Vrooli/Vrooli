/* eslint-disable no-magic-numbers */
import { DUMMY_ID, InputType, ResourceUsedFor, RoutineType, RoutineVersion, RunRoutine, RunStatus, Tag, User, endpointsRoutineVersion, endpointsRunRoutine, getObjectUrl, uuid } from "@local/shared";
import { HttpResponse, http } from "msw";
import { API_URL, loggedOutSession, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { RoutineSingleStepView } from "./RoutineSingleStepView.js";

// Create simplified mock data for Routine responses
const mockRoutineVersionData: RoutineVersion = {
    __typename: "RoutineVersion" as const,
    id: uuid(),
    calledByRoutineVersionsCount: Math.floor(Math.random() * 100),
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
        })) as any,
        translations: [],
        updated_at: new Date().toISOString(),
    },
    root: {
        __typename: "Routine" as const,
        id: uuid(),
        isPrivate: false,
        owner: { __typename: "User" as const, id: uuid() } as User,
        tags: Array.from({ length: Math.floor(Math.random() * 5) + 2 }, () => ({
            __typename: "Tag" as const,
            id: uuid(),
            tag: ["AI", "Generate", "Content", "Automation", "Workflow", "Productivity", "Tools", "Development"][Math.floor(Math.random() * 8)],
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
        })) as Tag[],
        versions: [],
        views: Math.floor(Math.random() * 10000),
    } as any,
    routineType: RoutineType.Generate,
    simplicity: 5,
    subroutineLinks: [],
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
        isBookmarked: false,
        isOwner: false,
        isStarred: false,
    },
};

// Mock run data
const mockRunRoutineData: RunRoutine = {
    __typename: "RunRoutine" as const,
    id: uuid(),
    completedComplexity: 2,
    complexity: 2,
    created_at: new Date().toISOString(),
    creator: { __typename: "User" as const, id: uuid() } as User,
    isCompleted: true,
    inputs: [],
    outputs: [],
    routineVersion: mockRoutineVersionData,
    simplicity: 5,
    status: RunStatus.Completed,
    timeElapsed: 3240,
    timeStarted: new Date().toISOString(),
    totalComplexity: 2,
    updated_at: new Date().toISOString(),
    you: {
        canDelete: true,
        canRead: true,
        canUpdate: true,
        isOwner: true,
    },
};

export default {
    title: "Views/Objects/Routine/RoutineSingleStepView",
    component: RoutineSingleStepView,
};

export function NoResult() {
    return (
        <RoutineSingleStepView display="page" />
    );
}
NoResult.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

export function Loading() {
    return (
        <RoutineSingleStepView display="page" />
    );
}
Loading.parameters = {
    session: signedInNoPremiumNoCreditsSession,
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
        path: `${API_URL}/v2/rest${getObjectUrl(mockRoutineVersionData)}`,
    },
};

export function SignedInWithResults() {
    return (
        <RoutineSingleStepView display="page" />
    );
}
SignedInWithResults.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsRoutineVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockRoutineVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockRoutineVersionData)}`,
    },
};

export function LoggedOutWithResults() {
    return (
        <RoutineSingleStepView display="page" />
    );
}
LoggedOutWithResults.parameters = {
    session: loggedOutSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsRoutineVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockRoutineVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockRoutineVersionData)}`,
    },
};

export function WithActiveRun() {
    return (
        <RoutineSingleStepView display="page" />
    );
}
WithActiveRun.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsRoutineVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockRoutineVersionData });
            }),
            http.get(`${API_URL}/v2/rest${endpointsRunRoutine.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockRunRoutineData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockRoutineVersionData)}?runId=${uuid()}`,
    },
};

export function Own() {
    return (
        <RoutineSingleStepView display="page" />
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
                        isBookmarked: false,
                        isOwner: true,
                        isStarred: false,
                    },
                };

                return HttpResponse.json({ data: mockWithOwnerPermissions });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockRoutineVersionData)}`,
    },
}; 
