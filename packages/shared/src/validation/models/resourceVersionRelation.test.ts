import { describe } from "vitest";
import { resourceVersionRelationFixtures, resourceVersionRelationTestDataFactory } from "../../__test/fixtures/api-inputs/resourceVersionRelationFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { resourceVersionRelationValidation } from "./resourceVersionRelation.js";

describe("resourceVersionRelationValidation", () => {
    runComprehensiveValidationTests(
        resourceVersionRelationValidation,
        resourceVersionRelationFixtures,
        resourceVersionRelationTestDataFactory,
        "resourceVersionRelation",
    );

    // All standard tests are covered by runComprehensiveValidationTests.
    // No additional business logic tests needed for this model.
});
