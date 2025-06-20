import type { ResourceVersionRelationCreateInput, ResourceVersionRelationUpdateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TestDataFactory, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { resourceVersionRelationValidation } from "../../../validation/models/resourceVersionRelation.js";

// Magic number constants for testing
const LABEL_TOO_LONG_LENGTH = 129;
const LABEL_MAX_LENGTH = 128;

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    id4: "123456789012345681",
    id5: "123456789012345682",
    id6: "123456789012345683",
    id7: "123456789012345684",
    id8: "123456789012345685",
};

// Extended create input type that makes labels optional for testing scenarios where it might be missing
// Also allows unknown fields for testing field stripping behavior
type ExtendedResourceVersionRelationCreateInput = Omit<ResourceVersionRelationCreateInput, "labels"> & {
    labels?: string[];
    unknownField1?: string;
    unknownField2?: number;
    unknownField3?: boolean;
};

// Extended update input type that allows unknown fields for testing field stripping behavior
type ExtendedResourceVersionRelationUpdateInput = ResourceVersionRelationUpdateInput & {
    unknownField1?: string;
    unknownField2?: number;
};

// Shared resourceVersionRelation test fixtures
export const resourceVersionRelationFixtures: ModelTestFixtures<ResourceVersionRelationCreateInput | ExtendedResourceVersionRelationCreateInput, ResourceVersionRelationUpdateInput | ExtendedResourceVersionRelationUpdateInput> = {
    minimal: {
        create: {
            id: validIds.id1,
            toVersionConnect: validIds.id3,
            labels: [],
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id1,
            labels: ["dependency", "upgrade", "replaces"],
            toVersionConnect: validIds.id3,
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 123,
            unknownField3: true,
        } as ExtendedResourceVersionRelationCreateInput,
        update: {
            id: validIds.id1,
            labels: ["updated", "verified", "tested"],
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 456,
        } as ExtendedResourceVersionRelationUpdateInput,
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing required id, fromVersionConnect, and toVersionConnect
                labels: ["test"],
            } as unknown as ResourceVersionRelationCreateInput,
            update: {
                // Missing required id
                labels: ["updated"],
            } as unknown as ResourceVersionRelationUpdateInput,
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                labels: "not-an-array", // Should be array
                fromVersionConnect: 456, // Should be string
                toVersionConnect: 789, // Should be string
            } as unknown as ResourceVersionRelationCreateInput,
            update: {
                id: validIds.id1,
                labels: { not: "array" }, // Should be array
            } as unknown as ResourceVersionRelationUpdateInput,
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                labels: [],
                fromVersionConnect: validIds.id2,
                toVersionConnect: validIds.id3,
            },
            update: {
                id: "invalid-id",
            },
        },
        missingFromVersion: {
            create: {
                id: validIds.id1,
                labels: [],
                toVersionConnect: validIds.id3,
                // Missing required fromVersionConnect
            } as unknown as ResourceVersionRelationCreateInput,
        },
        missingToVersion: {
            create: {
                id: validIds.id1,
                labels: [],
                fromVersionConnect: validIds.id2,
                // Missing required toVersionConnect
            } as unknown as ResourceVersionRelationCreateInput,
        },
        longLabel: {
            create: {
                id: validIds.id1,
                labels: ["x".repeat(LABEL_TOO_LONG_LENGTH)], // Too long (exceeds 128)
                toVersionConnect: validIds.id3,
            },
        },
        emptyLabel: {
            create: {
                id: validIds.id1,
                labels: [""], // Empty string should be removed
                toVersionConnect: validIds.id3,
            },
        },
    },
    edgeCases: {
        singleLabel: {
            create: {
                id: validIds.id1,
                labels: ["dependency"],
                toVersionConnect: validIds.id3,
            },
        },
        multipleLabels: {
            create: {
                id: validIds.id1,
                labels: ["dependency", "upgrade", "replaces", "successor", "predecessor"],
                toVersionConnect: validIds.id3,
            },
        },
        dependencyRelation: {
            create: {
                id: validIds.id1,
                labels: ["dependency"],
                toVersionConnect: validIds.id3,
            },
        },
        upgradeRelation: {
            create: {
                id: validIds.id1,
                labels: ["upgrade", "successor"],
                toVersionConnect: validIds.id3,
            },
        },
        replacementRelation: {
            create: {
                id: validIds.id1,
                labels: ["replaces", "deprecated"],
                toVersionConnect: validIds.id3,
            },
        },
        withoutLabels: {
            create: {
                id: validIds.id1,
                toVersionConnect: validIds.id3,
                labels: [],
                // No labels array
            },
        },
        emptyLabels: {
            create: {
                id: validIds.id1,
                labels: [], // Empty array
                toVersionConnect: validIds.id3,
            },
        },
        maxLengthLabels: {
            create: {
                id: validIds.id1,
                labels: ["a".repeat(LABEL_MAX_LENGTH), "b".repeat(LABEL_MAX_LENGTH)], // Maximum length labels
                toVersionConnect: validIds.id3,
            },
        },
        commonRelationTypes: {
            create: {
                id: validIds.id1,
                labels: ["dependency", "requirement", "uses", "includes"],
                toVersionConnect: validIds.id3,
            },
        },
        versionEvolution: {
            create: {
                id: validIds.id1,
                labels: ["upgrade", "next-version", "improvement"],
                toVersionConnect: validIds.id3,
            },
        },
        compatibilityRelation: {
            create: {
                id: validIds.id1,
                labels: ["compatible", "tested-with"],
                toVersionConnect: validIds.id3,
            },
        },
        updateOnlyId: {
            update: {
                id: validIds.id1,
                // Only required field
            },
        },
        updateLabels: {
            update: {
                id: validIds.id1,
                labels: ["updated", "verified"],
            },
        },
        updateEmptyLabels: {
            update: {
                id: validIds.id1,
                labels: [], // Clear all labels
            },
        },
        updateSingleLabel: {
            update: {
                id: validIds.id1,
                labels: ["hotfix"],
            },
        },
        labelsWithSpaces: {
            create: {
                id: validIds.id1,
                labels: ["version upgrade", "bug fix", "security patch"],
                toVersionConnect: validIds.id3,
            },
        },
        labelsWithSpecialChars: {
            create: {
                id: validIds.id1,
                labels: ["v1.0->v2.0", "api-change", "breaking_change"],
                toVersionConnect: validIds.id3,
            },
        },
        relationToSameVersion: {
            create: {
                id: validIds.id1,
                labels: ["self-reference"],
                toVersionConnect: validIds.id2, // Same version
            },
        },
    },
};

// Custom factory that always generates valid IDs and required fields
const customizers = {
    create: (base: ResourceVersionRelationCreateInput | ExtendedResourceVersionRelationCreateInput): ResourceVersionRelationCreateInput => ({
        id: base.id || validIds.id1,
        toVersionConnect: base.toVersionConnect || validIds.id3,
        labels: base.labels || [],
    }),
    update: (base: ResourceVersionRelationUpdateInput | ExtendedResourceVersionRelationUpdateInput): ResourceVersionRelationUpdateInput => ({
        id: base.id || validIds.id1,
        ...(base.labels !== undefined && { labels: base.labels }),
    }),
};

// Export a factory for creating test data programmatically
export const resourceVersionRelationTestDataFactory = new TypedTestDataFactory(resourceVersionRelationFixtures, resourceVersionRelationValidation, customizers);

// Export typed fixtures with validation
export const typedResourceVersionRelationFixtures = createTypedFixtures(resourceVersionRelationFixtures, resourceVersionRelationValidation);

// Legacy export for backward compatibility
export const legacyResourceVersionRelationTestDataFactory = new TestDataFactory(resourceVersionRelationFixtures, customizers);
