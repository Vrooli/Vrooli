/**
 * Transfer API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for transfer endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { rest, type RestHandler } from 'msw';
import type { 
    Transfer, 
    TransferCreateInput, 
    TransferUpdateInput,
    TransferStatus
} from '@vrooli/shared';
import { 
    transferValidation,
    TransferStatus as TransferStatusEnum 
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
 * Transfer API response factory
 */
export class TransferResponseFactory {
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
     * Generate unique transfer ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Create successful transfer response
     */
    createSuccessResponse(transfer: Transfer): APIResponse<Transfer> {
        return {
            data: transfer,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: '1.0',
                links: {
                    self: `${this.baseUrl}/api/transfer/${transfer.id}`,
                    related: {
                        from: `${this.baseUrl}/api/user/${transfer.from?.id}`,
                        to: `${this.baseUrl}/api/user/${transfer.to?.id}`,
                        object: transfer.object ? `${this.baseUrl}/api/${transfer.object.__typename.toLowerCase()}/${transfer.object.id}` : undefined
                    }
                }
            }
        };
    }
    
    /**
     * Create transfer list response
     */
    createTransferListResponse(transfers: Transfer[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<Transfer> {
        const paginationData = pagination || {
            page: 1,
            pageSize: transfers.length,
            totalCount: transfers.length
        };
        
        return {
            data: transfers,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: '1.0',
                links: {
                    self: `${this.baseUrl}/api/transfer?page=${paginationData.page}&limit=${paginationData.pageSize}`
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
                path: '/api/transfer'
            }
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(transferId: string): APIErrorResponse {
        return {
            error: {
                code: 'TRANSFER_NOT_FOUND',
                message: `Transfer with ID '${transferId}' was not found`,
                details: {
                    transferId,
                    searchCriteria: { id: transferId }
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/transfer/${transferId}`
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
                message: `You do not have permission to ${operation} this transfer`,
                details: {
                    operation,
                    requiredPermissions: ['transfer:write'],
                    userPermissions: ['transfer:read']
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: '/api/transfer'
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
                path: '/api/transfer'
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
                path: '/api/transfer'
            }
        };
    }
    
    /**
     * Create mock transfer data
     */
    createMockTransfer(overrides?: Partial<Transfer>): Transfer {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultTransfer: Transfer = {
            __typename: "Transfer",
            id,
            status: TransferStatusEnum.Pending,
            message: "Transfer ownership request",
            from: {
                __typename: "User",
                id: `user_from_${id}`,
                handle: "sender_user",
                name: "Sender User",
                created_at: now,
                updated_at: now,
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
                }
            },
            to: {
                __typename: "User",
                id: `user_to_${id}`,
                handle: "recipient_user",
                name: "Recipient User",
                created_at: now,
                updated_at: now,
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
                }
            },
            object: {
                __typename: "Routine",
                id: `routine_${id}`,
                created_at: now,
                updated_at: now,
                isInternal: false,
                isPrivate: false,
                completedAt: null,
                createdBy: null,
                hasCompleteVersion: true,
                score: 4.5,
                bookmarks: 25,
                views: 150,
                you: {
                    __typename: "RoutineYou",
                    canComment: true,
                    canDelete: true,
                    canBookmark: true,
                    canUpdate: true,
                    canRead: true,
                    canReact: true,
                    isBookmarked: false,
                    reaction: null
                }
            },
            you: {
                __typename: "TransferYou",
                canUpdate: true
            }
        };
        
        return {
            ...defaultTransfer,
            ...overrides
        };
    }
    
    /**
     * Create transfer from API input
     */
    createTransferFromInput(input: TransferCreateInput): Transfer {
        const transfer = this.createMockTransfer();
        
        // Update transfer based on input
        if (input.message) {
            transfer.message = input.message;
        }
        
        if (input.toConnect) {
            transfer.to.id = input.toConnect;
        }
        
        if (input.objectConnect) {
            transfer.object.id = input.objectConnect.objectId;
            transfer.object.__typename = input.objectConnect.objectType as any;
        }
        
        return transfer;
    }
    
    /**
     * Create transfers with different statuses
     */
    createTransfersWithAllStatuses(): Transfer[] {
        return Object.values(TransferStatusEnum).map(status => 
            this.createMockTransfer({
                status,
                message: `Transfer with ${status.toLowerCase()} status`
            })
        );
    }
    
    /**
     * Create transfers for different object types
     */
    createTransfersForDifferentObjects(): Transfer[] {
        const objectTypes = ['Routine', 'Project', 'Api', 'Team'];
        
        return objectTypes.map(objectType => 
            this.createMockTransfer({
                object: {
                    ...this.createMockTransfer().object,
                    __typename: objectType as any,
                    id: `${objectType.toLowerCase()}_${this.generateId()}`
                },
                message: `Transfer ${objectType} ownership`
            })
        );
    }
    
    /**
     * Create pending transfers for user
     */
    createPendingTransfersForUser(userId: string, count: number = 3): Transfer[] {
        return Array.from({ length: count }, (_, index) => 
            this.createMockTransfer({
                status: TransferStatusEnum.Pending,
                to: {
                    ...this.createMockTransfer().to,
                    id: userId
                },
                message: `Pending transfer request ${index + 1}`
            })
        );
    }
    
    /**
     * Validate transfer create input
     */
    async validateCreateInput(input: TransferCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await transferValidation.create.validate(input);
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
 * MSW handlers factory for transfer endpoints
 */
export class TransferMSWHandlers {
    private responseFactory: TransferResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new TransferResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all transfer endpoints
     */
    createSuccessHandlers(): RestHandler[] {
        return [
            // Create transfer
            rest.post(`${this.responseFactory['baseUrl']}/api/transfer`, async (req, res, ctx) => {
                const body = await req.json() as TransferCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {}))
                    );
                }
                
                // Create transfer
                const transfer = this.responseFactory.createTransferFromInput(body);
                const response = this.responseFactory.createSuccessResponse(transfer);
                
                return res(
                    ctx.status(201),
                    ctx.json(response)
                );
            }),
            
            // Get transfer by ID
            rest.get(`${this.responseFactory['baseUrl']}/api/transfer/:id`, (req, res, ctx) => {
                const { id } = req.params;
                
                const transfer = this.responseFactory.createMockTransfer({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(transfer);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Update transfer
            rest.put(`${this.responseFactory['baseUrl']}/api/transfer/:id`, async (req, res, ctx) => {
                const { id } = req.params;
                const body = await req.json() as TransferUpdateInput;
                
                const transfer = this.responseFactory.createMockTransfer({ id: id as string });
                
                // Apply updates from body
                if (body.status) {
                    transfer.status = body.status;
                }
                
                if (body.message) {
                    transfer.message = body.message;
                }
                
                const response = this.responseFactory.createSuccessResponse(transfer);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Delete transfer
            rest.delete(`${this.responseFactory['baseUrl']}/api/transfer/:id`, (req, res, ctx) => {
                return res(ctx.status(204));
            }),
            
            // List transfers
            rest.get(`${this.responseFactory['baseUrl']}/api/transfer`, (req, res, ctx) => {
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get('page') || '1');
                const limit = parseInt(url.searchParams.get('limit') || '10');
                const status = url.searchParams.get('status') as TransferStatus;
                const userId = url.searchParams.get('userId');
                const objectType = url.searchParams.get('objectType');
                
                let transfers: Transfer[] = [];
                
                if (userId && status === TransferStatusEnum.Pending) {
                    transfers = this.responseFactory.createPendingTransfersForUser(userId);
                } else if (objectType) {
                    transfers = this.responseFactory.createTransfersForDifferentObjects()
                        .filter(t => t.object.__typename.toLowerCase() === objectType.toLowerCase());
                } else {
                    transfers = this.responseFactory.createTransfersWithAllStatuses();
                }
                
                // Filter by status if specified
                if (status && !userId) {
                    transfers = transfers.filter(t => t.status === status);
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedTransfers = transfers.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createTransferListResponse(
                    paginatedTransfers,
                    {
                        page,
                        pageSize: limit,
                        totalCount: transfers.length
                    }
                );
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Accept transfer
            rest.post(`${this.responseFactory['baseUrl']}/api/transfer/:id/accept`, (req, res, ctx) => {
                const { id } = req.params;
                
                const transfer = this.responseFactory.createMockTransfer({ 
                    id: id as string,
                    status: TransferStatusEnum.Approved
                });
                
                const response = this.responseFactory.createSuccessResponse(transfer);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Reject transfer
            rest.post(`${this.responseFactory['baseUrl']}/api/transfer/:id/reject`, (req, res, ctx) => {
                const { id } = req.params;
                
                const transfer = this.responseFactory.createMockTransfer({ 
                    id: id as string,
                    status: TransferStatusEnum.Rejected
                });
                
                const response = this.responseFactory.createSuccessResponse(transfer);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Cancel transfer
            rest.post(`${this.responseFactory['baseUrl']}/api/transfer/:id/cancel`, (req, res, ctx) => {
                const { id } = req.params;
                
                const transfer = this.responseFactory.createMockTransfer({ 
                    id: id as string,
                    status: TransferStatusEnum.Cancelled
                });
                
                const response = this.responseFactory.createSuccessResponse(transfer);
                
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
            rest.post(`${this.responseFactory['baseUrl']}/api/transfer`, (req, res, ctx) => {
                return res(
                    ctx.status(400),
                    ctx.json(this.responseFactory.createValidationErrorResponse({
                        toConnect: 'Recipient is required',
                        objectConnect: 'Object to transfer is required'
                    }))
                );
            }),
            
            // Not found error
            rest.get(`${this.responseFactory['baseUrl']}/api/transfer/:id`, (req, res, ctx) => {
                const { id } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse(id as string))
                );
            }),
            
            // Permission error
            rest.post(`${this.responseFactory['baseUrl']}/api/transfer`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse('create'))
                );
            }),
            
            // Server error
            rest.post(`${this.responseFactory['baseUrl']}/api/transfer`, (req, res, ctx) => {
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
            rest.post(`${this.responseFactory['baseUrl']}/api/transfer`, async (req, res, ctx) => {
                const body = await req.json() as TransferCreateInput;
                const transfer = this.responseFactory.createTransferFromInput(body);
                const response = this.responseFactory.createSuccessResponse(transfer);
                
                return res(
                    ctx.delay(delay),
                    ctx.status(201),
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
            rest.post(`${this.responseFactory['baseUrl']}/api/transfer`, (req, res, ctx) => {
                return res.networkError('Network connection failed');
            }),
            
            rest.get(`${this.responseFactory['baseUrl']}/api/transfer/:id`, (req, res, ctx) => {
                return res.networkError('Connection timeout');
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
export const transferResponseScenarios = {
    // Success scenarios
    createSuccess: (transfer?: Transfer) => {
        const factory = new TransferResponseFactory();
        return factory.createSuccessResponse(
            transfer || factory.createMockTransfer()
        );
    },
    
    listSuccess: (transfers?: Transfer[]) => {
        const factory = new TransferResponseFactory();
        return factory.createTransferListResponse(
            transfers || factory.createTransfersWithAllStatuses()
        );
    },
    
    pendingTransfers: (userId?: string) => {
        const factory = new TransferResponseFactory();
        return factory.createTransferListResponse(
            factory.createPendingTransfersForUser(userId || 'user-123')
        );
    },
    
    transfersByStatus: (status: TransferStatus) => {
        const factory = new TransferResponseFactory();
        const transfers = factory.createTransfersWithAllStatuses().filter(t => t.status === status);
        return factory.createTransferListResponse(transfers);
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new TransferResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                toConnect: 'Recipient is required',
                objectConnect: 'Object to transfer is required'
            }
        );
    },
    
    notFoundError: (transferId?: string) => {
        const factory = new TransferResponseFactory();
        return factory.createNotFoundErrorResponse(
            transferId || 'non-existent-id'
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new TransferResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || 'create'
        );
    },
    
    serverError: () => {
        const factory = new TransferResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new TransferMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new TransferMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new TransferMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new TransferMSWHandlers().createNetworkErrorHandlers()
};

// Export factory instances for easy use
export const transferResponseFactory = new TransferResponseFactory();
export const transferMSWHandlers = new TransferMSWHandlers();