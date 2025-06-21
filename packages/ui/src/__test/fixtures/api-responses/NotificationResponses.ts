/**
 * Notification API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for notification endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { rest, type RestHandler } from 'msw';
import type { 
    Notification, 
    NotificationSearchInput,
    NotificationSortBy 
} from '@vrooli/shared';

/**
 * Standard API response wrapper
 */
export interface APIResponse<T> {
    data: T;
    meta: {
        timestamp: string;
        requestId: string;
        version: string;
        links?: {
            self?: string;
            related?: Record<string, string>;
        };
    };
}

/**
 * API error response structure
 */
export interface APIErrorResponse {
    error: {
        code: string;
        message: string;
        details?: Record<string, unknown>;
        timestamp: string;
        requestId: string;
        path: string;
    };
}

/**
 * Paginated response structure
 */
export interface PaginatedAPIResponse<T> extends APIResponse<T[]> {
    pagination: {
        page: number;
        pageSize: number;
        totalCount: number;
        totalPages: number;
        hasNextPage: boolean;
        hasPreviousPage: boolean;
    };
}

/**
 * Notification update operation for bulk actions
 */
export interface NotificationUpdateOperation {
    id: string;
    isRead?: boolean;
}

/**
 * Bulk notification update input
 */
export interface NotificationBulkUpdateInput {
    operations: NotificationUpdateOperation[];
}

/**
 * Notification categories used throughout the application
 */
export enum NotificationCategory {
    CHAT_MESSAGE = "ChatMessage",
    MENTION = "Mention", 
    AWARD = "Award",
    RUN_COMPLETE = "RunComplete",
    TEAM_INVITE = "TeamInvite",
    REPORT_RESPONSE = "ReportResponse",
    REMINDER = "Reminder",
    SYSTEM = "System",
    API_CREDIT = "ApiCredit"
}

/**
 * Notification API response factory
 */
export class NotificationResponseFactory {
    private readonly baseUrl: string;
    
    constructor(baseUrl: string = process.env.VITE_SERVER_URL || 'http://localhost:5329') {
        this.baseUrl = baseUrl;
    }
    
    /**
     * Generate unique request ID
     */
    private generateRequestId(): string {
        return `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Generate unique resource ID
     */
    private generateId(): string {
        return `notif_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Create successful notification response
     */
    createSuccessResponse(notification: Notification): APIResponse<Notification> {
        return {
            data: notification,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: '1.0',
                links: {
                    self: `${this.baseUrl}/api/notification/${notification.id}`,
                    related: {
                        markRead: `${this.baseUrl}/api/notification/${notification.id}/read`,
                        markUnread: `${this.baseUrl}/api/notification/${notification.id}/unread`
                    }
                }
            }
        };
    }
    
    /**
     * Create notification list response
     */
    createNotificationListResponse(notifications: Notification[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<Notification> {
        const paginationData = pagination || {
            page: 1,
            pageSize: notifications.length,
            totalCount: notifications.length
        };
        
        return {
            data: notifications,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: '1.0',
                links: {
                    self: `${this.baseUrl}/api/notification?page=${paginationData.page}&limit=${paginationData.pageSize}`
                }
            },
            pagination: {
                ...paginationData,
                totalPages: Math.ceil(paginationData.totalCount / paginationData.pageSize),
                hasNextPage: paginationData.page * paginationData.pageSize < paginationData.totalCount,
                hasPreviousPage: paginationData.page > 1
            }
        };
    }
    
    /**
     * Create bulk update response
     */
    createBulkUpdateResponse(updatedNotifications: Notification[]): APIResponse<Notification[]> {
        return {
            data: updatedNotifications,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: '1.0',
                links: {
                    self: `${this.baseUrl}/api/notification/bulk-update`
                }
            }
        };
    }
    
