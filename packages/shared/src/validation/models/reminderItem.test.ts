import { describe } from "vitest";
import { reminderItemFixtures, reminderItemTestDataFactory } from "../../__test/fixtures/api/reminderItemFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { reminderItemValidation } from "./reminderItem.js";

describe("reminderItemValidation", () => {
    runComprehensiveValidationTests(
        reminderItemValidation,
        reminderItemFixtures,
        reminderItemTestDataFactory,
        "reminderItem",
    );

    // All standard tests are covered by runComprehensiveValidationTests.
    // No additional business logic tests needed for this model.
});
