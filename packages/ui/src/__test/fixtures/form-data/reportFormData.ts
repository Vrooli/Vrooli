import { ReportFor, DUMMY_ID, type ReportShape } from "@vrooli/shared";

/**
 * Form data fixtures for report creation and editing
 * These represent data as it appears in form state before submission
 */

/**
 * Minimal report form input - just required fields
 */
export const minimalReportFormInput: Partial<ReportShape> = {
    createdFor: {
        __typename: ReportFor.User,
        id: "123456789012345678",
    },
    language: "en",
    reason: "Spam",
};

/**
 * Complete report form input with all fields
 */
export const completeReportFormInput: Partial<ReportShape> = {
    createdFor: {
        __typename: ReportFor.Comment,
        id: "234567890123456789",
    },
    language: "en",
    reason: "Harassment",
    details: "This comment contains targeted harassment against another user. The language is threatening and violates community guidelines.",
    otherReason: null,
};

/**
 * Report with custom reason
 */
export const customReasonReportFormInput: Partial<ReportShape> = {
    createdFor: {
        __typename: ReportFor.ResourceVersion,
        id: "345678901234567890",
    },
    language: "en",
    reason: "Other",
    otherReason: "Contains malicious code that could harm users",
    details: "The resource contains JavaScript code that attempts to access sensitive browser APIs without user consent.",
};

/**
 * Report for different target types
 */
export const reportForChatMessageFormInput: Partial<ReportShape> = {
    createdFor: {
        __typename: ReportFor.ChatMessage,
        id: "456789012345678901",
    },
    language: "en",
    reason: "Inappropriate content",
    details: "Message contains explicit content that is not suitable for this platform.",
};

export const reportForCommentFormInput: Partial<ReportShape> = {
    createdFor: {
        __typename: ReportFor.Comment,
        id: "567890123456789012",
    },
    language: "en",
    reason: "Misinformation",
    details: "Comment spreads false information about health topics.",
};

export const reportForIssueFormInput: Partial<ReportShape> = {
    createdFor: {
        __typename: ReportFor.Issue,
        id: "678901234567890123",
    },
    language: "en",
    reason: "Duplicate",
    details: "This issue is a duplicate of issue #123.",
};

export const reportForResourceVersionFormInput: Partial<ReportShape> = {
    createdFor: {
        __typename: ReportFor.ResourceVersion,
        id: "789012345678901234",
    },
    language: "en",
    reason: "Copyright violation",
    details: "This resource contains copyrighted material without permission.",
};

export const reportForTagFormInput: Partial<ReportShape> = {
    createdFor: {
        __typename: ReportFor.Tag,
        id: "890123456789012345",
    },
    language: "en",
    reason: "Offensive language",
    details: "Tag contains inappropriate language.",
};

export const reportForTeamFormInput: Partial<ReportShape> = {
    createdFor: {
        __typename: ReportFor.Team,
        id: "901234567890123456",
    },
    language: "en",
    reason: "Fraudulent activity",
    details: "Team is impersonating a legitimate organization and requesting personal information.",
};

export const reportForUserFormInput: Partial<ReportShape> = {
    createdFor: {
        __typename: ReportFor.User,
        id: "012345678901234567",
    },
    language: "en",
    reason: "Spam account",
    details: "User is sending automated spam messages to multiple users.",
};

/**
 * Multi-language report form input
 */
export const spanishReportFormInput: Partial<ReportShape> = {
    createdFor: {
        __typename: ReportFor.Comment,
        id: "123456789012345678",
    },
    language: "es",
    reason: "Contenido inapropiado",
    details: "Este comentario contiene lenguaje ofensivo y no es apropiado para esta plataforma.",
};

export const frenchReportFormInput: Partial<ReportShape> = {
    createdFor: {
        __typename: ReportFor.User,
        id: "234567890123456789",
    },
    language: "fr",
    reason: "Harcèlement",
    details: "Cet utilisateur envoie des messages harcelants répétés.",
};

/**
 * Detailed report with extensive information
 */
export const detailedReportFormInput: Partial<ReportShape> = {
    createdFor: {
        __typename: ReportFor.Team,
        id: "345678901234567890",
    },
    language: "en",
    reason: "Multiple violations",
    details: `This team has been involved in multiple community guideline violations:

1. Spam: Sending unsolicited promotional messages to users
2. Harassment: Team members targeting specific users with negative comments
3. Misinformation: Sharing false information about platform features
4. Copyright: Using copyrighted logos without permission

Evidence has been documented and screenshots are available upon request.`,
};

