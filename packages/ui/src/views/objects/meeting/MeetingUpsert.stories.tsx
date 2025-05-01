/* eslint-disable @typescript-eslint/no-empty-function */
/* eslint-disable react-perf/jsx-no-new-function-as-prop */
/* eslint-disable no-magic-numbers */
import { DUMMY_ID, Meeting, MeetingInviteStatus, endpointsMeeting, generatePKString, generatePublicId, getObjectUrl } from "@local/shared";
import { HttpResponse, http } from "msw";
import { API_URL, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { MeetingUpsert } from "./MeetingUpsert.js";

// Create simplified mock data for Meeting responses
const mockMeetingData: Meeting = {
    __typename: "Meeting" as const,
    id: generatePKString(),
    publicId: generatePublicId(),
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    attendees: [],
    attendeesCount: 0,
    invites: [
        {
            __typename: "MeetingInvite" as const,
            id: generatePKString(),
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            status: MeetingInviteStatus.Pending,
            message: "Please join our weekly planning meeting",
            meeting: {} as any, // This will be set by the circular reference
            user: {
                __typename: "User" as const,
                id: generatePKString(),
            } as any,
            you: {
                __typename: "MeetingInviteYou" as const,
                canDelete: true,
                canUpdate: true,
            },
        },
    ],
    invitesCount: 1,
    openToAnyoneWithInvite: true,
    schedule: {
        __typename: "Schedule" as const,
        id: generatePKString(),
        startTime: new Date().toISOString(),
        endTime: new Date(Date.now() + 3600000).toISOString(), // 1 hour from now
        timezone: "America/New_York",
        exceptions: [],
        meetings: [],
        runs: [],
        recurrences: [
            {
                __typename: "ScheduleRecurrence" as const,
                id: generatePKString(),
                dayOfMonth: null,
                dayOfWeek: 2, // Tuesday
                duration: 60, // 60 minutes
                endDate: null,
                interval: 1, // Every week
                month: null,
                recurrenceType: "Weekly",
                schedule: {} as any, // This will be set by the circular reference
            },
        ],
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
    } as any,
    showOnTeamProfile: true,
    team: {
        __typename: "Team" as const,
        id: generatePKString(),
        bannerImage: null,
        profileImage: null,
        handle: "example-team",
        you: {
            __typename: "TeamYou" as const,
            canAddMembers: true,
            canBookmark: true,
            canDelete: true,
            canRead: true,
            canReport: true,
            canUpdate: true,
            isBookmarked: false,
            isViewed: true,
            yourMembership: {
                __typename: "Member" as const,
                id: generatePKString(),
                isAdmin: true,
                permissions: "{}",
            } as any,
        },
    } as any,
    translations: [
        {
            __typename: "MeetingTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            name: "Weekly Planning Meeting",
            description: "This is our weekly planning meeting to discuss progress and next steps.",
            link: "https://meet.example.com/123",
        },
    ],
    translationsCount: 1,
    you: {
        __typename: "MeetingYou" as const,
        canDelete: true,
        canInvite: true,
        canUpdate: true,
    },
};

export default {
    title: "Views/Objects/Meeting/MeetingUpsert",
    component: MeetingUpsert,
};

// Create a new Meeting
export function Create() {
    return (
        <MeetingUpsert display="Page" isCreate={true} />
    );
}
Create.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Create a new Meeting in a dialog
export function CreateDialog() {
    return (
        <MeetingUpsert
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

// Update an existing Meeting
export function Update() {
    return (
        <MeetingUpsert display="Page" isCreate={false} />
    );
}
Update.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsMeeting.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockMeetingData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockMeetingData)}/edit`,
    },
};

// Update an existing Meeting in a dialog
export function UpdateDialog() {
    return (
        <MeetingUpsert
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
            http.get(`${API_URL}/v2/rest${endpointsMeeting.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockMeetingData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockMeetingData)}/edit`,
    },
};

// Loading state
export function Loading() {
    return (
        <MeetingUpsert display="Page" isCreate={false} />
    );
}
Loading.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsMeeting.findOne.endpoint}`, async () => {
                // Delay the response to simulate loading
                const LOADING_DELAY = 120_000; // 2 minutes delay
                await new Promise(resolve => setTimeout(resolve, LOADING_DELAY));
                return HttpResponse.json({ data: mockMeetingData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockMeetingData)}/edit`,
    },
};

// Non-premium user
export function NonPremiumUser() {
    return (
        <MeetingUpsert display="Page" isCreate={true} />
    );
}
NonPremiumUser.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

// With Override Object (using dialog display)
export function WithOverrideObject() {
    return (
        <MeetingUpsert
            display="Dialog"
            isCreate={true}
            isOpen={true}
            onClose={() => { }}
            onCancel={() => { }}
            onCompleted={() => { }}
            onDeleted={() => { }}
            overrideObject={mockMeetingData}
        />
    );
}
WithOverrideObject.parameters = {
    session: signedInPremiumWithCreditsSession,
}; 
