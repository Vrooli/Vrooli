import { describe } from "vitest";
import { reminderFixtures, reminderTestDataFactory } from "../../__test/fixtures/api/reminderFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { reminderValidation } from "./reminder.js";

describe("reminderValidation", () => {
    runComprehensiveValidationTests(
        reminderValidation,
        reminderFixtures,
        reminderTestDataFactory,
        "reminder",
    );

    // All standard tests are covered by runComprehensiveValidationTests.
    // No additional business logic tests needed for this model.
});
