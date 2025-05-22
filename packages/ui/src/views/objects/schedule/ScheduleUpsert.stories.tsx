/* eslint-disable @typescript-eslint/no-empty-function */
/* eslint-disable react-perf/jsx-no-new-function-as-prop */
/* eslint-disable no-magic-numbers */
import { type Schedule, type ScheduleException, type ScheduleRecurrence, ScheduleRecurrenceType, endpointsSchedule, generatePKString, getObjectUrl } from "@local/shared";
import { HttpResponse, http } from "msw";
import { API_URL, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { ScheduleUpsert } from "./ScheduleUpsert.js";

// Create simplified mock data for Schedule responses
const mockScheduleData: any = {
    __typename: "Schedule" as const,
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
            name: "Team Meeting",
            description: "Scheduled team meeting",
        }],
        schedules: [],
        participants: [],
        invites: [],
        you: {
            __typename: "MeetingYou" as const,
            canDelete: true,
            canUpdate: true,
            isInvited: false,
            isParticipant: false,
        },
    }],
    runs: [],
};

export default {
    title: "Views/Objects/Schedule/ScheduleUpsert",
    component: ScheduleUpsert,
};

// Create a new Schedule
export function Create() {
    return (
        <ScheduleUpsert display="Page" isCreate={true} />
    );
}
Create.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Create a new Schedule in a dialog with Meeting type
export function CreateDialog() {
    return (
        <ScheduleUpsert
            display="Dialog"
            isCreate={true}
            isOpen={true}
            defaultScheduleFor="Meeting"
            canSetScheduleFor={true}
            isMutate={true}
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

// Create a new Schedule in a dialog with Meeting type
export function CreateDialogForMeeting() {
    return (
        <ScheduleUpsert
            display="Dialog"
            isCreate={true}
            isOpen={true}
            defaultScheduleFor="Meeting"
            canSetScheduleFor={true}
            isMutate={true}
            onClose={() => { }}
            onCancel={() => { }}
            onCompleted={() => { }}
            onDeleted={() => { }}
        />
    );
}
CreateDialogForMeeting.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Update an existing Schedule
export function Update() {
    return (
        <ScheduleUpsert display="Page" isCreate={false} />
    );
}
Update.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2${endpointsSchedule.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockScheduleData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2${getObjectUrl(mockScheduleData)}/edit`,
    },
};

// Update an existing Schedule in a dialog
export function UpdateDialog() {
    return (
        <ScheduleUpsert
            display="Dialog"
            isCreate={false}
            isOpen={true}
            canSetScheduleFor={false}
            defaultScheduleFor="Meeting"
            isMutate={true}
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
            http.get(`${API_URL}/v2${endpointsSchedule.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockScheduleData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2${getObjectUrl(mockScheduleData)}/edit`,
    },
};

// Loading state
export function Loading() {
    return (
        <ScheduleUpsert display="Page" isCreate={false} />
    );
}
Loading.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2${endpointsSchedule.findOne.endpoint}`, async () => {
                // Delay the response to simulate loading
                await new Promise(resolve => setTimeout(resolve, 120000));
                return HttpResponse.json({ data: mockScheduleData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2${getObjectUrl(mockScheduleData)}/edit`,
    },
};

// Non-premium user
export function NonPremiumUser() {
    return (
        <ScheduleUpsert display="Page" isCreate={true} />
    );
}
NonPremiumUser.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

// With Override Object (using dialog display)
export function WithOverrideObject() {
    return (
        <ScheduleUpsert
            display="Dialog"
            isCreate={true}
            isOpen={true}
            canSetScheduleFor={true}
            defaultScheduleFor="Meeting"
            isMutate={false}
            onClose={() => { }}
            onCancel={() => { }}
            onCompleted={() => { }}
            onDeleted={() => { }}
            overrideObject={mockScheduleData}
        />
    );
}
WithOverrideObject.parameters = {
    session: signedInPremiumWithCreditsSession,
}; 
