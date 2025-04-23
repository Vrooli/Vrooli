/* eslint-disable @typescript-eslint/no-empty-function */
/* eslint-disable react-perf/jsx-no-new-function-as-prop */
/* eslint-disable no-magic-numbers */
import { FocusMode, Meeting, Resource, ResourceUsedFor, Schedule, ScheduleException, ScheduleRecurrence, ScheduleRecurrenceType, endpointsSchedule, getObjectUrl, uuid } from "@local/shared";
import { HttpResponse, http } from "msw";
import { API_URL, loggedOutSession, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { ScheduleView } from "./ScheduleView.js";

// Create simplified mock data for Schedule responses
const mockScheduleData: Schedule = {
    __typename: "Schedule" as const,
    id: uuid(),
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    startTime: new Date().toISOString(),
    endTime: new Date(Date.now() + 3600000).toISOString(), // 1 hour from now
    timezone: "America/New_York",
    exceptions: [] as ScheduleException[],
    recurrences: [
        {
            __typename: "ScheduleRecurrence" as const,
            id: uuid(),
            recurrenceType: ScheduleRecurrenceType.Weekly,
            interval: 1,
            dayOfWeek: 2, // Tuesday
            dayOfMonth: null,
            month: null,
            duration: 60, // 60 minutes
            endDate: null,
            schedule: {} as Schedule, // Circular reference
        } as ScheduleRecurrence,
    ],
    focusModes: [{
        __typename: "FocusMode" as const,
        id: uuid(),
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        name: "Deep Work Session",
        description: "Focus time for deep work and concentration",
        filters: [],
        labels: [],
        reminderList: null,
        resourceList: {
            __typename: "ResourceList" as const,
            id: uuid(),
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            listFor: {} as any, // Circular reference
            resources: [
                {
                    __typename: "Resource" as const,
                    id: uuid(),
                    created_at: new Date().toISOString(),
                    updated_at: new Date().toISOString(),
                    usedFor: ResourceUsedFor.Context,
                    link: "https://example.com/deep-work-technique",
                    list: {} as any, // Circular reference
                    translations: [{
                        __typename: "ResourceTranslation" as const,
                        id: uuid(),
                        language: "en",
                        name: "Deep Work Technique Guide",
                        description: "A guide on how to implement deep work",
                    }],
                } as Resource,
            ],
            translations: [],
        },
        you: {
            __typename: "FocusModeYou" as const,
            canDelete: true,
            canRead: true,
            canUpdate: true,
        },
    } as FocusMode],
    meetings: [],
    runProjects: [],
    runRoutines: [],
    labels: [],
} as any; // Cast as any to bypass type checking for stories

// Create the second mock data with a meeting instead of a focus mode
const mockScheduleWithMeetingData: Schedule = {
    ...mockScheduleData,
    id: uuid(),
    focusModes: [],
    meetings: [{
        __typename: "Meeting" as const,
        id: uuid(),
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        translations: [{
            __typename: "MeetingTranslation" as const,
            id: uuid(),
            language: "en",
            name: "Weekly Team Sync",
            description: "Regular team sync meeting to discuss progress and blockers",
        }],
        schedules: [],
        participants: [],
        invites: [],
        you: {
            __typename: "MeetingYou" as const,
            canDelete: true,
            canUpdate: true,
            isInvited: true,
            isParticipant: true,
        },
    } as unknown as Meeting],
} as any; // Cast as any to bypass type checking for stories

export default {
    title: "Views/Objects/Schedule/ScheduleView",
    component: ScheduleView,
};

// No results state
export function NoResults() {
    return (
        <ScheduleView display="page" />
    );
}
NoResults.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

// Loading state
export function Loading() {
    return (
        <ScheduleView display="page" />
    );
}
Loading.parameters = {
    session: signedInNoPremiumNoCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsSchedule.findOne.endpoint}`, async () => {
                // Delay the response to simulate loading
                await new Promise(resolve => setTimeout(resolve, 120000));
                return HttpResponse.json({ data: mockScheduleData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockScheduleData)}`,
    },
};

// Signed in user viewing a focus mode schedule
export function SignedInWithFocusMode() {
    return (
        <ScheduleView display="page" />
    );
}
SignedInWithFocusMode.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsSchedule.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockScheduleData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockScheduleData)}`,
    },
};

// Signed in user viewing a meeting schedule
export function SignedInWithMeeting() {
    return (
        <ScheduleView display="page" />
    );
}
SignedInWithMeeting.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsSchedule.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockScheduleWithMeetingData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockScheduleWithMeetingData)}`,
    },
};

// Dialog view
export function DialogView() {
    // Define handler outside of JSX
    // eslint-disable-next-line func-style
    const handleClose = (): void => { };

    return (
        <ScheduleView
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
            http.get(`${API_URL}/v2/rest${endpointsSchedule.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockScheduleData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockScheduleData)}`,
    },
};

// Logged out user
export function LoggedOut() {
    return (
        <ScheduleView display="page" />
    );
}
LoggedOut.parameters = {
    session: loggedOutSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsSchedule.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockScheduleData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockScheduleData)}`,
    },
}; 
