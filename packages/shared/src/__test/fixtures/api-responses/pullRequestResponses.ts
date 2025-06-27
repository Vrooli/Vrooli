/* c8 ignore start */
/**
 * Pull Request API Response Fixtures
 * 
 * Comprehensive API response fixtures for pull request endpoints, including
 * success responses, error scenarios, and MSW handlers for testing.
 */

import type { 
    PullRequest, 
    PullRequestCreateInput, 
    PullRequestUpdateInput,
    PullRequestStatus,
    PullRequestTranslation,
    User,
    ResourceVersion,
    Resource,
} from "../../../api/types.js";
import { 
    PullRequestStatus as PullRequestStatusEnum,
} from "../../../run/enums.js";
import type { MockDataOptions } from "./types.js";
import { BaseAPIResponseFactory } from "./base.js";

// Constants for realistic data generation
const PR_PREFIXES = ["PR", "MR", "FEAT", "FIX", "DOCS"] as const;
const DEFAULT_COMMENT_COUNT = 0;
const MERGE_CONFLICT_FILES = ["src/main.ts", "package.json", "README.md"] as const;
const REVIEW_COMMENT_COUNT = 2;
const APPROVED_COMMENT_COUNT = 3;
const MERGED_COMMENT_COUNT = 4;
const MINUTES_IN_MS = 60 * 1000;

/**
 * Factory for generating PullRequest API responses
 */
export class PullRequestAPIResponseFactory extends BaseAPIResponseFactory<
    PullRequest,
    PullRequestCreateInput,
    PullRequestUpdateInput
