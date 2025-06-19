import { IssueFor, DUMMY_ID, type IssueShape, type IssueTranslationShape } from "@vrooli/shared";

/**
 * Form data fixtures for issue creation and editing
 * These represent data as it appears in form state before submission
 */

/**
 * Minimal issue form input - just name and target
 */
export const minimalIssueFormInput: Partial<IssueShape> = {
    issueFor: IssueFor.Resource,
    for: { id: "123456789012345678" }, // Valid snowflake ID
    translations: [{
        __typename: "IssueTranslation",
        id: DUMMY_ID,
        language: "en",
        name: "Simple bug report",
    }],
};

/**
 * Complete issue form input with all fields
 */
export const completeIssueFormInput: Partial<IssueShape> = {
    issueFor: IssueFor.Team,
    for: { id: "234567890123456789" }, // Valid snowflake ID
    translations: [{
        __typename: "IssueTranslation",
        id: DUMMY_ID,
        language: "en",
        name: "Critical performance issue in team dashboard",
        description: "The team dashboard takes over 30 seconds to load when there are more than 100 members. This is affecting our daily operations and needs urgent attention.",
    }],
};

/**
 * Issue for Resource
 */
export const issueForResourceFormInput: Partial<IssueShape> = {
    issueFor: IssueFor.Resource,
    for: { id: "345678901234567890" },
    translations: [{
        __typename: "IssueTranslation",
        id: DUMMY_ID,
        language: "en",
        name: "Broken link in documentation",
        description: "The API reference link on page 3 returns a 404 error",
    }],
};

/**
 * Issue for Team
 */
export const issueForTeamFormInput: Partial<IssueShape> = {
    issueFor: IssueFor.Team,
    for: { id: "456789012345678901" },
    translations: [{
        __typename: "IssueTranslation",
        id: DUMMY_ID,
        language: "en",
        name: "Unable to add new team members",
        description: "Getting an error when trying to invite new members to the team",
    }],
};

/**
 * Multi-language issue form input
 */
export const multiLanguageIssueFormInput: Partial<IssueShape> = {
    issueFor: IssueFor.Resource,
    for: { id: "567890123456789012" },
    translations: [
        {
            __typename: "IssueTranslation",
            id: DUMMY_ID,
            language: "en",
            name: "Feature request: Dark mode",
            description: "Please add a dark mode option to reduce eye strain during night work",
        },
        {
            __typename: "IssueTranslation",
            id: DUMMY_ID,
            language: "es",
            name: "Solicitud de función: Modo oscuro",
            description: "Por favor, agregue una opción de modo oscuro para reducir la fatiga ocular durante el trabajo nocturno",
        },
        {
            __typename: "IssueTranslation",
            id: DUMMY_ID,
            language: "fr",
            name: "Demande de fonctionnalité : Mode sombre",
            description: "Veuillez ajouter une option de mode sombre pour réduire la fatigue oculaire pendant le travail de nuit",
        },
    ],
};

/**
 * Detailed issue with markdown formatting
 */
export const detailedIssueFormInput: Partial<IssueShape> = {
    issueFor: IssueFor.Resource,
    for: { id: "678901234567890123" },
    translations: [{
        __typename: "IssueTranslation",
        id: DUMMY_ID,
        language: "en",
        name: "Data export fails for large datasets",
        description: `## Issue Description
The data export feature fails when attempting to export more than 10,000 records.

### Steps to Reproduce:
1. Navigate to the data export page
2. Select "All Records"
3. Choose CSV format
4. Click "Export"

### Expected Behavior:
Export should complete successfully and download the CSV file

### Actual Behavior:
Export times out after 60 seconds with error message: "Export failed"

### Environment:
- Browser: Chrome 120.0
- OS: Windows 11
- Account type: Premium

### Error Log:
\`\`\`
Error: Timeout exceeded (60000ms)
    at exportData (export.js:45)
    at processExport (handler.js:123)
\`\`\``,
    }],
};

/**
 * Security issue form input
 */
