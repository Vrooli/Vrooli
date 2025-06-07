import { describe, it, expect } from "vitest";
import { meetingInviteValidation } from "./meetingInvite.js";
import { meetingInviteFixtures } from "./__test__/fixtures/meetingInviteFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test__/validationTestUtils.js";

describe("meetingInviteValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        meetingInviteValidation,
        meetingInviteFixtures,
        "meetingInvite",
    );

    describe("create specific validations", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = meetingInviteValidation.create(defaultParams);

        it("should require id in create", async () => {
            const dataWithoutId = { ...meetingInviteFixtures.minimal.create };
            delete dataWithoutId.id;

            await testValidation(
                createSchema,
                dataWithoutId,
                false,
                /required/i,
            );
        });

        it("should require meetingConnect in create", async () => {
            await testValidation(
                createSchema,
                meetingInviteFixtures.invalid.missingMeetingConnect.create,
                false,
                /required/i,
            );
        });

        it("should require userConnect in create", async () => {
            await testValidation(
                createSchema,
                meetingInviteFixtures.invalid.missingUserConnect.create,
                false,
                /required/i,
            );
        });

        it("should allow minimal create with required fields", async () => {
            await testValidation(
                createSchema,
                meetingInviteFixtures.minimal.create,
                true,
            );
        });

        it("should allow complete create with all fields", async () => {
            await testValidation(
                createSchema,
                meetingInviteFixtures.complete.create,
                true,
            );
        });

        describe("message field", () => {
            it("should be optional", async () => {
                await testValidation(createSchema, meetingInviteFixtures.minimal.create, true);
            });

            it("should accept valid message strings", async () => {
                const scenarios = [
                    {
                        data: {
                            ...meetingInviteFixtures.minimal.create,
                            message: "Short message",
                        },
                        shouldPass: true,
                        description: "short message",
                    },
                    {
                        data: {
                            ...meetingInviteFixtures.minimal.create,
                            message: "A longer invitation message with more details about the meeting",
                        },
                        shouldPass: true,
                        description: "longer message",
                    },
                    {
                        data: meetingInviteFixtures.edgeCases.multilineMessage.create,
                        shouldPass: true,
                        description: "multiline message",
                    },
                    {
                        data: meetingInviteFixtures.edgeCases.specialCharactersMessage.create,
                        shouldPass: true,
                        description: "message with special characters",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject messages that are too long", async () => {
                await testValidation(
                    createSchema,
                    meetingInviteFixtures.invalid.tooLongMessage.create,
                    false,
                    /over the limit/,
                );
            });

            it("should handle empty messages", async () => {
                await testValidation(
                    createSchema,
                    meetingInviteFixtures.edgeCases.emptyMessage.create,
                    true,
                );
            });

            it("should trim whitespace from messages", async () => {
                const result = await testValidation(
                    createSchema,
                    meetingInviteFixtures.edgeCases.whitespaceMessage.create,
                    true,
                );
                // Whitespace-only strings should be converted to undefined
                expect(result.message).to.be.undefined;
            });

            it("should accept maximum length messages", async () => {
                await testValidation(
                    createSchema,
                    meetingInviteFixtures.edgeCases.maxLengthMessage.create,
                    true,
                );
            });
        });

        describe("meetingConnect field", () => {
            it("should accept valid meeting ID", async () => {
                const validData = {
                    ...meetingInviteFixtures.minimal.create,
                    meetingConnect: "123456789012345678",
                };

                await testValidation(createSchema, validData, true);
            });

            it("should reject invalid meeting ID", async () => {
                const invalidData = {
                    ...meetingInviteFixtures.minimal.create,
                    meetingConnect: "invalid-id",
                };

                await testValidation(
                    createSchema,
                    invalidData,
                    false,
                    /valid ID/i,
                );
            });
        });

        describe("userConnect field", () => {
            it("should accept valid user ID", async () => {
                const validData = {
                    ...meetingInviteFixtures.minimal.create,
                    userConnect: "123456789012345678",
                };

                await testValidation(createSchema, validData, true);
            });

            it("should reject invalid user ID", async () => {
                const invalidData = {
                    ...meetingInviteFixtures.minimal.create,
                    userConnect: "invalid-id",
                };

                await testValidation(
                    createSchema,
                    invalidData,
                    false,
                    /valid ID/i,
                );
            });
        });
    });

    describe("update specific validations", () => {
        const defaultParams = { omitFields: [] };
        const updateSchema = meetingInviteValidation.update(defaultParams);

        it("should require id in update", async () => {
            await testValidation(
                updateSchema,
                meetingInviteFixtures.invalid.missingRequired.update,
                false,
                /required/i,
            );
        });

        it("should allow minimal updates with just id", async () => {
            await testValidation(
                updateSchema,
                meetingInviteFixtures.minimal.update,
                true,
            );
        });

        it("should allow message updates", async () => {
            await testValidation(
                updateSchema,
                meetingInviteFixtures.complete.update,
                true,
            );
        });

        it("should not allow meetingConnect updates", async () => {
            const updateData = {
                id: "123456789012345678",
                meetingConnect: "123456789012345679",
            };

            const result = await testValidation(updateSchema, updateData, true);
            expect(result).to.not.have.property("meetingConnect");
        });

        it("should not allow userConnect updates", async () => {
            const updateData = {
                id: "123456789012345678",
                userConnect: "123456789012345679",
            };

            const result = await testValidation(updateSchema, updateData, true);
            expect(result).to.not.have.property("userConnect");
        });

        describe("message field updates", () => {
            it("should accept valid message updates", async () => {
                const scenarios = [
                    {
                        data: {
                            ...meetingInviteFixtures.minimal.update,
                            message: "Updated invitation message",
                        },
                        shouldPass: true,
                        description: "simple message update",
                    },
                    {
                        data: meetingInviteFixtures.edgeCases.maxLengthMessage.update,
                        shouldPass: true,
                        description: "max length message update",
                    },
                    {
                        data: meetingInviteFixtures.edgeCases.emptyMessage.update,
                        shouldPass: true,
                        description: "empty message update",
                    },
                ];

                await testValidationBatch(updateSchema, scenarios);
            });

            it("should reject messages that are too long in update", async () => {
                await testValidation(
                    updateSchema,
                    meetingInviteFixtures.invalid.tooLongMessage.update,
                    false,
                    /over the limit/,
                );
            });
        });
    });

    describe("id validation", () => {
        const createSchema = meetingInviteValidation.create({ omitFields: [] });

        it("should accept valid Snowflake IDs", async () => {
            const validIds = [
                "123456789012345678",
                "999999999999999999",
                "100000000000000000",
            ];

            const scenarios = validIds.map(id => ({
                data: {
                    ...meetingInviteFixtures.minimal.create,
                    id,
                },
                shouldPass: true,
                description: `valid ID: ${id}`,
            }));

            await testValidationBatch(createSchema, scenarios);
        });

        it("should reject invalid IDs", async () => {
            const invalidIds = [
                "not-a-number",
                "abc123", // Contains letters
                "", // Empty string
                "-123", // Negative number
            ];

            const scenarios = invalidIds.map(id => ({
                data: {
                    ...meetingInviteFixtures.minimal.create,
                    id,
                },
                shouldPass: false,
                expectedError: id === "" ? /required/i : /valid ID/i,
                description: `invalid ID: ${id}`,
            }));

            await testValidationBatch(createSchema, scenarios);
        });
    });

    describe("edge cases", () => {
        const createSchema = meetingInviteValidation.create({ omitFields: [] });

        it("should handle invites with different user and meeting IDs", async () => {
            await testValidation(
                createSchema,
                meetingInviteFixtures.edgeCases.differentUserAndMeetingIds.create,
                true,
            );
        });

        it("should handle complex message content", async () => {
            const scenarios = [
                {
                    data: meetingInviteFixtures.edgeCases.multilineMessage.create,
                    shouldPass: true,
                    description: "multiline message with formatting",
                },
                {
                    data: meetingInviteFixtures.edgeCases.specialCharactersMessage.create,
                    shouldPass: true,
                    description: "message with emojis and special characters",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle type conversions", async () => {
            const dataWithNumberMessage = {
                ...meetingInviteFixtures.minimal.create,
                message: 123, // Number instead of string
            };

            // The message field converts numbers to strings
            const result = await testValidation(
                createSchema,
                dataWithNumberMessage,
                true,
            );
            expect(result.message).to.equal("123");
        });
    });

    describe("required fields validation", () => {
        const createSchema = meetingInviteValidation.create({ omitFields: [] });

        it("should fail when missing all required connections", async () => {
            await testValidation(
                createSchema,
                meetingInviteFixtures.invalid.missingRequired.create,
                false,
                /required/i,
            );
        });

        it("should fail when missing only meetingConnect", async () => {
            await testValidation(
                createSchema,
                meetingInviteFixtures.invalid.missingMeetingConnect.create,
                false,
                /required/i,
            );
        });

        it("should fail when missing only userConnect", async () => {
            await testValidation(
                createSchema,
                meetingInviteFixtures.invalid.missingUserConnect.create,
                false,
                /required/i,
            );
        });
    });
});
