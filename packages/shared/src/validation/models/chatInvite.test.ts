import { describe, it, expect } from "vitest";
import { chatInviteValidation } from "./chatInvite.js";
import { chatInviteFixtures } from "./__test__/fixtures/chatInviteFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test__/validationTestUtils.js";

describe("chatInviteValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        chatInviteValidation,
        chatInviteFixtures,
        "chatInvite",
    );

    describe("specific field validations", () => {
        const defaultParams = { omitFields: [] };

        describe("message field", () => {
            const createSchema = chatInviteValidation.create(defaultParams);
            const updateSchema = chatInviteValidation.update(defaultParams);

            it("should accept valid messages", async () => {
                const scenarios = [
                    {
                        data: {
                            ...chatInviteFixtures.minimal.create,
                            message: "Short message",
                        },
                        shouldPass: true,
                        description: "short valid message",
                    },
                    {
                        data: {
                            ...chatInviteFixtures.minimal.create,
                            message: "A longer invitation message that contains more details about the chat",
                        },
                        shouldPass: true,
                        description: "longer valid message",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should be optional in both create and update", async () => {
                await testValidation(
                    createSchema,
                    chatInviteFixtures.minimal.create,
                    true,
                );

                await testValidation(
                    updateSchema,
                    chatInviteFixtures.edgeCases.updateWithoutMessage.update,
                    true,
                );
            });

            it("should trim whitespace from message", async () => {
                const result = await createSchema.validate(
                    chatInviteFixtures.edgeCases.whitespaceMessage.create,
                );
                expect(result.message).to.equal("Invitation message with whitespace");
            });

            it("should handle empty string message", async () => {
                const result = await createSchema.validate(
                    chatInviteFixtures.edgeCases.emptyMessage.create,
                );
                // Empty string should be removed by removeEmptyString()
                expect(result.message).to.be.undefined;
            });

            it("should reject non-string message types", async () => {
                await testValidation(
                    createSchema,
                    {
                        ...chatInviteFixtures.minimal.create,
                        message: 123,
                    },
                    false,
                );

                await testValidation(
                    createSchema,
                    {
                        ...chatInviteFixtures.minimal.create,
                        message: false,
                    },
                    false,
                );
            });
        });

        describe("required relationship fields", () => {
            const createSchema = chatInviteValidation.create(defaultParams);

            it("should require chatConnect in create", async () => {
                await testValidation(
                    createSchema,
                    chatInviteFixtures.invalid.missingChatConnect.create,
                    false,
                    /required/i,
                );
            });

            it("should require userConnect in create", async () => {
                await testValidation(
                    createSchema,
                    chatInviteFixtures.invalid.missingUserConnect.create,
                    false,
                    /required/i,
                );
            });

            it("should reject invalid chatConnect ID", async () => {
                await testValidation(
                    createSchema,
                    chatInviteFixtures.invalid.invalidChatConnect.create,
                    false,
                );
            });

            it("should reject invalid userConnect ID", async () => {
                await testValidation(
                    createSchema,
                    chatInviteFixtures.invalid.invalidUserConnect.create,
                    false,
                );
            });
        });

        describe("id field validation", () => {
            const createSchema = chatInviteValidation.create(defaultParams);
            const updateSchema = chatInviteValidation.update(defaultParams);

            it("should accept valid snowflake IDs", async () => {
                const validIds = [
                    "123456789012345678",
                    "123456789012345679",
                    "999999999999999999",
                ];

                for (const validId of validIds) {
                    await testValidation(
                        createSchema,
                        {
                            ...chatInviteFixtures.minimal.create,
                            id: validId,
                        },
                        true,
                    );

                    await testValidation(
                        updateSchema,
                        {
                            ...chatInviteFixtures.minimal.update,
                            id: validId,
                        },
                        true,
                    );
                }
            });

            it("should reject invalid IDs", async () => {
                const invalidIds = [
                    "not-a-snowflake",
                    "123",
                    "",
                    null,
                    undefined,
                    123,
                ];

                for (const invalidId of invalidIds) {
                    await testValidation(
                        createSchema,
                        {
                            ...chatInviteFixtures.minimal.create,
                            id: invalidId,
                        },
                        false,
                    );

                    await testValidation(
                        updateSchema,
                        {
                            ...chatInviteFixtures.minimal.update,
                            id: invalidId,
                        },
                        false,
                    );
                }
            });
        });

        describe("update operation", () => {
            const updateSchema = chatInviteValidation.update(defaultParams);

            it("should not require chat and user connections in update", async () => {
                await testValidation(
                    updateSchema,
                    chatInviteFixtures.minimal.update,
                    true,
                );
            });

            it("should accept message updates", async () => {
                await testValidation(
                    updateSchema,
                    chatInviteFixtures.complete.update,
                    true,
                );
            });

            it("should handle empty message in update", async () => {
                const result = await updateSchema.validate(
                    chatInviteFixtures.edgeCases.updateWithEmptyMessage.update,
                );
                expect(result.message).to.be.undefined;
            });
        });
    });
});
