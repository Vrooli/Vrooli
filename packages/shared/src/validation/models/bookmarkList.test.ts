import { describe } from "vitest";
import { bookmarkListFixtures, bookmarkListTestDataFactory } from "../../__test/fixtures/api/bookmarkListFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { bookmarkListValidation } from "./bookmarkList.js";

describe("bookmarkListValidation", () => {
    // Run comprehensive validation tests using shared fixtures
    runComprehensiveValidationTests(
        bookmarkListValidation,
        bookmarkListFixtures,
        bookmarkListTestDataFactory,
        "bookmarkList",
    );

    // No additional business logic tests needed - all validation is covered by standard tests
    // Label length and whitespace trimming are basic field validation covered by fixtures
});

