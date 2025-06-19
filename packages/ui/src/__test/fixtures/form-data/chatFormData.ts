import type { ChatCreateInput, ChatUpdateInput } from "@vrooli/shared";

/**
 * Form data fixtures for chat-related forms
 * These represent data as it appears in form state before submission
 */

/**
 * Chat creation form data
 */
export const minimalChatCreateFormInput: Partial<ChatCreateInput> = {
    name: "General Discussion",
    participants: ["user_123456789", "user_987654321"],
};

export const completeChatCreateFormInput = {
    name: "Project Team Chat",
    description: "Discussion channel for the AI Assistant project team",
    participants: [
        "user_123456789",
        "user_987654321",
        "user_111222333",
    ],
    team: "team_123456789", // Team ID if team chat
    openToAnyoneWithInvite: true,
    restrictedToRoles: [], // Role IDs for team chats
};

/**
 * Direct message form data
 */
export const directMessageFormInput = {
    recipient: "user_987654321",
    message: "Hey, I have a question about the project",
};

/**
 * Group chat creation form data
 */
export const groupChatCreateFormInput = {
    name: "Development Team",
    participants: [
        { userId: "user_123456789", role: "Admin" },
        { userId: "user_987654321", role: "Member" },
        { userId: "user_111222333", role: "Member" },
        { userId: "user_444555666", role: "Member" },
    ],
    isPrivate: true,
    allowInvites: true,
};

/**
 * Chat message form data
 */
export const textMessageFormInput = {
    text: "Hello everyone! How's the project going?",
    replyTo: null,
};

export const replyMessageFormInput = {
    text: "It's going great! We just finished the authentication module.",
    replyTo: "message_123456789", // Parent message ID
};

export const messageWithMentionsFormInput = {
    text: "Hey @johndoe and @janedoe, can you review this PR?",
    mentions: ["user_123456789", "user_987654321"],
};

export const messageWithAttachmentsFormInput = {
    text: "Here are the design mockups for review",
    attachments: [
        {
            file: null, // File object
            type: "image/png",
            name: "dashboard-mockup.png",
            size: 234567,
        },
        {
            file: null, // File object
            type: "application/pdf",
            name: "specifications.pdf",
            size: 1234567,
        },
    ],
};

/**
 * Chat update form data
 */
export const chatUpdateFormInput = {
    name: "Updated Chat Name",
    description: "Updated description for the chat",
    openToAnyoneWithInvite: false,
};

/**
 * Chat participant management form data
 */
export const addParticipantFormInput = {
    userId: "user_new_123456",
    role: "Member",
    sendNotification: true,
};

export const bulkAddParticipantsFormInput = {
    participants: [
        { userId: "user_111", role: "Member" },
        { userId: "user_222", role: "Member" },
        { userId: "user_333", role: "Moderator" },
    ],
    message: "Welcome to our project chat!",
    sendNotifications: true,
};

export const updateParticipantRoleFormInput = {
    participantId: "participant_123456",
    newRole: "Moderator", // "Admin" | "Moderator" | "Member"
};

export const removeParticipantFormInput = {
    participantId: "participant_123456",
    reason: "User requested to leave",
    notifyUser: true,
};

/**
 * Chat settings form data
 */
export const chatGeneralSettingsFormInput = {
    name: "Team Chat",
    description: "Main communication channel",
    allowGuestAccess: false,
    requireApproval: true,
    messageRetention: 90, // days
};

export const chatNotificationSettingsFormInput = {
    muteNotifications: false,
    notifyOnMention: true,
    notifyOnReply: true,
    notifyOnNewMessage: false,
    soundEnabled: true,
    desktopNotifications: true,
};

export const chatPrivacySettingsFormInput = {
    isPrivate: true,
    allowInvites: true,
    restrictToTeam: true,
    restrictedRoles: ["Admin", "Moderator"],
    messageHistory: "all", // "all" | "join" | "none"
};

/**
 * Chat invite form data
 */
