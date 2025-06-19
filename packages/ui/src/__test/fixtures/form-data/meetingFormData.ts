import type { MeetingCreateInput, MeetingUpdateInput } from "@vrooli/shared";

/**
 * Form data fixtures for meeting-related forms
 * These represent data as it appears in form state before submission
 */

/**
 * Meeting creation form data
 */
export const minimalMeetingCreateFormInput: Partial<MeetingCreateInput> = {
    name: "Weekly Team Meeting",
    team: "team_123456789",
    openToAnyoneWithInvite: false,
};

export const completeMeetingCreateFormInput = {
    name: "Weekly Team Meeting",
    description: "Our regular weekly synchronization meeting to discuss progress and planning",
    team: "team_123456789",
    link: "https://example.com/meeting",
    openToAnyoneWithInvite: true,
    showOnTeamProfile: false,
    startTime: "2024-01-15T10:00:00Z",
    endTime: "2024-01-15T11:00:00Z",
    location: "Conference Room A",
    locationLink: "https://maps.example.com/conf-room-a",
    participants: [
        "user_123456789",
        "user_987654321",
        "user_111222333",
    ],
    invitees: [
        { userId: "user_444555666", message: "You're invited to our weekly team meeting" },
        { userId: "user_777888999", message: "Please join us for planning discussion" },
    ],
    tags: ["weekly", "planning", "team"],
};

/**
 * Meeting update form data
 */
export const minimalMeetingUpdateFormInput = {
    name: "Updated Weekly Meeting",
    description: "Updated description for our weekly meeting",
};

export const completeMeetingUpdateFormInput = {
    name: "Updated Weekly Team Meeting",
    description: "Updated description for our weekly meeting with new agenda items",
    link: "https://example.com/updated-meeting",
    openToAnyoneWithInvite: false,
    showOnTeamProfile: true,
    startTime: "2024-01-15T11:00:00Z",
    endTime: "2024-01-15T12:00:00Z",
    location: "Virtual - Zoom",
    locationLink: "https://zoom.us/j/123456789",
    tags: ["weekly", "virtual", "all-hands"],
};

/**
 * Meeting scheduling form data
 */
export const oneTimeMeetingScheduleFormInput = {
    recurrenceType: "Once",
    startDate: "2024-01-15",
    startTime: "10:00",
    endTime: "11:00",
    timezone: "America/New_York",
};

export const recurringMeetingScheduleFormInput = {
    recurrenceType: "Weekly",
    startDate: "2024-01-15",
    endDate: "2024-12-31",
    startTime: "10:00",
    endTime: "11:00",
    timezone: "America/New_York",
    daysOfWeek: [1, 3, 5], // Monday, Wednesday, Friday
    exceptions: [
        { date: "2024-01-22", reason: "National Holiday" },
        { date: "2024-02-14", reason: "Team Offsite" },
    ],
};

export const dailyMeetingScheduleFormInput = {
    recurrenceType: "Daily",
    startDate: "2024-01-15",
    endDate: "2024-01-31",
    startTime: "09:00",
    endTime: "09:15",
    timezone: "America/New_York",
    excludeWeekends: true,
};

export const monthlyMeetingScheduleFormInput = {
    recurrenceType: "Monthly",
    startDate: "2024-01-15",
    endDate: "2024-12-31",
    startTime: "14:00",
    endTime: "15:30",
    timezone: "America/New_York",
    dayOfMonth: 15, // 15th of each month
    onWeekend: "nextBusinessDay", // "skip" | "nextBusinessDay" | "previousBusinessDay"
};

/**
 * Meeting invite form data
 */
export const singleInviteFormInput = {
    userId: "user_123456789",
    message: "We'd love to have you join our weekly planning meeting",
    sendEmail: true,
};

export const bulkInviteFormInput = {
    invites: [
        {
            userId: "user_111111111",
            message: "Join us for the weekly sync",
        },
        {
            userId: "user_222222222",
            message: "Your expertise would be valuable",
        },
        {
            userId: "user_333333333",
            message: "Please attend for project updates",
        },
    ],
    sendEmails: true,
    includeCalendarInvite: true,
};

export const emailInviteFormInput = {
    emails: [
        "external1@example.com",
        "external2@example.com",
        "partner@company.com",
    ],
    message: "You're invited to join our project kickoff meeting",
    includeJoinLink: true,
    requireRSVP: true,
};

/**
 * Meeting participant management form data
 */
export const addParticipantFormInput = {
    userId: "user_new_123456",
    role: "Attendee", // "Host" | "Co-host" | "Presenter" | "Attendee"
    canPresent: false,
    canRecord: false,
};

export const updateParticipantRoleFormInput = {
    participantId: "participant_123456",
    newRole: "Presenter",
    canPresent: true,
    canRecord: true,
};

export const removeParticipantFormInput = {
    participantId: "participant_123456",
    reason: "No longer part of the project",
    notifyParticipant: true,
};

/**
 * Meeting settings form data
 */
export const meetingGeneralSettingsFormInput = {
    name: "Team Weekly Sync",
    description: "Weekly team synchronization meeting",
    isPrivate: false,
    requirePassword: false,
    password: "",
    allowGuestAccess: true,
    autoRecord: false,
    waitingRoomEnabled: true,
};

export const meetingNotificationSettingsFormInput = {
    reminders: [
        { time: 15, unit: "minutes" }, // 15 minutes before
        { time: 1, unit: "day" }, // 1 day before
    ],
    notifyOnJoin: true,
    notifyOnLeave: true,
    notifyOnCancel: true,
    notifyOnReschedule: true,
};

