/**
 * Business logic error fixtures
 * 
 * These fixtures provide business rule violation scenarios including
 * resource limits, workflow errors, data conflicts, and domain-specific errors.
 * 
 * Enhanced with unified error fixture factory pattern:
 * - userImpact: 'blocking' for hard limits, 'degraded' for soft limits, 'none' for informational
 * - recovery: Strategies include 'fail', 'fallback', 'retry', and 'circuit-break'
 * - BusinessErrorFactory: Extends BaseErrorFactory for consistent error creation
 */

import type {
    EnhancedBusinessError,
    ErrorContext
} from "./types.js";
import { BaseErrorFactory } from "./types.js";

/**
 * Factory for creating business error fixtures with enhanced properties
 */
export class BusinessErrorFactory extends BaseErrorFactory<EnhancedBusinessError> {
    standard: EnhancedBusinessError = {
        code: "BUSINESS_RULE_VIOLATION",
        message: "A business rule has been violated",
        type: "constraint",
        userImpact: "blocking",
        recovery: { strategy: "fail" },
    };

    withDetails: EnhancedBusinessError = {
        code: "RESOURCE_LIMIT_EXCEEDED",
        message: "You have exceeded the allowed resource limit",
        type: "limit",
        details: {
            current: 100,
            limit: 50,
            suggestion: "Upgrade your plan to increase limits",
        },
        userAction: {
            label: "Upgrade Plan",
            action: "upgrade_plan",
            url: "/billing/upgrade",
        },
        userImpact: "blocking",
        recovery: {
            strategy: "fallback",
            fallbackAction: "upgrade_plan",
        },
    };

    variants = {
        softLimit: {
            code: "SOFT_LIMIT_WARNING",
            message: "You are approaching your resource limit",
            type: "limit",
            details: {
                current: 80,
                limit: 100,
                percentage: 80,
            },
            userImpact: "degraded",
            recovery: { strategy: "retry", attempts: 3 },
        } satisfies EnhancedBusinessError,

        hardConstraint: {
            code: "HARD_CONSTRAINT_VIOLATION",
            message: "This operation violates a critical business constraint",
            type: "constraint",
            userImpact: "blocking",
            recovery: { strategy: "fail" },
        } satisfies EnhancedBusinessError,
    };

    create(overrides?: Partial<EnhancedBusinessError>): EnhancedBusinessError {
        return {
            ...this.standard,
            ...overrides,
        };
    }

    createWithContext(context: ErrorContext): EnhancedBusinessError {
        return {
            ...this.standard,
            context,
            telemetry: {
                traceId: `trace-${Date.now()}`,
                spanId: `span-${Date.now()}`,
                tags: {
                    operation: context.operation || "unknown",
                    environment: context.environment || "development",
                },
                level: "error",
            },
        };
    }
}

