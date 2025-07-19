/* c8 ignore start */
/**
 * Report Response API Response Fixtures
 * 
 * Comprehensive API response fixtures for report response endpoints, covering
 * moderation workflows, administrative actions, and reporting system scenarios.
 */

import type {
    Report,
    ReportResponse,
    ReportResponseCreateInput,
    ReportResponseUpdateInput,
    ReportStatus,
    ReportSuggestedAction,
    User,
} from "../../../api/types.js";
import { ReportSuggestedAction as ReportSuggestedActionEnum } from "../../../api/types.js";
import {
    ReportFor as ReportForEnum,
    ReportStatus as ReportStatusEnum,
} from "../../../run/enums.js";
import { BaseAPIResponseFactory } from "./base.js";
import type { MockDataOptions } from "./types.js";

// Constants for realistic data generation
const MODERATION_REASONS = [
    "Inappropriate content",
    "Harassment",
    "Spam",
    "Copyright violation",
    "Hate speech",
    "Privacy violation",
    "Technical issue",
    "False information",
] as const;

const ACTION_DETAILS = {
    [ReportSuggestedActionEnum.Delete]: "Content violated community guidelines and has been removed.",
    [ReportSuggestedActionEnum.FalseReport]: "Investigation found no violation. Report marked as false.",
    [ReportSuggestedActionEnum.HideUntilFixed]: "Content hidden until issues are resolved by the creator.",
    [ReportSuggestedActionEnum.NonIssue]: "Review determined content does not violate any guidelines.",
    [ReportSuggestedActionEnum.SuspendUser]: "User account suspended for repeated violations.",
} as const;

const MODERATOR_ROLES = ["moderator", "admin", "super_admin"] as const;
const HOURS_IN_MS = 60 * 60 * 1000;
const PREMIUM_EXPIRY_DAYS = 365;
const MINUTES_IN_MS = 60 * 1000;

/**
 * Factory for generating ReportResponse API responses
 */
export class ReportResponseAPIResponseFactory extends BaseAPIResponseFactory<
    ReportResponse,
    ReportResponseCreateInput,
    ReportResponseUpdateInput
