/**
 * Enhanced notification event fixtures using the factory pattern
 * Provides comprehensive testing for notifications, API credits, and event flows
 */

import { type Notification } from "../../../api/types.js";
import { EventTypes, type UserSocketEventPayloads } from "../../../consts/socketEvents.js";
import { BaseEventFactory } from "./BaseEventFactory.js";
import { type SocketEventFixture } from "./types.js";
 
// Notification event types
type NotificationEvent = SocketEventFixture<Notification> & {
    event: "notification";
};

type ApiCreditEvent = SocketEventFixture<UserSocketEventPayloads[typeof EventTypes.USER.CREDITS_UPDATED]> & {
    event: typeof EventTypes.USER.CREDITS_UPDATED;
};

// Notification categories and priorities
export interface NotificationCategory {
    chatMessage: "ChatMessage";
    mention: "Mention";
    award: "Award";
    taskComplete: "RunComplete";
    teamInvite: "TeamInvite";
    reportResponse: "ReportResponse";
    reminder: "Reminder";
    system: "System";
    apiCredit: "ApiCredit";
}

export interface NotificationPriority {
    low: "low";
    medium: "medium";
    high: "high";
    urgent: "urgent";
}

/**
 * Factory for creating notification events with various types and scenarios
 */
export class NotificationEventFactory extends BaseEventFactory<NotificationEvent, Notification> {
    constructor() {
        super("notification", {
            validation: (data: Notification) => {
                if (!data.id || !data.title) {
                    return "Notification must have id and title";
                }
                return true;
            },
            transform: (data: Notification) => ({
                ...data,
                createdAt: data.createdAt || new Date().toISOString(),
            }),
        });
    }

    get single(): NotificationEvent {
        return {
            event: "notification",
            data: {
                __typename: "Notification",
                id: "notif_123",
                category: "ChatMessage",
                title: "New message from John",
                description: "Hey, are you available?",
                link: "/chat/chat_123",
                isRead: false,
                createdAt: new Date().toISOString(),
                imgLink: null,
            },
        };
    }

    get sequence(): NotificationEvent[] {
        return [
            this.createNewMessage("John", "Hey there!"),
            this.createMention("@you Check this out!"),
            this.createTaskCompletion("Data Analysis"),
            this.createAward("Power User", "Great job!"),
            this.createReminder("Meeting in 15 minutes"),
        ];
    }

    get variants(): Record<string, NotificationEvent | NotificationEvent[]> {
        return {
            // Single notification types
            newMessage: this.createNewMessage("Alice", "How are you?"),
            mention: this.createMention("@you Please review this"),
            award: this.createAward("Contributor", "Thanks for your help!"),
            taskComplete: this.createTaskCompletion("Image Processing"),
            teamInvite: this.createTeamInvite("AI Researchers"),
            reportResponse: this.createReportResponse("Your report was reviewed"),
            reminder: this.createReminder("Daily standup in 10 minutes"),
            systemUpdate: this.createSystemUpdate("New features available"),

            // Priority variations
            lowPriority: this.createWithPriority("Low priority notification", "low"),
            mediumPriority: this.createWithPriority("Medium priority notification", "medium"),
            highPriority: this.createWithPriority("High priority notification", "high"),
            urgentPriority: this.createWithPriority("Urgent notification", "urgent"),

            // Read state variations
            unread: this.createWithReadState("Unread notification", false),
            read: this.createWithReadState("Read notification", true),

            // With and without links
            withLink: this.createWithLink("Notification with link", "/some/path"),
            withoutLink: this.createWithoutLink("Notification without link"),

            // With and without images
            withImage: this.createWithImage("Notification with image", "/img/notification.png"),
            withoutImage: this.createWithoutImage("Notification without image"),

            // Batch notifications
            messageBurst: this.createMessageBurst(5),
            mixedNotifications: this.createMixedBatch(),
            priorityEscalation: this.createPriorityEscalation(),
        };
    }

    // Specific notification creators
    createNewMessage(fromUser: string, message: string): NotificationEvent {
        return this.create({
            __typename: "Notification",
            id: `notif_msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
            category: "ChatMessage",
            title: `New message from ${fromUser}`,
            description: message,
            link: "/chat",
            isRead: false,
            createdAt: new Date().toISOString(),
            imgLink: null,
        });
    }

    createMention(content: string): NotificationEvent {
        return this.create({
            __typename: "Notification",
            id: `notif_mention_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
            category: "Mention",
            title: "You were mentioned",
            description: content,
            link: "/notifications",
            isRead: false,
            createdAt: new Date().toISOString(),
            imgLink: null,
        });
    }

