import { describe, it, expect } from "vitest";
import { BookmarkFor } from "../../api/types.js";
import { bookmarkValidation } from "./bookmark.js";
import { bookmarkFixtures } from "./__test__/fixtures/bookmarkFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test__/validationTestUtils.js";

describe("bookmarkValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        bookmarkValidation,
        bookmarkFixtures,
        "bookmark",
    );

    describe("specific field validations", () => {
        const defaultParams = { omitFields: [] };

        describe("bookmarkFor field", () => {
            const createSchema = bookmarkValidation.create(defaultParams);

            it("should accept all valid BookmarkFor enum values", async () => {
                const validValues = Object.values(BookmarkFor);
                const scenarios = validValues.map(value => ({
                    data: {
                        ...bookmarkFixtures.minimal.create,
                        bookmarkFor: value,
                    },
                    shouldPass: true,
                    description: `bookmarkFor: ${value}`,
                }));

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject invalid BookmarkFor values", async () => {
                const scenarios = [
                    {
                        data: {
                            ...bookmarkFixtures.minimal.create,
                            bookmarkFor: "InvalidType",
                        },
                        shouldPass: false,
                        expectedError: /must be one of the following values/i,
                        description: "invalid enum value",
                    },
                    {
                        data: {
                            ...bookmarkFixtures.minimal.create,
                            bookmarkFor: 123,
                        },
                        shouldPass: false,
                        expectedError: /must be one of the following values/i,
                        description: "number instead of string",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should be required in create", async () => {
                const dataWithoutBookmarkFor = { ...bookmarkFixtures.minimal.create };
                delete dataWithoutBookmarkFor.bookmarkFor;

                await testValidation(
                    createSchema,
                    dataWithoutBookmarkFor,
                    false,
                    /required/i,
                );
            });
        });

        describe("forConnect field", () => {
            const createSchema = bookmarkValidation.create(defaultParams);

            it("should require forConnect in create", async () => {
                await testValidation(
                    createSchema,
                    bookmarkFixtures.invalid.missingForConnection.create,
                    false,
                    /required/i,
                );
            });

            it("should accept valid forConnect ID", async () => {
                const validData = {
                    ...bookmarkFixtures.minimal.create,
                    forConnect: "123456789012345678",
                };

                await testValidation(createSchema, validData, true);
            });

            it("should reject invalid forConnect ID", async () => {
                const invalidData = {
                    ...bookmarkFixtures.minimal.create,
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

        describe("list relationship fields", () => {
            const createSchema = bookmarkValidation.create(defaultParams);
            const updateSchema = bookmarkValidation.update(defaultParams);

            it("should accept optional listConnect in create", async () => {
                const dataWithListConnect = {
                    ...bookmarkFixtures.minimal.create,
                    listConnect: "123456789012345678",
                };

                await testValidation(createSchema, dataWithListConnect, true);
            });

            it("should accept optional listCreate in create", async () => {
                const dataWithListCreate = {
                    ...bookmarkFixtures.minimal.create,
                    listCreate: {
                        id: "123456789012345679",
                        label: "New List",
                    },
                };

                await testValidation(createSchema, dataWithListCreate, true);
            });

            it("should work without list relationship", async () => {
                await testValidation(createSchema, bookmarkFixtures.minimal.create, true);
            });

            it("should accept list operations in update", async () => {
                const scenarios = [
                    {
                        data: {
                            ...bookmarkFixtures.minimal.update,
                            listConnect: "123456789012345678",
                        },
                        shouldPass: true,
                        description: "listConnect in update",
                    },
                    {
                        data: {
                            ...bookmarkFixtures.minimal.update,
                            listUpdate: {
                                id: "123456789012345678",
                                label: "Updated Label",
                            },
                        },
                        shouldPass: true,
                        description: "listUpdate in update",
                    },
                ];

                await testValidationBatch(updateSchema, scenarios);
            });

            it("should enforce exclusivity between listConnect and listCreate", async () => {
                const dataWithBoth = {
                    ...bookmarkFixtures.minimal.create,
                    listConnect: "123456789012345680",
                    listCreate: {
                        id: "123456789012345681",
                        label: "New List",
                    },
                };

                await testValidation(
                    createSchema,
                    dataWithBoth,
                    false,
                    /Only one of the following fields can be present/i,
                );
            });
        });
    });

    describe("update specific validations", () => {
        const updateSchema = bookmarkValidation.update({ omitFields: [] });

        it("should allow minimal updates with just id", async () => {
            await testValidation(
                updateSchema,
                bookmarkFixtures.minimal.update,
                true,
            );
        });

        it("should allow partial updates", async () => {
            const scenarios = [
                {
                    data: {
                        id: "123456789012345678",
                        listConnect: "123456789012345679",
                    },
                    shouldPass: true,
                    description: "update only list connection",
                },
                {
                    data: {
                        id: "123456789012345678",
                        listUpdate: {
                            id: "123456789012345679",
                            label: "New Label",
                        },
                    },
                    shouldPass: true,
                    description: "update list details",
                },
            ];

            await testValidationBatch(updateSchema, scenarios);
        });

        it("should not allow bookmarkFor updates", async () => {
            const updateData = {
                id: "123456789012345678",
                bookmarkFor: BookmarkFor.Tag,
            };

            const result = await testValidation(updateSchema, updateData, true);
            expect(result).to.not.have.property("bookmarkFor");
        });

        it("should not allow forConnect updates", async () => {
            const updateData = {
                id: "123456789012345678",
                forConnect: "123456789012345679",
            };

            const result = await testValidation(updateSchema, updateData, true);
            expect(result).to.not.have.property("forConnect");
        });
    });

    describe("id validation", () => {
        const createSchema = bookmarkValidation.create({ omitFields: [] });

        it("should accept valid Snowflake IDs", async () => {
            const validIds = [
                "123456789012345678",
                "999999999999999999",
                "100000000000000000",
            ];

            const scenarios = validIds.map(id => ({
                data: {
                    ...bookmarkFixtures.minimal.create,
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
                    ...bookmarkFixtures.minimal.create,
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
