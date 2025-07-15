/**
 * MemberInvite API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for member invite endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */
// AI_CHECK: TYPE_SAFETY=fixed-user-properties | LAST: 2025-07-02 - Fixed User properties by removing many invalid fields

import { http, HttpResponse, type RequestHandler } from "msw";
import type { 
    MemberInvite, 
    MemberInviteCreateInput, 
    MemberInviteUpdateInput,
    MemberInviteStatus,
    Team,
    User,
} from "@vrooli/shared";
import { 
    memberInviteValidation,
    MemberInviteStatus as MemberInviteStatusEnum,
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
 * MemberInvite API response factory
 */
export class MemberInviteResponseFactory {
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
     * Generate unique resource ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
    }
    
    /**
     * Create successful member invite response
     */
    createSuccessResponse(memberInvite: MemberInvite): APIResponse<MemberInvite> {
        return {
            data: memberInvite,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/member-invite/${memberInvite.id}`,
                    related: {
                        team: `${this.baseUrl}/api/team/${memberInvite.team.id}`,
                        user: `${this.baseUrl}/api/user/${memberInvite.user.id}`,
                        accept: `${this.baseUrl}/api/member-invite/${memberInvite.id}/accept`,
                        decline: `${this.baseUrl}/api/member-invite/${memberInvite.id}/decline`,
                    },
                },
            },
        };
    }
    
    /**
     * Create member invite list response
     */
    createMemberInviteListResponse(memberInvites: MemberInvite[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<MemberInvite> {
        const paginationData = pagination || {
            page: 1,
            pageSize: memberInvites.length,
            totalCount: memberInvites.length,
        };
        
        return {
            data: memberInvites,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/member-invite?page=${paginationData.page}&limit=${paginationData.pageSize}`,
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
                path: "/api/member-invite",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(memberInviteId: string): APIErrorResponse {
        return {
            error: {
                code: "MEMBER_INVITE_NOT_FOUND",
                message: `Member invite with ID '${memberInviteId}' was not found`,
                details: {
                    memberInviteId,
                    searchCriteria: { id: memberInviteId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/member-invite/${memberInviteId}`,
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
                message: `You do not have permission to ${operation} this member invite`,
                details: {
                    operation,
                    requiredPermissions: ["team:invite", "team:admin"],
                    userPermissions: ["team:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/member-invite",
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
                path: "/api/member-invite",
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
                path: "/api/member-invite",
            },
        };
    }
    
    /**
     * Create invite already processed error
     */
    createInviteAlreadyProcessedErrorResponse(status: MemberInviteStatus): APIErrorResponse {
        return {
            error: {
                code: "INVITE_ALREADY_PROCESSED",
                message: `This invitation has already been ${(status as string).toLowerCase()}`,
                details: {
                    currentStatus: status,
                    allowedTransitions: status === MemberInviteStatusEnum.Pending ? ["Accepted", "Declined"] : [],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/member-invite",
            },
        };
    }
    
    /**
     * Create team full error response
     */
    createTeamFullErrorResponse(): APIErrorResponse {
        return {
            error: {
                code: "TEAM_MEMBER_LIMIT_REACHED",
                message: "This team has reached its maximum member limit",
                details: {
                    maxMembers: 100,
                    currentMembers: 100,
                    upgradeRequired: true,
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/member-invite",
            },
        };
    }
    
    /**
     * Create mock team data
     */
    createMockTeam(overrides?: Partial<Team>): Team {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultTeam: Team = {
            __typename: "Team",
            id: `team_${id}`,
            createdAt: now,
            updatedAt: now,
            handle: "dev-team",
            isOpenToNewMembers: true,
            isPrivate: false,
            permissions: JSON.stringify({
                default: ["read"],
                admin: ["read", "write", "invite", "manage"],
            }),
            bookmarkedBy: [],
            bookmarks: 0,
            comments: [],
            commentsCount: 0,
            config: null,
            forks: [],
            issues: [],
            issuesCount: 0,
            meetings: [],
            meetingsCount: 0,
            members: [],
            membersCount: 3,
            parent: null,
            paymentHistory: [],
            premium: null,
            profileImage: null,
            bannerImage: null,
            reports: [],
            reportsCount: 0,
            resources: [],
            resourcesCount: 0,
            tags: [],
            transfersIncoming: [],
            transfersOutgoing: [],
            translations: [],
            translationsCount: 0,
            views: 42,
            publicId: `team_public_${id}`,
            stats: [],
            translatedName: "Dev Team",
            wallets: [],
            you: {
                __typename: "TeamYou",
                canAddMembers: true,
                canBookmark: true,
                canDelete: false,
                canRead: true,
                canReport: true,
                canUpdate: true,
                isBookmarked: false,
                isViewed: true,
                yourMembership: null,
            },
        };
        
        return {
            ...defaultTeam,
            ...overrides,
        };
    }
    
    /**
     * Create mock user data
     */
    createMockUser(overrides?: Partial<User>): User {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultUser: User = {
            __typename: "User",
            id: `user_${id}`,
            createdAt: now,
            updatedAt: now,
            handle: "testuser",
            name: "Test User",
            isBot: false,
            isPrivate: false,
            profileImage: null,
            bannerImage: null,
            premium: null,
            publicId: `user_${id}`,
            bookmarkedBy: [],
            bookmarks: 0,
            isBotDepictingPerson: false,
            isPrivateBookmarks: true,
            isPrivateMemberships: true,
            isPrivatePullRequests: true,
            isPrivateResources: false,
            isPrivateResourcesCreated: false,
            isPrivateTeamsCreated: true,
            isPrivateVotes: true,
            membershipsCount: 0,
            reportsReceived: [],
            reportsReceivedCount: 0,
            resourcesCount: 0,
            translations: [],
            views: 123,
            wallets: [],
            you: {
                __typename: "UserYou",
                canDelete: false,
                canReport: false,
                canUpdate: false,
                isBookmarked: false,
                isViewed: false,
            },
        };
        
        return {
            ...defaultUser,
            ...overrides,
        };
    }
    
    /**
     * Create mock member invite data
     */
    createMockMemberInvite(overrides?: Partial<MemberInvite>): MemberInvite {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultMemberInvite: MemberInvite = {
            __typename: "MemberInvite",
            id: `invite_${id}`,
            createdAt: now,
            updatedAt: now,
            message: "You've been invited to join our team!",
            status: MemberInviteStatusEnum.Pending,
            willBeAdmin: false,
            willHavePermissions: JSON.stringify(["read", "write"]),
            team: this.createMockTeam(),
            user: this.createMockUser(),
            you: {
                __typename: "MemberInviteYou",
                canDelete: true,
                canUpdate: true,
            },
        };
        
        return {
            ...defaultMemberInvite,
            ...overrides,
        };
    }
    
    /**
     * Create member invite from API input
     */
    createMemberInviteFromInput(input: MemberInviteCreateInput): MemberInvite {
        const memberInvite = this.createMockMemberInvite();
        
        // Update based on input
        if (input.id) memberInvite.id = input.id;
        if (input.message) memberInvite.message = input.message;
        if (input.willBeAdmin !== undefined) memberInvite.willBeAdmin = input.willBeAdmin;
        if (input.willHavePermissions) memberInvite.willHavePermissions = input.willHavePermissions;
        
        // Update team and user connections
        memberInvite.team = this.createMockTeam({ id: input.teamConnect });
        memberInvite.user = this.createMockUser({ id: input.userConnect });
        
        return memberInvite;
    }
    
    /**
     * Create member invites for different statuses
     */
    createMemberInvitesForAllStatuses(): MemberInvite[] {
        return Object.values(MemberInviteStatusEnum).map(status => 
            this.createMockMemberInvite({
                status,
                id: `invite_${(status as string).toLowerCase()}_${this.generateId()}`,
            }),
        );
    }
    
    /**
     * Create member invites with different role configurations
     */
    createMemberInvitesForRoles(): MemberInvite[] {
        const roles = [
            { willBeAdmin: true, permissions: ["read", "write", "admin", "invite"] },
            { willBeAdmin: false, permissions: ["read", "write"] },
            { willBeAdmin: false, permissions: ["read"] },
            { willBeAdmin: false, permissions: [] },
        ];
        
        return roles.map((role, index) => 
            this.createMockMemberInvite({
                id: `invite_role_${index}_${this.generateId()}`,
                willBeAdmin: role.willBeAdmin,
                willHavePermissions: JSON.stringify(role.permissions),
            }),
        );
    }
    
    /**
     * Validate member invite create input
     */
    async validateCreateInput(input: MemberInviteCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await memberInviteValidation.create({}).validate(input);
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
    
    /**
     * Validate member invite update input
     */
    async validateUpdateInput(input: MemberInviteUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await memberInviteValidation.update({}).validate(input);
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
 * MSW handlers factory for member invite endpoints
 */
export class MemberInviteMSWHandlers {
    private responseFactory: MemberInviteResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new MemberInviteResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all member invite endpoints
     */
    createSuccessHandlers(): RequestHandler[] {
        return [
            // Create member invite
            http.post(`${this.responseFactory["baseUrl"]}/api/member-invite`, async ({ request, params }) => {
                const body = await request.json() as MemberInviteCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return HttpResponse.json(
                        this.responseFactory.createValidationErrorResponse(validation.errors || {}),
                        { status: 400 }
                    );
                }
                
                // Create member invite
                const memberInvite = this.responseFactory.createMemberInviteFromInput(body);
                const response = this.responseFactory.createSuccessResponse(memberInvite);
                
                return HttpResponse.json(
                    response,
                    { status: 201 }
                );
            }),
            
            // Get member invite by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/member-invite/:id`, ({ request, params }) => {
                const { id } = params;
                
                const memberInvite = this.responseFactory.createMockMemberInvite({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(memberInvite);
                
                return HttpResponse.json(
                    response,
                    { status: 200 }
                );
            }),
            
            // Update member invite
            http.put(`${this.responseFactory["baseUrl"]}/api/member-invite/:id`, async ({ request, params }) => {
                const { id } = params;
                const body = await request.json() as MemberInviteUpdateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateUpdateInput(body);
                if (!validation.valid) {
                    return HttpResponse.json(
                        this.responseFactory.createValidationErrorResponse(validation.errors || {}),
                        { status: 400 }
                    );
                }
                
                const memberInvite = this.responseFactory.createMockMemberInvite({ 
                    id: id as string,
                    updatedAt: new Date().toISOString(),
                    message: body.message,
                    willBeAdmin: body.willBeAdmin,
                    willHavePermissions: body.willHavePermissions,
                });
                
                const response = this.responseFactory.createSuccessResponse(memberInvite);
                
                return HttpResponse.json(
                    response,
                    { status: 200 }
                );
            }),
            
            // Delete member invite
            http.delete(`${this.responseFactory["baseUrl"]}/api/member-invite/:id`, ({ request, params }) => {
                return new HttpResponse(null, { status: 204 });
            }),
            
            // Accept member invite
            http.put(`${this.responseFactory["baseUrl"]}/api/member-invite/:id/accept`, ({ request, params }) => {
                const { id } = params;
                
                const memberInvite = this.responseFactory.createMockMemberInvite({ 
                    id: id as string,
                    status: MemberInviteStatusEnum.Accepted,
                    updatedAt: new Date().toISOString(),
                });
                
                const response = this.responseFactory.createSuccessResponse(memberInvite);
                
                return HttpResponse.json(
                    response,
                    { status: 200 }
                );
            }),
            
            // Decline member invite
            http.put(`${this.responseFactory["baseUrl"]}/api/member-invite/:id/decline`, ({ request, params }) => {
                const { id } = params;
                
                const memberInvite = this.responseFactory.createMockMemberInvite({ 
                    id: id as string,
                    status: MemberInviteStatusEnum.Declined,
                    updatedAt: new Date().toISOString(),
                });
                
                const response = this.responseFactory.createSuccessResponse(memberInvite);
                
                return HttpResponse.json(
                    response,
                    { status: 200 }
                );
            }),
            
            // List member invites
            http.get(`${this.responseFactory["baseUrl"]}/api/member-invite`, ({ request, params }) => {
                const url = new URL(request.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const status = url.searchParams.get("status") as MemberInviteStatus;
                const teamId = url.searchParams.get("teamId");
                const userId = url.searchParams.get("userId");
                
                let memberInvites = this.responseFactory.createMemberInvitesForAllStatuses();
                
                // Filter by status if specified
                if (status) {
                    memberInvites = memberInvites.filter(invite => invite.status === status);
                }
                
                // Filter by team if specified
                if (teamId) {
                    memberInvites = memberInvites.filter(invite => invite.team.id === teamId);
                }
                
                // Filter by user if specified
                if (userId) {
                    memberInvites = memberInvites.filter(invite => invite.user.id === userId);
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedMemberInvites = memberInvites.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createMemberInviteListResponse(
                    paginatedMemberInvites,
                    {
                        page,
                        pageSize: limit,
                        totalCount: memberInvites.length,
                    },
                );
                
                return HttpResponse.json(
                    response,
                    { status: 200 }
                );
            }),
        ];
    }
    
    /**
     * Create error handlers for testing error scenarios
     */
    createErrorHandlers(): RequestHandler[] {
        return [
            // Validation error
            http.post(`${this.responseFactory["baseUrl"]}/api/member-invite`, ({ request, params }) => {
                return HttpResponse.json(
                    this.responseFactory.createValidationErrorResponse({
                        teamConnect: "Team ID is required",
                        userConnect: "User ID is required",
                    }),
                    { status: 400 }
                );
            }),
            
            // Not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/member-invite/:id`, ({ request, params }) => {
                const { id } = params;
                return HttpResponse.json(
                    this.responseFactory.createNotFoundErrorResponse(id as string),
                    { status: 404 }
                );
            }),
            
            // Permission error
            http.post(`${this.responseFactory["baseUrl"]}/api/member-invite`, ({ request, params }) => {
                return HttpResponse.json(
                    this.responseFactory.createPermissionErrorResponse("create"),
                    { status: 403 }
                );
            }),
            
            // Already processed error
            http.put(`${this.responseFactory["baseUrl"]}/api/member-invite/:id/accept`, ({ request, params }) => {
                return HttpResponse.json(
                    this.responseFactory.createInviteAlreadyProcessedErrorResponse(MemberInviteStatusEnum.Accepted),
                    { status: 409 }
                );
            }),
            
            // Team full error
            http.post(`${this.responseFactory["baseUrl"]}/api/member-invite`, ({ request, params }) => {
                return HttpResponse.json(
                    this.responseFactory.createTeamFullErrorResponse(),
                    { status: 422 }
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/member-invite`, ({ request, params }) => {
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
            http.post(`${this.responseFactory["baseUrl"]}/api/member-invite`, async ({ request, params }) => {
                const body = await request.json() as MemberInviteCreateInput;
                const memberInvite = this.responseFactory.createMemberInviteFromInput(body);
                const response = this.responseFactory.createSuccessResponse(memberInvite);
                
                await new Promise(resolve => setTimeout(resolve, delay));
                return HttpResponse.json(
                    response,
                    { status: 201 }
                );
            }),
            
            http.put(`${this.responseFactory["baseUrl"]}/api/member-invite/:id/accept`, async ({ request, params }) => {
                const { id } = params;
                const memberInvite = this.responseFactory.createMockMemberInvite({ 
                    id: id as string,
                    status: MemberInviteStatusEnum.Accepted,
                });
                const response = this.responseFactory.createSuccessResponse(memberInvite);
                
                await new Promise(resolve => setTimeout(resolve, delay));
                return HttpResponse.json(
                    response,
                    { status: 200 }
                );
            }),
        ];
    }
    
    /**
     * Create network error handlers
     */
    createNetworkErrorHandlers(): RequestHandler[] {
        return [
            http.post(`${this.responseFactory["baseUrl"]}/api/member-invite`, ({ request, params }) => {
                return HttpResponse.error();
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/member-invite/:id`, ({ request, params }) => {
                return HttpResponse.error();
            }),
            
            http.put(`${this.responseFactory["baseUrl"]}/api/member-invite/:id/accept`, ({ request, params }) => {
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
        
        const httpMethod = method.toLowerCase() as keyof typeof http;
        return http[httpMethod](fullEndpoint, async ({ request, params }) => {
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
export const memberInviteResponseScenarios = {
    // Success scenarios
    createSuccess: (memberInvite?: MemberInvite) => {
        const factory = new MemberInviteResponseFactory();
        return factory.createSuccessResponse(
            memberInvite || factory.createMockMemberInvite(),
        );
    },
    
    listSuccess: (memberInvites?: MemberInvite[]) => {
        const factory = new MemberInviteResponseFactory();
        return factory.createMemberInviteListResponse(
            memberInvites || factory.createMemberInvitesForAllStatuses(),
        );
    },
    
    acceptSuccess: (memberInviteId?: string) => {
        const factory = new MemberInviteResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockMemberInvite({
                id: memberInviteId || "invite_123",
                status: MemberInviteStatusEnum.Accepted,
            }),
        );
    },
    
    declineSuccess: (memberInviteId?: string) => {
        const factory = new MemberInviteResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockMemberInvite({
                id: memberInviteId || "invite_123",
                status: MemberInviteStatusEnum.Declined,
            }),
        );
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new MemberInviteResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                teamConnect: "Team ID is required",
                userConnect: "User ID is required",
            },
        );
    },
    
    notFoundError: (memberInviteId?: string) => {
        const factory = new MemberInviteResponseFactory();
        return factory.createNotFoundErrorResponse(
            memberInviteId || "non-existent-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new MemberInviteResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
        );
    },
    
    alreadyProcessedError: (status?: MemberInviteStatus) => {
        const factory = new MemberInviteResponseFactory();
        return factory.createInviteAlreadyProcessedErrorResponse(
            status || MemberInviteStatusEnum.Accepted,
        );
    },
    
    teamFullError: () => {
        const factory = new MemberInviteResponseFactory();
        return factory.createTeamFullErrorResponse();
    },
    
    serverError: () => {
        const factory = new MemberInviteResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new MemberInviteMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new MemberInviteMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new MemberInviteMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new MemberInviteMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const memberInviteResponseFactory = new MemberInviteResponseFactory();
export const memberInviteMSWHandlers = new MemberInviteMSWHandlers();