    createAward(badgeName: string, description: string): NotificationEvent {
        return this.create({
            __typename: "Notification",
            id: `notif_award_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
            category: "Award",
            title: "Achievement Unlocked!",
            description: `You earned the '${badgeName}' badge: ${description}`,
            link: "/profile/awards",
            isRead: false,
            createdAt: new Date().toISOString(),
            imgLink: `/img/badges/${badgeName.toLowerCase().replace(/\s+/g, "-")}.png`,
        });
    }

    createTaskCompletion(taskName: string): NotificationEvent {
        return this.create({
            __typename: "Notification",
            id: `notif_task_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
            category: "RunComplete",
            title: "Task completed successfully",
            description: `Your routine '${taskName}' has finished`,
            link: "/runs",
            isRead: false,
            createdAt: new Date().toISOString(),
            imgLink: null,
        });
    }

    createTeamInvite(teamName: string): NotificationEvent {
        return this.create({
            __typename: "Notification",
            id: `notif_invite_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
            category: "TeamInvite",
            title: "Team invitation",
            description: `You've been invited to join '${teamName}'`,
            link: "/teams/invitations",
            isRead: false,
            createdAt: new Date().toISOString(),
            imgLink: null,
        });
    }

    createReportResponse(message: string): NotificationEvent {
        return this.create({
            __typename: "Notification",
            id: `notif_report_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
            category: "ReportResponse",
            title: "Report update",
            description: message,
            link: "/reports",
            isRead: false,
            createdAt: new Date().toISOString(),
            imgLink: null,
        });
    }

    createReminder(message: string): NotificationEvent {
        return this.create({
            __typename: "Notification",
            id: `notif_reminder_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
            category: "Reminder",
            title: "Reminder",
            description: message,
            link: "/calendar",
            isRead: false,
            createdAt: new Date().toISOString(),
            imgLink: null,
        });
    }

    createSystemUpdate(message: string): NotificationEvent {
        return this.create({
            __typename: "Notification",
            id: `notif_system_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
            category: "System",
            title: "System Update",
            description: message,
            link: "/changelog",
            isRead: false,
            createdAt: new Date().toISOString(),
            imgLink: null,
        });
    }

    // Utility creators for variations
    createWithPriority(title: string, priority: keyof NotificationPriority): NotificationEvent {
        return this.create({
            __typename: "Notification",
            id: `notif_${priority}_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
            category: "System",
            title,
            description: `This is a ${priority} priority notification`,
            link: "/notifications",
            isRead: false,
            createdAt: new Date().toISOString(),
            imgLink: null,
        });
    }

    createWithReadState(title: string, isRead: boolean): NotificationEvent {
        return this.create({
            __typename: "Notification",
            id: `notif_read_${isRead}_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
            category: "System",
            title,
            description: `This notification is ${isRead ? "read" : "unread"}`,
            link: "/notifications",
            isRead,
            createdAt: new Date().toISOString(),
            imgLink: null,
        });
    }

    createWithLink(title: string, link: string): NotificationEvent {
        return this.create({
            __typename: "Notification",
            id: `notif_link_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
            category: "System",
            title,
            description: "This notification has a link",
            link,
            isRead: false,
            createdAt: new Date().toISOString(),
            imgLink: null,
        });
    }

    createWithoutLink(title: string): NotificationEvent {
        return this.create({
            __typename: "Notification",
            id: `notif_nolink_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
            category: "System",
            title,
            description: "This notification has no link",
            link: null,
            isRead: false,
            createdAt: new Date().toISOString(),
            imgLink: null,
        });
    }

    createWithImage(title: string, imgLink: string): NotificationEvent {
        return this.create({
            __typename: "Notification",
            id: `notif_img_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
            category: "System",
            title,
            description: "This notification has an image",
            link: "/notifications",
            isRead: false,
            createdAt: new Date().toISOString(),
            imgLink,
        });
    }

