import {
    resourceFormConfig,
    resourceValidation,
    ResourceUsedFor,
    type ResourceCreateInput,
    type ResourceShape,
    type ResourceUpdateInput,
    type Session,
} from "@vrooli/shared";
import { transformResourceValues } from "../../../views/objects/resource/ResourceUpsert.js";
import { createUIFormTestFactory, type UIFormTestConfig } from "./UIFormTestFactory.js";

/**
 * Configuration for Resource form testing with data-driven test scenarios
 */
const resourceFormTestConfig: UIFormTestConfig<ResourceShape, ResourceShape, ResourceCreateInput, ResourceUpdateInput, ResourceShape> = {
    // Form metadata
    objectType: "Resource",
    formFixtures: {
        minimal: {
            __typename: "Resource" as const,
            id: "resource_minimal",
            index: 0,
            link: "",
            usedFor: ResourceUsedFor.Context,
            list: {
                __typename: "ResourceList" as const,
                id: "list_minimal",
                listFor: { __typename: "RoutineVersion", id: "rv_minimal" },
            },
            translations: [{
                __typename: "ResourceTranslation" as const,
                id: "rt_minimal",
                language: "en",
                description: "",
                name: "Test Resource",
            }],
        },
        complete: {
            __typename: "Resource" as const,
            id: "resource_complete",
            index: 1,
            link: "https://example.com/resource",
            usedFor: ResourceUsedFor.ExternalLink,
            list: {
                __typename: "ResourceList" as const,
                id: "list_complete",
                listFor: { __typename: "RoutineVersion", id: "rv_complete" },
            },
            translations: [{
                __typename: "ResourceTranslation" as const,
                id: "rt_complete",
                language: "en",
                description: "A helpful resource for learning",
                name: "Example Resource",
            }],
        },
        invalid: {
            __typename: "Resource" as const,
            id: "resource_invalid",
            index: -1, // Invalid: negative index
            link: "not-a-valid-url", // Invalid: malformed URL
            usedFor: ResourceUsedFor.Context,
            list: {
                __typename: "ResourceList" as const,
                id: "",
                listFor: { __typename: "RoutineVersion", id: "" },
            },
            translations: [],
        },
        edgeCase: {
            __typename: "Resource" as const,
            id: "resource_edge",
            index: 999,
            link: "https://very-long-domain-name-that-tests-edge-cases.example.com/very/long/path/to/resource/that/might/cause/issues",
            usedFor: ResourceUsedFor.Tutorial,
            list: {
                __typename: "ResourceList" as const,
                id: "list_edge",
                listFor: { __typename: "ProjectVersion", id: "pv_edge" },
            },
            translations: [{
                __typename: "ResourceTranslation" as const,
                id: "rt_edge",
                language: "en",
                description: "A".repeat(1024), // Edge case: long description
                name: "Edge Case Resource with Long Name",
            }],
        },
    },

    // Validation schemas from shared package
    validation: resourceValidation,

    // API endpoints from shared package
    endpoints: {
        create: resourceFormConfig.endpoints.createOne,
        update: resourceFormConfig.endpoints.updateOne,
    },

    // Transform functions - form already uses ResourceShape, so no transformation needed
    formToShape: (formData: ResourceShape) => formData,

    transformFunction: (shape: ResourceShape, existing: ResourceShape, isCreate: boolean) => {
        const result = transformResourceValues(shape, existing, isCreate);
        if (!result) {
            throw new Error("Transform function returned undefined");
        }
        return result;
    },

    initialValuesFunction: (session?: Session, existing?: Partial<ResourceShape>): ResourceShape => {
        return resourceFormConfig.transformations.getInitialValues(session, existing || {});
    },

    // DATA-DRIVEN TEST SCENARIOS - replaces all custom wrapper methods
    testScenarios: {
        linkValidation: {
            description: "Test resource link validation",
            testCases: [
                {
                    name: "Valid HTTP URL",
                    field: "link",
                    value: "http://example.com",
                    shouldPass: true,
                },
                {
                    name: "Valid HTTPS URL",
                    field: "link",
                    value: "https://example.com/resource",
                    shouldPass: true,
                },
                {
                    name: "Empty link",
                    field: "link",
                    value: "",
                    shouldPass: true, // Links are optional
                },
                {
                    name: "Invalid URL format",
                    field: "link",
                    value: "not-a-url",
                    shouldPass: false,
                },
            ],
        },

        usedForValidation: {
            description: "Test resource usedFor field",
            testCases: [
                {
                    name: "Context usage",
                    field: "usedFor",
                    value: ResourceUsedFor.Context,
                    shouldPass: true,
                },
                {
                    name: "ExternalLink usage",
                    field: "usedFor",
                    value: ResourceUsedFor.ExternalLink,
                    shouldPass: true,
                },
                {
                    name: "Tutorial usage",
                    field: "usedFor",
                    value: ResourceUsedFor.Tutorial,
                    shouldPass: true,
                },
            ],
        },

        indexValidation: {
            description: "Test resource index field",
            testCases: [
                {
                    name: "Zero index",
                    field: "index",
                    value: 0,
                    shouldPass: true,
                },
                {
                    name: "Positive index",
                    field: "index",
                    value: 5,
                    shouldPass: true,
                },
                {
                    name: "Large index",
                    field: "index",
                    value: 999,
                    shouldPass: true,
                },
            ],
        },

        translationValidation: {
            description: "Test resource translations",
            testCases: [
                {
                    name: "Valid name and description",
                    data: {
                        translations: [{
                            __typename: "ResourceTranslation",
                            id: "rt_test",
                            language: "en",
                            name: "Test Resource",
                            description: "A test resource",
                        }],
                    },
                    shouldPass: true,
                },
                {
                    name: "Empty translations",
                    data: {
                        translations: [{
                            __typename: "ResourceTranslation",
                            id: "rt_empty",
                            language: "en",
                            name: "",
                            description: "",
                        }],
                    },
                    shouldPass: true,
                },
            ],
        },
    },
};

/**
 * SIMPLIFIED: Direct factory export - no wrapper function needed!
 */
export const resourceFormTestFactory = createUIFormTestFactory(resourceFormTestConfig);

/**
 * Type exports for use in other test files
 */
export { resourceFormTestConfig };
