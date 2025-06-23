import { describe } from "vitest";
import { botFixtures, botTestDataFactory, botTranslationFixtures, botTranslationTestDataFactory } from "../../__test/fixtures/api/botFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { botTranslationValidation, botValidation } from "./bot.js";

describe("botValidation", () => {
    // Run comprehensive validation tests from API fixtures
    runComprehensiveValidationTests(botValidation, botFixtures, botTestDataFactory, "bot");

    // No additional business logic tests needed - botSettings and isBotDepictingPerson 
    // are basic field validation covered by fixtures
});

describe("botTranslationValidation", () => {
    // Run comprehensive validation tests from API fixtures
    runComprehensiveValidationTests(botTranslationValidation, botTranslationFixtures, botTranslationTestDataFactory, "botTranslation");

    // Note: botTranslation has standard validation rules covered by comprehensive tests
});
