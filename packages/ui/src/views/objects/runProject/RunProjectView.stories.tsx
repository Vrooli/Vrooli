/* eslint-disable @typescript-eslint/no-empty-function */
/* eslint-disable react-perf/jsx-no-new-function-as-prop */
/* eslint-disable no-magic-numbers */
/* eslint-disable @typescript-eslint/consistent-type-assertions */
/* eslint-disable @typescript-eslint/no-unused-vars */
import { RunProject, RunStatus, Schedule, ScheduleRecurrenceType, endpointsRunProject, getObjectUrl, uuid } from "@local/shared";
import { HttpResponse, http } from "msw";
import { API_URL, loggedOutSession, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { RunProjectView } from "./RunProjectView.js";

// Create mock data structures for testing
// Note: These are NOT complete implementations of the types - only the props needed for component display
const mockScheduleId = uuid();
const mockScheduleData = {
    __typename: "Schedule",
    id: mockScheduleId,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
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
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
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
} as unknown as RunProject;

// Create a version with RunStatus.InProgress for some stories
const mockRunProjectInProgressData = {
    ...mockRunProjectData,
    id: uuid(),
    status: RunStatus.InProgress,
    completedComplexity: 35, // Some progress made
    timeElapsed: 2700000, // 45 minutes in milliseconds
} as unknown as RunProject;

// Create a version with RunStatus.Completed for some stories
const mockRunProjectCompletedData = {
    ...mockRunProjectData,
    id: uuid(),
    status: RunStatus.Completed,
    completedComplexity: 100, // Fully completed
    timeElapsed: 7200000, // 2 hours in milliseconds
} as unknown as RunProject;

export default {
    title: "Views/Objects/RunProject/RunProjectView",
    component: RunProjectView,
};

// No results state
export function NoResults() {
    return (
        <RunProjectView display="page" />
    );
}
NoResults.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

// Loading state
export function Loading() {
    return (
        <RunProjectView display="page" />
    );
}
Loading.parameters = {
    session: signedInNoPremiumNoCreditsSession,
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
        path: `${API_URL}/v2/rest${getObjectUrl(mockRunProjectData)}`,
    },
};

// Scheduled RunProject
export function ScheduledRun() {
    return (
        <RunProjectView display="page" />
    );
}
ScheduledRun.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsRunProject.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockRunProjectData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockRunProjectData)}`,
    },
};

// In Progress RunProject
export function InProgressRun() {
    return (
        <RunProjectView display="page" />
    );
}
InProgressRun.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsRunProject.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockRunProjectInProgressData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockRunProjectInProgressData)}`,
    },
};

// Completed RunProject
export function CompletedRun() {
    return (
        <RunProjectView display="page" />
    );
}
CompletedRun.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsRunProject.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockRunProjectCompletedData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockRunProjectCompletedData)}`,
    },
};

// Dialog view
export function DialogView() {
    // Define handler outside of JSX
    // eslint-disable-next-line func-style
    const handleClose = (): void => { };

    return (
        <RunProjectView
            display="Dialog"
            isOpen={true}
            onClose={handleClose}
        />
    );
}
DialogView.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsRunProject.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockRunProjectData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockRunProjectData)}`,
    },
};

// Logged out user
export function LoggedOut() {
    return (
        <RunProjectView display="page" />
    );
}
LoggedOut.parameters = {
    session: loggedOutSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsRunProject.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockRunProjectData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockRunProjectData)}`,
    },
}; 
