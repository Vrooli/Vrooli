/**
 * ReminderItem API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for reminder item endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { rest, type RestHandler } from 'msw';
import type { 
    ReminderItem, 
    ReminderItemCreateInput, 
    ReminderItemUpdateInput,
    Reminder,
    ReminderList,
    User
} from '@vrooli/shared';
import { 
    reminderItemValidation
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
        details?: Record<string, any>;
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
 * ReminderItem API response factory
 */
export class ReminderItemResponseFactory {
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
        return `${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Create successful reminder item response
     */
    createSuccessResponse(reminderItem: ReminderItem): APIResponse<ReminderItem> {
        return {
            data: reminderItem,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: '1.0',
                links: {
                    self: `${this.baseUrl}/api/reminder-item/${reminderItem.id}`,
                    related: {
                        reminder: `${this.baseUrl}/api/reminder/${reminderItem.reminder.id}`,
                        reminderList: `${this.baseUrl}/api/reminder-list/${reminderItem.reminder.reminderList.id}`
                    }
                }
            }
        };
    }
    
    /**
     * Create reminder item list response
     */
    createReminderItemListResponse(reminderItems: ReminderItem[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<ReminderItem> {
        const paginationData = pagination || {
            page: 1,
            pageSize: reminderItems.length,
            totalCount: reminderItems.length
        };
        
        return {
            data: reminderItems,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: '1.0',
                links: {
                    self: `${this.baseUrl}/api/reminder-item?page=${paginationData.page}&limit=${paginationData.pageSize}`
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
                path: '/api/reminder-item'
            }
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(reminderItemId: string): APIErrorResponse {
        return {
            error: {
                code: 'REMINDER_ITEM_NOT_FOUND',
                message: `Reminder item with ID '${reminderItemId}' was not found`,
                details: {
                    reminderItemId,
                    searchCriteria: { id: reminderItemId }
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/reminder-item/${reminderItemId}`
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
                message: `You do not have permission to ${operation} this reminder item`,
                details: {
                    operation,
                    requiredPermissions: ['reminder:write'],
                    userPermissions: ['reminder:read']
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: '/api/reminder-item'
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
                path: '/api/reminder-item'
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
                path: '/api/reminder-item'
            }
        };
    }
    
    /**
     * Create mock user data
     */
    private createMockUser(overrides?: Partial<User>): User {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        return {
            __typename: "User",
            id: `user_${id}`,
            handle: "testuser",
            name: "Test User",
            createdAt: now,
            updatedAt: now,
            isBot: false,
            isPrivate: false,
            profileImage: null,
            bannerImage: null,
            premium: false,
            premiumExpiration: null,
            roles: [],
            wallets: [],
            translations: [],
            translationsCount: 0,
            you: {
                __typename: "UserYou",
                isBlocked: false,
                isBlockedByYou: false,
                canDelete: false,
                canReport: false,
                canUpdate: false,
                isBookmarked: false,
                isReacted: false,
                reactionSummary: {
                    __typename: "ReactionSummary",
                    emotion: null,
                    count: 0
                }
            },
            ...overrides
        };
    }
    
    /**
     * Create mock reminder list data
     */
    private createMockReminderList(overrides?: Partial<ReminderList>): ReminderList {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        return {
            __typename: "ReminderList",
            id: `reminderlist_${id}`,
            createdAt: now,
            updatedAt: now,
            reminders: [],
            ...overrides
        };
    }
    
    /**
     * Create mock reminder data
     */
    private createMockReminder(overrides?: Partial<Reminder>): Reminder {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        return {
            __typename: "Reminder",
            id: `reminder_${id}`,
            createdAt: now,
            updatedAt: now,
            name: "Test Reminder",
            description: "A test reminder",
            dueDate: null,
            index: 0,
            isComplete: false,
            reminderItems: [],
            reminderList: this.createMockReminderList(),
            ...overrides
        };
    }
    
    /**
     * Create mock reminder item data
     */
    createMockReminderItem(overrides?: Partial<ReminderItem>): ReminderItem {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultReminderItem: ReminderItem = {
            __typename: "ReminderItem",
            id,
            createdAt: now,
            updatedAt: now,
            name: "Test Task",
            description: "A test task to complete",
            dueDate: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(), // 7 days from now
            index: 0,
            isComplete: false,
            reminder: this.createMockReminder()
        };
        
        return {
            ...defaultReminderItem,
            ...overrides
        };
    }
    
    /**
     * Create reminder item from API input
     */
    createReminderItemFromInput(input: ReminderItemCreateInput): ReminderItem {
        const reminderItem = this.createMockReminderItem();
        
        // Update reminder item based on input
        reminderItem.id = input.id;
        reminderItem.name = input.name;
        reminderItem.description = input.description || null;
        reminderItem.dueDate = input.dueDate || null;
        reminderItem.index = input.index;
        reminderItem.isComplete = input.isComplete || false;
        
        // Connect to reminder
        reminderItem.reminder.id = input.reminderConnect;
        
        return reminderItem;
    }
    
    /**
     * Create multiple reminder items with different scenarios
     */
    createReminderItemScenarios(): ReminderItem[] {
        const now = new Date();
        const reminder = this.createMockReminder();
        
        return [
            // Overdue task
            this.createMockReminderItem({
                name: "Overdue Task",
                description: "This task is overdue",
                dueDate: new Date(now.getTime() - 2 * 24 * 60 * 60 * 1000).toISOString(),
                isComplete: false,
                index: 0,
                reminder
            }),
            
            // Due today
            this.createMockReminderItem({
                name: "Today's Task",
                description: "This task is due today",
                dueDate: now.toISOString(),
                isComplete: false,
                index: 1,
                reminder
            }),
            
            // Due this week
            this.createMockReminderItem({
                name: "Weekly Task",
                description: "Task due this week",
                dueDate: new Date(now.getTime() + 3 * 24 * 60 * 60 * 1000).toISOString(),
                isComplete: false,
                index: 2,
                reminder
            }),
            
            // No due date
            this.createMockReminderItem({
                name: "Ongoing Task",
                description: "No specific deadline",
                dueDate: null,
                isComplete: false,
                index: 3,
                reminder
            }),
            
            // Completed task
            this.createMockReminderItem({
                name: "Completed Task",
                description: "This task has been completed",
                dueDate: new Date(now.getTime() - 1 * 24 * 60 * 60 * 1000).toISOString(),
                isComplete: true,
                index: 4,
                reminder
            }),
            
            // High priority (low index)
            this.createMockReminderItem({
                name: "High Priority Task",
                description: "Important task with low index",
                dueDate: new Date(now.getTime() + 1 * 24 * 60 * 60 * 1000).toISOString(),
                isComplete: false,
                index: 0,
                reminder
            })
        ];
    }
    
    /**
     * Validate reminder item create input
     */
    async validateCreateInput(input: ReminderItemCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await reminderItemValidation.create.validate(input);
            return { valid: true };
        } catch (error: any) {
            const fieldErrors: Record<string, string> = {};
            
            if (error.inner) {
                error.inner.forEach((err: any) => {
                    if (err.path) {
                        fieldErrors[err.path] = err.message;
                    }
                });
            } else if (error.message) {
                fieldErrors.general = error.message;
            }
            
            return {
                valid: false,
                errors: fieldErrors
            };
        }
    }
}

