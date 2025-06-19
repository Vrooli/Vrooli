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

// Shared push device test fixtures
export const pushDeviceFixtures: ModelTestFixtures = {
    minimal: {
        create: {
            endpoint: "https://fcm.googleapis.com/fcm/send/example-endpoint",
            keys: {
                p256dh: "BKJYlp5Q8hJJ6RRXL8dV6wJG3l4zDWr-EXAMPLE-KEY",
                auth: "AUTH-KEY-EXAMPLE",
            },
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            endpoint: "https://fcm.googleapis.com/fcm/send/complete-example-endpoint",
            expires: 86400, // 24 hours in seconds
            keys: {
                p256dh: "BKJYlp5Q8hJJ6RRXL8dV6wJG3l4zDWr-COMPLETE-EXAMPLE-KEY",
                auth: "COMPLETE-AUTH-KEY-EXAMPLE",
            },
            name: "My Phone",
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 123,
            unknownField3: true,
        },
        update: {
            id: validIds.id1,
            name: "Updated Device Name",
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 456,
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing required endpoint and keys
                expires: 86400,
                name: "Incomplete Device",
            },
            update: {
                // Missing required id
                name: "Updated Name",
            },
        },
        invalidTypes: {
            create: {
                endpoint: 123, // Should be string URL
                expires: "not-a-number", // Should be number
                keys: "not-an-object", // Should be object
                name: 456, // Should be string
            },
            update: {
                id: 123, // Should be string
                name: 789, // Should be string
            },
        },
        invalidId: {
            create: {
                endpoint: "https://valid-endpoint.com/path",
                keys: {
                    p256dh: "VALID-P256DH-KEY",
                    auth: "VALID-AUTH-KEY",
                },
            },
            update: {
                id: "not-a-valid-snowflake",
            },
        },
        invalidEndpoint: {
            create: {
                endpoint: "not-a-valid-url",
                keys: {
                    p256dh: "VALID-P256DH-KEY",
                    auth: "VALID-AUTH-KEY",
                },
            },
        },
        invalidKeys: {
            create: {
                endpoint: "https://valid-endpoint.com/path",
                keys: {
                    // Missing required auth field
                    p256dh: "VALID-P256DH-KEY",
                },
            },
        },
        missingKeysAuth: {
            create: {
                endpoint: "https://valid-endpoint.com/path",
                keys: {
                    p256dh: "VALID-P256DH-KEY",
                    // Missing required auth field
                },
            },
        },
        missingKeysP256dh: {
            create: {
                endpoint: "https://valid-endpoint.com/path",
                keys: {
                    // Missing required p256dh field
                    auth: "VALID-AUTH-KEY",
                },
            },
        },
        invalidExpires: {
            create: {
                endpoint: "https://valid-endpoint.com/path",
                expires: -100, // Should be positive or zero
                keys: {
                    p256dh: "VALID-P256DH-KEY",
                    auth: "VALID-AUTH-KEY",
                },
            },
        },
    },
    edgeCases: {
        withoutOptionalFields: {
            create: {
                endpoint: "https://minimal-endpoint.com/path",
                keys: {
                    p256dh: "MINIMAL-P256DH-KEY",
                    auth: "MINIMAL-AUTH-KEY",
                },
                // No expires or name
            },
        },
        withZeroExpires: {
            create: {
                endpoint: "https://zero-expires.com/path",
                expires: 0, // Valid zero value
                keys: {
                    p256dh: "ZERO-EXPIRES-P256DH-KEY",
                    auth: "ZERO-EXPIRES-AUTH-KEY",
                },
            },
        },
        withLargeExpires: {
            create: {
                endpoint: "https://large-expires.com/path",
                expires: 2147483647, // Max 32-bit signed integer
                keys: {
                    p256dh: "LARGE-EXPIRES-P256DH-KEY",
                    auth: "LARGE-EXPIRES-AUTH-KEY",
                },
            },
        },
        longDeviceName: {
            create: {
                endpoint: "https://long-name.com/path",
                keys: {
                    p256dh: "LONG-NAME-P256DH-KEY",
                    auth: "LONG-NAME-AUTH-KEY",
                },
                name: "This is a long device name within 50 chars max", // 47 characters
            },
        },
        differentEndpoints: {
            create: {
                endpoint: "https://different-push-service.com/v1/send/abcd1234",
                keys: {
                    p256dh: "DIFFERENT-SERVICE-P256DH-KEY",
                    auth: "DIFFERENT-SERVICE-AUTH-KEY",
                },
                name: "Different Service Device",
            },
        },
        updateWithSameName: {
            update: {
                id: validIds.id1,
                name: "Same Name",
            },
        },
        updateWithNewName: {
            update: {
                id: validIds.id1,
                name: "Completely New Name",
            },
        },
        updateOnlyId: {
            update: {
                id: validIds.id1,
                // Only required field
            },
        },
        httpEndpoint: {
            create: {
                endpoint: "http://insecure-endpoint.com/path", // HTTP instead of HTTPS
                keys: {
                    p256dh: "HTTP-P256DH-KEY",
                    auth: "HTTP-AUTH-KEY",
                },
            },
        },
        longKeys: {
            create: {
                endpoint: "https://long-keys.com/path",
                keys: {
                    p256dh: "A".repeat(255), // Very long key near max length
                    auth: "B".repeat(255), // Very long key near max length
                },
                name: "Long Keys Device",
            },
        },
    },
};

// Custom factory that always generates valid IDs and required fields
const customizers = {
    create: (base: any) => ({
        ...base,
        endpoint: base.endpoint || "https://default-endpoint.com/path",
        keys: base.keys || {
            p256dh: "DEFAULT-P256DH-KEY",
            auth: "DEFAULT-AUTH-KEY",
        },
    }),
    update: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export a factory for creating test data programmatically
export const pushDeviceTestDataFactory = new TestDataFactory(pushDeviceFixtures, customizers);
