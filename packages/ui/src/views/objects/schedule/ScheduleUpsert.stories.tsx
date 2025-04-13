/* eslint-disable @typescript-eslint/no-empty-function */
/* eslint-disable react-perf/jsx-no-new-function-as-prop */
/* eslint-disable no-magic-numbers */
import { FocusMode, Resource, ResourceUsedFor, Schedule, ScheduleException, ScheduleRecurrence, ScheduleRecurrenceType, endpointsSchedule, getObjectUrl, uuid } from "@local/shared";
import { HttpResponse, http } from "msw";
import { API_URL, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { ScheduleUpsert } from "./ScheduleUpsert.js";

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
};

export default {
    title: "Views/Objects/Schedule/ScheduleUpsert",
    component: ScheduleUpsert,
};

// Create a new Schedule
export function Create() {
    return (
        <ScheduleUpsert display="page" isCreate={true} />
    );
}
Create.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Create a new Schedule in a dialog with FocusMode type
export function CreateDialog() {
    return (
        <ScheduleUpsert
            display="dialog"
            isCreate={true}
            isOpen={true}
            defaultScheduleFor="FocusMode"
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
            display="dialog"
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
        <ScheduleUpsert display="page" isCreate={false} />
    );
}
Update.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsSchedule.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockScheduleData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockScheduleData)}/edit`,
    },
};

// Update an existing Schedule in a dialog
export function UpdateDialog() {
    return (
        <ScheduleUpsert
            display="dialog"
            isCreate={false}
            isOpen={true}
            canSetScheduleFor={false}
            defaultScheduleFor="FocusMode"
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
            http.get(`${API_URL}/v2/rest${endpointsSchedule.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockScheduleData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockScheduleData)}/edit`,
    },
};

// Loading state
export function Loading() {
    return (
        <ScheduleUpsert display="page" isCreate={false} />
    );
}
Loading.parameters = {
    session: signedInPremiumWithCreditsSession,
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
        path: `${API_URL}/v2/rest${getObjectUrl(mockScheduleData)}/edit`,
    },
};

// Non-premium user
export function NonPremiumUser() {
    return (
        <ScheduleUpsert display="page" isCreate={true} />
    );
}
NonPremiumUser.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

// With Override Object (using dialog display)
export function WithOverrideObject() {
    return (
        <ScheduleUpsert
            display="dialog"
            isCreate={true}
            isOpen={true}
            canSetScheduleFor={true}
            defaultScheduleFor="FocusMode"
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
