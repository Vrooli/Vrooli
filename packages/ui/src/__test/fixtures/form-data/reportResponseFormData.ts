import { ReportSuggestedAction, DUMMY_ID, type ReportResponseShape } from "@vrooli/shared";

/**
 * Form data fixtures for report response creation and editing
 * These represent data as it appears in form state before submission
 */

/**
 * Minimal report response form input - just required fields
 */
export const minimalReportResponseFormInput: Partial<ReportResponseShape> = {
    actionSuggested: ReportSuggestedAction.NonIssue,
    reportConnect: "123456789012345678",
};

/**
 * Complete report response form input with all fields
 */
export const completeReportResponseFormInput: Partial<ReportResponseShape> = {
    actionSuggested: ReportSuggestedAction.HideUntilFixed,
    details: "After reviewing the reported content, we have determined that it violates community guidelines regarding spam and promotional content. The recommended action is to hide the content until the issues are addressed by the content creator.",
    language: "en",
    reportConnect: "234567890123456789",
};

/**
 * Report response form inputs for different action types
 */
export const deleteActionFormInput: Partial<ReportResponseShape> = {
    actionSuggested: ReportSuggestedAction.Delete,
    details: "Content confirmed as spam and should be permanently deleted due to severe policy violations.",
    language: "en",
    reportConnect: "345678901234567890",
};

export const falseReportActionFormInput: Partial<ReportResponseShape> = {
    actionSuggested: ReportSuggestedAction.FalseReport,
    details: "After review, the reported content does not violate community guidelines. This appears to be a disagreement rather than a policy violation.",
    language: "en",
    reportConnect: "456789012345678901",
};

export const suspendUserActionFormInput: Partial<ReportResponseShape> = {
    actionSuggested: ReportSuggestedAction.SuspendUser,
    details: "Reviewed the user's activity. Confirmed pattern of harassment and policy violations. Recommending user suspension.",
    language: "en",
    reportConnect: "567890123456789012",
};

export const nonIssueActionFormInput: Partial<ReportResponseShape> = {
    actionSuggested: ReportSuggestedAction.NonIssue,
    details: "After thorough review, this does not violate any community guidelines. No action is required.",
    language: "en",
    reportConnect: "678901234567890123",
};

/**
 * Multi-language report response form inputs
 */
export const spanishReportResponseFormInput: Partial<ReportResponseShape> = {
    actionSuggested: ReportSuggestedAction.HideUntilFixed,
    details: "Después de revisar el contenido reportado, hemos determinado que viola las pautas de la comunidad. Se recomienda ocultar hasta que se resuelvan los problemas.",
    language: "es",
    reportConnect: "789012345678901234",
};

export const frenchReportResponseFormInput: Partial<ReportResponseShape> = {
    actionSuggested: ReportSuggestedAction.Delete,
    details: "Après examen du contenu signalé, nous avons confirmé qu'il viole les directives communautaires. Suppression recommandée.",
    language: "fr",
    reportConnect: "890123456789012345",
};

/**
 * Brief and detailed response variations
 */
export const briefResponseFormInput: Partial<ReportResponseShape> = {
    actionSuggested: ReportSuggestedAction.NonIssue,
    details: "No violation found.",
    language: "en",
    reportConnect: "901234567890123456",
};

export const detailedResponseFormInput: Partial<ReportResponseShape> = {
    actionSuggested: ReportSuggestedAction.Delete,
    details: `After thorough investigation by our moderation team, we have determined that the reported content violates multiple community guidelines including:

1. Harassment: Contains personal attacks and threatening language
2. Spam: Promotional content posted repeatedly across multiple locations
3. Misinformation: Shares false information about platform features
4. Copyright: Uses copyrighted material without permission

The content has been reviewed by multiple moderators and the decision is unanimous. The recommended action is permanent deletion of the content along with appropriate user account penalties. This decision is final and based on clear evidence of policy violations.

Evidence has been documented and screenshots are available upon request.`,
    language: "en",
    reportConnect: "012345678901234567",
};

/**
 * Response for different severity levels
 */
export const criticalViolationFormInput: Partial<ReportResponseShape> = {
    actionSuggested: ReportSuggestedAction.SuspendUser,
    details: "CRITICAL: Content contains explicit threats of violence against another user. Immediate suspension required for safety.",
    language: "en",
    reportConnect: "123456789012345679",
};

export const minorViolationFormInput: Partial<ReportResponseShape> = {
    actionSuggested: ReportSuggestedAction.HideUntilFixed,
    details: "Minor policy violation detected. Content should be hidden until author makes necessary corrections.",
    language: "en",
    reportConnect: "123456789012345680",
};

/**
 * Update-specific form inputs (with id for editing existing responses)
 */