export const businessErrorFixtures = {
    // Resource limit errors
    limits: {
        creditLimit: {
            code: "INSUFFICIENT_CREDITS",
            message: "You don't have enough credits for this operation",
            type: "limit",
            details: {
                required: 100,
                current: 25,
                operation: "run_routine",
            },
            userAction: {
                label: "Purchase Credits",
                action: "purchase_credits",
                url: "/billing/credits",
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fallback",
                fallbackAction: "upgrade_plan",
            },
        } satisfies EnhancedBusinessError,

        projectLimit: {
            code: "PROJECT_LIMIT_REACHED",
            message: "You have reached the maximum number of projects for your plan",
            type: "limit",
            details: {
                current: 10,
                limit: 10,
                plan: "free",
            },
            userAction: {
                label: "Upgrade Plan",
                action: "upgrade_plan",
                url: "/billing/upgrade",
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fallback",
                fallbackAction: "upgrade_plan",
            },
        } satisfies EnhancedBusinessError,

        teamMemberLimit: {
            code: "TEAM_MEMBER_LIMIT",
            message: "This team has reached its member limit",
            type: "limit",
            details: {
                current: 50,
                limit: 50,
                teamId: "team_123",
            },
            userAction: {
                label: "Increase Team Size",
                action: "upgrade_team",
                url: "/team/billing",
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fallback",
                fallbackAction: "upgrade_plan",
            },
        } satisfies EnhancedBusinessError,

        storageLimit: {
            code: "STORAGE_LIMIT_EXCEEDED",
            message: "You have exceeded your storage limit",
            type: "limit",
            details: {
                current: "5.2 GB",
                limit: "5 GB",
                overage: "200 MB",
            },
            userAction: {
                label: "Manage Storage",
                action: "manage_storage",
                url: "/settings/storage",
            },
            userImpact: "degraded",
            recovery: {
                strategy: "fallback",
                fallbackAction: "upgrade_plan",
            },
        } satisfies EnhancedBusinessError,
    },

    // Workflow errors
    workflow: {
        invalidStateTransition: {
            code: "INVALID_STATE_TRANSITION",
            message: "Cannot transition from current state to requested state",
            type: "workflow",
            details: {
                currentState: "draft",
                requestedState: "published",
                allowedStates: ["review", "archived"],
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
        } satisfies EnhancedBusinessError,

        prerequisiteNotMet: {
            code: "PREREQUISITE_NOT_MET",
            message: "Required steps must be completed first",
            type: "workflow",
            details: {
                missingSteps: ["email_verification", "profile_completion"],
                currentStep: "create_project",
            },
            userAction: {
                label: "Complete Setup",
                action: "complete_prerequisites",
                url: "/onboarding",
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
        } satisfies EnhancedBusinessError,

        approvalRequired: {
            code: "APPROVAL_REQUIRED",
            message: "This action requires approval from an administrator",
            type: "workflow",
            details: {
                action: "publish_routine",
                approver: "team_admin",
                requestId: "req_456",
            },
            userAction: {
                label: "Request Approval",
                action: "request_approval",
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
        } satisfies EnhancedBusinessError,

        dependencyNotReady: {
            code: "DEPENDENCY_NOT_READY",
            message: "Required dependencies are not available",
            type: "workflow",
            details: {
                dependencies: ["data_source_connection", "api_key_validation"],
                readyDependencies: ["data_source_connection"],
            },
            userImpact: "blocking",
            recovery: {
                strategy: "retry",
                attempts: 2,
                delay: 1000,
            },
        } satisfies EnhancedBusinessError,
    },

    // Data conflict errors
    conflicts: {
        duplicateName: {
            code: "DUPLICATE_NAME",
            message: "A resource with this name already exists",
            type: "conflict",
            details: {
                field: "name",
                value: "My Project",
                existingId: "proj_789",
                suggestion: "My Project (2)",
            },
            userImpact: "none",
            recovery: {
                strategy: "fail",
            },
        } satisfies EnhancedBusinessError,

        versionConflict: {
            code: "VERSION_CONFLICT",
            message: "The resource has been modified by another user",
            type: "conflict",
            details: {
                yourVersion: "v1.2.3",
                currentVersion: "v1.2.4",
                modifiedBy: "user_456",
                modifiedAt: new Date(Date.now() - 300000).toISOString(),
            },
            userAction: {
                label: "Review Changes",
                action: "review_conflicts",
            },
            userImpact: "degraded",
            recovery: {
                strategy: "retry",
                attempts: 2,
            },
        } satisfies EnhancedBusinessError,

        scheduleConflict: {
            code: "SCHEDULE_CONFLICT",
            message: "This time slot conflicts with an existing schedule",
            type: "conflict",
            details: {
                requestedTime: "2024-03-15T14:00:00Z",
                conflictWith: "Team Meeting",
                conflictId: "meeting_123",
                suggestion: "2024-03-15T15:00:00Z",
            },
            userImpact: "none",
            recovery: {
                strategy: "fail",
            },
        } satisfies EnhancedBusinessError,

        resourceLocked: {
            code: "RESOURCE_LOCKED",
            message: "This resource is currently being edited by another user",
            type: "conflict",
            details: {
                lockedBy: "user_789",
                lockedAt: new Date(Date.now() - 120000).toISOString(),
                expectedRelease: new Date(Date.now() + 180000).toISOString(),
            },
            userImpact: "degraded",
            recovery: {
                strategy: "retry",
                attempts: 3,
                delay: 60000,
            },
        } satisfies EnhancedBusinessError,
    },

    // State errors
    state: {
        alreadyCompleted: {
            code: "ALREADY_COMPLETED",
            message: "This action has already been completed",
            type: "state",
            details: {
                completedAt: new Date(Date.now() - 3600000).toISOString(),
                completedBy: "user_123",
            },
            userImpact: "none",
            recovery: {
                strategy: "fail",
            },
        } satisfies EnhancedBusinessError,

        notPublished: {
            code: "NOT_PUBLISHED",
            message: "This resource must be published before it can be used",
            type: "state",
            details: {
                resourceType: "routine",
                resourceId: "routine_456",
                currentStatus: "draft",
            },
            userAction: {
                label: "Publish Now",
                action: "publish_resource",
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
        } satisfies EnhancedBusinessError,

        archived: {
            code: "RESOURCE_ARCHIVED",
            message: "This resource has been archived and cannot be modified",
            type: "state",
            details: {
                archivedAt: new Date(Date.now() - 86400000).toISOString(),
                archivedBy: "admin_123",
            },
            userAction: {
                label: "Restore Resource",
                action: "restore_resource",
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
        } satisfies EnhancedBusinessError,

        maintenanceMode: {
            code: "MAINTENANCE_MODE",
            message: "This feature is currently under maintenance",
            type: "state",
            details: {
                feature: "ai_execution",
                estimatedEndTime: new Date(Date.now() + 3600000).toISOString(),
                alternativeAction: "manual_execution",
            },
            userImpact: "degraded",
            recovery: {
                strategy: "fallback",
                fallbackAction: "manual_execution",
            },
        } satisfies EnhancedBusinessError,
    },

    // Constraint violations
    constraints: {
        circularDependency: {
            code: "CIRCULAR_DEPENDENCY",
            message: "This would create a circular dependency",
            type: "constraint",
            details: {
                path: ["A", "B", "C", "A"],
                newConnection: { from: "C", to: "A" },
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
        } satisfies EnhancedBusinessError,

        invalidHierarchy: {
            code: "INVALID_HIERARCHY",
            message: "Invalid parent-child relationship",
            type: "constraint",
            details: {
                parent: "team_123",
                child: "team_456",
                reason: "Child team is already a parent of the proposed parent",
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
        } satisfies EnhancedBusinessError,

        dataIntegrity: {
            code: "DATA_INTEGRITY_VIOLATION",
            message: "This operation would violate data integrity constraints",
            type: "constraint",
            details: {
                constraint: "foreign_key",
                table: "projects",
                referencedTable: "teams",
                missingId: "team_999",
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
        } satisfies EnhancedBusinessError,

        businessRule: {
            code: "BUSINESS_RULE_VIOLATION",
            message: "This violates a business rule",
            type: "constraint",
            details: {
                rule: "minimum_team_size",
                description: "Teams must have at least 2 members",
                current: 1,
                required: 2,
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
        } satisfies EnhancedBusinessError,
    },

    // Policy errors
    policy: {
        ageRestriction: {
            code: "AGE_RESTRICTION",
            message: "You must be 18 or older to access this feature",
            type: "policy",
            details: {
                requiredAge: 18,
                feature: "payment_processing",
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
        } satisfies EnhancedBusinessError,

        regionRestriction: {
            code: "REGION_RESTRICTED",
            message: "This feature is not available in your region",
            type: "policy",
            details: {
                userRegion: "EU",
                allowedRegions: ["US", "CA"],
                reason: "regulatory_compliance",
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
        } satisfies EnhancedBusinessError,

        contentPolicy: {
            code: "CONTENT_POLICY_VIOLATION",
            message: "This content violates our community guidelines",
            type: "policy",
            details: {
                policy: "prohibited_content",
                category: "spam",
                action: "content_removed",
            },
            userAction: {
                label: "Review Guidelines",
                action: "view_guidelines",
                url: "/policies/community",
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
        } satisfies EnhancedBusinessError,

        usagePolicy: {
            code: "USAGE_POLICY_VIOLATION",
            message: "This usage pattern violates our fair use policy",
            type: "policy",
            details: {
                violation: "excessive_api_calls",
                threshold: 10000,
                period: "hour",
                current: 15000,
            },
            userAction: {
                label: "View Usage Policy",
                action: "view_policy",
                url: "/policies/fair-use",
            },
            userImpact: "degraded",
            recovery: {
                strategy: "circuit-break",
                delay: 300000, // 5 minutes
            },
        } satisfies EnhancedBusinessError,
    },

    // Factory functions
    factories: {
        /**
         * Create a limit exceeded error
         */
        createLimitError: (
            resource: string,
            current: number,
            limit: number,
            upgradeUrl?: string,
        ): EnhancedBusinessError => ({
            code: `${resource.toUpperCase()}_LIMIT_EXCEEDED`,
            message: `You have reached the ${resource} limit`,
            type: "limit",
            details: { current, limit },
            userImpact: "blocking",
            recovery: {
                strategy: "fallback",
                fallbackAction: "upgrade_plan",
            },
            ...(upgradeUrl && {
                userAction: {
                    label: "Upgrade",
                    action: "upgrade",
                    url: upgradeUrl,
                },
            }),
        }),

        /**
         * Create a conflict error
         */
        createConflictError: (field: string, value: string, suggestion?: string): EnhancedBusinessError => ({
            code: "CONFLICT",
            message: `A conflict was detected for ${field}`,
            type: "conflict",
            details: {
                field,
                value,
                ...(suggestion && { suggestion }),
            },
            userImpact: "none",
            recovery: {
                strategy: "fail",
            },
        }),

        /**
         * Create a workflow error
         */
        createWorkflowError: (
            currentStep: string,
            blockedBy: string[],
        ): EnhancedBusinessError => ({
            code: "WORKFLOW_BLOCKED",
            message: "Cannot proceed with current workflow",
            type: "workflow",
            details: {
                currentStep,
                blockedBy,
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
        }),

        /**
         * Create a custom business error
         */
        createBusinessError: (
            code: string,
            message: string,
            type: EnhancedBusinessError["type"],
            details?: any,
        ): EnhancedBusinessError => ({
            code,
            message,
            type,
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
            ...(details && { details }),
        }),
    },
};

// Create factory instance
export const businessErrorFactory = new BusinessErrorFactory();
