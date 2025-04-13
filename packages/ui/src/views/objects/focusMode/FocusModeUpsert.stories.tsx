/* eslint-disable react-perf/jsx-no-new-function-as-prop */
/* eslint-disable no-magic-numbers */
/* eslint-disable @typescript-eslint/no-empty-function */
import { FocusMode, FocusModeFilterType, ResourceUsedFor, ScheduleRecurrenceType, endpointsFocusMode, getObjectUrl, uuid } from "@local/shared";
import { HttpResponse, http } from "msw";
import { API_URL, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { FocusModeUpsert } from "./FocusModeUpsert.js";

// Create simplified mock data for FocusMode responses
const mockFocusModeData: FocusMode = {
    __typename: "FocusMode" as const,
    id: uuid(),
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    name: "Deep Work Focus Mode",
    description: "This is a **detailed** description for the mock focus mode using markdown.\nLorem ipsum dolor sit amet, consectetur adipiscing elit.",
    filters: [{
        __typename: "FocusModeFilter" as const,
        id: uuid(),
        filterType: FocusModeFilterType.Hide,
        focusMode: {} as FocusMode, // Circular reference will be filled
        tag: {
            __typename: "Tag" as const,
            id: uuid(),
            tag: "Social Media",
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            apis: [],
            bookmarkedBy: [],
            bookmarks: 0,
            codes: [],
            notes: [],
            posts: [],
            projects: [],
            reports: [],
            routines: [],
            standards: [],
            teams: [],
            translations: [],
            you: {
                __typename: "TagYou",
                isBookmarked: false,
                isOwn: false,
            },
        },
    }],
    labels: [{
        __typename: "Label" as const,
        id: uuid(),
        color: "#4287f5",
        label: "Productivity",
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        apis: [],
        apisCount: 0,
        codes: [],
        codesCount: 0,
        focusModes: [],
        focusModesCount: 0,
        issues: [],
        issuesCount: 0,
        meetings: [],
        meetingsCount: 0,
        notes: [],
        notesCount: 0,
        projects: [],
        projectsCount: 0,
        routines: [],
        routinesCount: 0,
        schedules: [],
        schedulesCount: 0,
        standards: [],
        standardsCount: 0,
        translations: [],
        translationsCount: 0,
        owner: {
            __typename: "User",
            id: uuid(),
        } as any,
        you: {
            __typename: "LabelYou",
            canDelete: true,
            canUpdate: true,
        },
    }],
    reminderList: {
        __typename: "ReminderList" as const,
        id: uuid(),
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        reminders: [{
            __typename: "Reminder" as const,
            id: uuid(),
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            name: "Prepare workspace",
            description: "Clear desk and set up environment",
            dueDate: new Date().toISOString(),
            index: 0,
            isComplete: false,
            reminderList: {} as any, // Will be filled by circular reference
            reminderItems: [],
        }],
        focusMode: {} as any, // Will be filled by circular reference
    },
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
            },
        ],
        translations: [],
    },
    schedule: {
        __typename: "Schedule" as const,
        id: uuid(),
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        startTime: new Date().toISOString(),
        endTime: new Date(new Date().getTime() + 2 * 60 * 60 * 1000).toISOString(), // 2 hours later
        timezone: "America/New_York",
        exceptions: [],
        recurrences: [{
            __typename: "ScheduleRecurrence" as const,
            id: uuid(),
            recurrenceType: ScheduleRecurrenceType.Weekly,
            dayOfWeek: 1, // Monday
            dayOfMonth: null,
            month: null,
            duration: 120, // 2 hours in minutes
            interval: 1, // Every week
            endDate: null,
            schedule: {} as any, // Circular reference
        }],
        focusModes: [],
        labels: [],
        meetings: [],
        runProjects: [],
        runRoutines: [],
    },
    you: {
        __typename: "FocusModeYou" as const,
        canDelete: true,
        canRead: true,
        canUpdate: true,
    },
};

// Resolve circular references
mockFocusModeData.filters[0].focusMode = mockFocusModeData;
if (mockFocusModeData.reminderList) {
    mockFocusModeData.reminderList.focusMode = mockFocusModeData;
    mockFocusModeData.reminderList.reminders[0].reminderList = mockFocusModeData.reminderList;
}
if (mockFocusModeData.resourceList) {
    mockFocusModeData.resourceList.listFor = mockFocusModeData;
    mockFocusModeData.resourceList.resources[0].list = mockFocusModeData.resourceList;
}
if (mockFocusModeData.schedule) {
    mockFocusModeData.schedule.focusModes = [mockFocusModeData];
    mockFocusModeData.schedule.recurrences[0].schedule = mockFocusModeData.schedule;
}

export default {
    title: "Views/Objects/FocusMode/FocusModeUpsert",
    component: FocusModeUpsert,
};

// Create a new FocusMode
export function Create() {
    return (
        <FocusModeUpsert display="page" isCreate={true} />
    );
}
Create.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Create a new FocusMode in a dialog
export function CreateDialog() {
    return (
        <FocusModeUpsert
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

// Update an existing FocusMode
export function Update() {
    return (
        <FocusModeUpsert display="page" isCreate={false} />
    );
}
Update.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsFocusMode.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockFocusModeData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockFocusModeData)}/edit`,
    },
};

// Update an existing FocusMode in a dialog
export function UpdateDialog() {
    return (
        <FocusModeUpsert
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
            http.get(`${API_URL}/v2/rest${endpointsFocusMode.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockFocusModeData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockFocusModeData)}/edit`,
    },
};

// Loading state
export function Loading() {
    return (
        <FocusModeUpsert display="page" isCreate={false} />
    );
}
Loading.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsFocusMode.findOne.endpoint}`, async () => {
                // Delay the response to simulate loading
                await new Promise(resolve => setTimeout(resolve, 120000));
                return HttpResponse.json({ data: mockFocusModeData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockFocusModeData)}/edit`,
    },
};

// Non-premium user
export function NonPremiumUser() {
    return (
        <FocusModeUpsert display="page" isCreate={true} />
    );
}
NonPremiumUser.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

// With Override Object (using dialog display)
export function WithOverrideObject() {
    return (
        <FocusModeUpsert
            display="dialog"
            isCreate={true}
            isOpen={true}
            onClose={() => { }}
            onCancel={() => { }}
            onCompleted={() => { }}
            onDeleted={() => { }}
            overrideObject={mockFocusModeData}
        />
    );
}
WithOverrideObject.parameters = {
    session: signedInPremiumWithCreditsSession,
}; 
