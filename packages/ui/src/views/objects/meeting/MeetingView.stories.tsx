// AI_CHECK: TYPE_SAFETY=fixed-6-type-assertions | LAST: 2025-07-01
/* eslint-disable no-magic-numbers */
import { DUMMY_ID, MeetingInviteStatus, endpointsMeeting, generatePK, getObjectUrl, type Meeting } from "@vrooli/shared";
import { HttpResponse, http } from "msw";
import { API_URL, loggedOutSession, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { getMockUrl, getStoryRoutePath } from "../../../__test/helpers/storybookMocking.js";
import { MeetingView } from "./MeetingView.js";

// Create simplified mock data for Meeting responses
const mockMeetingData: Meeting = {
    __typename: "Meeting" as const,
    id: generatePK().toString(),
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    attendees: [],
    attendeesCount: 0,
    invites: [
        {
            __typename: "MeetingInvite" as const,
            id: generatePK().toString(),
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            status: MeetingInviteStatus.Pending,
            message: "Please join our weekly planning meeting",
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            meeting: {} as any, // This will be set by the circular reference
            user: {
                __typename: "User" as const,
                id: generatePK().toString(),
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
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
        id: generatePK().toString(),
        startTime: new Date().toISOString(),
        endTime: new Date(Date.now() + 3600000).toISOString(), // 1 hour from now
        timezone: "America/New_York",
        exceptions: [],
        meetings: [],
        runs: [],
        recurrences: [
            {
                __typename: "ScheduleRecurrence" as const,
                id: generatePK().toString(),
                dayOfMonth: null,
                dayOfWeek: 2, // Tuesday
                duration: 60, // 60 minutes
                endDate: null,
                interval: 1, // Every week
                month: null,
                recurrenceType: "Weekly",
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                schedule: {} as any, // This will be set by the circular reference
            },
        ],
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any,
    showOnTeamProfile: true,
    team: {
        __typename: "Team" as const,
        id: generatePK().toString(),
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
                id: generatePK().toString(),
                isAdmin: true,
                permissions: "{}",
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            } as any,
        },
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
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
        <MeetingView display="Page" />
    );
}
NoResult.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

export function Loading() {
    return (
        <MeetingView display="Page" />
    );
}
Loading.parameters = {
    session: signedInNoPremiumNoCreditsSession,
    msw: {
        handlers: [
            http.get(getMockUrl(endpointsMeeting.findOne), async () => {
                // Delay the response to simulate loading
                const LOADING_DELAY = 120_000; // 2 minutes delay
                await new Promise(resolve => setTimeout(resolve, LOADING_DELAY));
                return HttpResponse.json({ data: mockMeetingData });
            }),
        ],
    },
    route: {
        path: getStoryRoutePath(mockMeetingData),
    },
};

export function SignInWithResults() {
    return (
        <MeetingView display="Page" />
    );
}
SignInWithResults.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(getMockUrl(endpointsMeeting.findOne), () => {
                return HttpResponse.json({ data: mockMeetingData });
            }),
        ],
    },
    route: {
        path: getStoryRoutePath(mockMeetingData),
    },
};

export function LoggedOutWithResults() {
    return (
        <MeetingView display="Page" />
    );
}
LoggedOutWithResults.parameters = {
    session: loggedOutSession,
    msw: {
        handlers: [
            http.get(getMockUrl(endpointsMeeting.findOne), () => {
                return HttpResponse.json({ data: mockMeetingData });
            }),
        ],
    },
    route: {
        path: getStoryRoutePath(mockMeetingData),
    },
}; 
