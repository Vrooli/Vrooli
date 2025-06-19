import type { ResourceVersionRelationCreateInput, ResourceVersionRelationUpdateInput } from "@vrooli/shared";

/**
 * Form data fixtures for resource version relation creation and editing
 * These represent data as it appears in form state before submission
 * 
 * ResourceVersionRelations define relationships between different versions of resources,
 * such as dependencies, upgrades, replacements, and other version-to-version connections.
 * They use labels to categorize the type of relationship.
 */

/**
 * Minimal resource version relation form input - basic relationship
 */
export const minimalResourceVersionRelationFormInput: Partial<ResourceVersionRelationCreateInput> = {
    fromVersionConnect: "123456789012345678", // 18-digit snowflake ID
    toVersionConnect: "123456789012345679",   // 18-digit snowflake ID
    labels: [],
};

/**
 * Complete resource version relation form input with labels
 */
export const completeResourceVersionRelationFormInput = {
    fromVersionConnect: "123456789012345680", // 18-digit snowflake ID
    toVersionConnect: "123456789012345681",   // 18-digit snowflake ID
    labels: ["dependency", "upgrade", "replaces"],
};

/**
 * Resource version relation form variants for different relationship types
 */
export const resourceVersionRelationFormVariants = {
    dependency: {
        fromVersionConnect: "123456789012345682",
        toVersionConnect: "123456789012345683",
        labels: ["dependency"],
    },
    upgrade: {
        fromVersionConnect: "123456789012345684",
        toVersionConnect: "123456789012345685",
        labels: ["upgrade", "successor"],
    },
    replacement: {
        fromVersionConnect: "123456789012345686",
        toVersionConnect: "123456789012345687",
        labels: ["replaces", "deprecated"],
    },
    compatibility: {
        fromVersionConnect: "123456789012345688",
        toVersionConnect: "123456789012345689",
        labels: ["compatible", "tested-with"],
    },
    versionEvolution: {
        fromVersionConnect: "123456789012345690",
        toVersionConnect: "123456789012345691",
        labels: ["upgrade", "next-version", "improvement"],
    },
    bugFix: {
        fromVersionConnect: "123456789012345692",
        toVersionConnect: "123456789012345693",
        labels: ["hotfix", "bug-fix", "patch"],
    },
    apiChange: {
        fromVersionConnect: "123456789012345694",
        toVersionConnect: "123456789012345695",
        labels: ["api-change", "breaking_change", "v1.0->v2.0"],
    },
    securityUpdate: {
        fromVersionConnect: "123456789012345696",
        toVersionConnect: "123456789012345697",
        labels: ["security patch", "vulnerability fix"],
    },
    featureAddition: {
        fromVersionConnect: "123456789012345698",
        toVersionConnect: "123456789012345699",
        labels: ["feature addition", "enhancement"],
    },
    performanceImprovement: {
        fromVersionConnect: "123456789012345700",
        toVersionConnect: "123456789012345701",
        labels: ["performance", "optimization"],
    },
};

/**
 * Resource version relation update form data
 */
export const updateResourceVersionRelationFormInput = {
    labels: ["updated", "verified", "tested"],
};

/**
 * Multi-label resource version relation form data
 */
export const multiLabelResourceVersionRelationFormInput = {
    fromVersionConnect: "123456789012345702",
    toVersionConnect: "123456789012345703",
    labels: ["dependency", "upgrade", "replaces", "successor", "predecessor"],
};

/**
 * Form validation test cases
 */
export const invalidResourceVersionRelationFormInputs = {
    missingFromVersion: {
        fromVersionConnect: "", // Invalid: empty string
        toVersionConnect: "123456789012345679",
        labels: ["dependency"],
    },
    missingToVersion: {
        fromVersionConnect: "123456789012345678",
        toVersionConnect: "", // Invalid: empty string
        labels: ["dependency"],
    },
    invalidFromVersionId: {
        fromVersionConnect: "invalid-id", // Invalid: not a snowflake ID
        toVersionConnect: "123456789012345679",
        labels: ["dependency"],
    },
    invalidToVersionId: {
        fromVersionConnect: "123456789012345678",
        toVersionConnect: "not-valid-snowflake", // Invalid: not a snowflake ID
        labels: ["dependency"],
    },
    sameVersion: {
        fromVersionConnect: "123456789012345678",
        toVersionConnect: "123456789012345678", // Invalid: same version
        labels: ["self-reference"],
    },
    emptyLabels: {
        fromVersionConnect: "123456789012345678",
        toVersionConnect: "123456789012345679",
        labels: [""], // Invalid: empty string in labels
    },
    longLabel: {
        fromVersionConnect: "123456789012345678",
        toVersionConnect: "123456789012345679",
        labels: ["x".repeat(129)], // Invalid: exceeds 128 character limit
    },
    invalidLabelsType: {
        fromVersionConnect: "123456789012345678",
        toVersionConnect: "123456789012345679",
        // @ts-expect-error - Testing invalid input
        labels: "not-an-array", // Invalid: should be array
    },
    missingBothVersions: {
        fromVersionConnect: "", // Invalid: missing both
        toVersionConnect: "",   // Invalid: missing both
        labels: ["dependency"],
    },
};

