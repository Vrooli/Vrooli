import type { ChatParticipantUpdateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { chatParticipantValidation } from "../../../validation/models/chatParticpant.js";

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
};

// Shared chatParticipant test fixtures
// Note: ChatParticipant only has update operations, no create
export const chatParticipantFixtures: ModelTestFixtures<never, ChatParticipantUpdateInput> = {
    minimal: {
        create: {} as never, // No create operation defined
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {} as never, // No create operation defined
        update: {
            id: validIds.id2,
        },
    },
    invalid: {
        missingRequired: {
            create: {} as never, // No create operation defined
            update: {
                // Missing required id
            },
        },
        invalidTypes: {
            create: {} as never, // No create operation defined
            update: {
                id: 123, // Should be string
            },
        },
        invalidId: {
            update: {
                id: "not-a-valid-snowflake",
            },
        },
        emptyId: {
            update: {
                id: "",
            },
        },
        nullId: {
            update: {
                id: null,
            },
        },
        undefinedId: {
            update: {
                id: undefined,
            },
        },
    },
    edgeCases: {
        maxLengthId: {
            update: {
                id: "999999999999999999", // Max valid snowflake ID
            },
        },
        minLengthId: {
            update: {
                id: "100000000000000000", // Min valid snowflake ID
            },
        },
        extraFields: {
            update: {
                id: validIds.id1,
                unknownField1: "should be stripped",
                unknownField2: 123,
                unknownField3: { nested: "object" },
            },
        },
    },
};

// Custom factory that always generates valid IDs
const customizers = {
    create: (base: never): never => base,
    update: (base: ChatParticipantUpdateInput): ChatParticipantUpdateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export a factory for creating test data programmatically
export const chatParticipantTestDataFactory = new TypedTestDataFactory(chatParticipantFixtures, chatParticipantValidation, customizers);

// Export typed fixtures with validation methods
export const typedChatParticipantFixtures = createTypedFixtures(chatParticipantFixtures, chatParticipantValidation);
