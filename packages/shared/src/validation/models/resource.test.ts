import { describe } from "vitest";
import { resourceFixtures, resourceTestDataFactory } from "../../__test/fixtures/api/resourceFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { resourceValidation } from "./resource.js";

describe("resourceValidation", () => {
    runComprehensiveValidationTests(
        resourceValidation,
        resourceFixtures,
        resourceTestDataFactory,
        "resource",
    );

    // All standard tests are covered by runComprehensiveValidationTests.
    // No additional business logic tests needed for this model.
});
