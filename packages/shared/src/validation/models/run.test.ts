import { describe } from "vitest";
import { runFixtures, runTestDataFactory } from "../../__test/fixtures/api-inputs/runFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { runValidation } from "./run.js";

describe("runValidation", () => {
    // Run comprehensive validation tests
    runComprehensiveValidationTests(runValidation, runFixtures, runTestDataFactory, "run");

    // No additional business logic tests needed - all validation is basic field validation
    // covered by standard tests and fixtures (status enums, time fields, data length)
});
