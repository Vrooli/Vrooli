import { type ChatConfigObject } from "../../../shape/configs/chat.js";
import { LATEST_CONFIG_VERSION } from "../../../shape/configs/utils.js";
import { type ConfigTestFixtures } from "./baseConfigFixtures.js";

/**
 * Chat configuration fixtures for testing chat behavior settings
 */
export const chatConfigFixtures: ConfigTestFixtures<ChatConfigObject> = {
    minimal: {
        __version: LATEST_CONFIG_VERSION,
    },

    complete: {
        __version: LATEST_CONFIG_VERSION,
        autoDelete: {
            enabled: true,
            daysAfterLastMessage: 30,
        },
        messageRetention: {
            maxMessages: 1000,
            maxDays: 90,
        },
        permissions: {
            allowGuestMessages: false,
            allowFileUploads: true,
            maxFileSize: 10485760, // 10MB
            allowedFileTypes: ["image/*", "application/pdf", "text/*"],
        },
        moderation: {
            enabled: true,
            filterProfanity: true,
            requireApproval: false,
            customFilters: ["spam", "advertising"],
        },
        notifications: {
            onNewMessage: true,
            onMention: true,
            onReaction: false,
            digest: {
                enabled: true,
                frequency: "daily",
            },
        },
        resources: [{
            link: "https://example.com/chat-rules",
            usedFor: "Community",
            translations: [{
                language: "en",
                name: "Chat Guidelines",
                description: "Community guidelines for this chat",
            }],
        }],
    },

    withDefaults: {
        __version: LATEST_CONFIG_VERSION,
        autoDelete: {
            enabled: false,
            daysAfterLastMessage: 0,
        },
        permissions: {
            allowGuestMessages: true,
            allowFileUploads: false,
        },
    },

    invalid: {
        missingVersion: {
            autoDelete: {
                enabled: true,
                daysAfterLastMessage: 30,
            },
        },
        invalidVersion: {
            __version: "0.5",
            permissions: {},
        },
        malformedStructure: {
            __version: LATEST_CONFIG_VERSION,
            autoDelete: "should be object", // Wrong type
        },
        invalidTypes: {
            __version: LATEST_CONFIG_VERSION,
            autoDelete: {
                enabled: "yes", // Should be boolean
                daysAfterLastMessage: "thirty", // Should be number
            },
        },
    },

    variants: {
        publicChat: {
            __version: LATEST_CONFIG_VERSION,
            permissions: {
                allowGuestMessages: true,
                allowFileUploads: false,
                allowReactions: true,
                allowThreads: true,
            },
            moderation: {
                enabled: true,
                filterProfanity: true,
                requireApproval: false,
                autoModerateLinks: true,
            },
        },

        privateTeamChat: {
            __version: LATEST_CONFIG_VERSION,
            permissions: {
                allowGuestMessages: false,
                allowFileUploads: true,
                maxFileSize: 52428800, // 50MB
                allowedFileTypes: ["*"], // All file types
            },
            moderation: {
                enabled: false,
            },
            encryption: {
                enabled: true,
                algorithm: "AES-256-GCM",
            },
        },

        supportChat: {
            __version: LATEST_CONFIG_VERSION,
            autoDelete: {
                enabled: false, // Keep support conversations
            },
            permissions: {
                allowGuestMessages: true,
                allowFileUploads: true,
                maxFileSize: 5242880, // 5MB
                allowedFileTypes: ["image/*", "text/*"],
            },
            routing: {
                enabled: true,
                defaultAgent: "support_bot",
                escalationRules: {
                    keywords: ["urgent", "emergency", "broken"],
                    timeoutMinutes: 5,
                },
            },
            businessHours: {
                enabled: true,
                timezone: "America/New_York",
                schedule: {
                    monday: { start: "09:00", end: "17:00" },
                    tuesday: { start: "09:00", end: "17:00" },
                    wednesday: { start: "09:00", end: "17:00" },
                    thursday: { start: "09:00", end: "17:00" },
                    friday: { start: "09:00", end: "17:00" },
                    saturday: { closed: true },
                    sunday: { closed: true },
                },
            },
        },

        ephemeralChat: {
            __version: LATEST_CONFIG_VERSION,
            autoDelete: {
                enabled: true,
                daysAfterLastMessage: 1,
            },
            messageRetention: {
                maxMessages: 100,
                maxDays: 1,
            },
            permissions: {
                allowGuestMessages: true,
                allowFileUploads: false,
                allowEditing: false,
                allowDeleting: false,
            },
        },

        broadcastChannel: {
            __version: LATEST_CONFIG_VERSION,
            permissions: {
                allowGuestMessages: false,
                allowFileUploads: false,
                onlyAdminsCanPost: true,
                allowReactions: true,
            },
            notifications: {
                onNewMessage: true,
                forcePushNotifications: true,
                digest: {
                    enabled: false, // No digests for broadcasts
                },
            },
        },
    },
};

/**
 * Create a chat config with specific permission settings
 */
export function createChatConfigWithPermissions(
    permissions: Partial<ChatConfigObject["permissions"]>,
): ChatConfigObject {
    return {
        __version: LATEST_CONFIG_VERSION,
        permissions: {
            allowGuestMessages: true,
            allowFileUploads: false,
            ...permissions,
        },
    };
}

/**
 * Create a moderated chat config
 */
export function createModeratedChatConfig(
    moderationSettings: Partial<ChatConfigObject["moderation"]> = {},
): ChatConfigObject {
    return {
        __version: LATEST_CONFIG_VERSION,
        moderation: {
            enabled: true,
            filterProfanity: true,
            requireApproval: false,
            ...moderationSettings,
        },
    };
}
