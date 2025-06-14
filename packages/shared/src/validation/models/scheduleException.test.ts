import { describe, expect, it } from "vitest";
import { scheduleExceptionFixtures } from "./__test/fixtures/scheduleExceptionFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test/validationTestUtils.js";
import { scheduleExceptionValidation } from "./scheduleException.js";

describe("scheduleExceptionValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        scheduleExceptionValidation,
        scheduleExceptionFixtures,
        "scheduleException",
    );

    describe("create specific validations", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = scheduleExceptionValidation.create(defaultParams);

        it("should require id in create", async () => {
            const dataWithoutId = { ...scheduleExceptionFixtures.minimal.create };
            delete dataWithoutId.id;

            await testValidation(
                createSchema,
                dataWithoutId,
                false,
                /required/i,
            );
        });

        it("should require newEndTime in create", async () => {
            const dataWithoutEndTime = {
                id: "123456789012345678",
                newStartTime: new Date("2025-07-05T10:00:00Z"),
                scheduleConnect: "123456789012345679",
                // Missing required newEndTime
            };

            await testValidation(
                createSchema,
                dataWithoutEndTime,
                false,
                /required/i,
            );
        });

        it("should require scheduleConnect in create", async () => {
            await testValidation(
                createSchema,
                scheduleExceptionFixtures.invalid.missingScheduleConnect.create,
                false,
                /required/i,
            );
        });

        it("should allow minimal create with required fields", async () => {
            await testValidation(
                createSchema,
                scheduleExceptionFixtures.minimal.create,
                true,
            );
        });

        it("should allow complete create with all fields", async () => {
            await testValidation(
                createSchema,
                scheduleExceptionFixtures.complete.create,
                true,
            );
        });

        describe("originalStartTime field", () => {
            it("should be optional in create", async () => {
                await testValidation(
                    createSchema,
                    scheduleExceptionFixtures.minimal.create,
                    true,
                );
            });

            it("should accept valid dates", async () => {
                const scenarios = [
                    {
                        data: scheduleExceptionFixtures.edgeCases.movedEarlier.create,
                        shouldPass: true,
                        description: "original time in afternoon",
                    },
                    {
                        data: scheduleExceptionFixtures.edgeCases.movedLater.create,
                        shouldPass: true,
                        description: "original time in morning",
                    },
                    {
                        data: scheduleExceptionFixtures.edgeCases.differentMonth.create,
                        shouldPass: true,
                        description: "original time in different month",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject invalid date types", async () => {
                await testValidation(
                    createSchema,
                    scheduleExceptionFixtures.invalid.invalidTypes.create,
                    false,
                    /type, but the final value was/i,
                );
            });
        });

        describe("newStartTime and newEndTime fields", () => {
            it("should allow both times to be provided", async () => {
                await testValidation(
                    createSchema,
                    scheduleExceptionFixtures.complete.create,
                    true,
                );
            });

            it("should reject endTime without startTime", async () => {
                await testValidation(
                    createSchema,
                    scheduleExceptionFixtures.invalid.endTimeWithoutStartTime.create,
                    false,
                    /End time must be at least a second after start time/i,
                );
            });

            it("should reject endTime before startTime", async () => {
                await testValidation(
                    createSchema,
                    scheduleExceptionFixtures.invalid.invalidTimeOrder.create,
                    false,
                    /End time must be at least a second after start time/i,
                );
            });

            it("should reject endTime equal to startTime", async () => {
                const dataWithEqualTimes = {
                    id: "123456789012345678",
                    newStartTime: new Date("2025-07-05T12:00:00Z"),
                    newEndTime: new Date("2025-07-05T12:00:00Z"),
                    scheduleConnect: "123456789012345679",
                };

                await testValidation(
                    createSchema,
                    dataWithEqualTimes,
                    false,
                    /End time must be at least a second after start time/i,
                );
            });

            it("should accept minimal valid time range", async () => {
                await testValidation(
                    createSchema,
                    scheduleExceptionFixtures.edgeCases.minimalTimeRange.create,
                    true,
                );
            });

            it("should accept large valid time range", async () => {
                await testValidation(
                    createSchema,
                    scheduleExceptionFixtures.edgeCases.largeTimeRange.create,
                    true,
                );
            });

            it("should accept multi-day events", async () => {
                await testValidation(
                    createSchema,
                    scheduleExceptionFixtures.edgeCases.multiDayEvent.create,
                    true,
                );
            });
        });

        describe("time manipulation scenarios", () => {
            it("should handle cancelled meetings", async () => {
                await testValidation(
                    createSchema,
                    scheduleExceptionFixtures.edgeCases.cancelledMeeting.create,
                    true,
                );
            });

            it("should handle moving meetings earlier", async () => {
                await testValidation(
                    createSchema,
                    scheduleExceptionFixtures.edgeCases.movedEarlier.create,
                    true,
                );
            });

            it("should handle moving meetings later", async () => {
                await testValidation(
                    createSchema,
                    scheduleExceptionFixtures.edgeCases.movedLater.create,
                    true,
                );
            });

            it("should handle extending duration", async () => {
                await testValidation(
                    createSchema,
                    scheduleExceptionFixtures.edgeCases.extendedDuration.create,
                    true,
                );
            });

            it("should handle shortening duration", async () => {
                await testValidation(
                    createSchema,
                    scheduleExceptionFixtures.edgeCases.shortenedDuration.create,
                    true,
                );
            });

            it("should handle moving to different day", async () => {
                await testValidation(
                    createSchema,
                    scheduleExceptionFixtures.edgeCases.differentDay.create,
                    true,
                );
            });

            it("should handle moving to different month", async () => {
                await testValidation(
                    createSchema,
                    scheduleExceptionFixtures.edgeCases.differentMonth.create,
                    true,
                );
            });

            it("should handle moving to different year", async () => {
                await testValidation(
                    createSchema,
                    scheduleExceptionFixtures.edgeCases.differentYear.create,
                    true,
                );
            });
        });

        describe("scheduleConnect field", () => {
            it("should be required in create", async () => {
                await testValidation(
                    createSchema,
                    scheduleExceptionFixtures.invalid.missingScheduleConnect.create,
                    false,
                    /required/i,
                );
            });

            it("should accept valid schedule ID", async () => {
                await testValidation(
                    createSchema,
                    scheduleExceptionFixtures.minimal.create,
                    true,
                );
            });

            it("should accept different schedule IDs", async () => {
                await testValidation(
                    createSchema,
                    scheduleExceptionFixtures.edgeCases.differentScheduleId.create,
                    true,
                );
            });

            it("should reject invalid schedule ID", async () => {
                const dataWithInvalidSchedule = {
                    id: "123456789012345678",
                    newStartTime: new Date("2025-07-05T10:00:00Z"),
                    newEndTime: new Date("2025-07-05T17:00:00Z"),
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
    });

    describe("update specific validations", () => {
        const defaultParams = { omitFields: [] };
        const updateSchema = scheduleExceptionValidation.update(defaultParams);

        it("should require id in update", async () => {
            await testValidation(
                updateSchema,
                scheduleExceptionFixtures.invalid.missingRequired.update,
                false,
                /required/i,
            );
        });

        it("should allow minimal updates with just id", async () => {
            await testValidation(
                updateSchema,
                scheduleExceptionFixtures.minimal.update,
                true,
            );
        });

        it("should allow complete updates", async () => {
            await testValidation(
                updateSchema,
                scheduleExceptionFixtures.complete.update,
                true,
            );
        });

        describe("time field updates", () => {
            it("should allow updating only originalStartTime", async () => {
                await testValidation(
                    updateSchema,
                    scheduleExceptionFixtures.edgeCases.updateOnlyOriginalTime.update,
                    true,
                );
            });

            it("should allow updating only new times", async () => {
                await testValidation(
                    updateSchema,
                    scheduleExceptionFixtures.edgeCases.updateOnlyNewTimes.update,
                    true,
                );
            });

            it("should allow updating all time fields", async () => {
                await testValidation(
                    updateSchema,
                    scheduleExceptionFixtures.complete.update,
                    true,
                );
            });

            it("should make newEndTime optional in update", async () => {
                const dataWithoutEndTime = {
                    id: "123456789012345678",
                    newStartTime: new Date("2025-07-05T10:00:00Z"),
                    // No newEndTime - should be fine in update
                };

                await testValidation(updateSchema, dataWithoutEndTime, true);
            });

            it("should still validate time order in update", async () => {
                await testValidation(
                    updateSchema,
                    scheduleExceptionFixtures.invalid.invalidTimeOrder.update,
                    false,
                    /End time must be at least a second after start time/i,
                );
            });
        });

        describe("scheduleConnect restrictions", () => {
            it("should not allow updating scheduleConnect", async () => {
                const dataWithSchedule = {
                    id: "123456789012345678",
                    newStartTime: new Date("2025-07-05T10:00:00Z"), // Need startTime for validation
                    scheduleConnect: "123456789012345679",
                };

                const result = await testValidation(updateSchema, dataWithSchedule, true);
                expect(result).to.not.have.property("scheduleConnect");
            });
        });
    });

    describe("id validation", () => {
        const createSchema = scheduleExceptionValidation.create({ omitFields: [] });

        it("should accept valid Snowflake IDs", async () => {
            const validIds = [
                "123456789012345678",
                "999999999999999999",
                "100000000000000000",
            ];

            const scenarios = validIds.map(id => ({
                data: {
                    ...scheduleExceptionFixtures.minimal.create,
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
                    { ...scheduleExceptionFixtures.minimal.create, id },
                    false,
                    error,
                );
            }
        });
    });

    describe("type conversions", () => {
        const createSchema = scheduleExceptionValidation.create({ omitFields: [] });

        it("should handle ID conversion", async () => {
            const dataWithNumberId = {
                id: 123456789012345, // Smaller number to avoid precision issues
                newStartTime: new Date("2025-07-05T10:00:00Z"),
                newEndTime: new Date("2025-07-05T17:00:00Z"),
                scheduleConnect: "123456789012345679",
            };

            const result = await testValidation(
                createSchema,
                dataWithNumberId,
                true,
            );
            expect(result.id).to.be.a("string");
            expect(result.id).toBe("123456789012345");
        });

        it("should handle date string to Date conversion", async () => {
            const dataWithDateStrings = {
                id: "123456789012345678",
                originalStartTime: "2025-07-04T09:00:00Z",
                newStartTime: "2025-07-05T10:00:00Z",
                newEndTime: "2025-07-05T17:00:00Z",
                scheduleConnect: "123456789012345679",
            };

            const result = await testValidation(
                createSchema,
                dataWithDateStrings,
                true,
            );
            expect(result.originalStartTime).to.be.instanceof(Date);
            expect(result.newStartTime).to.be.instanceof(Date);
            expect(result.newEndTime).to.be.instanceof(Date);
        });
    });

    describe("edge cases", () => {
        const createSchema = scheduleExceptionValidation.create({ omitFields: [] });
        const updateSchema = scheduleExceptionValidation.update({ omitFields: [] });

        it("should handle minimal exception creation", async () => {
            await testValidation(
                createSchema,
                scheduleExceptionFixtures.minimal.create,
                true,
            );
        });

        it("should handle empty update operations", async () => {
            await testValidation(
                updateSchema,
                scheduleExceptionFixtures.edgeCases.onlyRequiredInUpdate.update,
                true,
            );
        });

        it("should handle exceptions across different time periods", async () => {
            const scenarios = [
                {
                    data: scheduleExceptionFixtures.edgeCases.differentDay.create,
                    shouldPass: true,
                    description: "different day",
                },
                {
                    data: scheduleExceptionFixtures.edgeCases.differentMonth.create,
                    shouldPass: true,
                    description: "different month",
                },
                {
                    data: scheduleExceptionFixtures.edgeCases.differentYear.create,
                    shouldPass: true,
                    description: "different year",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle various duration changes", async () => {
            const scenarios = [
                {
                    data: scheduleExceptionFixtures.edgeCases.extendedDuration.create,
                    shouldPass: true,
                    description: "extended duration",
                },
                {
                    data: scheduleExceptionFixtures.edgeCases.shortenedDuration.create,
                    shouldPass: true,
                    description: "shortened duration",
                },
                {
                    data: scheduleExceptionFixtures.edgeCases.minimalTimeRange.create,
                    shouldPass: true,
                    description: "minimal 1 second duration",
                },
                {
                    data: scheduleExceptionFixtures.edgeCases.largeTimeRange.create,
                    shouldPass: true,
                    description: "all day duration",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle cancelled meeting representation", async () => {
            await testValidation(
                createSchema,
                scheduleExceptionFixtures.edgeCases.cancelledMeeting.create,
                true,
            );
        });
    });
});