/**
 * MSW handlers factory for reminder item endpoints
 */
export class ReminderItemMSWHandlers {
    private responseFactory: ReminderItemResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new ReminderItemResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all reminder item endpoints
     */
    createSuccessHandlers(): RestHandler[] {
        return [
            // Create reminder item
            rest.post(`${this.responseFactory['baseUrl']}/api/reminder-item`, async (req, res, ctx) => {
                const body = await req.json() as ReminderItemCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {}))
                    );
                }
                
                // Create reminder item
                const reminderItem = this.responseFactory.createReminderItemFromInput(body);
                const response = this.responseFactory.createSuccessResponse(reminderItem);
                
                return res(
                    ctx.status(201),
                    ctx.json(response)
                );
            }),
            
            // Get reminder item by ID
            rest.get(`${this.responseFactory['baseUrl']}/api/reminder-item/:id`, (req, res, ctx) => {
                const { id } = req.params;
                
                const reminderItem = this.responseFactory.createMockReminderItem({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(reminderItem);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Update reminder item
            rest.put(`${this.responseFactory['baseUrl']}/api/reminder-item/:id`, async (req, res, ctx) => {
                const { id } = req.params;
                const body = await req.json() as ReminderItemUpdateInput;
                
                const reminderItem = this.responseFactory.createMockReminderItem({ 
                    id: id as string,
                    updatedAt: new Date().toISOString(),
                    ...body
                });
                
                const response = this.responseFactory.createSuccessResponse(reminderItem);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Delete reminder item
            rest.delete(`${this.responseFactory['baseUrl']}/api/reminder-item/:id`, (req, res, ctx) => {
                return res(ctx.status(204));
            }),
            
            // List reminder items
            rest.get(`${this.responseFactory['baseUrl']}/api/reminder-item`, (req, res, ctx) => {
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get('page') || '1');
                const limit = parseInt(url.searchParams.get('limit') || '10');
                const reminderId = url.searchParams.get('reminderId');
                const isComplete = url.searchParams.get('isComplete');
                const sortBy = url.searchParams.get('sortBy') || 'index';
                
                let reminderItems = this.responseFactory.createReminderItemScenarios();
                
                // Filter by reminder if specified
                if (reminderId) {
                    reminderItems = reminderItems.filter(item => item.reminder.id === reminderId);
                }
                
                // Filter by completion status
                if (isComplete !== null) {
                    const completeStatus = isComplete === 'true';
                    reminderItems = reminderItems.filter(item => item.isComplete === completeStatus);
                }
                
                // Sort items
                reminderItems.sort((a, b) => {
                    switch (sortBy) {
                        case 'dueDate':
                            if (!a.dueDate) return 1;
                            if (!b.dueDate) return -1;
                            return new Date(a.dueDate).getTime() - new Date(b.dueDate).getTime();
                        case 'name':
                            return a.name.localeCompare(b.name);
                        case 'index':
                        default:
                            return a.index - b.index;
                    }
                });
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedItems = reminderItems.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createReminderItemListResponse(
                    paginatedItems,
                    {
                        page,
                        pageSize: limit,
                        totalCount: reminderItems.length
                    }
                );
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Toggle completion status
            rest.post(`${this.responseFactory['baseUrl']}/api/reminder-item/:id/toggle-complete`, (req, res, ctx) => {
                const { id } = req.params;
                
                const reminderItem = this.responseFactory.createMockReminderItem({ 
                    id: id as string,
                    isComplete: true,
                    updatedAt: new Date().toISOString()
                });
                
                const response = this.responseFactory.createSuccessResponse(reminderItem);
                
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
            // Validation error
            rest.post(`${this.responseFactory['baseUrl']}/api/reminder-item`, (req, res, ctx) => {
                return res(
                    ctx.status(400),
                    ctx.json(this.responseFactory.createValidationErrorResponse({
                        name: 'Task name is required',
                        reminderConnect: 'Must be connected to a reminder'
                    }))
                );
            }),
            
            // Not found error
            rest.get(`${this.responseFactory['baseUrl']}/api/reminder-item/:id`, (req, res, ctx) => {
                const { id } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse(id as string))
                );
            }),
            
            // Permission error
            rest.put(`${this.responseFactory['baseUrl']}/api/reminder-item/:id`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse('update'))
                );
            }),
            
            // Server error
            rest.post(`${this.responseFactory['baseUrl']}/api/reminder-item`, (req, res, ctx) => {
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
            rest.post(`${this.responseFactory['baseUrl']}/api/reminder-item`, async (req, res, ctx) => {
                const body = await req.json() as ReminderItemCreateInput;
                const reminderItem = this.responseFactory.createReminderItemFromInput(body);
                const response = this.responseFactory.createSuccessResponse(reminderItem);
                
                return res(
                    ctx.delay(delay),
                    ctx.status(201),
                    ctx.json(response)
                );
            }),
            
            rest.put(`${this.responseFactory['baseUrl']}/api/reminder-item/:id`, async (req, res, ctx) => {
                const { id } = req.params;
                const body = await req.json() as ReminderItemUpdateInput;
                
                const reminderItem = this.responseFactory.createMockReminderItem({ 
                    id: id as string,
                    updatedAt: new Date().toISOString(),
                    ...body
                });
                
                const response = this.responseFactory.createSuccessResponse(reminderItem);
                
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
            rest.post(`${this.responseFactory['baseUrl']}/api/reminder-item`, (req, res, ctx) => {
                return res.networkError('Network connection failed');
            }),
            
            rest.get(`${this.responseFactory['baseUrl']}/api/reminder-item/:id`, (req, res, ctx) => {
                return res.networkError('Connection timeout');
            }),
            
            rest.put(`${this.responseFactory['baseUrl']}/api/reminder-item/:id`, (req, res, ctx) => {
                return res.networkError('Request timeout');
            })
        ];
    }
    
    /**
     * Create custom handler with specific configuration
     */
    createCustomHandler(config: {
        endpoint: string;
        method: 'GET' | 'POST' | 'PUT' | 'DELETE';
        status: number;
        response: any;
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
export const reminderItemResponseScenarios = {
    // Success scenarios
    createSuccess: (reminderItem?: ReminderItem) => {
        const factory = new ReminderItemResponseFactory();
        return factory.createSuccessResponse(
            reminderItem || factory.createMockReminderItem()
        );
    },
    
    listSuccess: (reminderItems?: ReminderItem[]) => {
        const factory = new ReminderItemResponseFactory();
        return factory.createReminderItemListResponse(
            reminderItems || factory.createReminderItemScenarios()
        );
    },
    
    toggleCompleteSuccess: (reminderItemId: string, isComplete: boolean = true) => {
        const factory = new ReminderItemResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockReminderItem({ 
                id: reminderItemId, 
                isComplete,
                updatedAt: new Date().toISOString()
            })
        );
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new ReminderItemResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                name: 'Task name is required',
                reminderConnect: 'Must be connected to a reminder'
            }
        );
    },
    
    notFoundError: (reminderItemId?: string) => {
        const factory = new ReminderItemResponseFactory();
        return factory.createNotFoundErrorResponse(
            reminderItemId || 'non-existent-id'
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new ReminderItemResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || 'update'
        );
    },
    
    serverError: () => {
        const factory = new ReminderItemResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // Specific scenarios
    overdueItems: () => {
        const factory = new ReminderItemResponseFactory();
        const now = new Date();
        const items = [
            factory.createMockReminderItem({
                name: "Urgent Task",
                dueDate: new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000).toISOString(),
                isComplete: false
            }),
            factory.createMockReminderItem({
                name: "Overdue Project",
                dueDate: new Date(now.getTime() - 3 * 24 * 60 * 60 * 1000).toISOString(),
                isComplete: false
            })
        ];
        return factory.createReminderItemListResponse(items);
    },
    
    todayItems: () => {
        const factory = new ReminderItemResponseFactory();
        const today = new Date();
        today.setHours(12, 0, 0, 0);
        
        const items = [
            factory.createMockReminderItem({
                name: "Morning Task",
                dueDate: today.toISOString(),
                isComplete: false
            }),
            factory.createMockReminderItem({
                name: "Afternoon Meeting",
                dueDate: new Date(today.getTime() + 4 * 60 * 60 * 1000).toISOString(),
                isComplete: false
            })
        ];
        return factory.createReminderItemListResponse(items);
    },
    
    completedItems: () => {
        const factory = new ReminderItemResponseFactory();
        const items = factory.createReminderItemScenarios()
            .filter(item => item.isComplete);
        return factory.createReminderItemListResponse(items);
    },
    
    // MSW handlers
    successHandlers: () => new ReminderItemMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new ReminderItemMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new ReminderItemMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new ReminderItemMSWHandlers().createNetworkErrorHandlers()
};

// Export factory instances for easy use
export const reminderItemResponseFactory = new ReminderItemResponseFactory();
export const reminderItemMSWHandlers = new ReminderItemMSWHandlers();