/**
 * ReminderList API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for reminder list endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { http, HttpResponse, type HttpHandler } from 'msw';
import { 
    reminderListValidation,
    type ReminderList, 
    type ReminderListCreateInput, 
    type ReminderListUpdateInput,
    type Reminder,
    type ReminderItem,
    type ReminderCreateInput,
    type ReminderItemCreateInput
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
 * ReminderList API response factory
 */
export class ReminderListResponseFactory {
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
     * Create successful reminder list response
     */
    createSuccessResponse(reminderList: ReminderList): APIResponse<ReminderList> {
        return {
            data: reminderList,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: '1.0',
                links: {
                    self: `${this.baseUrl}/api/reminder-list/${reminderList.id}`,
                    related: {
                        reminders: `${this.baseUrl}/api/reminder-list/${reminderList.id}/reminders`
                    }
                }
            }
        };
    }
    
    /**
     * Create reminder list collection response
     */
    createReminderListsResponse(reminderLists: ReminderList[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<ReminderList> {
        const paginationData = pagination || {
            page: 1,
            pageSize: reminderLists.length,
            totalCount: reminderLists.length
        };
        
        return {
            data: reminderLists,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: '1.0',
                links: {
                    self: `${this.baseUrl}/api/reminder-list?page=${paginationData.page}&limit=${paginationData.pageSize}`
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
                path: '/api/reminder-list'
            }
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(reminderListId: string): APIErrorResponse {
        return {
            error: {
                code: 'REMINDER_LIST_NOT_FOUND',
                message: `Reminder list with ID '${reminderListId}' was not found`,
                details: {
                    reminderListId,
                    searchCriteria: { id: reminderListId }
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/reminder-list/${reminderListId}`
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
                message: `You do not have permission to ${operation} this reminder list`,
                details: {
                    operation,
                    requiredPermissions: ['reminder-list:write'],
                    userPermissions: ['reminder-list:read']
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: '/api/reminder-list'
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
                path: '/api/reminder-list'
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
                path: '/api/reminder-list'
            }
        };
    }
    
    /**
     * Create mock reminder item data
     */
    createMockReminderItem(overrides?: Partial<ReminderItem>): ReminderItem {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultItem: ReminderItem = {
            __typename: "ReminderItem",
            id,
            createdAt: now,
            updatedAt: now,
            name: "Sample task item",
            description: "Complete this task",
            dueDate: new Date(Date.now() + 86400000).toISOString(), // Tomorrow
            index: 0,
            isComplete: false,
            reminder: {} as Reminder // Will be populated by parent
        };
        
        return {
            ...defaultItem,
            ...overrides
        };
    }
    
    /**
     * Create mock reminder data
     */
    createMockReminder(overrides?: Partial<Reminder>): Reminder {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const reminderItems = [
            this.createMockReminderItem({ index: 0, name: "First task" }),
            this.createMockReminderItem({ index: 1, name: "Second task", isComplete: true }),
            this.createMockReminderItem({ index: 2, name: "Third task" })
        ];
        
        const defaultReminder: Reminder = {
            __typename: "Reminder",
            id,
            createdAt: now,
            updatedAt: now,
            name: "Sample reminder",
            description: "A reminder with multiple items",
            dueDate: new Date(Date.now() + 604800000).toISOString(), // Next week
            index: 0,
            isComplete: false,
            reminderItems,
            reminderList: {} as ReminderList // Will be populated by parent
        };
        
        // Set parent reference in items
        reminderItems.forEach(item => {
            item.reminder = defaultReminder;
        });
        
        return {
            ...defaultReminder,
            ...overrides
        };
    }
    
    /**
     * Create mock reminder list data
     */
    createMockReminderList(overrides?: Partial<ReminderList>): ReminderList {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const reminders = [
            this.createMockReminder({ index: 0, name: "Daily tasks" }),
            this.createMockReminder({ index: 1, name: "Weekly goals" }),
            this.createMockReminder({ index: 2, name: "Project milestones", isComplete: true })
        ];
        
        const defaultReminderList: ReminderList = {
            __typename: "ReminderList",
            id,
            createdAt: now,
            updatedAt: now,
            reminders
        };
        
        // Set parent reference in reminders
        reminders.forEach(reminder => {
            reminder.reminderList = defaultReminderList;
        });
        
        return {
            ...defaultReminderList,
            ...overrides
        };
    }
    
    /**
     * Create reminder list from API input
     */
    createReminderListFromInput(input: ReminderListCreateInput): ReminderList {
        const reminderList = this.createMockReminderList();
        reminderList.id = input.id;
        
        // Handle reminder creation
        if (input.remindersCreate && input.remindersCreate.length > 0) {
            reminderList.reminders = input.remindersCreate.map((reminderInput, index) => 
                this.createReminderFromInput(reminderInput, index)
            );
            
            // Set parent references
            reminderList.reminders.forEach(reminder => {
                reminder.reminderList = reminderList;
            });
        }
        
        return reminderList;
    }
    
    /**
     * Create reminder from API input
     */
    private createReminderFromInput(input: ReminderCreateInput, index: number): Reminder {
        const reminder = this.createMockReminder({
            id: input.id,
            name: input.name,
            description: input.description || undefined,
            dueDate: input.dueDate || undefined,
            index: input.index ?? index,
            isComplete: false
        });
        
        // Handle reminder items creation
        if (input.reminderItemsCreate && input.reminderItemsCreate.length > 0) {
            reminder.reminderItems = input.reminderItemsCreate.map((itemInput, itemIndex) => 
                this.createReminderItemFromInput(itemInput, itemIndex, reminder)
            );
        }
        
        return reminder;
    }
    
    /**
     * Create reminder item from API input
     */
    private createReminderItemFromInput(input: ReminderItemCreateInput, index: number, parentReminder: Reminder): ReminderItem {
        return this.createMockReminderItem({
            id: input.id,
            name: input.name,
            description: input.description || undefined,
            dueDate: input.dueDate || undefined,
            index: input.index ?? index,
            isComplete: input.isComplete ?? false,
            reminder: parentReminder
        });
    }
    
    /**
     * Create multiple reminder lists for different scenarios
     */
    createReminderListScenarios(): ReminderList[] {
        return [
            // Empty list
            this.createMockReminderList({
                reminders: []
            }),
            
            // Personal to-do list
            this.createMockReminderList({
                reminders: [
                    this.createMockReminder({
                        name: "Morning routine",
                        reminderItems: [
                            this.createMockReminderItem({ name: "Exercise", index: 0 }),
                            this.createMockReminderItem({ name: "Breakfast", index: 1 }),
                            this.createMockReminderItem({ name: "Review schedule", index: 2 })
                        ]
                    })
                ]
            }),
            
            // Project task list
            this.createMockReminderList({
                reminders: [
                    this.createMockReminder({
                        name: "Sprint 1 tasks",
                        dueDate: new Date(Date.now() + 1209600000).toISOString(), // 2 weeks
                        reminderItems: [
                            this.createMockReminderItem({ name: "Design API", isComplete: true }),
                            this.createMockReminderItem({ name: "Implement backend", isComplete: true }),
                            this.createMockReminderItem({ name: "Write tests" }),
                            this.createMockReminderItem({ name: "Documentation" })
                        ]
                    }),
                    this.createMockReminder({
                        name: "Sprint 2 planning",
                        reminderItems: []
                    })
                ]
            }),
            
            // Shopping list
            this.createMockReminderList({
                reminders: [
                    this.createMockReminder({
                        name: "Grocery shopping",
                        reminderItems: [
                            this.createMockReminderItem({ name: "Milk", isComplete: true }),
                            this.createMockReminderItem({ name: "Bread", isComplete: true }),
                            this.createMockReminderItem({ name: "Eggs" }),
                            this.createMockReminderItem({ name: "Vegetables" })
                        ]
                    })
                ]
            })
        ];
    }
    
    /**
     * Validate reminder list create input
     */
    async validateCreateInput(input: ReminderListCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await reminderListValidation.create.validate(input);
            return { valid: true };
        } catch (error: unknown) {
            const fieldErrors: Record<string, string> = {};
            
            if (error && typeof error === 'object' && 'inner' in error) {
                const validationError = error as { inner?: Array<{ path?: string; message: string }> };
                if (validationError.inner) {
                    validationError.inner.forEach((err) => {
                        if (err.path) {
                            fieldErrors[err.path] = err.message;
                        }
                    });
                }
            } else if (error && typeof error === 'object' && 'message' in error) {
                fieldErrors.general = String(error.message);
            }
            
            return {
                valid: false,
                errors: fieldErrors
            };
        }
    }
}

/**
 * MSW handlers factory for reminder list endpoints
 */
export class ReminderListMSWHandlers {
    private responseFactory: ReminderListResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new ReminderListResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all reminder list endpoints
     */
    createSuccessHandlers(): HttpHandler[] {
        return [
            // Create reminder list
            http.post(`${this.responseFactory['baseUrl']}/api/reminder-list`, async ({ request }) => {
                const body = await request.json() as ReminderListCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return HttpResponse.json(
                        this.responseFactory.createValidationErrorResponse(validation.errors || {}),
                        { status: 400 }
                    );
                }
                
                // Create reminder list
                const reminderList = this.responseFactory.createReminderListFromInput(body);
                const response = this.responseFactory.createSuccessResponse(reminderList);
                
                return HttpResponse.json(response, { status: 201 });
            }),
            
            // Get reminder list by ID
            http.get(`${this.responseFactory['baseUrl']}/api/reminder-list/:id`, ({ params }) => {
                const { id } = params;
                
                const reminderList = this.responseFactory.createMockReminderList({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(reminderList);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Update reminder list
            http.put(`${this.responseFactory['baseUrl']}/api/reminder-list/:id`, async ({ params, request }) => {
                const { id } = params;
                const body = await request.json() as ReminderListUpdateInput;
                
                const reminderList = this.responseFactory.createMockReminderList({ 
                    id: id as string,
                    updatedAt: new Date().toISOString()
                });
                
                // Handle updates
                if (body.remindersCreate) {
                    const newReminders = body.remindersCreate.map((r, i) => 
                        this.responseFactory['createReminderFromInput'](r, reminderList.reminders.length + i)
                    );
                    reminderList.reminders.push(...newReminders);
                }
                
                const response = this.responseFactory.createSuccessResponse(reminderList);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Delete reminder list
            http.delete(`${this.responseFactory['baseUrl']}/api/reminder-list/:id`, () => {
                return new HttpResponse(null, { status: 204 });
            }),
            
            // List reminder lists
            http.get(`${this.responseFactory['baseUrl']}/api/reminder-list`, ({ request }) => {
                const url = new URL(request.url);
                const page = parseInt(url.searchParams.get('page') || '1');
                const limit = parseInt(url.searchParams.get('limit') || '10');
                
                const reminderLists = this.responseFactory.createReminderListScenarios();
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedLists = reminderLists.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createReminderListsResponse(
                    paginatedLists,
                    {
                        page,
                        pageSize: limit,
                        totalCount: reminderLists.length
                    }
                );
                
                return HttpResponse.json(response, { status: 200 });
            })
        ];
    }
    
    /**
     * Create error handlers for testing error scenarios
     */
    createErrorHandlers(): HttpHandler[] {
        return [
            // Validation error
            http.post(`${this.responseFactory['baseUrl']}/api/reminder-list`, () => {
                return HttpResponse.json(
                    this.responseFactory.createValidationErrorResponse({
                        id: 'ID is required',
                        reminders: 'At least one reminder is required'
                    }),
                    { status: 400 }
                );
            }),
            
            // Not found error
            http.get(`${this.responseFactory['baseUrl']}/api/reminder-list/:id`, ({ params }) => {
                const { id } = params;
                return HttpResponse.json(
                    this.responseFactory.createNotFoundErrorResponse(id as string),
                    { status: 404 }
                );
            }),
            
            // Permission error
            http.post(`${this.responseFactory['baseUrl']}/api/reminder-list`, () => {
                return HttpResponse.json(
                    this.responseFactory.createPermissionErrorResponse('create'),
                    { status: 403 }
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory['baseUrl']}/api/reminder-list`, () => {
                return HttpResponse.json(
                    this.responseFactory.createServerErrorResponse(),
                    { status: 500 }
                );
            })
        ];
    }
    
    /**
     * Create loading simulation handlers
     */
    createLoadingHandlers(delay: number = 2000): HttpHandler[] {
        return [
            http.post(`${this.responseFactory['baseUrl']}/api/reminder-list`, async ({ request }) => {
                const body = await request.json() as ReminderListCreateInput;
                const reminderList = this.responseFactory.createReminderListFromInput(body);
                const response = this.responseFactory.createSuccessResponse(reminderList);
                
                // Simulate delay
                await new Promise(resolve => setTimeout(resolve, delay));
                
                return HttpResponse.json(response, { status: 201 });
            })
        ];
    }
    
    /**
     * Create network error handlers
     */
    createNetworkErrorHandlers(): HttpHandler[] {
        return [
            http.post(`${this.responseFactory['baseUrl']}/api/reminder-list`, () => {
                return HttpResponse.error();
            }),
            
            http.get(`${this.responseFactory['baseUrl']}/api/reminder-list/:id`, () => {
                return HttpResponse.error();
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
        response: unknown;
        delay?: number;
    }): HttpHandler {
        const { endpoint, method, status, response, delay } = config;
        const fullEndpoint = `${this.responseFactory['baseUrl']}${endpoint}`;
        
        const handler = http[method.toLowerCase() as keyof typeof http];
        
        return handler(fullEndpoint, async () => {
            if (delay) {
                await new Promise(resolve => setTimeout(resolve, delay));
            }
            
            return HttpResponse.json(response, { status });
        });
    }
}

/**
 * Pre-configured response scenarios
 */
export const reminderListResponseScenarios = {
    // Success scenarios
    createSuccess: (reminderList?: ReminderList) => {
        const factory = new ReminderListResponseFactory();
        return factory.createSuccessResponse(
            reminderList || factory.createMockReminderList()
        );
    },
    
    listSuccess: (reminderLists?: ReminderList[]) => {
        const factory = new ReminderListResponseFactory();
        return factory.createReminderListsResponse(
            reminderLists || factory.createReminderListScenarios()
        );
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new ReminderListResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                id: 'ID is required',
                reminders: 'Invalid reminder data'
            }
        );
    },
    
    notFoundError: (reminderListId?: string) => {
        const factory = new ReminderListResponseFactory();
        return factory.createNotFoundErrorResponse(
            reminderListId || 'non-existent-id'
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new ReminderListResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || 'create'
        );
    },
    
    serverError: () => {
        const factory = new ReminderListResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new ReminderListMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new ReminderListMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new ReminderListMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new ReminderListMSWHandlers().createNetworkErrorHandlers()
};

// Export factory instances for easy use
export const reminderListResponseFactory = new ReminderListResponseFactory();
export const reminderListMSWHandlers = new ReminderListMSWHandlers();