/**
 * Edited report form input
 */
export const editedReportFormInput: Partial<ReportShape> = {
    id: "456789012345678901",
    createdFor: {
        __typename: ReportFor.ChatMessage,
        id: "567890123456789012",
    },
    language: "en",
    reason: "Updated: Threats of violence",
    details: "UPDATE: Upon further review, this message contains explicit threats of violence against another user.",
};

/**
 * Invalid form inputs for testing validation
 */
export const invalidReportFormInputs = {
    missingCreatedFor: {
        // @ts-expect-error - Testing missing required createdFor
        createdFor: undefined,
        language: "en",
        reason: "Test reason",
    },
    missingLanguage: {
        createdFor: {
            __typename: ReportFor.User as const,
            id: "123456789012345678",
        },
        // @ts-expect-error - Testing missing required language
        language: undefined,
        reason: "Test reason",
    },
    missingReason: {
        createdFor: {
            __typename: ReportFor.User as const,
            id: "123456789012345678",
        },
        language: "en",
        // @ts-expect-error - Testing missing required reason
        reason: undefined,
    },
    emptyReason: {
        createdFor: {
            __typename: ReportFor.User as const,
            id: "123456789012345678",
        },
        language: "en",
        reason: "",
    },
    invalidCreatedForType: {
        createdFor: {
            // @ts-expect-error - Testing invalid type
            __typename: "InvalidType",
            id: "123456789012345678",
        },
        language: "en",
        reason: "Test reason",
    },
    invalidCreatedForId: {
        createdFor: {
            __typename: ReportFor.User as const,
            id: "invalid-id", // Not a valid snowflake ID
        },
        language: "en",
        reason: "Test reason",
    },
    invalidLanguage: {
        createdFor: {
            __typename: ReportFor.User as const,
            id: "123456789012345678",
        },
        language: "xyz", // Invalid language code
        reason: "Test reason",
    },
    reasonTooLong: {
        createdFor: {
            __typename: ReportFor.User as const,
            id: "123456789012345678",
        },
        language: "en",
        reason: "x".repeat(129), // Over the 128 character limit
    },
    detailsTooLong: {
        createdFor: {
            __typename: ReportFor.User as const,
            id: "123456789012345678",
        },
        language: "en",
        reason: "Test reason",
        details: "x".repeat(8193), // Over the 8192 character limit
    },
    otherReasonTooLong: {
        createdFor: {
            __typename: ReportFor.User as const,
            id: "123456789012345678",
        },
        language: "en",
        reason: "Other",
        otherReason: "x".repeat(129), // Over the 128 character limit
    },
    whitespaceOnlyReason: {
        createdFor: {
            __typename: ReportFor.User as const,
            id: "123456789012345678",
        },
        language: "en",
        reason: "   \n\t   ",
    },
};

/**
 * Helper function to transform form data to API input
 */
export const transformReportFormToApiInput = (formData: Partial<ReportShape>): ReportShape => {
    if (!formData.createdFor) {
        throw new Error("createdFor is required for report creation");
    }
    if (!formData.language) {
        throw new Error("language is required for report creation");
    }
    if (!formData.reason) {
        throw new Error("reason is required for report creation");
    }

    return {
        __typename: "Report",
        id: formData.id || DUMMY_ID,
        createdFor: formData.createdFor,
        language: formData.language,
        reason: formData.reason,
        details: formData.details || null,
        otherReason: formData.otherReason || null,
    };
};

/**
 * Helper function to validate report content
 */
export const validateReportContent = (report: Partial<ReportShape>): string | null => {
    if (!report.createdFor) {
        return "Report target is required";
    }
    if (!report.language) {
        return "Language is required";
    }
    if (!report.reason || !report.reason.trim()) {
        return "Reason is required";
    }
    if (report.reason.length > 128) {
        return "Reason is too long (max 128 characters)";
    }
    if (report.details && report.details.length > 8192) {
        return "Details are too long (max 8192 characters)";
    }
    if (report.otherReason && report.otherReason.length > 128) {
        return "Other reason is too long (max 128 characters)";
    }
    if (report.reason === "Other" && !report.otherReason?.trim()) {
        return "Please specify a reason when selecting 'Other'";
    }

    return null;
};

