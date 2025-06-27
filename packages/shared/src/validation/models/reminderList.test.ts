import { describe } from "vitest";
import { reminderListFixtures, reminderListTestDataFactory } from "../../__test/fixtures/api-inputs/reminderListFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { reminderListValidation } from "./reminderList.js";

describe("reminderListValidation", () => {
    runComprehensiveValidationTests(
        reminderListValidation,
        reminderListFixtures,
        reminderListTestDataFactory,
        "reminderList",
    );

    // All standard tests are covered by runComprehensiveValidationTests.
    // No additional business logic tests needed for this model.
});
