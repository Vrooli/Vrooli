import { DUMMY_ID, SubscribableObject, type NotificationSubscriptionCreateInput } from "@vrooli/shared";

/**
 * Form data fixtures for notification subscription creation and editing
 * 
 * Notification subscriptions allow users to subscribe to updates on various objects
 * like teams, resources, comments, issues, meetings, etc.
 */

/**
 * Minimal notification subscription form input - just subscribing to a resource
 */
export const minimalNotificationSubscriptionFormInput: NotificationSubscriptionCreateInput = {
    id: DUMMY_ID,
    objectType: SubscribableObject.Resource,
    objectConnect: "123456789012345678", // Resource ID (18 digits)
};

/**
 * Complete notification subscription form input with silent option
 */
export const completeNotificationSubscriptionFormInput: NotificationSubscriptionCreateInput = {
    id: DUMMY_ID,
    silent: true,
    objectType: SubscribableObject.Team,
    objectConnect: "987654321098765432", // Team ID (18 digits)
};

/**
 * Notification subscription variants for different object types
 */
export const notificationSubscriptionFormVariants = {
    comment: {
        id: DUMMY_ID,
        objectType: SubscribableObject.Comment,
        objectConnect: "111222333444555666", // Comment ID (18 digits)
    },
    issue: {
        id: DUMMY_ID,
        objectType: SubscribableObject.Issue,
        objectConnect: "222333444555666777", // Issue ID (18 digits)
    },
    meeting: {
        id: DUMMY_ID,
        objectType: SubscribableObject.Meeting,
        objectConnect: "333444555666777888", // Meeting ID (18 digits)
    },
    pullRequest: {
        id: DUMMY_ID,
        objectType: SubscribableObject.PullRequest,
        objectConnect: "444555666777888999", // Pull Request ID (18 digits)
    },
    report: {
        id: DUMMY_ID,
        objectType: SubscribableObject.Report,
        objectConnect: "555666777888999000", // Report ID (18 digits)
    },
    resource: {
        id: DUMMY_ID,
        objectType: SubscribableObject.Resource,
        objectConnect: "666777888999000111", // Resource ID (18 digits)
    },
    schedule: {
        id: DUMMY_ID,
        objectType: SubscribableObject.Schedule,
        objectConnect: "777888999000111222", // Schedule ID (18 digits)
    },
    team: {
        id: DUMMY_ID,
        objectType: SubscribableObject.Team,
        objectConnect: "888999000111222333", // Team ID (18 digits)
    },
};

/**
 * Silent notification subscription variants
 */
export const silentNotificationSubscriptionFormInputs = {
    silentTeam: {
        id: DUMMY_ID,
        silent: true,
        objectType: SubscribableObject.Team,
        objectConnect: "123456789012345678",
    },
    notSilentResource: {
        id: DUMMY_ID,
        silent: false,
        objectType: SubscribableObject.Resource,
        objectConnect: "234567890123456789",
    },
};

/**
 * Update form inputs (only id and optional silent field)
 */
export const updateNotificationSubscriptionFormInputs = {
    minimalUpdate: {
        id: "345678901234567890",
    },
    toggleSilent: {
        id: "456789012345678901",
        silent: true,
    },
    unsilence: {
        id: "567890123456789012",
        silent: false,
    },
};

/**
 * Invalid form inputs for validation testing
 */
export const invalidNotificationSubscriptionFormInputs = {
    missingObjectType: {
        id: DUMMY_ID,
        // @ts-expect-error - Testing invalid input
        objectType: undefined,
        objectConnect: "123456789012345678",
    },
    missingObjectConnect: {
        id: DUMMY_ID,
        objectType: SubscribableObject.Resource,
        // @ts-expect-error - Testing invalid input
        objectConnect: undefined,
    },
    invalidObjectType: {
        id: DUMMY_ID,
        // @ts-expect-error - Testing invalid input
        objectType: "InvalidType",
        objectConnect: "123456789012345678",
    },
    invalidObjectConnect: {
        id: DUMMY_ID,
        objectType: SubscribableObject.Resource,
        objectConnect: "not-a-valid-id", // Invalid: not a snowflake ID
    },
    invalidSilentType: {
        id: DUMMY_ID,
        // @ts-expect-error - Testing invalid input
        silent: "yes", // Should be boolean
        objectType: SubscribableObject.Team,
        objectConnect: "123456789012345678",
    },
};

/**
 * Helper function to create initial form values for notification subscription
 */
export const createNotificationSubscriptionFormInitialValues = (
    objectType: SubscribableObject,
    objectId: string,
    silent?: boolean,
) => ({
    id: DUMMY_ID,
    objectType,
    objectConnect: objectId,
    ...(silent !== undefined && { silent }),
});

/**
 * Helper function to transform UI state to API input
 */
export const transformNotificationSubscriptionFormToApiInput = (formData: any): NotificationSubscriptionCreateInput => ({
    id: formData.id || DUMMY_ID,
    objectType: formData.objectType,
    objectConnect: formData.objectConnect,
    ...(formData.silent !== undefined && { silent: formData.silent }),
});