import { type ModelTestFixtures, TestDataFactory } from "../validationTestUtils.js";

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    id4: "123456789012345681",
    id5: "123456789012345682",
    id6: "123456789012345683",
};

// Shared report test fixtures
export const reportFixtures: ModelTestFixtures = {
    minimal: {
        create: {
            id: validIds.id1,
            createdForType: "User",
            language: "en",
            reason: "Inappropriate content",
            createdForConnect: validIds.id2,
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id1,
            createdForType: "Comment",
            details: "This comment contains spam and violates community guidelines. It includes inappropriate links and promotional content that is not relevant to the discussion.",
            language: "en",
            reason: "Spam and promotional content",
            createdForConnect: validIds.id3,
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 123,
            unknownField3: true,
        },
        update: {
            id: validIds.id1,
            details: "Updated report details with additional information about the violation.",
            language: "es",
            reason: "Updated reason for the report",
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 456,
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing required id, createdForType, language, reason, and createdForConnect
                details: "Incomplete report",
            },
            update: {
                // Missing required id
                details: "Updated details",
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                createdForType: "InvalidType", // Invalid enum value
                details: 456, // Should be string
                language: 789, // Should be string
                reason: 101112, // Should be string
                createdForConnect: 131415, // Should be string
            },
            update: {
                id: validIds.id1,
                details: 123, // Should be string
                language: 456, // Should be string
                reason: 789, // Should be string
            },
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                createdForType: "User",
                language: "en",
                reason: "Test reason",
                createdForConnect: validIds.id2,
            },
            update: {
                id: "invalid-id",
            },
        },
        invalidCreatedForType: {
            create: {
                id: validIds.id1,
                createdForType: "UnknownType", // Not a valid enum value
                language: "en",
                reason: "Test reason",
                createdForConnect: validIds.id2,
            },
        },
        missingCreatedFor: {
            create: {
                id: validIds.id1,
                createdForType: "User",
                language: "en",
                reason: "Test reason",
                // Missing required createdForConnect
            },
        },
        invalidCreatedForConnect: {
            create: {
                id: validIds.id1,
                createdForType: "User",
                language: "en",
                reason: "Test reason",
                createdForConnect: "invalid-connect-id",
            },
        },
        invalidLanguage: {
            create: {
                id: validIds.id1,
                createdForType: "User",
                language: "xyz", // Invalid language code
                reason: "Test reason",
                createdForConnect: validIds.id2,
            },
        },
        emptyReason: {
            create: {
                id: validIds.id1,
                createdForType: "User",
                language: "en",
                reason: "", // Empty reason (should fail)
                createdForConnect: validIds.id2,
            },
        },
        longReason: {
            create: {
                id: validIds.id1,
                createdForType: "User",
                language: "en",
                reason: "x".repeat(129), // Exceeds max length
                createdForConnect: validIds.id2,
            },
        },
        longDetails: {
            create: {
                id: validIds.id1,
                createdForType: "User",
                language: "en",
                reason: "Test reason",
                details: "x".repeat(8193), // Exceeds max length
                createdForConnect: validIds.id2,
            },
        },
    },
    edgeCases: {
        withoutDetails: {
            create: {
                id: validIds.id1,
                createdForType: "Team",
                language: "en",
                reason: "Policy violation",
                createdForConnect: validIds.id4,
                // No details field
            },
        },
        chatMessageReport: {
            create: {
                id: validIds.id1,
                createdForType: "ChatMessage",
                language: "en",
                reason: "Harassment",
                details: "This message contains threats and harassment.",
                createdForConnect: validIds.id2,
            },
        },
        commentReport: {
            create: {
                id: validIds.id1,
                createdForType: "Comment",
                language: "en",
                reason: "Misinformation",
                details: "Comment spreads false information.",
                createdForConnect: validIds.id3,
            },
        },
        issueReport: {
            create: {
                id: validIds.id1,
                createdForType: "Issue",
                language: "en",
                reason: "Duplicate issue",
                details: "This issue is a duplicate of an existing one.",
                createdForConnect: validIds.id4,
            },
        },
        resourceVersionReport: {
            create: {
                id: validIds.id1,
                createdForType: "ResourceVersion",
                language: "en",
                reason: "Copyright violation",
                details: "This resource version contains copyrighted material.",
                createdForConnect: validIds.id5,
            },
        },
        tagReport: {
            create: {
                id: validIds.id1,
                createdForType: "Tag",
                language: "en",
                reason: "Inappropriate tag",
                details: "Tag contains offensive language.",
                createdForConnect: validIds.id6,
            },
        },
        teamReport: {
            create: {
                id: validIds.id1,
                createdForType: "Team",
                language: "en",
                reason: "Fraudulent activity",
                details: "Team is involved in fraudulent activities.",
                createdForConnect: validIds.id2,
            },
        },
        userReport: {
            create: {
                id: validIds.id1,
                createdForType: "User",
                language: "en",
                reason: "Spam account",
                details: "User account is posting spam content.",
                createdForConnect: validIds.id3,
            },
        },
        updateOnlyId: {
            update: {
                id: validIds.id1,
                // Only required field
            },
        },
        updateAllFields: {
            update: {
                id: validIds.id1,
                details: "Comprehensive update with all optional fields modified.",
                language: "fr",
                reason: "Updated violation type",
            },
        },
        differentLanguages: {
            create: {
                id: validIds.id1,
                createdForType: "User",
                language: "es",
                reason: "Contenido inapropiado",
                details: "Este usuario está publicando contenido inapropiado.",
                createdForConnect: validIds.id2,
            },
        },
        maxLengthReason: {
            create: {
                id: validIds.id1,
                createdForType: "User",
                language: "en",
                reason: "x".repeat(128), // Exactly max length
                createdForConnect: validIds.id2,
            },
        },
        minLengthReason: {
            create: {
                id: validIds.id1,
                createdForType: "User",
                language: "en",
                reason: "x", // Exactly min length (1)
                createdForConnect: validIds.id2,
            },
        },
        maxLengthDetails: {
            create: {
                id: validIds.id1,
                createdForType: "User",
                language: "en",
                reason: "Detailed violation",
                details: "x".repeat(8192), // Exactly max length
                createdForConnect: validIds.id2,
            },
        },
        differentIds: {
            create: {
                id: validIds.id6,
                createdForType: "Comment",
                language: "en",
                reason: "Different ID test",
                createdForConnect: validIds.id5,
            },
        },
        longDetailedReport: {
            create: {
                id: validIds.id1,
                createdForType: "ResourceVersion",
                language: "en",
                reason: "Multiple policy violations",
                details: "This resource version violates multiple community guidelines including copyright infringement, inappropriate content, spam, and misinformation. The violations are severe and require immediate attention from the moderation team. Additional details include specific examples of the violations and evidence supporting the claims.",
                createdForConnect: validIds.id2,
            },
        },
    },
};

// Custom factory that always generates valid IDs and required fields
const customizers = {
    create: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
        createdForType: base.createdForType || "User",
        language: base.language || "en",
        reason: base.reason || "Test reason",
        createdForConnect: base.createdForConnect || validIds.id2,
    }),
    update: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export a factory for creating test data programmatically
export const reportTestDataFactory = new TestDataFactory(reportFixtures, customizers);