export const securityIssueFormInput: Partial<IssueShape> = {
    issueFor: IssueFor.Team,
    for: { id: "789012345678901234" },
    translations: [{
        __typename: "IssueTranslation",
        id: DUMMY_ID,
        language: "en",
        name: "[SECURITY] Potential XSS vulnerability in comment section",
        description: "Found a potential XSS vulnerability. Details sent via secure channel.",
    }],
};

/**
 * Edited issue form input
 */
export const editedIssueFormInput: Partial<IssueShape> = {
    id: "890123456789012345",
    issueFor: IssueFor.Resource,
    for: { id: "901234567890123456" },
    translations: [{
        __typename: "IssueTranslation",
        id: "012345678901234567",
        language: "en",
        name: "[UPDATED] Login button not responding",
        description: "UPDATE: This only happens on mobile devices. Desktop works fine.",
    }],
};

/**
 * Invalid form inputs for testing validation
 */
export const invalidIssueFormInputs = {
    emptyName: {
        issueFor: IssueFor.Resource,
        for: { id: "123456789012345678" },
        translations: [{
            __typename: "IssueTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            name: "",
        }],
    },
    whitespaceOnlyName: {
        issueFor: IssueFor.Resource,
        for: { id: "123456789012345678" },
        translations: [{
            __typename: "IssueTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            name: "   \n\t   ",
        }],
    },
    missingTranslations: {
        issueFor: IssueFor.Resource,
        for: { id: "123456789012345678" },
        // @ts-expect-error - Testing missing required translations
        translations: [],
    },
    noTranslations: {
        issueFor: IssueFor.Resource,
        for: { id: "123456789012345678" },
        // @ts-expect-error - Testing undefined translations
        translations: undefined,
    },
    missingName: {
        issueFor: IssueFor.Resource,
        for: { id: "123456789012345678" },
        translations: [{
            __typename: "IssueTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            // @ts-expect-error - Testing missing required name
            name: undefined,
        }],
    },
    missingLanguage: {
        issueFor: IssueFor.Resource,
        for: { id: "123456789012345678" },
        translations: [{
            __typename: "IssueTranslation" as const,
            id: DUMMY_ID,
            // @ts-expect-error - Testing missing required language
            language: undefined,
            name: "Valid name",
        }],
    },
    nameTooLong: {
        issueFor: IssueFor.Resource,
        for: { id: "123456789012345678" },
        translations: [{
            __typename: "IssueTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            name: "x".repeat(256), // Over typical name length limit
        }],
    },
    descriptionTooLong: {
        issueFor: IssueFor.Resource,
        for: { id: "123456789012345678" },
        translations: [{
            __typename: "IssueTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            name: "Valid name",
            description: "x".repeat(10001), // Over typical description length limit
        }],
    },
    invalidIssueFor: {
        // @ts-expect-error - Testing invalid issueFor
        issueFor: "InvalidType",
        for: { id: "123456789012345678" },
        translations: [{
            __typename: "IssueTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            name: "Valid name",
        }],
    },
    missingIssueFor: {
        // @ts-expect-error - Testing missing issueFor
        issueFor: undefined,
        for: { id: "123456789012345678" },
        translations: [{
            __typename: "IssueTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            name: "Valid name",
        }],
    },
    missingFor: {
        issueFor: IssueFor.Resource,
        // @ts-expect-error - Testing missing for
        for: undefined,
        translations: [{
            __typename: "IssueTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            name: "Valid name",
        }],
    },
    invalidForId: {
        issueFor: IssueFor.Resource,
        for: { id: "invalid-id" }, // Not a valid snowflake ID
        translations: [{
            __typename: "IssueTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            name: "Valid name",
        }],
    },
};

/**
 * Helper function to transform form data to API input
 */
export const transformIssueFormToApiInput = (formData: Partial<IssueShape>): IssueShape => {
    if (!formData.issueFor) {
        throw new Error("issueFor is required for issue creation");
    }
    if (!formData.for) {
        throw new Error("for is required for issue creation");
    }

    return {
        __typename: "Issue",
        id: formData.id || DUMMY_ID,
        issueFor: formData.issueFor,
        for: formData.for,
        translations: formData.translations || [],
    };
};