/**
 * Form validation states for testing
 */
export const resourceVersionRelationFormValidationStates = {
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            fromVersionConnect: "",
            toVersionConnect: "invalid-id",
            labels: ["x".repeat(129)],
        },
        errors: {
            fromVersionConnect: "From version is required",
            toVersionConnect: "Please enter a valid version ID",
            "labels.0": "Label must be 128 characters or less",
        },
        touched: {
            fromVersionConnect: true,
            toVersionConnect: true,
            "labels.0": true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: minimalResourceVersionRelationFormInput,
        errors: {},
        touched: {
            fromVersionConnect: true,
            toVersionConnect: true,
        },
        isValid: true,
        isSubmitting: false,
    },
    submitting: {
        values: completeResourceVersionRelationFormInput,
        errors: {},
        touched: {},
        isValid: true,
        isSubmitting: true,
    },
};

/**
 * Helper function to create resource version relation form initial values
 */
export const createResourceVersionRelationFormInitialValues = (relationData?: Partial<any>) => ({
    fromVersionConnect: relationData?.fromVersionConnect || "",
    toVersionConnect: relationData?.toVersionConnect || "",
    labels: relationData?.labels || [],
    ...relationData,
});

/**
 * Helper function to validate resource version relation form
 */
export const validateResourceVersionRelationForm = (values: any): Record<string, string> => {
    const errors: Record<string, string> = {};

    if (!values.fromVersionConnect) {
        errors.fromVersionConnect = "From version is required";
    } else if (!/^\d{18,19}$/.test(values.fromVersionConnect)) {
        errors.fromVersionConnect = "Please enter a valid version ID";
    }

    if (!values.toVersionConnect) {
        errors.toVersionConnect = "To version is required";
    } else if (!/^\d{18,19}$/.test(values.toVersionConnect)) {
        errors.toVersionConnect = "Please enter a valid version ID";
    }

    if (values.fromVersionConnect && values.toVersionConnect && 
        values.fromVersionConnect === values.toVersionConnect) {
        errors.toVersionConnect = "To version cannot be the same as from version";
    }

    if (values.labels && Array.isArray(values.labels)) {
        values.labels.forEach((label: string, index: number) => {
            if (label && label.length > 128) {
                errors[`labels.${index}`] = "Label must be 128 characters or less";
            }
        });
    }

    return errors;
};

/**
 * Helper function to transform form data to API input format
 */
export const transformResourceVersionRelationFormToApiInput = (formData: any) => ({
    id: formData.id || `${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
    fromVersionConnect: formData.fromVersionConnect,
    toVersionConnect: formData.toVersionConnect,
    labels: Array.isArray(formData.labels) 
        ? formData.labels.filter((label: string) => label && label.trim().length > 0)
        : [],
});

/**
 * Mock relationship type suggestions for form inputs
 */
export const mockRelationshipTypes = [
    { value: "dependency", label: "Dependency", description: "This version depends on the target version" },
    { value: "upgrade", label: "Upgrade", description: "Target version is an upgrade of this version" },
    { value: "replaces", label: "Replaces", description: "Target version replaces this version" },
    { value: "compatible", label: "Compatible", description: "Versions are compatible with each other" },
    { value: "successor", label: "Successor", description: "Target version is the successor" },
    { value: "predecessor", label: "Predecessor", description: "Target version is the predecessor" },
    { value: "hotfix", label: "Hotfix", description: "Target version is a hotfix" },
    { value: "patch", label: "Patch", description: "Target version is a patch release" },
    { value: "feature", label: "Feature", description: "Target version adds new features" },
    { value: "breaking", label: "Breaking Change", description: "Target version has breaking changes" },
    { value: "security", label: "Security", description: "Target version addresses security issues" },
    { value: "performance", label: "Performance", description: "Target version improves performance" },
];

/**
 * Common label patterns for different relationship scenarios
 */
export const commonLabelPatterns = {
    versionUpgrade: ["upgrade", "next-version", "improvement"],
    dependencyRelation: ["dependency", "requirement", "uses"],
    replacementFlow: ["replaces", "deprecated", "obsolete"],
    compatibilityCheck: ["compatible", "tested-with", "verified"],
    bugFixChain: ["hotfix", "bug-fix", "patch", "stability"],
    featureEvolution: ["feature addition", "enhancement", "new-feature"],
    apiChanges: ["api-change", "breaking_change", "interface-update"],
    securityUpdates: ["security patch", "vulnerability fix", "security-enhancement"],
    performanceImprovements: ["performance", "optimization", "speed-improvement"],
    maintenanceUpdates: ["maintenance", "cleanup", "refactor"],
};

/**
 * Helper to generate valid snowflake IDs for testing
 */
export const generateTestVersionId = (suffix: number = 0): string => {
    const base = "123456789012345";
    const paddedSuffix = suffix.toString().padStart(3, "0");
    return base + paddedSuffix;
};

/**
 * Helper to create relationship form data with generated IDs
 */
export const createRelationshipFormData = (
    labels: string[] = [],
    fromSuffix: number = 1,
    toSuffix: number = 2
) => ({
    fromVersionConnect: generateTestVersionId(fromSuffix),
    toVersionConnect: generateTestVersionId(toSuffix),
    labels,
});