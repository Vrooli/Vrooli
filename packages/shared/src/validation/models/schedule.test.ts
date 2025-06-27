import { describe } from "vitest";
import { scheduleFixtures, scheduleTestDataFactory } from "../../__test/fixtures/api-inputs/scheduleFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { scheduleValidation } from "./schedule.js";

describe("scheduleValidation", () => {
    // Run standard validation tests
    runComprehensiveValidationTests(
        scheduleValidation,
        scheduleFixtures,
        scheduleTestDataFactory,
        "schedule",
    );

    // No additional business logic tests needed - all validation is basic field/relationship 
    // validation covered by standard tests and fixtures
});
