/**
 * Notification event fixtures for testing push notifications and alerts
 */

import { type Notification } from "../../../api/types.js";
import { type UserSocketEventPayloads } from "../../../consts/socketEvents.js";
import { userFixtures } from "../api/userFixtures.js";

export const notificationEventFixtures = {
    notifications: {
        // New message notification
        newMessage: {
            event: "notification",
            data: {
                id: "notif_123",
                type: "ChatMessage",
                title: "New message from John",
                description: "Hey, are you available?",
                link: "/chat/chat_123",
                isRead: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                from: userFixtures.minimal.find,
            } satisfies Notification,
        },

        // Mention notification
        mention: {
            event: "notification",
            data: {
                id: "notif_mention",
                type: "Mention",
                title: "You were mentioned",
                description: "@you Check out this project!",
                link: "/project/proj_123",
                isRead: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                from: userFixtures.minimal.find,
            } satisfies Notification,
        },

        // System notification
        systemUpdate: {
            event: "notification",
            data: {
                id: "notif_system",
                type: "System",
                title: "System Update",
                description: "New features are now available",
                link: "/changelog",
                isRead: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
            } satisfies Notification,
        },

        // Task completion
        taskComplete: {
            event: "notification",
            data: {
                id: "notif_task",
                type: "RunComplete",
                title: "Task completed successfully",
                description: "Your routine 'Data Analysis' has finished",
                link: "/run/run_123",
                isRead: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
            } satisfies Notification,
        },

        // Award earned
        awardEarned: {
            event: "notification",
            data: {
                id: "notif_award",
                type: "Award",
                title: "Achievement Unlocked!",
                description: "You earned the 'Power User' badge",
                link: "/profile/awards",
                isRead: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
            } satisfies Notification,
        },

        // Team invitation
        teamInvite: {
            event: "notification",
            data: {
                id: "notif_invite",
                type: "TeamInvite",
                title: "Team invitation",
                description: "You've been invited to join 'AI Researchers'",
                link: "/team/team_123/invite",
                isRead: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                from: userFixtures.minimal.find,
            } satisfies Notification,
        },

        // Report response
        reportResponse: {
            event: "notification",
            data: {
                id: "notif_report",
                type: "ReportResponse",
                title: "Report update",
                description: "Your report has been reviewed",
                link: "/reports/report_123",
                isRead: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
            } satisfies Notification,
        },

        // Reminder
        reminder: {
            event: "notification",
            data: {
                id: "notif_reminder",
                type: "Reminder",
                title: "Reminder: Meeting in 15 minutes",
                description: "Team standup at 10:00 AM",
                link: "/calendar/event_123",
                isRead: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
            } satisfies Notification,
        },
    },

    apiCredits: {
        // Credit update
        creditUpdate: {
            event: "apiCredits",
            data: {
                credits: "1000000", // Stringified BigInt
            } satisfies UserSocketEventPayloads["apiCredits"],
        },

        // Low credits warning
        lowCredits: {
            event: "apiCredits",
            data: {
                credits: "100", // Stringified BigInt
            } satisfies UserSocketEventPayloads["apiCredits"],
        },

        // Credits exhausted
        noCredits: {
            event: "apiCredits",
            data: {
                credits: "0", // Stringified BigInt
            } satisfies UserSocketEventPayloads["apiCredits"],
        },

        // Credits refilled
        creditsRefilled: {
            event: "apiCredits",
            data: {
                credits: "5000000", // Stringified BigInt
            } satisfies UserSocketEventPayloads["apiCredits"],
        },
    },

    // Event sequences for testing notification flows
    sequences: {
        // Multiple notifications
        notificationBurst: [
            { event: "notification", data: notificationEventFixtures.notifications.newMessage.data },
            { delay: 100 },
            { event: "notification", data: notificationEventFixtures.notifications.mention.data },
            { delay: 100 },
            { event: "notification", data: notificationEventFixtures.notifications.teamInvite.data },
        ],

        // Task completion flow
        taskCompletionFlow: [
            { event: "notification", data: { ...notificationEventFixtures.notifications.taskComplete.data, description: "Task started: Data Analysis" } },
            { delay: 30000 },
            { event: "notification", data: notificationEventFixtures.notifications.taskComplete.data },
            { delay: 1000 },
            { event: "apiCredits", data: notificationEventFixtures.apiCredits.creditUpdate.data },
        ],

        // Credit warning flow
        creditWarningFlow: [
            { event: "apiCredits", data: notificationEventFixtures.apiCredits.lowCredits.data },
            { event: "notification", data: {
                id: "notif_credit_warning",
                type: "System",
                title: "Low API Credits",
                description: "You have less than 100 credits remaining",
                link: "/settings/billing",
                isRead: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
            }},
            { delay: 10000 },
            { event: "apiCredits", data: notificationEventFixtures.apiCredits.noCredits.data },
            { event: "notification", data: {
                id: "notif_credit_exhausted",
                type: "System",
                title: "API Credits Exhausted",
                description: "Please purchase more credits to continue",
                link: "/settings/billing",
                isRead: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
            }},
        ],

        // Achievement flow
        achievementFlow: [
            { event: "notification", data: notificationEventFixtures.notifications.awardEarned.data },
            { delay: 2000 },
            { event: "notification", data: {
                ...notificationEventFixtures.notifications.awardEarned.data,
                id: "notif_award_2",
                description: "You earned the 'Contributor' badge",
            }},
            { delay: 1000 },
            { event: "notification", data: {
                id: "notif_level_up",
                type: "System",
                title: "Level Up!",
                description: "You've reached Level 5",
                link: "/profile",
                isRead: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
            }},
        ],
    },

    // Factory functions for dynamic event creation
    factories: {
        createNotification: (type: string, title: string, description: string, link?: string) => ({
            event: "notification",
            data: {
                id: `notif_${Date.now()}`,
                type,
                title,
                description,
                link,
                isRead: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
            } satisfies Notification,
        }),

        createMessageNotification: (fromUser: typeof userFixtures.minimal.find, message: string) => ({
            event: "notification",
            data: {
                id: `notif_msg_${Date.now()}`,
                type: "ChatMessage",
                title: `New message from ${fromUser.name}`,
                description: message,
                link: "/chat",
                isRead: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                from: fromUser,
            } satisfies Notification,
        }),

        createCreditUpdate: (credits: number | bigint) => ({
            event: "apiCredits",
            data: {
                credits: credits.toString(),
            } satisfies UserSocketEventPayloads["apiCredits"],
        }),

        createSystemNotification: (title: string, description: string, priority?: "low" | "medium" | "high") => ({
            event: "notification",
            data: {
                id: `notif_sys_${Date.now()}`,
                type: "System",
                title,
                description,
                link: "/notifications",
                isRead: false,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                metadata: priority ? { priority } : undefined,
            } satisfies Notification,
        }),
    },
};