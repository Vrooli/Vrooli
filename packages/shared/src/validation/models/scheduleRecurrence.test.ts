import { describe, it, expect } from "vitest";
import { scheduleRecurrenceValidation } from "./scheduleRecurrence.js";
import { scheduleRecurrenceFixtures } from "./__test__/fixtures/scheduleRecurrenceFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test__/validationTestUtils.js";

describe("scheduleRecurrenceValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        scheduleRecurrenceValidation,
        scheduleRecurrenceFixtures,
        "scheduleRecurrence",
    );

    describe("create specific validations", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = scheduleRecurrenceValidation.create(defaultParams);

        it("should require id in create", async () => {
            const dataWithoutId = { ...scheduleRecurrenceFixtures.minimal.create };
            delete dataWithoutId.id;

            await testValidation(
                createSchema,
                dataWithoutId,
                false,
                /required/i,
            );
        });

        it("should require recurrenceType in create", async () => {
            const dataWithoutType = {
                id: "123456789012345678",
                interval: 1,
                scheduleConnect: "123456789012345679",
                // Missing required recurrenceType
            };

            await testValidation(
                createSchema,
                dataWithoutType,
                false,
                /required/i,
            );
        });

        it("should require interval in create", async () => {
            const dataWithoutInterval = {
                id: "123456789012345678",
                recurrenceType: "Daily",
                scheduleConnect: "123456789012345679",
                // Missing required interval
            };

            await testValidation(
                createSchema,
                dataWithoutInterval,
                false,
                /required/i,
            );
        });

        it("should require scheduleConnect in create", async () => {
            await testValidation(
                createSchema,
                scheduleRecurrenceFixtures.invalid.missingScheduleConnect.create,
                false,
                /required/i,
            );
        });

        it("should allow minimal create with required fields", async () => {
            await testValidation(
                createSchema,
                scheduleRecurrenceFixtures.minimal.create,
                true,
            );
        });

        it("should allow complete create with all fields", async () => {
            await testValidation(
                createSchema,
                scheduleRecurrenceFixtures.complete.create,
                true,
            );
        });

        describe("recurrenceType field", () => {
            it("should accept valid recurrence types", async () => {
                const scenarios = [
                    {
                        data: scheduleRecurrenceFixtures.edgeCases.dailyRecurrence.create,
                        shouldPass: true,
                        description: "Daily recurrence",
                    },
                    {
                        data: scheduleRecurrenceFixtures.edgeCases.weeklyMonday.create,
                        shouldPass: true,
                        description: "Weekly recurrence",
                    },
                    {
                        data: scheduleRecurrenceFixtures.edgeCases.monthlyFirstDay.create,
                        shouldPass: true,
                        description: "Monthly recurrence",
                    },
                    {
                        data: scheduleRecurrenceFixtures.edgeCases.yearlyRecurrence.create,
                        shouldPass: true,
                        description: "Yearly recurrence",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject invalid recurrence types", async () => {
                await testValidation(
                    createSchema,
                    scheduleRecurrenceFixtures.invalid.invalidRecurrenceType.create,
                    false,
                    /must be one of/i,
                );
            });
        });

        describe("interval field", () => {
            it("should accept positive intervals", async () => {
                const scenarios = [
                    {
                        data: {
                            ...scheduleRecurrenceFixtures.minimal.create,
                            interval: 1,
                        },
                        shouldPass: true,
                        description: "interval of 1",
                    },
                    {
                        data: scheduleRecurrenceFixtures.edgeCases.biweekly.create,
                        shouldPass: true,
                        description: "interval of 2",
                    },
                    {
                        data: scheduleRecurrenceFixtures.edgeCases.quarterlyRecurrence.create,
                        shouldPass: true,
                        description: "interval of 3",
                    },
                    {
                        data: scheduleRecurrenceFixtures.edgeCases.maxInterval.create,
                        shouldPass: true,
                        description: "large interval",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject zero interval", async () => {
                await testValidation(
                    createSchema,
                    scheduleRecurrenceFixtures.invalid.invalidInterval.create,
                    false,
                    /Minimum value is 1/i,
                );
            });

            it("should reject negative intervals", async () => {
                const dataWithNegativeInterval = {
                    id: "123456789012345678",
                    recurrenceType: "Daily",
                    interval: -1,
                    scheduleConnect: "123456789012345679",
                };

                await testValidation(
                    createSchema,
                    dataWithNegativeInterval,
                    false,
                    /Minimum value is 1/i,
                );
            });
        });

        describe("dayOfWeek field", () => {
            it("should be optional", async () => {
                await testValidation(
                    createSchema,
                    scheduleRecurrenceFixtures.edgeCases.dailyRecurrence.create,
                    true,
                );
            });

            it("should accept valid days of week (1-7)", async () => {
                const scenarios = [
                    {
                        data: scheduleRecurrenceFixtures.edgeCases.weeklyMonday.create,
                        shouldPass: true,
                        description: "Monday (1)",
                    },
                    {
                        data: scheduleRecurrenceFixtures.edgeCases.weeklyFriday.create,
                        shouldPass: true,
                        description: "Friday (5)",
                    },
                    {
                        data: scheduleRecurrenceFixtures.edgeCases.weeklySunday.create,
                        shouldPass: true,
                        description: "Sunday (7)",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject day of week less than 1", async () => {
                await testValidation(
                    createSchema,
                    scheduleRecurrenceFixtures.invalid.invalidDayOfWeek.create,
                    false,
                    /Minimum value is 1/i,
                );
            });

            it("should reject day of week greater than 7", async () => {
                const dataWithInvalidDay = {
                    id: "123456789012345678",
                    recurrenceType: "Weekly",
                    interval: 1,
                    dayOfWeek: 8,
                    scheduleConnect: "123456789012345679",
                };

                await testValidation(
                    createSchema,
                    dataWithInvalidDay,
                    false,
                    /Minimum value is 7/i,
                );
            });
        });

        describe("dayOfMonth field", () => {
            it("should be optional", async () => {
                await testValidation(
                    createSchema,
                    scheduleRecurrenceFixtures.edgeCases.dailyRecurrence.create,
                    true,
                );
            });

            it("should accept valid days of month (1-31)", async () => {
                const scenarios = [
                    {
                        data: scheduleRecurrenceFixtures.edgeCases.monthlyFirstDay.create,
                        shouldPass: true,
                        description: "First day (1)",
                    },
                    {
                        data: scheduleRecurrenceFixtures.edgeCases.monthlyMidMonth.create,
                        shouldPass: true,
                        description: "Mid month (15)",
                    },
                    {
                        data: scheduleRecurrenceFixtures.edgeCases.monthlyLastDay.create,
                        shouldPass: true,
                        description: "Last possible day (31)",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject day of month less than 1", async () => {
                await testValidation(
                    createSchema,
                    scheduleRecurrenceFixtures.invalid.invalidDayOfMonth.create,
                    false,
                    /Minimum value is 1/i,
                );
            });

            it("should reject day of month greater than 31", async () => {
                const dataWithInvalidDay = {
                    id: "123456789012345678",
                    recurrenceType: "Monthly",
                    interval: 1,
                    dayOfMonth: 32,
                    scheduleConnect: "123456789012345679",
                };

                await testValidation(
                    createSchema,
                    dataWithInvalidDay,
                    false,
                    /Minimum value is 31/i,
                );
            });
        });

        describe("duration field", () => {
            it("should be optional", async () => {
                await testValidation(
                    createSchema,
                    scheduleRecurrenceFixtures.minimal.create,
                    true,
                );
            });

            it("should accept positive durations", async () => {
                const scenarios = [
                    {
                        data: scheduleRecurrenceFixtures.edgeCases.shortDuration.create,
                        shouldPass: true,
                        description: "15 minute duration",
                    },
                    {
                        data: scheduleRecurrenceFixtures.edgeCases.longDuration.create,
                        shouldPass: true,
                        description: "8 hour duration",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject zero duration", async () => {
                await testValidation(
                    createSchema,
                    scheduleRecurrenceFixtures.invalid.invalidDuration.create,
                    false,
                    /Minimum value is 1/i,
                );
            });

            it("should reject negative durations", async () => {
                const dataWithNegativeDuration = {
                    id: "123456789012345678",
                    recurrenceType: "Daily",
                    interval: 1,
                    duration: -30,
                    scheduleConnect: "123456789012345679",
                };

                await testValidation(
                    createSchema,
                    dataWithNegativeDuration,
                    false,
                    /Minimum value is 1/i,
                );
            });
        });

        describe("endDate field", () => {
            it("should be optional", async () => {
                await testValidation(
                    createSchema,
                    scheduleRecurrenceFixtures.edgeCases.withoutEndDate.create,
                    true,
                );
            });

            it("should accept valid dates", async () => {
                await testValidation(
                    createSchema,
                    scheduleRecurrenceFixtures.edgeCases.withEndDate.create,
                    true,
                );
            });

            it("should reject invalid date types", async () => {
                await testValidation(
                    createSchema,
                    scheduleRecurrenceFixtures.invalid.invalidTypes.create,
                    false,
                    /type, but the final value was/i,
                );
            });
        });

        describe("scheduleConnect field", () => {
            it("should be required in create", async () => {
                await testValidation(
                    createSchema,
                    scheduleRecurrenceFixtures.invalid.missingScheduleConnect.create,
                    false,
                    /required/i,
                );
            });

            it("should accept valid schedule ID", async () => {
                await testValidation(
                    createSchema,
                    scheduleRecurrenceFixtures.minimal.create,
                    true,
                );
            });

            it("should accept different schedule IDs", async () => {
                await testValidation(
                    createSchema,
                    scheduleRecurrenceFixtures.edgeCases.differentScheduleId.create,
                    true,
                );
            });

            it("should reject invalid schedule ID", async () => {
                const dataWithInvalidSchedule = {
                    id: "123456789012345678",
                    recurrenceType: "Daily",
                    interval: 1,
                    scheduleConnect: "invalid-id",
                };

                await testValidation(
                    createSchema,
                    dataWithInvalidSchedule,
                    false,
                    /valid ID/i,
                );
            });
        });

        describe("recurrence patterns", () => {
            it("should handle daily recurrence", async () => {
                await testValidation(
                    createSchema,
                    scheduleRecurrenceFixtures.edgeCases.dailyRecurrence.create,
                    true,
                );
            });

            it("should handle weekly recurrence with day of week", async () => {
                await testValidation(
                    createSchema,
                    scheduleRecurrenceFixtures.edgeCases.weeklyFriday.create,
                    true,
                );
            });

            it("should handle monthly recurrence with day of month", async () => {
                await testValidation(
                    createSchema,
                    scheduleRecurrenceFixtures.edgeCases.monthlyMidMonth.create,
                    true,
                );
            });

            it("should handle yearly recurrence", async () => {
                await testValidation(
                    createSchema,
                    scheduleRecurrenceFixtures.edgeCases.yearlyRecurrence.create,
                    true,
                );
            });

            it("should handle complex patterns", async () => {
                await testValidation(
                    createSchema,
                    scheduleRecurrenceFixtures.edgeCases.complexWeeklyPattern.create,
                    true,
                );
            });
        });
    });

    describe("update specific validations", () => {
        const defaultParams = { omitFields: [] };
        const updateSchema = scheduleRecurrenceValidation.update(defaultParams);

        it("should require id in update", async () => {
            await testValidation(
                updateSchema,
                scheduleRecurrenceFixtures.invalid.missingRequired.update,
                false,
                /required/i,
            );
        });

        it("should allow minimal updates with just id", async () => {
            await testValidation(
                updateSchema,
                scheduleRecurrenceFixtures.minimal.update,
                true,
            );
        });

        it("should allow complete updates", async () => {
            await testValidation(
                updateSchema,
                scheduleRecurrenceFixtures.complete.update,
                true,
            );
        });

        it("should make all fields optional in update except id", async () => {
            await testValidation(
                updateSchema,
                scheduleRecurrenceFixtures.edgeCases.onlyRequiredInUpdate.update,
                true,
            );
        });

        describe("field updates", () => {
            it("should allow updating recurrenceType", async () => {
                await testValidation(
                    updateSchema,
                    scheduleRecurrenceFixtures.edgeCases.updateRecurrenceType.update,
                    true,
                );
            });

            it("should allow updating interval", async () => {
                await testValidation(
                    updateSchema,
                    scheduleRecurrenceFixtures.edgeCases.updateInterval.update,
                    true,
                );
            });

            it("should allow updating endDate", async () => {
                await testValidation(
                    updateSchema,
                    scheduleRecurrenceFixtures.edgeCases.updateEndDate.update,
                    true,
                );
            });

            it("should still validate field values in update", async () => {
                await testValidation(
                    updateSchema,
                    scheduleRecurrenceFixtures.invalid.invalidTypes.update,
                    false,
                );
            });

            it("should validate interval in update", async () => {
                await testValidation(
                    updateSchema,
                    scheduleRecurrenceFixtures.invalid.invalidInterval.update,
                    false,
                    /Minimum value is 1/i,
                );
            });

            it("should validate dayOfWeek in update", async () => {
                await testValidation(
                    updateSchema,
                    scheduleRecurrenceFixtures.invalid.invalidDayOfWeek.update,
                    false,
                    /Minimum value is 7/i,
                );
            });

            it("should validate dayOfMonth in update", async () => {
                await testValidation(
                    updateSchema,
                    scheduleRecurrenceFixtures.invalid.invalidDayOfMonth.update,
                    false,
                    /Minimum value is 31/i,
                );
            });
        });

        describe("scheduleConnect restrictions", () => {
            it("should not allow updating scheduleConnect", async () => {
                const dataWithSchedule = {
                    id: "123456789012345678",
                    scheduleConnect: "123456789012345679",
                };

                const result = await testValidation(updateSchema, dataWithSchedule, true);
                expect(result).to.not.have.property("scheduleConnect");
            });
        });
    });

    describe("id validation", () => {
        const createSchema = scheduleRecurrenceValidation.create({ omitFields: [] });

        it("should accept valid Snowflake IDs", async () => {
            const validIds = [
                "123456789012345678",
                "999999999999999999",
                "100000000000000000",
            ];

            const scenarios = validIds.map(id => ({
                data: {
                    ...scheduleRecurrenceFixtures.minimal.create,
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
                    { ...scheduleRecurrenceFixtures.minimal.create, id },
                    false,
                    error,
                );
            }
        });
    });

    describe("type conversions", () => {
        const createSchema = scheduleRecurrenceValidation.create({ omitFields: [] });

        it("should handle ID conversion", async () => {
            const dataWithNumberId = {
                id: 123456789012345, // Smaller number to avoid precision issues
                recurrenceType: "Daily",
                interval: 1,
                scheduleConnect: "123456789012345679",
            };

            const result = await testValidation(
                createSchema,
                dataWithNumberId,
                true,
            );
            expect(result.id).to.be.a("string");
            expect(result.id).to.equal("123456789012345");
        });

        it("should handle date string to Date conversion", async () => {
            const dataWithDateString = {
                id: "123456789012345678",
                recurrenceType: "Daily",
                interval: 1,
                endDate: "2025-12-31T23:59:59Z",
                scheduleConnect: "123456789012345679",
            };

            const result = await testValidation(
                createSchema,
                dataWithDateString,
                true,
            );
            expect(result.endDate).to.be.instanceof(Date);
        });

        it("should reject non-integer numeric fields", async () => {
            const dataWithFloats = {
                id: "123456789012345678",
                recurrenceType: "Weekly",
                interval: 1.5, // Should be rejected as non-integer
                dayOfWeek: 3.7, // Should be rejected as non-integer
                scheduleConnect: "123456789012345679",
            };

            await testValidation(
                createSchema,
                dataWithFloats,
                false,
                /must be an integer/i,
            );
        });
    });

    describe("edge cases", () => {
        const createSchema = scheduleRecurrenceValidation.create({ omitFields: [] });
        const updateSchema = scheduleRecurrenceValidation.update({ omitFields: [] });

        it("should handle minimal recurrence creation", async () => {
            await testValidation(
                createSchema,
                scheduleRecurrenceFixtures.minimal.create,
                true,
            );
        });

        it("should handle empty update operations", async () => {
            await testValidation(
                updateSchema,
                scheduleRecurrenceFixtures.edgeCases.onlyRequiredInUpdate.update,
                true,
            );
        });

        it("should handle various recurrence intervals", async () => {
            const scenarios = [
                {
                    data: scheduleRecurrenceFixtures.edgeCases.dailyRecurrence.create,
                    shouldPass: true,
                    description: "every day",
                },
                {
                    data: scheduleRecurrenceFixtures.edgeCases.biweekly.create,
                    shouldPass: true,
                    description: "every 2 weeks",
                },
                {
                    data: scheduleRecurrenceFixtures.edgeCases.quarterlyRecurrence.create,
                    shouldPass: true,
                    description: "every 3 months",
                },
                {
                    data: scheduleRecurrenceFixtures.edgeCases.biYearlyRecurrence.create,
                    shouldPass: true,
                    description: "every 2 years",
                },
                {
                    data: scheduleRecurrenceFixtures.edgeCases.maxInterval.create,
                    shouldPass: true,
                    description: "every 365 days",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle recurrences with and without end dates", async () => {
            const scenarios = [
                {
                    data: scheduleRecurrenceFixtures.edgeCases.withEndDate.create,
                    shouldPass: true,
                    description: "with end date",
                },
                {
                    data: scheduleRecurrenceFixtures.edgeCases.withoutEndDate.create,
                    shouldPass: true,
                    description: "without end date (infinite)",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle all days of the week", async () => {
            for (let day = 1; day <= 7; day++) {
                const dataWithDay = {
                    id: "123456789012345678",
                    recurrenceType: "Weekly",
                    interval: 1,
                    dayOfWeek: day,
                    scheduleConnect: "123456789012345679",
                };

                await testValidation(createSchema, dataWithDay, true);
            }
        });
    });
});
