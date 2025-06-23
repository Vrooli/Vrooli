import { describe } from "vitest";
import { teamFixtures, teamTestDataFactory } from "../../__test/fixtures/api/teamFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { teamValidation } from "./team.js";

describe("teamValidation", () => {
    runComprehensiveValidationTests(
        teamValidation,
        teamFixtures,
        teamTestDataFactory,
        "team",
    );

    // No additional business logic tests needed - all validation is basic field validation
    // covered by comprehensive tests (handle format, translations, member invites)
});
