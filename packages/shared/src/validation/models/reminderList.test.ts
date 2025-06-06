import { describe, it } from "mocha";
import { expect } from "chai";
import { reminderListValidation } from "./reminderList.js";
import { reminderListFixtures } from "./__test__/fixtures/reminderListFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test__/validationTestUtils.js";

describe("reminderListValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        reminderListValidation,
        reminderListFixtures,
        "reminderList",
    );

    describe("create specific validations", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = reminderListValidation.create(defaultParams);

        it("should require id in create", async () => {
            const dataWithoutId = { ...reminderListFixtures.minimal.create };
            delete dataWithoutId.id;

            await testValidation(
                createSchema,
                dataWithoutId,
                false,
                /required/i,
            );
        });

        it("should allow minimal create with just id", async () => {
            await testValidation(
                createSchema,
                reminderListFixtures.minimal.create,
                true,
            );
        });

        it("should allow complete create with reminders", async () => {
            await testValidation(
                createSchema,
                reminderListFixtures.complete.create,
                true,
            );
        });

        describe("reminders field", () => {
            it("should be optional", async () => {
                await testValidation(
                    createSchema,
                    reminderListFixtures.edgeCases.emptyList.create,
                    true,
                );
            });

            it("should accept single reminder", async () => {
                await testValidation(
                    createSchema,
                    reminderListFixtures.edgeCases.singleReminder.create,
                    true,
                );
            });

            it("should accept multiple reminders", async () => {
                await testValidation(
                    createSchema,
                    reminderListFixtures.edgeCases.manyReminders.create,
                    true,
                );
            });

            it("should validate reminder objects", async () => {
                await testValidation(
                    createSchema,
                    reminderListFixtures.invalid.invalidReminders.create,
                    false,
                    /required/i,
                );
            });

            it("should reject non-array values", async () => {
                const invalidData = {
                    id: "123456789012345678",
                    remindersCreate: "not-an-array",
                };

                await testValidation(
                    createSchema,
                    invalidData,
                    false,
                    /must be a \`array\` type/i,
                );
            });

            it("should allow reminders with nested items", async () => {
                await testValidation(
                    createSchema,
                    reminderListFixtures.edgeCases.nestedReminderItems.create,
                    true,
                );
            });

            it("should allow circular reference to parent list", async () => {
                await testValidation(
                    createSchema,
                    reminderListFixtures.edgeCases.circularListReference.create,
                    true,
                );
            });

            it("should allow reminders with all fields", async () => {
                await testValidation(
                    createSchema,
                    reminderListFixtures.edgeCases.reminderWithAllFields.create,
                    true,
                );
            });
        });

        describe("create operation limitations", () => {
            it("should only allow remindersCreate in create operation", async () => {
                const dataWithUpdate = {
                    id: "123456789012345678",
                    remindersUpdate: [{
                        id: "123456789012345679",
                        name: "Can't update in create",
                    }],
                };

                const result = await testValidation(createSchema, dataWithUpdate, true);
                expect(result).to.not.have.property("remindersUpdate");
            });

            it("should not allow remindersDelete in create operation", async () => {
                const dataWithDelete = {
                    id: "123456789012345678",
                    remindersDelete: ["123456789012345679"],
                };

                const result = await testValidation(createSchema, dataWithDelete, true);
                expect(result).to.not.have.property("remindersDelete");
            });
        });
    });

    describe("update specific validations", () => {
        const defaultParams = { omitFields: [] };
        const updateSchema = reminderListValidation.update(defaultParams);

        it("should require id in update", async () => {
            await testValidation(
                updateSchema,
                reminderListFixtures.invalid.missingRequired.update,
                false,
                /required/i,
            );
        });

        it("should allow minimal updates with just id", async () => {
            await testValidation(
                updateSchema,
                reminderListFixtures.minimal.update,
                true,
            );
        });

        it("should allow complete updates", async () => {
            await testValidation(
                updateSchema,
                reminderListFixtures.complete.update,
                true,
            );
        });

        describe("reminders operations", () => {
            it("should allow creating new reminders", async () => {
                const dataWithCreate = {
                    id: "123456789012345678",
                    remindersCreate: [{
                        id: "123456789012345679",
                        name: "New reminder",
                    }],
                };

                await testValidation(updateSchema, dataWithCreate, true);
            });

            it("should allow updating existing reminders", async () => {
                const dataWithUpdate = {
                    id: "123456789012345678",
                    remindersUpdate: [{
                        id: "123456789012345679",
                        name: "Updated name",
                        description: "Updated description",
                    }],
                };

                await testValidation(updateSchema, dataWithUpdate, true);
            });

            it("should allow deleting reminders", async () => {
                const dataWithDelete = {
                    id: "123456789012345678",
                    remindersDelete: ["123456789012345679", "123456789012345680"],
                };

                await testValidation(updateSchema, dataWithDelete, true);
            });

            it("should allow all operations together", async () => {
                await testValidation(
                    updateSchema,
                    reminderListFixtures.edgeCases.complexUpdate.update,
                    true,
                );
            });

            it("should allow update-only operations", async () => {
                await testValidation(
                    updateSchema,
                    reminderListFixtures.edgeCases.updateOnlyOperations.update,
                    true,
                );
            });

            it("should allow delete-only operations", async () => {
                await testValidation(
                    updateSchema,
                    reminderListFixtures.edgeCases.deleteOnlyOperations.update,
                    true,
                );
            });

            it("should validate reminder objects in create", async () => {
                await testValidation(
                    updateSchema,
                    reminderListFixtures.invalid.invalidReminders.update,
                    false,
                    /required/i,
                );
            });

            it("should validate delete array contains valid IDs", async () => {
                await testValidation(
                    updateSchema,
                    reminderListFixtures.invalid.invalidReminderDelete.update,
                    false,
                    /valid ID/i,
                );
            });
        });
    });

    describe("id validation", () => {
        const createSchema = reminderListValidation.create({ omitFields: [] });

        it("should accept valid Snowflake IDs", async () => {
            const validIds = [
                "123456789012345678",
                "999999999999999999",
                "100000000000000000",
            ];

            const scenarios = validIds.map(id => ({
                data: {
                    ...reminderListFixtures.minimal.create,
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
                { id: "abc123", error: /Must be a valid ID/i }, // Contains letters
                { id: "", error: /required/i }, // Empty string
                { id: "-123", error: /Must be a valid ID/i }, // Negative number
                { id: "0", error: /Must be a valid ID/i }, // Zero is not valid
            ];

            for (const { id, error } of invalidIds) {
                await testValidation(
                    createSchema,
                    { ...reminderListFixtures.minimal.create, id },
                    false,
                    error,
                );
            }
        });
    });

    describe("type conversions", () => {
        const createSchema = reminderListValidation.create({ omitFields: [] });

        it("should handle ID conversion", async () => {
            const dataWithNumberId = {
                id: 123456789012345, // Smaller number to avoid precision issues
            };

            // Should convert to string
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
        const createSchema = reminderListValidation.create({ omitFields: [] });
        const updateSchema = reminderListValidation.update({ omitFields: [] });

        it("should handle empty reminder list creation", async () => {
            await testValidation(
                createSchema,
                reminderListFixtures.edgeCases.emptyList.create,
                true,
            );
        });

        it("should handle empty update operations", async () => {
            await testValidation(
                updateSchema,
                reminderListFixtures.edgeCases.emptyList.update,
                true,
            );
        });

        it("should handle reminder lists with many reminders", async () => {
            const result = await testValidation(
                createSchema,
                reminderListFixtures.edgeCases.manyReminders.create,
                true,
            );
            expect(result.remindersCreate).to.have.lengthOf(10);
        });

        it("should handle complex nested structures", async () => {
            const result = await testValidation(
                createSchema,
                reminderListFixtures.edgeCases.nestedReminderItems.create,
                true,
            );
            expect(result.remindersCreate[0].reminderItemsCreate).to.have.lengthOf(5);
        });

        it("should properly validate nested reminder validation", async () => {
            const dataWithInvalidNestedReminder = {
                id: "123456789012345678",
                remindersCreate: [{
                    id: "123456789012345679",
                    name: "", // Empty name should fail
                }],
            };

            await testValidation(
                createSchema,
                dataWithInvalidNestedReminder,
                false,
                /required/i,
            );
        });
    });

    describe("relationship validation", () => {
        const createSchema = reminderListValidation.create({ omitFields: [] });

        it("should validate reminders have proper structure", async () => {
            const scenarios = [
                {
                    data: {
                        id: "123456789012345678",
                        remindersCreate: [{
                            id: "123456789012345679",
                            name: "Valid reminder",
                            description: "With description",
                        }],
                    },
                    shouldPass: true,
                    description: "valid reminder structure",
                },
                {
                    data: {
                        id: "123456789012345678",
                        remindersCreate: [{
                            id: "123456789012345679",
                            name: "Reminder with items",
                            reminderItemsCreate: [{
                                id: "123456789012345680",
                                name: "Item 1",
                                reminderConnect: "123456789012345679",
                            }],
                        }],
                    },
                    shouldPass: true,
                    description: "reminder with items",
                },
                {
                    data: {
                        id: "123456789012345678",
                        remindersCreate: [{
                            // Missing id
                            name: "Reminder without ID",
                        }],
                    },
                    shouldPass: false,
                    expectedError: /required/i,
                    description: "missing reminder id",
                },
                {
                    data: {
                        id: "123456789012345678",
                        remindersCreate: [{
                            id: "invalid-id",
                            name: "Reminder with invalid ID",
                        }],
                    },
                    shouldPass: false,
                    expectedError: /valid ID/i,
                    description: "invalid reminder id",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle back-references to parent list", async () => {
            const dataWithBackReference = {
                id: "123456789012345678",
                remindersCreate: [{
                    id: "123456789012345679",
                    name: "Reminder referencing parent",
                    reminderListConnect: "123456789012345678",
                }],
            };

            await testValidation(createSchema, dataWithBackReference, true);
        });
    });
});