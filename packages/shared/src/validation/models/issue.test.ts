import { describe, expect, it } from "vitest";
import { IssueFor } from "../../api/types.js";
import { issueFixtures } from "../../__test/fixtures/api/issueFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test/validationTestUtils.js";
import { issueValidation } from "./issue.js";

describe("issueValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        issueValidation,
        issueFixtures,
        "issue",
    );

    describe("specific field validations", () => {
        const defaultParams = { omitFields: [] };

        describe("issueFor field", () => {
            const createSchema = issueValidation.create(defaultParams);

            it("should accept all valid IssueFor enum values", async () => {
                const validValues = Object.values(IssueFor);
                const scenarios = validValues.map(value => ({
                    data: {
                        ...issueFixtures.minimal.create,
                        issueFor: value,
                    },
                    shouldPass: true,
                    description: `issueFor: ${value}`,
                }));

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject invalid IssueFor values", async () => {
                const scenarios = [
                    {
                        data: {
                            ...issueFixtures.minimal.create,
                            issueFor: "InvalidType",
                        },
                        shouldPass: false,
                        expectedError: /must be one of the following values/i,
                        description: "invalid enum value",
                    },
                    {
                        data: {
                            ...issueFixtures.minimal.create,
                            issueFor: 123,
                        },
                        shouldPass: false,
                        description: "number instead of enum",
                    },
                    {
                        data: {
                            ...issueFixtures.minimal.create,
                            issueFor: null,
                        },
                        shouldPass: false,
                        description: "null instead of enum",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should be required in create", async () => {
                await testValidation(
                    createSchema,
                    issueFixtures.invalid.missingRequired.create,
                    false,
                    /required/i,
                );
            });

            it("should not be present in update", async () => {
                const updateSchema = issueValidation.update(defaultParams);

                // Update schema should not include issueFor field
                const result = await updateSchema.validate(issueFixtures.minimal.update);
                expect(result).to.not.have.property("issueFor");
            });
        });

        describe("for relationship field", () => {
            const createSchema = issueValidation.create(defaultParams);

            it("should require forConnect in create", async () => {
                await testValidation(
                    createSchema,
                    issueFixtures.invalid.missingForConnect.create,
                    false,
                    /required/i,
                );
            });

            it("should reject invalid forConnect ID", async () => {
                await testValidation(
                    createSchema,
                    issueFixtures.invalid.invalidForConnect.create,
                    false,
                );
            });

            it("should accept valid forConnect with different IssueFor types", async () => {
                await testValidationBatch(
                    createSchema,
                    issueFixtures.edgeCases.allIssueForTypes.map(data => ({
                        data,
                        shouldPass: true,
                        description: `IssueFor: ${data.issueFor}`,
                    })),
                );
            });
        });

        describe("translations relationship", () => {
            const createSchema = issueValidation.create(defaultParams);
            const updateSchema = issueValidation.update(defaultParams);

            it("should be optional in both create and update", async () => {
                await testValidation(
                    createSchema,
                    issueFixtures.minimal.create,
                    true,
                );

                await testValidation(
                    updateSchema,
                    issueFixtures.minimal.update,
                    true,
                );
            });

            it("should accept translation creation", async () => {
                await testValidation(
                    createSchema,
                    issueFixtures.complete.create,
                    true,
                );
            });

            it("should accept translation updates", async () => {
                await testValidation(
                    updateSchema,
                    issueFixtures.complete.update,
                    true,
                );
            });

            it("should accept translation without description", async () => {
                await testValidation(
                    createSchema,
                    issueFixtures.edgeCases.translationWithoutDescription.create,
                    true,
                );
            });

            it("should accept multiple translations", async () => {
                await testValidation(
                    createSchema,
                    issueFixtures.edgeCases.multipleTranslations.create,
                    true,
                );
            });

            it("should accept all translation operations in update", async () => {
                await testValidation(
                    updateSchema,
                    issueFixtures.edgeCases.updateWithAllTranslationOperations.update,
                    true,
                );
            });

            it("should handle translation name requirements correctly", async () => {
                // Name is required in create translation
                await testValidation(
                    createSchema,
                    issueFixtures.invalid.invalidTranslationName.create,
                    false,
                    /required/i,
                );

                // Name is optional in update translation
                await testValidation(
                    updateSchema,
                    issueFixtures.edgeCases.updateOptionalTranslationName.update,
                    true,
                );
            });

            it("should handle long name and description", async () => {
                await testValidation(
                    createSchema,
                    issueFixtures.edgeCases.longNameAndDescription.create,
                    true,
                );
            });
        });

        describe("id field validation", () => {
            const createSchema = issueValidation.create(defaultParams);
            const updateSchema = issueValidation.update(defaultParams);

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
                            ...issueFixtures.minimal.create,
                            id: validId,
                        },
                        true,
                    );

                    await testValidation(
                        updateSchema,
                        {
                            ...issueFixtures.minimal.update,
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
                            ...issueFixtures.minimal.create,
                            id: invalidId,
                        },
                        false,
                    );

                    await testValidation(
                        updateSchema,
                        {
                            ...issueFixtures.minimal.update,
                            id: invalidId,
                        },
                        false,
                    );
                }
            });

            it("should be required in both create and update", async () => {
                await testValidation(
                    createSchema,
                    issueFixtures.invalid.missingRequired.create,
                    false,
                    /required/i,
                );

                await testValidation(
                    updateSchema,
                    issueFixtures.invalid.missingRequired.update,
                    false,
                    /required/i,
                );
            });
        });

        describe("update operation specifics", () => {
            const updateSchema = issueValidation.update(defaultParams);

            it("should not require issueFor or forConnect in update", async () => {
                await testValidation(
                    updateSchema,
                    issueFixtures.minimal.update,
                    true,
                );
            });

            it("should allow only translation operations in update", async () => {
                const result = await updateSchema.validate(issueFixtures.minimal.update);

                // Should only have id and translation operations
                expect(result).toHaveProperty("id");
                expect(result).to.not.have.property("issueFor");
                expect(result).to.not.have.property("forConnect");
            });
        });
    });
});
