import { type ModelTestFixtures, TestDataFactory } from "../../../validation/models/__test/validationTestUtils.js";

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    id4: "123456789012345681",
    id5: "123456789012345682",
    id6: "123456789012345683",
};

// Shared reportResponse test fixtures
export const reportResponseFixtures: ModelTestFixtures = {
    minimal: {
        create: {
            id: validIds.id1,
            actionSuggested: "NonIssue",
            reportConnect: validIds.id2,
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id1,
            actionSuggested: "HideUntilFixed",
            details: "After reviewing the reported content, we have determined that it violates community guidelines regarding spam and promotional content. The recommended action is to hide the content until the issues are addressed by the content creator.",
            language: "en",
            reportConnect: validIds.id3,
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 123,
            unknownField3: true,
        },
        update: {
            id: validIds.id1,
            actionSuggested: "Delete",
            details: "Updated response after further review. The content should be permanently deleted due to severe policy violations.",
            language: "es",
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 456,
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing required id, actionSuggested, and reportConnect
                details: "Incomplete response",
                language: "en",
            },
            update: {
                // Missing required id
                details: "Updated details",
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                actionSuggested: "InvalidAction", // Invalid enum value
                details: 456, // Should be string
                language: 789, // Should be string
                reportConnect: 101112, // Should be string
            },
            update: {
                id: validIds.id1,
                actionSuggested: "InvalidAction", // Invalid enum value
                details: 123, // Should be string
                language: 456, // Should be string
            },
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                actionSuggested: "NonIssue",
                reportConnect: validIds.id2,
            },
            update: {
                id: "invalid-id",
            },
        },
        invalidActionSuggested: {
            create: {
                id: validIds.id1,
                actionSuggested: "UnknownAction", // Not a valid enum value
                reportConnect: validIds.id2,
            },
        },
        missingReport: {
            create: {
                id: validIds.id1,
                actionSuggested: "NonIssue",
                // Missing required reportConnect
            },
        },
        invalidReportConnect: {
            create: {
                id: validIds.id1,
                actionSuggested: "NonIssue",
                reportConnect: "invalid-report-id",
            },
        },
        longDetails: {
            create: {
                id: validIds.id1,
                actionSuggested: "NonIssue",
                details: "x".repeat(8193), // Exceeds max length
                reportConnect: validIds.id2,
            },
        },
    },
    edgeCases: {
        withoutDetails: {
            create: {
                id: validIds.id1,
                actionSuggested: "FalseReport",
                reportConnect: validIds.id4,
                // No details field
            },
        },
        withoutLanguage: {
            create: {
                id: validIds.id1,
                actionSuggested: "NonIssue",
                details: "Response without language specification.",
                reportConnect: validIds.id2,
                // No language field
            },
        },
        deleteAction: {
            create: {
                id: validIds.id1,
                actionSuggested: "Delete",
                details: "Content should be deleted due to severe violations.",
                language: "en",
                reportConnect: validIds.id2,
            },
        },
        falseReportAction: {
            create: {
                id: validIds.id1,
                actionSuggested: "FalseReport",
                details: "This report appears to be false or malicious in nature.",
                language: "en",
                reportConnect: validIds.id3,
            },
        },
        hideUntilFixedAction: {
            create: {
                id: validIds.id1,
                actionSuggested: "HideUntilFixed",
                details: "Content needs to be hidden until the reported issues are resolved.",
                language: "en",
                reportConnect: validIds.id4,
            },
        },
        nonIssueAction: {
            create: {
                id: validIds.id1,
                actionSuggested: "NonIssue",
                details: "After review, this does not violate any community guidelines.",
                language: "en",
                reportConnect: validIds.id5,
            },
        },
        suspendUserAction: {
            create: {
                id: validIds.id1,
                actionSuggested: "SuspendUser",
                details: "User account should be suspended due to repeated violations.",
                language: "en",
                reportConnect: validIds.id6,
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
                actionSuggested: "Delete",
                details: "Comprehensive update with all optional fields modified.",
                language: "fr",
            },
        },
        differentLanguages: {
            create: {
                id: validIds.id1,
                actionSuggested: "NonIssue",
                details: "Esta respuesta está en español para probar la funcionalidad de idiomas.",
                language: "es",
                reportConnect: validIds.id2,
            },
        },
        maxLengthDetails: {
            create: {
                id: validIds.id1,
                actionSuggested: "HideUntilFixed",
                details: "x".repeat(8192), // Exactly max length
                language: "en",
                reportConnect: validIds.id2,
            },
        },
        differentIds: {
            create: {
                id: validIds.id6,
                actionSuggested: "SuspendUser",
                reportConnect: validIds.id5,
            },
        },
        longDetailedResponse: {
            create: {
                id: validIds.id1,
                actionSuggested: "Delete",
                details: "After thorough investigation by our moderation team, we have determined that the reported content violates multiple community guidelines including harassment, spam, and misinformation. The content has been reviewed by multiple moderators and the decision is unanimous. The recommended action is permanent deletion of the content along with appropriate user account penalties. This decision is final and based on clear evidence of policy violations.",
                language: "en",
                reportConnect: validIds.id2,
            },
        },
        updateActionSuggested: {
            update: {
                id: validIds.id1,
                actionSuggested: "SuspendUser",
            },
        },
        updateDetails: {
            update: {
                id: validIds.id1,
                details: "Updated response details after additional review.",
            },
        },
        updateLanguage: {
            update: {
                id: validIds.id1,
                language: "fr",
            },
        },
        multipleFieldsUpdate: {
            update: {
                id: validIds.id1,
                actionSuggested: "HideUntilFixed",
                details: "Response updated with new recommendation after appeal review.",
                language: "de",
            },
        },
    },
};

// Custom factory that always generates valid IDs and required fields
const customizers = {
    create: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
        actionSuggested: base.actionSuggested || "NonIssue",
        reportConnect: base.reportConnect || validIds.id2,
    }),
    update: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export a factory for creating test data programmatically
export const reportResponseTestDataFactory = new TestDataFactory(reportResponseFixtures, customizers);
