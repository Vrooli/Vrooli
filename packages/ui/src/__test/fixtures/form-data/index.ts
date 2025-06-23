/**
 * Form Data Fixtures - Central Exports
 * 
 * This file provides a centralized export point for all form data fixtures
 * in the fixtures-updated architecture.
 */

// Core object fixtures
export * from "./userFormData.js";
export * from "./teamFormData.js";
export * from "./projectFormData.js";

// Communication object fixtures
export * from "./chatFormData.js";
export * from "./commentFormData.js";

// Organizational object fixtures  
export * from "./meetingFormData.js";
export * from "./bookmarkFormData.js"; // Existing from current implementation

// Export commonly used types
export type {
    // User-related types
    UserRegistrationFormData,
    UserProfileFormData,
    UserLoginFormData,
    UserFormState,
    
    // Team-related types
    TeamCreateFormData,
    TeamProfileFormData,
    TeamMemberFormData,
    TeamFormState,
    
    // Project-related types
    ProjectFormData,
    ProjectFormState,
    
    // Chat-related types
    ChatCreateFormData,
    ChatMessageFormData,
    ChatSettingsFormData,
    ChatFormState,
    
    // Comment-related types
    CommentFormData,
    CommentFormState,
    
    // Meeting-related types
    MeetingFormData,
    MeetingFormState,
    
    // Bookmark-related types (from existing implementation)
    BookmarkFormData,
    BookmarkFormState,
} from "./userFormData.js";

// Re-export just the type imports to avoid circular dependencies
export type { TeamCreateFormData, TeamProfileFormData, TeamMemberFormData, TeamFormState } from "./teamFormData.js";
export type { ProjectFormData, ProjectFormState } from "./projectFormData.js";
export type { ChatCreateFormData, ChatMessageFormData, ChatSettingsFormData, ChatFormState } from "./chatFormData.js";
export type { CommentFormData, CommentFormState } from "./commentFormData.js";
export type { MeetingFormData, MeetingFormState } from "./meetingFormData.js";
export type { BookmarkFormData, BookmarkFormState } from "./bookmarkFormData.js";

// Export factory instances for easy access
export {
    userFormFactory,
    userFormSimulator,
    userFormScenarios,
} from "./userFormData.js";

export {
    teamFormFactory,
    teamFormSimulator,
    teamFormScenarios,
} from "./teamFormData.js";

export {
    projectFormFactory,
    projectFormSimulator,
    projectFormScenarios,
} from "./projectFormData.js";

export {
    chatFormFactory,
    chatFormSimulator,
    chatFormScenarios,
} from "./chatFormData.js";

export {
    commentFormFactory,
    commentFormSimulator,
    commentFormScenarios,
} from "./commentFormData.js";

export {
    meetingFormFactory,
    meetingFormSimulator,
    meetingFormScenarios,
} from "./meetingFormData.js";

export {
    bookmarkFormFactory,
    bookmarkFormSimulator,
    bookmarkFormScenarios,
} from "./bookmarkFormData.js";

/**
 * Convenience object grouping all form factories
 */
export const formFactories = {
    user: userFormFactory,
    team: teamFormFactory,
    project: projectFormFactory,
    chat: chatFormFactory,
    comment: commentFormFactory,
    meeting: meetingFormFactory,
    bookmark: bookmarkFormFactory,
} as const;

/**
 * Convenience object grouping all form simulators
 */
export const formSimulators = {
    user: userFormSimulator,
    team: teamFormSimulator,
    project: projectFormSimulator,
    chat: chatFormSimulator,
    comment: commentFormSimulator,
    meeting: meetingFormSimulator,
    bookmark: bookmarkFormSimulator,
} as const;

/**
 * Convenience object grouping all form scenarios
 */
export const formScenarios = {
    user: userFormScenarios,
    team: teamFormScenarios,
    project: projectFormScenarios,
    chat: chatFormScenarios,
    comment: commentFormScenarios,
    meeting: meetingFormScenarios,
    bookmark: bookmarkFormScenarios,
} as const;

/**
 * Helper function to get all available form types
 */
export function getAvailableFormTypes(): string[] {
    return Object.keys(formFactories);
}

/**
 * Helper function to create a form instance for any supported type
 */
export function createFormInstance<T extends keyof typeof formFactories>(
    formType: T,
    ...args: Parameters<typeof formFactories[T]["createFormInstance"]>
): ReturnType<typeof formFactories[T]["createFormInstance"]> {
    const factory = formFactories[formType];
    return factory.createFormInstance(...args as any);
}

/**
 * Helper function to validate form data for any supported type
 */
export async function validateFormData<T extends keyof typeof formFactories>(
    formType: T,
    formData: any,
    ...additionalArgs: any[]
): Promise<{ isValid: boolean; errors?: Record<string, string>; apiInput?: any }> {
    const factory = formFactories[formType];
    
    // Handle different factory method signatures
    if ("validateFormData" in factory && typeof factory.validateFormData === "function") {
        return factory.validateFormData(formData, ...additionalArgs);
    }
    
    throw new Error(`Form type ${formType} does not support validation`);
}

/**
 * Helper function to get form scenarios for any supported type
 */
export function getFormScenarios<T extends keyof typeof formScenarios>(
    formType: T,
): typeof formScenarios[T] {
    return formScenarios[formType];
}

/**
 * Type guard to check if a form type is supported
 */
export function isSupportedFormType(formType: string): formType is keyof typeof formFactories {
    return formType in formFactories;
}

/**
 * Meta information about the form fixtures
 */
export const formFixturesMeta = {
    version: "2.0.0",
    description: "Production-grade form fixtures with React Hook Form integration",
    features: [
        "Type-safe form data generation",
        "Real validation integration",
        "User interaction simulation",
        "React Hook Form compatibility",
        "Comprehensive error scenarios",
        "Accessibility testing support",
    ],
    supportedFormTypes: getAvailableFormTypes(),
    totalFixtures: Object.keys(formFactories).length,
} as const;
