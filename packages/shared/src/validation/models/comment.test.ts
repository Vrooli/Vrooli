import { describe, expect, it } from "vitest";
import { CommentFor } from "../../api/types.js";
import { commentFixtures } from "./__test/fixtures/commentFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test/validationTestUtils.js";
import { commentValidation } from "./comment.js";

describe("commentValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        commentValidation,
        commentFixtures,
        "comment",
    );

    describe("specific field validations", () => {
        const defaultParams = { omitFields: [] };

        describe("createdFor field", () => {
            const createSchema = commentValidation.create(defaultParams);

            it("should accept all valid CommentFor enum values", async () => {
                const validValues = Object.values(CommentFor);
                const scenarios = validValues.map(value => ({
                    data: {
                        ...commentFixtures.minimal.create,
                        createdFor: value,
                    },
                    shouldPass: true,
                    description: `createdFor: ${value}`,
                }));

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject invalid CommentFor values", async () => {
                const scenarios = [
                    {
                        data: {
                            ...commentFixtures.minimal.create,
                            createdFor: "InvalidType",
                        },
                        shouldPass: false,
                        expectedError: /must be one of the following values/i,
                        description: "invalid enum value",
                    },
                    {
                        data: {
                            ...commentFixtures.minimal.create,
                            createdFor: 123,
                        },
                        shouldPass: false,
                        expectedError: /must be one of the following values/i,
                        description: "number instead of string",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should be required in create", async () => {
                const dataWithoutCreatedFor = { ...commentFixtures.minimal.create };
                delete dataWithoutCreatedFor.createdFor;

                await testValidation(
                    createSchema,
                    dataWithoutCreatedFor,
                    false,
                    /required/i,
                );
            });
        });

        describe("forConnect field", () => {
            const createSchema = commentValidation.create(defaultParams);

            it("should require forConnect in create", async () => {
                await testValidation(
                    createSchema,
                    commentFixtures.invalid.missingForConnection.create,
                    false,
                    /required/i,
                );
            });

            it("should accept valid forConnect ID", async () => {
                const validData = {
                    ...commentFixtures.minimal.create,
                    forConnect: "123456789012345678",
                };

                await testValidation(createSchema, validData, true);
            });

            it("should reject invalid forConnect ID", async () => {
                const invalidData = {
                    ...commentFixtures.minimal.create,
                    forConnect: "invalid-id",
                };

                await testValidation(
                    createSchema,
                    invalidData,
                    false,
                    /valid ID/i,
                );
            });
        });

        describe("parentConnect field", () => {
            const createSchema = commentValidation.create(defaultParams);

            it("should accept optional parentConnect in create", async () => {
                const dataWithParent = {
                    ...commentFixtures.minimal.create,
                    parentConnect: "123456789012345678",
                };

                await testValidation(createSchema, dataWithParent, true);
            });

            it("should work without parentConnect", async () => {
                await testValidation(createSchema, commentFixtures.minimal.create, true);
            });

            it("should reject invalid parentConnect ID", async () => {
                const invalidData = {
                    ...commentFixtures.minimal.create,
                    parentConnect: "invalid-id",
                };

                await testValidation(
                    createSchema,
                    invalidData,
                    false,
                    /valid ID/i,
                );
            });
        });

        describe("translations", () => {
            const createSchema = commentValidation.create(defaultParams);
            const updateSchema = commentValidation.update(defaultParams);

            it("should accept optional translations in create", async () => {
                const dataWithTranslations = {
                    ...commentFixtures.minimal.create,
                    translationsCreate: [{
                        id: "123456789012345678",
                        language: "en",
                        text: "Test comment text",
                    }],
                };

                await testValidation(createSchema, dataWithTranslations, true);
            });

            it("should work without translations in create", async () => {
                await testValidation(createSchema, commentFixtures.minimal.create, true);
            });

            it("should accept multiple translations", async () => {
                await testValidation(
                    createSchema,
                    commentFixtures.edgeCases.multipleTranslations.create,
                    true,
                );
            });

            it("should validate translation text requirements", async () => {
                const scenarios = [
                    {
                        data: commentFixtures.invalid.invalidTranslationText.create,
                        shouldPass: false,
                        expectedError: /required/i,
                        description: "empty translation text",
                    },
                    {
                        data: commentFixtures.invalid.tooLongTranslationText.create,
                        shouldPass: false,
                        expectedError: /over the limit/,
                        description: "translation text too long",
                    },
                    {
                        data: commentFixtures.edgeCases.maxLengthTranslationText.create,
                        shouldPass: true,
                        description: "translation text at max length",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should accept all translation operations in update", async () => {
                const scenarios = [
                    {
                        data: {
                            ...commentFixtures.minimal.update,
                            translationsCreate: [{
                                id: "123456789012345678",
                                language: "en",
                                text: "New translation",
                            }],
                        },
                        shouldPass: true,
                        description: "create translations in update",
                    },
                    {
                        data: {
                            ...commentFixtures.minimal.update,
                            translationsUpdate: [{
                                id: "123456789012345678",
                                language: "en",
                                text: "Updated translation",
                            }],
                        },
                        shouldPass: true,
                        description: "update translations in update",
                    },
                    {
                        data: {
                            ...commentFixtures.minimal.update,
                            translationsDelete: ["123456789012345678"],
                        },
                        shouldPass: true,
                        description: "delete translations in update",
                    },
                ];

                await testValidationBatch(updateSchema, scenarios);
            });
        });
    });

    describe("update specific validations", () => {
        const updateSchema = commentValidation.update({ omitFields: [] });

        it("should allow minimal updates with just id", async () => {
            await testValidation(
                updateSchema,
                commentFixtures.minimal.update,
                true,
            );
        });

        it("should allow complex translation updates", async () => {
            await testValidation(
                updateSchema,
                commentFixtures.complete.update,
                true,
            );
        });

        it("should not allow createdFor updates", async () => {
            const updateData = {
                id: "123456789012345678",
                createdFor: CommentFor.Issue,
            };

            const result = await testValidation(updateSchema, updateData, true);
            expect(result).to.not.have.property("createdFor");
        });

        it("should not allow forConnect updates", async () => {
            const updateData = {
                id: "123456789012345678",
                forConnect: "123456789012345679",
            };

            const result = await testValidation(updateSchema, updateData, true);
            expect(result).to.not.have.property("forConnect");
        });

        it("should not allow parentConnect updates", async () => {
            const updateData = {
                id: "123456789012345678",
                parentConnect: "123456789012345679",
            };

            const result = await testValidation(updateSchema, updateData, true);
            expect(result).to.not.have.property("parentConnect");
        });
    });

    describe("edge cases", () => {
        const createSchema = commentValidation.create({ omitFields: [] });

        it("should handle all CommentFor types", async () => {
            const scenarios = commentFixtures.edgeCases.allCreatedForTypes.map(data => ({
                data,
                shouldPass: true,
                description: `CommentFor: ${data.createdFor}`,
            }));

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle nested comment structure", async () => {
            await testValidation(
                createSchema,
                commentFixtures.edgeCases.withParentComment.create,
                true,
            );
        });
    });

    describe("id validation", () => {
        const createSchema = commentValidation.create({ omitFields: [] });

        it("should accept valid Snowflake IDs", async () => {
            const validIds = [
                "123456789012345678",
                "999999999999999999",
                "100000000000000000",
            ];

            const scenarios = validIds.map(id => ({
                data: {
                    ...commentFixtures.minimal.create,
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
                    ...commentFixtures.minimal.create,
                    id,
                },
                shouldPass: false,
                expectedError: id === "" ? /required/i : /valid ID/i,
                description: `invalid ID: ${id}`,
            }));

            await testValidationBatch(createSchema, scenarios);
        });
    });
});
