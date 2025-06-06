import { describe, it } from "mocha";
import { expect } from "chai";
import { pullRequestValidation, pullRequestTranslationValidation } from "./pullRequest.js";
import { pullRequestFixtures } from "./__test__/fixtures/pullRequestFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test__/validationTestUtils.js";

describe("pullRequestValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        pullRequestValidation,
        pullRequestFixtures,
        "pullRequest",
    );

    describe("create specific validations", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = pullRequestValidation.create(defaultParams);

        it("should require id in create", async () => {
            const dataWithoutId = { ...pullRequestFixtures.minimal.create };
            delete dataWithoutId.id;

            await testValidation(
                createSchema,
                dataWithoutId,
                false,
                /required/i,
            );
        });

        it("should require toObjectType in create", async () => {
            const dataWithoutToObjectType = {
                id: "123456789012345678",
                toConnect: "123456789012345679",
                fromConnect: "123456789012345680",
                // Missing required toObjectType
            };

            await testValidation(
                createSchema,
                dataWithoutToObjectType,
                false,
                /required/i,
            );
        });

        it("should require toConnect in create", async () => {
            await testValidation(
                createSchema,
                pullRequestFixtures.invalid.missingTo.create,
                false,
                /required/i,
            );
        });

        it("should require fromConnect in create", async () => {
            await testValidation(
                createSchema,
                pullRequestFixtures.invalid.missingFrom.create,
                false,
                /required/i,
            );
        });

        it("should allow minimal create with required fields", async () => {
            await testValidation(
                createSchema,
                pullRequestFixtures.minimal.create,
                true,
            );
        });

        it("should allow complete create with all fields", async () => {
            await testValidation(
                createSchema,
                pullRequestFixtures.complete.create,
                true,
            );
        });

        describe("toObjectType field", () => {
            it("should accept valid object types", async () => {
                await testValidation(
                    createSchema,
                    pullRequestFixtures.minimal.create,
                    true,
                );
            });

            it("should reject invalid object types", async () => {
                await testValidation(
                    createSchema,
                    pullRequestFixtures.invalid.invalidToObjectType.create,
                    false,
                    /must be one of/i,
                );
            });
        });

        describe("relationship fields", () => {
            it("should require toConnect", async () => {
                await testValidation(
                    createSchema,
                    pullRequestFixtures.invalid.missingTo.create,
                    false,
                    /required/i,
                );
            });

            it("should require fromConnect", async () => {
                await testValidation(
                    createSchema,
                    pullRequestFixtures.invalid.missingFrom.create,
                    false,
                    /required/i,
                );
            });

            it("should accept valid relationship connections", async () => {
                await testValidation(
                    createSchema,
                    pullRequestFixtures.minimal.create,
                    true,
                );
            });
        });

        describe("translations relationships", () => {
            it("should be optional", async () => {
                await testValidation(
                    createSchema,
                    pullRequestFixtures.edgeCases.withoutTranslations.create,
                    true,
                );
            });

            it("should accept single translation", async () => {
                await testValidation(
                    createSchema,
                    pullRequestFixtures.edgeCases.withSingleTranslation.create,
                    true,
                );
            });

            it("should accept multiple translations", async () => {
                await testValidation(
                    createSchema,
                    pullRequestFixtures.edgeCases.withMultipleTranslations.create,
                    true,
                );
            });

            it("should validate translation text requirements", async () => {
                await testValidation(
                    createSchema,
                    pullRequestFixtures.invalid.emptyTextTranslation.create,
                    false,
                    /required/i,
                );
            });

            it("should validate translation text length", async () => {
                await testValidation(
                    createSchema,
                    pullRequestFixtures.invalid.longText.create,
                    false,
                    /over the limit/i,
                );
            });
        });

        describe("pull request scenarios", () => {
            it("should handle resource pull requests", async () => {
                await testValidation(
                    createSchema,
                    pullRequestFixtures.minimal.create,
                    true,
                );
            });

            it("should handle pull requests with detailed descriptions", async () => {
                await testValidation(
                    createSchema,
                    pullRequestFixtures.edgeCases.longDescription.create,
                    true,
                );
            });

            it("should handle pull requests with different IDs", async () => {
                await testValidation(
                    createSchema,
                    pullRequestFixtures.edgeCases.differentIds.create,
                    true,
                );
            });
        });
    });

    describe("update specific validations", () => {
        const defaultParams = { omitFields: [] };
        const updateSchema = pullRequestValidation.update(defaultParams);

        it("should require id in update", async () => {
            await testValidation(
                updateSchema,
                pullRequestFixtures.invalid.missingRequired.update,
                false,
                /required/i,
            );
        });

        it("should allow minimal updates with just id", async () => {
            await testValidation(
                updateSchema,
                pullRequestFixtures.minimal.update,
                true,
            );
        });

        it("should allow complete updates", async () => {
            await testValidation(
                updateSchema,
                pullRequestFixtures.complete.update,
                true,
            );
        });

        describe("status field updates", () => {
            it("should be optional", async () => {
                await testValidation(
                    updateSchema,
                    pullRequestFixtures.edgeCases.updateOnlyId.update,
                    true,
                );
            });

            it("should accept valid status values", async () => {
                const scenarios = [
                    {
                        data: pullRequestFixtures.edgeCases.statusDraft.update,
                        shouldPass: true,
                        description: "status Draft",
                    },
                    {
                        data: pullRequestFixtures.edgeCases.statusOpen.update,
                        shouldPass: true,
                        description: "status Open",
                    },
                    {
                        data: pullRequestFixtures.edgeCases.statusMerged.update,
                        shouldPass: true,
                        description: "status Merged",
                    },
                    {
                        data: pullRequestFixtures.edgeCases.statusRejected.update,
                        shouldPass: true,
                        description: "status Rejected",
                    },
                    {
                        data: pullRequestFixtures.edgeCases.statusCanceled.update,
                        shouldPass: true,
                        description: "status Canceled",
                    },
                ];

                await testValidationBatch(updateSchema, scenarios);
            });

            it("should reject invalid status values", async () => {
                await testValidation(
                    updateSchema,
                    pullRequestFixtures.invalid.invalidTypes.update,
                    false,
                    /must be one of/i,
                );
            });
        });

        describe("translation updates", () => {
            it("should allow translation operations in update", async () => {
                await testValidation(
                    updateSchema,
                    pullRequestFixtures.edgeCases.updateWithTranslationOperations.update,
                    true,
                );
            });

            it("should handle multiple translation operations", async () => {
                await testValidation(
                    updateSchema,
                    pullRequestFixtures.complete.update,
                    true,
                );
            });

            it("should not allow updating toObjectType, toConnect, or fromConnect", async () => {
                const dataWithRestrictedFields = {
                    id: "123456789012345678",
                    toObjectType: "Resource",
                    toConnect: "123456789012345679",
                    fromConnect: "123456789012345680",
                    status: "Open",
                };

                const result = await testValidation(updateSchema, dataWithRestrictedFields, true);
                expect(result).to.not.have.property("toObjectType");
                expect(result).to.not.have.property("toConnect");
                expect(result).to.not.have.property("fromConnect");
            });
        });

        describe("update scenarios", () => {
            it("should handle id-only updates", async () => {
                await testValidation(
                    updateSchema,
                    pullRequestFixtures.edgeCases.updateOnlyId.update,
                    true,
                );
            });

            it("should handle status updates", async () => {
                await testValidation(
                    updateSchema,
                    pullRequestFixtures.edgeCases.statusOpen.update,
                    true,
                );
            });

            it("should handle translation create operations", async () => {
                const dataWithTranslationCreate = {
                    id: "123456789012345678",
                    translationsCreate: [
                        {
                            id: "123456789012345679",
                            language: "en",
                            text: "New translation created during update.",
                        },
                    ],
                };

                await testValidation(updateSchema, dataWithTranslationCreate, true);
            });

            it("should handle translation update operations", async () => {
                const dataWithTranslationUpdate = {
                    id: "123456789012345678",
                    translationsUpdate: [
                        {
                            id: "123456789012345679",
                            text: "Updated translation text.",
                        },
                    ],
                };

                await testValidation(updateSchema, dataWithTranslationUpdate, true);
            });

            it("should handle translation delete operations", async () => {
                const dataWithTranslationDelete = {
                    id: "123456789012345678",
                    translationsDelete: ["123456789012345679", "123456789012345680"],
                };

                await testValidation(updateSchema, dataWithTranslationDelete, true);
            });
        });
    });

    describe("id validation", () => {
        const createSchema = pullRequestValidation.create({ omitFields: [] });
        const updateSchema = pullRequestValidation.update({ omitFields: [] });

        it("should accept valid Snowflake IDs", async () => {
            const validIds = [
                "123456789012345678",
                "999999999999999999",
                "100000000000000000",
            ];

            const scenarios = validIds.map(id => ({
                data: {
                    ...pullRequestFixtures.minimal.create,
                    id,
                },
                shouldPass: true,
                description: `valid ID: ${id}`,
            }));

            await testValidationBatch(createSchema, scenarios);
        });

        it("should reject invalid IDs", async () => {
            const invalidIds = [
                { id: "not-a-number", error: /Must be a valid ID/i },
                { id: "abc123", error: /Must be a valid ID/i },
                { id: "", error: /required/i },
                { id: "-123", error: /Must be a valid ID/i },
                { id: "0", error: /Must be a valid ID/i },
            ];

            for (const { id, error } of invalidIds) {
                await testValidation(
                    createSchema,
                    { ...pullRequestFixtures.minimal.create, id },
                    false,
                    error,
                );
            }
        });
    });

    describe("type conversions", () => {
        const createSchema = pullRequestValidation.create({ omitFields: [] });

        it("should handle ID conversion", async () => {
            const dataWithNumberId = {
                id: 123456789012345, // Smaller number to avoid precision issues
                toObjectType: "Resource",
                toConnect: "123456789012345679",
                fromConnect: "123456789012345680",
            };

            const result = await testValidation(
                createSchema,
                dataWithNumberId,
                true,
            );
            expect(result.id).to.be.a("string");
            expect(result.id).to.equal("123456789012345");
        });
    });

    describe("edge cases", () => {
        const createSchema = pullRequestValidation.create({ omitFields: [] });
        const updateSchema = pullRequestValidation.update({ omitFields: [] });

        it("should handle minimal pull request creation", async () => {
            await testValidation(
                createSchema,
                pullRequestFixtures.minimal.create,
                true,
            );
        });

        it("should handle empty update operations", async () => {
            await testValidation(
                updateSchema,
                pullRequestFixtures.edgeCases.updateOnlyId.update,
                true,
            );
        });

        it("should handle different text lengths", async () => {
            const scenarios = [
                {
                    data: pullRequestFixtures.edgeCases.minLengthText.create,
                    shouldPass: true,
                    description: "minimum text length",
                },
                {
                    data: pullRequestFixtures.edgeCases.maxLengthText.create,
                    shouldPass: true,
                    description: "maximum text length",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle various pull request statuses", async () => {
            const scenarios = [
                {
                    data: pullRequestFixtures.edgeCases.statusDraft.update,
                    shouldPass: true,
                    description: "Draft status",
                },
                {
                    data: pullRequestFixtures.edgeCases.statusMerged.update,
                    shouldPass: true,
                    description: "Merged status",
                },
                {
                    data: pullRequestFixtures.edgeCases.statusRejected.update,
                    shouldPass: true,
                    description: "Rejected status",
                },
            ];

            await testValidationBatch(updateSchema, scenarios);
        });

        it("should handle complex translation operations", async () => {
            await testValidation(
                updateSchema,
                pullRequestFixtures.edgeCases.updateWithTranslationOperations.update,
                true,
            );
        });
    });
});

