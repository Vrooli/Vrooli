import type { ChatInviteCreateInput, ChatInviteUpdateInput } from "@vrooli/shared";

/**
 * Form data fixtures for chat invite-related forms
 * These represent data as it appears in form state before submission
 */

/**
 * Chat invite creation form data
 */
export const minimalChatInviteCreateFormInput: Partial<ChatInviteCreateInput> = {
    chatId: "123456789012345681",
    userId: "123456789012345683",
};

export const completeChatInviteCreateFormInput = {
    chatId: "123456789012345681",
    userId: "123456789012345683",
    message: "You've been invited to join our project discussion!",
};

/**
 * Bulk chat invite creation form data
 */
export const bulkChatInviteCreateFormInput = {
    chatId: "123456789012345681",
    invites: [
        {
            userId: "123456789012345683",
            message: "Welcome to the team chat!",
        },
        {
            userId: "123456789012345684",
            message: "Join us for project discussions",
        },
        {
            userId: "123456789012345685",
            message: "Looking forward to collaborating with you",
        },
    ],
};

/**
 * Chat invite by email form data
 */
export const chatInviteByEmailFormInput = {
    chatId: "123456789012345681",
    emails: [
        "newuser1@example.com",
        "newuser2@example.com",
        "colleague@example.com",
    ],
    message: "You're invited to join our chat. Click the link to get started!",
    sendEmail: true,
};

/**
 * Chat invite link generation form data
 */
export const chatInviteLinkFormInput = {
    chatId: "123456789012345681",
    expiresIn: 7, // days
    maxUses: 10,
    message: "Join our community chat!",
};

/**
 * Chat invite update form data
 */
export const chatInviteUpdateFormInput = {
    message: "Updated invitation - Join our amazing project team!",
};

/**
 * Chat invite acceptance form data
 */
export const chatInviteAcceptFormInput = {
    inviteId: "123456789012345678",
    joinAsGuest: false,
};

/**
 * Chat invite with custom permissions form data
 */
export const chatInviteWithPermissionsFormInput = {
    chatId: "123456789012345681",
    userId: "123456789012345683",
    message: "You're invited as a moderator",
    permissions: {
        canSendMessages: true,
        canInviteOthers: true,
        canModerate: true,
        canManageChat: false,
    },
    role: "Moderator",
};

/**
 * Form validation states
 */
export const chatInviteFormValidationStates = {
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            chatId: "", // Required but empty
            userId: "invalid-id", // Invalid format
            message: "a".repeat(1001), // Too long
        },
        errors: {
            chatId: "Chat selection is required",
            userId: "Invalid user ID format",
            message: "Message is too long (max 1000 characters)",
        },
        touched: {
            chatId: true,
            userId: true,
            message: true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: completeChatInviteCreateFormInput,
        errors: {},
        touched: {
            chatId: true,
            userId: true,
            message: true,
        },
        isValid: true,
        isSubmitting: false,
    },
};

/**
 * Helper function to create chat invite form initial values
 */
export const createChatInviteFormInitialValues = (inviteData?: Partial<any>) => ({
    chatId: inviteData?.chatId || "",
    userId: inviteData?.userId || "",
    message: inviteData?.message || "",
    ...inviteData,
});

/**
 * Helper function to validate invite message
 */
export const validateInviteMessage = (message: string): string | null => {
    if (message && message.length > 1000) {
        return "Message is too long (max 1000 characters)";
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
    
    if (emails.length > 50) return "Cannot invite more than 50 users at once";
    
    return null;
};

/**
 * Helper function to transform form data to API format
 */
export const transformChatInviteFormToApiInput = (formData: any) => ({
    id: formData.id || formData.inviteId,
    message: formData.message?.trim() || undefined,
    chatConnect: formData.chatId,
    userConnect: formData.userId,
    // Handle bulk invites
    invites: formData.invites?.map((invite: any) => ({
        userConnect: invite.userId,
        message: invite.message?.trim() || undefined,
    })) || undefined,
    // Handle email invites
    emailInvites: formData.emails?.map((email: string) => ({
        email: email.trim(),
    })) || undefined,
});

/**
 * Mock user suggestions for invite form
 */
export const mockInviteUserSuggestions = [
    { id: "123456789012345683", handle: "johndoe", name: "John Doe", avatar: null },
    { id: "123456789012345684", handle: "janedoe", name: "Jane Doe", avatar: null },
    { id: "123456789012345685", handle: "bobsmith", name: "Bob Smith", avatar: null },
    { id: "123456789012345686", handle: "alice", name: "Alice Johnson", avatar: null },
    { id: "123456789012345687", handle: "charlie", name: "Charlie Brown", avatar: null },
];

/**
 * Mock chat suggestions for invite form
 */
export const mockInviteChatSuggestions = [
    { id: "123456789012345681", name: "General Discussion", memberCount: 25 },
    { id: "123456789012345682", name: "Project Team", memberCount: 8 },
    { id: "123456789012345690", name: "Design Reviews", memberCount: 12 },
    { id: "123456789012345691", name: "Development Chat", memberCount: 15 },
];

/**
 * Mock invite templates
 */
export const mockInviteTemplates = {
    welcome: "Welcome to our chat! We're excited to have you join our discussions.",
    project: "You've been invited to join the project team chat. Let's build something amazing together!",
    community: "Join our community chat to connect with like-minded individuals and share ideas.",
    team: "You're invited to our team chat where we collaborate and stay connected.",
    custom: "",
};

/**
 * Mock invite status states
 */
export const mockInviteStates = {
    pending: {
        status: "pending",
        sentAt: new Date().toISOString(),
        expiresAt: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
    },
    accepted: {
        status: "accepted",
        sentAt: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString(),
        acceptedAt: new Date(Date.now() - 1 * 24 * 60 * 60 * 1000).toISOString(),
    },
    expired: {
        status: "expired",
        sentAt: new Date(Date.now() - 10 * 24 * 60 * 60 * 1000).toISOString(),
        expiresAt: new Date(Date.now() - 3 * 24 * 60 * 60 * 1000).toISOString(),
    },
    declined: {
        status: "declined",
        sentAt: new Date(Date.now() - 5 * 24 * 60 * 60 * 1000).toISOString(),
        declinedAt: new Date(Date.now() - 4 * 24 * 60 * 60 * 1000).toISOString(),
    },
};