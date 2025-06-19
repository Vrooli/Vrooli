import type { ChatParticipantUpdateInput } from "@vrooli/shared";

/**
 * Form data fixtures for chat participant-related forms
 * These represent data as it appears in form state before submission
 */

/**
 * Add participant to chat form data
 */
export const minimalAddParticipantFormInput = {
    userId: "user_123456789",
};

export const completeAddParticipantFormInput = {
    userId: "user_123456789",
    role: "Member", // "Admin" | "Moderator" | "Member"
    sendNotification: true,
    welcomeMessage: "Welcome to our team chat!",
};

/**
 * Bulk add participants form data
 */
export const bulkAddParticipantsFormInput = {
    participants: [
        { userId: "user_111111111", role: "Member" },
        { userId: "user_222222222", role: "Member" },
        { userId: "user_333333333", role: "Moderator" },
    ],
    sendNotifications: true,
    welcomeMessage: "Welcome everyone to our project discussion!",
};

/**
 * Update participant role form data
 */
export const updateParticipantRoleFormInput = {
    participantId: "123456789012345678",
    newRole: "Moderator", // "Admin" | "Moderator" | "Member"
    reason: "Promoted for excellent contributions",
};

/**
 * Remove participant form data
 */
export const removeParticipantFormInput = {
    participantId: "123456789012345678",
    reason: "User violated chat guidelines",
    notifyUser: true,
    banDuration: null, // null for no ban, number for days
};

/**
 * Participant settings form data
 */
export const participantNotificationSettingsFormInput = {
    muteChat: false,
    notifyOnMention: true,
    notifyOnReply: true,
    notifyOnNewMessage: false,
    soundEnabled: true,
    desktopNotifications: true,
};

export const participantDisplaySettingsFormInput = {
    nickname: "Johnny",
    showOnlineStatus: true,
    showTypingIndicator: true,
    theme: "default", // "default" | "dark" | "light"
};

/**
 * Participant permissions form data
 */
export const updateParticipantPermissionsFormInput = {
    participantId: "123456789012345678",
    permissions: {
        canSendMessages: true,
        canSendMedia: true,
        canAddParticipants: false,
        canRemoveParticipants: false,
        canChangeInfo: false,
        canPinMessages: true,
        canCreatePolls: true,
    },
};

/**
 * Batch update participants form data
 */
export const batchUpdateParticipantsFormInput = {
    participantIds: [
        "123456789012345678",
        "123456789012345679",
        "123456789012345680",
    ],
    action: "mute", // "mute" | "unmute" | "remove" | "promote" | "demote"
    duration: 24, // hours for mute action
    reason: "Temporary mute for spam",
};

/**
 * Participant invite acceptance form data
 */
export const acceptInviteFormInput = {
    inviteId: "invite_123456789",
    displayName: "John Doe",
    acceptTerms: true,
};

/**
 * Form validation states
 */
export const chatParticipantFormValidationStates = {
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            userId: "", // Required but empty
            participantId: "invalid-id", // Invalid format
            newRole: "", // Required but not selected
        },
        errors: {
            userId: "User ID is required",
            participantId: "Invalid participant ID format",
            newRole: "Please select a role",
        },
        touched: {
            userId: true,
            participantId: true,
            newRole: true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: minimalAddParticipantFormInput,
        errors: {},
        touched: {
            userId: true,
        },
        isValid: true,
        isSubmitting: false,
    },
};

/**
 * Helper function to create participant form initial values
 */
export const createParticipantFormInitialValues = (participantData?: Partial<any>) => ({
    userId: participantData?.userId || "",
    participantId: participantData?.participantId || "",
    role: participantData?.role || "Member",
    permissions: participantData?.permissions || {},
    sendNotification: participantData?.sendNotification ?? true,
    ...participantData,
});

/**
 * Helper function to validate participant ID
 */
export const validateParticipantId = (id: string): string | null => {
    if (!id) return "Participant ID is required";
    // Snowflake ID validation (18-19 digits)
    if (!/^\d{18,19}$/.test(id)) {
        return "Invalid participant ID format";
    }
    return null;
};

/**
 * Helper function to validate user selection
 */
export const validateUserSelection = (userId: string): string | null => {
    if (!userId) return "Please select a user";
    if (!userId.startsWith("user_")) return "Invalid user ID format";
    return null;
};

/**
 * Helper function to transform form data to API format
 */
export const transformParticipantFormToApiInput = (formData: any) => ({
    ...formData,
    // Convert role to proper enum value
    role: formData.role || "Member",
    // Ensure permissions object has all required fields
    permissions: {
        canSendMessages: true,
        canSendMedia: true,
        canAddParticipants: false,
        canRemoveParticipants: false,
        canChangeInfo: false,
        canPinMessages: false,
        canCreatePolls: false,
        ...formData.permissions,
    },
    // Convert notification settings
    notificationSettings: formData.notificationSettings ? {
        muteChat: formData.notificationSettings.muteChat || false,
        notifyOnMention: formData.notificationSettings.notifyOnMention ?? true,
        notifyOnReply: formData.notificationSettings.notifyOnReply ?? true,
        notifyOnNewMessage: formData.notificationSettings.notifyOnNewMessage || false,
    } : undefined,
});

/**
 * Mock participant suggestions
 */
export const mockParticipantSuggestions = {
    users: [
        { id: "user_123456789", handle: "johndoe", name: "John Doe", avatar: null },
        { id: "user_987654321", handle: "janedoe", name: "Jane Doe", avatar: null },
        { id: "user_111222333", handle: "bobsmith", name: "Bob Smith", avatar: null },
        { id: "user_444555666", handle: "alicejones", name: "Alice Jones", avatar: null },
    ],
    roles: [
        { value: "Admin", label: "Admin", description: "Full control over the chat" },
        { value: "Moderator", label: "Moderator", description: "Can manage messages and participants" },
        { value: "Member", label: "Member", description: "Can send messages and view chat" },
    ],
};

/**
 * Mock participant states
 */
export const mockParticipantStates = {
    online: {
        userId: "user_123456789",
        status: "online",
        lastSeen: new Date().toISOString(),
    },
    offline: {
        userId: "user_987654321",
        status: "offline",
        lastSeen: new Date(Date.now() - 3600000).toISOString(), // 1 hour ago
    },
    typing: {
        userId: "user_111222333",
        status: "online",
        isTyping: true,
        lastSeen: new Date().toISOString(),
    },
    away: {
        userId: "user_444555666",
        status: "away",
        lastSeen: new Date(Date.now() - 900000).toISOString(), // 15 minutes ago
    },
};

/**
 * Participant action responses
 */
export const mockParticipantActionResponses = {
    added: {
        success: true,
        message: "Participant added successfully",
        participant: {
            id: "123456789012345678",
            userId: "user_123456789",
            chatId: "chat_987654321",
            role: "Member",
            joinedAt: new Date().toISOString(),
        },
    },
    removed: {
        success: true,
        message: "Participant removed from chat",
    },
    roleUpdated: {
        success: true,
        message: "Role updated successfully",
        participant: {
            id: "123456789012345678",
            role: "Moderator",
        },
    },
    permissionsUpdated: {
        success: true,
        message: "Permissions updated successfully",
    },
};