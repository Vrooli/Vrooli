import { describe } from "vitest";
import { pullRequestFixtures, pullRequestTestDataFactory } from "../../__test/fixtures/api-inputs/pullRequestFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { pullRequestValidation } from "./pullRequest.js";

describe("pullRequestValidation", () => {
    // Run standard validation tests
    runComprehensiveValidationTests(
        pullRequestValidation,
        pullRequestFixtures,
        pullRequestTestDataFactory,
        "pullRequest",
    );

    // No additional business logic tests needed - all validation is basic field validation
    // covered by standard tests and fixtures (merge/pull types, status fields)
});