    createWithoutImage(title: string): NotificationEvent {
        return this.create({
            __typename: "Notification",
            id: `notif_noimg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
            category: "System",
            title,
            description: "This notification has no image",
            link: "/notifications",
            isRead: false,
            createdAt: new Date().toISOString(),
            imgLink: null,
        });
    }

    createMessageBurst(count: number): NotificationEvent[] {
        return Array.from({ length: count }, (_, i) =>
            this.createNewMessage(`User${i + 1}`, `Message ${i + 1}`),
        );
    }

    createMixedBatch(): NotificationEvent[] {
        return [
            this.createNewMessage("Alice", "Hello!"),
            this.createAward("Helper", "Thanks for helping others"),
            this.createTaskCompletion("Data Processing"),
            this.createReminder("Meeting in 30 minutes"),
            this.createSystemUpdate("Security update applied"),
        ];
    }

    createPriorityEscalation(): NotificationEvent[] {
        return [
            this.createWithPriority("Initial notification", "low"),
            this.createWithPriority("Follow-up notification", "medium"),
            this.createWithPriority("Important notification", "high"),
            this.createWithPriority("Critical notification", "urgent"),
        ];
    }

    protected applyEventToState(
        state: Record<string, unknown>,
        event: NotificationEvent,
    ): Record<string, unknown> {
        const notificationCount = (state.notificationCount as number) || 0;
        const unreadCount = (state.unreadCount as number) || 0;

        return {
            ...state,
            lastNotification: event.data,
            notificationCount: notificationCount + 1,
            unreadCount: event.data.isRead ? unreadCount : unreadCount + 1,
            lastEventTime: Date.now(),
        };
    }
}

/**
 * Factory for creating API credit events and credit-related notifications
 */
export class ApiCreditEventFactory extends BaseEventFactory<ApiCreditEvent, UserSocketEventPayloads[typeof EventTypes.USER.CREDITS_UPDATED]> {
    constructor() {
        super(EventTypes.USER.CREDITS_UPDATED, {
            validation: (data: UserSocketEventPayloads[typeof EventTypes.USER.CREDITS_UPDATED]) => {
                if (!data.credits || isNaN(Number(data.credits))) {
                    return "Credits must be a valid stringified number";
                }
                return true;
            },
        });
    }

    get single(): ApiCreditEvent {
        return {
            event: EventTypes.USER.CREDITS_UPDATED,
            data: {
                userId: "user123",
                credits: "1000000", // Stringified BigInt
            },
        };
    }

    get sequence(): ApiCreditEvent[] {
        return [
            this.createCreditUpdate(5000000),
            this.createCreditUpdate(4000000),
            this.createCreditUpdate(2000000),
            this.createCreditUpdate(500000),
            this.createCreditUpdate(100),
            this.createCreditUpdate(0),
        ];
    }

    get variants(): Record<string, ApiCreditEvent | ApiCreditEvent[]> {
        return {
            // Single credit states
            plenty: this.createCreditUpdate(10000000),
            sufficient: this.createCreditUpdate(1000000),
            low: this.createCreditUpdate(10000),
            veryLow: this.createCreditUpdate(100),
            exhausted: this.createCreditUpdate(0),

            // Credit change sequences
            consumption: this.createConsumptionSequence(1000000, 100000, 6),
            refill: this.createRefillSequence(0, 5000000),
            depletion: this.createDepletionSequence(5000000),
            recovery: this.createRecoverySequence(),

            // Edge cases
            maxCredits: this.createCreditUpdate(Number.MAX_SAFE_INTEGER),
            negativeCredits: this.createCreditUpdate(-1000), // Should be handled gracefully
            floatCredits: this.createCreditUpdate(1000.5), // Should be handled
        };
    }

    createCreditUpdate(credits: number): ApiCreditEvent {
        return this.create({
            credits: credits.toString(),
        });
    }

    createConsumptionSequence(start: number, decrement: number, steps: number): ApiCreditEvent[] {
        return Array.from({ length: steps }, (_, i) =>
            this.createCreditUpdate(Math.max(0, start - (i * decrement))),
        );
    }

    createRefillSequence(from: number, to: number): ApiCreditEvent[] {
        const steps = 5;
        const increment = (to - from) / steps;
        return Array.from({ length: steps + 1 }, (_, i) =>
            this.createCreditUpdate(Math.floor(from + (i * increment))),
        );
    }

    createDepletionSequence(startAmount: number): ApiCreditEvent[] {
        const consumptionRates = [100000, 200000, 500000, 1000000, 2000000];
        const events: ApiCreditEvent[] = [this.createCreditUpdate(startAmount)];

        let current = startAmount;
        for (const rate of consumptionRates) {
            current = Math.max(0, current - rate);
            events.push(this.createCreditUpdate(current));
            if (current === 0) break;
        }

        return events;
    }

    createRecoverySequence(): ApiCreditEvent[] {
        return [
            this.createCreditUpdate(0),        // Exhausted
            this.createCreditUpdate(100000),   // Small refill
            this.createCreditUpdate(500000),   // Medium refill
            this.createCreditUpdate(2000000),  // Large refill
            this.createCreditUpdate(5000000),  // Full refill
        ];
    }

    protected applyEventToState(
        state: Record<string, unknown>,
        event: ApiCreditEvent,
    ): Record<string, unknown> {
        const previousCredits = Number(state.credits || "0");
        const newCredits = Number(event.data.credits);

        return {
            ...state,
            credits: event.data.credits,
            creditChange: newCredits - previousCredits,
            lastCreditUpdate: Date.now(),
        };
    }
}

/**
 * Factory for creating notification burst scenarios and flood handling
 */
export class NotificationBurstEventFactory extends BaseEventFactory<NotificationEvent, Notification> {
    private notificationFactory: NotificationEventFactory;

    constructor() {
        super("notificationBurst");
        this.notificationFactory = new NotificationEventFactory();
    }

    get single(): NotificationEvent {
        return this.notificationFactory.single;
    }

    get sequence(): NotificationEvent[] {
        return this.createNotificationFlood(10);
    }

    get variants(): Record<string, NotificationEvent | NotificationEvent[]> {
        return {
            // Different burst sizes
            smallBurst: this.createNotificationFlood(3),
            mediumBurst: this.createNotificationFlood(10),
            largeBurst: this.createNotificationFlood(25),
            floodBurst: this.createNotificationFlood(100),

            // Burst patterns
            messageBurst: this.createMessageFlood(15),
            mixedBurst: this.createMixedNotificationBurst(20),
            priorityBurst: this.createPriorityBurst(),
            escalatingBurst: this.createEscalatingBurst(),

            // Throttling scenarios
            rapidBurst: this.createRapidBurst(10, 50),    // 10 notifications, 50ms apart
            slowBurst: this.createSlowBurst(5, 2000),     // 5 notifications, 2s apart
            batchedBurst: this.createBatchedBurst(20, 5), // 20 notifications in batches of 5

            // Real-world scenarios
            chatConversation: this.createChatConversation(),
            systemMaintenance: this.createSystemMaintenanceSequence(),
            achievementSpree: this.createAchievementSpree(),
            taskCompletionBatch: this.createTaskCompletionBatch(8),
        };
    }

    createNotificationFlood(count: number): NotificationEvent[] {
        return Array.from({ length: count }, (_, i) =>
            this.notificationFactory.createNewMessage(
                `User${i + 1}`,
                `Flood message ${i + 1}`,
            ),
        );
    }

    createMessageFlood(count: number): NotificationEvent[] {
        const users = ["Alice", "Bob", "Charlie", "Diana", "Eve", "Frank"];
        return Array.from({ length: count }, (_, i) =>
            this.notificationFactory.createNewMessage(
                users[i % users.length],
                `Message ${i + 1} from ${users[i % users.length]}`,
            ),
        );
    }

    createMixedNotificationBurst(count: number): NotificationEvent[] {
        const types = [
            () => this.notificationFactory.createNewMessage("Alice", "Hello!"),
            () => this.notificationFactory.createMention("@you Check this"),
            () => this.notificationFactory.createAward("Badge", "Achievement"),
            () => this.notificationFactory.createTaskCompletion("Task"),
            () => this.notificationFactory.createReminder("Reminder"),
            () => this.notificationFactory.createSystemUpdate("Update"),
        ];

        return Array.from({ length: count }, (_, i) =>
            types[i % types.length](),
        );
    }

    createPriorityBurst(): NotificationEvent[] {
        return [
            this.notificationFactory.createWithPriority("Info 1", "low"),
            this.notificationFactory.createWithPriority("Info 2", "low"),
            this.notificationFactory.createWithPriority("Warning 1", "medium"),
            this.notificationFactory.createWithPriority("Alert 1", "high"),
            this.notificationFactory.createWithPriority("Warning 2", "medium"),
            this.notificationFactory.createWithPriority("Critical", "urgent"),
            this.notificationFactory.createWithPriority("Alert 2", "high"),
        ];
    }

    createEscalatingBurst(): NotificationEvent[] {
        const notifications: NotificationEvent[] = [];

        // Start with low priority, escalate over time
        for (let i = 0; i < 3; i++) {
            notifications.push(this.notificationFactory.createWithPriority(`Low priority ${i + 1}`, "low"));
        }
        for (let i = 0; i < 2; i++) {
            notifications.push(this.notificationFactory.createWithPriority(`Medium priority ${i + 1}`, "medium"));
        }
        for (let i = 0; i < 2; i++) {
            notifications.push(this.notificationFactory.createWithPriority(`High priority ${i + 1}`, "high"));
        }
        notifications.push(this.notificationFactory.createWithPriority("Urgent priority", "urgent"));

        return notifications;
    }

    createRapidBurst(count: number, intervalMs: number): NotificationEvent[] {
        return this.withTiming(
            this.createNotificationFlood(count),
            Array.from({ length: count }, (_, i) => i * intervalMs),
        ) as NotificationEvent[];
    }

    createSlowBurst(count: number, intervalMs: number): NotificationEvent[] {
        return this.withTiming(
            this.createMixedNotificationBurst(count),
            Array.from({ length: count }, (_, i) => i * intervalMs),
        ) as NotificationEvent[];
    }

    createBatchedBurst(totalCount: number, batchSize: number): NotificationEvent[] {
        const notifications: NotificationEvent[] = [];
        const batchCount = Math.ceil(totalCount / batchSize);

        for (let batch = 0; batch < batchCount; batch++) {
            const itemsInBatch = Math.min(batchSize, totalCount - (batch * batchSize));
            const batchNotifications = this.createNotificationFlood(itemsInBatch);

            // Add timing: all items in batch have same delay, batches are spaced apart
            const batchDelay = batch * 5000; // 5 seconds between batches
            const timedBatch = batchNotifications.map(notif =>
                this.withDelay(notif, batchDelay),
            ) as NotificationEvent[];

            notifications.push(...timedBatch);
        }

        return notifications;
    }

    createChatConversation(): NotificationEvent[] {
        const conversation = [
            { user: "Alice", message: "Hey everyone!" },
            { user: "Bob", message: "Hi Alice! How are you?" },
            { user: "Alice", message: "Great! Working on the new project" },
            { user: "Charlie", message: "@Alice Which project?" },
            { user: "Alice", message: "The AI integration one" },
            { user: "Bob", message: "Sounds exciting!" },
            { user: "Diana", message: "Can I help with that?" },
            { user: "Alice", message: "@Diana Absolutely! I'll send you the details" },
            { user: "Charlie", message: "Keep us posted on progress" },
            { user: "Alice", message: "Will do! Thanks everyone" },
        ];

        return conversation.map((msg, i) => {
            const notification = msg.message.includes("@")
                ? this.notificationFactory.createMention(msg.message)
                : this.notificationFactory.createNewMessage(msg.user, msg.message);

            return this.withDelay(notification, i * 2000) as NotificationEvent; // 2 seconds between messages
        });
    }

    createSystemMaintenanceSequence(): NotificationEvent[] {
        const maintenanceEvents = [
            "Scheduled maintenance will begin in 30 minutes",
            "Scheduled maintenance will begin in 15 minutes",
            "Scheduled maintenance will begin in 5 minutes",
            "System maintenance has started",
            "Maintenance in progress - some features may be unavailable",
            "Maintenance completed - all systems restored",
            "Thank you for your patience during maintenance",
        ];

        return maintenanceEvents.map((message, i) => {
            const notification = this.notificationFactory.createSystemUpdate(message);
            return this.withDelay(notification, i * 15 * 60 * 1000) as NotificationEvent; // 15 minutes apart
        });
    }

    createAchievementSpree(): NotificationEvent[] {
        const achievements = [
            { badge: "First Steps", desc: "Welcome to the platform!" },
            { badge: "Explorer", desc: "You've visited 10 different sections" },
            { badge: "Contributor", desc: "You've made your first contribution" },
            { badge: "Helper", desc: "You've helped other users" },
            { badge: "Power User", desc: "You've mastered the advanced features" },
            { badge: "Legend", desc: "You've reached the highest level" },
        ];

        return achievements.map((achievement, i) => {
            const notification = this.notificationFactory.createAward(achievement.badge, achievement.desc);
            return this.withDelay(notification, i * 3000) as NotificationEvent; // 3 seconds apart
        });
    }

    createTaskCompletionBatch(count: number): NotificationEvent[] {
        const taskNames = [
            "Data Processing", "Image Analysis", "Report Generation", "Email Automation",
            "File Organization", "Backup Creation", "Security Scan", "Performance Test",
            "Code Review", "Documentation Update", "Database Cleanup", "Log Analysis",
        ];

        return Array.from({ length: count }, (_, i) => {
            const taskName = taskNames[i % taskNames.length];
            const notification = this.notificationFactory.createTaskCompletion(`${taskName} ${Math.floor(i / taskNames.length) + 1}`);
            return this.withDelay(notification, i * 5000) as NotificationEvent; // 5 seconds apart
        });
    }

    protected applyEventToState(
        state: Record<string, unknown>,
        event: NotificationEvent,
    ): Record<string, unknown> {
        const burstCount = (state.burstCount as number) || 0;
        const burstSize = (state.currentBurstSize as number) || 0;

        return {
            ...state,
            burstCount: burstCount + 1,
            currentBurstSize: burstSize + 1,
            lastBurstEvent: event.data,
            burstStartTime: state.burstStartTime || Date.now(),
            lastBurstTime: Date.now(),
        };
    }
}

// Factory instances for easy access
export const notificationEventFactory = new NotificationEventFactory();
export const apiCreditEventFactory = new ApiCreditEventFactory();
export const notificationBurstEventFactory = new NotificationBurstEventFactory();

// Backward compatibility: maintain existing notificationEventFixtures export
export const notificationEventFixtures = {
    notifications: {
        newMessage: notificationEventFactory.variants.newMessage as NotificationEvent,
        mention: notificationEventFactory.variants.mention as NotificationEvent,
        award: notificationEventFactory.variants.award as NotificationEvent,
        taskComplete: notificationEventFactory.variants.taskComplete as NotificationEvent,
        teamInvite: notificationEventFactory.variants.teamInvite as NotificationEvent,
        reportResponse: notificationEventFactory.variants.reportResponse as NotificationEvent,
        reminder: notificationEventFactory.variants.reminder as NotificationEvent,
        systemUpdate: notificationEventFactory.variants.systemUpdate as NotificationEvent,
    },
    apiCredits: {
        creditUpdate: apiCreditEventFactory.variants.sufficient as ApiCreditEvent,
        lowCredits: apiCreditEventFactory.variants.veryLow as ApiCreditEvent,
        noCredits: apiCreditEventFactory.variants.exhausted as ApiCreditEvent,
        creditsRefilled: apiCreditEventFactory.variants.plenty as ApiCreditEvent,
    },
    sequences: {
        notificationBurst: notificationBurstEventFactory.variants.smallBurst as NotificationEvent[],
        taskCompletionFlow: [
            notificationEventFactory.createTaskCompletion("Data Analysis"),
            ...apiCreditEventFactory.variants.consumption as ApiCreditEvent[],
        ],
        creditWarningFlow: apiCreditEventFactory.variants.depletion as ApiCreditEvent[],
        achievementFlow: notificationBurstEventFactory.variants.achievementSpree as NotificationEvent[],
    },
    factories: {
        createNotification: (category: string, title: string, description: string, link?: string) =>
            notificationEventFactory.create({
                __typename: "Notification",
                id: `notif_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
                category,
                title,
                description,
                link: link || null,
                isRead: false,
                createdAt: new Date().toISOString(),
                imgLink: null,
            }),
        createMessageNotification: (fromUser: string, message: string) =>
            notificationEventFactory.createNewMessage(fromUser, message),
        createCreditUpdate: (credits: number | bigint) =>
            apiCreditEventFactory.createCreditUpdate(Number(credits)),
        createSystemNotification: (title: string, description: string, priority?: "low" | "medium" | "high") =>
            notificationEventFactory.createSystemUpdate(`${title}: ${description}`),
    },
};
