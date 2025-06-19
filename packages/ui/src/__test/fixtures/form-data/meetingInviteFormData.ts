import type { MeetingInviteCreateInput, MeetingInviteUpdateInput } from "@vrooli/shared";

/**
 * Form data fixtures for meeting invite-related forms
 * These represent data as it appears in form state before submission
 */

/**
 * Meeting invite creation form data
 */
export const minimalMeetingInviteCreateFormInput: Partial<MeetingInviteCreateInput> = {
    meetingId: "123456789012345678",
    userId: "123456789012345679",
};

export const completeMeetingInviteCreateFormInput = {
    meetingId: "123456789012345678",
    userId: "123456789012345679",
    message: "You're invited to our weekly team meeting. Please join us to discuss the upcoming project milestones.",
};

/**
 * Bulk meeting invite creation form data
 */
export const bulkMeetingInviteCreateFormInput = {
    meetingId: "123456789012345678",
    invites: [
        {
            userId: "123456789012345679",
            message: "Join us for our weekly sync meeting",
        },
        {
            userId: "123456789012345680",
            message: "Your expertise would be valuable in our planning session",
        },
        {
            userId: "123456789012345681",
            message: "Please attend our project kickoff meeting",
        },
    ],
};

/**
 * Meeting invite by email form data
 */
export const meetingInviteByEmailFormInput = {
    meetingId: "123456789012345678",
    emails: [
        "external.partner@example.com",
        "client@company.com",
        "consultant@agency.com",
    ],
    message: "You're invited to join our meeting. Click the link below to join.",
    sendEmail: true,
    includeCalendarInvite: true,
};

/**
 * Meeting invite with RSVP form data
 */
export const meetingInviteWithRsvpFormInput = {
    meetingId: "123456789012345678",
    userId: "123456789012345679",
    message: "Please RSVP for our upcoming quarterly review meeting",
    requireRsvp: true,
    rsvpDeadline: new Date(Date.now() + 3 * 24 * 60 * 60 * 1000).toISOString(), // 3 days from now
};

/**
 * Meeting invite update form data
 */
export const meetingInviteUpdateFormInput = {
    message: "Updated invitation message with new meeting details and agenda items.",
};

/**
 * Meeting invite response form data
 */
export const meetingInviteAcceptFormInput = {
    inviteId: "123456789012345678",
    response: "accepted",
    message: "Looking forward to it!",
    addToCalendar: true,
};

export const meetingInviteDeclineFormInput = {
    inviteId: "123456789012345678",
    response: "declined",
    message: "Sorry, I have a conflict at that time",
    suggestAlternative: true,
    alternativeTimes: [
        { date: "2024-01-16", time: "10:00" },
        { date: "2024-01-17", time: "14:00" },
    ],
};

export const meetingInviteTentativeFormInput = {
    inviteId: "123456789012345678",
    response: "tentative",
    message: "I'll confirm by end of day",
    notifyWhenDecided: true,
};

/**
 * Meeting invite with role assignment form data
 */
export const meetingInviteWithRoleFormInput = {
    meetingId: "123456789012345678",
    userId: "123456789012345679",
    message: "You're invited as a presenter for our technical review",
    role: "Presenter", // "Attendee" | "Presenter" | "Co-host" | "Observer"
    permissions: {
        canPresent: true,
        canRecord: false,
        canInviteOthers: false,
        canModerate: false,
    },
};

/**
 * Recurring meeting invite form data
 */
export const recurringMeetingInviteFormInput = {
    meetingId: "123456789012345678",
    userId: "123456789012345679",
    message: "You're invited to our weekly standup meetings",
    applyToAllOccurrences: true,
    exceptions: [
        { date: "2024-01-22", skip: true, reason: "National Holiday" },
        { date: "2024-02-14", skip: true, reason: "Team Offsite" },
    ],
};

/**
 * Form validation states
 */
export const meetingInviteFormValidationStates = {
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            meetingId: "", // Required but empty
            userId: "invalid-id", // Invalid format
            message: "A".repeat(4097), // Too long (max 4096)
        },
        errors: {
            meetingId: "Please select a meeting",
            userId: "Invalid user ID format",
            message: "Message is too long (max 4096 characters)",
        },
        touched: {
            meetingId: true,
            userId: true,
            message: true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: completeMeetingInviteCreateFormInput,
        errors: {},
        touched: {
            meetingId: true,
            userId: true,
            message: true,
        },
        isValid: true,
        isSubmitting: false,
    },
};

/**
 * Helper function to create meeting invite form initial values
 */
export const createMeetingInviteFormInitialValues = (inviteData?: Partial<any>) => ({
    meetingId: inviteData?.meetingId || "",
    userId: inviteData?.userId || "",
    message: inviteData?.message || "",
    requireRsvp: inviteData?.requireRsvp || false,
    role: inviteData?.role || "Attendee",
    ...inviteData,
});

/**
 * Helper function to validate invite message
 */
export const validateMeetingInviteMessage = (message: string): string | null => {
    if (message && message.length > 4096) {
        return "Message is too long (max 4096 characters)";
    }
    return null;
};

/**
 * Helper function to validate user selection
 */
export const validateUserSelection = (userId: string): string | null => {
    if (!userId) return "Please select a user to invite";
    if (!/^\d{18,19}$/.test(userId)) return "Invalid user ID format";
    return null;
};

/**
 * Helper function to validate meeting selection
 */
export const validateMeetingSelection = (meetingId: string): string | null => {
    if (!meetingId) return "Please select a meeting";
    if (!/^\d{18,19}$/.test(meetingId)) return "Invalid meeting ID format";
    return null;
};

/**
 * Helper function to validate email list
 */
