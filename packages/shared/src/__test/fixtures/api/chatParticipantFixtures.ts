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
        create: null as never, // No create operation defined
        update: {
            id: validIds.id1,
        } as ChatParticipantUpdateInput,
    },
    complete: {
        create: null as never, // No create operation defined
        update: {
            id: validIds.id2,
        } as ChatParticipantUpdateInput,
    },
    invalid: {
        missingRequired: {
            create: null as never, // No create operation defined
            update: {
                // Missing required id
            } as unknown as ChatParticipantUpdateInput,
        },
        invalidTypes: {
            create: null as never, // No create operation defined
            update: {
                id: 123, // Should be string
            } as unknown as ChatParticipantUpdateInput,
        },
        invalidId: {
            create: null as never, // No create operation defined
            update: {
                id: "not-a-valid-snowflake",
            } as ChatParticipantUpdateInput,
        },
        emptyId: {
            create: null as never, // No create operation defined
            update: {
                id: "",
            } as ChatParticipantUpdateInput,
        },
        nullId: {
            create: null as never, // No create operation defined
            update: {
                id: null,
            } as unknown as ChatParticipantUpdateInput,
        },
        undefinedId: {
            create: null as never, // No create operation defined
            update: {
                id: undefined,
            } as unknown as ChatParticipantUpdateInput,
        },
    },
    edgeCases: {
        maxLengthId: {
            create: null as never, // No create operation defined
            update: {
                id: "999999999999999999", // Max valid snowflake ID
            } as ChatParticipantUpdateInput,
        },
        minLengthId: {
            create: null as never, // No create operation defined
            update: {
                id: "100000000000000000", // Min valid snowflake ID
            } as ChatParticipantUpdateInput,
        },
        extraFields: {
            create: null as never, // No create operation defined
            update: {
                id: validIds.id1,
                unknownField1: "should be stripped",
                unknownField2: 123,
                unknownField3: { nested: "object" },
            } as unknown as ChatParticipantUpdateInput,
        },
    },
};

// Custom factory that always generates valid IDs
const customizers = {
    create: (_base: never): never => null as never, // No create operation
    update: (base: Partial<ChatParticipantUpdateInput>): ChatParticipantUpdateInput => ({
        id: validIds.id1,
        ...base,
    }),
};

// Export a factory for creating test data programmatically
export const chatParticipantTestDataFactory = new TypedTestDataFactory(chatParticipantFixtures, chatParticipantValidation, customizers);

// Export typed fixtures with validation methods
export const typedChatParticipantFixtures = createTypedFixtures(chatParticipantFixtures, chatParticipantValidation);
