import { describe } from "vitest";
import { transferFixtures, transferTestDataFactory } from "../../__test/fixtures/api-inputs/transferFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { transferValidation } from "./transfer.js";

describe("transferValidation", () => {
    runComprehensiveValidationTests(
        transferValidation,
        transferFixtures,
        transferTestDataFactory,
        "transfer",
    );

    // No additional business logic tests needed - all validation is basic field validation
    // covered by comprehensive tests (transfer status, team/user relationships)
});