> {
    protected readonly entityName = "pull-request";

    /**
     * Generate a realistic public ID for pull requests
     */
    private generatePublicId(): string {
        const prefix = PR_PREFIXES[Math.floor(Math.random() * PR_PREFIXES.length)];
        const number = Math.floor(Math.random() * 9999) + 1;
        return `${prefix}-${number}`;
    }

    /**
     * Create mock user data
     */
    private createMockUser(id?: string): User {
        const userId = id || this.generateId();
        const now = new Date().toISOString();
        const handle = `user-${userId.slice(-8)}`;
        
        return {
            __typename: "User",
            id: userId,
            handle,
            name: `User ${handle}`,
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
    }

    /**
     * Create mock resource version (source of PR)
     */
    private createMockResourceVersion(id?: string): ResourceVersion {
        const versionId = id || this.generateId();
        const now = new Date().toISOString();
        
        return {
            __typename: "ResourceVersion",
            id: versionId,
            createdAt: now,
            updatedAt: now,
            commentsCount: DEFAULT_COMMENT_COUNT,
            directoryListingsCount: 0,
            forksCount: 0,
            isLatest: true,
            isPrivate: false,
            reportsCount: 0,
            versionIndex: 1,
            versionLabel: "1.0.0",
            comments: [],
            translations: [],
            translationsCount: 0,
            you: {
                __typename: "ResourceVersionYou",
                canComment: false,
                canDelete: false,
                canReport: false,
                canUpdate: false,
                canUse: true,
                canRead: true,
                isBookmarked: false,
                isReacted: false,
                reaction: null,
            },
        };
    }

    /**
     * Create mock resource (target of PR)
     */
    private createMockResource(id?: string): Resource {
        const resourceId = id || this.generateId();
        const now = new Date().toISOString();
        
        return {
            __typename: "Resource",
            id: resourceId,
            createdAt: now,
            updatedAt: now,
            isInternal: false,
            isPrivate: false,
            usedBy: [],
            usedByCount: 0,
            versions: [],
            versionsCount: 1,
            you: {
                __typename: "ResourceYou",
                canDelete: false,
                canUpdate: false,
                canReport: false,
                isBookmarked: false,
                isReacted: false,
                reaction: null,
            },
        };
    }

    /**
     * Create mock pull request data
     */
    createMockData(options?: MockDataOptions): PullRequest {
        const now = new Date().toISOString();
        const id = this.generateId();
        const status = options?.overrides?.status as PullRequestStatus || PullRequestStatusEnum.Open;
        
        // Create default translation
        const defaultTranslation: PullRequestTranslation = {
            __typename: "PullRequestTranslation",
            id: `trans_${id}`,
            language: "en",
            text: "This pull request improves the system functionality with bug fixes and enhancements.",
        };

        const basePR: PullRequest = {
            __typename: "PullRequest",
            id,
            createdAt: now,
            updatedAt: now,
            publicId: this.generatePublicId(),
            status,
            closedAt: [PullRequestStatusEnum.Merged, PullRequestStatusEnum.Rejected, PullRequestStatusEnum.Canceled]
                .includes(status) ? now : null,
            createdBy: options?.withRelations !== false ? this.createMockUser() : null,
            from: this.createMockResourceVersion(),
            to: this.createMockResource(),
            comments: [],
            commentsCount: DEFAULT_COMMENT_COUNT,
            translations: [defaultTranslation],
            translationsCount: 1,
            you: {
                __typename: "PullRequestYou",
                canComment: true,
                canDelete: false,
                canReport: false,
                canUpdate: false,
            },
        };

        // Apply scenario-specific overrides
        if (options?.scenario) {
            switch (options.scenario) {
                case "minimal":
                    basePR.translations = [];
                    basePR.translationsCount = 0;
                    basePR.createdBy = null;
                    break;
                case "complete":
                    basePR.commentsCount = APPROVED_COMMENT_COUNT;
                    basePR.you.canUpdate = true;
                    basePR.you.canDelete = true;
                    break;
                case "edge-case":
                    basePR.status = PullRequestStatusEnum.Draft;
                    basePR.you.canUpdate = true;
                    basePR.you.canDelete = true;
                    break;
            }
        }

        // Apply explicit overrides
        if (options?.overrides) {
            Object.assign(basePR, options.overrides);
        }

        return basePR;
    }

    /**
     * Create entity from create input
     */
    createFromInput(input: PullRequestCreateInput): PullRequest {
        const pullRequest = this.createMockData();
        
        // Update based on input
        if (input.id) pullRequest.id = input.id;
        if (input.fromConnect) pullRequest.from.id = input.fromConnect;
        if (input.toConnect) pullRequest.to.id = input.toConnect;
        
        // Handle translations
        if (input.translationsCreate && input.translationsCreate.length > 0) {
            pullRequest.translations = input.translationsCreate.map(trans => ({
                __typename: "PullRequestTranslation" as const,
                id: trans.id,
                language: trans.language,
                text: trans.text,
            }));
            pullRequest.translationsCount = input.translationsCreate.length;
        }
        
        return pullRequest;
    }

    /**
     * Update entity from update input
     */
    updateFromInput(existing: PullRequest, input: PullRequestUpdateInput): PullRequest {
        const updated = { ...existing };
        updated.updatedAt = new Date().toISOString();
        
        if (input.status !== undefined) {
            updated.status = input.status;
            if ([PullRequestStatusEnum.Merged, PullRequestStatusEnum.Rejected, PullRequestStatusEnum.Canceled]
                .includes(input.status)) {
                updated.closedAt = new Date().toISOString();
            }
        }
        
        // Handle translation updates
        if (input.translationsUpdate) {
            // Apply translation updates (simplified for fixtures)
            updated.translations = input.translationsUpdate.map(trans => ({
                __typename: "PullRequestTranslation" as const,
                id: trans.id,
                language: trans.language,
                text: trans.text,
            }));
        }
        
        return updated;
    }

    /**
     * Validate create input
     */
    async validateCreateInput(input: PullRequestCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};
        
        if (!input.fromConnect) {
            errors.fromConnect = "Source resource version ID is required";
        }
        if (!input.toConnect) {
            errors.toConnect = "Target resource ID is required";
        }
        if (!input.translationsCreate || input.translationsCreate.length === 0) {
            errors.translations = "At least one translation is required";
        } else {
            const englishTranslation = input.translationsCreate.find(t => t.language === "en");
            if (!englishTranslation || !englishTranslation.text?.trim()) {
                errors.translations = "English translation text is required";
            }
        }
        
        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: PullRequestUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};
        
        if (!input.id) {
            errors.id = "Pull request ID is required for updates";
        }
        
        if (input.status && !Object.values(PullRequestStatusEnum).includes(input.status)) {
            errors.status = "Invalid pull request status";
        }
        
        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create pull requests for all status types
     */
    createAllStatusPullRequests(): PullRequest[] {
        return Object.values(PullRequestStatusEnum).map(status => 
            this.createMockData({
                overrides: {
                    status,
                    publicId: `${String(status).toUpperCase()}-${this.generatePublicId()}`,
                },
            }),
        );
    }

    /**
     * Create workflow-specific pull requests
     */
    createWorkflowPullRequest(workflow: "draft" | "review" | "approved" | "merged" | "rejected"): PullRequest {
        const baseOptions: MockDataOptions = { withRelations: true };
        
        switch (workflow) {
            case "draft":
                return this.createMockData({
                    ...baseOptions,
                    overrides: {
                        status: PullRequestStatusEnum.Draft,
                        commentsCount: DEFAULT_COMMENT_COUNT,
                    },
                    scenario: "edge-case",
                });
            
            case "review":
                return this.createMockData({
                    ...baseOptions,
                    overrides: {
                        status: PullRequestStatusEnum.Open,
                        commentsCount: REVIEW_COMMENT_COUNT,
                    },
                });
            
            case "approved":
                return this.createMockData({
                    ...baseOptions,
                    overrides: {
                        status: PullRequestStatusEnum.Open,
                        commentsCount: APPROVED_COMMENT_COUNT,
                    },
                    scenario: "complete",
                });
            
            case "merged":
                return this.createMockData({
                    ...baseOptions,
                    overrides: {
                        status: PullRequestStatusEnum.Merged,
                        commentsCount: MERGED_COMMENT_COUNT,
                        closedAt: new Date().toISOString(),
                    },
                });
            
            case "rejected":
                return this.createMockData({
                    ...baseOptions,
                    overrides: {
                        status: PullRequestStatusEnum.Rejected,
                        commentsCount: REVIEW_COMMENT_COUNT,
                        closedAt: new Date().toISOString(),
                    },
                });
            
            default:
                return this.createMockData(baseOptions);
        }
    }

    /**
     * Create business error for merge conflicts
     */
    createMergeConflictError(pullRequestId?: string): any {
        return this.createBusinessErrorResponse("conflict", {
            reason: "Merge conflict detected",
            conflictResolutionRequired: true,
            conflictFiles: Array.from(MERGE_CONFLICT_FILES),
            pullRequestId: pullRequestId || this.generateId(),
        });
    }

    /**
     * Create business error for invalid state transitions
     */
    createInvalidStateTransitionError(fromStatus: PullRequestStatus, toStatus: PullRequestStatus): any {
        return this.createBusinessErrorResponse("state", {
            reason: `Cannot transition from ${fromStatus} to ${toStatus}`,
            currentState: fromStatus,
            requestedState: toStatus,
            validTransitions: this.getValidTransitions(fromStatus),
        });
    }

    /**
     * Get valid status transitions for a given status
     */
    private getValidTransitions(status: PullRequestStatus): PullRequestStatus[] {
        switch (status) {
            case PullRequestStatusEnum.Draft:
                return [PullRequestStatusEnum.Open, PullRequestStatusEnum.Canceled];
            case PullRequestStatusEnum.Open:
                return [PullRequestStatusEnum.Merged, PullRequestStatusEnum.Rejected, PullRequestStatusEnum.Draft];
            case PullRequestStatusEnum.Merged:
            case PullRequestStatusEnum.Rejected:
            case PullRequestStatusEnum.Canceled:
                return []; // Terminal states
            default:
                return [];
        }
    }
}

