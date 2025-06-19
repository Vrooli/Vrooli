/**
 * Example demonstrating the new type-safe fixture usage
 * This file shows how the enhanced fixtures provide compile-time type safety
 */

import { userFixtures, userTestDataFactory, typedUserFixtures } from "./userFixtures.js";

// Example 1: Type-safe fixture usage with compile-time validation
function demonstrateTypeSafety() {
    // ✅ This works - all required fields are present and properly typed
    const validBotCreate = userFixtures.minimal.create;
    console.log("Valid bot create:", validBotCreate.name); // TypeScript knows this is a string
    console.log("Bot settings version:", validBotCreate.botSettings.__version); // TypeScript knows about botSettings structure

    // ✅ This works - profile update with proper typing
    const validProfileUpdate = userFixtures.minimal.update;
    console.log("Profile ID:", validProfileUpdate.id); // TypeScript knows this is required

    // ❌ This would cause TypeScript compilation errors:
    // validBotCreate.invalidField; // Property 'invalidField' does not exist
    // validBotCreate.name = 123; // Type 'number' is not assignable to type 'string'
    // validProfileUpdate.botSettings = {}; // Property 'botSettings' does not exist on type 'ProfileUpdateInput'
}

// Example 2: Type-safe factory usage
async function demonstrateFactoryUsage() {
    // ✅ Create test data with proper typing and optional overrides
    const customBot = userTestDataFactory.createMinimal({
        name: "Custom Test Bot",
        handle: "custombot123"
    });
    console.log("Custom bot:", customBot.name);

    // ✅ Create test data with schema validation
    try {
        const validatedBot = await userTestDataFactory.createMinimalValidated({
            name: "Validated Bot"
        });
        console.log("Validated bot passed schema validation:", validatedBot.name);
    } catch (error) {
        console.error("Validation failed:", error);
    }

    // ✅ Runtime validation of fixtures
    if (typedUserFixtures.validateCreate) {
        try {
            const validated = await typedUserFixtures.validateCreate(userFixtures.minimal.create);
            console.log("Fixture validation passed for:", validated.name);
        } catch (error) {
            console.error("Fixture validation failed:", error);
        }
    }
}

// Example 3: IntelliSense and autocomplete benefits
function demonstrateIntelliSense() {
    const bot = userTestDataFactory.createComplete({
        // TypeScript provides autocomplete for all valid properties:
        // - name (string)
        // - handle (string)
        // - isPrivate (boolean)
        // - isBotDepictingPerson (boolean)
        // - botSettings (BotConfigObject)
        // - bannerImage (Upload)
        // - profileImage (Upload)
        // - translationsCreate (UserTranslationCreateInput[])
        name: "IntelliSense Bot",
        isBotDepictingPerson: false,
        botSettings: {
            __version: "1.0",
            model: "gpt-4",
            // TypeScript knows about BotConfigObject structure
        }
    });
    
    // TypeScript knows the return type is BotCreateInput
    console.log("Bot name:", bot.name); // string
    console.log("Is private:", bot.isPrivate); // boolean
    console.log("Bot settings:", bot.botSettings); // BotConfigObject
}

export {
    demonstrateTypeSafety,
    demonstrateFactoryUsage,
    demonstrateIntelliSense
};