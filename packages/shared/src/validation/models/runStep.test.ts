import { describe } from "vitest";
import { runStepFixtures, runStepTestDataFactory } from "../../__test/fixtures/api-inputs/runStepFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { runRoutineStepValidation } from "./runStep.js";

describe("runRoutineStepValidation", () => {
    // Run standard validation tests
    runComprehensiveValidationTests(
        runRoutineStepValidation,
        runStepFixtures,
        runStepTestDataFactory,
        "runStep",
    );

    // No additional business logic tests needed - all validation is basic field validation
    // covered by standard tests and fixtures (required fields, field types, status enums, etc.)
});