> {
    protected readonly entityName = "report-response";

    /**
     * Generate a realistic moderation reason
     */
    private generateModerationReason(): string {
        return MODERATION_REASONS[Math.floor(Math.random() * MODERATION_REASONS.length)];
    }

    /**
     * Generate realistic details based on action
     */
    private generateActionDetails(action: ReportSuggestedAction, customDetails?: string): string {
        if (customDetails) return customDetails;

        const baseDetail = ACTION_DETAILS[action] || "Moderation action taken.";
        const timestamp = new Date().toISOString().toLocaleDateString();
        return `${baseDetail} Action taken on ${timestamp}.`;
    }

    /**
     * Create mock user with moderation role
     */
    private createMockModerator(role: typeof MODERATOR_ROLES[number] = "moderator"): User {
        const now = new Date().toISOString();
        const id = this.generateId();
        const handle = `${role}_${id.slice(-8)}`;

        return {
            __typename: "User",
            id: `user_${id}`,
            handle,
            name: `${role.charAt(0).toUpperCase() + role.slice(1)} User`,
            createdAt: now,
            updatedAt: now,
            isBot: false,
            isPrivate: false,
            profileImage: null,
            bannerImage: null,
            premium: role !== "moderator",
            premiumExpiration: role !== "moderator" ?
                new Date(Date.now().toISOString() + PREMIUM_EXPIRY_DAYS * 24 * HOURS_IN_MS).toISOString() : null,
            roles: role === "admin" ? ["Admin"] :
                role === "super_admin" ? ["SuperAdmin"] :
                    ["Moderator"],
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
    }

    /**
     * Create mock report
     */
    private createMockReport(overrides?: Partial<Report>): Report {
        const now = new Date().toISOString();
        const id = this.generateId();
        const reason = this.generateModerationReason();

        const defaultReport: Report = {
            __typename: "Report",
            id: `report_${id}`,
            publicId: `pub_${id}`,
            createdAt: now,
            updatedAt: now,
            createdFor: ReportForEnum.User,
            reason,
            details: `Detailed report about ${reason.toLowerCase()}. User behavior observed multiple times.`,
            language: "en",
            status: ReportStatusEnum.Open,
            responses: [],
            responsesCount: 0,
            you: {
                __typename: "ReportYou",
                canDelete: false,
                canUpdate: false,
                canRespond: true,
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
    createMockData(options?: MockDataOptions): ReportResponse {
        const now = new Date().toISOString();
        const id = this.generateId();
        const action = options?.overrides?.actionSuggested as ReportSuggestedAction ||
            ReportSuggestedActionEnum.HideUntilFixed;

        const baseResponse: ReportResponse = {
            __typename: "ReportResponse",
            id: `response_${id}`,
            createdAt: now,
            updatedAt: now,
            actionSuggested: action,
            details: this.generateActionDetails(action, options?.overrides?.details as string),
            language: "en",
            report: this.createMockReport(),
            you: {
                __typename: "ReportResponseYou",
                canDelete: true,
                canUpdate: true,
            },
        };

        // Apply scenario-specific overrides
        if (options?.scenario) {
            switch (options.scenario) {
                case "minimal":
                    baseResponse.details = undefined;
                    baseResponse.you.canDelete = false;
                    baseResponse.you.canUpdate = false;
                    break;
                case "complete":
                    baseResponse.details = this.generateActionDetails(action) +
                        " Additional context: Reviewed by senior moderator. User notified via email.";
                    baseResponse.report = this.createMockReport({
                        status: this.getStatusForAction(action),
                        responsesCount: 1,
                    });
                    break;
                case "edge-case":
                    baseResponse.actionSuggested = ReportSuggestedActionEnum.FalseReport;
                    baseResponse.details = "Edge case: Automated system flagged legitimate content.";
                    baseResponse.report = this.createMockReport({
                        status: ReportStatusEnum.ClosedFalseReport,
                    });
                    break;
            }
        }

        // Apply explicit overrides
        if (options?.overrides) {
            Object.assign(baseResponse, options.overrides);
        }

        return baseResponse;
    }

    /**
     * Get appropriate status for an action
     */
    private getStatusForAction(action: ReportSuggestedAction): ReportStatus {
        switch (action) {
            case ReportSuggestedActionEnum.Delete:
                return ReportStatusEnum.ClosedDeleted;
            case ReportSuggestedActionEnum.FalseReport:
                return ReportStatusEnum.ClosedFalseReport;
            case ReportSuggestedActionEnum.HideUntilFixed:
                return ReportStatusEnum.ClosedHidden;
            case ReportSuggestedActionEnum.NonIssue:
                return ReportStatusEnum.ClosedNonIssue;
            case ReportSuggestedActionEnum.SuspendUser:
                return ReportStatusEnum.ClosedSuspended;
            default:
                return ReportStatusEnum.Open;
        }
    }

    /**
     * Create entity from create input
     */
    createFromInput(input: ReportResponseCreateInput): ReportResponse {
        const response = this.createMockData();

        // Update based on input
        if (input.id) response.id = input.id;
        response.actionSuggested = input.actionSuggested;
        if (input.details) response.details = input.details;
        if (input.language) response.language = input.language;

        // Connect to report
        if (input.reportConnect) {
            response.report.id = input.reportConnect;
            response.report.status = this.getStatusForAction(input.actionSuggested);
        }

        return response;
    }

    /**
     * Update entity from update input
     */
    updateFromInput(existing: ReportResponse, input: ReportResponseUpdateInput): ReportResponse {
        const updated = { ...existing };
        updated.updatedAt = new Date().toISOString();

        if (input.actionSuggested !== undefined) {
            updated.actionSuggested = input.actionSuggested;
            updated.report.status = this.getStatusForAction(input.actionSuggested);
        }

        if (input.details !== undefined) {
            updated.details = input.details || undefined;
        }

        if (input.language !== undefined) {
            updated.language = input.language;
        }

        return updated;
    }

    /**
     * Validate create input
     */
    async validateCreateInput(input: ReportResponseCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.reportConnect) {
            errors.reportConnect = "Report connection is required";
        }

        if (!input.actionSuggested) {
            errors.actionSuggested = "Suggested action is required";
        } else if (!Object.values(ReportSuggestedActionEnum).includes(input.actionSuggested)) {
            errors.actionSuggested = "Invalid suggested action";
        }

        if (input.language && !/^[a-z]{2}(-[A-Z]{2})?$/.test(input.language)) {
            errors.language = "Language must be a valid language code (e.g., 'en', 'en-US')";
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: ReportResponseUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.id) {
            errors.id = "Report response ID is required for updates";
        }

        if (input.actionSuggested && !Object.values(ReportSuggestedActionEnum).includes(input.actionSuggested)) {
            errors.actionSuggested = "Invalid suggested action";
        }

        if (input.language && !/^[a-z]{2}(-[A-Z]{2})?$/.test(input.language)) {
            errors.language = "Language must be a valid language code";
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create responses for all action types
     */
    createAllActionTypeResponses(): ReportResponse[] {
        return Object.values(ReportSuggestedActionEnum).map(action =>
            this.createMockData({
                overrides: {
                    actionSuggested: action,
                    details: this.generateActionDetails(action),
                    report: this.createMockReport({
                        status: this.getStatusForAction(action),
                    }),
                },
                scenario: "complete",
            }),
        );
    }

    /**
     * Create moderation workflow responses
     */
    createModerationWorkflow(reportId?: string): ReportResponse[] {
        const baseReport = this.createMockReport({
            id: reportId,
            createdFor: ReportForEnum.ChatMessage,
            reason: "Harassment",
            details: "User is sending threatening messages to other users",
            status: ReportStatusEnum.Open,
        });

        // Initial assessment
        const initialResponse = this.createMockData({
            overrides: {
                actionSuggested: ReportSuggestedActionEnum.HideUntilFixed,
                details: "Content temporarily hidden pending further investigation. User has been notified.",
                createdAt: new Date(Date.now().toISOString() - 4 * HOURS_IN_MS).toISOString(), // 4 hours ago
                report: { ...baseReport, status: ReportStatusEnum.Open },
            },
            scenario: "complete",
        });

        // Follow-up investigation
        const followupResponse = this.createMockData({
            overrides: {
                actionSuggested: ReportSuggestedActionEnum.SuspendUser,
                details: "Investigation confirmed pattern of harassment. User suspended for 7 days. Warning issued.",
                createdAt: new Date(Date.now().toISOString() - 1 * HOURS_IN_MS).toISOString(), // 1 hour ago
                report: { ...baseReport, status: ReportStatusEnum.ClosedSuspended },
            },
            scenario: "complete",
        });

        return [initialResponse, followupResponse];
    }

    /**
     * Create escalation workflow responses
     */
    createEscalationWorkflow(): ReportResponse[] {
        const reportId = this.generateId();

        // Moderator response
        const moderatorResponse = this.createMockData({
            overrides: {
                actionSuggested: ReportSuggestedActionEnum.NonIssue,
                details: "Initial review found no clear violation. Escalating to senior moderator.",
                createdAt: new Date(Date.now().toISOString() - 6 * HOURS_IN_MS).toISOString(),
                report: this.createMockReport({
                    id: reportId,
                    status: ReportStatusEnum.Open,
                }),
            },
        });

        // Senior moderator response
        const seniorResponse = this.createMockData({
            overrides: {
                actionSuggested: ReportSuggestedActionEnum.Delete,
                details: "Senior review identified subtle policy violation. Content removed and user educated.",
                createdAt: new Date(Date.now().toISOString() - 2 * HOURS_IN_MS).toISOString(),
                report: this.createMockReport({
                    id: reportId,
                    status: ReportStatusEnum.ClosedDeleted,
                }),
            },
        });

        return [moderatorResponse, seniorResponse];
    }

    /**
     * Create business error for unauthorized moderation
     */
    createUnauthorizedModerationError(): any {
        return this.createPermissionErrorResponse(
            "moderate",
            ["report:moderate", "admin:moderate"],
        );
    }

    /**
     * Create business error for invalid report state
     */
    createInvalidReportStateError(currentStatus: ReportStatus, attemptedAction: ReportSuggestedAction): any {
        return this.createBusinessErrorResponse("state", {
            reason: "Invalid action for current report state",
            currentStatus,
            attemptedAction,
            validActions: this.getValidActionsForStatus(currentStatus),
        });
    }

    /**
     * Get valid actions for a report status
     */
    private getValidActionsForStatus(status: ReportStatus): ReportSuggestedAction[] {
        switch (status) {
            case ReportStatusEnum.Open:
                return Object.values(ReportSuggestedActionEnum);
            case ReportStatusEnum.ClosedDeleted:
            case ReportStatusEnum.ClosedFalseReport:
            case ReportStatusEnum.ClosedHidden:
            case ReportStatusEnum.ClosedNonIssue:
            case ReportStatusEnum.ClosedSuspended:
                return []; // Terminal states - no further actions allowed
            default:
                return [];
        }
    }

    /**
     * Create business error for duplicate action
     */
    createDuplicateActionError(existingAction: ReportSuggestedAction): any {
        return this.createBusinessErrorResponse("conflict", {
            reason: "Action already taken on this report",
            existingAction,
            message: `Report already has a response with action: ${existingAction}`,
        });
    }
}

/**
 * Pre-configured response scenarios for common use cases
 */
export const reportResponseResponseScenarios = {
    // Success scenarios
    createSuccess: (response?: ReportResponse) => {
        const factory = new ReportResponseAPIResponseFactory();
        return factory.createSuccessResponse(
            response || factory.createMockData(),
        );
    },

    listSuccess: (responses?: ReportResponse[], pagination?: { page: number; pageSize: number; totalCount: number }) => {
        const factory = new ReportResponseAPIResponseFactory();
        return factory.createPaginatedResponse(
            responses || factory.createAllActionTypeResponses(),
            pagination || { page: 1, pageSize: 20, totalCount: 50 },
        );
    },

    moderationWorkflow: (reportId?: string) => {
        const factory = new ReportResponseAPIResponseFactory();
        const workflow = factory.createModerationWorkflow(reportId);
        return factory.createPaginatedResponse(
            workflow,
            { page: 1, pageSize: workflow.length, totalCount: workflow.length },
        );
    },

    escalationWorkflow: () => {
        const factory = new ReportResponseAPIResponseFactory();
        const workflow = factory.createEscalationWorkflow();
        return factory.createPaginatedResponse(
            workflow,
            { page: 1, pageSize: workflow.length, totalCount: workflow.length },
        );
    },

    deleteActionResponse: () => {
        const factory = new ReportResponseAPIResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    actionSuggested: ReportSuggestedActionEnum.Delete,
                },
                scenario: "complete",
            }),
        );
    },

    suspendUserResponse: () => {
        const factory = new ReportResponseAPIResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    actionSuggested: ReportSuggestedActionEnum.SuspendUser,
                },
                scenario: "complete",
            }),
        );
    },

    falseReportResponse: () => {
        const factory = new ReportResponseAPIResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    actionSuggested: ReportSuggestedActionEnum.FalseReport,
                },
                scenario: "complete",
            }),
        );
    },

    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new ReportResponseAPIResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                reportConnect: "Report connection is required",
                actionSuggested: "Suggested action is required",
                language: "Language must be a valid language code",
            },
        );
    },

    notFoundError: (responseId?: string) => {
        const factory = new ReportResponseAPIResponseFactory();
        return factory.createNotFoundErrorResponse(
            responseId || "non-existent-response-id",
            "report response",
        );
    },

    unauthorizedModerationError: () => {
        const factory = new ReportResponseAPIResponseFactory();
        return factory.createUnauthorizedModerationError();
    },

    invalidReportStateError: (currentStatus?: ReportStatus, attemptedAction?: ReportSuggestedAction) => {
        const factory = new ReportResponseAPIResponseFactory();
        return factory.createInvalidReportStateError(
            currentStatus || ReportStatusEnum.ClosedDeleted,
            attemptedAction || ReportSuggestedActionEnum.Delete,
        );
    },

    duplicateActionError: (existingAction?: ReportSuggestedAction) => {
        const factory = new ReportResponseAPIResponseFactory();
        return factory.createDuplicateActionError(
            existingAction || ReportSuggestedActionEnum.HideUntilFixed,
        );
    },

    serverError: () => {
        const factory = new ReportResponseAPIResponseFactory();
        return factory.createServerErrorResponse("report-response-service", "moderate");
    },

    rateLimitError: () => {
        const factory = new ReportResponseAPIResponseFactory();
        const resetTime = new Date(Date.now().toISOString() + 15 * MINUTES_IN_MS); // 15 minutes from now
        return factory.createRateLimitErrorResponse(20, 0, resetTime);
    },
};

// Export factory instance for direct use
export const reportResponseAPIResponseFactory = new ReportResponseAPIResponseFactory();