export const updateResponseFormInput: Partial<ReportResponseShape> = {
    id: "response_123456789012345678",
    actionSuggested: ReportSuggestedAction.Delete,
    details: "Updated response after further review and additional evidence. Escalating to deletion.",
    language: "en",
};

export const updateActionOnlyFormInput: Partial<ReportResponseShape> = {
    id: "response_234567890123456789",
    actionSuggested: ReportSuggestedAction.SuspendUser,
};

export const updateDetailsOnlyFormInput: Partial<ReportResponseShape> = {
    id: "response_345678901234567890",
    details: "Updated details after appeal review: Additional context shows pattern of behavior.",
};

/**
 * Response without optional fields
 */
export const actionOnlyFormInput: Partial<ReportResponseShape> = {
    actionSuggested: ReportSuggestedAction.FalseReport,
    reportConnect: "456789012345678902",
};

export const withLanguageOnlyFormInput: Partial<ReportResponseShape> = {
    actionSuggested: ReportSuggestedAction.NonIssue,
    language: "de",
    reportConnect: "567890123456789013",
};

/**
 * Invalid form inputs for testing validation
 */
export const invalidReportResponseFormInputs = {
    missingActionSuggested: {
        // @ts-expect-error - Testing missing required actionSuggested
        actionSuggested: undefined,
        details: "Missing action suggested",
        language: "en",
        reportConnect: "123456789012345678",
    },
    missingReportConnect: {
        actionSuggested: ReportSuggestedAction.NonIssue,
        details: "Missing report connection",
        language: "en",
        // @ts-expect-error - Testing missing required reportConnect
        reportConnect: undefined,
    },
    emptyReportConnect: {
        actionSuggested: ReportSuggestedAction.NonIssue,
        details: "Empty report connection",
        language: "en",
        reportConnect: "",
    },
    invalidActionSuggested: {
        // @ts-expect-error - Testing invalid action type
        actionSuggested: "InvalidAction",
        details: "Invalid action type",
        language: "en",
        reportConnect: "123456789012345678",
    },
    invalidReportConnectId: {
        actionSuggested: ReportSuggestedAction.NonIssue,
        details: "Invalid report ID format",
        language: "en",
        reportConnect: "invalid-id", // Not a valid snowflake ID
    },
    invalidLanguage: {
        actionSuggested: ReportSuggestedAction.NonIssue,
        details: "Invalid language code",
        language: "xyz", // Invalid language code
        reportConnect: "123456789012345678",
    },
    detailsTooLong: {
        actionSuggested: ReportSuggestedAction.NonIssue,
        details: "x".repeat(8193), // Over the 8192 character limit
        language: "en",
        reportConnect: "123456789012345678",
    },
    whitespaceOnlyDetails: {
        actionSuggested: ReportSuggestedAction.NonIssue,
        details: "   \n\t   ",
        language: "en",
        reportConnect: "123456789012345678",
    },
    updateWithoutId: {
        // @ts-expect-error - Testing update without required id
        id: undefined,
        actionSuggested: ReportSuggestedAction.Delete,
        details: "Update without ID",
    },
    updateWithInvalidId: {
        id: "invalid-id", // Not a valid snowflake ID
        actionSuggested: ReportSuggestedAction.Delete,
        details: "Update with invalid ID",
    },
};

/**
 * Helper function to transform form data to API input
 */
export const transformReportResponseFormToApiInput = (formData: Partial<ReportResponseShape>): ReportResponseShape => {
    if (!formData.actionSuggested) {
        throw new Error("actionSuggested is required for report response");
    }
    if (!formData.reportConnect && !formData.id) {
        throw new Error("reportConnect is required for report response creation");
    }

    return {
        __typename: "ReportResponse",
        id: formData.id || DUMMY_ID,
        actionSuggested: formData.actionSuggested,
        details: formData.details || null,
        language: formData.language || null,
        reportConnect: formData.reportConnect,
    };
};

/**
 * Helper function to validate report response content
 */
export const validateReportResponseContent = (response: Partial<ReportResponseShape>): string | null => {
    if (!response.actionSuggested) {
        return "Action suggestion is required";
    }
    if (!response.reportConnect && !response.id) {
        return "Report connection is required for new responses";
    }
    if (response.details && response.details.length > 8192) {
        return "Details are too long (max 8192 characters)";
    }
    if (response.language && !/^[a-z]{2}(-[A-Z]{2})?$/.test(response.language)) {
        return "Invalid language code format";
    }
    if (response.reportConnect && !/^\d{18,19}$/.test(response.reportConnect)) {
        return "Invalid report ID format";
    }
    if (response.id && !/^\d{18,19}$/.test(response.id)) {
        return "Invalid response ID format";
    }

    return null;
};

