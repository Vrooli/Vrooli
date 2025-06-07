import { describe, it, expect } from "vitest";
import { reminderItemValidation } from "./reminderItem.js";
import { reminderItemFixtures } from "./__test__/fixtures/reminderItemFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test__/validationTestUtils.js";

describe("reminderItemValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        reminderItemValidation,
        reminderItemFixtures,
        "reminderItem",
    );

    describe("create specific validations", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = reminderItemValidation.create(defaultParams);

        it("should require id in create", async () => {
            const dataWithoutId = { ...reminderItemFixtures.minimal.create };
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
                reminderItemFixtures.invalid.missingRequired.create,
                false,
                /required/i,
            );
        });

        it("should require reminderConnect in create", async () => {
            await testValidation(
                createSchema,
                reminderItemFixtures.invalid.missingReminderConnect.create,
                false,
                /required/i,
            );
        });

        it("should allow minimal create with required fields", async () => {
            await testValidation(
                createSchema,
                reminderItemFixtures.minimal.create,
                true,
            );
        });

        it("should allow complete create with all fields", async () => {
            await testValidation(
                createSchema,
                reminderItemFixtures.complete.create,
                true,
            );
        });

        describe("name field", () => {
            it("should be required in create", async () => {
                await testValidation(
                    createSchema,
                    reminderItemFixtures.invalid.missingRequired.create,
                    false,
                    /required/i,
                );
            });

            it("should reject empty names", async () => {
                await testValidation(
                    createSchema,
                    reminderItemFixtures.invalid.nameTooShort.create,
                    false,
                    /required/i,
                );
            });

            it("should reject names that are too long", async () => {
                await testValidation(
                    createSchema,
                    reminderItemFixtures.invalid.nameTooLong.create,
                    false,
                    /over the limit/,
                );
            });

            it("should accept minimum length names (1 char)", async () => {
                await testValidation(
                    createSchema,
                    reminderItemFixtures.edgeCases.minimalName.create,
                    true,
                );
            });

            it("should accept maximum length names", async () => {
                await testValidation(
                    createSchema,
                    reminderItemFixtures.edgeCases.maxLengthName.create,
                    true,
                );
            });

            it("should trim whitespace from names", async () => {
                const result = await testValidation(
                    createSchema,
                    reminderItemFixtures.edgeCases.whitespaceStrings.create,
                    true,
                );
                expect(result.name).to.equal("Trimmed task name");
                expect(result.description).to.equal("Trimmed description");
            });
        });

        describe("description field", () => {
            it("should be optional", async () => {
                await testValidation(createSchema, reminderItemFixtures.minimal.create, true);
            });

            it("should accept valid descriptions", async () => {
                const scenarios = [
                    {
                        data: {
                            ...reminderItemFixtures.minimal.create,
                            description: "Short description",
                        },
                        shouldPass: true,
                        description: "short description",
                    },
                    {
                        data: {
                            ...reminderItemFixtures.minimal.create,
                            description: "A longer description with more details about the task",
                        },
                        shouldPass: true,
                        description: "longer description",
                    },
                    {
                        data: reminderItemFixtures.edgeCases.maxLengthDescription.create,
                        shouldPass: true,
                        description: "max length description",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject descriptions that are too long", async () => {
                await testValidation(
                    createSchema,
                    reminderItemFixtures.invalid.descriptionTooLong.create,
                    false,
                    /over the limit/,
                );
            });

            it("should handle empty descriptions", async () => {
                const result = await testValidation(
                    createSchema,
                    reminderItemFixtures.edgeCases.emptyDescription.create,
                    true,
                );
                // Empty strings should be converted to undefined
                expect(result.description).to.be.undefined;
            });
        });

        describe("dueDate field", () => {
            it("should be optional", async () => {
                await testValidation(createSchema, reminderItemFixtures.minimal.create, true);
            });

            it("should accept valid dates", async () => {
                const scenarios = [
                    {
                        data: reminderItemFixtures.edgeCases.pastDueDate.create,
                        shouldPass: true,
                        description: "past due date",
                    },
                    {
                        data: reminderItemFixtures.edgeCases.futureDueDate.create,
                        shouldPass: true,
                        description: "future due date",
                    },
                    {
                        data: {
                            ...reminderItemFixtures.minimal.create,
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
                    ...reminderItemFixtures.minimal.create,
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
                await testValidation(createSchema, reminderItemFixtures.minimal.create, true);
            });

            it("should accept valid indices", async () => {
                const scenarios = [
                    {
                        data: reminderItemFixtures.edgeCases.zeroIndex.create,
                        shouldPass: true,
                        description: "zero index",
                    },
                    {
                        data: reminderItemFixtures.edgeCases.largeIndex.create,
                        shouldPass: true,
                        description: "large index",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject negative indices", async () => {
                await testValidation(
                    createSchema,
                    reminderItemFixtures.invalid.negativeIndex.create,
                    false,
                    /Minimum value is 0/i,
                );
            });

            it("should reject float values", async () => {
                await testValidation(
                    createSchema,
                    reminderItemFixtures.edgeCases.floatIndex.create,
                    false,
                    /must be an integer/i,
                );
            });
        });

        describe("isComplete field", () => {
            it("should be optional", async () => {
                await testValidation(createSchema, reminderItemFixtures.minimal.create, true);
            });

            it("should accept boolean values", async () => {
                const scenarios = [
                    {
                        data: reminderItemFixtures.edgeCases.completedTask.create,
                        shouldPass: true,
                        description: "isComplete true",
                    },
                    {
                        data: reminderItemFixtures.edgeCases.incompleteTask.create,
                        shouldPass: true,
                        description: "isComplete false",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should convert string values to boolean", async () => {
                const result = await testValidation(
                    createSchema,
                    reminderItemFixtures.edgeCases.booleanConversions.create,
                    true,
                );
                expect(result.isComplete).to.equal(true);
            });
        });

        describe("reminderConnect field", () => {
            it("should be required", async () => {
                await testValidation(
                    createSchema,
                    reminderItemFixtures.invalid.missingReminderConnect.create,
                    false,
                    /required/i,
                );
            });

            it("should accept valid reminder ID", async () => {
                const validData = {
                    ...reminderItemFixtures.minimal.create,
                    reminderConnect: "123456789012345678",
                };

                await testValidation(createSchema, validData, true);
            });

            it("should reject invalid reminder ID", async () => {
                const invalidData = {
                    ...reminderItemFixtures.minimal.create,
                    reminderConnect: "invalid-id",
                };

                await testValidation(
                    createSchema,
                    invalidData,
                    false,
                    /valid ID/i,
                );
            });

            it("should accept different reminder IDs", async () => {
                await testValidation(
                    createSchema,
                    reminderItemFixtures.edgeCases.differentReminderId.create,
                    true,
                );
            });
        });
    });

    describe("update specific validations", () => {
        const defaultParams = { omitFields: [] };
        const updateSchema = reminderItemValidation.update(defaultParams);

        it("should require id in update", async () => {
            await testValidation(
                updateSchema,
                reminderItemFixtures.invalid.missingRequired.update,
                false,
                /required/i,
            );
        });

        it("should allow minimal updates with just id", async () => {
            await testValidation(
                updateSchema,
                reminderItemFixtures.minimal.update,
                true,
            );
        });

        it("should allow complete updates", async () => {
            await testValidation(
                updateSchema,
                reminderItemFixtures.complete.update,
                true,
            );
        });

        it("should not allow reminderConnect updates", async () => {
            const updateData = {
                id: "123456789012345678",
                reminderConnect: "123456789012345679",
            };

            const result = await testValidation(updateSchema, updateData, true);
            expect(result).to.not.have.property("reminderConnect");
        });

        describe("name field updates", () => {
            it("should be optional in update", async () => {
                await testValidation(updateSchema, reminderItemFixtures.minimal.update, true);
            });

            it("should accept valid name updates", async () => {
                const scenarios = [
                    {
                        data: {
                            ...reminderItemFixtures.minimal.update,
                            name: "Updated task name",
                        },
                        shouldPass: true,
                        description: "simple name update",
                    },
                    {
                        data: {
                            ...reminderItemFixtures.minimal.update,
                            name: "A", // Min length (1 char)
                        },
                        shouldPass: true,
                        description: "min length name update",
                    },
                    {
                        data: {
                            ...reminderItemFixtures.minimal.update,
                            name: "A".repeat(50), // Max length
                        },
                        shouldPass: true,
                        description: "max length name update",
                    },
                ];

                await testValidationBatch(updateSchema, scenarios);
            });
        });

        describe("isComplete field updates", () => {
            it("should accept boolean updates", async () => {
                const scenarios = [
                    {
                        data: reminderItemFixtures.edgeCases.completedTask.update,
                        shouldPass: true,
                        description: "mark as incomplete",
                    },
                    {
                        data: reminderItemFixtures.edgeCases.incompleteTask.update,
                        shouldPass: true,
                        description: "mark as complete",
                    },
                    {
                        data: reminderItemFixtures.edgeCases.booleanConversions.update,
                        shouldPass: true,
                        description: "string to boolean conversion",
                    },
                ];

                await testValidationBatch(updateSchema, scenarios);
            });
        });

        describe("other field updates", () => {
            it("should allow updating description", async () => {
                const dataWithDescription = {
                    ...reminderItemFixtures.minimal.update,
                    description: "New description",
                };

                await testValidation(updateSchema, dataWithDescription, true);
            });

            it("should allow updating dueDate", async () => {
                const dataWithDueDate = {
                    ...reminderItemFixtures.minimal.update,
                    dueDate: new Date("2025-06-01T00:00:00Z"),
                };

                await testValidation(updateSchema, dataWithDueDate, true);
            });

            it("should allow updating index", async () => {
                const dataWithIndex = {
                    ...reminderItemFixtures.minimal.update,
                    index: 5,
                };

                await testValidation(updateSchema, dataWithIndex, true);
            });
        });
    });

    describe("id validation", () => {
        const createSchema = reminderItemValidation.create({ omitFields: [] });

        it("should accept valid Snowflake IDs", async () => {
            const validIds = [
                "123456789012345678",
                "999999999999999999",
                "100000000000000000",
            ];

            const scenarios = validIds.map(id => ({
                data: {
                    ...reminderItemFixtures.minimal.create,
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
                    ...reminderItemFixtures.minimal.create,
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
        const createSchema = reminderItemValidation.create({ omitFields: [] });

        it("should handle type conversions for string fields", async () => {
            const dataWithNumberName = {
                ...reminderItemFixtures.minimal.create,
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
                ...reminderItemFixtures.minimal.create,
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

        it("should handle boolean conversions", async () => {
            const scenarios = [
                {
                    data: {
                        ...reminderItemFixtures.minimal.create,
                        isComplete: "yes",
                    },
                    shouldPass: true,
                    expectedValue: true,
                    description: "yes to true",
                },
                {
                    data: {
                        ...reminderItemFixtures.minimal.create,
                        isComplete: "no",
                    },
                    shouldPass: true,
                    expectedValue: false,
                    description: "no to false",
                },
                {
                    data: {
                        ...reminderItemFixtures.minimal.create,
                        isComplete: 1,
                    },
                    shouldPass: true,
                    expectedValue: true,
                    description: "1 to true",
                },
                {
                    data: {
                        ...reminderItemFixtures.minimal.create,
                        isComplete: 0,
                    },
                    shouldPass: true,
                    expectedValue: false,
                    description: "0 to false",
                },
            ];

            for (const scenario of scenarios) {
                const result = await testValidation(
                    createSchema,
                    scenario.data,
                    scenario.shouldPass,
                );
                expect(result.isComplete).to.equal(scenario.expectedValue);
            }
        });
    });

    describe("edge cases", () => {
        const createSchema = reminderItemValidation.create({ omitFields: [] });

        it("should handle tasks with all optional fields", async () => {
            const dataWithAllFields = {
                ...reminderItemFixtures.minimal.create,
                description: "Complete task description",
                dueDate: new Date("2025-01-01T00:00:00Z"),
                index: 5,
                isComplete: false,
            };

            await testValidation(createSchema, dataWithAllFields, true);
        });

        it("should handle string trimming correctly", async () => {
            const result = await testValidation(
                createSchema,
                reminderItemFixtures.edgeCases.whitespaceStrings.create,
                true,
            );

            expect(result.name).to.equal("Trimmed task name");
            expect(result.description).to.equal("Trimmed description");
        });

        it("should handle minimal name length", async () => {
            const result = await testValidation(
                createSchema,
                reminderItemFixtures.edgeCases.minimalName.create,
                true,
            );
            expect(result.name).to.equal("A");
            expect(result.name.length).to.equal(1);
        });
    });
});
