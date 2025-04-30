/* eslint-disable @typescript-eslint/no-empty-function */
/* eslint-disable react-perf/jsx-no-new-function-as-prop */
/* eslint-disable no-magic-numbers */
/* eslint-disable @typescript-eslint/consistent-type-assertions */
import { RunProject, RunStatus, Schedule, ScheduleRecurrenceType, endpointsRunProject, getObjectUrl, uuid } from "@local/shared";
import { HttpResponse, http } from "msw";
import { API_URL, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { RunProjectUpsert } from "./RunProjectUpsert.js";

// Create mock data structures for testing
// Note: Type assertions are used to bypass strict typing for story purposes
const mockScheduleId = uuid();
const mockScheduleData = {
    __typename: "Schedule",
    id: mockScheduleId,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    startTime: new Date().toISOString(),
    endTime: new Date(Date.now() + 3600000).toISOString(), // 1 hour from now
    timezone: "America/New_York",
    exceptions: [],
    recurrences: [
        {
            __typename: "ScheduleRecurrence",
            id: uuid(),
            recurrenceType: ScheduleRecurrenceType.Weekly,
            interval: 1,
            dayOfWeek: 2, // Tuesday
            dayOfMonth: null,
            month: null,
            duration: 60, // 60 minutes
            endDate: null,
            schedule: {
                __typename: "Schedule",
                id: mockScheduleId,
            },
        },
    ],
    meetings: [],
    runProjects: [],
    runRoutines: [],
    labels: [],
} as Schedule;

const mockProjectVersionId = uuid();
const mockRunProjectId = uuid();

const mockRunProjectData = {
    __typename: "RunProject",
    id: mockRunProjectId,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    completedComplexity: 0,
    contextSwitches: 0,
    isPrivate: true,
    name: "Web App Development Run",
    projectVersion: {
        __typename: "ProjectVersion",
        id: mockProjectVersionId,
        // Include essential display properties
        versionLabel: "v1.0",
        translations: [
            {
                __typename: "ProjectVersionTranslation",
                id: uuid(),
                language: "en",
                name: "Web App Development",
                description: "Project to develop a responsive web application",
            },
        ],
    },
    schedule: mockScheduleData,
    status: RunStatus.Scheduled,
    steps: [],
    timeElapsed: 0,
    stepsCount: 0,
    completedAt: null,
    data: null,
    lastStep: null,
    you: {
        __typename: "RunProjectYou",
        canDelete: true,
        canRead: true,
        canUpdate: true,
    },
} as RunProject;

// Create a version without a schedule for some stories
const mockRunProjectWithoutScheduleData = {
    ...mockRunProjectData,
    id: uuid(),
    schedule: null,
} as RunProject;

export default {
    title: "Views/Objects/RunProject/RunProjectUpsert",
    component: RunProjectUpsert,
};

// Create a new RunProject
export function Create() {
    return (
        <RunProjectUpsert display="page" isCreate={true} />
    );
}
Create.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Create a new RunProject in a dialog
export function CreateDialog() {
    return (
        <RunProjectUpsert
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

// Update an existing RunProject
export function Update() {
    return (
        <RunProjectUpsert display="page" isCreate={false} />
    );
}
Update.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsRunProject.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockRunProjectData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockRunProjectData)}/edit`,
    },
};

// Update a RunProject without schedule
export function UpdateNoSchedule() {
    return (
        <RunProjectUpsert display="page" isCreate={false} />
    );
}
UpdateNoSchedule.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsRunProject.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockRunProjectWithoutScheduleData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockRunProjectWithoutScheduleData)}/edit`,
    },
};

// Update an existing RunProject in a dialog
export function UpdateDialog() {
    return (
        <RunProjectUpsert
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
            http.get(`${API_URL}/v2/rest${endpointsRunProject.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockRunProjectData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockRunProjectData)}/edit`,
    },
};

// Loading state
export function Loading() {
    return (
        <RunProjectUpsert display="page" isCreate={false} />
    );
}
Loading.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsRunProject.findOne.endpoint}`, async () => {
                // Delay the response to simulate loading
                await new Promise(resolve => setTimeout(resolve, 120000));
                return HttpResponse.json({ data: mockRunProjectData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockRunProjectData)}/edit`,
    },
};

// Non-premium user
export function NonPremiumUser() {
    return (
        <RunProjectUpsert display="page" isCreate={true} />
    );
}
NonPremiumUser.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

// With Override Object
export function WithOverrideObject() {
    return (
        <RunProjectUpsert
            display="Dialog"
            isCreate={true}
            isOpen={true}
            onClose={() => { }}
            onCancel={() => { }}
            onCompleted={() => { }}
            onDeleted={() => { }}
            overrideObject={mockRunProjectData}
        />
    );
}
WithOverrideObject.parameters = {
    session: signedInPremiumWithCreditsSession,
}; 
