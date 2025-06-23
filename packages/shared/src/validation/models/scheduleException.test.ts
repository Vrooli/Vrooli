import { describe, it } from "vitest";
import { scheduleExceptionFixtures, scheduleExceptionTestDataFactory } from "../../__test/fixtures/api/scheduleExceptionFixtures.js";
import { runComprehensiveValidationTests, testValidation } from "./__test/validationTestUtils.js";
import { scheduleExceptionValidation } from "./scheduleException.js";

describe("scheduleExceptionValidation", () => {
    // Run comprehensive validation tests
    runComprehensiveValidationTests(
        scheduleExceptionValidation,
        scheduleExceptionFixtures,
        scheduleExceptionTestDataFactory,
        "scheduleException",
    );

    // Business logic validation - cross-field time validation
    describe("cross-field time validation", () => {
        const createSchema = scheduleExceptionValidation.create({ omitFields: [] });

        it("should enforce newEndTime must be after newStartTime", async () => {
            // Business rule: End time must be at least 1 second after start time
            const testCases = [
                {
                    data: scheduleExceptionFixtures.invalid.endTimeWithoutStartTime.create,
                    shouldPass: false,
                    expectedError: /End time must be at least a second after start time/i,
                    description: "endTime without startTime",
                },
                {
                    data: scheduleExceptionFixtures.invalid.invalidTimeOrder.create,
                    shouldPass: false,
                    expectedError: /End time must be at least a second after start time/i,
                    description: "endTime before startTime",
                },
                {
                    data: {
                        id: "123456789012345678",
                        newStartTime: new Date("2025-07-05T12:00:00Z"),
                        newEndTime: new Date("2025-07-05T12:00:00Z"), // Equal times
                        scheduleConnect: "123456789012345679",
                    },
                    shouldPass: false,
                    expectedError: /End time must be at least a second after start time/i,
                    description: "endTime equal to startTime",
                },
            ];

            for (const testCase of testCases) {
                await testValidation(
                    createSchema,
                    testCase.data,
                    testCase.shouldPass,
                    testCase.expectedError,
                );
            }
        });

        it("should accept valid time ranges", async () => {
            // Verify edge cases pass validation
            await testValidation(
                createSchema,
                scheduleExceptionFixtures.edgeCases.minimalTimeRange.create,
                true,
            );
            await testValidation(
                createSchema,
                scheduleExceptionFixtures.edgeCases.multiDayEvent.create,
                true,
            );
        });
    });
});