export const validateEmailList = (emails: string[]): string | null => {
    if (!emails || emails.length === 0) return "Please enter at least one email";
    
    const invalidEmails = emails.filter(email => 
        !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)
    );
    
    if (invalidEmails.length > 0) {
        return `Invalid email format: ${invalidEmails.join(", ")}`;
    }
    
    if (emails.length > 100) return "Cannot invite more than 100 users at once";
    
    return null;
};

/**
 * Helper function to validate RSVP deadline
 */
export const validateRsvpDeadline = (deadline: string, meetingStart: string): string | null => {
    if (!deadline) return null; // Optional field
    
    const deadlineDate = new Date(deadline);
    const meetingDate = new Date(meetingStart);
    
    if (deadlineDate >= meetingDate) {
        return "RSVP deadline must be before the meeting starts";
    }
    
    if (deadlineDate <= new Date()) {
        return "RSVP deadline must be in the future";
    }
    
    return null;
};

/**
 * Helper function to transform form data to API format
 */
export const transformMeetingInviteFormToApiInput = (formData: any) => ({
    id: formData.id || formData.inviteId,
    message: formData.message?.trim() || undefined,
    meetingConnect: formData.meetingId,
    userConnect: formData.userId,
    // Handle bulk invites
    invites: formData.invites?.map((invite: any) => ({
        userConnect: invite.userId,
        message: invite.message?.trim() || undefined,
    })) || undefined,
    // Handle email invites
    emailInvites: formData.emails?.map((email: string) => ({
        email: email.trim(),
        sendCalendarInvite: formData.includeCalendarInvite || false,
    })) || undefined,
    // Handle role and permissions
    role: formData.role || undefined,
    permissions: formData.permissions || undefined,
    // Handle RSVP settings
    requireRsvp: formData.requireRsvp || undefined,
    rsvpDeadline: formData.rsvpDeadline || undefined,
});

/**
 * Mock user suggestions for invite form
 */
export const mockInviteUserSuggestions = [
    { id: "123456789012345679", handle: "johndoe", name: "John Doe", avatar: null },
    { id: "123456789012345680", handle: "janedoe", name: "Jane Doe", avatar: null },
    { id: "123456789012345681", handle: "bobsmith", name: "Bob Smith", avatar: null },
    { id: "123456789012345682", handle: "alice", name: "Alice Johnson", avatar: null },
    { id: "123456789012345683", handle: "charlie", name: "Charlie Brown", avatar: null },
];

/**
 * Mock meeting suggestions for invite form
 */
export const mockInviteMeetingSuggestions = [
    { id: "123456789012345678", name: "Weekly Team Sync", date: "2024-01-15", time: "10:00", participantCount: 12 },
    { id: "123456789012345684", name: "Project Kickoff", date: "2024-01-16", time: "14:00", participantCount: 8 },
    { id: "123456789012345685", name: "Monthly All Hands", date: "2024-01-31", time: "15:00", participantCount: 50 },
    { id: "123456789012345686", name: "Design Review", date: "2024-01-18", time: "11:00", participantCount: 6 },
];

/**
 * Mock invite templates
 */
export const mockMeetingInviteTemplates = {
    formal: "You are cordially invited to attend our upcoming meeting. Your participation and insights would be greatly valued.",
    casual: "Hey! We're having a meeting and would love for you to join us.",
    reminder: "This is a friendly reminder about our upcoming meeting. Looking forward to seeing you there!",
    urgent: "Your attendance is requested for an important meeting. Please confirm your availability.",
    recurring: "You're invited to join our regular meeting series. This invite covers all upcoming sessions.",
    custom: "",
};

/**
 * Mock invite status states
 */
export const mockMeetingInviteStates = {
    pending: {
        status: "pending",
        sentAt: new Date().toISOString(),
        reminderSentAt: null,
    },
    accepted: {
        status: "accepted",
        sentAt: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString(),
        acceptedAt: new Date(Date.now() - 1 * 24 * 60 * 60 * 1000).toISOString(),
        addedToCalendar: true,
    },
    declined: {
        status: "declined",
        sentAt: new Date(Date.now() - 3 * 24 * 60 * 60 * 1000).toISOString(),
        declinedAt: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString(),
        reason: "Schedule conflict",
    },
    tentative: {
        status: "tentative",
        sentAt: new Date(Date.now() - 1 * 24 * 60 * 60 * 1000).toISOString(),
        respondedAt: new Date(Date.now() - 12 * 60 * 60 * 1000).toISOString(),
        note: "Will confirm tomorrow",
    },
};

/**
 * Mock meeting roles with permissions
 */
export const mockMeetingRoles = {
    host: {
        role: "Host",
        permissions: {
            canPresent: true,
            canRecord: true,
            canInviteOthers: true,
            canModerate: true,
            canEndMeeting: true,
            canManageParticipants: true,
        },
    },
    coHost: {
        role: "Co-host",
        permissions: {
            canPresent: true,
            canRecord: true,
            canInviteOthers: true,
            canModerate: true,
            canEndMeeting: false,
            canManageParticipants: true,
        },
    },
    presenter: {
        role: "Presenter",
        permissions: {
            canPresent: true,
            canRecord: false,
            canInviteOthers: false,
            canModerate: false,
            canEndMeeting: false,
            canManageParticipants: false,
        },
    },
    attendee: {
        role: "Attendee",
        permissions: {
            canPresent: false,
            canRecord: false,
            canInviteOthers: false,
            canModerate: false,
            canEndMeeting: false,
            canManageParticipants: false,
        },
    },
    observer: {
        role: "Observer",
        permissions: {
            canPresent: false,
            canRecord: false,
            canInviteOthers: false,
            canModerate: false,
            canEndMeeting: false,
            canManageParticipants: false,
            canChat: false,
            canReact: false,
        },
    },
};