    /**
     * Create notification count response
     */
    createNotificationCountResponse(unreadCount: number, totalCount: number): APIResponse<{unreadCount: number; totalCount: number}> {
        return {
            data: {
                unreadCount,
                totalCount
            },
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: '1.0',
                links: {
                    self: `${this.baseUrl}/api/notification/count`
                }
            }
        };
    }
    
    /**
     * Create validation error response
     */
    createValidationErrorResponse(fieldErrors: Record<string, string>): APIErrorResponse {
        return {
            error: {
                code: 'VALIDATION_ERROR',
                message: 'The request contains invalid data',
                details: {
                    fieldErrors,
                    invalidFields: Object.keys(fieldErrors)
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: '/api/notification'
            }
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(notificationId: string): APIErrorResponse {
        return {
            error: {
                code: 'NOTIFICATION_NOT_FOUND',
                message: `Notification with ID '${notificationId}' was not found`,
                details: {
                    notificationId,
                    searchCriteria: { id: notificationId }
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/notification/${notificationId}`
            }
        };
    }
    
    /**
     * Create permission error response
     */
    createPermissionErrorResponse(operation: string): APIErrorResponse {
        return {
            error: {
                code: 'PERMISSION_DENIED',
                message: `You do not have permission to ${operation} this notification`,
                details: {
                    operation,
                    requiredPermissions: ['notification:read', 'notification:write'],
                    userPermissions: ['notification:read']
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: '/api/notification'
            }
        };
    }
    
    /**
     * Create network error response
     */
    createNetworkErrorResponse(): APIErrorResponse {
        return {
            error: {
                code: 'NETWORK_ERROR',
                message: 'Network request failed',
                details: {
                    reason: 'Connection timeout',
                    retryable: true,
                    retryAfter: 5000
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: '/api/notification'
            }
        };
    }
    
    /**
     * Create server error response
     */
    createServerErrorResponse(): APIErrorResponse {
        return {
            error: {
                code: 'INTERNAL_SERVER_ERROR',
                message: 'An unexpected server error occurred',
                details: {
                    errorId: `ERR_${Date.now()}`,
                    reportable: true,
                    retryable: true
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: '/api/notification'
            }
        };
    }
    
    /**
     * Create rate limit error response
     */
    createRateLimitErrorResponse(): APIErrorResponse {
        return {
            error: {
                code: 'RATE_LIMIT_EXCEEDED',
                message: 'Too many notification requests in a short period',
                details: {
                    limit: 100,
                    window: '15min',
                    retryAfter: 900000, // 15 minutes in ms
                    retryable: true
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: '/api/notification'
            }
        };
    }
    
    /**
     * Create mock notification data
     */
    createMockNotification(overrides?: Partial<Notification>): Notification {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultNotification: Notification = {
            __typename: "Notification",
            id,
            category: NotificationCategory.SYSTEM,
            title: "System Notification",
            description: "This is a test notification message",
            link: "/notifications",
            isRead: false,
            createdAt: now,
            imgLink: null
        };
        
        return {
            ...defaultNotification,
            ...overrides
        };
    }
    
    /**
     * Create notification for each category
     */
    createNotificationsForAllCategories(): Notification[] {
        return Object.values(NotificationCategory).map(category => 
            this.createMockNotification({
                category,
                title: this.getTitleForCategory(category),
                description: this.getDescriptionForCategory(category),
                link: this.getLinkForCategory(category),
                imgLink: this.getImageLinkForCategory(category)
            })
        );
    }
    
    /**
     * Create notifications with different read states
     */
    createNotificationsWithReadStates(): Notification[] {
        return [
            this.createMockNotification({
                category: NotificationCategory.CHAT_MESSAGE,
                title: "Unread chat message",
                isRead: false
            }),
            this.createMockNotification({
                category: NotificationCategory.MENTION,
                title: "Read mention",
                isRead: true
            }),
            this.createMockNotification({
                category: NotificationCategory.SYSTEM,
                title: "Unread system update",
                isRead: false
            })
        ];
    }
    
    /**
     * Create notifications with various priorities
     */
    createNotificationsWithPriorities(): Notification[] {
        return [
            this.createMockNotification({
                category: NotificationCategory.SYSTEM,
                title: "Low priority system message",
                description: "Minor update available"
            }),
            this.createMockNotification({
                category: NotificationCategory.REMINDER,
                title: "Medium priority reminder",
                description: "Meeting starting in 15 minutes"
            }),
            this.createMockNotification({
                category: NotificationCategory.TEAM_INVITE,
                title: "High priority team invitation",
                description: "Urgent: You've been invited to join critical project team"
            })
        ];
    }
    
    /**
     * Create chat message notification
     */
    createChatMessageNotification(fromUser: string, message: string, chatId?: string): Notification {
        return this.createMockNotification({
            category: NotificationCategory.CHAT_MESSAGE,
            title: `New message from ${fromUser}`,
            description: message,
            link: chatId ? `/chat/${chatId}` : '/chat',
            isRead: false
        });
    }
    
    /**
     * Create mention notification
     */
    createMentionNotification(content: string, contextLink?: string): Notification {
        return this.createMockNotification({
            category: NotificationCategory.MENTION,
            title: "You were mentioned",
            description: content,
            link: contextLink || '/notifications',
            isRead: false
        });
    }
    
    /**
     * Create award notification
     */
    createAwardNotification(badgeName: string, description: string): Notification {
        return this.createMockNotification({
            category: NotificationCategory.AWARD,
            title: "Achievement Unlocked!",
            description: `You earned the '${badgeName}' badge: ${description}`,
            link: '/profile/awards',
            imgLink: `/img/badges/${badgeName.toLowerCase().replace(/\s+/g, '-')}.png`,
            isRead: false
        });
    }
    
    /**
     * Create task completion notification
     */
    createTaskCompletionNotification(taskName: string, runId?: string): Notification {
        return this.createMockNotification({
            category: NotificationCategory.RUN_COMPLETE,
            title: "Task completed successfully",
            description: `Your routine '${taskName}' has finished`,
            link: runId ? `/run/${runId}` : '/runs',
            isRead: false
        });
    }
    
    /**
     * Create team invitation notification
     */
    createTeamInviteNotification(teamName: string, inviteId?: string): Notification {
        return this.createMockNotification({
            category: NotificationCategory.TEAM_INVITE,
            title: "Team invitation",
            description: `You've been invited to join '${teamName}'`,
            link: inviteId ? `/teams/invitations/${inviteId}` : '/teams/invitations',
            isRead: false
        });
    }
    
    /**
     * Create report response notification
     */
    createReportResponseNotification(message: string, reportId?: string): Notification {
        return this.createMockNotification({
            category: NotificationCategory.REPORT_RESPONSE,
            title: "Report update",
            description: message,
            link: reportId ? `/reports/${reportId}` : '/reports',
            isRead: false
        });
    }
    
    /**
     * Create reminder notification
     */
    createReminderNotification(message: string, reminderId?: string): Notification {
        return this.createMockNotification({
            category: NotificationCategory.REMINDER,
            title: "Reminder",
            description: message,
            link: reminderId ? `/reminders/${reminderId}` : '/calendar',
            isRead: false
        });
    }
    
    /**
     * Create system notification
     */
    createSystemNotification(title: string, description: string): Notification {
        return this.createMockNotification({
            category: NotificationCategory.SYSTEM,
            title,
            description,
            link: '/settings',
            isRead: false
        });
    }
    
    /**
     * Validate notification search input
     */
    validateSearchInput(input: NotificationSearchInput): {
        valid: boolean;
        errors?: Record<string, string>;
    } {
        const errors: Record<string, string> = {};
        
        // Validate pagination
        if (input.take !== undefined && (input.take < 1 || input.take > 100)) {
            errors.take = 'Take must be between 1 and 100';
        }
        
        // Validate search string
        if (input.searchString !== undefined && typeof input.searchString !== 'string') {
            errors.searchString = 'Search string must be a string';
        }
        
        // Validate user ID format
        if (input.userId !== undefined && (typeof input.userId !== 'string' || input.userId.length === 0)) {
            errors.userId = 'User ID must be a non-empty string';
        }
        
        // Validate sort by
        if (input.sortBy !== undefined && !Object.values(NotificationSortBy).includes(input.sortBy)) {
            errors.sortBy = 'Invalid sort option';
        }
        
        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined
        };
    }
    
    /**
     * Validate bulk update input
     */
    validateBulkUpdateInput(input: NotificationBulkUpdateInput): {
        valid: boolean;
        errors?: Record<string, string>;
    } {
        const errors: Record<string, string> = {};
        
        if (!input.operations || !Array.isArray(input.operations)) {
            errors.operations = 'Operations array is required';
            return { valid: false, errors };
        }
        
        if (input.operations.length === 0) {
            errors.operations = 'At least one operation is required';
        }
        
        if (input.operations.length > 100) {
            errors.operations = 'Cannot update more than 100 notifications at once';
        }
        
        input.operations.forEach((op, index) => {
            if (!op.id || typeof op.id !== 'string') {
                errors[`operations[${index}].id`] = 'Operation ID is required and must be a string';
            }
            
            if (op.isRead !== undefined && typeof op.isRead !== 'boolean') {
                errors[`operations[${index}].isRead`] = 'isRead must be a boolean';
            }
        });
        
        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined
        };
    }
    
    // Helper methods for creating category-specific content
    private getTitleForCategory(category: string): string {
        switch (category) {
            case NotificationCategory.CHAT_MESSAGE:
                return "New message from Alice";
            case NotificationCategory.MENTION:
                return "You were mentioned";
            case NotificationCategory.AWARD:
                return "Achievement Unlocked!";
            case NotificationCategory.RUN_COMPLETE:
                return "Task completed successfully";
            case NotificationCategory.TEAM_INVITE:
                return "Team invitation";
            case NotificationCategory.REPORT_RESPONSE:
                return "Report update";
            case NotificationCategory.REMINDER:
                return "Reminder";
            case NotificationCategory.SYSTEM:
                return "System Update";
            case NotificationCategory.API_CREDIT:
                return "API Credits Update";
            default:
                return "Notification";
        }
    }
    
    private getDescriptionForCategory(category: string): string {
        switch (category) {
            case NotificationCategory.CHAT_MESSAGE:
                return "Hey, how are you doing?";
            case NotificationCategory.MENTION:
                return "@you Check out this new feature!";
            case NotificationCategory.AWARD:
                return "You earned the 'Contributor' badge for helping others";
            case NotificationCategory.RUN_COMPLETE:
                return "Your routine 'Data Analysis' has finished";
            case NotificationCategory.TEAM_INVITE:
                return "You've been invited to join 'AI Research Team'";
            case NotificationCategory.REPORT_RESPONSE:
                return "Your report has been reviewed and approved";
            case NotificationCategory.REMINDER:
                return "Meeting starts in 15 minutes";
            case NotificationCategory.SYSTEM:
                return "New features are now available";
            case NotificationCategory.API_CREDIT:
                return "Your API credits have been updated";
            default:
                return "This is a notification message";
        }
    }
    
    private getLinkForCategory(category: string): string | null {
        switch (category) {
            case NotificationCategory.CHAT_MESSAGE:
                return "/chat";
            case NotificationCategory.MENTION:
                return "/notifications";
            case NotificationCategory.AWARD:
                return "/profile/awards";
            case NotificationCategory.RUN_COMPLETE:
                return "/runs";
            case NotificationCategory.TEAM_INVITE:
                return "/teams/invitations";
            case NotificationCategory.REPORT_RESPONSE:
                return "/reports";
            case NotificationCategory.REMINDER:
                return "/calendar";
            case NotificationCategory.SYSTEM:
                return "/settings";
            case NotificationCategory.API_CREDIT:
                return "/settings/billing";
            default:
                return "/notifications";
        }
    }
    
    private getImageLinkForCategory(category: string): string | null {
        switch (category) {
            case NotificationCategory.AWARD:
                return "/img/badges/contributor.png";
            case NotificationCategory.SYSTEM:
                return "/img/system-update.png";
            default:
                return null;
        }
    }
}

/**
 * MSW handlers factory for notification endpoints
 */
export class NotificationMSWHandlers {
    private responseFactory: NotificationResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new NotificationResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all notification endpoints
     */
    createSuccessHandlers(): RestHandler[] {
        return [
            // Get notification by ID
            rest.get(`${this.responseFactory['baseUrl']}/api/notification/:id`, (req, res, ctx) => {
                const { id } = req.params;
                
                const notification = this.responseFactory.createMockNotification({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(notification);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Update notification (mark as read/unread)
            rest.put(`${this.responseFactory['baseUrl']}/api/notification/:id`, (req, res, ctx) => {
                const { id } = req.params;
                
                const notification = this.responseFactory.createMockNotification({ 
                    id: id as string,
                    isRead: true
                });
                
                const response = this.responseFactory.createSuccessResponse(notification);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Mark notification as read
            rest.patch(`${this.responseFactory['baseUrl']}/api/notification/:id/read`, (req, res, ctx) => {
                const { id } = req.params;
                
                const notification = this.responseFactory.createMockNotification({ 
                    id: id as string,
                    isRead: true
                });
                
                const response = this.responseFactory.createSuccessResponse(notification);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Mark notification as unread
            rest.patch(`${this.responseFactory['baseUrl']}/api/notification/:id/unread`, (req, res, ctx) => {
                const { id } = req.params;
                
                const notification = this.responseFactory.createMockNotification({ 
                    id: id as string,
                    isRead: false
                });
                
                const response = this.responseFactory.createSuccessResponse(notification);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Delete notification
            rest.delete(`${this.responseFactory['baseUrl']}/api/notification/:id`, (req, res, ctx) => {
                return res(ctx.status(204));
            }),
            
            // List notifications
            rest.get(`${this.responseFactory['baseUrl']}/api/notification`, (req, res, ctx) => {
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get('page') || '1');
                const limit = parseInt(url.searchParams.get('limit') || '10');
                const searchString = url.searchParams.get('searchString');
                const category = url.searchParams.get('category');
                const isRead = url.searchParams.get('isRead');
                
                let notifications = this.responseFactory.createNotificationsForAllCategories();
                
                // Filter by category if specified
                if (category) {
                    notifications = notifications.filter(n => n.category === category);
                }
                
                // Filter by read status if specified
                if (isRead !== null) {
                    const readStatus = isRead === 'true';
                    notifications = notifications.filter(n => n.isRead === readStatus);
                }
                
                // Filter by search string if specified
                if (searchString) {
                    const search = searchString.toLowerCase();
                    notifications = notifications.filter(n => 
                        n.title.toLowerCase().includes(search) ||
                        (n.description && n.description.toLowerCase().includes(search))
                    );
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedNotifications = notifications.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createNotificationListResponse(
                    paginatedNotifications,
                    {
                        page,
                        pageSize: limit,
                        totalCount: notifications.length
                    }
                );
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Bulk update notifications
            rest.post(`${this.responseFactory['baseUrl']}/api/notification/bulk-update`, async (req, res, ctx) => {
                const body = await req.json() as NotificationBulkUpdateInput;
                
                // Validate input
                const validation = this.responseFactory.validateBulkUpdateInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {}))
                    );
                }
                
                // Create updated notifications
                const updatedNotifications = body.operations.map(op => 
                    this.responseFactory.createMockNotification({
                        id: op.id,
                        isRead: op.isRead !== undefined ? op.isRead : false
                    })
                );
                
                const response = this.responseFactory.createBulkUpdateResponse(updatedNotifications);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Mark all as read
            rest.post(`${this.responseFactory['baseUrl']}/api/notification/mark-all-read`, (req, res, ctx) => {
                const notifications = this.responseFactory.createNotificationsForAllCategories()
                    .map(n => ({ ...n, isRead: true }));
                
                const response = this.responseFactory.createBulkUpdateResponse(notifications);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Get notification count
            rest.get(`${this.responseFactory['baseUrl']}/api/notification/count`, (req, res, ctx) => {
                const totalCount = 25;
                const unreadCount = 8;
                
                const response = this.responseFactory.createNotificationCountResponse(unreadCount, totalCount);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            })
        ];
    }
    
    /**
     * Create error handlers for testing error scenarios
     */
    createErrorHandlers(): RestHandler[] {
        return [
            // Validation error for bulk update
            rest.post(`${this.responseFactory['baseUrl']}/api/notification/bulk-update`, (req, res, ctx) => {
                return res(
                    ctx.status(400),
                    ctx.json(this.responseFactory.createValidationErrorResponse({
                        operations: 'Operations array is required',
                        'operations[0].id': 'Operation ID is required'
                    }))
                );
            }),
            
            // Not found error
            rest.get(`${this.responseFactory['baseUrl']}/api/notification/:id`, (req, res, ctx) => {
                const { id } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse(id as string))
                );
            }),
            
            // Permission error
            rest.put(`${this.responseFactory['baseUrl']}/api/notification/:id`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse('update'))
                );
            }),
            
            // Rate limit error
            rest.get(`${this.responseFactory['baseUrl']}/api/notification`, (req, res, ctx) => {
                return res(
                    ctx.status(429),
                    ctx.json(this.responseFactory.createRateLimitErrorResponse())
                );
            }),
            
            // Server error
            rest.post(`${this.responseFactory['baseUrl']}/api/notification/mark-all-read`, (req, res, ctx) => {
                return res(
                    ctx.status(500),
                    ctx.json(this.responseFactory.createServerErrorResponse())
                );
            })
        ];
    }
    
    /**
     * Create loading simulation handlers
     */
    createLoadingHandlers(delay: number = 2000): RestHandler[] {
        return [
            rest.get(`${this.responseFactory['baseUrl']}/api/notification`, (req, res, ctx) => {
                const notifications = this.responseFactory.createNotificationsForAllCategories();
                const response = this.responseFactory.createNotificationListResponse(notifications);
                
                return res(
                    ctx.delay(delay),
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            rest.post(`${this.responseFactory['baseUrl']}/api/notification/bulk-update`, async (req, res, ctx) => {
                const body = await req.json() as NotificationBulkUpdateInput;
                const updatedNotifications = body.operations.map(op => 
                    this.responseFactory.createMockNotification({
                        id: op.id,
                        isRead: op.isRead !== undefined ? op.isRead : false
                    })
                );
                
                const response = this.responseFactory.createBulkUpdateResponse(updatedNotifications);
                
                return res(
                    ctx.delay(delay),
                    ctx.status(200),
                    ctx.json(response)
                );
            })
        ];
    }
    
    /**
     * Create network error handlers
     */
    createNetworkErrorHandlers(): RestHandler[] {
        return [
            rest.get(`${this.responseFactory['baseUrl']}/api/notification`, (req, res, ctx) => {
                return res.networkError('Network connection failed');
            }),
            
            rest.get(`${this.responseFactory['baseUrl']}/api/notification/:id`, (req, res, ctx) => {
                return res.networkError('Connection timeout');
            }),
            
            rest.post(`${this.responseFactory['baseUrl']}/api/notification/bulk-update`, (req, res, ctx) => {
                return res.networkError('Request timeout');
            })
        ];
    }
    
    /**
     * Create custom handler with specific configuration
     */
    createCustomHandler(config: {
        endpoint: string;
        method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
        status: number;
        response: unknown;
        delay?: number;
    }): RestHandler {
        const { endpoint, method, status, response, delay } = config;
        const fullEndpoint = `${this.responseFactory['baseUrl']}${endpoint}`;
        
        return rest[method.toLowerCase() as keyof typeof rest](fullEndpoint, (req, res, ctx) => {
            const responseCtx = [ctx.status(status), ctx.json(response)];
            
            if (delay) {
                responseCtx.unshift(ctx.delay(delay));
            }
            
            return res(...responseCtx);
        });
    }
}

/**
 * Pre-configured response scenarios
 */
export const notificationResponseScenarios = {
    // Success scenarios
    singleNotification: (notification?: Notification) => {
        const factory = new NotificationResponseFactory();
        return factory.createSuccessResponse(
            notification || factory.createMockNotification()
        );
    },
    
    notificationList: (notifications?: Notification[]) => {
        const factory = new NotificationResponseFactory();
        return factory.createNotificationListResponse(
            notifications || factory.createNotificationsForAllCategories()
        );
    },
    
    unreadNotifications: () => {
        const factory = new NotificationResponseFactory();
        const notifications = factory.createNotificationsForAllCategories()
            .map(n => ({ ...n, isRead: false }));
        return factory.createNotificationListResponse(notifications);
    },
    
    mixedReadStates: () => {
        const factory = new NotificationResponseFactory();
        return factory.createNotificationListResponse(
            factory.createNotificationsWithReadStates()
        );
    },
    
    chatNotifications: () => {
        const factory = new NotificationResponseFactory();
        const notifications = [
            factory.createChatMessageNotification("Alice", "Hey there!"),
            factory.createChatMessageNotification("Bob", "How's the project going?"),
            factory.createMentionNotification("@you Check this out!")
        ];
        return factory.createNotificationListResponse(notifications);
    },
    
    systemNotifications: () => {
        const factory = new NotificationResponseFactory();
        const notifications = [
            factory.createSystemNotification("System Update", "New features available"),
            factory.createSystemNotification("Maintenance", "Scheduled maintenance tonight"),
            factory.createSystemNotification("Security", "Password policy updated")
        ];
        return factory.createNotificationListResponse(notifications);
    },
    
    achievementNotifications: () => {
        const factory = new NotificationResponseFactory();
        const notifications = [
            factory.createAwardNotification("First Steps", "Welcome to the platform!"),
            factory.createAwardNotification("Contributor", "Thanks for helping others"),
            factory.createAwardNotification("Power User", "You're mastering the features")
        ];
        return factory.createNotificationListResponse(notifications);
    },
    
    taskNotifications: () => {
        const factory = new NotificationResponseFactory();
        const notifications = [
            factory.createTaskCompletionNotification("Data Analysis"),
            factory.createTaskCompletionNotification("Image Processing"),
            factory.createTaskCompletionNotification("Report Generation")
        ];
        return factory.createNotificationListResponse(notifications);
    },
    
    notificationCounts: (unread?: number, total?: number) => {
        const factory = new NotificationResponseFactory();
        return factory.createNotificationCountResponse(unread || 5, total || 20);
    },
    
    bulkUpdateSuccess: (notificationIds?: string[]) => {
        const factory = new NotificationResponseFactory();
        const ids = notificationIds || ['notif_1', 'notif_2', 'notif_3'];
        const notifications = ids.map(id => 
            factory.createMockNotification({ id, isRead: true })
        );
        return factory.createBulkUpdateResponse(notifications);
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new NotificationResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                operations: 'Operations array is required',
                'operations[0].id': 'Operation ID is required'
            }
        );
    },
    
    notFoundError: (notificationId?: string) => {
        const factory = new NotificationResponseFactory();
        return factory.createNotFoundErrorResponse(
            notificationId || 'non-existent-id'
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new NotificationResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || 'update'
        );
    },
    
    rateLimitError: () => {
        const factory = new NotificationResponseFactory();
        return factory.createRateLimitErrorResponse();
    },
    
    serverError: () => {
        const factory = new NotificationResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new NotificationMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new NotificationMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new NotificationMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new NotificationMSWHandlers().createNetworkErrorHandlers()
};

// Export factory instances for easy use
export const notificationResponseFactory = new NotificationResponseFactory();
export const notificationMSWHandlers = new NotificationMSWHandlers();