export const meetingAccessSettingsFormInput = {
    requireAuthentication: true,
    allowedDomains: ["company.com", "partner.com"],
    maxParticipants: 100,
    allowJoinBeforeHost: false,
    muteOnEntry: true,
    disableVideo: false,
    allowScreenShare: true,
    allowChat: true,
    allowReactions: true,
};

/**
 * Meeting RSVP form data
 */
export const rsvpAcceptFormInput = {
    response: "yes", // "yes" | "no" | "maybe"
    message: "Looking forward to it!",
    addToCalendar: true,
};

export const rsvpDeclineFormInput = {
    response: "no",
    message: "Sorry, I have a conflict at that time",
    suggestAlternative: true,
    alternativeTimes: [
        { date: "2024-01-16", time: "10:00" },
        { date: "2024-01-16", time: "14:00" },
        { date: "2024-01-17", time: "09:00" },
    ],
};

export const rsvpMaybeFormInput = {
    response: "maybe",
    message: "I'll confirm by end of day",
    notifyWhenDecided: true,
};

/**
 * Meeting cancel/reschedule form data
 */
export const cancelMeetingFormInput = {
    reason: "Due to unexpected circumstances, we need to cancel this meeting",
    notifyAllParticipants: true,
    suggestReschedule: true,
    proposedDates: [
        { date: "2024-01-16", time: "10:00" },
        { date: "2024-01-17", time: "14:00" },
    ],
};

export const rescheduleMeetingFormInput = {
    newDate: "2024-01-16",
    newStartTime: "11:00",
    newEndTime: "12:00",
    reason: "Conflict with another important meeting",
    notifyAllParticipants: true,
    requestNewRSVP: true,
};

/**
 * Form validation states
 */
export const meetingFormValidationStates = {
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            name: "AB", // Too short (min 3 chars)
            team: "", // Required but empty
            link: "not-a-valid-url", // Invalid URL format
            startTime: "2024-01-15T14:00:00Z",
            endTime: "2024-01-15T13:00:00Z", // End before start
        },
        errors: {
            name: "Meeting name must be at least 3 characters",
            team: "Please select a team",
            link: "Please enter a valid URL",
            endTime: "End time must be after start time",
        },
        touched: {
            name: true,
            team: true,
            link: true,
            startTime: true,
            endTime: true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: minimalMeetingCreateFormInput,
        errors: {},
        touched: {
            name: true,
            team: true,
            openToAnyoneWithInvite: true,
        },
        isValid: true,
        isSubmitting: false,
    },
};

/**
 * Helper function to create meeting form initial values
 */
export const createMeetingFormInitialValues = (meetingData?: Partial<any>) => ({
    name: meetingData?.name || "",
    description: meetingData?.description || "",
    team: meetingData?.team || null,
    link: meetingData?.link || "",
    openToAnyoneWithInvite: meetingData?.openToAnyoneWithInvite || false,
    showOnTeamProfile: meetingData?.showOnTeamProfile || false,
    startTime: meetingData?.startTime || null,
    endTime: meetingData?.endTime || null,
    location: meetingData?.location || "",
    locationLink: meetingData?.locationLink || "",
    participants: meetingData?.participants || [],
    tags: meetingData?.tags || [],
    ...meetingData,
});

/**
 * Helper function to validate meeting time
 */
export const validateMeetingTime = (startTime: string, endTime: string): string | null => {
    if (!startTime || !endTime) return null;
    
    const start = new Date(startTime);
    const end = new Date(endTime);
    
    if (end <= start) return "End time must be after start time";
    
    const duration = end.getTime() - start.getTime();
    const maxDuration = 8 * 60 * 60 * 1000; // 8 hours
    
    if (duration > maxDuration) return "Meeting cannot be longer than 8 hours";
    
    return null;
};

/**
 * Helper function to validate meeting link
 */
export const validateMeetingLink = (link: string): string | null => {
    if (!link) return null; // Link is optional
    
    try {
        new URL(link);
        return null;
    } catch {
        return "Please enter a valid URL";
    }
};

/**
 * Helper function to transform form data to API format
 */
export const transformMeetingFormToApiInput = (formData: any) => ({
    ...formData,
    // Convert team selection to proper connect format
    teamConnect: formData.team || undefined,
    // Convert participant IDs to invite creates
    invitesCreate: formData.invitees?.map((invite: any) => ({
        userConnect: invite.userId,
        message: invite.message,
    })) || [],
    // Convert tags to proper format
    tags: formData.tags?.map((tag: string) => ({ tag })) || [],
    // Remove form-specific fields
    team: undefined,
    invitees: undefined,
    participants: undefined,
});

/**
 * Mock autocomplete suggestions
 */
export const mockMeetingSuggestions = {
    participants: [
        { id: "user_123", handle: "johndoe", name: "John Doe" },
        { id: "user_456", handle: "janedoe", name: "Jane Doe" },
        { id: "user_789", handle: "bobsmith", name: "Bob Smith" },
    ],
    locations: [
        "Conference Room A",
        "Conference Room B",
        "Virtual - Zoom",
        "Virtual - Google Meet",
        "Cafeteria",
        "Executive Board Room",
    ],
    tags: [
        { tag: "weekly", count: 234 },
        { tag: "planning", count: 156 },
        { tag: "standup", count: 145 },
        { tag: "review", count: 98 },
        { tag: "all-hands", count: 87 },
        { tag: "one-on-one", count: 76 },
    ],
    timezones: [
        { value: "America/New_York", label: "Eastern Time (ET)" },
        { value: "America/Chicago", label: "Central Time (CT)" },
        { value: "America/Denver", label: "Mountain Time (MT)" },
        { value: "America/Los_Angeles", label: "Pacific Time (PT)" },
        { value: "Europe/London", label: "British Time (GMT)" },
        { value: "Europe/Paris", label: "Central European Time (CET)" },
    ],
};