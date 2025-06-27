/* c8 ignore start */
/**
 * Notification API Response Fixtures
 * 
 * Comprehensive fixtures for user notification endpoints including
 * notification management, bulk operations, and category filtering.
 */

import type {
    Notification,
    NotificationCreateInput,
    NotificationUpdateInput,
} from "../../../api/types.js";
import { BaseAPIResponseFactory } from "./base.js";
import type { MockDataOptions } from "./types.js";
import { generatePK } from "../../../id/index.js";

// Constants
const DEFAULT_COUNT = 10;
const DEFAULT_ERROR_RATE = 0.1;
const DEFAULT_DELAY_MS = 500;
const MAX_BULK_OPERATIONS = 100;
const MAX_TITLE_LENGTH = 200;
const MAX_DESCRIPTION_LENGTH = 1000;

// Notification categories
type NotificationCategory = 
    | "ChatMessage"
    | "Mention" 
    | "Award"
    | "RunComplete"
    | "TeamInvite"
    | "ReportResponse"
    | "Reminder"
    | "System"
    | "ApiCredit";

/**
 * Notification API response factory
 */
export class NotificationResponseFactory extends BaseAPIResponseFactory<
    Notification,
    NotificationCreateInput,
    NotificationUpdateInput
> {
    protected readonly entityName = "notification";

    /**
     * Create mock notification data
     */
    createMockData(options?: MockDataOptions): Notification {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();
        const notificationId = options?.overrides?.id || generatePK().toString();

        const baseNotification: Notification = {
            __typename: "Notification",
            id: notificationId,
            created_at: now,
            updated_at: now,
            category: "System",
            title: "System Notification",
            description: "This is a test notification message",
            link: "/notifications",
            isRead: false,
            imgLink: null,
        };

        if (scenario === "complete" || scenario === "edge-case") {
            return {
                ...baseNotification,
                category: "RunComplete",
                title: "Routine Execution Completed Successfully",
                description: "Your advanced data processing routine has finished execution with 98% success rate. All validation checks passed and results are ready for review.",
                link: "/runs/12345",
                imgLink: "/img/success-badge.png",
                isRead: false,
                ...options?.overrides,
            };
        }

        return {
            ...baseNotification,
            ...options?.overrides,
        };
    }

    /**
     * Create notification from input
     */
    createFromInput(input: NotificationCreateInput): Notification {
        const now = new Date().toISOString();
        const notificationId = generatePK().toString();

        return {
            __typename: "Notification",
            id: notificationId,
            created_at: now,
            updated_at: now,
            category: input.category || "System",
            title: input.title || "Notification",
            description: input.description || null,
            link: input.link || "/notifications",
            isRead: false,
            imgLink: input.imgLink || null,
        };
    }

    /**
     * Update notification from input
     */
    updateFromInput(existing: Notification, input: NotificationUpdateInput): Notification {
        const updates: Partial<Notification> = {
            updated_at: new Date().toISOString(),
        };

        if (input.isRead !== undefined) updates.isRead = input.isRead;
        if (input.title !== undefined) updates.title = input.title;
        if (input.description !== undefined) updates.description = input.description;
        if (input.link !== undefined) updates.link = input.link;
        if (input.imgLink !== undefined) updates.imgLink = input.imgLink;

        return {
            ...existing,
            ...updates,
        };
    }

    /**
     * Validate create input
     */
    async validateCreateInput(input: NotificationCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.title) {
            errors.title = "Title is required";
        } else if (input.title.length > MAX_TITLE_LENGTH) {
            errors.title = "Title must be 200 characters or less";
        }

        if (input.description && input.description.length > MAX_DESCRIPTION_LENGTH) {
            errors.description = "Description must be 1000 characters or less";
        }

        if (input.category) {
            const validCategories: NotificationCategory[] = [
                "ChatMessage", "Mention", "Award", "RunComplete", "TeamInvite", 
                "ReportResponse", "Reminder", "System", "ApiCredit",
            ];
            if (!validCategories.includes(input.category as NotificationCategory)) {
                errors.category = "Invalid notification category";
            }
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: NotificationUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (input.title !== undefined) {
            if (input.title.length === 0) {
                errors.title = "Title cannot be empty";
            } else if (input.title.length > MAX_TITLE_LENGTH) {
                errors.title = "Title must be 200 characters or less";
            }
        }

        if (input.description !== undefined && input.description.length > MAX_DESCRIPTION_LENGTH) {
            errors.description = "Description must be 1000 characters or less";
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create notifications for different categories
     */
    createNotificationsForAllCategories(): Notification[] {
        const categories: { category: NotificationCategory; title: string; description: string; link: string; imgLink?: string }[] = [
            {
                category: "ChatMessage",
                title: "New message from Alice",
                description: "Hey, how are you doing?",
                link: "/chat/alice",
            },
            {
                category: "Mention",
                title: "You were mentioned",
                description: "@you Check out this new feature!",
                link: "/notifications",
            },
            {
                category: "Award",
                title: "Achievement Unlocked!",
                description: "You earned the 'Contributor' badge for helping others",
                link: "/profile/awards",
                imgLink: "/img/badges/contributor.png",
            },
            {
                category: "RunComplete",
                title: "Task completed successfully",
                description: "Your routine 'Data Analysis' has finished",
                link: "/runs/12345",
            },
            {
                category: "TeamInvite",
                title: "Team invitation",
                description: "You've been invited to join 'AI Research Team'",
                link: "/teams/invitations",
            },
            {
                category: "ReportResponse",
                title: "Report update",
                description: "Your report has been reviewed and approved",
                link: "/reports/67890",
            },
            {
                category: "Reminder",
                title: "Reminder",
                description: "Meeting starts in 15 minutes",
                link: "/calendar",
            },
            {
                category: "System",
                title: "System Update",
                description: "New features are now available",
                link: "/settings",
                imgLink: "/img/system-update.png",
            },
            {
                category: "ApiCredit",
                title: "API Credits Update",
                description: "Your API credits have been updated",
                link: "/settings/billing",
            },
        ];

        return categories.map(({ category, title, description, link, imgLink }) =>
            this.createMockData({
                overrides: {
                    category,
                    title,
                    description,
                    link,
                    imgLink: imgLink || null,
                },
            }),
        );
    }

    /**
     * Create notifications with mixed read states
     */
    createNotificationsWithReadStates(): Notification[] {
        return [
            this.createMockData({
                overrides: {
                    category: "ChatMessage",
                    title: "Unread chat message",
                    isRead: false,
                },
            }),
            this.createMockData({
                overrides: {
                    category: "Mention",
                    title: "Read mention",
                    isRead: true,
                },
            }),
            this.createMockData({
                overrides: {
                    category: "System",
                    title: "Unread system update",
                    isRead: false,
                },
            }),
        ];
    }

    /**
     * Create chat message notification
     */
    createChatMessageNotification(fromUser: string, message: string, chatId?: string): Notification {
        return this.createMockData({
            overrides: {
                category: "ChatMessage",
                title: `New message from ${fromUser}`,
                description: message,
                link: chatId ? `/chat/${chatId}` : "/chat",
                isRead: false,
            },
        });
    }

    /**
     * Create mention notification
     */
    createMentionNotification(content: string, contextLink?: string): Notification {
        return this.createMockData({
            overrides: {
                category: "Mention",
                title: "You were mentioned",
                description: content,
                link: contextLink || "/notifications",
                isRead: false,
            },
        });
    }

    /**
     * Create award notification
     */
    createAwardNotification(badgeName: string, description: string): Notification {
        return this.createMockData({
            overrides: {
                category: "Award",
                title: "Achievement Unlocked!",
                description: `You earned the '${badgeName}' badge: ${description}`,
                link: "/profile/awards",
                imgLink: `/img/badges/${badgeName.toLowerCase().replace(/\s+/g, "-")}.png`,
                isRead: false,
            },
        });
    }

    /**
     * Create task completion notification
     */
    createTaskCompletionNotification(taskName: string, runId?: string): Notification {
        return this.createMockData({
            overrides: {
                category: "RunComplete",
                title: "Task completed successfully",
                description: `Your routine '${taskName}' has finished`,
                link: runId ? `/runs/${runId}` : "/runs",
                isRead: false,
            },
        });
    }

    /**
     * Create team invitation notification
     */
    createTeamInviteNotification(teamName: string, inviteId?: string): Notification {
        return this.createMockData({
            overrides: {
                category: "TeamInvite",
                title: "Team invitation",
                description: `You've been invited to join '${teamName}'`,
                link: inviteId ? `/teams/invitations/${inviteId}` : "/teams/invitations",
                isRead: false,
            },
        });
    }

    /**
     * Create system notification
     */
    createSystemNotification(title: string, description: string): Notification {
        return this.createMockData({
            overrides: {
                category: "System",
                title,
                description,
                link: "/settings",
                isRead: false,
            },
        });
    }

    /**
     * Create urgent notification
     */
    createUrgentNotification(): Notification {
        return this.createMockData({
            scenario: "complete",
            overrides: {
                category: "System",
                title: "URGENT: Security Alert",
                description: "Suspicious activity detected on your account. Please review your recent login activity and update your password immediately.",
                link: "/settings/security",
                imgLink: "/img/alert-icon.png",
                isRead: false,
            },
        });
    }

    /**
     * Create bulk update response
     */
    createBulkUpdateResponse(updatedNotifications: Notification[]) {
        return {
            data: updatedNotifications,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: generatePK().toString(),
                version: "1.0",
                links: {
                    self: "/api/notification/bulk-update",
                },
            },
        };
    }

    /**
     * Create notification count response
     */
    createNotificationCountResponse(unreadCount: number, totalCount: number) {
        return {
            data: {
                unreadCount,
                totalCount,
            },
            meta: {
                timestamp: new Date().toISOString(),
                requestId: generatePK().toString(),
                version: "1.0",
                links: {
                    self: "/api/notification/count",
                },
            },
        };
    }

    /**
     * Create bulk operation limit error response
     */
    createBulkOperationLimitErrorResponse(operationCount: number) {
        return this.createBusinessErrorResponse("limit", {
            resource: "bulk operations",
            limit: MAX_BULK_OPERATIONS,
            current: operationCount,
            message: `Cannot update more than ${MAX_BULK_OPERATIONS} notifications at once`,
        });
    }

    /**
     * Create rate limit error response
     */
    createRateLimitErrorResponse(limit = 100, windowMinutes = 15) {
        const retryAfterMs = windowMinutes * 60 * 1000;
        return this.createRateLimitErrorResponse(
            limit,
            0,
            new Date(Date.now() + retryAfterMs),
        );
    }
}

/**
 * Pre-configured notification response scenarios
 */
export const notificationResponseScenarios = {
    // Success scenarios
    createSuccess: (input?: Partial<NotificationCreateInput>) => {
        const factory = new NotificationResponseFactory();
        const defaultInput: NotificationCreateInput = {
            category: "System",
            title: "Test Notification",
            description: "This is a test notification",
            link: "/notifications",
            ...input,
        };
        return factory.createSuccessResponse(
            factory.createFromInput(defaultInput),
        );
    },

    findSuccess: (notification?: Notification) => {
        const factory = new NotificationResponseFactory();
        return factory.createSuccessResponse(
            notification || factory.createMockData(),
        );
    },

    findCompleteSuccess: () => {
        const factory = new NotificationResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({ scenario: "complete" }),
        );
    },

    updateSuccess: (existing?: Notification, updates?: Partial<NotificationUpdateInput>) => {
        const factory = new NotificationResponseFactory();
        const notification = existing || factory.createMockData({ scenario: "complete" });
        const input: NotificationUpdateInput = {
            id: notification.id,
            ...updates,
        };
        return factory.createSuccessResponse(
            factory.updateFromInput(notification, input),
        );
    },

    markAsReadSuccess: (notificationId?: string) => {
        const factory = new NotificationResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    id: notificationId,
                    isRead: true,
                },
            }),
        );
    },

    markAsUnreadSuccess: (notificationId?: string) => {
        const factory = new NotificationResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    id: notificationId,
                    isRead: false,
                },
            }),
        );
    },

    listSuccess: (notifications?: Notification[]) => {
        const factory = new NotificationResponseFactory();
        return factory.createPaginatedResponse(
            notifications || Array.from({ length: DEFAULT_COUNT }, () => factory.createMockData()),
            { page: 1, totalCount: notifications?.length || DEFAULT_COUNT },
        );
    },

    allCategoriesSuccess: () => {
        const factory = new NotificationResponseFactory();
        return factory.createPaginatedResponse(
            factory.createNotificationsForAllCategories(),
            { page: 1, totalCount: 9 },
        );
    },

    unreadOnlySuccess: () => {
        const factory = new NotificationResponseFactory();
        const notifications = factory.createNotificationsForAllCategories()
            .map(n => ({ ...n, isRead: false }));
        return factory.createPaginatedResponse(
            notifications,
            { page: 1, totalCount: notifications.length },
        );
    },

    mixedReadStatesSuccess: () => {
        const factory = new NotificationResponseFactory();
        return factory.createPaginatedResponse(
            factory.createNotificationsWithReadStates(),
            { page: 1, totalCount: 3 },
        );
    },

    categoryFilteredSuccess: (category: NotificationCategory) => {
        const factory = new NotificationResponseFactory();
        const notifications = factory.createNotificationsForAllCategories()
            .filter(n => n.category === category);
        return factory.createPaginatedResponse(
            notifications,
            { page: 1, totalCount: notifications.length },
        );
    },

    urgentNotificationSuccess: () => {
        const factory = new NotificationResponseFactory();
        return factory.createSuccessResponse(
            factory.createUrgentNotification(),
        );
    },

    chatNotificationsSuccess: () => {
        const factory = new NotificationResponseFactory();
        const notifications = [
            factory.createChatMessageNotification("Alice", "Hey there!", "chat_1"),
            factory.createChatMessageNotification("Bob", "How's the project going?", "chat_2"),
            factory.createMentionNotification("@you Check this out!", "/posts/123"),
        ];
        return factory.createPaginatedResponse(
            notifications,
            { page: 1, totalCount: notifications.length },
        );
    },

    countSuccess: (unreadCount = 5, totalCount = 20) => {
        const factory = new NotificationResponseFactory();
        return factory.createNotificationCountResponse(unreadCount, totalCount);
    },

    bulkUpdateSuccess: (notificationIds?: string[]) => {
        const factory = new NotificationResponseFactory();
        const ids = notificationIds || [generatePK().toString(), generatePK().toString(), generatePK().toString()];
        const updatedNotifications = ids.map(id =>
            factory.createMockData({ overrides: { id, isRead: true } }),
        );
        return factory.createBulkUpdateResponse(updatedNotifications);
    },

    markAllReadSuccess: () => {
        const factory = new NotificationResponseFactory();
        const notifications = factory.createNotificationsForAllCategories()
            .map(n => ({ ...n, isRead: true }));
        return factory.createBulkUpdateResponse(notifications);
    },

    // Error scenarios
    createValidationError: () => {
        const factory = new NotificationResponseFactory();
        return factory.createValidationErrorResponse({
            title: "Title is required",
            description: "Description must be 1000 characters or less",
            category: "Invalid notification category",
        });
    },

    notFoundError: (notificationId?: string) => {
        const factory = new NotificationResponseFactory();
        return factory.createNotFoundErrorResponse(
            notificationId || "non-existent-notification",
        );
    },

    permissionError: (operation?: string) => {
        const factory = new NotificationResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "update",
            ["notification:update"],
        );
    },

    bulkOperationLimitError: (operationCount = 150) => {
        const factory = new NotificationResponseFactory();
        return factory.createBulkOperationLimitErrorResponse(operationCount);
    },

    rateLimitError: () => {
        const factory = new NotificationResponseFactory();
        return factory.createRateLimitErrorResponse();
    },

    // MSW handlers
    handlers: {
        success: () => new NotificationResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate?: number) {
            return new NotificationResponseFactory().createMSWHandlers({ errorRate: errorRate ?? DEFAULT_ERROR_RATE });
        },
        withDelay: function createWithDelay(delay?: number) {
            return new NotificationResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
    },
};

// Export factory instance for direct use
export const notificationResponseFactory = new NotificationResponseFactory();
