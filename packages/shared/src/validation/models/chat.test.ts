import { describe, expect, it } from "vitest";
import { chatFixtures, chatTestDataFactory, chatTranslationFixtures } from "./__test/fixtures/chatFixtures.js";
import { runStandardValidationTests, testValidation } from "./__test/validationTestUtils.js";
import { chatTranslationValidation, chatValidation } from "./chat.js";

describe("chatValidation", () => {
    // Run standard test suite
    runStandardValidationTests(chatValidation, chatFixtures, "chat");

    // Additional chat-specific tests
    describe("chat-specific validation", () => {
        const defaultParams = { omitFields: [] };

        describe("create validation", () => {
            const createSchema = chatValidation.create(defaultParams);

            it("should accept chat without any relations", async () => {
                const result = await testValidation(
                    createSchema,
                    {
                        id: "123456789012345678",
                    },
                    true,
                );
                expect(result).toHaveProperty("id");
                expect(result).to.not.have.property("invites");
                expect(result).to.not.have.property("messages");
                expect(result).to.not.have.property("team");
                expect(result).to.not.have.property("translations");
            });

            it("should handle complex nested invites creation", async () => {
                const data = {
                    id: "123456789012345678",
                    invitesCreate: [
                        {
                            id: "223456789012345678",
                            message: "Invite 1",
                            userConnect: "323456789012345678",
                        },
                        {
                            id: "223456789012345679",
                            message: "Invite 2",
                            userConnect: "323456789012345679",
                        },
                    ],
                };
                const result = await testValidation(
                    createSchema,
                    data,
                    true,
                );
                expect(result).toHaveProperty("invitesCreate");
                expect(result.invitesCreate).to.have.length(2);
            });

            it("should validate nested message creation", async () => {
                const data = {
                    id: "123456789012345678",
                    messagesCreate: [{
                        id: "223456789012345678",
                        config: {
                            __version: "1.0.0",
                            resources: [],
                        },
                        chatConnect: "123456789012345678",
                        userConnect: "323456789012345678",
                        translationsCreate: [{
                            id: "333456789012345678",
                            language: "en",
                            text: "Test message",
                        }],
                    }],
                };

                try {
                    const result = await createSchema.validate(data, {
                        abortEarly: false,
                        stripUnknown: true,
                    });
                    expect(result).toHaveProperty("messagesCreate");
                    expect(result.messagesCreate).to.have.length(1);
                    expect(result.messagesCreate[0].translationsCreate[0].text).toBe("Test message");
                } catch (error: any) {
                    console.error("Direct validation error:", {
                        message: error.message,
                        errors: error.errors,
                        inner: error.inner?.map((e: any) => ({ path: e.path, message: e.message, type: e.type })),
                    });
                    throw error;
                }
            });

            it("should reject invalid nested relations", async () => {
                await testValidation(
                    createSchema,
                    chatFixtures.invalid.invalidRelations.create,
                    false,
                );
            });

            it("should validate team connection", async () => {
                const result = await testValidation(
                    createSchema,
                    {
                        id: "123456789012345678",
                        teamConnect: "423456789012345678",
                    },
                    true,
                );
                expect(result.teamConnect).toBe("423456789012345678");
            });

            it("should handle omitFields for nested relations", async () => {
                const schemaWithOmit = chatValidation.create({
                    omitFields: ["invitesCreate", "messagesCreate"],
                });
                const result = await testValidation(
                    schemaWithOmit,
                    {
                        id: "123456789012345678",
                        invitesCreate: [{ id: "invalid" }], // Should be ignored
                        messagesCreate: [{ id: "invalid" }], // Should be ignored
                    },
                    true,
                );
                expect(result).to.not.have.property("invitesCreate");
                expect(result).to.not.have.property("messagesCreate");
            });
        });

        describe("update validation", () => {
            const updateSchema = chatValidation.update(defaultParams);

            it("should accept update with only id", async () => {
                const result = await testValidation(
                    updateSchema,
                    {
                        id: "123456789012345678",
                    },
                    true,
                );
                expect(result).toEqual({ id: "123456789012345678" });
            });

            it("should handle participant deletion", async () => {
                const result = await testValidation(
                    updateSchema,
                    {
                        id: "123456789012345678",
                        participantsDelete: ["523456789012345678"],
                    },
                    true,
                );
                expect(result.participantsDelete).to.have.length(1);
            });

            it("should validate complex update operations", async () => {
                try {
                    const result = await updateSchema.validate(chatFixtures.complete.update, {
                        abortEarly: false,
                        stripUnknown: true,
                    });
                    expect(result).toHaveProperty("invitesCreate");
                    expect(result).toHaveProperty("invitesUpdate");
                    expect(result).toHaveProperty("invitesDelete");
                    expect(result).toHaveProperty("messagesCreate");
                    expect(result).toHaveProperty("messagesUpdate");
                    expect(result).toHaveProperty("messagesDelete");
                } catch (error: any) {
                    console.error("Update validation error:", {
                        message: error.message,
                        errors: error.errors,
                        inner: error.inner?.map((e: any) => ({ path: e.path, message: e.message, type: e.type })),
                    });
                    throw error;
                }
            });

            it("should reject update without id", async () => {
                await testValidation(
                    updateSchema,
                    {
                        openToAnyoneWithInvite: true,
                    },
                    false,
                    /required/i,
                );
            });

            it("should handle empty arrays in update operations", async () => {
                const result = await testValidation(
                    updateSchema,
                    {
                        id: "123456789012345678",
                        invitesCreate: [],
                        invitesUpdate: [],
                        invitesDelete: [],
                        participantsDelete: [],
                    },
                    true,
                );
                expect(result.invitesCreate).to.be.an("array").with.length(0);
                expect(result.invitesUpdate).to.be.an("array").with.length(0);
                expect(result.invitesDelete).to.be.an("array").with.length(0);
                expect(result.participantsDelete).to.be.an("array").with.length(0);
            });

            it("should validate nested translation operations", async () => {
                await testValidation(
                    updateSchema,
                    chatFixtures.invalid.invalidTranslations.update,
                    false,
                );
            });
        });

        describe("edge cases", () => {
            it("should handle multiple translations", async () => {
                const createSchema = chatValidation.create(defaultParams);
                const result = await testValidation(
                    createSchema,
                    chatFixtures.edgeCases.multipleTranslations.create,
                    true,
                );
                expect(result.translationsCreate).to.have.length(3);
            });

            it("should validate max length fields", async () => {
                const createSchema = chatValidation.create(defaultParams);
                const result = await testValidation(
                    createSchema,
                    chatFixtures.edgeCases.maxLengthFields.create,
                    true,
                );
                expect(result.translationsCreate[0].name).to.have.length(50);
            });

            it("should handle empty translation arrays", async () => {
                const createSchema = chatValidation.create(defaultParams);
                const result = await testValidation(
                    createSchema,
                    chatFixtures.edgeCases.emptyTranslations.create,
                    true,
                );
                expect(result.translationsCreate).to.be.an("array").with.length(0);
            });
        });

        describe("data factory tests", () => {
            it("should generate valid create data", async () => {
                const createSchema = chatValidation.create(defaultParams);
                const generatedData = chatTestDataFactory.createMinimal();
                const result = await testValidation(
                    createSchema,
                    generatedData,
                    true,
                );
                expect(result).toHaveProperty("id");
            });

            it("should generate valid update data", async () => {
                const updateSchema = chatValidation.update(defaultParams);
                const generatedData = chatTestDataFactory.updateMinimal();
                const result = await testValidation(
                    updateSchema,
                    generatedData,
                    true,
                );
                expect(result).toHaveProperty("id");
            });
        });
    });
});

