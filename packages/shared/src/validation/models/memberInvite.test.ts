import { describe } from "vitest";
import { memberInviteFixtures, memberInviteTestDataFactory } from "../../__test/fixtures/api-inputs/memberInviteFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { memberInviteValidation } from "./memberInvite.js";

describe("memberInviteValidation", () => {
    runComprehensiveValidationTests(
        memberInviteValidation,
        memberInviteFixtures,
        memberInviteTestDataFactory,
        "memberInvite",
    );

    // No additional business logic tests needed - memberInvite validation is straightforward
    // The comprehensive tests already cover:
    // - Required fields (id, teamConnect, userConnect for create)
    // - Optional fields (message, willBeAdmin, willHavePermissions)
    // - String trimming for message field
    // - Boolean validation for willBeAdmin
    // - JSON string validation for willHavePermissions
    // - Required relationship connections (team, user) in create
    // - Update doesn't require relationship connections
});
