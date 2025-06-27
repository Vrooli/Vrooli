import { describe } from "vitest";
import { phoneFixtures, typedPhoneTestDataFactory } from "../../__test/fixtures/api-inputs/phoneFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { phoneValidation } from "./phone.js";

describe("phoneValidation", () => {
    runComprehensiveValidationTests(
        phoneValidation,
        phoneFixtures,
        typedPhoneTestDataFactory,
        "phone",
    );

    // No additional business logic tests needed - phone validation is straightforward
    // The comprehensive tests already cover:
    // - Required fields (id, phoneNumber)
    // - Phone number format validation (various formats with spaces, dashes, parentheses, etc.)
    // - Max length validation
    // - International formats and vanity numbers
});
