import { describe, it } from "vitest";
import { chatMessageFixtures } from "./__test/fixtures/chatMessageFixtures.js";
import { runStandardValidationTests, testValidation } from "./__test/validationTestUtils.js";
import { chatMessageValidation, messageConfigObjectValidationSchema } from "./chatMessage.js";

describe("chatMessageValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        chatMessageValidation,
        chatMessageFixtures,
        "chatMessage",
    );

    describe("specific field validations", () => {
        const defaultParams = { omitFields: [] };

        describe("config field", () => {
            const createSchema = chatMessageValidation.create(defaultParams);
            const updateSchema = chatMessageValidation.update(defaultParams);

            it("should require config in create", async () => {
                await testValidation(
                    createSchema,
                    chatMessageFixtures.invalid.missingRequired.create,
                    false,
                    /required/i,
                );
            });

            it("should make config optional in update", async () => {
                await testValidation(
                    updateSchema,
                    chatMessageFixtures.minimal.update,
                    true,
                );
            });

            it("should validate config object structure", async () => {
                await testValidation(
                    createSchema,
                    chatMessageFixtures.invalid.invalidConfig.create,
                    false,
                    /required/i,
                );
            });

            it("should reject non-object config", async () => {
                await testValidation(
                    createSchema,
                    chatMessageFixtures.invalid.invalidTypes.create,
                    false,
                );
            });

            it("should accept complex config", async () => {
                await testValidation(
                    createSchema,
                    chatMessageFixtures.edgeCases.complexConfig.create,
                    true,
                );
            });
        });

        describe("versionIndex field", () => {
            const createSchema = chatMessageValidation.create(defaultParams);

            it("should be optional in create", async () => {
                await testValidation(
                    createSchema,
                    chatMessageFixtures.minimal.create,
                    true,
                );
            });

            it("should accept zero as version index", async () => {
                await testValidation(
                    createSchema,
                    chatMessageFixtures.edgeCases.zeroVersionIndex.create,
                    true,
                );
            });

            it("should accept large version index", async () => {
                await testValidation(
                    createSchema,
                    chatMessageFixtures.edgeCases.maxVersionIndex.create,
                    true,
                );
            });

            it("should reject negative version index", async () => {
                await testValidation(
                    createSchema,
                    chatMessageFixtures.invalid.invalidVersionIndex.create,
                    false,
                );
            });

            it("should reject non-number version index", async () => {
                await testValidation(
                    createSchema,
                    chatMessageFixtures.invalid.invalidTypes.create,
                    false,
                );
            });
        });

        describe("chat relationship", () => {
            const createSchema = chatMessageValidation.create(defaultParams);

            it("should require chatConnect in create", async () => {
                await testValidation(
                    createSchema,
                    chatMessageFixtures.invalid.missingRequired.create,
                    false,
                    /required/i,
                );
            });
        });

        describe("user relationship", () => {
            const createSchema = chatMessageValidation.create(defaultParams);

            it("should be optional in create", async () => {
                await testValidation(
                    createSchema,
                    chatMessageFixtures.minimal.create,
                    true,
                );
            });

            it("should accept valid userConnect", async () => {
                await testValidation(
                    createSchema,
                    chatMessageFixtures.complete.create,
                    true,
                );
            });
        });

        describe("translations relationship", () => {
            const createSchema = chatMessageValidation.create(defaultParams);
            const updateSchema = chatMessageValidation.update(defaultParams);

            it("should be optional in both create and update", async () => {
                await testValidation(
                    createSchema,
                    chatMessageFixtures.minimal.create,
                    true,
                );

                await testValidation(
                    updateSchema,
                    chatMessageFixtures.minimal.update,
                    true,
                );
            });

            it("should accept valid translation creation", async () => {
                await testValidation(
                    createSchema,
                    chatMessageFixtures.complete.create,
                    true,
                );
            });

            it("should accept translation updates", async () => {
                await testValidation(
                    updateSchema,
                    chatMessageFixtures.complete.update,
                    true,
                );
            });

            it("should reject empty translation text", async () => {
                await testValidation(
                    createSchema,
                    chatMessageFixtures.invalid.invalidTranslationText.create,
                    false,
                    /required/i,
                );
            });

            it("should accept max length translation text", async () => {
                await testValidation(
                    createSchema,
                    chatMessageFixtures.edgeCases.maxLengthText.create,
                    true,
                );
            });

            it("should reject text that exceeds max length", async () => {
                await testValidation(
                    createSchema,
                    chatMessageFixtures.edgeCases.textTooLong.create,
                    false,
                    /over the limit/i,
                );
            });
        });
    });
});

