import { type ModelTestFixtures, TestDataFactory } from "../validationTestUtils.js";

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

// Shared resourceVersionRelation test fixtures
export const resourceVersionRelationFixtures: ModelTestFixtures = {
    minimal: {
        create: {
            id: validIds.id1,
            fromVersionConnect: validIds.id2,
            toVersionConnect: validIds.id3,
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id1,
            labels: ["dependency", "upgrade", "replaces"],
            fromVersionConnect: validIds.id2,
            toVersionConnect: validIds.id3,
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 123,
            unknownField3: true,
        },
        update: {
            id: validIds.id1,
            labels: ["updated", "verified", "tested"],
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 456,
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing required id, fromVersionConnect, and toVersionConnect
                labels: ["test"],
            },
            update: {
                // Missing required id
                labels: ["updated"],
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                labels: "not-an-array", // Should be array
                fromVersionConnect: 456, // Should be string
                toVersionConnect: 789, // Should be string
            },
            update: {
                id: validIds.id1,
                labels: { not: "array" }, // Should be array
            },
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
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
                toVersionConnect: validIds.id3,
                // Missing required fromVersionConnect
            },
        },
        missingToVersion: {
            create: {
                id: validIds.id1,
                fromVersionConnect: validIds.id2,
                // Missing required toVersionConnect
            },
        },
        longLabel: {
            create: {
                id: validIds.id1,
                labels: ["x".repeat(129)], // Too long (exceeds 128)
                fromVersionConnect: validIds.id2,
                toVersionConnect: validIds.id3,
            },
        },
        emptyLabel: {
            create: {
                id: validIds.id1,
                labels: [""], // Empty string should be removed
                fromVersionConnect: validIds.id2,
                toVersionConnect: validIds.id3,
            },
        },
    },
    edgeCases: {
        singleLabel: {
            create: {
                id: validIds.id1,
                labels: ["dependency"],
                fromVersionConnect: validIds.id2,
                toVersionConnect: validIds.id3,
            },
        },
        multipleLabels: {
            create: {
                id: validIds.id1,
                labels: ["dependency", "upgrade", "replaces", "successor", "predecessor"],
                fromVersionConnect: validIds.id2,
                toVersionConnect: validIds.id3,
            },
        },
        dependencyRelation: {
            create: {
                id: validIds.id1,
                labels: ["dependency"],
                fromVersionConnect: validIds.id2,
                toVersionConnect: validIds.id3,
            },
        },
        upgradeRelation: {
            create: {
                id: validIds.id1,
                labels: ["upgrade", "successor"],
                fromVersionConnect: validIds.id2,
                toVersionConnect: validIds.id3,
            },
        },
        replacementRelation: {
            create: {
                id: validIds.id1,
                labels: ["replaces", "deprecated"],
                fromVersionConnect: validIds.id2,
                toVersionConnect: validIds.id3,
            },
        },
        withoutLabels: {
            create: {
                id: validIds.id1,
                fromVersionConnect: validIds.id2,
                toVersionConnect: validIds.id3,
                // No labels array
            },
        },
        emptyLabels: {
            create: {
                id: validIds.id1,
                labels: [], // Empty array
                fromVersionConnect: validIds.id2,
                toVersionConnect: validIds.id3,
            },
        },
        maxLengthLabels: {
            create: {
                id: validIds.id1,
                labels: ["a".repeat(128), "b".repeat(128)], // Maximum length labels
                fromVersionConnect: validIds.id2,
                toVersionConnect: validIds.id3,
            },
        },
        commonRelationTypes: {
            create: {
                id: validIds.id1,
                labels: ["dependency", "requirement", "uses", "includes"],
                fromVersionConnect: validIds.id2,
                toVersionConnect: validIds.id3,
            },
        },
        versionEvolution: {
            create: {
                id: validIds.id1,
                labels: ["upgrade", "next-version", "improvement"],
                fromVersionConnect: validIds.id2,
                toVersionConnect: validIds.id3,
            },
        },
        compatibilityRelation: {
            create: {
                id: validIds.id1,
                labels: ["compatible", "tested-with"],
                fromVersionConnect: validIds.id2,
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
                fromVersionConnect: validIds.id2,
                toVersionConnect: validIds.id3,
            },
        },
        labelsWithSpecialChars: {
            create: {
                id: validIds.id1,
                labels: ["v1.0->v2.0", "api-change", "breaking_change"],
                fromVersionConnect: validIds.id2,
                toVersionConnect: validIds.id3,
            },
        },
        relationToSameVersion: {
            create: {
                id: validIds.id1,
                labels: ["self-reference"],
                fromVersionConnect: validIds.id2,
                toVersionConnect: validIds.id2, // Same version
            },
        },
    },
};

// Custom factory that always generates valid IDs and required fields
const customizers = {
    create: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
        fromVersionConnect: base.fromVersionConnect || validIds.id2,
        toVersionConnect: base.toVersionConnect || validIds.id3,
    }),
    update: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export a factory for creating test data programmatically
export const resourceVersionRelationTestDataFactory = new TestDataFactory(resourceVersionRelationFixtures, customizers);