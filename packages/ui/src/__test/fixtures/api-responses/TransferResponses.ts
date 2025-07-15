/**
 * Transfer API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for transfer endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { http, HttpResponse, type RequestHandler } from "msw";
import type { 
    Transfer, 
    TransferRequestSendInput, 
    TransferUpdateInput,
    TransferStatus,
} from "@vrooli/shared";
import { 
    TransferStatus as TransferStatusEnum, 
} from "@vrooli/shared";

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
    
    constructor(baseUrl: string = process.env.VITE_SERVER_URL || "http://localhost:5329") {
        this.baseUrl = baseUrl;
    }
    
    /**
     * Generate unique request ID
     */
    private generateRequestId(): string {
        return `req_${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
    }
    
    /**
     * Generate unique transfer ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
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
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/transfer/${transfer.id}`,
                    related: {
                        fromOwner: transfer.fromOwner ? `${this.baseUrl}/api/user/${transfer.fromOwner.id}` : undefined,
                        toOwner: transfer.toOwner ? `${this.baseUrl}/api/user/${transfer.toOwner.id}` : undefined,
                        object: transfer.object ? `${this.baseUrl}/api/${transfer.object.__typename.toLowerCase()}/${transfer.object.id}` : undefined,
                    },
                },
            },
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
            totalCount: transfers.length,
        };
        
        return {
            data: transfers,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/transfer?page=${paginationData.page}&limit=${paginationData.pageSize}`,
                },
            },
            pagination: {
                ...paginationData,
                totalPages: Math.ceil(paginationData.totalCount / paginationData.pageSize),
                hasNextPage: paginationData.page * paginationData.pageSize < paginationData.totalCount,
                hasPreviousPage: paginationData.page > 1,
            },
        };
    }
    
    /**
     * Create validation error response
     */
    createValidationErrorResponse(fieldErrors: Record<string, string>): APIErrorResponse {
        return {
            error: {
                code: "VALIDATION_ERROR",
                message: "The request contains invalid data",
                details: {
                    fieldErrors,
                    invalidFields: Object.keys(fieldErrors),
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/transfer",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(transferId: string): APIErrorResponse {
        return {
            error: {
                code: "TRANSFER_NOT_FOUND",
                message: `Transfer with ID '${transferId}' was not found`,
                details: {
                    transferId,
                    searchCriteria: { id: transferId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/transfer/${transferId}`,
            },
        };
    }
    
    /**
     * Create permission error response
     */
    createPermissionErrorResponse(operation: string): APIErrorResponse {
        return {
            error: {
                code: "PERMISSION_DENIED",
                message: `You do not have permission to ${operation} this transfer`,
                details: {
                    operation,
                    requiredPermissions: ["transfer:write"],
                    userPermissions: ["transfer:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/transfer",
            },
        };
    }
    
    /**
     * Create network error response
     */
    createNetworkErrorResponse(): APIErrorResponse {
        return {
            error: {
                code: "NETWORK_ERROR",
                message: "Network request failed",
                details: {
                    reason: "Connection timeout",
                    retryable: true,
                    retryAfter: 5000,
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/transfer",
            },
        };
    }
    
    /**
     * Create server error response
     */
    createServerErrorResponse(): APIErrorResponse {
        return {
            error: {
                code: "INTERNAL_SERVER_ERROR",
                message: "An unexpected server error occurred",
                details: {
                    errorId: `ERR_${Date.now()}`,
                    reportable: true,
                    retryable: true,
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/transfer",
            },
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
            createdAt: now,
            updatedAt: now,
            status: TransferStatusEnum.Pending,
            closedAt: null,
            fromOwner: {
                __typename: "User",
                id: `user_from_${id}`,
                handle: "sender_user",
                name: "Sender User",
            } as any,
            toOwner: {
                __typename: "User",
                id: `user_to_${id}`,
                handle: "recipient_user",
                name: "Recipient User",
            } as any,
            object: {
                __typename: "Resource",
                id: `resource_${id}`,
                createdAt: now,
                isDeleted: false,
                isInternal: false,
                isPrivate: false,
                completedAt: null,
                createdBy: null,
                hasCompleteVersion: true,
                bookmarks: 25,
                bookmarkedBy: [],
                issues: [],
                issuesCount: 0,
                owner: null,
                parent: null,
                permissions: "{}",
                publicId: `pub_${id}`,
                you: {
                    __typename: "ResourceYou",
                    canComment: true,
                    canDelete: true,
                    canBookmark: true,
                    canUpdate: true,
                    canRead: true,
                    canReact: true,
                    canTransfer: true,
                    isBookmarked: false,
                    isViewed: false,
                    reaction: null,
                },
            } as any,
            you: {
                __typename: "TransferYou",
                canUpdate: true,
                canDelete: false,
            },
        };
        
        return {
            ...defaultTransfer,
            ...overrides,
        };
    }
    
    /**
     * Create transfer from API input
     */
    createTransferFromInput(input: TransferRequestSendInput): Transfer {
        const transfer = this.createMockTransfer();
        
        // Update transfer based on input
        if (input.toUserConnect && transfer.toOwner) {
            transfer.toOwner.id = input.toUserConnect;
        }
        
        if (input.objectConnect) {
            transfer.object.id = input.objectConnect;
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
            }),
        );
    }
    
    /**
     * Create transfers for different object types
     */
    createTransfersForDifferentObjects(): Transfer[] {
        // Only Resource is supported for transfers
        return [
            this.createMockTransfer({
                object: {
                    ...this.createMockTransfer().object,
                    __typename: "Resource",
                    id: `resource_${this.generateId()}`,
                } as any,
            })
        ];
    }
    
    /**
     * Create pending transfers for user
     */
    createPendingTransfersForUser(userId: string, count = 3): Transfer[] {
        return Array.from({ length: count }, (_, index) => 
            this.createMockTransfer({
                status: TransferStatusEnum.Pending,
                toOwner: {
                    ...this.createMockTransfer().toOwner,
                    id: userId,
                } as any,
            }),
        );
    }
    
    /**
     * Validate transfer create input
     */
    async validateCreateInput(input: TransferRequestSendInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};
        
        if (!input.objectConnect) {
            errors.objectConnect = "Object to transfer is required";
        }
        
        if (!input.toUserConnect && !input.toTeamConnect) {
            errors.toConnect = "Recipient is required";
        }
        
        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
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
    createSuccessHandlers(): RequestHandler[] {
        return [
            // Create transfer
            http.post(`${this.responseFactory["baseUrl"]}/api/transfer`, async ({ request, params }) => {
                const body = await request.json() as TransferRequestSendInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return HttpResponse.json(
                        this.responseFactory.createValidationErrorResponse(validation.errors || {}),
                        { status: 400 }
                    );
                }
                
                // Create transfer
                const transfer = this.responseFactory.createTransferFromInput(body);
                const response = this.responseFactory.createSuccessResponse(transfer);
                
                return HttpResponse.json(response, { status: 201 });
            }),
            
            // Get transfer by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/transfer/:id`, ({ request, params }) => {
                const { id } = params;
                
                const transfer = this.responseFactory.createMockTransfer({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(transfer);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Update transfer
            http.put(`${this.responseFactory["baseUrl"]}/api/transfer/:id`, async ({ request, params }) => {
                const { id } = params;
                const body = await request.json() as TransferUpdateInput;
                
                const transfer = this.responseFactory.createMockTransfer({ id: id as string });
                
                // Apply updates from body (limited updates allowed)
                
                const response = this.responseFactory.createSuccessResponse(transfer);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Delete transfer
            http.delete(`${this.responseFactory["baseUrl"]}/api/transfer/:id`, ({ request, params }) => {
                return new HttpResponse(null, { status: 204 });
            }),
            
            // List transfers
            http.get(`${this.responseFactory["baseUrl"]}/api/transfer`, ({ request, params }) => {
                const url = new URL(request.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const status = url.searchParams.get("status") as TransferStatus;
                const userId = url.searchParams.get("userId");
                const objectType = url.searchParams.get("objectType");
                
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
                        totalCount: transfers.length,
                    },
                );
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Accept transfer
            http.post(`${this.responseFactory["baseUrl"]}/api/transfer/:id/accept`, ({ request, params }) => {
                const { id } = params;
                
                const transfer = this.responseFactory.createMockTransfer({ 
                    id: id as string,
                    status: TransferStatusEnum.Accepted,
                });
                
                const response = this.responseFactory.createSuccessResponse(transfer);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Reject transfer
            http.post(`${this.responseFactory["baseUrl"]}/api/transfer/:id/reject`, ({ request, params }) => {
                const { id } = params;
                
                const transfer = this.responseFactory.createMockTransfer({ 
                    id: id as string,
                    status: TransferStatusEnum.Denied,
                });
                
                const response = this.responseFactory.createSuccessResponse(transfer);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Deny transfer (there's no cancel in the API, only deny)
            http.post(`${this.responseFactory["baseUrl"]}/api/transfer/:id/deny`, ({ request, params }) => {
                const { id } = params;
                
                const transfer = this.responseFactory.createMockTransfer({ 
                    id: id as string,
                    status: TransferStatusEnum.Denied,
                });
                
                const response = this.responseFactory.createSuccessResponse(transfer);
                
                return HttpResponse.json(response, { status: 200 });
            }),
        ];
    }
    
    /**
     * Create error handlers for testing error scenarios
     */
    createErrorHandlers(): RequestHandler[] {
        return [
            // Validation error
            http.post(`${this.responseFactory["baseUrl"]}/api/transfer`, ({ request, params }) => {
                return HttpResponse.json(
                    this.responseFactory.createValidationErrorResponse({
                        toUserConnect: "Recipient is required",
                        objectConnect: "Object to transfer is required",
                    }),
                    { status: 400 }
                );
            }),
            
            // Not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/transfer/:id`, ({ request, params }) => {
                const { id } = params;
                return HttpResponse.json(
                    this.responseFactory.createNotFoundErrorResponse(id as string),
                    { status: 404 }
                );
            }),
            
            // Permission error
            http.post(`${this.responseFactory["baseUrl"]}/api/transfer`, ({ request, params }) => {
                return HttpResponse.json(
                    this.responseFactory.createPermissionErrorResponse("create"),
                    { status: 403 }
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/transfer`, ({ request, params }) => {
                return HttpResponse.json(
                    this.responseFactory.createServerErrorResponse(),
                    { status: 500 }
                );
            }),
        ];
    }
    
    /**
     * Create loading simulation handlers
     */
    createLoadingHandlers(delay = 2000): RequestHandler[] {
        return [
            http.post(`${this.responseFactory["baseUrl"]}/api/transfer`, async ({ request, params }) => {
                const body = await request.json() as TransferRequestSendInput;
                const transfer = this.responseFactory.createTransferFromInput(body);
                const response = this.responseFactory.createSuccessResponse(transfer);
                
                await new Promise(resolve => setTimeout(resolve, delay));
                return HttpResponse.json(response, { status: 201 });
            }),
        ];
    }
    
    /**
     * Create network error handlers
     */
    createNetworkErrorHandlers(): RequestHandler[] {
        return [
            http.post(`${this.responseFactory["baseUrl"]}/api/transfer`, ({ request, params }) => {
                return HttpResponse.error();
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/transfer/:id`, ({ request, params }) => {
                return HttpResponse.error();
            }),
        ];
    }
    
    /**
     * Create custom handler with specific configuration
     */
    createCustomHandler(config: {
        endpoint: string;
        method: "GET" | "POST" | "PUT" | "DELETE";
        status: number;
        response: any;
        delay?: number;
    }): RequestHandler {
        const { endpoint, method, status, response, delay } = config;
        const fullEndpoint = `${this.responseFactory["baseUrl"]}${endpoint}`;
        
        const httpMethod = http[method.toLowerCase() as keyof typeof http] as any;
        return httpMethod(fullEndpoint, async ({ request, params }) => {
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
export const transferResponseScenarios = {
    // Success scenarios
    createSuccess: (transfer?: Transfer) => {
        const factory = new TransferResponseFactory();
        return factory.createSuccessResponse(
            transfer || factory.createMockTransfer(),
        );
    },
    
    listSuccess: (transfers?: Transfer[]) => {
        const factory = new TransferResponseFactory();
        return factory.createTransferListResponse(
            transfers || factory.createTransfersWithAllStatuses(),
        );
    },
    
    pendingTransfers: (userId?: string) => {
        const factory = new TransferResponseFactory();
        return factory.createTransferListResponse(
            factory.createPendingTransfersForUser(userId || "user-123"),
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
                toConnect: "Recipient is required",
                objectConnect: "Object to transfer is required",
            },
        );
    },
    
    notFoundError: (transferId?: string) => {
        const factory = new TransferResponseFactory();
        return factory.createNotFoundErrorResponse(
            transferId || "non-existent-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new TransferResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
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
    networkErrorHandlers: () => new TransferMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const transferResponseFactory = new TransferResponseFactory();
export const transferMSWHandlers = new TransferMSWHandlers();
