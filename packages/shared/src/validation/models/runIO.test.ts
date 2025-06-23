import { describe } from "vitest";
import { runIOFixtures, runIOTestDataFactory } from "../../__test/fixtures/api/runIOFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { runIOValidation } from "./runIO.js";

describe("runIOValidation", () => {
    // Run standard validation tests
    runComprehensiveValidationTests(
        runIOValidation,
        runIOFixtures,
        runIOTestDataFactory,
        "runIO",
    );

    // No additional business logic tests needed - all validation is basic field validation
    // covered by standard tests and fixtures (required fields, data structure, node references)
});
