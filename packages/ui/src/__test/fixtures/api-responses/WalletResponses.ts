/**
 * Wallet API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for wallet endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { http, type RestHandler } from "msw";
import type { 
    Wallet, 
    WalletCreateInput, 
    WalletUpdateInput,
    WalletType,
} from "@vrooli/shared";
import { 
    walletValidation, 
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
 * Wallet API response factory
 */
export class WalletResponseFactory {
    private readonly baseUrl: string;
    
    constructor(baseUrl: string = process.env.VITE_SERVER_URL || "http://localhost:5329") {
        this.baseUrl = baseUrl;
    }
    
    /**
     * Generate unique request ID
     */
    private generateRequestId(): string {
        return `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Generate unique wallet ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Create successful wallet response
     */
    createSuccessResponse(wallet: Wallet): APIResponse<Wallet> {
        return {
            data: wallet,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/wallet/${wallet.id}`,
                    related: {
                        user: `${this.baseUrl}/api/user/${wallet.user?.id}`,
                        team: wallet.team ? `${this.baseUrl}/api/team/${wallet.team.id}` : undefined,
                    },
                },
            },
        };
    }
    
    /**
     * Create wallet list response
     */
    createWalletListResponse(wallets: Wallet[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<Wallet> {
        const paginationData = pagination || {
            page: 1,
            pageSize: wallets.length,
            totalCount: wallets.length,
        };
        
        return {
            data: wallets,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/wallet?page=${paginationData.page}&limit=${paginationData.pageSize}`,
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
                path: "/api/wallet",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(walletId: string): APIErrorResponse {
        return {
            error: {
                code: "WALLET_NOT_FOUND",
                message: `Wallet with ID '${walletId}' was not found`,
                details: {
                    walletId,
                    searchCriteria: { id: walletId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/wallet/${walletId}`,
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
                message: `You do not have permission to ${operation} this wallet`,
                details: {
                    operation,
                    requiredPermissions: ["wallet:write"],
                    userPermissions: ["wallet:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/wallet",
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
                path: "/api/wallet",
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
                path: "/api/wallet",
            },
        };
    }
    
    /**
     * Create mock wallet data
     */
    createMockWallet(overrides?: Partial<Wallet>): Wallet {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultWallet: Wallet = {
            __typename: "Wallet",
            id,
            created_at: now,
            updated_at: now,
            handle: `wallet_${id}`,
            name: "Default Wallet",
            publicAddress: `0x${Math.random().toString(16).substring(2, 42)}`,
            stakingAddress: `stake1${Math.random().toString(16).substring(2, 42)}`,
            verified: false,
            user: {
                __typename: "User",
                id: `user_${id}`,
                handle: "wallet_owner",
                name: "Wallet Owner",
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
                        count: 0,
                    },
                },
            },
            team: null,
            you: {
                __typename: "WalletYou",
                canDelete: true,
                canUpdate: true,
            },
        };
        
        return {
            ...defaultWallet,
            ...overrides,
        };
    }
    
    /**
     * Create wallet from API input
     */
    createWalletFromInput(input: WalletCreateInput): Wallet {
        const wallet = this.createMockWallet();
        
        // Update wallet based on input
        if (input.handle) {
            wallet.handle = input.handle;
        }
        
        if (input.name) {
            wallet.name = input.name;
        }
        
        if (input.publicAddress) {
            wallet.publicAddress = input.publicAddress;
        }
        
        if (input.stakingAddress) {
            wallet.stakingAddress = input.stakingAddress;
        }
        
        if (input.verified !== undefined) {
            wallet.verified = input.verified;
        }
        
        // Handle user/team connections
        if (input.userConnect) {
            wallet.user.id = input.userConnect;
            wallet.team = null;
        } else if (input.teamConnect) {
            wallet.team = {
                __typename: "Team",
                id: input.teamConnect,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                handle: "team_wallet_owner",
                name: "Team Wallet Owner",
                isOpenToNewMembers: true,
                isPrivate: false,
                bookmarks: 0,
                views: 0,
                you: {
                    __typename: "TeamYou",
                    canAddMembers: true,
                    canBookmark: false,
                    canDelete: false,
                    canReport: false,
                    canUpdate: false,
                    canRead: true,
                    isBookmarked: false,
                    isViewed: false,
                    yourMembership: null,
                },
            };
            wallet.user = null as any; // Team wallet doesn't have user
        }
        
        return wallet;
    }
    
    /**
     * Create different types of wallets
     */
    createWalletTypes(): Wallet[] {
        return [
            // Personal verified wallet
            this.createMockWallet({
                name: "Primary Wallet",
                handle: "primary_wallet",
                verified: true,
                publicAddress: "0xa0b86991c431e59b79c1d2f01e81b0d7b4a39e5c7",
                stakingAddress: "stake1ux3kcqc8xljgdw85xgzz7w7pcf6j8z3z2z9g3z3z3z3z3z",
            }),
            
            // Personal unverified wallet
            this.createMockWallet({
                name: "Secondary Wallet",
                handle: "secondary_wallet",
                verified: false,
                publicAddress: "0x95ad61b0a150d79219dcf64e1e6cc01f0b64c4ce",
                stakingAddress: null,
            }),
            
            // Team wallet
            this.createMockWallet({
                name: "Team Treasury",
                handle: "team_treasury",
                verified: true,
                user: null,
                team: {
                    __typename: "Team",
                    id: `team_${this.generateId()}`,
                    created_at: new Date().toISOString(),
                    updated_at: new Date().toISOString(),
                    handle: "development_team",
                    name: "Development Team",
                    isOpenToNewMembers: true,
                    isPrivate: false,
                    bookmarks: 0,
                    views: 0,
                    you: {
                        __typename: "TeamYou",
                        canAddMembers: true,
                        canBookmark: false,
                        canDelete: false,
                        canReport: false,
                        canUpdate: false,
                        canRead: true,
                        isBookmarked: false,
                        isViewed: false,
                        yourMembership: null,
                    },
                },
            }),
        ];
    }
    
    /**
     * Create wallets for a specific user
     */
    createWalletsForUser(userId: string, count = 3): Wallet[] {
        return Array.from({ length: count }, (_, index) => 
            this.createMockWallet({
                user: {
                    ...this.createMockWallet().user,
                    id: userId,
                },
                name: `Wallet ${index + 1}`,
                handle: `wallet_${index + 1}_${userId}`,
                verified: index === 0, // First wallet is verified
                publicAddress: `0x${Math.random().toString(16).substring(2, 42)}`,
            }),
        );
    }
    
    /**
     * Generate realistic crypto addresses
     */
    generateCryptoAddresses() {
        return {
            ethereum: `0x${Math.random().toString(16).substring(2, 42)}`,
            bitcoin: this.generateBitcoinAddress(),
            cardano: this.generateCardanoAddress(),
            solana: this.generateSolanaAddress(),
        };
    }
    
    private generateBitcoinAddress(): string {
        const prefixes = ["1", "3", "bc1"];
        const prefix = prefixes[Math.floor(Math.random() * prefixes.length)];
        const address = Math.random().toString(36).substring(2, 35);
        return `${prefix}${address}`;
    }
    
    private generateCardanoAddress(): string {
        return `addr1${Math.random().toString(16).substring(2, 98)}`;
    }
    
    private generateSolanaAddress(): string {
        return Math.random().toString(36).substring(2, 44);
    }
    
    /**
     * Validate wallet create input
     */
    async validateCreateInput(input: WalletCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await walletValidation.create.validate(input);
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
                errors: fieldErrors,
            };
        }
    }
}

