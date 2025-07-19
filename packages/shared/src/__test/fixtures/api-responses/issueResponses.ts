/* c8 ignore start */
/**
 * Issue API Response Fixtures
 * 
 * Comprehensive fixtures for issue reporting and tracking endpoints including
 * bug reports, feature requests, and support tickets across all resource types.
 */

import type {
    Issue,
    IssueCreateInput,
    IssueFor,
    IssueStatus,
    IssueUpdateInput,
    Resource,
    Team,
} from "../../../api/types.js";
import { generatePK } from "../../../id/index.js";
import { BaseAPIResponseFactory } from "./base.js";
import { resourceResponseFactory } from "./resourceResponses.js";
import { teamResponseFactory } from "./teamResponses.js";
import type { MockDataOptions } from "./types.js";
import { userResponseFactory } from "./userResponses.js";

// Constants
const DEFAULT_COUNT = 10;
const DEFAULT_ERROR_RATE = 0.1;
const DEFAULT_DELAY_MS = 500;
const MAX_TITLE_LENGTH = 200;
const MAX_DESCRIPTION_LENGTH = 5000;

// Issue status constants
const ISSUE_STATUSES: IssueStatus[] = ["Draft", "Open", "ClosedResolved", "ClosedUnresolved", "Rejected"];

/**
 * Issue API response factory
 */
export class IssueResponseFactory extends BaseAPIResponseFactory<
    Issue,
    IssueCreateInput,
    IssueUpdateInput
