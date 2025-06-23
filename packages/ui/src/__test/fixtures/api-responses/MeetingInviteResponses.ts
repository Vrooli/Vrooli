/**
 * MeetingInvite API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for meeting invite endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { rest, type RestHandler } from "msw";
import type { 
    MeetingInvite, 
    MeetingInviteCreateInput, 
    MeetingInviteUpdateInput,
    MeetingInviteStatus,
    Meeting,
    User,
    MeetingInviteYou,
    MeetingInviteSortBy,
} from "@vrooli/shared";
import { 
    meetingInviteValidation,
    MeetingInviteStatus as MeetingInviteStatusEnum, 
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
 * MeetingInvite API response factory
 */
export class MeetingInviteResponseFactory {
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
     * Generate unique resource ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Create successful meeting invite response
     */
    createSuccessResponse(meetingInvite: MeetingInvite): APIResponse<MeetingInvite> {
        return {
            data: meetingInvite,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/meeting-invite/${meetingInvite.id}`,
                    related: {
                        meeting: `${this.baseUrl}/api/meeting/${meetingInvite.meeting.id}`,
                        user: `${this.baseUrl}/api/user/${meetingInvite.user.id}`,
                    },
                },
            },
        };
    }
    
    /**
     * Create meeting invite list response
     */
    createMeetingInviteListResponse(meetingInvites: MeetingInvite[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<MeetingInvite> {
        const paginationData = pagination || {
            page: 1,
            pageSize: meetingInvites.length,
            totalCount: meetingInvites.length,
        };
        
        return {
            data: meetingInvites,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/meeting-invite?page=${paginationData.page}&limit=${paginationData.pageSize}`,
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
                path: "/api/meeting-invite",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(meetingInviteId: string): APIErrorResponse {
        return {
            error: {
                code: "MEETING_INVITE_NOT_FOUND",
                message: `Meeting invite with ID '${meetingInviteId}' was not found`,
                details: {
                    meetingInviteId,
                    searchCriteria: { id: meetingInviteId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/meeting-invite/${meetingInviteId}`,
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
                message: `You do not have permission to ${operation} this meeting invite`,
                details: {
                    operation,
                    requiredPermissions: ["meeting:manage"],
                    userPermissions: ["meeting:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/meeting-invite",
            },
        };
    }
    
    /**
     * Create RSVP status conflict error response
     */
    createRSVPConflictErrorResponse(currentStatus: MeetingInviteStatus, requestedStatus: MeetingInviteStatus): APIErrorResponse {
        return {
            error: {
                code: "RSVP_STATUS_CONFLICT",
                message: `Cannot change RSVP status from ${currentStatus} to ${requestedStatus}`,
                details: {
                    currentStatus,
                    requestedStatus,
                    validTransitions: ["Pending -> Accepted", "Pending -> Declined", "Accepted -> Declined", "Declined -> Accepted"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/meeting-invite",
            },
        };
    }
    
    /**
     * Create meeting capacity error response
     */
    createMeetingCapacityErrorResponse(currentCount: number, maxCapacity: number): APIErrorResponse {
        return {
            error: {
                code: "MEETING_CAPACITY_EXCEEDED",
                message: "Meeting has reached maximum capacity",
                details: {
                    currentAttendees: currentCount,
                    maxCapacity,
                    waitingListAvailable: true,
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/meeting-invite",
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
                path: "/api/meeting-invite",
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
                path: "/api/meeting-invite",
            },
        };
    }
    
    /**
     * Create mock meeting data
     */
    createMockMeeting(overrides?: Partial<Meeting>): Meeting {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultMeeting: Meeting = {
            __typename: "Meeting",
            id,
            createdAt: now,
            updatedAt: now,
            attendees: [],
            attendeesCount: 0,
            invites: [],
            invitesCount: 1,
            openToAnyoneWithInvite: false,
            publicId: `meeting_${id}`,
            schedule: {
                __typename: "Schedule",
                id: `schedule_${id}`,
                createdAt: now,
                updatedAt: now,
                startTime: new Date(Date.now() + 86400000).toISOString(), // Tomorrow
                endTime: new Date(Date.now() + 90000000).toISOString(), // Tomorrow + 1 hour
                timezone: "UTC",
                exceptions: [],
                exceptionsCount: 0,
                focusModes: [],
                focusModesCount: 0,
                meetings: [],
                meetingsCount: 1,
                recurrences: [],
                recurrencesCount: 0,
                reminders: [],
                remindersCount: 0,
                runProjects: [],
                runProjectsCount: 0,
                runRoutines: [],
                runRoutinesCount: 0,
                you: {
                    __typename: "ScheduleYou",
                    canDelete: true,
                    canUpdate: true,
                },
            },
            showOnTeamProfile: true,
            team: {
                __typename: "Team",
                id: `team_${id}`,
                handle: "example-team",
                name: "Example Team",
                createdAt: now,
                updatedAt: now,
                apis: [],
                apisCount: 0,
                bannerImage: null,
                bookmarked: [],
                bookmarkedCount: 0,
                bookmarks: [],
                bookmarksCount: 0,
                codes: [],
                codesCount: 0,
                comments: [],
                commentsCount: 0,
                focusModes: [],
                focusModesCount: 0,
                isOpenToNewMembers: true,
                isPrivate: false,
                issues: [],
                issuesCount: 0,
                labels: [],
                labelsCount: 0,
                meetings: [],
                meetingsCount: 1,
                members: [],
                membersCount: 1,
                notes: [],
                notesCount: 0,
                profileImage: null,
                projects: [],
                projectsCount: 0,
                pullRequests: [],
                pullRequestsCount: 0,
                questions: [],
                questionsCount: 0,
                reminders: [],
                remindersCount: 0,
                reports: [],
                reportsCount: 0,
                roles: [],
                rolesCount: 0,
                routines: [],
                routinesCount: 0,
                standards: [],
                standardsCount: 0,
                tags: [],
                transfers: [],
                transfersCount: 0,
                translations: [],
                translationsCount: 0,
                you: {
                    __typename: "TeamYou",
                    canAddMembers: true,
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
            translations: [],
            translationsCount: 0,
            you: {
                __typename: "MeetingYou",
                canDelete: false,
                canInvite: false,
                canUpdate: false,
            },
        };
        
        return {
            ...defaultMeeting,
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
            id,
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
                    count: 0,
                },
            },
        };
        
        return {
            ...defaultUser,
            ...overrides,
        };
    }
    
    /**
     * Create mock meeting invite data
     */
    createMockMeetingInvite(overrides?: Partial<MeetingInvite>): MeetingInvite {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultMeetingInvite: MeetingInvite = {
            __typename: "MeetingInvite",
            id,
            createdAt: now,
            updatedAt: now,
            meeting: this.createMockMeeting(),
            message: "You're invited to join our team meeting. Please RSVP at your earliest convenience.",
            status: MeetingInviteStatusEnum.Pending,
            user: this.createMockUser(),
            you: {
                __typename: "MeetingInviteYou",
                canDelete: true,
                canUpdate: true,
            },
        };
        
        return {
            ...defaultMeetingInvite,
            ...overrides,
        };
    }
    
    /**
     * Create meeting invite from API input
     */
    createMeetingInviteFromInput(input: MeetingInviteCreateInput): MeetingInvite {
        const meetingInvite = this.createMockMeetingInvite();
        
        // Update meeting invite based on input
        meetingInvite.id = input.id;
        meetingInvite.meeting.id = input.meetingConnect;
        meetingInvite.user.id = input.userConnect;
        
        if (input.message) {
            meetingInvite.message = input.message;
        }
        
        return meetingInvite;
    }
    
    /**
     * Create multiple meeting invites for different statuses
     */
    createMeetingInvitesForAllStatuses(): MeetingInvite[] {
        return Object.values(MeetingInviteStatusEnum).map((status, index) => 
            this.createMockMeetingInvite({
                id: `${this.generateId()}_${index}`,
                status,
                user: this.createMockUser({
                    id: `user_${this.generateId()}_${index}`,
                    handle: `user${index + 1}`,
                    name: `User ${index + 1}`,
                }),
            }),
        );
    }
    
    /**
     * Create meeting invites with different meeting contexts
     */
    createMeetingInvitesWithVariedContexts(): MeetingInvite[] {
        const contexts = [
            {
                meetingName: "Weekly Team Standup",
                message: "Join us for our weekly standup to discuss progress and blockers.",
                isUrgent: false,
            },
            {
                meetingName: "Quarterly Planning Session",
                message: "Important quarterly planning session - attendance required.",
                isUrgent: true,
            },
            {
                meetingName: "Product Demo",
                message: "Demo of new features - optional attendance.",
                isUrgent: false,
            },
            {
                meetingName: "Emergency Bug Triage",
                message: "Urgent bug triage meeting - immediate attention required.",
                isUrgent: true,
            },
        ];
        
        return contexts.map((context, index) => 
            this.createMockMeetingInvite({
                id: `${this.generateId()}_${index}`,
                message: context.message,
                meeting: this.createMockMeeting({
                    id: `meeting_${this.generateId()}_${index}`,
                    translations: [{
                        __typename: "MeetingTranslation",
                        id: `trans_${this.generateId()}_${index}`,
                        language: "en",
                        name: context.meetingName,
                        description: context.message,
                    }],
                }),
                status: context.isUrgent ? MeetingInviteStatusEnum.Pending : MeetingInviteStatusEnum.Accepted,
            }),
        );
    }
    
    /**
     * Validate meeting invite create input
     */
    async validateCreateInput(input: MeetingInviteCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await meetingInviteValidation.create.validate(input);
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
     * Validate meeting invite update input
     */
    async validateUpdateInput(input: MeetingInviteUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await meetingInviteValidation.update.validate(input);
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
 * MSW handlers factory for meeting invite endpoints
 */
export class MeetingInviteMSWHandlers {
    private responseFactory: MeetingInviteResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new MeetingInviteResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all meeting invite endpoints
     */
    createSuccessHandlers(): RestHandler[] {
        return [
            // Create meeting invite
            rest.post(`${this.responseFactory["baseUrl"]}/api/meeting-invite`, async (req, res, ctx) => {
                const body = await req.json() as MeetingInviteCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {})),
                    );
                }
                
                // Create meeting invite
                const meetingInvite = this.responseFactory.createMeetingInviteFromInput(body);
                const response = this.responseFactory.createSuccessResponse(meetingInvite);
                
                return res(
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
            
            // Get meeting invite by ID
            rest.get(`${this.responseFactory["baseUrl"]}/api/meeting-invite/:id`, (req, res, ctx) => {
                const { id } = req.params;
                
                const meetingInvite = this.responseFactory.createMockMeetingInvite({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(meetingInvite);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Update meeting invite
            rest.put(`${this.responseFactory["baseUrl"]}/api/meeting-invite/:id`, async (req, res, ctx) => {
                const { id } = req.params;
                const body = await req.json() as MeetingInviteUpdateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateUpdateInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {})),
                    );
                }
                
                const meetingInvite = this.responseFactory.createMockMeetingInvite({ 
                    id: id as string,
                    message: body.message,
                    updatedAt: new Date().toISOString(),
                });
                
                const response = this.responseFactory.createSuccessResponse(meetingInvite);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // RSVP to meeting invite (accept/decline)
            rest.patch(`${this.responseFactory["baseUrl"]}/api/meeting-invite/:id/rsvp`, async (req, res, ctx) => {
                const { id } = req.params;
                const body = await req.json() as { status: MeetingInviteStatus };
                
                const meetingInvite = this.responseFactory.createMockMeetingInvite({ 
                    id: id as string,
                    status: body.status,
                    updatedAt: new Date().toISOString(),
                });
                
                const response = this.responseFactory.createSuccessResponse(meetingInvite);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Delete meeting invite
            rest.delete(`${this.responseFactory["baseUrl"]}/api/meeting-invite/:id`, (req, res, ctx) => {
                return res(ctx.status(204));
            }),
            
            // List meeting invites
            rest.get(`${this.responseFactory["baseUrl"]}/api/meeting-invite`, (req, res, ctx) => {
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const status = url.searchParams.get("status") as MeetingInviteStatus;
                const meetingId = url.searchParams.get("meetingId");
                const userId = url.searchParams.get("userId");
                
                let meetingInvites = this.responseFactory.createMeetingInvitesForAllStatuses();
                
                // Filter by status if specified
                if (status) {
                    meetingInvites = meetingInvites.filter(invite => invite.status === status);
                }
                
                // Filter by meeting ID if specified
                if (meetingId) {
                    meetingInvites = meetingInvites.filter(invite => invite.meeting.id === meetingId);
                }
                
                // Filter by user ID if specified
                if (userId) {
                    meetingInvites = meetingInvites.filter(invite => invite.user.id === userId);
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedMeetingInvites = meetingInvites.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createMeetingInviteListResponse(
                    paginatedMeetingInvites,
                    {
                        page,
                        pageSize: limit,
                        totalCount: meetingInvites.length,
                    },
                );
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Get meeting invites for specific meeting
            rest.get(`${this.responseFactory["baseUrl"]}/api/meeting/:meetingId/invites`, (req, res, ctx) => {
                const { meetingId } = req.params;
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                
                const meetingInvites = this.responseFactory.createMeetingInvitesForAllStatuses()
                    .map(invite => ({
                        ...invite,
                        meeting: { ...invite.meeting, id: meetingId as string },
                    }));
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedMeetingInvites = meetingInvites.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createMeetingInviteListResponse(
                    paginatedMeetingInvites,
                    {
                        page,
                        pageSize: limit,
                        totalCount: meetingInvites.length,
                    },
                );
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
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
            rest.post(`${this.responseFactory["baseUrl"]}/api/meeting-invite`, (req, res, ctx) => {
                return res(
                    ctx.status(400),
                    ctx.json(this.responseFactory.createValidationErrorResponse({
                        meetingConnect: "Meeting ID is required",
                        userConnect: "User ID is required",
                    })),
                );
            }),
            
            // Not found error
            rest.get(`${this.responseFactory["baseUrl"]}/api/meeting-invite/:id`, (req, res, ctx) => {
                const { id } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse(id as string)),
                );
            }),
            
            // Permission error
            rest.post(`${this.responseFactory["baseUrl"]}/api/meeting-invite`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse("create")),
                );
            }),
            
            // RSVP conflict error
            rest.patch(`${this.responseFactory["baseUrl"]}/api/meeting-invite/:id/rsvp`, async (req, res, ctx) => {
                const body = await req.json() as { status: MeetingInviteStatus };
                return res(
                    ctx.status(409),
                    ctx.json(this.responseFactory.createRSVPConflictErrorResponse(
                        MeetingInviteStatusEnum.Accepted,
                        body.status,
                    )),
                );
            }),
            
            // Meeting capacity error
            rest.patch(`${this.responseFactory["baseUrl"]}/api/meeting-invite/:id/rsvp`, (req, res, ctx) => {
                return res(
                    ctx.status(409),
                    ctx.json(this.responseFactory.createMeetingCapacityErrorResponse(100, 100)),
                );
            }),
            
            // Server error
            rest.post(`${this.responseFactory["baseUrl"]}/api/meeting-invite`, (req, res, ctx) => {
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
            rest.post(`${this.responseFactory["baseUrl"]}/api/meeting-invite`, async (req, res, ctx) => {
                const body = await req.json() as MeetingInviteCreateInput;
                const meetingInvite = this.responseFactory.createMeetingInviteFromInput(body);
                const response = this.responseFactory.createSuccessResponse(meetingInvite);
                
                return res(
                    ctx.delay(delay),
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
            
            rest.patch(`${this.responseFactory["baseUrl"]}/api/meeting-invite/:id/rsvp`, async (req, res, ctx) => {
                const { id } = req.params;
                const body = await req.json() as { status: MeetingInviteStatus };
                
                const meetingInvite = this.responseFactory.createMockMeetingInvite({ 
                    id: id as string,
                    status: body.status,
                    updatedAt: new Date().toISOString(),
                });
                
                const response = this.responseFactory.createSuccessResponse(meetingInvite);
                
                return res(
                    ctx.delay(delay),
                    ctx.status(200),
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
            rest.post(`${this.responseFactory["baseUrl"]}/api/meeting-invite`, (req, res, ctx) => {
                return res.networkError("Network connection failed");
            }),
            
            rest.get(`${this.responseFactory["baseUrl"]}/api/meeting-invite/:id`, (req, res, ctx) => {
                return res.networkError("Connection timeout");
            }),
            
            rest.patch(`${this.responseFactory["baseUrl"]}/api/meeting-invite/:id/rsvp`, (req, res, ctx) => {
                return res.networkError("RSVP update failed - network error");
            }),
        ];
    }
    
    /**
     * Create custom handler with specific configuration
     */
    createCustomHandler(config: {
        endpoint: string;
        method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
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
export const meetingInviteResponseScenarios = {
    // Success scenarios
    createSuccess: (meetingInvite?: MeetingInvite) => {
        const factory = new MeetingInviteResponseFactory();
        return factory.createSuccessResponse(
            meetingInvite || factory.createMockMeetingInvite(),
        );
    },
    
    listSuccess: (meetingInvites?: MeetingInvite[]) => {
        const factory = new MeetingInviteResponseFactory();
        return factory.createMeetingInviteListResponse(
            meetingInvites || factory.createMeetingInvitesForAllStatuses(),
        );
    },
    
    rsvpSuccess: (status: MeetingInviteStatus) => {
        const factory = new MeetingInviteResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockMeetingInvite({ status }),
        );
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new MeetingInviteResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                meetingConnect: "Meeting ID is required",
                userConnect: "User ID is required",
            },
        );
    },
    
    notFoundError: (meetingInviteId?: string) => {
        const factory = new MeetingInviteResponseFactory();
        return factory.createNotFoundErrorResponse(
            meetingInviteId || "non-existent-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new MeetingInviteResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
        );
    },
    
    rsvpConflictError: (currentStatus?: MeetingInviteStatus, requestedStatus?: MeetingInviteStatus) => {
        const factory = new MeetingInviteResponseFactory();
        return factory.createRSVPConflictErrorResponse(
            currentStatus || MeetingInviteStatusEnum.Accepted,
            requestedStatus || MeetingInviteStatusEnum.Pending,
        );
    },
    
    meetingCapacityError: (currentCount?: number, maxCapacity?: number) => {
        const factory = new MeetingInviteResponseFactory();
        return factory.createMeetingCapacityErrorResponse(
            currentCount || 50,
            maxCapacity || 50,
        );
    },
    
    serverError: () => {
        const factory = new MeetingInviteResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new MeetingInviteMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new MeetingInviteMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new MeetingInviteMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new MeetingInviteMSWHandlers().createNetworkErrorHandlers(),
    
    // Specific scenarios for meeting invites
    pendingInvites: () => {
        const factory = new MeetingInviteResponseFactory();
        const invites = factory.createMeetingInvitesForAllStatuses()
            .filter(invite => invite.status === MeetingInviteStatusEnum.Pending);
        return factory.createMeetingInviteListResponse(invites);
    },
    
    acceptedInvites: () => {
        const factory = new MeetingInviteResponseFactory();
        const invites = factory.createMeetingInvitesForAllStatuses()
            .filter(invite => invite.status === MeetingInviteStatusEnum.Accepted);
        return factory.createMeetingInviteListResponse(invites);
    },
    
    declinedInvites: () => {
        const factory = new MeetingInviteResponseFactory();
        const invites = factory.createMeetingInvitesForAllStatuses()
            .filter(invite => invite.status === MeetingInviteStatusEnum.Declined);
        return factory.createMeetingInviteListResponse(invites);
    },
    
    urgentMeetingInvites: () => {
        const factory = new MeetingInviteResponseFactory();
        const invites = factory.createMeetingInvitesWithVariedContexts()
            .filter(invite => invite.status === MeetingInviteStatusEnum.Pending);
        return factory.createMeetingInviteListResponse(invites);
    },
};

// Export factory instances for easy use
export const meetingInviteResponseFactory = new MeetingInviteResponseFactory();
export const meetingInviteMSWHandlers = new MeetingInviteMSWHandlers();
