import { describe } from "vitest";
import { chatInviteFixtures, chatInviteTestDataFactory } from "../../__test/fixtures/api-inputs/chatInviteFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { chatInviteValidation } from "./chatInvite.js";

describe("chatInviteValidation", () => {
    // Run standard validation tests using shared fixtures
    runComprehensiveValidationTests(
        chatInviteValidation,
        chatInviteFixtures,
        chatInviteTestDataFactory,
        "chatInvite",
    );

    // No additional business logic tests needed - message length and whitespace trimming
    // are basic field validation covered by fixtures
});
