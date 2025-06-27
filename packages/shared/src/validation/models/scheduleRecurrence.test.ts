import { describe } from "vitest";
import { scheduleRecurrenceFixtures, scheduleRecurrenceTestDataFactory } from "../../__test/fixtures/api-inputs/scheduleRecurrenceFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { scheduleRecurrenceValidation } from "./scheduleRecurrence.js";

describe("scheduleRecurrenceValidation", () => {
    // Run standard validation tests
    runComprehensiveValidationTests(
        scheduleRecurrenceValidation,
        scheduleRecurrenceFixtures,
        scheduleRecurrenceTestDataFactory,
        "scheduleRecurrence",
    );

    // No additional business logic tests needed - all validation is basic field validation
    // covered by standard tests and fixtures (recurrence types, intervals, dayOfWeek/Month fields)
});
