import { describe } from "vitest";
import { emailFixtures, emailTestDataFactory } from "../../__test/fixtures/api/emailFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { emailValidation } from "./email.js";

describe("emailValidation", () => {
    runComprehensiveValidationTests(
        emailValidation,
        emailFixtures,
        emailTestDataFactory,
        "email",
    );

    // No additional business logic tests needed - email validation is straightforward
    // The comprehensive tests already cover:
    // - Required fields (id, emailAddress)
    // - Email format validation (including edge cases like plus signs, dots, hyphens)
    // - Max length validation
    // - Invalid formats and empty strings
});
