/**
 * Business logic error fixtures
 * 
 * These fixtures provide business rule violation scenarios including
 * resource limits, workflow errors, data conflicts, and domain-specific errors.
 */

export interface BusinessError {
    code: string;
    message: string;
    type: "limit" | "conflict" | "state" | "workflow" | "constraint" | "policy";
    details?: {
        current?: number | string;
        limit?: number | string;
        required?: number | string;
        conflictWith?: string;
        suggestion?: string;
        [key: string]: any;
    };
    userAction?: {
        label: string;
        action: string;
        url?: string;
    };
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
        } satisfies BusinessError,

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
        } satisfies BusinessError,

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
        } satisfies BusinessError,

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
        } satisfies BusinessError,
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
        } satisfies BusinessError,

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
        } satisfies BusinessError,

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
        } satisfies BusinessError,

        dependencyNotReady: {
            code: "DEPENDENCY_NOT_READY",
            message: "Required dependencies are not available",
            type: "workflow",
            details: {
                dependencies: ["data_source_connection", "api_key_validation"],
                readyDependencies: ["data_source_connection"],
            },
        } satisfies BusinessError,
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
        } satisfies BusinessError,

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
        } satisfies BusinessError,

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
        } satisfies BusinessError,

        resourceLocked: {
            code: "RESOURCE_LOCKED",
            message: "This resource is currently being edited by another user",
            type: "conflict",
            details: {
                lockedBy: "user_789",
                lockedAt: new Date(Date.now() - 120000).toISOString(),
                expectedRelease: new Date(Date.now() + 180000).toISOString(),
            },
        } satisfies BusinessError,
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
        } satisfies BusinessError,

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
        } satisfies BusinessError,

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
        } satisfies BusinessError,

        maintenanceMode: {
            code: "MAINTENANCE_MODE",
            message: "This feature is currently under maintenance",
            type: "state",
            details: {
                feature: "ai_execution",
                estimatedEndTime: new Date(Date.now() + 3600000).toISOString(),
                alternativeAction: "manual_execution",
            },
        } satisfies BusinessError,
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
        } satisfies BusinessError,

        invalidHierarchy: {
            code: "INVALID_HIERARCHY",
            message: "Invalid parent-child relationship",
            type: "constraint",
            details: {
                parent: "team_123",
                child: "team_456",
                reason: "Child team is already a parent of the proposed parent",
            },
        } satisfies BusinessError,

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
        } satisfies BusinessError,

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
        } satisfies BusinessError,
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
        } satisfies BusinessError,

        regionRestriction: {
            code: "REGION_RESTRICTED",
            message: "This feature is not available in your region",
            type: "policy",
            details: {
                userRegion: "EU",
                allowedRegions: ["US", "CA"],
                reason: "regulatory_compliance",
            },
        } satisfies BusinessError,

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
        } satisfies BusinessError,

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
        } satisfies BusinessError,
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
        ): BusinessError => ({
            code: `${resource.toUpperCase()}_LIMIT_EXCEEDED`,
            message: `You have reached the ${resource} limit`,
            type: "limit",
            details: { current, limit },
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
        createConflictError: (field: string, value: string, suggestion?: string): BusinessError => ({
            code: "CONFLICT",
            message: `A conflict was detected for ${field}`,
            type: "conflict",
            details: {
                field,
                value,
                ...(suggestion && { suggestion }),
            },
        }),

        /**
         * Create a workflow error
         */
        createWorkflowError: (
            currentStep: string,
            blockedBy: string[],
        ): BusinessError => ({
            code: "WORKFLOW_BLOCKED",
            message: "Cannot proceed with current workflow",
            type: "workflow",
            details: {
                currentStep,
                blockedBy,
            },
        }),

        /**
         * Create a custom business error
         */
        createBusinessError: (
            code: string,
            message: string,
            type: BusinessError["type"],
            details?: any,
        ): BusinessError => ({
            code,
            message,
            type,
            ...(details && { details }),
        }),
    },
};