> {
    protected readonly entityName = "issue";

    /**
     * Create mock issue data
     */
    createMockData(options?: MockDataOptions): Issue {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();
        const issueId = options?.overrides?.id || generatePK().toString();

        const baseIssue: Issue = {
            __typename: "Issue",
            id: issueId,
            createdAt: now,
            updatedAt: now,
            status: "Open",
            score: 0,
            bookmarks: 0,
            views: 0,
            commentsCount: 0,
            reportsCount: 0,
            translationsCount: 1,
            closedAt: null,
            referencedVersionId: null,
            createdBy: userResponseFactory.createMockData(),
            closedBy: null,
            to: resourceResponseFactory.createMockData(),
            translations: [{
                __typename: "IssueTranslation",
                id: generatePK().toString(),
                language: "en",
                name: "Sample Issue",
                description: "This is a sample issue description for testing purposes.",
            }],
            comments: [],
            reports: [],
            bookmarkedBy: [],
            you: {
                canBookmark: true,
                canComment: true,
                canDelete: false,
                canReact: true,
                canRead: true,
                canReport: true,
                canUpdate: false,
                isBookmarked: false,
                reaction: null,
            },
        };

        if (scenario === "complete" || scenario === "edge-case") {
            const closedBy = userResponseFactory.createMockData({ scenario: "complete" });
            return {
                ...baseIssue,
                status: "ClosedResolved",
                score: 85,
                bookmarks: 12,
                views: 156,
                commentsCount: 8,
                reportsCount: 1,
                closedAt: new Date(Date.now().toISOString() - (24 * 60 * 60 * 1000)).toISOString(), // 1 day ago
                closedBy,
                createdBy: userResponseFactory.createMockData({ scenario: "complete" }),
                to: scenario === "edge-case"
                    ? teamResponseFactory.createMockData({ scenario: "complete" })
                    : resourceResponseFactory.createMockData({ scenario: "complete" }),
                translations: [{
                    __typename: "IssueTranslation",
                    id: generatePK().toString(),
                    language: "en",
                    name: "Critical Performance Bug in Authentication Module",
                    description: "Detailed description of a critical issue that affects user authentication flows, causing intermittent login failures during peak traffic periods. This issue has been reproduced consistently in our staging environment and requires immediate attention.",
                }],
                you: {
                    canBookmark: true,
                    canComment: true,
                    canDelete: true,
                    canReact: true,
                    canRead: true,
                    canReport: false,
                    canUpdate: true,
                    isBookmarked: true,
                    reaction: "thumbsUp",
                },
                ...options?.overrides,
            };
        }

        return {
            ...baseIssue,
            ...options?.overrides,
        };
    }

    /**
     * Create issue from input
     */
    createFromInput(input: IssueCreateInput): Issue {
        const now = new Date().toISOString();
        const issueId = generatePK().toString();

        // Create the target object based on issueFor
        let target: Resource | Team;
        if (input.issueFor === "Team") {
            target = teamResponseFactory.createMockData({ overrides: { id: input.forConnect } });
        } else {
            target = resourceResponseFactory.createMockData({ overrides: { id: input.forConnect } });
        }

        return {
            __typename: "Issue",
            id: issueId,
            createdAt: now,
            updatedAt: now,
            status: "Draft", // New issues start as draft
            score: 0,
            bookmarks: 0,
            views: 0,
            commentsCount: 0,
            reportsCount: 0,
            translationsCount: input.translationsCreate?.length || 1,
            closedAt: null,
            referencedVersionId: input.referencedVersionIdConnect || null,
            createdBy: userResponseFactory.createMockData(),
            closedBy: null,
            to: target,
            translations: input.translationsCreate?.map(t => ({
                __typename: "IssueTranslation" as const,
                id: generatePK().toString(),
                language: t.language,
                name: t.name,
                description: t.description || null,
            })) || [{
                __typename: "IssueTranslation" as const,
                id: generatePK().toString(),
                language: "en",
                name: "New Issue",
                description: "Issue description",
            }],
            comments: [],
            reports: [],
            bookmarkedBy: [],
            you: {
                canBookmark: true,
                canComment: true,
                canDelete: true,
                canReact: true,
                canRead: true,
                canReport: false,
                canUpdate: true,
                isBookmarked: false,
                reaction: null,
            },
        };
    }

    /**
     * Update issue from input
     */
    updateFromInput(existing: Issue, input: IssueUpdateInput): Issue {
        const updates: Partial<Issue> = {
            updatedAt: new Date().toISOString(),
        };

        if (input.status !== undefined) {
            updates.status = input.status;

            // Handle status transitions
            if (input.status === "ClosedResolved" || input.status === "ClosedUnresolved") {
                updates.closedAt = new Date().toISOString();
                updates.closedBy = userResponseFactory.createMockData();
            } else if (existing.status === "ClosedResolved" || existing.status === "ClosedUnresolved") {
                // Reopening issue
                updates.closedAt = null;
                updates.closedBy = null;
            }
        }

        // Handle translation updates
        if (input.translationsUpdate) {
            updates.translations = existing.translations?.map(translation => {
                const update = input.translationsUpdate?.find(u => u.id === translation.id);
                return update ? { ...translation, ...update } : translation;
            });
        }

        return {
            ...existing,
            ...updates,
        };
    }

    /**
     * Validate create input
     */
    async validateCreateInput(input: IssueCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.forConnect) {
            errors.forConnect = "Target object ID is required";
        }

        if (!input.issueFor) {
            errors.issueFor = "Issue type must be specified";
        }

        if (!input.translationsCreate || input.translationsCreate.length === 0) {
            errors.translationsCreate = "At least one translation is required";
        } else {
            input.translationsCreate.forEach((translation, index) => {
                if (!translation.name || translation.name.trim().length === 0) {
                    errors[`translationsCreate.${index}.name`] = "Issue name is required";
                } else if (translation.name.length > MAX_TITLE_LENGTH) {
                    errors[`translationsCreate.${index}.name`] = `Issue name must be ${MAX_TITLE_LENGTH} characters or less`;
                }

                if (translation.description && translation.description.length > MAX_DESCRIPTION_LENGTH) {
                    errors[`translationsCreate.${index}.description`] = `Description must be ${MAX_DESCRIPTION_LENGTH} characters or less`;
                }

                if (!translation.language) {
                    errors[`translationsCreate.${index}.language`] = "Language is required";
                }
            });
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: IssueUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (input.status !== undefined && !ISSUE_STATUSES.includes(input.status)) {
            errors.status = "Invalid issue status";
        }

        // Validate translation updates
        if (input.translationsUpdate) {
            input.translationsUpdate.forEach((translation, index) => {
                if (translation.name !== undefined) {
                    if (!translation.name || translation.name.trim().length === 0) {
                        errors[`translationsUpdate.${index}.name`] = "Issue name cannot be empty";
                    } else if (translation.name.length > MAX_TITLE_LENGTH) {
                        errors[`translationsUpdate.${index}.name`] = `Issue name must be ${MAX_TITLE_LENGTH} characters or less`;
                    }
                }

                if (translation.description !== undefined && translation.description && translation.description.length > MAX_DESCRIPTION_LENGTH) {
                    errors[`translationsUpdate.${index}.description`] = `Description must be ${MAX_DESCRIPTION_LENGTH} characters or less`;
                }
            });
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create issues with all possible statuses
     */
    createIssuesWithAllStatuses(): Issue[] {
        return ISSUE_STATUSES.map((status, index) => {
            const issue = this.createMockData({
                overrides: {
                    id: `issue_${status.toLowerCase()}_${index}`,
                    status,
                },
            });

            // Set closed fields for closed statuses
            if (status === "ClosedResolved" || status === "ClosedUnresolved") {
                issue.closedAt = new Date(Date.now().toISOString() - (index * 24 * 60 * 60 * 1000)).toISOString();
                issue.closedBy = userResponseFactory.createMockData();
            }

            // Update translation based on status
            issue.translations[0].name = `${status} Issue Example`;
            issue.translations[0].description = `This is an example of an issue in ${status} status.`;

            return issue;
        });
    }

    /**
     * Create issues for different target types
     */
    createIssuesForAllTypes(): Issue[] {
        const types: IssueFor[] = ["Team", "Resource"];

        return types.map((issueFor, index) => {
            let target: Resource | Team;

            if (issueFor === "Team") {
                target = teamResponseFactory.createMockData({ scenario: "complete" });
            } else {
                target = resourceResponseFactory.createMockData({ scenario: "complete" });
            }

            return this.createMockData({
                overrides: {
                    id: `issue_${issueFor.toLowerCase()}_${index}`,
                    to: target,
                    translations: [{
                        __typename: "IssueTranslation" as const,
                        id: generatePK().toString(),
                        language: "en",
                        name: `Issue for ${issueFor}`,
                        description: `This issue is reported against a ${issueFor}.`,
                    }],
                },
            });
        });
    }

    /**
     * Create issues with different severity levels
     */
    createIssuesWithDifferentSeverity(): Issue[] {
        const severityLevels = [
            { name: "Critical", score: 100, description: "Critical issue requiring immediate attention" },
            { name: "High", score: 75, description: "High priority issue" },
            { name: "Medium", score: 50, description: "Medium priority issue" },
            { name: "Low", score: 25, description: "Low priority issue" },
            { name: "Enhancement", score: 10, description: "Enhancement request" },
        ];

        return severityLevels.map((severity, index) =>
            this.createMockData({
                overrides: {
                    id: `issue_${severity.name.toLowerCase()}_${index}`,
                    score: severity.score,
                    translations: [{
                        __typename: "IssueTranslation" as const,
                        id: generatePK().toString(),
                        language: "en",
                        name: `${severity.name} Priority Issue`,
                        description: severity.description,
                    }],
                },
            }),
        );
    }

    /**
     * Create status conflict error response
     */
    createStatusConflictErrorResponse(currentStatus: IssueStatus, attemptedStatus: IssueStatus) {
        return this.createBusinessErrorResponse("status_conflict", {
            currentStatus,
            attemptedStatus,
            allowedTransitions: this.getAllowedStatusTransitions(currentStatus),
            message: `Cannot change issue status from ${currentStatus} to ${attemptedStatus}`,
        });
    }

    /**
     * Create issue already exists error response
     */
    createDuplicateIssueErrorResponse(existingIssueId: string) {
        return this.createBusinessErrorResponse("duplicate", {
            resource: "issue",
            existingIssueId,
            message: "Similar issue already exists for this target",
        });
    }

    /**
     * Create issue locked error response
     */
    createIssueLockedErrorResponse(lockReason: string) {
        return this.createBusinessErrorResponse("locked", {
            resource: "issue",
            lockReason,
            message: "Issue is locked and cannot be modified",
        });
    }

    /**
     * Get allowed status transitions for a given status
     */
    private getAllowedStatusTransitions(status: IssueStatus): IssueStatus[] {
        switch (status) {
            case "Draft":
                return ["Open", "Rejected"];
            case "Open":
                return ["ClosedResolved", "ClosedUnresolved", "Rejected"];
            case "ClosedResolved":
                return ["Open"];
            case "ClosedUnresolved":
                return ["Open"];
            case "Rejected":
                return ["Draft", "Open"];
            default:
                return [];
        }
    }
}

/**
 * Pre-configured issue response scenarios
 */
export const issueResponseScenarios = {
    // Success scenarios
    createSuccess: (input?: Partial<IssueCreateInput>) => {
        const factory = new IssueResponseFactory();
        const defaultInput: IssueCreateInput = {
            forConnect: generatePK().toString(),
            issueFor: "Resource",
            translationsCreate: [{
                id: generatePK().toString(),
                language: "en",
                name: "New Issue",
                description: "Issue description",
            }],
            ...input,
        };
        return factory.createSuccessResponse(
            factory.createFromInput(defaultInput),
        );
    },

    findSuccess: (issue?: Issue) => {
        const factory = new IssueResponseFactory();
        return factory.createSuccessResponse(
            issue || factory.createMockData(),
        );
    },

    findCompleteSuccess: () => {
        const factory = new IssueResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({ scenario: "complete" }),
        );
    },

    updateSuccess: (existing?: Issue, updates?: Partial<IssueUpdateInput>) => {
        const factory = new IssueResponseFactory();
        const issue = existing || factory.createMockData({ scenario: "complete" });
        const input: IssueUpdateInput = {
            id: issue.id,
            ...updates,
        };
        return factory.createSuccessResponse(
            factory.updateFromInput(issue, input),
        );
    },

    closeSuccess: (issueId?: string, status: IssueStatus = "ClosedResolved") => {
        const factory = new IssueResponseFactory();
        const existing = factory.createMockData({ overrides: { id: issueId } });
        return factory.createSuccessResponse(
            factory.updateFromInput(existing, { id: existing.id, status }),
        );
    },

    listSuccess: (issues?: Issue[]) => {
        const factory = new IssueResponseFactory();
        return factory.createPaginatedResponse(
            issues || Array.from({ length: DEFAULT_COUNT }, () => factory.createMockData()),
            { page: 1, totalCount: issues?.length || DEFAULT_COUNT },
        );
    },

    listByStatusSuccess: (status: IssueStatus) => {
        const factory = new IssueResponseFactory();
        const issues = factory.createIssuesWithAllStatuses().filter(i => i.status === status);
        return factory.createPaginatedResponse(
            issues,
            { page: 1, totalCount: issues.length },
        );
    },

    listByTypeSuccess: (issueFor: IssueFor) => {
        const factory = new IssueResponseFactory();
        const issues = factory.createIssuesForAllTypes().filter(i =>
            i.to.__typename === issueFor,
        );
        return factory.createPaginatedResponse(
            issues,
            { page: 1, totalCount: issues.length },
        );
    },

    listBySeveritySuccess: () => {
        const factory = new IssueResponseFactory();
        const issues = factory.createIssuesWithDifferentSeverity();
        return factory.createPaginatedResponse(
            issues,
            { page: 1, totalCount: issues.length },
        );
    },

    criticalIssuesSuccess: () => {
        const factory = new IssueResponseFactory();
        const criticalIssues = factory.createIssuesWithDifferentSeverity()
            .filter(i => i.score >= 75);
        return factory.createPaginatedResponse(
            criticalIssues,
            { page: 1, totalCount: criticalIssues.length },
        );
    },

    openIssuesSuccess: () => {
        const factory = new IssueResponseFactory();
        const openIssues = factory.createIssuesWithAllStatuses()
            .filter(i => i.status === "Open");
        return factory.createPaginatedResponse(
            openIssues,
            { page: 1, totalCount: openIssues.length },
        );
    },

    // Error scenarios
    createValidationError: () => {
        const factory = new IssueResponseFactory();
        return factory.createValidationErrorResponse({
            forConnect: "Target object ID is required",
            issueFor: "Issue type must be specified",
            "translationsCreate.0.name": "Issue name is required",
        });
    },

    notFoundError: (issueId?: string) => {
        const factory = new IssueResponseFactory();
        return factory.createNotFoundErrorResponse(
            issueId || "non-existent-issue",
        );
    },

    permissionError: (operation?: string) => {
        const factory = new IssueResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
            ["issue:write"],
        );
    },

    statusConflictError: (currentStatus: IssueStatus = "ClosedResolved", attemptedStatus: IssueStatus = "Draft") => {
        const factory = new IssueResponseFactory();
        return factory.createStatusConflictErrorResponse(currentStatus, attemptedStatus);
    },

    duplicateIssueError: (existingIssueId?: string) => {
        const factory = new IssueResponseFactory();
        return factory.createDuplicateIssueErrorResponse(existingIssueId || generatePK().toString());
    },

    issueLockedError: (lockReason = "Issue is being reviewed by moderators") => {
        const factory = new IssueResponseFactory();
        return factory.createIssueLockedErrorResponse(lockReason);
    },

    titleTooLongError: () => {
        const factory = new IssueResponseFactory();
        return factory.createValidationErrorResponse({
            "translationsCreate.0.name": `Issue name must be ${MAX_TITLE_LENGTH} characters or less`,
        });
    },

    invalidTargetError: () => {
        const factory = new IssueResponseFactory();
        return factory.createValidationErrorResponse({
            forConnect: "Target object does not exist or is not accessible",
        });
    },

    // MSW handlers
    handlers: {
        success: () => new IssueResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate?: number) {
            return new IssueResponseFactory().createMSWHandlers({ errorRate: errorRate ?? DEFAULT_ERROR_RATE });
        },
        withDelay: function createWithDelay(delay?: number) {
            return new IssueResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
    },
};

// Export factory instance for direct use
export const issueResponseFactory = new IssueResponseFactory();
