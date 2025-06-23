import { describe } from "vitest";
import { reportFixtures, reportTestDataFactory } from "../../__test/fixtures/api/reportFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { reportValidation } from "./report.js";

describe("reportValidation", () => {
    // Run standard validation tests
    runComprehensiveValidationTests(
        reportValidation,
        reportFixtures,
        reportTestDataFactory,
        "report",
    );

    // No additional business logic tests needed - all validation is basic field validation
    // covered by standard tests and fixtures (report reasons, languages, entity references)
});
