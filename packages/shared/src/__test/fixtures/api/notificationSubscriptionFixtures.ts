import { type NotificationSubscriptionCreateInput, type NotificationSubscriptionUpdateInput, SubscribableObject } from "../../../api/types.js";
import { type ModelTestFixtures, TestDataFactory, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { notificationSubscriptionValidation } from "../../../validation/models/notificationSubscription.js";

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    id4: "123456789012345681",
    objectId1: "123456789012345682",
    objectId2: "123456789012345683",
    objectId3: "123456789012345684",
    objectId4: "123456789012345685",
};

// Shared notificationSubscription test fixtures
export const notificationSubscriptionFixtures: ModelTestFixtures<NotificationSubscriptionCreateInput, NotificationSubscriptionUpdateInput> = {
    minimal: {
        create: {
            id: validIds.id1,
            objectType: SubscribableObject.Resource,
            objectConnect: validIds.objectId1,
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id2,
            silent: true,
            objectType: SubscribableObject.Team,
            objectConnect: validIds.objectId2,
            context: "Full subscription with context",
        },
        update: {
            id: validIds.id2,
            silent: false,
            context: "Updated context",
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing id, objectType, and objectConnect
                silent: true,
            },
            update: {
                // Missing id
                silent: false,
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                silent: "not-a-boolean", // Should be boolean
                objectType: "InvalidType", // Should be valid SubscribableObject enum
                objectConnect: 456, // Should be string
            },
            update: {
                id: validIds.id3,
                silent: "false", // Should be boolean, not string
            },
        },
        invalidObjectType: {
            create: {
                id: validIds.id1,
                objectType: "InvalidObjectType",
                objectConnect: validIds.objectId1,
            },
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                objectType: SubscribableObject.Resource,
                objectConnect: validIds.objectId1,
            },
        },
        invalidObjectConnect: {
            create: {
                id: validIds.id1,
                objectType: SubscribableObject.Resource,
                objectConnect: "not-a-valid-snowflake",
            },
        },
        missingObjectConnect: {
            create: {
                id: validIds.id1,
                objectType: SubscribableObject.Resource,
                // Missing required objectConnect
            },
        },
        missingObjectType: {
            create: {
                id: validIds.id1,
                objectConnect: validIds.objectId1,
                // Missing required objectType
            },
        },
    },
    edgeCases: {
        allObjectTypes: [
            {
                id: validIds.id1,
                objectType: SubscribableObject.Comment,
                objectConnect: validIds.objectId1,
            },
            {
                id: validIds.id2,
                objectType: SubscribableObject.Issue,
                objectConnect: validIds.objectId2,
            },
            {
                id: validIds.id3,
                objectType: SubscribableObject.Meeting,
                objectConnect: validIds.objectId3,
            },
            {
                id: validIds.id4,
                objectType: SubscribableObject.PullRequest,
                objectConnect: validIds.objectId4,
            },
            {
                id: validIds.id1,
                objectType: SubscribableObject.Report,
                objectConnect: validIds.objectId1,
            },
            {
                id: validIds.id2,
                objectType: SubscribableObject.Resource,
                objectConnect: validIds.objectId2,
            },
            {
                id: validIds.id3,
                objectType: SubscribableObject.Schedule,
                objectConnect: validIds.objectId3,
            },
            {
                id: validIds.id4,
                objectType: SubscribableObject.Team,
                objectConnect: validIds.objectId4,
            },
        ],
        silentOptions: [
            {
                id: validIds.id1,
                silent: true,
                objectType: SubscribableObject.Resource,
                objectConnect: validIds.objectId1,
            },
            {
                id: validIds.id2,
                silent: false,
                objectType: SubscribableObject.Team,
                objectConnect: validIds.objectId2,
            },
        ],
        withoutSilent: {
            create: {
                id: validIds.id1,
                // silent is optional
                objectType: SubscribableObject.Resource,
                objectConnect: validIds.objectId1,
            },
        },
        withContext: {
            create: {
                id: validIds.id3,
                objectType: SubscribableObject.Meeting,
                objectConnect: validIds.objectId3,
                context: "Subscription context information",
            },
            update: {
                id: validIds.id3,
                context: null, // Test clearing context
            },
        },
        updateSilentOnly: {
            update: {
                id: validIds.id1,
                silent: true,
                // No objectType or objectConnect in update
            },
        },
        updateWithoutSilent: {
            update: {
                id: validIds.id1,
                // No silent field - should be allowed
            },
        },
        differentObjectConnections: [
            {
                id: validIds.id1,
                objectType: SubscribableObject.Resource,
                objectConnect: "123456789012345690",
            },
            {
                id: validIds.id2,
                objectType: SubscribableObject.Team,
                objectConnect: "987654321098765432",
            },
            {
                id: validIds.id3,
                objectType: SubscribableObject.Comment,
                objectConnect: "555555555555555555",
            },
        ],
    },
};

// Custom factory that always generates valid IDs
const customizers = {
    create: (base: NotificationSubscriptionCreateInput) => ({
        ...base,
        id: base.id || validIds.id1,
    }),
    update: (base: NotificationSubscriptionUpdateInput) => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export a factory for creating test data programmatically
export const notificationSubscriptionTestDataFactory = new TypedTestDataFactory(notificationSubscriptionFixtures, notificationSubscriptionValidation, customizers);

// Export typed fixtures for direct use in tests
export const typedNotificationSubscriptionFixtures = createTypedFixtures(notificationSubscriptionFixtures, notificationSubscriptionValidation);

// Maintain backward compatibility with the old factory for existing tests
export const notificationSubscriptionTestDataFactoryLegacy = new TestDataFactory(notificationSubscriptionFixtures, customizers);
