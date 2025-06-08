import { describe, expect, it } from "vitest";
import { bookmarkListFixtures } from "./__test/fixtures/bookmarkListFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test/validationTestUtils.js";
import { bookmarkListValidation } from "./bookmarkList.js";

describe("bookmarkListValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        bookmarkListValidation,
        bookmarkListFixtures,
        "bookmarkList",
    );

    describe("specific field validations", () => {
        const defaultParams = { omitFields: [] };

        describe("label field", () => {
            const createSchema = bookmarkListValidation.create(defaultParams);

            it("should accept valid label lengths", async () => {
                const scenarios = [
                    {
                        data: {
                            ...bookmarkListFixtures.minimal.create,
                            label: "A",
                        },
                        shouldPass: true,
                        description: "minimum length label",
                    },
                    {
                        data: {
                            ...bookmarkListFixtures.minimal.create,
                            label: "x".repeat(128),
                        },
                        shouldPass: true,
                        description: "maximum length label",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject label that exceeds maximum length", async () => {
                await testValidation(
                    createSchema,
                    {
                        ...bookmarkListFixtures.minimal.create,
                        label: "x".repeat(129),
                    },
                    false,
                    /1 character over the limit/,
                );
            });

            it("should trim whitespace from label", async () => {
                const result = await createSchema.validate({
                    ...bookmarkListFixtures.minimal.create,
                    label: "  Valid Label  ",
                });
                expect(result.label).to.equal("Valid Label");
            });

            it("should handle empty string label", async () => {
                await testValidation(
                    createSchema,
                    {
                        ...bookmarkListFixtures.minimal.create,
                        label: "",
                    },
                    false,
                    /required/i,
                );
            });
        });

        describe("id field validation", () => {
            const createSchema = bookmarkListValidation.create(defaultParams);

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
                            ...bookmarkListFixtures.minimal.create,
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
                            ...bookmarkListFixtures.minimal.create,
                            id: invalidId,
                        },
                        false,
                    );
                }
            });
        });

        describe("bookmark relationship operations", () => {
            const createSchema = bookmarkListValidation.create(defaultParams);
            const updateSchema = bookmarkListValidation.update(defaultParams);

            it("should accept bookmarks creation in create operation", async () => {
                await testValidation(
                    createSchema,
                    bookmarkListFixtures.edgeCases.multipleBookmarks.create,
                    true,
                );
            });

            it("should accept all bookmark operations in update", async () => {
                await testValidation(
                    updateSchema,
                    bookmarkListFixtures.edgeCases.updateWithAllBookmarkOperations.update,
                    true,
                );
            });
        });
    });
});
