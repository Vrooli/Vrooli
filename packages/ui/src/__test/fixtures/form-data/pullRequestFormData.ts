import type { PullRequestCreateInput, PullRequestUpdateInput } from "@vrooli/shared";

/**
 * Form data fixtures for pull request-related forms
 * These represent data as it appears in form state before submission
 */

/**
 * Pull request creation form data
 */
export const minimalPullRequestCreateFormInput: Partial<PullRequestCreateInput> = {
    toObjectType: "Resource",
    toId: "resource_123456789",
    fromId: "routine_987654321",
    description: "Adding validation improvements to the resource",
};

export const completePullRequestCreateFormInput = {
    toObjectType: "Resource",
    toId: "resource_123456789",
    fromId: "routine_987654321",
    description: "This pull request adds comprehensive validation improvements to enhance data integrity and user experience. Changes include:\n\n- Added input validation for all form fields\n- Improved error messaging\n- Added unit tests for validation logic\n- Updated documentation",
    language: "en",
    translations: [
        {
            language: "es",
            description: "Esta solicitud de extracción añade mejoras completas de validación para mejorar la integridad de los datos y la experiencia del usuario.",
        },
        {
            language: "fr",
            description: "Cette demande de tirage ajoute des améliorations de validation complètes pour améliorer l'intégrité des données et l'expérience utilisateur.",
        },
    ],
};

/**
 * Pull request update form data
 */
export const pullRequestUpdateFormInput = {
    status: "Open",
    description: "Updated description with implementation details:\n\n- Added input validation for all form fields\n- Improved error messaging with internationalization\n- Added comprehensive unit tests\n- Updated API documentation\n- Fixed edge cases found in testing",
};

export const pullRequestStatusUpdateFormInput = {
    status: "Merged",
    mergeNote: "Looks good! All tests passing and documentation is updated.",
};

/**
 * Pull request review form data
 */
export const pullRequestReviewFormInput = {
    reviewType: "Approve",
    comment: "Great work! The validation improvements look solid and the tests provide good coverage.",
    suggestedChanges: [],
};

export const pullRequestRequestChangesFormInput = {
    reviewType: "RequestChanges",
    comment: "Good start, but needs some adjustments before merging.",
    suggestedChanges: [
        "Add validation for edge case when input is empty array",
        "Update error message to be more descriptive",
        "Add test case for concurrent updates",
    ],
};

/**
 * Pull request merge form data
 */
export const pullRequestMergeFormInput = {
    mergeStrategy: "merge", // "merge" | "squash" | "rebase"
    commitMessage: "Merge: Add validation improvements (#123)",
    deleteSourceBranch: true,
};

/**
 * Pull request comment form data
 */
export const pullRequestCommentFormInput = {
    comment: "Thanks for the review! I've addressed all the suggested changes.",
    codeSnippet: null,
    attachments: [],
};

/**
 * Pull request filters form data
 */
export const pullRequestFilterFormInput = {
    status: ["Open", "Draft"],
    author: "user_123456789",
    assignee: "user_987654321",
    labels: ["enhancement", "validation"],
    dateRange: {
        from: "2024-01-01",
        to: "2024-12-31",
    },
    sortBy: "updatedAt", // "createdAt" | "updatedAt" | "status"
    sortOrder: "desc", // "asc" | "desc"
};

/**
 * Form validation states
 */
export const pullRequestFormValidationStates = {
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            toObjectType: "", // Required but empty
            toId: "invalid-id", // Invalid format
            fromId: "", // Required but empty
            description: "", // Required but empty
        },
        errors: {
            toObjectType: "Target object type is required",
            toId: "Invalid object ID format",
            fromId: "Source object is required",
            description: "Description is required",
        },
        touched: {
            toObjectType: true,
            toId: true,
            fromId: true,
            description: true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: minimalPullRequestCreateFormInput,
        errors: {},
        touched: {
            toObjectType: true,
            toId: true,
            fromId: true,
            description: true,
        },
        isValid: true,
        isSubmitting: false,
    },
};

/**
 * Helper function to create pull request form initial values
 */
export const createPullRequestFormInitialValues = (pullRequestData?: Partial<any>) => ({
    toObjectType: pullRequestData?.toObjectType || "Resource",
    toId: pullRequestData?.toId || "",
    fromId: pullRequestData?.fromId || "",
    description: pullRequestData?.description || "",
    status: pullRequestData?.status || "Draft",
    language: pullRequestData?.language || "en",
    translations: pullRequestData?.translations || [],
    ...pullRequestData,
});

/**
 * Helper function to validate object IDs
 */
export const validateObjectId = (id: string): string | null => {
    if (!id) return "ID is required";
    if (!/^\w+_\d{9,}$/.test(id)) {
        return "Invalid ID format (expected: type_123456789)";
    }
    return null;
};

/**
 * Helper function to transform form data to API format
 */
export const transformPullRequestFormToApiInput = (formData: any) => ({
    toObjectType: formData.toObjectType,
    toConnect: formData.toId,
    fromConnect: formData.fromId,
    // Transform main description to translations format
    translationsCreate: [
        {
            language: formData.language || "en",
            text: formData.description,
        },
        ...(formData.translations?.map((t: any) => ({
            language: t.language,
            text: t.description,
        })) || []),
    ],
    status: formData.status,
});

/**
 * Mock suggestions for pull request forms
 */
export const mockPullRequestSuggestions = {
    objectTypes: [
        { value: "Resource", label: "Resource" },
        { value: "Routine", label: "Routine" },
        { value: "Standard", label: "Standard" },
        { value: "Code", label: "Code" },
    ],
    statuses: [
        { value: "Draft", label: "Draft", icon: "edit" },
        { value: "Open", label: "Open", icon: "merge_type" },
        { value: "Merged", label: "Merged", icon: "check_circle" },
        { value: "Rejected", label: "Rejected", icon: "cancel" },
        { value: "Canceled", label: "Canceled", icon: "block" },
    ],
    mergeStrategies: [
        {
            value: "merge",
            label: "Create merge commit",
            description: "All commits will be added to the base branch",
        },
        {
            value: "squash",
            label: "Squash and merge",
            description: "Combine all commits into a single commit",
        },
        {
            value: "rebase",
            label: "Rebase and merge",
            description: "Add commits individually to the base branch",
        },
    ],
};

/**
 * Pull request template form data
 */
export const pullRequestTemplates = {
    feature: {
        title: "Feature: ",
        description: `## Description
Brief description of the feature

## Changes Made
- [ ] Implementation detail 1
- [ ] Implementation detail 2
- [ ] Implementation detail 3

## Testing
- [ ] Unit tests added
- [ ] Integration tests added
- [ ] Manual testing completed

## Documentation
- [ ] Code comments added
- [ ] README updated
- [ ] API docs updated`,
    },
    bugfix: {
        title: "Fix: ",
        description: `## Bug Description
What was broken?

## Root Cause
Why was it broken?

## Solution
How was it fixed?

## Testing
- [ ] Bug reproduced before fix
- [ ] Fix verified
- [ ] Regression tests added`,
    },
    refactor: {
        title: "Refactor: ",
        description: `## Refactoring Goal
What are we improving?

## Changes Made
- [ ] Change 1
- [ ] Change 2
- [ ] Change 3

## Impact
- Performance: [improved/same/degraded]
- Maintainability: [improved/same/degraded]
- Test coverage: [improved/same/degraded]`,
    },
};