/**
 * Helper function to validate issue content
 */
export const validateIssueContent = (translations: IssueTranslationShape[]): string | null => {
    if (!translations || translations.length === 0) {
        return "At least one translation is required";
    }

    for (const translation of translations) {
        if (!translation.name || !translation.name.trim()) {
            return `Issue name cannot be empty for ${translation.language || "unknown"} language`;
        }
        if (translation.name.length > 255) {
            return `Issue name is too long for ${translation.language || "unknown"} language (max 255 characters)`;
        }
        if (translation.description && translation.description.length > 10000) {
            return `Issue description is too long for ${translation.language || "unknown"} language (max 10,000 characters)`;
        }
        if (!translation.language) {
            return "Language code is required for all translations";
        }
    }

    return null;
};

/**
 * Helper function to get issue severity from name
 */
export const getIssueSeverity = (name: string): "critical" | "high" | "medium" | "low" => {
    const lowerName = name.toLowerCase();
    if (lowerName.includes("[critical]") || lowerName.includes("urgent")) return "critical";
    if (lowerName.includes("[security]") || lowerName.includes("crash")) return "high";
    if (lowerName.includes("bug") || lowerName.includes("error")) return "medium";
    return "low";
};

/**
 * Mock form states for testing
 */
export const issueFormStates = {
    pristine: {
        values: { 
            issueFor: IssueFor.Resource,
            for: { id: "" },
            translations: [{ language: "en", name: "", description: "" }] 
        },
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    typing: {
        values: { 
            issueFor: IssueFor.Resource,
            for: { id: "123456789012345678" },
            translations: [{ language: "en", name: "Bug in the syst", description: "" }] 
        },
        errors: {},
        touched: { translations: [{ name: true }] },
        isValid: true,
        isSubmitting: false,
    },
    withErrors: {
        values: { 
            issueFor: IssueFor.Resource,
            for: { id: "123456789012345678" },
            translations: [{ language: "en", name: "", description: "" }] 
        },
        errors: { translations: [{ name: "Issue name cannot be empty" }] },
        touched: { translations: [{ name: true }] },
        isValid: false,
        isSubmitting: false,
    },
    submitting: {
        values: { 
            issueFor: IssueFor.Resource,
            for: { id: "123456789012345678" },
            translations: [{ language: "en", name: "Login button not working", description: "When I click login, nothing happens" }] 
        },
        errors: {},
        touched: { translations: [{ name: true, description: true }] },
        isValid: true,
        isSubmitting: true,
    },
    submitted: {
        values: { 
            issueFor: IssueFor.Resource,
            for: { id: "" },
            translations: [{ language: "en", name: "", description: "" }] 
        }, // Reset after successful submit
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
};

/**
 * Issue form scenarios for different contexts
 */
export const issueFormScenarios = {
    bugReport: {
        __typename: "Issue" as const,
        issueFor: IssueFor.Resource,
        for: { id: "123456789012345678" },
        translations: [{
            __typename: "IssueTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            name: "Export button displays incorrect text",
            description: "The export button shows 'undefined' instead of 'Export to CSV'",
        }],
    },
    featureRequest: {
        __typename: "Issue" as const,
        issueFor: IssueFor.Team,
        for: { id: "234567890123456789" },
        translations: [{
            __typename: "IssueTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            name: "Feature Request: Bulk member invite",
            description: "It would be helpful to invite multiple team members at once using a CSV upload",
        }],
    },
    performanceIssue: {
        __typename: "Issue" as const,
        issueFor: IssueFor.Resource,
        for: { id: "345678901234567890" },
        translations: [{
            __typename: "IssueTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            name: "[Performance] Search takes too long with large datasets",
            description: "Search queries take over 10 seconds when database has more than 1M records",
        }],
    },
    accessibilityIssue: {
        __typename: "Issue" as const,
        issueFor: IssueFor.Team,
        for: { id: "456789012345678901" },
        translations: [{
            __typename: "IssueTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            name: "Screen reader cannot read member list",
            description: "The team member list is not properly labeled for screen readers, making it inaccessible",
        }],
    },
};