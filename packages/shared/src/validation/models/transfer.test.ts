import { describe, it } from "mocha";
import { expect } from "chai";
import { transferValidation, transferRequestSendValidation, transferRequestReceiveValidation } from "./transfer.js";
import { transferFixtures, transferRequestSendFixtures, transferRequestReceiveFixtures } from "./__test__/fixtures/transferFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test__/validationTestUtils.js";

describe("transferValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        transferValidation,
        transferFixtures,
        "transfer",
    );

    describe("update specific validations", () => {
        const defaultParams = { omitFields: [] };
        const updateSchema = transferValidation.update(defaultParams);

        it("should require id in update", async () => {
            await testValidation(
                updateSchema,
                transferFixtures.invalid.missingRequired.update,
                false,
                /required/i,
            );
        });

        it("should allow minimal updates with just id", async () => {
            await testValidation(
                updateSchema,
                transferFixtures.minimal.update,
                true,
            );
        });

        it("should allow updating message", async () => {
            await testValidation(
                updateSchema,
                transferFixtures.complete.update,
                true,
            );
        });

        describe("message field updates", () => {
            it("should allow updating message", async () => {
                await testValidation(
                    updateSchema,
                    transferFixtures.edgeCases.updateWithMessage.update,
                    true,
                );
            });

            it("should allow updates without message", async () => {
                await testValidation(
                    updateSchema,
                    transferFixtures.edgeCases.updateWithoutMessage.update,
                    true,
                );
            });

            it("should not allow updating object or recipient fields", async () => {
                const dataWithRestrictedFields = {
                    id: "123456789012345678",
                    objectType: "NewType",
                    objectConnect: "123456789012345679",
                    toUserConnect: "123456789012345680",
                    toTeamConnect: "123456789012345681",
                };

                const result = await testValidation(updateSchema, dataWithRestrictedFields, true);
                expect(result).to.not.have.property("objectType");
                expect(result).to.not.have.property("objectConnect");
                expect(result).to.not.have.property("toUserConnect");
                expect(result).to.not.have.property("toTeamConnect");
            });
        });

        describe("update scenarios", () => {
            it("should handle id-only updates", async () => {
                await testValidation(
                    updateSchema,
                    transferFixtures.edgeCases.updateWithoutMessage.update,
                    true,
                );
            });

            it("should handle message updates", async () => {
                await testValidation(
                    updateSchema,
                    transferFixtures.edgeCases.updateWithMessage.update,
                    true,
                );
            });
        });
    });

    describe("id validation", () => {
        const updateSchema = transferValidation.update({ omitFields: [] });

        it("should accept valid Snowflake IDs in update", async () => {
            const validIds = [
                "123456789012345678",
                "999999999999999999",
                "100000000000000000",
            ];

            const scenarios = validIds.map(id => ({
                data: {
                    ...transferFixtures.minimal.update,
                    id,
                },
                shouldPass: true,
                description: `valid ID: ${id}`,
            }));

            await testValidationBatch(updateSchema, scenarios);
        });

        it("should reject invalid IDs in update", async () => {
            const invalidIds = [
                { id: "not-a-number", error: /Must be a valid ID/i },
                { id: "abc123", error: /Must be a valid ID/i },
                { id: "", error: /required/i },
                { id: "-123", error: /Must be a valid ID/i },
                { id: "0", error: /Must be a valid ID/i },
            ];

            for (const { id, error } of invalidIds) {
                await testValidation(
                    updateSchema,
                    { ...transferFixtures.minimal.update, id },
                    false,
                    error,
                );
            }
        });
    });

    describe("type conversions", () => {
        const updateSchema = transferValidation.update({ omitFields: [] });

        it("should handle ID conversion", async () => {
            const dataWithNumberId = {
                id: 123456789012345, // Smaller number to avoid precision issues
                message: "ID conversion test",
            };

            const result = await testValidation(
                updateSchema,
                dataWithNumberId,
                true,
            );
            expect(result.id).to.be.a("string");
            expect(result.id).to.equal("123456789012345");
        });
    });

    describe("edge cases", () => {
        const updateSchema = transferValidation.update({ omitFields: [] });

        it("should handle minimal transfer updates", async () => {
            await testValidation(
                updateSchema,
                transferFixtures.minimal.update,
                true,
            );
        });

        it("should handle empty update operations", async () => {
            await testValidation(
                updateSchema,
                transferFixtures.edgeCases.updateWithoutMessage.update,
                true,
            );
        });
    });
});