/**
 * MSW handlers factory for wallet endpoints
 */
export class WalletMSWHandlers {
    private responseFactory: WalletResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new WalletResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all wallet endpoints
     */
    createSuccessHandlers(): RestHandler[] {
        return [
            // Create wallet
            http.post(`${this.responseFactory["baseUrl"]}/api/wallet`, async (req, res, ctx) => {
                const body = await req.json() as WalletCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {})),
                    );
                }
                
                // Create wallet
                const wallet = this.responseFactory.createWalletFromInput(body);
                const response = this.responseFactory.createSuccessResponse(wallet);
                
                return res(
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
            
            // Get wallet by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/wallet/:id`, (req, res, ctx) => {
                const { id } = req.params;
                
                const wallet = this.responseFactory.createMockWallet({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(wallet);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Update wallet
            http.put(`${this.responseFactory["baseUrl"]}/api/wallet/:id`, async (req, res, ctx) => {
                const { id } = req.params;
                const body = await req.json() as WalletUpdateInput;
                
                const wallet = this.responseFactory.createMockWallet({ 
                    id: id as string,
                    updated_at: new Date().toISOString(),
                });
                
                // Apply updates from body
                if (body.name) {
                    wallet.name = body.name;
                }
                
                if (body.handle) {
                    wallet.handle = body.handle;
                }
                
                if (body.verified !== undefined) {
                    wallet.verified = body.verified;
                }
                
                const response = this.responseFactory.createSuccessResponse(wallet);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Delete wallet
            http.delete(`${this.responseFactory["baseUrl"]}/api/wallet/:id`, (req, res, ctx) => {
                return res(ctx.status(204));
            }),
            
            // List wallets
            http.get(`${this.responseFactory["baseUrl"]}/api/wallet`, (req, res, ctx) => {
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const userId = url.searchParams.get("userId");
                const teamId = url.searchParams.get("teamId");
                const verified = url.searchParams.get("verified") === "true";
                
                let wallets: Wallet[] = [];
                
                if (userId) {
                    wallets = this.responseFactory.createWalletsForUser(userId);
                } else if (teamId) {
                    wallets = [this.responseFactory.createMockWallet({
                        team: {
                            ...this.responseFactory.createMockWallet().team!,
                            id: teamId,
                        },
                        user: null,
                    })];
                } else {
                    wallets = this.responseFactory.createWalletTypes();
                }
                
                // Filter by verified status if specified
                if (verified !== null) {
                    wallets = wallets.filter(w => w.verified === verified);
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedWallets = wallets.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createWalletListResponse(
                    paginatedWallets,
                    {
                        page,
                        pageSize: limit,
                        totalCount: wallets.length,
                    },
                );
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Verify wallet
            http.post(`${this.responseFactory["baseUrl"]}/api/wallet/:id/verify`, (req, res, ctx) => {
                const { id } = req.params;
                
                const wallet = this.responseFactory.createMockWallet({ 
                    id: id as string,
                    verified: true,
                    updated_at: new Date().toISOString(),
                });
                
                const response = this.responseFactory.createSuccessResponse(wallet);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Check wallet verification status
            http.get(`${this.responseFactory["baseUrl"]}/api/wallet/:id/verification-status`, (req, res, ctx) => {
                const { id } = req.params;
                
                // Simulate verification status check
                const verificationStatus = {
                    id: id as string,
                    verified: Math.random() > 0.5,
                    verificationPending: Math.random() > 0.7,
                    lastVerificationAttempt: new Date().toISOString(),
                };
                
                return res(
                    ctx.status(200),
                    ctx.json({
                        data: verificationStatus,
                        meta: {
                            timestamp: new Date().toISOString(),
                            requestId: this.responseFactory["generateRequestId"](),
                            version: "1.0",
                        },
                    }),
                );
            }),
        ];
    }
    
    /**
     * Create error handlers for testing error scenarios
     */
    createErrorHandlers(): RestHandler[] {
        return [
            // Validation error
            http.post(`${this.responseFactory["baseUrl"]}/api/wallet`, (req, res, ctx) => {
                return res(
                    ctx.status(400),
                    ctx.json(this.responseFactory.createValidationErrorResponse({
                        publicAddress: "Invalid wallet address format",
                        name: "Wallet name is required",
                    })),
                );
            }),
            
            // Not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/wallet/:id`, (req, res, ctx) => {
                const { id } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse(id as string)),
                );
            }),
            
            // Permission error
            http.post(`${this.responseFactory["baseUrl"]}/api/wallet`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse("create")),
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/wallet`, (req, res, ctx) => {
                return res(
                    ctx.status(500),
                    ctx.json(this.responseFactory.createServerErrorResponse()),
                );
            }),
        ];
    }
    
    /**
     * Create loading simulation handlers
     */
    createLoadingHandlers(delay = 2000): RestHandler[] {
        return [
            http.post(`${this.responseFactory["baseUrl"]}/api/wallet`, async (req, res, ctx) => {
                const body = await req.json() as WalletCreateInput;
                const wallet = this.responseFactory.createWalletFromInput(body);
                const response = this.responseFactory.createSuccessResponse(wallet);
                
                return res(
                    ctx.delay(delay),
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
        ];
    }
    
    /**
     * Create network error handlers
     */
    createNetworkErrorHandlers(): RestHandler[] {
        return [
            http.post(`${this.responseFactory["baseUrl"]}/api/wallet`, (req, res, ctx) => {
                return res.networkError("Network connection failed");
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/wallet/:id`, (req, res, ctx) => {
                return res.networkError("Connection timeout");
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
    }): RestHandler {
        const { endpoint, method, status, response, delay } = config;
        const fullEndpoint = `${this.responseFactory["baseUrl"]}${endpoint}`;
        
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
export const walletResponseScenarios = {
    // Success scenarios
    createSuccess: (wallet?: Wallet) => {
        const factory = new WalletResponseFactory();
        return factory.createSuccessResponse(
            wallet || factory.createMockWallet(),
        );
    },
    
    listSuccess: (wallets?: Wallet[]) => {
        const factory = new WalletResponseFactory();
        return factory.createWalletListResponse(
            wallets || factory.createWalletTypes(),
        );
    },
    
    verifiedWallet: () => {
        const factory = new WalletResponseFactory();
        const wallet = factory.createMockWallet({
            verified: true,
            name: "Verified Wallet",
            publicAddress: "0xa0b86991c431e59b79c1d2f01e81b0d7b4a39e5c7",
        });
        return factory.createSuccessResponse(wallet);
    },
    
    userWallets: (userId?: string) => {
        const factory = new WalletResponseFactory();
        return factory.createWalletListResponse(
            factory.createWalletsForUser(userId || "user-123"),
        );
    },
    
    teamWallet: (teamId?: string) => {
        const factory = new WalletResponseFactory();
        const wallet = factory.createMockWallet({
            team: {
                ...factory.createMockWallet().team!,
                id: teamId || "team-456",
            },
            user: null,
            name: "Team Treasury Wallet",
            verified: true,
        });
        return factory.createSuccessResponse(wallet);
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new WalletResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                publicAddress: "Invalid wallet address format",
                name: "Wallet name is required",
            },
        );
    },
    
    notFoundError: (walletId?: string) => {
        const factory = new WalletResponseFactory();
        return factory.createNotFoundErrorResponse(
            walletId || "non-existent-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new WalletResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
        );
    },
    
    serverError: () => {
        const factory = new WalletResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new WalletMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new WalletMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new WalletMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new WalletMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const walletResponseFactory = new WalletResponseFactory();
export const walletMSWHandlers = new WalletMSWHandlers();