describe("chatTranslationValidation", () => {
    const defaultParams = { omitFields: [] };

    describe("create validation", () => {
        const createSchema = chatTranslationValidation.create(defaultParams);

        it("should accept minimal translation", async () => {
            const result = await testValidation(
                createSchema,
                chatTranslationFixtures.minimal.create,
                true,
            );
            expect(result).toHaveProperty("language");
        });

        it("should accept complete translation", async () => {
            const result = await testValidation(
                createSchema,
                chatTranslationFixtures.complete.create,
                true,
            );
            expect(result).toHaveProperty("language");
            expect(result).toHaveProperty("name");
            expect(result).toHaveProperty("description");
        });

        it("should reject missing language", async () => {
            await testValidation(
                createSchema,
                chatTranslationFixtures.invalid.missingRequired.create,
                false,
                /required/i,
            );
        });

        it("should reject invalid types", async () => {
            await testValidation(
                createSchema,
                chatTranslationFixtures.invalid.invalidTypes.create,
                false,
            );
        });

        it("should strip unknown fields from translations", async () => {
            const result = await testValidation(
                createSchema,
                {
                    id: "123456789012345678",
                    language: "en",
                    name: "Chat",
                    unknownField: "should be removed",
                },
                true,
            );
            expect(result).to.not.have.property("unknownField");
        });
    });

    describe("update validation", () => {
        const updateSchema = chatTranslationValidation.update(defaultParams);

        it("should accept minimal update", async () => {
            const result = await testValidation(
                updateSchema,
                chatTranslationFixtures.minimal.update,
                true,
            );
            expect(result).toHaveProperty("language");
        });

        it("should accept complete update", async () => {
            const result = await testValidation(
                updateSchema,
                chatTranslationFixtures.complete.update,
                true,
            );
            expect(result).toHaveProperty("language");
            expect(result).toHaveProperty("name");
            expect(result).toHaveProperty("description");
        });

        it("should allow partial translation updates", async () => {
            const result = await testValidation(
                updateSchema,
                {
                    id: "123456789012345678",
                    language: "en",
                    description: "Updated description only",
                },
                true,
            );
            expect(result).toHaveProperty("description");
            expect(result).to.not.have.property("name");
        });
    });
});
