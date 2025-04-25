/* eslint-disable no-magic-numbers */
import { DUMMY_ID, Meeting, MeetingInviteStatus, endpointsMeeting, getObjectUrl, uuid } from "@local/shared";
import { HttpResponse, http } from "msw";
import { API_URL, loggedOutSession, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { MeetingView } from "./MeetingView.js";

// Create simplified mock data for Meeting responses
const mockMeetingData: Meeting = {
    __typename: "Meeting" as const,
    id: uuid(),
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    attendees: [],
    attendeesCount: 0,
    invites: [
        {
            __typename: "MeetingInvite" as const,
            id: uuid(),
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            status: MeetingInviteStatus.Pending,
            message: "Please join our weekly planning meeting",
            meeting: {} as any, // This will be set by the circular reference
            user: {
                __typename: "User" as const,
                id: uuid(),
            } as any,
            you: {
                __typename: "MeetingInviteYou" as const,
                canDelete: true,
                canUpdate: true,
            },
        },
    ],
    invitesCount: 1,
    labels: [
        {
            __typename: "Label" as const,
            id: uuid(),
            color: "#FF5722",
            text: "Planning",
        } as any,
    ],
    labelsCount: 1,
    openToAnyoneWithInvite: true,
    restrictedToRoles: [],
    schedule: {
        __typename: "Schedule" as const,
        id: uuid(),
        startTime: new Date().toISOString(),
        endTime: new Date(Date.now() + 3600000).toISOString(), // 1 hour from now
        timezone: "America/New_York",
        exceptions: [],
        labels: [],
        meetings: [],
        runProjects: [],
        runRoutines: [],
        recurrences: [
            {
                __typename: "ScheduleRecurrence" as const,
                id: uuid(),
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
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
    } as any,
    showOnTeamProfile: true,
    team: {
        __typename: "Team" as const,
        id: uuid(),
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
                id: uuid(),
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
    title: "Views/Objects/Meeting/MeetingView",
    component: MeetingView,
};

export function NoResult() {
    return (
        <MeetingView display="page" />
    );
}
NoResult.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

export function Loading() {
    return (
        <MeetingView display="page" />
    );
}
Loading.parameters = {
    session: signedInNoPremiumNoCreditsSession,
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
        path: `${API_URL}/v2/rest${getObjectUrl(mockMeetingData)}`,
    },
};

export function SignInWithResults() {
    return (
        <MeetingView display="page" />
    );
}
SignInWithResults.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsMeeting.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockMeetingData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockMeetingData)}`,
    },
};

export function LoggedOutWithResults() {
    return (
        <MeetingView display="page" />
    );
}
LoggedOutWithResults.parameters = {
    session: loggedOutSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsMeeting.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockMeetingData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockMeetingData)}`,
    },
}; 