describe("messageConfigObjectValidationSchema", () => {
    describe("required fields", () => {
        it("should require __version", async () => {
            await testValidation(
                messageConfigObjectValidationSchema,
                {
                    resources: [],
                },
                false,
                /required/i,
            );
        });

        it("should require resources", async () => {
            await testValidation(
                messageConfigObjectValidationSchema,
                {
                    __version: "1.0.0",
                },
                false,
                /required/i,
            );
        });

        it("should accept minimal valid config", async () => {
            await testValidation(
                messageConfigObjectValidationSchema,
                {
                    __version: "1.0.0",
                    resources: [],
                },
                true,
            );
        });
    });

    describe("optional fields", () => {
        const baseConfig = {
            __version: "1.0.0",
            resources: [],
        };

        it("should accept valid role values", async () => {
            const validRoles = ["user", "assistant", "system", "tool"];

            for (const role of validRoles) {
                await testValidation(
                    messageConfigObjectValidationSchema,
                    {
                        ...baseConfig,
                        role,
                    },
                    true,
                );
            }
        });

        it("should reject invalid role values", async () => {
            await testValidation(
                messageConfigObjectValidationSchema,
                {
                    ...baseConfig,
                    role: "invalid_role",
                },
                false,
            );
        });

        it("should accept null turnId", async () => {
            await testValidation(
                messageConfigObjectValidationSchema,
                {
                    ...baseConfig,
                    turnId: null,
                },
                true,
            );
        });

        it("should accept valid turnId", async () => {
            await testValidation(
                messageConfigObjectValidationSchema,
                {
                    ...baseConfig,
                    turnId: 123,
                },
                true,
            );
        });

        it("should reject non-integer turnId", async () => {
            await testValidation(
                messageConfigObjectValidationSchema,
                {
                    ...baseConfig,
                    turnId: 123.45,
                },
                false,
            );
        });
    });

    describe("toolCalls field", () => {
        const baseConfig = {
            __version: "1.0.0",
            resources: [],
        };

        it("should accept valid tool calls", async () => {
            await testValidation(
                messageConfigObjectValidationSchema,
                {
                    ...baseConfig,
                    toolCalls: [
                        {
                            id: "123456789012345678",
                            function: {
                                name: "test_function",
                                arguments: "{}",
                            },
                        },
                    ],
                },
                true,
            );
        });

        it("should accept tool calls with results", async () => {
            await testValidation(
                messageConfigObjectValidationSchema,
                {
                    ...baseConfig,
                    toolCalls: [
                        {
                            id: "123456789012345678",
                            function: {
                                name: "test_function",
                                arguments: "{}",
                            },
                            result: {
                                success: true,
                                output: "success output",
                            },
                        },
                    ],
                },
                true,
            );
        });

        it("should accept tool calls with error results", async () => {
            await testValidation(
                messageConfigObjectValidationSchema,
                {
                    ...baseConfig,
                    toolCalls: [
                        {
                            id: "123456789012345678",
                            function: {
                                name: "test_function",
                                arguments: "{}",
                            },
                            result: {
                                success: false,
                                error: {
                                    code: "ERROR_CODE",
                                    message: "Error message",
                                },
                            },
                        },
                    ],
                },
                true,
            );
        });

        it("should reject tool calls with inconsistent results", async () => {
            await testValidation(
                messageConfigObjectValidationSchema,
                {
                    ...baseConfig,
                    toolCalls: [
                        {
                            id: "123456789012345678",
                            function: {
                                name: "test_function",
                                arguments: "{}",
                            },
                            result: {
                                success: true,
                                // Missing output when success is true
                                error: { code: "CODE", message: "msg" },
                            },
                        },
                    ],
                },
                false,
                /Tool result must be consistent/,
            );
        });

        it("should reject incomplete tool calls", async () => {
            await testValidation(
                messageConfigObjectValidationSchema,
                {
                    ...baseConfig,
                    toolCalls: [
                        {
                            // Missing id and function
                            result: {
                                success: true,
                                output: "output",
                            },
                        },
                    ],
                },
                false,
                /required/i,
            );
        });
    });

    describe("arrays fields", () => {
        const baseConfig = {
            __version: "1.0.0",
            resources: [],
        };

        it("should accept contextHints array", async () => {
            await testValidation(
                messageConfigObjectValidationSchema,
                {
                    ...baseConfig,
                    contextHints: ["hint1", "hint2"],
                },
                true,
            );
        });

        it("should accept respondingBots array", async () => {
            await testValidation(
                messageConfigObjectValidationSchema,
                {
                    ...baseConfig,
                    respondingBots: ["@all", "bot1"],
                },
                true,
            );
        });

        it("should accept mixed resources array", async () => {
            await testValidation(
                messageConfigObjectValidationSchema,
                {
                    ...baseConfig,
                    resources: [
                        { id: "1", type: "document" },
                        { id: "2", type: "image" },
                    ],
                },
                true,
            );
        });
    });
});
