/* eslint-disable @typescript-eslint/no-empty-function */
/* eslint-disable react-perf/jsx-no-new-function-as-prop */
/* eslint-disable no-magic-numbers */
import { Meeting, Schedule, ScheduleException, ScheduleRecurrence, ScheduleRecurrenceType, endpointsSchedule, generatePKString, getObjectUrl } from "@local/shared";
import { HttpResponse, http } from "msw";
import { API_URL, loggedOutSession, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { ScheduleView } from "./ScheduleView.js";

// Create simplified mock data for Schedule responses
const mockScheduleData: Schedule = {
    id: generatePKString(),
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    startTime: new Date().toISOString(),
    endTime: new Date(Date.now() + 3600000).toISOString(), // 1 hour from now
    timezone: "America/New_York",
    exceptions: [] as ScheduleException[],
    recurrences: [
        {
            __typename: "ScheduleRecurrence" as const,
            id: generatePKString(),
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
    meetings: [{
        __typename: "Meeting" as const,
        id: generatePKString(),
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        translations: [{
            __typename: "MeetingTranslation" as const,
            id: generatePKString(),
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
    runs: [],
} as any; // Cast as any to bypass type checking for stories

export default {
    title: "Views/Objects/Schedule/ScheduleView",
    component: ScheduleView,
};

// No results state
export function NoResults() {
    return (
        <ScheduleView display="Page" />
    );
}
NoResults.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

// Loading state
export function Loading() {
    return (
        <ScheduleView display="Page" />
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

// Rename SignedInWithMeeting to SignedIn or Default
export function SignedIn() {
    return (
        <ScheduleView display="Page" />
    );
}
SignedIn.parameters = {
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
        <ScheduleView display="Page" />
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
