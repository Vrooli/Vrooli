import { type Meeting, type MeetingInvite, type MeetingInviteStatus, type MeetingYou, type MeetingInviteYou, type Schedule, type ScheduleRecurrence, type ScheduleRecurrenceType } from "@vrooli/shared";
import { minimalUserResponse, completeUserResponse } from "./userResponses.js";
import { minimalTeamResponse, completeTeamResponse } from "./teamResponses.js";

/**
 * API response fixtures for meetings
 * These represent what components receive from API calls
 */

/**
 * Mock schedule data for meetings
 */
const oneTimeSchedule: Schedule = {
    __typename: "Schedule",
    id: "schedule_123456789012345",
    startTime: "2024-03-15T14:00:00Z",
    endTime: "2024-03-15T15:00:00Z",
    timezone: "America/New_York",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    publicId: "sched_onetime_123456",
    user: minimalUserResponse,
    exceptions: [],
    recurrences: [],
    meetings: [],
    runs: [],
};

const weeklyRecurringSchedule: Schedule = {
    __typename: "Schedule",
    id: "schedule_recurring_123456",
    startTime: "2024-03-01T10:00:00Z",
    endTime: "2024-03-01T11:00:00Z",
    timezone: "UTC",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    publicId: "sched_weekly_123456",
    user: completeUserResponse,
    exceptions: [],
    meetings: [],
    runs: [],
    recurrences: [
        {
            __typename: "ScheduleRecurrence",
            id: "recurrence_weekly_123456",
            recurrenceType: "Weekly" as ScheduleRecurrenceType,
            interval: 1,
            dayOfWeek: 1, // Monday
            duration: 3600, // 1 hour in seconds
            dayOfMonth: null,
            month: null,
            endDate: null,
            schedule: null as any, // Avoid circular reference
        },
    ],
};

/**
 * Mock meeting invites
 */
const pendingInvite: MeetingInvite = {
    __typename: "MeetingInvite",
    id: "invite_pending_123456789",
    status: "Pending" as MeetingInviteStatus,
    message: "Please join us for the weekly team standup",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    user: minimalUserResponse,
    meeting: null as any, // Avoid circular reference
    you: {
        __typename: "MeetingInviteYou",
        canDelete: false,
        canUpdate: false,
    },
};

const acceptedInvite: MeetingInvite = {
    __typename: "MeetingInvite",
    id: "invite_accepted_123456789",
    status: "Accepted" as MeetingInviteStatus,
    message: null,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-15T10:00:00Z",
    user: completeUserResponse,
    meeting: null as any, // Avoid circular reference
    you: {
        __typename: "MeetingInviteYou",
        canDelete: true,
        canUpdate: true,
    },
};

const declinedInvite: MeetingInvite = {
    __typename: "MeetingInvite",
    id: "invite_declined_123456789",
    status: "Declined" as MeetingInviteStatus,
    message: "Unfortunately I can't make it this time",
    createdAt: "2024-01-10T00:00:00Z",
    updatedAt: "2024-01-12T09:30:00Z",
    user: {
        ...minimalUserResponse,
        id: "user_decline_123456789",
        handle: "busyuser",
        name: "Busy User",
    },
    meeting: null as any, // Avoid circular reference
    you: {
        __typename: "MeetingInviteYou",
        canDelete: false,
        canUpdate: false,
    },
};

/**
 * Minimal meeting API response
 */
export const minimalMeetingResponse: Meeting = {
    __typename: "Meeting",
    id: "meeting_123456789012345",
    publicId: "meet_minimal_123456",
    openToAnyoneWithInvite: false,
    showOnTeamProfile: false,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    team: minimalTeamResponse,
    attendees: [],
    attendeesCount: 0,
    invites: [],
    invitesCount: 0,
    schedule: oneTimeSchedule,
    translations: [
        {
            __typename: "MeetingTranslation",
            id: "meetingtrans_123456789",
            language: "en",
            name: "Team Meeting",
            description: "A simple team meeting",
            link: null,
        },
    ],
    translationsCount: 1,
    you: {
        __typename: "MeetingYou",
        canDelete: false,
        canInvite: false,
        canUpdate: false,
    },
};

/**
 * Complete meeting API response with all fields
 */
export const completeMeetingResponse: Meeting = {
    __typename: "Meeting",
    id: "meeting_987654321098765",
    publicId: "meet_complete_987654",
    openToAnyoneWithInvite: true,
    showOnTeamProfile: true,
    createdAt: "2023-12-01T00:00:00Z",
    updatedAt: "2024-01-20T15:30:00Z",
    team: completeTeamResponse,
    attendees: [minimalUserResponse, completeUserResponse],
    attendeesCount: 2,
    invites: [acceptedInvite, pendingInvite],
    invitesCount: 3,
    schedule: weeklyRecurringSchedule,
    translations: [
        {
            __typename: "MeetingTranslation",
            id: "meetingtrans_987654321",
            language: "en",
            name: "Weekly Sprint Planning",
            description: "Weekly meeting to plan upcoming sprint tasks and review progress",
            link: "https://meet.example.com/sprint-planning",
        },
        {
            __typename: "MeetingTranslation",
            id: "meetingtrans_876543210",
            language: "es",
            name: "Planificación Semanal del Sprint",
            description: "Reunión semanal para planificar las tareas del próximo sprint y revisar el progreso",
            link: "https://meet.example.com/sprint-planning",
        },
    ],
    translationsCount: 2,
    you: {
        __typename: "MeetingYou",
        canDelete: true, // Meeting organizer
        canInvite: true,
        canUpdate: true,
    },
};

