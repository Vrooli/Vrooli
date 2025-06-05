import { expect } from "chai";
import { chatValidation, chatTranslationValidation } from "./chat.js";
import { chatFixtures, chatTestDataFactory, chatTranslationFixtures } from "./__test__/fixtures/chatFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test__/validationTestUtils.js";

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
                expect(result).to.have.property("id");
                expect(result).to.not.have.property("invites");
                expect(result).to.not.have.property("messages");
                expect(result).to.not.have.property("team");
                expect(result).to.not.have.property("translations");
            });

            it("should handle complex nested invites creation", async () => {
                const result = await testValidation(
                    createSchema,
                    {
                        id: "123456789012345678",
                        invites: {
                            create: [
                                {
                                    id: "223456789012345678",
                                    message: "Invite 1",
                                    user: { connect: { id: "323456789012345678" } },
                                },
                                {
                                    id: "223456789012345679",
                                    message: "Invite 2",
                                    user: { connect: { id: "323456789012345679" } },
                                },
                            ],
                        },
                    },
                    true,
                );
                expect(result.invites.create).to.have.length(2);
            });

            it("should validate nested message creation", async () => {
                const result = await testValidation(
                    createSchema,
                    {
                        id: "123456789012345678",
                        messages: {
                            create: [{
                                id: "223456789012345678",
                                content: "Test message",
                                user: { connect: { id: "323456789012345678" } },
                            }],
                        },
                    },
                    true,
                );
                expect(result.messages.create).to.have.length(1);
                expect(result.messages.create[0].content).to.equal("Test message");
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
                        team: {
                            connect: { id: "423456789012345678" },
                        },
                    },
                    true,
                );
                expect(result.team.connect.id).to.equal("423456789012345678");
            });

            it("should handle omitFields for nested relations", async () => {
                const schemaWithOmit = chatValidation.create({ 
                    omitFields: ["invites", "messages"] 
                });
                const result = await testValidation(
                    schemaWithOmit,
                    {
                        id: "123456789012345678",
                        invites: { create: [{ id: "invalid" }] }, // Should be ignored
                        messages: { create: [{ id: "invalid" }] }, // Should be ignored
                    },
                    true,
                );
                expect(result).to.not.have.property("invites");
                expect(result).to.not.have.property("messages");
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
                expect(result).to.deep.equal({ id: "123456789012345678" });
            });

            it("should handle participant deletion", async () => {
                const result = await testValidation(
                    updateSchema,
                    chatFixtures.edgeCases.onlyParticipantDelete.update,
                    true,
                );
                expect(result.participants.delete).to.have.length(1);
            });

            it("should validate complex update operations", async () => {
                const result = await testValidation(
                    updateSchema,
                    chatFixtures.complete.update,
                    true,
                );
                expect(result).to.have.property("invites");
                expect(result.invites).to.have.property("create");
                expect(result.invites).to.have.property("update");
                expect(result.invites).to.have.property("delete");
            });

            it("should reject update without id", async () => {
                await testValidation(
                    updateSchema,
                    {
                        openToAnyoneWithInvite: true,
                    },
                    false,
                    "This field is required",
                );
            });

            it("should handle empty arrays in update operations", async () => {
                const result = await testValidation(
                    updateSchema,
                    {
                        id: "123456789012345678",
                        invites: {
                            create: [],
                            update: [],
                            delete: [],
                        },
                        participants: {
                            delete: [],
                        },
                    },
                    true,
                );
                expect(result.invites.create).to.be.an("array").with.length(0);
                expect(result.invites.update).to.be.an("array").with.length(0);
                expect(result.invites.delete).to.be.an("array").with.length(0);
                expect(result.participants.delete).to.be.an("array").with.length(0);
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
                expect(result.translations.create).to.have.length(3);
            });

            it("should validate max length fields", async () => {
                const createSchema = chatValidation.create(defaultParams);
                const result = await testValidation(
                    createSchema,
                    chatFixtures.edgeCases.maxLengthFields.create,
                    true,
                );
                expect(result.translations.create[0].name).to.have.length(255);
            });

            it("should handle empty translation arrays", async () => {
                const createSchema = chatValidation.create(defaultParams);
                const result = await testValidation(
                    createSchema,
                    chatFixtures.edgeCases.emptyTranslations.create,
                    true,
                );
                expect(result.translations.create).to.be.an("array").with.length(0);
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
                expect(result).to.have.property("id");
            });

            it("should generate valid update data", async () => {
                const updateSchema = chatValidation.update(defaultParams);
                const generatedData = chatTestDataFactory.updateMinimal();
                const result = await testValidation(
                    updateSchema,
                    generatedData,
                    true,
                );
                expect(result).to.have.property("id");
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
            expect(result).to.have.property("language");
        });

        it("should accept complete translation", async () => {
            const result = await testValidation(
                createSchema,
                chatTranslationFixtures.complete.create,
                true,
            );
            expect(result).to.have.property("language");
            expect(result).to.have.property("name");
            expect(result).to.have.property("description");
        });

        it("should reject missing language", async () => {
            await testValidation(
                createSchema,
                chatTranslationFixtures.invalid.missingRequired.create,
                false,
                "This field is required",
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
            expect(result).to.have.property("language");
        });

        it("should accept complete update", async () => {
            const result = await testValidation(
                updateSchema,
                chatTranslationFixtures.complete.update,
                true,
            );
            expect(result).to.have.property("language");
            expect(result).to.have.property("name");
            expect(result).to.have.property("description");
        });

        it("should allow partial translation updates", async () => {
            const result = await testValidation(
                updateSchema,
                {
                    language: "en",
                    description: "Updated description only",
                },
                true,
            );
            expect(result).to.have.property("description");
            expect(result).to.not.have.property("name");
        });
    });
});