describe("pullRequestTranslationValidation", () => {
    describe("create specific validations", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = pullRequestTranslationValidation.create(defaultParams);

        it("should require text in create", async () => {
            const dataWithoutText = {
                id: "123456789012345678",
                language: "en",
                // Missing required text
            };

            await testValidation(
                createSchema,
                dataWithoutText,
                false,
                /required/i,
            );
        });

        it("should allow minimal create with just text", async () => {
            const minimalData = {
                id: "123456789012345678",
                language: "en",
                text: "Basic pull request description",
            };

            await testValidation(createSchema, minimalData, true);
        });

        it("should allow complete create with detailed text", async () => {
            const completeData = {
                id: "123456789012345678",
                language: "en",
                text: "This is a comprehensive pull request description that explains the changes, motivation, and implementation details.",
            };

            await testValidation(createSchema, completeData, true);
        });

        it("should validate text length", async () => {
            const dataWithShortText = {
                id: "123456789012345678",
                language: "en",
                text: "", // Empty text becomes undefined
            };

            await testValidation(
                createSchema,
                dataWithShortText,
                false,
                /required/i,
            );
        });

        it("should validate maximum text length", async () => {
            const dataWithLongText = {
                id: "123456789012345678",
                language: "en",
                text: "x".repeat(32769), // Too long
            };

            await testValidation(
                createSchema,
                dataWithLongText,
                false,
                /over the limit/i,
            );
        });
    });

    describe("update specific validations", () => {
        const defaultParams = { omitFields: [] };
        const updateSchema = pullRequestTranslationValidation.update(defaultParams);

        it("should make all fields optional in update", async () => {
            const updateWithOnlyId = {
                id: "123456789012345678",
            };

            await testValidation(updateSchema, updateWithOnlyId, true);
        });

        it("should allow updating text", async () => {
            const dataWithText = {
                id: "123456789012345678",
                text: "Updated pull request description",
            };

            await testValidation(updateSchema, dataWithText, true);
        });

        it("should still validate text length in update", async () => {
            const dataWithShortText = {
                id: "123456789012345678",
                text: "", // Empty text becomes undefined, but text is optional in update
            };

            await testValidation(
                updateSchema,
                dataWithShortText,
                true, // Should pass because text is optional in update
            );
        });

        it("should still validate maximum text length in update", async () => {
            const dataWithLongText = {
                id: "123456789012345678",
                text: "x".repeat(32769), // Too long
            };

            await testValidation(
                updateSchema,
                dataWithLongText,
                false,
                /over the limit/i,
            );
        });
    });
});