/**
 * Pre-configured response scenarios for common use cases
 */
export const pullRequestResponseScenarios = {
    // Success scenarios
    createSuccess: (pullRequest?: PullRequest) => {
        const factory = new PullRequestAPIResponseFactory();
        return factory.createSuccessResponse(
            pullRequest || factory.createMockData(),
        );
    },

    listSuccess: (pullRequests?: PullRequest[], pagination?: { page: number; pageSize: number; totalCount: number }) => {
        const factory = new PullRequestAPIResponseFactory();
        return factory.createPaginatedResponse(
            pullRequests || factory.createAllStatusPullRequests(),
            pagination || { page: 1, pageSize: 20, totalCount: 50 },
        );
    },

    draftPullRequest: () => {
        const factory = new PullRequestAPIResponseFactory();
        return factory.createSuccessResponse(
            factory.createWorkflowPullRequest("draft"),
        );
    },

    reviewPullRequest: () => {
        const factory = new PullRequestAPIResponseFactory();
        return factory.createSuccessResponse(
            factory.createWorkflowPullRequest("review"),
        );
    },

    mergedPullRequest: () => {
        const factory = new PullRequestAPIResponseFactory();
        return factory.createSuccessResponse(
            factory.createWorkflowPullRequest("merged"),
        );
    },

    rejectedPullRequest: () => {
        const factory = new PullRequestAPIResponseFactory();
        return factory.createSuccessResponse(
            factory.createWorkflowPullRequest("rejected"),
        );
    },

    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new PullRequestAPIResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                fromConnect: "Source resource version is required",
                toConnect: "Target resource is required",
                translations: "At least one translation is required",
            },
        );
    },

    notFoundError: (pullRequestId?: string) => {
        const factory = new PullRequestAPIResponseFactory();
        return factory.createNotFoundErrorResponse(
            pullRequestId || "non-existent-pr-id",
            "pull request",
        );
    },

    permissionError: (operation?: string) => {
        const factory = new PullRequestAPIResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
            ["pull-request:write"],
        );
    },

    mergeConflictError: (pullRequestId?: string) => {
        const factory = new PullRequestAPIResponseFactory();
        return factory.createMergeConflictError(pullRequestId);
    },

    invalidStateTransitionError: (fromStatus?: PullRequestStatus, toStatus?: PullRequestStatus) => {
        const factory = new PullRequestAPIResponseFactory();
        return factory.createInvalidStateTransitionError(
            fromStatus || PullRequestStatusEnum.Merged,
            toStatus || PullRequestStatusEnum.Open,
        );
    },

    serverError: () => {
        const factory = new PullRequestAPIResponseFactory();
        return factory.createServerErrorResponse("pull-request-service", "merge");
    },

    rateLimitError: () => {
        const factory = new PullRequestAPIResponseFactory();
        const resetTime = new Date(Date.now() + 5 * MINUTES_IN_MS); // 5 minutes from now
        return factory.createRateLimitErrorResponse(100, 0, resetTime);
    },
};

// Export factory instance for direct use
export const pullRequestAPIResponseFactory = new PullRequestAPIResponseFactory();

