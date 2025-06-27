import { describe } from "vitest";
import { chatMessageFixtures, chatMessageTestDataFactory } from "../../__test/fixtures/api-inputs/chatMessageFixtures.js";
import { runComprehensiveValidationTests, testValidation } from "./__test/validationTestUtils.js";
import { chatMessageValidation, messageConfigObjectValidationSchema } from "./chatMessage.js";

describe("chatMessageValidation", () => {
    // Run standard validation tests using shared fixtures
    runComprehensiveValidationTests(
        chatMessageValidation,
        chatMessageFixtures,
        chatMessageTestDataFactory,
        "chatMessage",
    );

    // No additional business logic tests needed - version index and config structure
    // are basic field validation covered by fixtures
});

describe("messageConfigObjectValidationSchema", () => {
    // Business logic validation - tool calls result consistency
    describe("tool calls business logic", () => {
        const baseConfig = {
            __version: "1.0.0",
            resources: [],
        };

        it("should enforce tool result consistency", async () => {
            // Business rule: Tool results must have output when success=true, error when success=false
            await testValidation(
                messageConfigObjectValidationSchema,
                {
                    ...baseConfig,
                    toolCalls: [{
                        id: "123456789012345678",
                        function: { name: "test_function", arguments: "{}" },
                        result: {
                            success: true,
                            // Missing output when success is true
                            error: { code: "CODE", message: "msg" },
                        },
                    }],
                },
                false,
                /Tool result must be consistent/,
            );
        });
    });
});
