import {
    endpointsReport,
    ReportFor,
    reportValidation,
    type ReportCreateInput,
    type ReportShape,
    type ReportUpdateInput,
    type Session,
} from "@vrooli/shared";
import { reportInitialValues, transformReportValues } from "../../../views/objects/report/ReportUpsert.js";
import { createUIFormTestFactory, type UIFormTestConfig } from "./UIFormTestFactory.js";

/**
 * Configuration for Report form testing with data-driven test scenarios
 */
const reportFormTestConfig: UIFormTestConfig<ReportShape, ReportShape, ReportCreateInput, ReportUpdateInput, ReportShape> = {
    // Form metadata
    objectType: "Report",
    formFixtures: {
        minimal: {
            __typename: "Report" as const,
            id: "report_minimal",
            details: null,
            language: "en",
            reason: "Spam",
            createdFor: { __typename: ReportFor.User, id: "user_123" },
            otherReason: null,
        },
        complete: {
            __typename: "Report" as const,
            id: "report_complete",
            details: "This user has been posting spam content repeatedly, including irrelevant links and promotional material that violates community guidelines.",
            language: "en",
            reason: "Spam",
            createdFor: { __typename: ReportFor.Team, id: "team_456" },
            otherReason: null,
        },
        invalid: {
            __typename: "Report" as const,
            id: "report_invalid",
            details: "A".repeat(5001), // Invalid: exceeds character limit
            language: "en",
            reason: "Other",
            createdFor: { __typename: ReportFor.User, id: "" }, // Invalid: empty ID
            otherReason: null, // Invalid: Other reason requires otherReason field
        },
        edgeCase: {
            __typename: "Report" as const,
            id: "report_edge",
            details: "A".repeat(5000), // Edge case: at character limit
            language: "en",
            reason: "Other",
            createdFor: { __typename: ReportFor.Comment, id: "comment_789" },
            otherReason: "Custom violation: Content contains misleading information and fake references to authoritative sources.",
        },
    },

    // Validation schemas from shared package
    validation: reportValidation,

    // API endpoints from shared package
    endpoints: {
        create: endpointsReport.createOne,
        update: endpointsReport.updateOne,
    },

    // Transform functions - form already uses ReportShape, so no transformation needed
    formToShape: (formData: ReportShape) => formData,

    transformFunction: (shape: ReportShape, existing: ReportShape, isCreate: boolean) => {
        const result = transformReportValues(shape, existing, isCreate);
        if (!result) {
            throw new Error("Transform function returned undefined");
        }
        return result;
    },

    initialValuesFunction: (session?: Session, existing?: Partial<ReportShape>): ReportShape => {
        return reportInitialValues(session, existing || {}, existing?.createdFor || { __typename: ReportFor.User, id: "default_id" });
    },

    // DATA-DRIVEN TEST SCENARIOS - replaces all custom wrapper methods
    testScenarios: {
        reasonValidation: {
            description: "Test different report reason types",
            testCases: [
                {
                    name: "Spam reason",
                    data: { reason: "Spam" },
                    shouldPass: true,
                },
                {
                    name: "Harassment reason",
                    data: { reason: "Harassment" },
                    shouldPass: true,
                },
                {
                    name: "Inappropriate content",
                    data: { reason: "InappropriateContent" },
                    shouldPass: true,
                },
                {
                    name: "Copyright violation",
                    data: { reason: "Copyright" },
                    shouldPass: true,
                },
                {
                    name: "Other reason with explanation",
                    data: {
                        reason: "Other",
                        otherReason: "Custom violation description",
                    },
                    shouldPass: true,
                },
                {
                    name: "Other reason without explanation",
                    data: {
                        reason: "Other",
                        otherReason: null,
                    },
                    shouldPass: false,
                },
            ],
        },

        detailsValidation: {
            description: "Test report details validation and length limits",
            testCases: [
                {
                    name: "Valid details",
                    field: "details",
                    value: "This content violates community guidelines.",
                    shouldPass: true,
                },
                {
                    name: "Empty details",
                    field: "details",
                    value: null,
                    shouldPass: true, // Details are optional
                },
                {
                    name: "Long valid details",
                    field: "details",
                    value: "A".repeat(1000),
                    shouldPass: true,
                },
                {
                    name: "At character limit",
                    field: "details",
                    value: "A".repeat(5000),
                    shouldPass: true,
                },
                {
                    name: "Over character limit",
                    field: "details",
                    value: "A".repeat(5001),
                    shouldPass: false,
                },
            ],
        },

        targetValidation: {
            description: "Test reporting different object types",
            testCases: [
                {
                    name: "Report user",
                    data: { createdFor: { __typename: ReportFor.User, id: "user_123" } },
                    shouldPass: true,
                },
                {
                    name: "Report team",
                    data: { createdFor: { __typename: ReportFor.Team, id: "team_456" } },
                    shouldPass: true,
                },
                {
                    name: "Report comment",
                    data: { createdFor: { __typename: ReportFor.Comment, id: "comment_789" } },
                    shouldPass: true,
                },
                {
                    name: "Report resource version",
                    data: { createdFor: { __typename: ReportFor.ResourceVersion, id: "resourceversion_101" } },
                    shouldPass: true,
                },
                {
                    name: "Missing target ID",
                    data: { createdFor: { __typename: ReportFor.User, id: "" } },
                    shouldPass: false,
                },
            ],
        },

        otherReasonValidation: {
            description: "Test other reason field validation",
            testCases: [
                {
                    name: "Other reason with valid explanation",
                    field: "otherReason",
                    value: "Custom policy violation that doesn't fit standard categories",
                    shouldPass: true,
                },
                {
                    name: "Empty other reason when required",
                    data: {
                        reason: "Other",
                        otherReason: "",
                    },
                    shouldPass: false,
                },
                {
                    name: "Very long other reason",
                    field: "otherReason",
                    value: "A".repeat(1000),
                    shouldPass: true,
                },
            ],
        },

        reportWorkflow: {
            description: "Test different report workflow scenarios",
            testCases: [
                {
                    name: "Quick spam report",
                    data: {
                        reason: "Spam",
                        details: null,
                        createdFor: { __typename: ReportFor.User, id: "spammer_123" },
                    },
                    shouldPass: true,
                },
                {
                    name: "Detailed harassment report",
                    data: {
                        reason: "Harassment",
                        details: "Multiple instances of targeted harassment including personal attacks and threats.",
                        createdFor: { __typename: ReportFor.User, id: "harasser_456" },
                    },
                    shouldPass: true,
                },
                {
                    name: "Copyright claim with evidence",
                    data: {
                        reason: "Copyright",
                        details: "This content infringes on copyrighted material owned by our organization. Original work can be verified at example.com/original.",
                        createdFor: { __typename: ReportFor.ResourceVersion, id: "infringing_resource_789" },
                    },
                    shouldPass: true,
                },
                {
                    name: "Custom violation report",
                    data: {
                        reason: "Other",
                        otherReason: "Misleading information",
                        details: "Content contains factually incorrect information that could mislead users.",
                        createdFor: { __typename: ReportFor.Comment, id: "misleading_comment_101" },
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
export const reportFormTestFactory = createUIFormTestFactory(reportFormTestConfig);

/**
 * Type exports for use in other test files
 */
export { reportFormTestConfig };
