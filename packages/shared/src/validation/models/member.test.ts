import { describe } from "vitest";
import { memberFixtures, memberTestDataFactory } from "../../__test/fixtures/api/memberFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { memberValidation } from "./member.js";

describe("memberValidation", () => {
    runComprehensiveValidationTests(
        memberValidation,
        memberFixtures,
        memberTestDataFactory,
        "member",
    );

    // No additional business logic tests needed - member validation is straightforward
    // The comprehensive tests already cover:
    // - Required field (id) for update
    // - Optional fields (isAdmin, permissions)
    // - Boolean conversion for isAdmin
    // - JSON string validation for permissions
    // - No create operation (member can only be created through invites)
});