/**
 * One-on-one meeting
 */
export const oneOnOneMeetingResponse: Meeting = {
    __typename: "Meeting",
    id: "meeting_1on1_123456789",
    publicId: "meet_1on1_123456",
    openToAnyoneWithInvite: false,
    showOnTeamProfile: false,
    createdAt: "2024-01-15T00:00:00Z",
    updatedAt: "2024-01-15T00:00:00Z",
    team: minimalTeamResponse,
    attendees: [minimalUserResponse],
    attendeesCount: 1,
    invites: [
        {
            ...acceptedInvite,
            id: "invite_1on1_123456789",
            message: "Let's catch up on your progress",
        },
    ],
    invitesCount: 1,
    schedule: {
        ...oneTimeSchedule,
        id: "schedule_1on1_123456789",
        startTime: "2024-03-20T13:00:00Z",
        endTime: "2024-03-20T13:30:00Z",
        publicId: "sched_1on1_123456",
    },
    translations: [
        {
            __typename: "MeetingTranslation",
            id: "meetingtrans_1on1_123",
            language: "en",
            name: "1:1 Check-in",
            description: "Personal check-in and progress discussion",
            link: "https://meet.example.com/1on1-checkin",
        },
    ],
    translationsCount: 1,
    you: {
        __typename: "MeetingYou",
        canDelete: false,
        canInvite: false, // 1:1 meetings don't typically allow additional invites
        canUpdate: false,
    },
};

/**
 * Public event/conference meeting
 */
export const publicEventMeetingResponse: Meeting = {
    __typename: "Meeting",
    id: "meeting_event_123456789",
    publicId: "meet_event_123456",
    openToAnyoneWithInvite: true,
    showOnTeamProfile: true,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-02-15T00:00:00Z",
    team: completeTeamResponse,
    attendees: [minimalUserResponse, completeUserResponse],
    attendeesCount: 150, // Large public event
    invites: [acceptedInvite],
    invitesCount: 200,
    schedule: {
        ...oneTimeSchedule,
        id: "schedule_event_123456789",
        startTime: "2024-04-15T09:00:00Z",
        endTime: "2024-04-15T17:00:00Z",
        publicId: "sched_event_123456",
        user: completeUserResponse,
    },
    translations: [
        {
            __typename: "MeetingTranslation",
            id: "meetingtrans_event_123",
            language: "en",
            name: "AI Development Conference 2024",
            description: "Annual conference showcasing the latest advances in AI development tools and methodologies",
            link: "https://conference.example.com/ai-dev-2024",
        },
    ],
    translationsCount: 1,
    you: {
        __typename: "MeetingYou",
        canDelete: false,
        canInvite: true, // Can invite others to public events
        canUpdate: false,
    },
};

/**
 * All hands meeting
 */
export const allHandsMeetingResponse: Meeting = {
    __typename: "Meeting",
    id: "meeting_allhands_123456",
    publicId: "meet_allhands_123456",
    openToAnyoneWithInvite: false,
    showOnTeamProfile: true,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    team: completeTeamResponse,
    attendees: [minimalUserResponse, completeUserResponse],
    attendeesCount: 25, // All team members
    invites: [acceptedInvite, pendingInvite, declinedInvite],
    invitesCount: 25,
    schedule: {
        ...oneTimeSchedule,
        id: "schedule_allhands_123456",
        startTime: "2024-03-01T16:00:00Z",
        endTime: "2024-03-01T17:00:00Z",
        timezone: "America/Los_Angeles",
        publicId: "sched_allhands_123456",
        user: completeUserResponse,
    },
    translations: [
        {
            __typename: "MeetingTranslation",
            id: "meetingtrans_allhands_123",
            language: "en",
            name: "All Hands Meeting - Q1 2024",
            description: "Quarterly all-hands meeting to discuss company updates, goals, and team achievements",
            link: "https://meet.example.com/allhands-q1-2024",
        },
    ],
    translationsCount: 1,
    you: {
        __typename: "MeetingYou",
        canDelete: false,
        canInvite: false, // HR/management handles invites
        canUpdate: false,
    },
};

/**
 * Recurring team meeting
 */