/**
 * Helper function to get report severity
 */
export const getReportSeverity = (reason: string): "critical" | "high" | "medium" | "low" => {
    const lowerReason = reason.toLowerCase();
    if (lowerReason.includes("violence") || lowerReason.includes("threat")) return "critical";
    if (lowerReason.includes("harassment") || lowerReason.includes("fraud")) return "high";
    if (lowerReason.includes("spam") || lowerReason.includes("misinformation")) return "medium";
    return "low";
};

/**
 * Helper function to check if report needs immediate attention
 */
export const needsImmediateAttention = (report: Partial<ReportShape>): boolean => {
    const severity = getReportSeverity(report.reason || "");
    return severity === "critical" || severity === "high";
};

/**
 * Mock form states for testing
 */
export const reportFormStates = {
    pristine: {
        values: {
            createdFor: null,
            language: "en",
            reason: "",
            details: "",
            otherReason: "",
        },
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    typing: {
        values: {
            createdFor: { __typename: ReportFor.User, id: "123456789012345678" },
            language: "en",
            reason: "Spam",
            details: "This user is sending sp",
            otherReason: "",
        },
        errors: {},
        touched: { reason: true, details: true },
        isValid: true,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            createdFor: { __typename: ReportFor.User, id: "123456789012345678" },
            language: "en",
            reason: "",
            details: "",
            otherReason: "",
        },
        errors: { reason: "Reason is required" },
        touched: { reason: true },
        isValid: false,
        isSubmitting: false,
    },
    submitting: {
        values: {
            createdFor: { __typename: ReportFor.Comment, id: "234567890123456789" },
            language: "en",
            reason: "Harassment",
            details: "This comment contains harassment",
            otherReason: "",
        },
        errors: {},
        touched: { reason: true, details: true },
        isValid: true,
        isSubmitting: true,
    },
    submitted: {
        values: {
            createdFor: null,
            language: "en",
            reason: "",
            details: "",
            otherReason: "",
        }, // Reset after successful submit
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
};

/**
 * Report form scenarios for different contexts
 */
export const reportFormScenarios = {
    spamReport: {
        __typename: "Report" as const,
        createdFor: { __typename: ReportFor.User as const, id: "123456789012345678" },
        language: "en",
        reason: "Spam",
        details: "User is posting promotional content repeatedly",
        otherReason: null,
    },
    harassmentReport: {
        __typename: "Report" as const,
        createdFor: { __typename: ReportFor.ChatMessage as const, id: "234567890123456789" },
        language: "en",
        reason: "Harassment",
        details: "Message contains personal attacks and threats",
        otherReason: null,
    },
    copyrightReport: {
        __typename: "Report" as const,
        createdFor: { __typename: ReportFor.ResourceVersion as const, id: "345678901234567890" },
        language: "en",
        reason: "Copyright violation",
        details: "This resource contains my copyrighted work without permission. Original work can be found at...",
        otherReason: null,
    },
    customReport: {
        __typename: "Report" as const,
        createdFor: { __typename: ReportFor.Team as const, id: "456789012345678901" },
        language: "en",
        reason: "Other",
        details: "Team is using platform for illegal activities",
        otherReason: "Illegal activity coordination",
    },
    urgentReport: {
        __typename: "Report" as const,
        createdFor: { __typename: ReportFor.Comment as const, id: "567890123456789012" },
        language: "en",
        reason: "Threats of violence",
        details: "Comment contains explicit threats against another user. Immediate action required.",
        otherReason: null,
    },
};

/**
 * Common report reasons by target type
 */
export const commonReportReasons = {
    [ReportFor.ChatMessage]: ["Spam", "Harassment", "Inappropriate content", "Threats"],
    [ReportFor.Comment]: ["Spam", "Harassment", "Misinformation", "Offensive language"],
    [ReportFor.Issue]: ["Duplicate", "Invalid", "Spam", "Not an issue"],
    [ReportFor.ResourceVersion]: ["Copyright violation", "Malicious content", "Inappropriate", "Misinformation"],
    [ReportFor.Tag]: ["Offensive language", "Inappropriate", "Misleading", "Spam"],
    [ReportFor.Team]: ["Fraudulent activity", "Impersonation", "Spam", "Harassment"],
    [ReportFor.User]: ["Spam account", "Harassment", "Impersonation", "Inappropriate content"],
};