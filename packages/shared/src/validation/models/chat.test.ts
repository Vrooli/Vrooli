import { describe } from "vitest";
import { chatFixtures, chatTestDataFactory, chatTranslationFixtures, chatTranslationTestDataFactory } from "../../__test/fixtures/api-inputs/chatFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { chatTranslationValidation, chatValidation } from "./chat.js";

describe("chatValidation", () => {
    // Run comprehensive test suite
    runComprehensiveValidationTests(chatValidation, chatFixtures, chatTestDataFactory, "chat");

    // No additional business logic tests needed - nested relationships and omitFields
    // are standard validation features covered by fixtures and standard tests
});

describe("chatTranslationValidation", () => {
    // Run comprehensive test suite for chat translations
    runComprehensiveValidationTests(chatTranslationValidation, chatTranslationFixtures, chatTranslationTestDataFactory, "chatTranslation");

    // No additional business logic tests needed - partial updates are standard validation
});
