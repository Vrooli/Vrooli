import { describe } from "vitest";
import { resourceVersionFixtures, resourceVersionTestDataFactory } from "../../__test/fixtures/api/resourceVersionFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { resourceVersionValidation } from "./resourceVersion.js";

describe("resourceVersionValidation", () => {
    runComprehensiveValidationTests(
        resourceVersionValidation,
        resourceVersionFixtures,
        resourceVersionTestDataFactory,
        "resourceVersion",
    );

    // All standard tests are covered by runComprehensiveValidationTests.
    // No additional business logic tests needed for this model.
});
