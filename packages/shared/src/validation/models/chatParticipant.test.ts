import { describe, expect, it } from "vitest";
import { chatParticipantFixtures } from "./__test/fixtures/chatParticipantFixtures.js";
import { runStandardValidationTests, testValidation } from "./__test/validationTestUtils.js";
import { chatParticipantValidation } from "./chatParticpant.js";

describe("chatParticipantValidation", () => {
    // Run standard validation tests using shared fixtures
    // Note: This model only has update operation, no create
    runStandardValidationTests(
        chatParticipantValidation,
        chatParticipantFixtures,
        "chatParticipant",
    );

    describe("update operation", () => {
        const defaultParams = { omitFields: [] };
        const updateSchema = chatParticipantValidation.update(defaultParams);

        it("should accept minimal valid data", async () => {
            await testValidation(
                updateSchema,
                chatParticipantFixtures.minimal.update,
                true,
            );
        });

        it("should accept complete valid data", async () => {
            await testValidation(
                updateSchema,
                chatParticipantFixtures.complete.update,
                true,
            );
        });

        it("should require id field", async () => {
            await testValidation(
                updateSchema,
                chatParticipantFixtures.invalid.missingRequired.update,
                false,
                /required/i,
            );
        });

        it("should reject non-string id types", async () => {
            await testValidation(
                updateSchema,
                chatParticipantFixtures.invalid.invalidTypes.update,
                false,
            );
        });

        it("should reject invalid id format", async () => {
            await testValidation(
                updateSchema,
                chatParticipantFixtures.invalid.invalidId.update,
                false,
            );
        });

        it("should reject empty string id", async () => {
            await testValidation(
                updateSchema,
                chatParticipantFixtures.invalid.emptyId.update,
                false,
            );
        });

        it("should reject null id", async () => {
            await testValidation(
                updateSchema,
                chatParticipantFixtures.invalid.nullId.update,
                false,
            );
        });

        it("should reject undefined id", async () => {
            await testValidation(
                updateSchema,
                chatParticipantFixtures.invalid.undefinedId.update,
                false,
            );
        });

        it("should strip unknown fields", async () => {
            const result = await updateSchema.validate(
                chatParticipantFixtures.edgeCases.extraFields.update,
                { stripUnknown: true },
            );

            expect(result).to.have.property("id");
            expect(result).to.not.have.property("unknownField1");
            expect(result).to.not.have.property("unknownField2");
            expect(result).to.not.have.property("unknownField3");
        });

        describe("id field edge cases", () => {
            it("should accept maximum length snowflake ID", async () => {
                await testValidation(
                    updateSchema,
                    chatParticipantFixtures.edgeCases.maxLengthId.update,
                    true,
                );
            });

            it("should accept minimum length snowflake ID", async () => {
                await testValidation(
                    updateSchema,
                    chatParticipantFixtures.edgeCases.minLengthId.update,
                    true,
                );
            });

            it("should accept various valid snowflake IDs", async () => {
                const validIds = [
                    "123456789012345678",
                    "987654321098765432",
                    "555555555555555555",
                    "111111111111111111",
                    "999999999999999999",
                ];

                for (const validId of validIds) {
                    await testValidation(
                        updateSchema,
                        { id: validId },
                        true,
                    );
                }
            });

            it("should reject various invalid ID formats", async () => {
                const invalidIds = [
                    "12345", // Too short
                    "not-a-number",
                    "123abc789012345678", // Contains letters
                    "123 456 789 012 345 678", // Contains spaces
                    "123-456-789-012-345-678", // Contains hyphens
                    "1234567890123456789012345678", // Too long
                ];

                for (const invalidId of invalidIds) {
                    await testValidation(
                        updateSchema,
                        { id: invalidId },
                        false,
                    );
                }
            });
        });
    });

    describe("create operation", () => {
        it("should not have create operation", () => {
            expect(chatParticipantValidation.create).to.be.undefined;
        });
    });
});