export const createInviteLinkFormInput = {
    expiresIn: 7, // days
    maxUses: 10,
    requireApproval: false,
};

export const sendInviteFormInput = {
    emails: [
        "friend1@example.com",
        "friend2@example.com",
        "colleague@example.com",
    ],
    message: "Join our discussion about the new project!",
    role: "Member",
};

/**
 * Chat moderation form data
 */
export const reportMessageFormInput = {
    messageId: "message_123456789",
    reason: "spam", // "spam" | "harassment" | "inappropriate" | "other"
    details: "This message contains promotional content",
};

export const moderateMessageFormInput = {
    messageId: "message_123456789",
    action: "delete", // "delete" | "edit" | "hide"
    reason: "Violates community guidelines",
    notifyUser: true,
};

/**
 * Form validation states
 */
export const chatFormValidationStates = {
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            name: "", // Required for group chats
            participants: [], // Must have at least 1 participant
            text: "   ", // Message can't be just whitespace
        },
        errors: {
            name: "Chat name is required",
            participants: "Please add at least one participant",
            text: "Message cannot be empty",
        },
        touched: {
            name: true,
            participants: true,
            text: true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: textMessageFormInput,
        errors: {},
        touched: {
            text: true,
        },
        isValid: true,
        isSubmitting: false,
    },
};

/**
 * Helper function to create chat form initial values
 */
export const createChatFormInitialValues = (chatData?: Partial<any>) => ({
    name: chatData?.name || "",
    description: chatData?.description || "",
    participants: chatData?.participants || [],
    team: chatData?.team || null,
    openToAnyoneWithInvite: chatData?.openToAnyoneWithInvite || false,
    ...chatData,
});

/**
 * Helper function to validate message content
 */
export const validateMessageContent = (message: string): string | null => {
    if (!message || !message.trim()) return "Message cannot be empty";
    if (message.length > 5000) return "Message is too long (max 5000 characters)";
    return null;
};

/**
 * Helper function to extract mentions from message
 */
export const extractMentions = (text: string): string[] => {
    const mentionRegex = /@(\w+)/g;
    const mentions = [];
    let match;
    
    while ((match = mentionRegex.exec(text)) !== null) {
        mentions.push(match[1]);
    }
    
    return mentions;
};

/**
 * Helper function to transform form data to API format
 */
export const transformChatFormToApiInput = (formData: any) => ({
    ...formData,
    // Convert participant arrays to proper format
    participants: formData.participants?.map((p: any) => 
        typeof p === "string" ? { userId: p } : p
    ) || [],
    // Extract mentions from message text
    mentions: formData.text ? extractMentions(formData.text) : [],
    // Convert attachment files to upload format
    attachments: formData.attachments?.map((a: any) => ({
        file: a.file,
        metadata: {
            name: a.name,
            type: a.type,
            size: a.size,
        },
    })) || [],
});

/**
 * Mock typing indicators
 */
export const mockTypingStates = {
    single: {
        users: ["user_123456789"],
        text: "John is typing...",
    },
    multiple: {
        users: ["user_123456789", "user_987654321"],
        text: "John and Jane are typing...",
    },
    many: {
        users: ["user_111", "user_222", "user_333", "user_444"],
        text: "Several people are typing...",
    },
};

/**
 * Mock chat suggestions
 */
export const mockChatSuggestions = {
    participants: [
        { id: "user_123", handle: "johndoe", name: "John Doe", avatar: null },
        { id: "user_456", handle: "janedoe", name: "Jane Doe", avatar: null },
        { id: "user_789", handle: "bobsmith", name: "Bob Smith", avatar: null },
    ],
    commands: [
        { command: "/help", description: "Show available commands" },
        { command: "/invite", description: "Invite someone to chat" },
        { command: "/leave", description: "Leave this chat" },
        { command: "/mute", description: "Mute notifications" },
    ],
    emojis: ["😀", "👍", "❤️", "😂", "🎉", "🚀", "💡", "✅"],
};