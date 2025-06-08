import { describe, expect, it } from "vitest";
import { reminderFixtures } from "./__test/fixtures/reminderFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test/validationTestUtils.js";
import { reminderValidation } from "./reminder.js";

describe("reminderValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        reminderValidation,
        reminderFixtures,
        "reminder",
    );

    describe("create specific validations", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = reminderValidation.create(defaultParams);

        it("should require id in create", async () => {
            const dataWithoutId = { ...reminderFixtures.minimal.create };
            delete dataWithoutId.id;

            await testValidation(
                createSchema,
                dataWithoutId,
                false,
                /required/i,
            );
        });

        it("should require name in create", async () => {
            await testValidation(
                createSchema,
                reminderFixtures.invalid.missingRequired.create,
                false,
                /required/i,
            );
        });

        it("should allow minimal create with required fields", async () => {
            await testValidation(
                createSchema,
                reminderFixtures.minimal.create,
                true,
            );
        });

        it("should allow complete create with all fields", async () => {
            await testValidation(
                createSchema,
                reminderFixtures.complete.create,
                true,
            );
        });

        describe("name field", () => {
            it("should be required in create", async () => {
                await testValidation(
                    createSchema,
                    reminderFixtures.invalid.missingRequired.create,
                    false,
                    /required/i,
                );
            });

            it("should reject empty names", async () => {
                await testValidation(
                    createSchema,
                    reminderFixtures.invalid.nameTooShort.create,
                    false,
                    /required/i,
                );
            });

            it("should reject names that are too long", async () => {
                await testValidation(
                    createSchema,
                    reminderFixtures.invalid.nameTooLong.create,
                    false,
                    /over the limit/,
                );
            });

            it("should accept minimum length names", async () => {
                await testValidation(
                    createSchema,
                    reminderFixtures.edgeCases.minimalName.create,
                    true,
                );
            });

            it("should accept maximum length names", async () => {
                await testValidation(
                    createSchema,
                    reminderFixtures.edgeCases.maxLengthName.create,
                    true,
                );
            });

            it("should trim whitespace from names", async () => {
                const result = await testValidation(
                    createSchema,
                    reminderFixtures.edgeCases.whitespaceStrings.create,
                    true,
                );
                expect(result.name).to.equal("Trimmed name");
                expect(result.description).to.equal("Trimmed description");
            });
        });

        describe("description field", () => {
            it("should be optional", async () => {
                await testValidation(createSchema, reminderFixtures.minimal.create, true);
            });

            it("should accept valid descriptions", async () => {
                const scenarios = [
                    {
                        data: {
                            ...reminderFixtures.minimal.create,
                            description: "Short description",
                        },
                        shouldPass: true,
                        description: "short description",
                    },
                    {
                        data: {
                            ...reminderFixtures.minimal.create,
                            description: "A longer description with more details about the reminder",
                        },
                        shouldPass: true,
                        description: "longer description",
                    },
                    {
                        data: reminderFixtures.edgeCases.maxLengthDescription.create,
                        shouldPass: true,
                        description: "max length description",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject descriptions that are too long", async () => {
                await testValidation(
                    createSchema,
                    reminderFixtures.invalid.descriptionTooLong.create,
                    false,
                    /over the limit/,
                );
            });

            it("should handle empty descriptions", async () => {
                const result = await testValidation(
                    createSchema,
                    reminderFixtures.edgeCases.emptyDescription.create,
                    true,
                );
                // Empty strings should be converted to undefined
                expect(result.description).to.be.undefined;
            });
        });

        describe("dueDate field", () => {
            it("should be optional", async () => {
                await testValidation(createSchema, reminderFixtures.minimal.create, true);
            });

            it("should accept valid dates", async () => {
                const scenarios = [
                    {
                        data: reminderFixtures.edgeCases.pastDueDate.create,
                        shouldPass: true,
                        description: "past due date",
                    },
                    {
                        data: reminderFixtures.edgeCases.futureDueDate.create,
                        shouldPass: true,
                        description: "future due date",
                    },
                    {
                        data: {
                            ...reminderFixtures.minimal.create,
                            dueDate: new Date(),
                        },
                        shouldPass: true,
                        description: "current date",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject invalid dates", async () => {
                const invalidData = {
                    ...reminderFixtures.minimal.create,
                    dueDate: "not-a-date",
                };

                await testValidation(
                    createSchema,
                    invalidData,
                    false,
                    /type, but the final value was/i,
                );
            });
        });

        describe("index field", () => {
            it("should be optional", async () => {
                await testValidation(createSchema, reminderFixtures.minimal.create, true);
            });

            it("should accept valid indices", async () => {
                const scenarios = [
                    {
                        data: reminderFixtures.edgeCases.zeroIndex.create,
                        shouldPass: true,
                        description: "zero index",
                    },
                    {
                        data: reminderFixtures.edgeCases.largeIndex.create,
                        shouldPass: true,
                        description: "large index",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject negative indices", async () => {
                await testValidation(
                    createSchema,
                    reminderFixtures.invalid.negativeIndex.create,
                    false,
                    /Minimum value is 0/i,
                );
            });

            it("should reject float values", async () => {
                await testValidation(
                    createSchema,
                    reminderFixtures.edgeCases.floatIndex.create,
                    false,
                    /must be an integer/i,
                );
            });
        });

        describe("reminderList relationships", () => {
            it("should allow connecting to existing reminder list", async () => {
                await testValidation(
                    createSchema,
                    reminderFixtures.edgeCases.withReminderList.create,
                    true,
                );
            });

            it("should allow creating a new reminder list", async () => {
                await testValidation(
                    createSchema,
                    reminderFixtures.edgeCases.createReminderList.create,
                    true,
                );
            });

            it("should reject having both connect and create for reminder list", async () => {
                await testValidation(
                    createSchema,
                    reminderFixtures.invalid.conflictingListConnections.create,
                    false,
                    /Only one of the following fields can be present/i,
                );
            });
        });

        describe("reminderItems relationships", () => {
            it("should allow creating reminder items", async () => {
                await testValidation(
                    createSchema,
                    reminderFixtures.complete.create,
                    true,
                );
            });

            it("should allow multiple reminder items", async () => {
                await testValidation(
                    createSchema,
                    reminderFixtures.edgeCases.multipleReminderItems.create,
                    true,
                );
            });

            it("should validate reminder items", async () => {
                await testValidation(
                    createSchema,
                    reminderFixtures.invalid.invalidReminderItem.create,
                    false,
                    /required/i,
                );
            });
        });
    });

    describe("update specific validations", () => {
        const defaultParams = { omitFields: [] };
        const updateSchema = reminderValidation.update(defaultParams);

        it("should require id in update", async () => {
            await testValidation(
                updateSchema,
                reminderFixtures.invalid.missingRequired.update,
                false,
                /required/i,
            );
        });

        it("should allow minimal updates with just id", async () => {
            await testValidation(
                updateSchema,
                reminderFixtures.minimal.update,
                true,
            );
        });

        it("should allow complete updates", async () => {
            await testValidation(
                updateSchema,
                reminderFixtures.complete.update,
                true,
            );
        });

        describe("name field updates", () => {
            it("should be optional in update", async () => {
                await testValidation(updateSchema, reminderFixtures.minimal.update, true);
            });

            it("should accept valid name updates", async () => {
                const scenarios = [
                    {
                        data: {
                            ...reminderFixtures.minimal.update,
                            name: "Updated reminder name",
                        },
                        shouldPass: true,
                        description: "simple name update",
                    },
                    {
                        data: {
                            ...reminderFixtures.minimal.update,
                            name: "A".repeat(50), // Max length
                        },
                        shouldPass: true,
                        description: "max length name update",
                    },
                ];

                await testValidationBatch(updateSchema, scenarios);
            });
        });

        describe("reminderItems operations in update", () => {
            it("should allow creating new reminder items", async () => {
                const dataWithCreate = {
                    ...reminderFixtures.minimal.update,
                    reminderItemsCreate: [{
                        id: "123456789012345678",
                        name: "New item",
                        reminderConnect: "123456789012345679",
                    }],
                };

                await testValidation(updateSchema, dataWithCreate, true);
            });

            it("should allow updating existing reminder items", async () => {
                const dataWithUpdate = {
                    ...reminderFixtures.minimal.update,
                    reminderItemsUpdate: [{
                        id: "123456789012345678",
                        isComplete: true,
                    }],
                };

                await testValidation(updateSchema, dataWithUpdate, true);
            });

            it("should allow deleting reminder items", async () => {
                const dataWithDelete = {
                    ...reminderFixtures.minimal.update,
                    reminderItemsDelete: ["123456789012345678", "123456789012345679"],
                };

                await testValidation(updateSchema, dataWithDelete, true);
            });

            it("should allow all operations together", async () => {
                await testValidation(
                    updateSchema,
                    reminderFixtures.complete.update,
                    true,
                );
            });
        });

        describe("reminderList operations in update", () => {
            it("should allow connecting to a reminder list", async () => {
                const dataWithConnect = {
                    ...reminderFixtures.minimal.update,
                    reminderListConnect: "123456789012345678",
                };

                await testValidation(updateSchema, dataWithConnect, true);
            });

            it("should allow creating a reminder list", async () => {
                const dataWithCreate = {
                    ...reminderFixtures.minimal.update,
                    reminderListCreate: {
                        id: "123456789012345678",
                    },
                };

                await testValidation(updateSchema, dataWithCreate, true);
            });
        });
    });

    describe("id validation", () => {
        const createSchema = reminderValidation.create({ omitFields: [] });

        it("should accept valid Snowflake IDs", async () => {
            const validIds = [
                "123456789012345678",
                "999999999999999999",
                "100000000000000000",
            ];

            const scenarios = validIds.map(id => ({
                data: {
                    ...reminderFixtures.minimal.create,
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
                    ...reminderFixtures.minimal.create,
                    id,
                },
                shouldPass: false,
                expectedError: id === "" ? /required/i : /valid ID/i,
                description: `invalid ID: ${id}`,
            }));

            await testValidationBatch(createSchema, scenarios);
        });
    });

    describe("type conversions", () => {
        const createSchema = reminderValidation.create({ omitFields: [] });

        it("should handle type conversions for string fields", async () => {
            const dataWithNumberName = {
                ...reminderFixtures.minimal.create,
                name: 123, // Number instead of string
            };

            // The name field converts numbers to strings
            const result = await testValidation(
                createSchema,
                dataWithNumberName,
                true,
            );
            expect(result.name).to.equal("123");
        });

        it("should handle type conversions for description", async () => {
            const dataWithNumberDescription = {
                ...reminderFixtures.minimal.create,
                description: 456, // Number instead of string
            };

            // The description field converts numbers to strings
            const result = await testValidation(
                createSchema,
                dataWithNumberDescription,
                true,
            );
            expect(result.description).to.equal("456");
        });
    });

    describe("edge cases", () => {
        const createSchema = reminderValidation.create({ omitFields: [] });

        it("should handle reminders with all optional fields", async () => {
            const dataWithAllFields = {
                ...reminderFixtures.minimal.create,
                description: "Complete reminder",
                dueDate: new Date("2025-01-01T00:00:00Z"),
                index: 5,
            };

            await testValidation(createSchema, dataWithAllFields, true);
        });

        it("should handle complex nested creation", async () => {
            await testValidation(
                createSchema,
                reminderFixtures.edgeCases.createReminderList.create,
                true,
            );
        });

        it("should handle string trimming correctly", async () => {
            const result = await testValidation(
                createSchema,
                reminderFixtures.edgeCases.whitespaceStrings.create,
                true,
            );

            expect(result.name).to.equal("Trimmed name");
            expect(result.description).to.equal("Trimmed description");
        });
    });
});
