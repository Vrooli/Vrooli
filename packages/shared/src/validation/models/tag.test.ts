import { describe } from "vitest";
import { tagFixtures, tagTestDataFactory } from "../../__test/fixtures/api/tagFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { tagValidation } from "./tag.js";

describe("tagValidation", () => {
    runComprehensiveValidationTests(
        tagValidation,
        tagFixtures,
        tagTestDataFactory,
        "tag",
    );

    // No additional business logic tests needed - all validation is basic field validation
    // covered by comprehensive tests (tag format, trimming, special characters)
    // Tag translation validation is handled in the main tag validation
});
