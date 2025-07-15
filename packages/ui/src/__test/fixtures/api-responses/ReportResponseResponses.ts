/**
 * ReportResponse API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for report response endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */
// AI_CHECK: TYPE_SAFETY=fixed-premium-property | LAST: 2025-07-02 - Fixed Premium type error by removing 'createdAt' property

import { http, type HttpHandler, HttpResponse } from "msw";
import type {
    ReportResponse,
    ReportResponseCreateInput,
    ReportResponseUpdateInput,
    Report,
    ReportFor,
    ReportStatus,
    ReportSuggestedAction,
    User,
} from "@vrooli/shared";
import {
    reportResponseValidation,
    ReportFor as ReportForEnum,
    ReportStatus as ReportStatusEnum,
    ReportSuggestedAction as ReportSuggestedActionEnum,
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
 * ReportResponse API response factory
 */
export class ReportResponseResponseFactory {
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
     * Create successful report response
     */
    createSuccessResponse(reportResponse: ReportResponse): APIResponse<ReportResponse> {
        return {
            data: reportResponse,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/report-response/${reportResponse.id}`,
                    related: {
                        report: `${this.baseUrl}/api/report/${reportResponse.report.id}`,
                        reportedObject: `${this.baseUrl}/api/${reportResponse.report.createdFor.toLowerCase()}/${reportResponse.report.publicId}`,
                    },
                },
            },
        };
    }

    /**
     * Create report response list response
     */
    createReportResponseListResponse(responses: ReportResponse[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<ReportResponse> {
        const paginationData = pagination || {
            page: 1,
            pageSize: responses.length,
            totalCount: responses.length,
        };

        return {
            data: responses,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/report-response?page=${paginationData.page}&limit=${paginationData.pageSize}`,
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
                path: "/api/report-response",
            },
        };
    }

    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(responseId: string): APIErrorResponse {
        return {
            error: {
                code: "REPORT_RESPONSE_NOT_FOUND",
                message: `Report response with ID '${responseId}' was not found`,
                details: {
                    responseId,
                    searchCriteria: { id: responseId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/report-response/${responseId}`,
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
                message: `You do not have permission to ${operation} report responses`,
                details: {
                    operation,
                    requiredPermissions: ["report:moderate"],
                    userPermissions: ["report:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/report-response",
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
                path: "/api/report-response",
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
                path: "/api/report-response",
            },
        };
    }

    /**
     * Create mock user data
     */
    private createMockUser(role: "moderator" | "admin" | "user" = "moderator"): User {
        const now = new Date().toISOString();
        const id = this.generateId();

        return {
            __typename: "User",
            id: `user_${id}`,
            handle: `${role}user`,
            name: `${role.charAt(0).toUpperCase() + role.slice(1)} User`,
            createdAt: now,
            updatedAt: now,
            isBot: false,
            isBotDepictingPerson: false,
            isPrivate: false,
            isPrivateBookmarks: true,
            isPrivateMemberships: true,
            isPrivatePullRequests: true,
            isPrivateResources: false,
            isPrivateResourcesCreated: false,
            isPrivateTeamsCreated: true,
            isPrivateVotes: true,
            bookmarkedBy: [],
            bookmarks: 0,
            profileImage: null,
            bannerImage: null,
            publicId: `user_public_${id}`,
            views: 0,
            membershipsCount: 0,
            reportsReceived: [],
            reportsReceivedCount: 0,
            resourcesCount: 0,
            premium: role !== "user" ? {
                __typename: "Premium",
                id: `premium_${this.generateId()}`,
                credits: 1000,
                customPlan: null,
                enabledAt: new Date().toISOString(),
                expiresAt: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000).toISOString(),
            } : null,
            wallets: [],
            translations: [],
            you: {
                __typename: "UserYou",
                canDelete: false,
                canReport: false,
                canUpdate: false,
                isBookmarked: false,
                isViewed: false,
            },
        };
    }

    /**
     * Create mock report data
     */
    private createMockReport(overrides?: Partial<Report>): Report {
        const now = new Date().toISOString();
        const id = this.generateId();

        const defaultReport: Report = {
            __typename: "Report",
            id: `report_${id}`,
            publicId: `pub_${id}`,
            createdAt: now,
            updatedAt: now,
            createdFor: ReportForEnum.User,
            reason: "Inappropriate content",
            details: "User is posting spam and offensive material",
            language: "en",
            status: ReportStatusEnum.Open,
            responses: [],
            responsesCount: 0,
            you: {
                __typename: "ReportYou",
                canDelete: false,
                canUpdate: false,
                canRespond: true,
                isOwn: false,
            },
        };

        return {
            ...defaultReport,
            ...overrides,
        };
    }

    /**
     * Create mock report response data
     */
    createMockReportResponse(overrides?: Partial<ReportResponse>): ReportResponse {
        const now = new Date().toISOString();
        const id = this.generateId();

        const defaultResponse: ReportResponse = {
            __typename: "ReportResponse",
            id: `response_${id}`,
            createdAt: now,
            updatedAt: now,
            actionSuggested: ReportSuggestedActionEnum.HideUntilFixed,
            details: "Content has been hidden pending review. User has been notified.",
            language: "en",
            report: this.createMockReport(),
            you: {
                __typename: "ReportResponseYou",
                canDelete: true,
                canUpdate: true,
            },
        };

        return {
            ...defaultResponse,
            ...overrides,
        };
    }

    /**
     * Create report response from API input
     */
    createReportResponseFromInput(input: ReportResponseCreateInput): ReportResponse {
        const response = this.createMockReportResponse();

        // Update response based on input
        response.id = input.id;
        response.actionSuggested = input.actionSuggested;
        response.details = input.details || undefined;
        response.language = input.language || "en";

        // Connect to report
        response.report.id = input.reportConnect;

        return response;
    }

    /**
     * Create multiple report responses for different scenarios
     */
    createReportResponseScenarios(): ReportResponse[] {
        const scenarios: Array<{
            action: ReportSuggestedAction;
            status: ReportStatus;
            details: string;
        }> = [
            {
                action: ReportSuggestedActionEnum.Delete,
                status: ReportStatusEnum.ClosedDeleted,
                details: "Content violated community guidelines and has been removed.",
            },
            {
                action: ReportSuggestedActionEnum.FalseReport,
                status: ReportStatusEnum.ClosedFalseReport,
                details: "Investigation found no violation. Report marked as false.",
            },
            {
                action: ReportSuggestedActionEnum.HideUntilFixed,
                status: ReportStatusEnum.ClosedHidden,
                details: "Content hidden until issues are resolved by the creator.",
            },
            {
                action: ReportSuggestedActionEnum.NonIssue,
                status: ReportStatusEnum.ClosedNonIssue,
                details: "Review determined content does not violate any guidelines.",
            },
            {
                action: ReportSuggestedActionEnum.SuspendUser,
                status: ReportStatusEnum.ClosedSuspended,
                details: "User account suspended for repeated violations.",
            },
        ];

        return scenarios.map(scenario => 
            this.createMockReportResponse({
                actionSuggested: scenario.action,
                details: scenario.details,
                report: this.createMockReport({
                    status: scenario.status,
                }),
            }),
        );
    }

    /**
     * Create moderator workflow response
     */
    createModeratorWorkflowResponse(): ReportResponse[] {
        const report = this.createMockReport({
            createdFor: ReportForEnum.ChatMessage,
            reason: "Harassment",
            details: "User is sending threatening messages",
        });

        // Initial review
        const initialReview = this.createMockReportResponse({
            report,
            actionSuggested: ReportSuggestedActionEnum.HideUntilFixed,
            details: "Content temporarily hidden. Investigating further.",
            createdAt: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
        });

        // Final action
        const finalAction = this.createMockReportResponse({
            report: {
                ...report,
                status: ReportStatusEnum.ClosedSuspended,
            },
            actionSuggested: ReportSuggestedActionEnum.SuspendUser,
            details: "User suspended for 7 days due to harassment violation.",
            createdAt: new Date().toISOString(),
        });

        return [initialReview, finalAction];
    }

    /**
     * Validate report response create input
     */
    async validateCreateInput(input: ReportResponseCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await reportResponseValidation.create({}).validate(input);
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
 * MSW handlers factory for report response endpoints
 */
export class ReportResponseMSWHandlers {
    private responseFactory: ReportResponseResponseFactory;

    constructor(baseUrl?: string) {
        this.responseFactory = new ReportResponseResponseFactory(baseUrl);
    }

    /**
     * Create success handlers for all report response endpoints
     */
    createSuccessHandlers(): HttpHandler[] {
        return [
            // Create report response
            http.post(`${this.responseFactory["baseUrl"]}/api/report-response`, async ({ request }) => {
                const body = await request.json() as ReportResponseCreateInput;

                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return HttpResponse.json(
                        this.responseFactory.createValidationErrorResponse(validation.errors || {}),
                        { status: 400 }
                    );
                }

                // Create response
                const reportResponse = this.responseFactory.createReportResponseFromInput(body);
                const response = this.responseFactory.createSuccessResponse(reportResponse);

                return HttpResponse.json(response, { status: 201 });
            }),

            // Get report response by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/report-response/:id`, ({ params }) => {
                const { id } = params;

                const reportResponse = this.responseFactory.createMockReportResponse({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(reportResponse);

                return HttpResponse.json(response, { status: 200 });
            }),

            // Update report response
            http.put(`${this.responseFactory["baseUrl"]}/api/report-response/:id`, async ({ request, params }) => {
                const { id } = params;
                const body = await request.json() as ReportResponseUpdateInput;

                const reportResponse = this.responseFactory.createMockReportResponse({
                    id: id as string,
                    updatedAt: new Date().toISOString(),
                    actionSuggested: body.actionSuggested || ReportSuggestedActionEnum.NonIssue,
                    details: body.details || undefined,
                });

                const response = this.responseFactory.createSuccessResponse(reportResponse);

                return HttpResponse.json(response, { status: 200 });
            }),

            // Delete report response
            http.delete(`${this.responseFactory["baseUrl"]}/api/report-response/:id`, () => {
                return new HttpResponse(null, { status: 204 });
            }),

            // List report responses
            http.get(`${this.responseFactory["baseUrl"]}/api/report-response`, ({ request }) => {
                const url = new URL(request.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const reportId = url.searchParams.get("reportId");

                let responses = this.responseFactory.createReportResponseScenarios();

                // Filter by report if specified
                if (reportId) {
                    responses = responses.filter(r => r.report.id === reportId);
                }

                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedResponses = responses.slice(startIndex, startIndex + limit);

                const response = this.responseFactory.createReportResponseListResponse(
                    paginatedResponses,
                    {
                        page,
                        pageSize: limit,
                        totalCount: responses.length,
                    },
                );

                return HttpResponse.json(response, { status: 200 });
            }),

            // Get report responses for a specific report
            http.get(`${this.responseFactory["baseUrl"]}/api/report/:reportId/responses`, ({ params }) => {
                const { reportId } = params;

                const responses = this.responseFactory.createModeratorWorkflowResponse();
                const response = this.responseFactory.createReportResponseListResponse(responses);

                return HttpResponse.json(response, { status: 200 });
            }),
        ];
    }

    /**
     * Create error handlers for testing error scenarios
     */
    createErrorHandlers(): HttpHandler[] {
        return [
            // Validation error
            http.post(`${this.responseFactory["baseUrl"]}/api/report-response`, () => {
                return HttpResponse.json(
                    this.responseFactory.createValidationErrorResponse({
                        reportConnect: "Report ID is required",
                        actionSuggested: "Action must be specified",
                    }),
                    { status: 400 }
                );
            }),

            // Not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/report-response/:id`, ({ params }) => {
                const { id } = params;
                return HttpResponse.json(
                    this.responseFactory.createNotFoundErrorResponse(id as string),
                    { status: 404 }
                );
            }),

            // Permission error
            http.post(`${this.responseFactory["baseUrl"]}/api/report-response`, () => {
                return HttpResponse.json(
                    this.responseFactory.createPermissionErrorResponse("create"),
                    { status: 403 }
                );
            }),

            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/report-response`, () => {
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
    createLoadingHandlers(delay = 2000): HttpHandler[] {
        return [
            http.post(`${this.responseFactory["baseUrl"]}/api/report-response`, async ({ request }) => {
                const body = await request.json() as ReportResponseCreateInput;
                const reportResponse = this.responseFactory.createReportResponseFromInput(body);
                const response = this.responseFactory.createSuccessResponse(reportResponse);

                // Simulate delay
                await new Promise(resolve => setTimeout(resolve, delay));

                return HttpResponse.json(response, { status: 201 });
            }),
        ];
    }

    /**
     * Create network error handlers
     */
    createNetworkErrorHandlers(): HttpHandler[] {
        return [
            http.post(`${this.responseFactory["baseUrl"]}/api/report-response`, () => {
                return HttpResponse.error();
            }),

            http.get(`${this.responseFactory["baseUrl"]}/api/report-response/:id`, () => {
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
    }): HttpHandler {
        const { endpoint, method, status, response, delay } = config;
        const fullEndpoint = `${this.responseFactory["baseUrl"]}${endpoint}`;

        const httpMethod = method.toLowerCase() as keyof typeof http;
        return http[httpMethod](fullEndpoint, async () => {
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
export const reportResponseScenarios = {
    // Success scenarios
    createSuccess: (response?: ReportResponse) => {
        const factory = new ReportResponseResponseFactory();
        return factory.createSuccessResponse(
            response || factory.createMockReportResponse(),
        );
    },

    listSuccess: (responses?: ReportResponse[]) => {
        const factory = new ReportResponseResponseFactory();
        return factory.createReportResponseListResponse(
            responses || factory.createReportResponseScenarios(),
        );
    },

    moderatorWorkflow: () => {
        const factory = new ReportResponseResponseFactory();
        return factory.createReportResponseListResponse(
            factory.createModeratorWorkflowResponse(),
        );
    },

    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new ReportResponseResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                reportConnect: "Report ID is required",
                actionSuggested: "Action must be specified",
            },
        );
    },

    notFoundError: (responseId?: string) => {
        const factory = new ReportResponseResponseFactory();
        return factory.createNotFoundErrorResponse(
            responseId || "non-existent-id",
        );
    },

    permissionError: (operation?: string) => {
        const factory = new ReportResponseResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
        );
    },

    serverError: () => {
        const factory = new ReportResponseResponseFactory();
        return factory.createServerErrorResponse();
    },

    // MSW handlers
    successHandlers: () => new ReportResponseMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new ReportResponseMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new ReportResponseMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new ReportResponseMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const reportResponseFactory = new ReportResponseResponseFactory();
export const reportResponseMSWHandlers = new ReportResponseMSWHandlers();