/**
 * Helper function to get response severity level
 */
export const getResponseSeverity = (actionSuggested: ReportSuggestedAction): "critical" | "high" | "medium" | "low" => {
    switch (actionSuggested) {
        case ReportSuggestedAction.SuspendUser:
            return "critical";
        case ReportSuggestedAction.Delete:
            return "high";
        case ReportSuggestedAction.HideUntilFixed:
            return "medium";
        case ReportSuggestedAction.FalseReport:
        case ReportSuggestedAction.NonIssue:
        default:
            return "low";
    }
};

/**
 * Helper function to check if response requires immediate action
 */
export const requiresImmediateAction = (response: Partial<ReportResponseShape>): boolean => {
    if (!response.actionSuggested) return false;
    const severity = getResponseSeverity(response.actionSuggested);
    return severity === "critical" || severity === "high";
};

/**
 * Mock form states for testing
 */
export const reportResponseFormStates = {
    pristine: {
        values: {
            actionSuggested: null,
            details: "",
            language: "en",
            reportConnect: null,
        },
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    typing: {
        values: {
            actionSuggested: ReportSuggestedAction.HideUntilFixed,
            details: "After review, this content",
            language: "en",
            reportConnect: "123456789012345678",
        },
        errors: {},
        touched: { actionSuggested: true, details: true },
        isValid: true,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            actionSuggested: null,
            details: "",
            language: "en",
            reportConnect: "123456789012345678",
        },
        errors: { actionSuggested: "Action suggestion is required" },
        touched: { actionSuggested: true },
        isValid: false,
        isSubmitting: false,
    },
    submitting: {
        values: {
            actionSuggested: ReportSuggestedAction.Delete,
            details: "Content confirmed as violation and should be deleted.",
            language: "en",
            reportConnect: "123456789012345678",
        },
        errors: {},
        touched: { actionSuggested: true, details: true },
        isValid: true,
        isSubmitting: true,
    },
    submitted: {
        values: {
            actionSuggested: null,
            details: "",
            language: "en",
            reportConnect: null,
        }, // Reset after successful submit
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
};

/**
 * Report response form scenarios for different contexts
 */
export const reportResponseFormScenarios = {
    spamResponse: {
        __typename: "ReportResponse" as const,
        actionSuggested: ReportSuggestedAction.Delete,
        details: "Content confirmed as spam. Permanent deletion recommended.",
        language: "en",
        reportConnect: "123456789012345678",
    },
    harassmentResponse: {
        __typename: "ReportResponse" as const,
        actionSuggested: ReportSuggestedAction.SuspendUser,
        details: "Pattern of harassment confirmed. User suspension recommended for community safety.",
        language: "en",
        reportConnect: "234567890123456789",
    },
    falseAlarmResponse: {
        __typename: "ReportResponse" as const,
        actionSuggested: ReportSuggestedAction.FalseReport,
        details: "Report appears to be false or malicious in nature. No policy violation found.",
        language: "en",
        reportConnect: "345678901234567890",
    },
    minorViolationResponse: {
        __typename: "ReportResponse" as const,
        actionSuggested: ReportSuggestedAction.HideUntilFixed,
        details: "Minor guideline violation. Content should be hidden until corrected.",
        language: "en",
        reportConnect: "456789012345678901",
    },
    nonIssueResponse: {
        __typename: "ReportResponse" as const,
        actionSuggested: ReportSuggestedAction.NonIssue,
        details: "Content reviewed and found to be within community guidelines. No action needed.",
        language: "en",
        reportConnect: "567890123456789012",
    },
};

/**
 * Common response templates by action type
 */
export const responseTemplates = {
    [ReportSuggestedAction.Delete]: [
        "Content confirmed as spam and will be deleted.",
        "Severe policy violation. Content should be permanently removed.",
        "Multiple violations detected. Deletion recommended.",
    ],
    [ReportSuggestedAction.SuspendUser]: [
        "Pattern of harassment confirmed. User suspension recommended.",
        "Repeated violations. User account should be suspended.",
        "Critical safety issue. Immediate user suspension required.",
    ],
    [ReportSuggestedAction.HideUntilFixed]: [
        "Content violates guidelines. Hide until issues are addressed.",
        "Minor violation. Content should be hidden pending corrections.",
        "Partial compliance issue. Hide until fully resolved.",
    ],
    [ReportSuggestedAction.FalseReport]: [
        "Report appears to be false or malicious in nature.",
        "No policy violation found. Report seems unfounded.",
        "Content reviewed and found to be acceptable.",
    ],
    [ReportSuggestedAction.NonIssue]: [
        "Content is within community guidelines. No action needed.",
        "After review, no violation detected.",
        "Report does not indicate a policy violation.",
    ],
};