describe("transferRequestSendValidation", () => {
    describe("create specific validations", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = transferRequestSendValidation(defaultParams);

        it("should require id in create", async () => {
            const dataWithoutId = { ...transferRequestSendFixtures.minimal.create };
            delete dataWithoutId.id;

            await testValidation(
                createSchema,
                dataWithoutId,
                false,
                /required/i,
            );
        });

        it("should require objectType in create", async () => {
            const dataWithoutObjectType = {
                id: "123456789012345678",
                objectConnect: "123456789012345679",
                toUserConnect: "123456789012345680",
                // Missing required objectType
            };

            await testValidation(
                createSchema,
                dataWithoutObjectType,
                false,
                /required/i,
            );
        });

        it("should require objectConnect in create", async () => {
            await testValidation(
                createSchema,
                transferRequestSendFixtures.invalid.missingRequired.create,
                false,
                /required/i,
            );
        });

        it("should require recipient (toUser or toTeam) in create", async () => {
            const dataWithoutRecipient = {
                id: "123456789012345678",
                objectType: "Resource",
                objectConnect: "123456789012345679",
                // Missing both toUserConnect and toTeamConnect
            };

            await testValidation(
                createSchema,
                dataWithoutRecipient,
                false,
                /Only one of the following fields can be present/i,
            );
        });

        it("should allow minimal create with required fields", async () => {
            await testValidation(
                createSchema,
                transferRequestSendFixtures.minimal.create,
                true,
            );
        });

        it("should allow complete create with all fields", async () => {
            await testValidation(
                createSchema,
                transferRequestSendFixtures.complete.create,
                true,
            );
        });

        describe("objectType field", () => {
            it("should accept valid object types", async () => {
                const scenarios = [
                    {
                        data: transferFixtures.edgeCases.apiTransfer.create,
                        shouldPass: true,
                        description: "Resource transfer (via API fixture)",
                    },
                    {
                        data: transferFixtures.edgeCases.routineTransfer.create,
                        shouldPass: true,
                        description: "Resource transfer (via routine fixture)",
                    },
                    {
                        data: transferFixtures.edgeCases.projectTransfer.create,
                        shouldPass: true,
                        description: "Resource transfer (via project fixture)",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject invalid object types", async () => {
                await testValidation(
                    createSchema,
                    transferRequestSendFixtures.invalid.invalidTypes.create,
                    false,
                    /must be one of/i,
                );
            });
        });

        describe("message field", () => {
            it("should be optional", async () => {
                await testValidation(
                    createSchema,
                    transferFixtures.edgeCases.withoutMessage.create,
                    true,
                );
            });

            it("should accept valid messages", async () => {
                const scenarios = [
                    {
                        data: transferFixtures.edgeCases.withMessage.create,
                        shouldPass: true,
                        description: "message with details",
                    },
                    {
                        data: transferFixtures.edgeCases.longMessage.create,
                        shouldPass: true,
                        description: "long message",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });
        });

        describe("object relationships", () => {
            it("should require objectConnect", async () => {
                const dataWithoutObject = {
                    id: "123456789012345678",
                    objectType: "Resource",
                    toUserConnect: "123456789012345679",
                    // Missing required objectConnect
                };

                await testValidation(
                    createSchema,
                    dataWithoutObject,
                    false,
                    /required/i,
                );
            });

            it("should accept valid object connections", async () => {
                await testValidation(
                    createSchema,
                    transferRequestSendFixtures.minimal.create,
                    true,
                );
            });

            it("should handle resource objects", async () => {
                const scenarios = [
                    {
                        data: transferFixtures.edgeCases.apiTransfer.create,
                        shouldPass: true,
                        description: "Resource object (via API fixture)",
                    },
                    {
                        data: transferFixtures.edgeCases.routineTransfer.create,
                        shouldPass: true,
                        description: "Resource object (via routine fixture)",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });
        });

        describe("recipient relationships", () => {
            it("should accept toUserConnect", async () => {
                await testValidation(
                    createSchema,
                    transferRequestSendFixtures.edgeCases.sendToUser.create,
                    true,
                );
            });

            it("should accept toTeamConnect", async () => {
                await testValidation(
                    createSchema,
                    transferRequestSendFixtures.edgeCases.sendToTeam.create,
                    true,
                );
            });

            it("should enforce exclusivity between toUser and toTeam", async () => {
                await testValidation(
                    createSchema,
                    transferRequestSendFixtures.invalid.bothRecipients.create,
                    false,
                    /Only one of the following fields can be present/i,
                );
            });

            it("should require at least one recipient", async () => {
                const dataWithoutRecipient = {
                    id: "123456789012345678",
                    objectType: "Resource",
                    objectConnect: "123456789012345679",
                    // Missing both recipients
                };

                await testValidation(
                    createSchema,
                    dataWithoutRecipient,
                    false,
                    /Only one of the following fields can be present/i,
                );
            });
        });

        describe("transfer scenarios", () => {
            it("should handle user-to-user transfers", async () => {
                await testValidation(
                    createSchema,
                    transferFixtures.edgeCases.transferToUser.create,
                    true,
                );
            });

            it("should handle user-to-team transfers", async () => {
                await testValidation(
                    createSchema,
                    transferFixtures.edgeCases.transferToTeam.create,
                    true,
                );
            });

            it("should handle transfers with messages", async () => {
                await testValidation(
                    createSchema,
                    transferFixtures.edgeCases.withMessage.create,
                    true,
                );
            });

            it("should handle transfers without messages", async () => {
                await testValidation(
                    createSchema,
                    transferFixtures.edgeCases.withoutMessage.create,
                    true,
                );
            });
        });
    });
});

describe("transferRequestReceiveValidation", () => {
    describe("create specific validations", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = transferRequestReceiveValidation(defaultParams);

        it("should require id in create", async () => {
            const dataWithoutId = { ...transferRequestReceiveFixtures.minimal.create };
            delete dataWithoutId.id;

            await testValidation(
                createSchema,
                dataWithoutId,
                false,
                /required/i,
            );
        });

        it("should require objectType in create", async () => {
            const dataWithoutObjectType = {
                id: "123456789012345678",
                objectConnect: "123456789012345679",
                // Missing required objectType
            };

            await testValidation(
                createSchema,
                dataWithoutObjectType,
                false,
                /required/i,
            );
        });

        it("should require objectConnect in create", async () => {
            await testValidation(
                createSchema,
                transferRequestReceiveFixtures.invalid.missingRequired.create,
                false,
                /required/i,
            );
        });

        it("should allow minimal create with required fields", async () => {
            await testValidation(
                createSchema,
                transferRequestReceiveFixtures.minimal.create,
                true,
            );
        });

        it("should allow complete create with all fields", async () => {
            await testValidation(
                createSchema,
                transferRequestReceiveFixtures.complete.create,
                true,
            );
        });

        describe("objectType field", () => {
            it("should accept valid object types", async () => {
                const scenarios = [
                    {
                        data: {
                            id: "123456789012345678",
                            objectType: "Resource",
                            objectConnect: "123456789012345679",
                        },
                        shouldPass: true,
                        description: "Resource receive",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject invalid object types", async () => {
                await testValidation(
                    createSchema,
                    transferRequestReceiveFixtures.invalid.invalidTypes.create,
                    false,
                    /must be one of/i,
                );
            });
        });

        describe("message field", () => {
            it("should be optional", async () => {
                await testValidation(
                    createSchema,
                    transferRequestReceiveFixtures.edgeCases.receiveWithoutTeam.create,
                    true,
                );
            });

            it("should accept valid messages", async () => {
                await testValidation(
                    createSchema,
                    transferRequestReceiveFixtures.complete.create,
                    true,
                );
            });
        });

        describe("object relationships", () => {
            it("should require objectConnect", async () => {
                const dataWithoutObject = {
                    id: "123456789012345678",
                    objectType: "Resource",
                    // Missing required objectConnect
                };

                await testValidation(
                    createSchema,
                    dataWithoutObject,
                    false,
                    /required/i,
                );
            });

            it("should accept valid object connections", async () => {
                await testValidation(
                    createSchema,
                    transferRequestReceiveFixtures.minimal.create,
                    true,
                );
            });
        });

        describe("toTeam relationships", () => {
            it("should accept toTeamConnect (optional)", async () => {
                await testValidation(
                    createSchema,
                    transferRequestReceiveFixtures.edgeCases.receiveForTeam.create,
                    true,
                );
            });

            it("should allow receiving without toTeamConnect", async () => {
                await testValidation(
                    createSchema,
                    transferRequestReceiveFixtures.edgeCases.receiveWithoutTeam.create,
                    true,
                );
            });
        });

        describe("receive scenarios", () => {
            it("should handle receiving for a team", async () => {
                await testValidation(
                    createSchema,
                    transferRequestReceiveFixtures.edgeCases.receiveForTeam.create,
                    true,
                );
            });

            it("should handle receiving for personal use", async () => {
                await testValidation(
                    createSchema,
                    transferRequestReceiveFixtures.edgeCases.receiveWithoutTeam.create,
                    true,
                );
            });

            it("should handle receiving with messages", async () => {
                await testValidation(
                    createSchema,
                    transferRequestReceiveFixtures.complete.create,
                    true,
                );
            });
        });
    });
});

describe("transfer validation edge cases", () => {
    describe("ID validation across all functions", () => {
        const sendSchema = transferRequestSendValidation({ omitFields: [] });
        const receiveSchema = transferRequestReceiveValidation({ omitFields: [] });
        const updateSchema = transferValidation.update({ omitFields: [] });

        it("should accept valid Snowflake IDs in all schemas", async () => {
            const validIds = [
                "123456789012345678",
                "999999999999999999",
                "100000000000000000",
            ];

            for (const id of validIds) {
                // Test send schema
                await testValidation(
                    sendSchema,
                    { ...transferRequestSendFixtures.minimal.create, id },
                    true,
                );

                // Test receive schema
                await testValidation(
                    receiveSchema,
                    { ...transferRequestReceiveFixtures.minimal.create, id },
                    true,
                );

                // Test update schema
                await testValidation(
                    updateSchema,
                    { ...transferFixtures.minimal.update, id },
                    true,
                );
            }
        });
    });

    describe("type conversions", () => {
        const sendSchema = transferRequestSendValidation({ omitFields: [] });

        it("should handle ID conversion", async () => {
            const dataWithNumberId = {
                id: 123456789012345, // Smaller number to avoid precision issues
                objectType: "Resource",
                objectConnect: "123456789012345679",
                toUserConnect: "123456789012345680",
            };

            const result = await testValidation(
                sendSchema,
                dataWithNumberId,
                true,
            );
            expect(result.id).to.be.a("string");
            expect(result.id).to.equal("123456789012345");
        });
    });

    describe("comprehensive transfer scenarios", () => {
        const sendSchema = transferRequestSendValidation({ omitFields: [] });
        const receiveSchema = transferRequestReceiveValidation({ omitFields: [] });

        it("should handle resource object type in send requests", async () => {
            const data = {
                id: "123456789012345678",
                objectType: "Resource",
                objectConnect: "123456789012345679",
                toUserConnect: "123456789012345680",
            };

            await testValidation(sendSchema, data, true);
        });

        it("should handle resource object type in receive requests", async () => {
            const data = {
                id: "123456789012345678",
                objectType: "Resource",
                objectConnect: "123456789012345679",
            };

            await testValidation(receiveSchema, data, true);
        });
    });
});