export const recurringMeetingResponse: Meeting = {
    __typename: "Meeting",
    id: "meeting_recurring_123456",
    publicId: "meet_recurring_123456", 
    openToAnyoneWithInvite: false,
    showOnTeamProfile: true,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-02-01T00:00:00Z",
    team: minimalTeamResponse,
    attendees: [minimalUserResponse, completeUserResponse],
    attendeesCount: 8,
    invites: [acceptedInvite],
    invitesCount: 8,
    schedule: weeklyRecurringSchedule,
    translations: [
        {
            __typename: "MeetingTranslation",
            id: "meetingtrans_recurring_123",
            language: "en",
            name: "Weekly Team Sync",
            description: "Regular weekly sync to align on priorities and blockers",
            link: "https://meet.example.com/weekly-sync",
        },
    ],
    translationsCount: 1,
    you: {
        __typename: "MeetingYou",
        canDelete: true, // Team lead
        canInvite: true,
        canUpdate: true,
    },
};

/**
 * Meeting variant states for testing
 */
export const meetingResponseVariants = {
    minimal: minimalMeetingResponse,
    complete: completeMeetingResponse,
    oneOnOne: oneOnOneMeetingResponse,
    publicEvent: publicEventMeetingResponse,
    allHands: allHandsMeetingResponse,
    recurring: recurringMeetingResponse,
    cancelled: {
        ...minimalMeetingResponse,
        id: "meeting_cancelled_123456",
        publicId: "meet_cancelled_123456",
        translations: [{
            __typename: "MeetingTranslation" as const,
            id: "meetingtrans_cancelled_123",
            language: "en",
            name: "Cancelled Meeting",
            description: "This meeting has been cancelled",
            link: null,
        }],
        you: {
            __typename: "MeetingYou" as const,
            canDelete: true,
            canInvite: false,
            canUpdate: true,
        },
    },
    pastMeeting: {
        ...completeMeetingResponse,
        id: "meeting_past_123456789",
        publicId: "meet_past_123456",
        schedule: {
            ...completeMeetingResponse.schedule!,
            id: "schedule_past_123456789",
            startTime: "2024-01-15T14:00:00Z",
            endTime: "2024-01-15T15:00:00Z",
            publicId: "sched_past_123456",
        },
        translations: [{
            __typename: "MeetingTranslation" as const,
            id: "meetingtrans_past_123",
            language: "en",
            name: "Completed Sprint Review",
            description: "Sprint review meeting that already occurred",
            link: "https://meet.example.com/sprint-review-completed",
        }],
    },
    largeConference: {
        ...publicEventMeetingResponse,
        id: "meeting_conference_123456",
        publicId: "meet_conference_123456",
        attendeesCount: 1000,
        invitesCount: 2500,
        translations: [{
            __typename: "MeetingTranslation" as const,
            id: "meetingtrans_conference_123",
            language: "en",
            name: "Annual Developer Conference",
            description: "Large-scale developer conference with multiple tracks",
            link: "https://conference.example.com/annual-dev-conf",
        }],
    },
} as const;

/**
 * Meeting search response
 */
export const meetingSearchResponse = {
    __typename: "MeetingSearchResult",
    edges: [
        {
            __typename: "MeetingEdge",
            cursor: "cursor_1",
            node: meetingResponseVariants.complete,
        },
        {
            __typename: "MeetingEdge",
            cursor: "cursor_2",
            node: meetingResponseVariants.recurring,
        },
        {
            __typename: "MeetingEdge",
            cursor: "cursor_3",
            node: meetingResponseVariants.publicEvent,
        },
    ],
    pageInfo: {
        __typename: "PageInfo",
        hasNextPage: true,
        hasPreviousPage: false,
        startCursor: "cursor_1",
        endCursor: "cursor_3",
    },
};

/**
 * Meeting invite search response
 */
export const meetingInviteSearchResponse = {
    __typename: "MeetingInviteSearchResult",
    edges: [
        {
            __typename: "MeetingInviteEdge",
            cursor: "cursor_1",
            node: acceptedInvite,
        },
        {
            __typename: "MeetingInviteEdge",
            cursor: "cursor_2", 
            node: pendingInvite,
        },
        {
            __typename: "MeetingInviteEdge",
            cursor: "cursor_3",
            node: declinedInvite,
        },
    ],
    pageInfo: {
        __typename: "PageInfo",
        hasNextPage: false,
        hasPreviousPage: false,
        startCursor: "cursor_1",
        endCursor: "cursor_3",
    },
};

/**
 * Loading and error states for UI testing
 */
export const meetingUIStates = {
    loading: null,
    error: {
        code: "MEETING_NOT_FOUND",
        message: "The requested meeting could not be found",
    },
    accessDenied: {
        code: "MEETING_ACCESS_DENIED",
        message: "You don't have permission to view this meeting",
    },
    inviteError: {
        code: "MEETING_INVITE_FAILED",
        message: "Failed to send meeting invite. Please try again.",
    },
    scheduleConflict: {
        code: "MEETING_SCHEDULE_CONFLICT",
        message: "This meeting conflicts with another scheduled meeting",
    },
    joinError: {
        code: "MEETING_JOIN_FAILED",
        message: "Unable to join the meeting at this time",
    },
    empty: {
        edges: [],
        pageInfo: {
            __typename: "PageInfo",
            hasNextPage: false,
            hasPreviousPage: false,
            startCursor: null,
            endCursor: null,
        },
    },
};