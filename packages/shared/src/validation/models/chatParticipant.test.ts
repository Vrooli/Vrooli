import { describe } from "vitest";
import { chatParticipantFixtures, chatParticipantTestDataFactory } from "../../__test/fixtures/api/chatParticipantFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { chatParticipantValidation } from "./chatParticipant.js";

describe("chatParticipantValidation", () => {
    // Run standard validation tests using shared fixtures
    // Note: This model only has update operation, no create
    runComprehensiveValidationTests(
        chatParticipantValidation,
        chatParticipantFixtures,
        chatParticipantTestDataFactory,
        "chatParticipant",
    );

    // No additional business logic tests needed - ID validation is basic field validation
    // covered by fixtures
});
