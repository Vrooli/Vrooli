/* eslint-disable @typescript-eslint/ban-ts-comment */
import { } from "@vrooli/shared";
import { describe, it, expect, beforeAll, afterAll, beforeEach } from "vitest";
import sinon from "sinon";
import { DbProvider } from "../db/provider.js";
import { logger } from "../events/logger.js";
import { convertPlaceholders, determineModelType, fetchAndMapPlaceholder, initializeInputMaps, inputToMaps, processConnectDisconnectOrDelete, processCreateOrUpdate, processInputObjectField, replacePlaceholdersInInputsById, replacePlaceholdersInInputsByType, replacePlaceholdersInMap, updateClosestWithId } from "./cudInputsToMaps.js";
import { InputNode } from "./inputNode.js";
import { type IdsByAction, type IdsByType, type InputsByType } from "./types.js";

describe("fetchAndMapPlaceholder", () => {
    let placeholderToIdMap;
    const userId = "user-10001";
    const premiumId = "premium-10002";
    const teamId = "team-10003";

    beforeAll(async function beforeAll() {
        // Initialize the ModelMap, which is used in fetchAndMapPlaceholder
        await DbProvider.deleteAll();
        await DbProvider.get().user.create({
            data: {
                id: userId,
                name: "Test User",
                premium: {
                    create: {
                        id: premiumId,
                        team: {
                            create: {
                                id: teamId,
                                permissions: JSON.stringify({}),
                            },
                        },
                    },
                },
            },
        });
    });

    beforeEach(() => {
        placeholderToIdMap = {};
    });

    afterAll(async function afterAll() {
        await DbProvider.deleteAll();
    });

    it("should fetch and return the correct ID for a new placeholder", async () => {
        const placeholder = `user|${userId}.prem|premium`;
        await fetchAndMapPlaceholder(placeholder, placeholderToIdMap);
        expect(placeholderToIdMap[placeholder]).to.equal(premiumId); // The profile's ID
    });

    it("should return the cached ID for an already processed placeholder", async () => {
        const placeholder = `user|${userId}.prem|premium`;
        placeholderToIdMap[placeholder] = premiumId;

        await fetchAndMapPlaceholder(placeholder, placeholderToIdMap);

        expect(placeholderToIdMap[placeholder]).to.equal(premiumId); // The cached ID
    });

    it("should return undefined if object is not found", async () => {
        const placeholder = `user|${"user-10004"}.prof|profile`;

        try {
            await fetchAndMapPlaceholder(placeholder, placeholderToIdMap);
            expect.fail("Expected fetchAndMapPlaceholder to throw");
        } catch (error) {
            expect(placeholderToIdMap[placeholder]).to.be.undefined;

        }
    });

    it("should handle nested relations correctly", async () => {
        const placeholder = `user|${userId}.prem|premium.owner|team`;
        await fetchAndMapPlaceholder(placeholder, placeholderToIdMap);

        expect(placeholderToIdMap[placeholder]).to.equal(teamId);
    });

    it("should handle placeholder with invalid format", async () => {
        const placeholder = "invalidFormatPlaceholder"; // No delimiters or parts
        try {
            await fetchAndMapPlaceholder(placeholder, placeholderToIdMap);
            expect.fail("Expected fetchAndMapPlaceholder to throw");
        } catch (error) {
            expect(placeholderToIdMap[placeholder]).to.be.undefined;
        }
    });

    it("should handle non-existent objectType in placeholder", async () => {
        const placeholder = "nonExistentType|123.relation|value";
        try {
            await fetchAndMapPlaceholder(placeholder, placeholderToIdMap);
            expect.fail("Expected fetchAndMapPlaceholder to throw");
        } catch (error) {
            expect(placeholderToIdMap[placeholder]).to.be.undefined;
        }
    });

    it("should handle valid objectType but invalid rootId", async () => {
        const placeholder = `user|${"user-10005"}.prem|premium`;
        try {
            await fetchAndMapPlaceholder(placeholder, placeholderToIdMap);
            expect.fail("Expected fetchAndMapPlaceholder to throw");
        } catch (error) {
            expect(placeholderToIdMap[placeholder]).to.be.undefined;
        }
    });

    it("should handle unnecessary placeholders (placeholders which contain the ID already)", async () => {
        const noteId = "note-10006";
        const placeholder = `Note|${noteId}`;
        await fetchAndMapPlaceholder(placeholder, placeholderToIdMap);

        expect(placeholderToIdMap[placeholder]).to.equal(noteId);
    });
});

describe("replacePlaceholdersInMap", () => {
    let placeholderToIdMap;
    const userId = "user-10001";
    const premiumId = "premium-10002";
    const teamId = "team-10003";
    const noteId = "note-10007";

    beforeAll(async function beforeAll() {
        await DbProvider.deleteAll();
        await DbProvider.get().user.create({
            data: {
                id: userId,
                name: "Bugs Bunny",
                premium: {
                    create: {
                        id: premiumId,
                        team: {
                            create: {
                                id: teamId,
                                permissions: JSON.stringify({}),
                            },
                        },
                    },
                },
            },
        });
        await DbProvider.get().note.create({
            data: {
                id: noteId,
                ownedByUser: { connect: { id: userId } },
                permissions: JSON.stringify({}),
            },
        });
    });

    beforeEach(() => {
        placeholderToIdMap = {};
    });

    afterAll(async function afterAll() {
        await DbProvider.deleteAll();
    });

    it("should replace placeholders with actual IDs", async () => {
        const testCases = [
            [`User|${userId}.Premium|premium`, premiumId],
            [`User|${userId}`, userId],
        ];
        const idsMap = {
            "User": [testCases[0][0]],
            "Update": [testCases[1][0]],
        };

        await replacePlaceholdersInMap(idsMap, placeholderToIdMap);

        const expectedKeyLength = Object.values(idsMap).flat().length;
        expect(Object.keys(placeholderToIdMap)).to.have.lengthOf(expectedKeyLength);
        expect(idsMap["User"]).to.deep.equal([testCases[0][1]]);
        expect(idsMap["Update"]).to.deep.equal([testCases[1][1]]);
        expect(placeholderToIdMap[testCases[0][0]]).to.deep.equal(testCases[0][1]);
        expect(placeholderToIdMap[testCases[1][0]]).to.deep.equal(testCases[1][1]);
    });

    it("should handle multiple placeholders", async () => {
        const testCases = [
            [`User|${userId}.Premium|premium`, premiumId],
            [`User|${userId}.Premium|premium.Team|team`, teamId],
            [`User|${userId}`, userId],
            [`Note|${noteId}.User|ownedByUser`, userId],
        ];
        const idsMap = {
            "User": [testCases[0][0], testCases[1][0]],
            "Update": [testCases[2][0], testCases[3][0]],
        };

        await replacePlaceholdersInMap(idsMap, placeholderToIdMap);

        const expectedKeyLength = Object.values(idsMap).flat().length;
        expect(Object.keys(placeholderToIdMap)).to.have.lengthOf(expectedKeyLength);
        expect(idsMap["User"]).to.deep.equal([testCases[0][1], testCases[1][1]]);
        expect(idsMap["Update"]).to.deep.equal([testCases[2][1], testCases[3][1]]);
        expect(placeholderToIdMap[testCases[0][0]]).to.deep.equal(testCases[0][1]);
        expect(placeholderToIdMap[testCases[1][0]]).to.deep.equal(testCases[1][1]);
        expect(placeholderToIdMap[testCases[2][0]]).to.deep.equal(testCases[2][1]);
        expect(placeholderToIdMap[testCases[3][0]]).to.deep.equal(testCases[3][1]);
    });

    it("should leave non-placeholder IDs unchanged", async () => {
        const id1 = "id-10008";
        const id2 = "id-10009";
        const testCases = [
            [id1, id1],
            [`User|${userId}.Premium|premium`, premiumId],
            [id2, id2],
        ];
        const idsMap = {
            "User": [testCases[0][0], testCases[1][0]],
            "Update": [testCases[2][0]],
        };

        await replacePlaceholdersInMap(idsMap, placeholderToIdMap);

        expect(Object.keys(placeholderToIdMap)).to.have.lengthOf(1); // There's only one actual placeholder this time
        expect(idsMap["User"]).to.deep.equal([testCases[0][1], testCases[1][1]]);
        expect(idsMap["Update"]).to.deep.equal([testCases[2][1]]);
        expect(placeholderToIdMap[testCases[0][0]]).to.be.undefined; // Not a placeholder, so it's not in the map
        expect(placeholderToIdMap[testCases[1][0]]).to.deep.equal(testCases[1][1]);
        expect(placeholderToIdMap[testCases[2][0]]).to.be.undefined; // Not a placeholder, so it's not in the map
    });

    it("should skip querying database when placeholder is already in map", async () => {
        const placeholder = "User|123.fdsafkafdafsa"; // Not valid, but should be fine since its corresponding ID will already be in the map
        const id = "420";
        const idsMap = {
            "User": [placeholder],
        };
        placeholderToIdMap[placeholder] = id;

        await replacePlaceholdersInMap(idsMap, placeholderToIdMap);

        expect(idsMap["User"][0]).to.deep.equal(id);
        expect(Object.keys(placeholderToIdMap)).to.have.lengthOf(1);
        expect(placeholderToIdMap[placeholder]).to.deep.equal(id);
    });
});

describe("replacePlaceholdersInInputsById", () => {
    let placeholderToIdMap;
    const userId = "user-10001";
    const premiumId = "premium-10002";
    const teamId = "team-10003";

    beforeAll(async function beforeAll() {
        await DbProvider.deleteAll();
        await DbProvider.get().user.create({
            data: {
                id: userId,
                name: "Bugs Bunny",
                premium: {
                    create: {
                        id: premiumId,
                        team: {
                            create: {
                                id: teamId,
                                permissions: JSON.stringify({}),
                            },
                        },
                    },
                },
            },
        });
    });

    beforeEach(() => {
        placeholderToIdMap = {};
    });

    afterAll(async function afterAll() {
        await DbProvider.deleteAll();
    });

    it("should replace placeholders with string (e.g. 'Delete') inputs", async () => {
        const placeholder = `User|${userId}.Premium|premium`;
        // Define input that says: "Delete the premium relationship of the user"
        const inputsById = { [placeholder]: { node: new InputNode("Premium", placeholder, "Delete"), input: placeholder } };

        await replacePlaceholdersInInputsById(inputsById, placeholderToIdMap);

        expect(Object.keys(inputsById)).to.have.lengthOf(1); // Same as what we started with
        expect(inputsById[placeholder]).to.be.undefined; // The placeholder is now replaced with the ID
        expect(inputsById[premiumId].node.id).to.deep.equal(premiumId); // The node ID is the ID of the object
        expect(inputsById[premiumId].input).to.deep.equal(premiumId); // The input (which for deletes is just an ID) is the ID of the object
        expect(Object.keys(placeholderToIdMap)).to.have.lengthOf(1); // The placeholder is now in the map
        expect(placeholderToIdMap[placeholder]).to.deep.equal(premiumId); // The placeholder is now mapped to the ID
    });

    it("should replace placeholders with object (e.g. 'Update') inputs", async () => {
        const placeholder = `User|${userId}.Premium|premium`;
        // Define input that says: "UPdate the premium relationship of the user"
        const inputsById = {
            [placeholder]: {
                node: new InputNode("Premium", placeholder, "Update"),
                input: {
                    id: placeholder,
                    name: "Updated name",
                },
            },
        };

        await replacePlaceholdersInInputsById(inputsById, placeholderToIdMap);

        expect(Object.keys(inputsById)).to.have.lengthOf(1); // Same as what we started with
        expect(inputsById[placeholder]).to.be.undefined; // The placeholder is now replaced with the ID
        expect(inputsById[premiumId].node.id).to.deep.equal(premiumId); // The node ID is the ID of the object
        expect(inputsById[premiumId].input).to.deep.equal({ id: premiumId, name: "Updated name" });
        expect(Object.keys(placeholderToIdMap)).to.have.lengthOf(1); // The placeholder is now in the map
        expect(placeholderToIdMap[placeholder]).to.deep.equal(premiumId); // The placeholder is now mapped to the ID
    });

    it("should leave non-placeholders unchanged", async () => {
        const placeholder = `User|${userId}.Premium|premium`;
        const inputsById = {
            [placeholder]: { node: new InputNode("Premium", placeholder, "Delete"), input: placeholder },
            "999": { node: new InputNode("User", "999", "Delete"), input: "999" },
        };

        await replacePlaceholdersInInputsById(inputsById, placeholderToIdMap);

        expect(Object.keys(inputsById)).to.have.lengthOf(2);
        expect(inputsById[placeholder]).to.be.undefined;
        expect(inputsById["999"].node.id).to.deep.equal("999");
        expect(inputsById[premiumId].node.id).to.deep.equal(premiumId);
        expect(inputsById[premiumId].input).to.deep.equal(premiumId);
        expect(Object.keys(placeholderToIdMap)).to.have.lengthOf(1);
        expect(placeholderToIdMap[placeholder]).to.deep.equal(premiumId);
    });

    it("should skip querying database when placeholder is already in map", async () => {
        const placeholder = `User|${userId}.Premium|premium`;
        const inputsById = { [placeholder]: { node: new InputNode("Premium", placeholder, "Delete"), input: placeholder } };
        const id = "420"; // Use a different ID so we know that it didn't use the database
        placeholderToIdMap[placeholder] = id;

        await replacePlaceholdersInInputsById(inputsById, placeholderToIdMap);

        expect(Object.keys(inputsById)).to.have.lengthOf(1);
        expect(inputsById[placeholder]).to.be.undefined;
        expect(inputsById[id].node.id).to.deep.equal(id);
        expect(inputsById[id].input).to.deep.equal(id);
        expect(Object.keys(placeholderToIdMap)).to.have.lengthOf(1);
        expect(placeholderToIdMap[placeholder]).to.deep.equal(id);
    });
});

describe("replacePlaceholdersInInputsByType", () => {
    let placeholderToIdMap;
    const userId = "user-10001";
    const premiumId = "premium-10002";
    const teamId = "team-10003";

    beforeAll(async function beforeAll() {
        await DbProvider.deleteAll();
        await DbProvider.get().user.create({
            data: {
                id: userId,
                name: "Bugs Bunny",
                premium: {
                    create: {
                        id: premiumId,
                        team: {
                            create: {
                                id: teamId,
                                permissions: JSON.stringify({}),
                            },
                        },
                    },
                },
            },
        });
    });

    beforeEach(() => {
        placeholderToIdMap = {};
    });

    afterAll(async function afterAll() {
        await DbProvider.deleteAll();
    });

    it("should replace placeholders with string (e.g. 'Delete') inputs", async () => {
        const placeholder = `User|${userId}.Premium|premium.Team|team`;
        const inputsByType = {
            Team: {
                Connect: [],
                Create: [],
                Delete: [{ node: new InputNode("Team", placeholder, "Delete"), input: placeholder }],
                Disconnect: [],
                Read: [],
                Update: [],
            },
        };

        await replacePlaceholdersInInputsByType(inputsByType, placeholderToIdMap);

        expect(inputsByType.Team.Delete).to.have.lengthOf(1);
        expect(inputsByType.Team.Delete[0].node.id).to.deep.equal(teamId);
        expect(inputsByType.Team.Delete[0].input).to.deep.equal(teamId);
        expect(Object.keys(placeholderToIdMap)).to.have.lengthOf(1);
        expect(placeholderToIdMap[placeholder]).to.deep.equal(teamId);
    });

    it("should replace placeholders with object (e.g. 'Update') inputs", async () => {
        const placeholder = `User|${userId}.Premium|premium.Team|team`;
        const inputsByType = {
            Team: {
                Connect: [],
                Create: [],
                Delete: [],
                Disconnect: [],
                Read: [],
                Update: [{ node: new InputNode("Team", placeholder, "Update"), input: { id: placeholder, name: "Test" } }],
            },
        };

        await replacePlaceholdersInInputsByType(inputsByType, placeholderToIdMap);

        expect(inputsByType.Team.Update).to.have.lengthOf(1);
        expect(inputsByType.Team.Update[0].node.id).to.deep.equal(teamId);
        expect(inputsByType.Team.Update[0].input).to.deep.equal({ id: teamId, name: "Test" });
        expect(Object.keys(placeholderToIdMap)).to.have.lengthOf(1);
        expect(placeholderToIdMap[placeholder]).to.deep.equal(teamId);
    });

    it("should leave non-placeholders unchanged", async () => {
        const placeholder = `User|${userId}`;
        const nonPlaceholder = "999";
        const inputsByType = {
            User: {
                Connect: [],
                Create: [],
                Delete: [
                    { node: new InputNode("User", placeholder, "Delete"), input: placeholder },
                    { node: new InputNode("User", nonPlaceholder, "Delete"), input: nonPlaceholder },
                ],
                Disconnect: [],
                Read: [],
                Update: [],
            },
        };

        await replacePlaceholdersInInputsByType(inputsByType, placeholderToIdMap);

        expect(inputsByType.User.Delete).to.have.lengthOf(2);
        expect(inputsByType.User.Delete[0].node.id).to.deep.equal(userId);
        expect(inputsByType.User.Delete[0].input).to.deep.equal(userId);
        expect(inputsByType.User.Delete[1].node.id).to.deep.equal(nonPlaceholder);
        expect(inputsByType.User.Delete[1].input).to.deep.equal(nonPlaceholder);
        expect(Object.keys(placeholderToIdMap)).to.have.lengthOf(1);
        expect(placeholderToIdMap[placeholder]).to.deep.equal(userId);
    });

    it("should skip querying database when placeholder is already in map", async () => {
        const placeholder = `User|${userId}`;
        const id = "420"; // Use a different ID so we know that it didn't use the database
        placeholderToIdMap[placeholder] = id;

        const inputsByType = {
            User: {
                Connect: [],
                Create: [],
                Delete: [{ node: new InputNode("User", placeholder, "Delete"), input: placeholder }],
                Disconnect: [],
                Read: [],
                Update: [],
            },
        };

        await replacePlaceholdersInInputsByType(inputsByType, placeholderToIdMap);

        expect(inputsByType.User.Delete).to.have.lengthOf(1);
        expect(inputsByType.User.Delete[0].node.id).to.deep.equal(id);
        expect(inputsByType.User.Delete[0].input).to.deep.equal(id);
        expect(Object.keys(placeholderToIdMap)).to.have.lengthOf(1);
        expect(placeholderToIdMap[placeholder]).to.deep.equal(id);
    });
});

describe("convertPlaceholders", () => {
    let idsByAction, idsByType, inputsById, inputsByType;
    const initialIdsByAction = { Create: [], Update: [], Connect: [], Disconnect: [] };
    const initialIdsByType = {};
    const initialInputsById = {};
    const initialInputsByType = {};
    const userId1 = "user-10010";
    const userId2 = "user-10011";
    const premiumId = "premium-10012";
    const teamId = "team-10013";

    beforeAll(async function beforeAll() {
        await DbProvider.deleteAll();
        await DbProvider.get().user.create({
            data: {
                id: userId1,
                name: "Bugs Bunny",
                premium: {
                    create: {
                        id: premiumId,
                        team: {
                            create: {
                                id: teamId,
                                permissions: JSON.stringify({}),
                            },
                        },
                    },
                },
            },
        });
        await DbProvider.get().user.create({
            data: {
                id: userId2,
                name: "Daffy Duck",
            },
        });
    });

    beforeEach(() => {
        idsByAction = JSON.parse(JSON.stringify(initialIdsByAction));
        idsByType = JSON.parse(JSON.stringify(initialIdsByType));
        inputsById = JSON.parse(JSON.stringify(initialInputsById));
        inputsByType = JSON.parse(JSON.stringify(initialInputsByType));
    });

    afterAll(async function afterAll() {
        await DbProvider.deleteAll();
    });

    it("should replace placeholders with actual IDs", async () => {
        const placeholder = `User|${userId1}.Premium|premium`;
        idsByType = { "Premium": [placeholder] };
        idsByAction = { "Update": [placeholder] };
        inputsById = { [placeholder]: { node: new InputNode("Premium", placeholder, "Delete"), input: placeholder } };
        inputsByType = { "Premium": { "Delete": [{ node: new InputNode("Premium", placeholder, "Delete"), input: placeholder }] } };

        await convertPlaceholders({ idsByAction, idsByType, inputsById, inputsByType });

        expect(idsByType["Premium"]).to.deep.equal([premiumId]);
        expect(idsByAction["Update"]).to.deep.equal([premiumId]);
        expect(Object.keys(inputsById)).to.have.lengthOf(1);
        expect(inputsById[premiumId].node.id).to.deep.equal(premiumId);
        expect(inputsById[premiumId].input).to.deep.equal(premiumId);
        expect(inputsByType["Premium"].Delete).to.have.lengthOf(1);
        expect(inputsByType["Premium"].Delete[0].node.id).to.deep.equal(premiumId);
        expect(inputsByType["Premium"].Delete[0].input).to.deep.equal(premiumId);
        expect(inputsByType["Premium"].Delete).to.have.lengthOf(1);
    });

    it("should not replace placeholders for objects that don't exist", async () => {
        const userIdNotInDb = "user-10014";
        const placeholder = `User|${userIdNotInDb}.Premium|premium`;
        idsByType = { "Premium": [placeholder] };
        idsByAction = { "Create": [placeholder] };
        inputsById = { [placeholder]: { node: new InputNode("Premium", placeholder, "Create"), input: placeholder } };
        inputsByType = { "Premium": { "Create": [{ node: new InputNode("Premium", placeholder, "Create"), input: placeholder }] } };

        await convertPlaceholders({ idsByAction, idsByType, inputsById, inputsByType });

        expect(idsByType["Premium"]).to.deep.equal([undefined]);
        expect(idsByAction["Create"]).to.deep.equal([undefined]);
        expect(Object.keys(inputsById)).to.have.lengthOf(0);
        expect(inputsById[placeholder]).to.be.undefined;
        expect(inputsByType["Premium"].Create).to.have.lengthOf(1);
        expect(inputsByType["Premium"].Create[0].node.id).to.deep.equal(undefined);
        expect(inputsByType["Premium"].Create[0].input).to.deep.equal(undefined);
    });

    it("should not perform any operation if there are no placeholders", async () => {
        const id = "123"; // Use a non-uuid so we know that it didn't call the database
        idsByType = { "User": [id] };
        idsByAction = { "Create": [id] };
        inputsById = { [id]: { node: new InputNode("User", id, "Create"), input: id } };
        inputsByType = { "User": { "Create": [{ node: new InputNode("User", id, "Create"), input: id }] } };

        await convertPlaceholders({ idsByAction, idsByType, inputsById, inputsByType });

        expect(idsByType["User"]).to.deep.equal([id]);
        expect(idsByAction["Create"]).to.deep.equal([id]);
        expect(Object.keys(inputsById)).to.have.lengthOf(1);
        expect(inputsById[id].node.id).to.deep.equal(id);
        expect(inputsById[id].input).to.deep.equal(id);
        expect(inputsByType["User"].Create).to.have.lengthOf(1);
        expect(inputsByType["User"].Create[0].node.id).to.deep.equal(id);
        expect(inputsByType["User"].Create[0].input).to.deep.equal(id);
    });

    it("should handle multiple placeholders and nested cases correctly", async () => {
        const placeholder1 = `User|${userId1}.Premium|premium`;
        const placeholder2 = `User|${userId1}.Premium|premium.Team|team`;
        const placeholder3 = `User|${userId1}`;
        const placeholder4 = `User|${userId2}`;
        idsByType = {
            "Premium": [placeholder1],
            "Team": [placeholder2],
            "User": [placeholder3, placeholder4],
        };
        idsByAction = {
            "Update": [placeholder1, placeholder3, placeholder4],
            "Delete": [placeholder2],
        };
        inputsById = {
            [placeholder1]: { node: new InputNode("Premium", placeholder1, "Update"), input: { id: placeholder1, name: "Update 1" } },
            [placeholder2]: { node: new InputNode("Team", placeholder2, "Delete"), input: placeholder2 },
            [placeholder3]: { node: new InputNode("User", placeholder3, "Update"), input: { id: placeholder3, name: "Update 2" } },
            [placeholder4]: { node: new InputNode("User", placeholder4, "Update"), input: { id: placeholder4, name: "Update 3" } },
        };
        inputsByType = {
            "Premium": { "Update": [{ node: new InputNode("Premium", placeholder1, "Update"), input: { id: placeholder1, name: "Update 1" } }] },
            "Team": { "Delete": [{ node: new InputNode("Team", placeholder2, "Delete"), input: placeholder2 }] },
            "User": {
                "Update": [
                    { node: new InputNode("User", placeholder3, "Update"), input: { id: placeholder3, name: "Update 2" } },
                    { node: new InputNode("User", placeholder4, "Update"), input: { id: placeholder4, name: "Update 3" } },
                ],
            },
        };

        await convertPlaceholders({ idsByAction, idsByType, inputsById, inputsByType });

        expect(idsByType["Premium"]).to.deep.equal([premiumId]);
        expect(idsByType["Team"]).to.deep.equal([teamId]);
        expect(idsByType["User"]).to.deep.equal([userId1, userId2]);
        expect(Object.keys(inputsById)).to.have.lengthOf(4);
        expect(inputsById[premiumId].node.id).to.deep.equal(premiumId);
        expect(inputsById[premiumId].input).to.deep.equal({ id: premiumId, name: "Update 1" });
        expect(inputsById[teamId].node.id).to.deep.equal(teamId);
        expect(inputsById[teamId].input).to.deep.equal(teamId);
        expect(inputsById[userId1].node.id).to.deep.equal(userId1);
        expect(inputsById[userId1].input).to.deep.equal({ id: userId1, name: "Update 2" });
        expect(inputsById[userId2].node.id).to.deep.equal(userId2);
        expect(inputsById[userId2].input).to.deep.equal({ id: userId2, name: "Update 3" });
        expect(inputsByType["Premium"].Update).to.have.lengthOf(1);
        expect(inputsByType["Premium"].Update[0].node.id).to.deep.equal(premiumId);
        expect(inputsByType["Premium"].Update[0].input).to.deep.equal({ id: premiumId, name: "Update 1" });
        expect(inputsByType["Team"].Delete).to.have.lengthOf(1);
        expect(inputsByType["Team"].Delete[0].node.id).to.deep.equal(teamId);
        expect(inputsByType["Team"].Delete[0].input).to.deep.equal(teamId);
        expect(inputsByType["User"].Update).to.have.lengthOf(2);
        expect(inputsByType["User"].Update[0].node.id).to.deep.equal(userId1);
        expect(inputsByType["User"].Update[0].input).to.deep.equal({ id: userId1, name: "Update 2" });
        expect(inputsByType["User"].Update[1].node.id).to.deep.equal(userId2);
        expect(inputsByType["User"].Update[1].input).to.deep.equal({ id: userId2, name: "Update 3" });
    });
});

describe("initializeInputMaps", () => {
    let idsByAction: IdsByAction;
    let idsByType: IdsByType;
    let inputsByType: InputsByType;

    beforeEach(() => {
        idsByAction = {};
        idsByType = {};
        inputsByType = {};
    });

    // Test initializing idsByAction and idsByType when both are not initialized
    it("should initialize idsByAction and idsByType if they are not already initialized", () => {
        const actionType = "Create";
        const objectType = "RoutineVersion";

        initializeInputMaps(actionType, objectType, idsByAction, idsByType, inputsByType);

        expect(idsByAction[actionType]).to.have.lengthOf(0);
        expect(idsByType[objectType]).to.have.lengthOf(0);
        expect(inputsByType[objectType]).to.deep.equal({
            Connect: [],
            Create: [],
            Delete: [],
            Disconnect: [],
            Read: [],
            Update: [],
        });
    });

    // Test behavior when idsByAction is already initialized
    it("should not modify idsByAction if it is already initialized", () => {
        const actionType = "Delete";
        const objectType = "RoutineVersion";
        idsByAction[actionType] = ["existingId"];

        initializeInputMaps(actionType, objectType, idsByAction, idsByType, inputsByType);

        expect(idsByAction[actionType]).to.deep.equal(["existingId"]);
    });

    // Test behavior when idsByType is already initialized
    it("should not modify idsByType if it is already initialized", () => {
        const actionType = "Update";
        const objectType = "RoutineVersion";
        idsByType[objectType] = ["existingId"];

        initializeInputMaps(actionType, objectType, idsByAction, idsByType, inputsByType);

        expect(idsByType[objectType]).to.deep.equal(["existingId"]);
    });

    // Test behavior when inputsByType is already initialized
    it("should not modify inputsByType if it is already initialized", () => {
        const actionType = "Read";
        const objectType = "RoutineVersion";
        inputsByType[objectType] = {
            Connect: [],
            Create: [],
            Delete: [],
            Disconnect: [],
            Read: [{ node: {} as any, input: "existingInput" as any }],
            Update: [],
        };

        initializeInputMaps(actionType, objectType, idsByAction, idsByType, inputsByType);

        expect(inputsByType[objectType]).to.deep.equal({
            Connect: [],
            Create: [],
            Delete: [],
            Disconnect: [],
            Read: [{ node: {} as any, input: "existingInput" }],
            Update: [],
        });
    });
});

describe("updateClosestWithId", () => {
    const idField = "id";

    it("should return null for Create action type", () => {
        const result = updateClosestWithId("Create", { id: "123" }, idField, "User", { __typename: "User", id: "123", path: "" });
        expect(result).to.be.null;
    });

    it("should return null if input has no ID and no relation", () => {
        const result = updateClosestWithId("Update", { name: "Test" }, idField, "User", { __typename: "User", id: "123", path: "" });
        expect(result).to.be.null;
    });

    it("should return updated path if input has no ID but has a relation", () => {
        const result = updateClosestWithId("Update", { name: "Test" }, idField, "User", { __typename: "User", id: "123", path: "profile" }, "avatar");
        expect(result).to.deep.equal({ __typename: "User", id: "123", path: "profile.avatar" });
    });

    it("should return null if input has no ID and no closestWithId but has a relation", () => {
        const result = updateClosestWithId("Update", { name: "Test" }, idField, "User", null, "avatar");
        expect(result).to.be.null;
    });

    it("should return object with empty path if input has an ID", () => {
        const result = updateClosestWithId("Update", { id: "456", name: "Test" }, idField, "User", { __typename: "User", id: "123", path: "profile" });
        expect(result).to.deep.equal({ __typename: "User", id: "456", path: "" });
    });

    it("should handle string inputs as IDs", () => {
        const result = updateClosestWithId("Update", "789", idField, "User", { __typename: "User", id: "123", path: "profile" });
        expect(result).to.deep.equal({ __typename: "User", id: "789", path: "" });
    });

    it("should return null if input is a boolean (implying a Disconnect or Delete), and there is no relation", () => {
        const result = updateClosestWithId("Update", true, idField, "User", { __typename: "User", id: "123", path: "profile" });
        expect(result).to.be.null;
    });

    it("should return updated path if input is a boolean (implying a Disconnect or Delete), and there is a relation", () => {
        const result = updateClosestWithId("Update", true, idField, "User", { __typename: "User", id: "123", path: "profile" }, "avatar");
        expect(result).to.deep.equal({ __typename: "User", id: "123", path: "profile.avatar" });
    });
});

describe("determineModelType", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    const CodeFormat = {
        apiRelMap: {
            __typename: "Code",
            createdBy: "User",
            issues: "Issue",
            owner: {
                ownedByTeam: "Team",
                ownedByUser: "User",
            },
            parent: "CodeVersion",
            pullRequests: "PullRequest",
            bookmarkedBy: "User",
            tags: "Tag",
            transfers: "Transfer",
            versions: "CodeVersion",
        },
        unionFields: {
            owner: {},
        },
    };

    const CommentFormat = {
        apiRelMap: {
            __typename: "Comment",
            owner: {
                ownedByUser: "User",
                ownedByTeam: "Team",
            },
            commentedOn: {
                issue: "Issue",
                post: "Post",
            },
            reports: "Report",
        },
        unionFields: {
            commentedOn: { connectField: "forConnect", typeField: "createdFor" },
            owner: {},
        },
    };

    const LabelFormat = {
        apiRelMap: {
            __typename: "Label",
            owner: {
                ownedByUser: "User",
                ownedByTeam: "Team",
            },
        },
        unionFields: {
            owner: {},
        },
    };

    beforeAll(() => {
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    afterAll(() => {
        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    it("returns correct __typename for standard fields", () => {
        const __typename = determineModelType("reportsCreate", "reports", {}, CommentFormat);
        expect(__typename).to.equal("Report");
    });

    describe("handles unions", () => {
        it("simple unions - test 1", () => {
            const __typename = determineModelType("ownedByUserConnect", "ownedByUser", {}, LabelFormat);
            expect(__typename).to.equal("User");
        });

        it("simple unions - test 2", () => {
            const __typename = determineModelType("ownedByTeamConnect", "ownedByTeam", {}, CodeFormat);
            expect(__typename).to.equal("Team");
        });

        it("complex unions - test 1", () => {
            const input = { createdFor: "Post" };
            const __typename = determineModelType("forConnect", "for", input, CommentFormat);
            expect(__typename).to.equal("Post");
        });
    });

    it("returns null for special cases like translations", () => {
        const __typename = determineModelType("translationsCreate", "translations", {}, CommentFormat);
        expect(__typename).to.be.null;
    });

    it("throws an error when field is not found in apiRelMap", () => {
        expect(() => {
            determineModelType("nonexistentFieldCreate", "nonexistentField", {}, CommentFormat);
        }).to.throw("InternalError");
    });

    it("throws an error for missing union typeField in input", () => {
        expect(() => {
            determineModelType("forConnect", "for", {}, CommentFormat);
        }).to.throw("InternalError");
    });
});

// NOTE: These relations only exist in either a Create or Update mutation's data. 
// When `closestWithId` is null, it means that the mutation is a Create mutation.
// Some actions are not allowed in Create mutations, so they should throw an error.
describe("processCreateOrUpdate", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;
    let idsByAction, idsByType, inputsById, inputsByType, parentNode, closestWithId;

    beforeAll(() => {
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(() => {
        // Initialize the maps and parentNode
        idsByAction = { Create: [], Update: [] };
        idsByType = {};
        inputsById = {};
        inputsByType = {};
        parentNode = new InputNode("RoutineVersion", "parentId", "Update");
        closestWithId = { __typename: "RoutineVersion", id: "parentId", path: "" };
    });

    afterAll(() => {
        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    it("correctly processes a 'Create' action with closestWithId", () => {
        const childFormat = { apiRelMap: { __typename: "User" as const } };
        const childInput = { id: "childID", name: "Test", someRelation: { id: "789" } };
        const action = "Create";
        const fieldName = "child";
        const idField = "id";

        processCreateOrUpdate(action, childInput, childFormat, fieldName, idField, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);

        expect(idsByAction.Create).to.deep.equal(["childID"]);
        expect(idsByType.User).to.deep.equal(["childID"]);
        expect(inputsById["childID"].node.id).to.deep.equal("childID");
        expect(inputsById["childID"].input).to.deep.equal(childInput);
        expect(inputsByType.User.Create[0].node.id).to.deep.equal("childID");
        expect(inputsByType.User.Create[0].input).to.deep.equal(childInput);
        expect(parentNode.children[0].id).to.deep.equal("childID");
        expect(parentNode.children[0].parent).to.equal(parentNode);
        // Shouldn't change closestWithId, since it's not relevant outside of the recursive call
        expect(closestWithId).to.equal(closestWithId);
    });

    it("correctly processes a 'Create' action without closestWithId", () => {
        const childFormat = { apiRelMap: { __typename: "User" as const } };
        const childInput = { id: "childID", name: "Test", someRelation: { id: "789" } };
        const action = "Create";
        const fieldName = "child";
        const idField = "id";

        processCreateOrUpdate(action, childInput, childFormat, fieldName, idField, parentNode, null, idsByAction, idsByType, inputsById, inputsByType);

        expect(idsByAction.Create).to.deep.equal(["childID"]);
        expect(idsByType.User).to.deep.equal(["childID"]);
        expect(inputsById["childID"].node.id).to.deep.equal("childID");
        expect(inputsById["childID"].input).to.deep.equal(childInput);
        expect(inputsByType.User.Create[0].node.id).to.deep.equal("childID");
        expect(inputsByType.User.Create[0].input).to.deep.equal(childInput);
        expect(parentNode.children[0].id).to.deep.equal("childID");
        expect(parentNode.children[0].parent).to.equal(parentNode);
        // Shouldn't change closestWithId, since it's not relevant outside of the recursive call
        expect(closestWithId).to.equal(closestWithId);
    });


    it("correctly processes an 'Update' action with closestWithId", () => {
        const childFormat = { apiRelMap: { __typename: "User" as const } };
        const childInput = { id: "childID", name: "Test", someRelation: { id: "789" } };
        const action = "Update";
        const fieldName = "child";
        const idField = "id";

        processCreateOrUpdate(action, childInput, childFormat, fieldName, idField, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);

        expect(idsByAction.Update).to.deep.equal(["childID"]);
        expect(idsByType.User).to.deep.equal(["childID"]);
        expect(inputsById["childID"].node.id).to.deep.equal("childID");
        expect(inputsById["childID"].input).to.deep.equal(childInput);
        expect(inputsByType.User.Update[0].node.id).to.deep.equal("childID");
        expect(inputsByType.User.Update[0].input).to.deep.equal(childInput);
        expect(parentNode.children[0].id).to.deep.equal("childID");
        expect(parentNode.children[0].parent).to.equal(parentNode);
        // Shouldn't change closestWithId, since it's not relevant outside of the recursive call
        expect(closestWithId).to.equal(closestWithId);
    });

    it("throws an error when processing an 'Update' action without closestWithId", () => {
        const childFormat = { apiRelMap: { __typename: "User" as const } };
        const childInput = { id: "childID", name: "Test", someRelation: { id: "789" } };
        const action = "Update";
        const fieldName = "child";
        const idField = "id";

        try {
            processCreateOrUpdate(action, childInput, childFormat, fieldName, idField, parentNode, null, idsByAction, idsByType, inputsById, inputsByType);
            expect.fail("Expected an error to be thrown");
        } catch (error) { /* empty */ }
    });

    it("throws an error for invalid actions", () => {
        const childFormat = { apiRelMap: { __typename: "User" as const } };
        const childInput = { id: "childID", name: "Test", someRelation: { id: "789" } };
        const action = "Connect"; // This is process CREATE or UPDATE, not CONNECT
        const fieldName = "child";
        const idField = "id";

        try {
            processCreateOrUpdate(action, childInput, childFormat, fieldName, idField, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);
            expect.fail("Expected an error to be thrown");
        } catch (error) { /* empty */ }
    });

    it("correctly processes multiple different objects", () => {
        const childFormat = { apiRelMap: { __typename: "User" as const } };
        const firstChildInput = { id: "childID1", name: "TestUser1", someRelation: { id: "789" } };
        const secondChildInput = { id: "childID2", name: "TestUser2", someRelation: { id: "790" } };
        const action = "Create";
        const fieldName = "child";
        const idField = "id";

        // Call the function with two different input objects
        processCreateOrUpdate(action, firstChildInput, childFormat, fieldName, idField, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        processCreateOrUpdate(action, secondChildInput, childFormat, fieldName, idField, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);

        expect(idsByAction[action]).to.include.members([firstChildInput.id, secondChildInput.id]);
        expect(idsByType[childFormat.apiRelMap.__typename]).to.include.members([firstChildInput.id, secondChildInput.id]);
        expect(parentNode.children).to.have.lengthOf(2);
        // Ensure each child node has the expected properties
        expect(parentNode.children[0].id).to.deep.equal(firstChildInput.id);
        expect(parentNode.children[1].id).to.deep.equal(secondChildInput.id);
    });
});

describe("processConnectDisconnectOrDelete", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;
    let idsByAction, idsByType, inputsById, inputsByType, parentNode, closestWithId;

    beforeAll(() => {
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(() => {
        idsByAction = {};
        idsByType = {};
        inputsById = {};
        inputsByType = {};
        parentNode = new InputNode("RoutineVersion", "parentId", "Update");
        closestWithId = { __typename: "RoutineVersion", id: "parentId", path: "" };
    });

    afterAll(() => {
        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    it("initializes input maps correctly", () => {
        processConnectDisconnectOrDelete("123", false, "Delete", "fieldName", "User", parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        expect(idsByAction.Delete).to.deep.equal(["123"]);
        expect(idsByType.User).to.deep.equal(["123"]);
        expect(parentNode.children[0].id).to.deep.equal("123");
    });

    it("adds ID to maps for toMany Connect action with closestWithId", () => {
        processConnectDisconnectOrDelete("123", false, "Connect", "fieldName", "User", parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        expect(idsByAction.Connect).to.deep.equal(["123"]);
        expect(idsByType.User).to.deep.equal(["123"]);
        expect(parentNode.children[0].id).to.deep.equal("123");
    });

    it("adds ID to maps for toMany Connect action without closestWithId", () => {
        processConnectDisconnectOrDelete("123", false, "Connect", "fieldName", "User", parentNode, null, idsByAction, idsByType, inputsById, inputsByType);
        expect(idsByAction.Connect).to.deep.equal(["123"]);
        expect(idsByType.User).to.deep.equal(["123"]);
        expect(parentNode.children[0].id).to.deep.equal("123");
    });

    it("adds ID to maps for toMany Disconnect action with closestWithId", () => {
        processConnectDisconnectOrDelete("123", false, "Disconnect", "fieldName", "User", parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        expect(idsByAction.Disconnect).to.include("123");
        expect(idsByType.User).to.include("123");
        expect(parentNode.children[0].id).to.deep.equal("123");
    });

    it("throws error for toMany Disconnect action without closestWithId", () => {
        try {
            processConnectDisconnectOrDelete("123", false, "Disconnect", "fieldName", "User", parentNode, null, idsByAction, idsByType, inputsById, inputsByType);
            expect.fail("Expected an error to be thrown");
        } catch (error) { /* empty */ }
    });

    it("adds ID and input to maps for toMany Delete action with closestWithId", () => {
        processConnectDisconnectOrDelete("123", false, "Delete", "fieldName", "User", parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        expect(idsByAction.Delete).to.include("123");
        expect(idsByType.User).to.include("123");
        expect(inputsById["123"].node.__typename).to.equal("User");
        expect(inputsById["123"].input).to.equal("123");
        expect(inputsByType.User.Delete[0].node.__typename).to.equal("User");
        expect(inputsByType.User.Delete[0].input).to.equal("123");
        expect(parentNode.children[0].id).to.deep.equal("123");
    });

    it("throws error for toMany Delete action without closestWithId", () => {
        try {
            processConnectDisconnectOrDelete("123", false, "Delete", "fieldName", "User", parentNode, null, idsByAction, idsByType, inputsById, inputsByType);
            expect.fail("Expected an error to be thrown");
        } catch (error) { /* empty */ }
    });

    it("adds ID and placeholder for toOne Connect relationships with closestWithId", () => {
        const expectedPlaceholder = "RoutineVersion|parentId.User|fieldName";

        processConnectDisconnectOrDelete("123", true, "Connect", "fieldName", "User", parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        expect(idsByAction.Connect).to.include("123");
        expect(idsByAction.Disconnect).to.include(expectedPlaceholder);
        expect(idsByType.User).to.include(expectedPlaceholder);
        expect(idsByType.User).to.include("123");
        expect(parentNode.children[0].id).to.deep.equal(expectedPlaceholder);
        expect(parentNode.children[0].action).to.equal("Disconnect");
        expect(parentNode.children[1].id).to.deep.equal("123");
        expect(parentNode.children[1].action).to.equal("Connect");
        expect(parentNode.children).to.have.lengthOf(2);
    });

    // NOTE: This only includes a placeholder because toOne Disconnects use a boolean 
    // to disconnect the relationship, not an ID. We typically pass in a blank string 
    // for the ID in reality
    it("adds placeholder for toOne Disconnect relationships with closestWithId", () => {
        const expectedPlaceholder = "RoutineVersion|parentId.User|fieldName";

        processConnectDisconnectOrDelete("123", true, "Disconnect", "fieldName", "User", parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        expect(idsByAction.Disconnect).to.include(expectedPlaceholder);
        expect(idsByAction.Disconnect).not.to.include("123");
        expect(idsByType.User).to.include(expectedPlaceholder);
        expect(idsByType.User).not.to.include("123");
        expect(parentNode.children[0].id).to.deep.equal(expectedPlaceholder);
        expect(parentNode.children[0].action).to.equal("Disconnect");
        expect(parentNode.children).to.have.lengthOf(1);
    });

    it("adds only ID for toOne Connect relationships without closestWithId", () => {
        processConnectDisconnectOrDelete("123", true, "Connect", "fieldName", "User", parentNode, null, idsByAction, idsByType, inputsById, inputsByType);
        expect(idsByAction.Connect).to.deep.equal(["123"]);
        expect(idsByType.User).to.deep.equal(["123"]);
        expect(parentNode.children[0].id).to.deep.equal("123");
    });

    it("throws error for toOne Disconnect relationships without closestWithId", () => {
        try {
            processConnectDisconnectOrDelete("123", true, "Disconnect", "fieldName", "User", parentNode, null, idsByAction, idsByType, inputsById, inputsByType);
            expect.fail("Expected an error to be thrown");
        } catch (error) { /* empty */ }
    });

    it("adds ID and input toOne Delete action with closestWithId", () => {
        const expectedPlaceholder = "RoutineVersion|parentId.User|fieldName";

        processConnectDisconnectOrDelete("123", true, "Delete", "fieldName", "User", parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        expect(idsByAction.Delete).to.include(expectedPlaceholder);
        expect(idsByAction.Delete).not.to.include("123");
        expect(idsByType.User).to.include(expectedPlaceholder);
        expect(idsByType.User).not.to.include("123");
        expect(inputsById[expectedPlaceholder].node.__typename).to.equal("User");
        expect(inputsById[expectedPlaceholder].input).to.equal(expectedPlaceholder);
        expect(inputsById["123"]).to.be.undefined;
        expect(inputsByType.User.Delete[0].node.__typename).to.equal("User");
        expect(inputsByType.User.Delete[0].input).to.equal(expectedPlaceholder);
        expect(parentNode.children[0].id).to.deep.equal(expectedPlaceholder);
        expect(parentNode.children[0].action).to.equal("Delete");
        expect(parentNode.children).to.have.lengthOf(1);
    });

    it("throws error for toOne Delete action without closestWithId", () => {
        try {
            processConnectDisconnectOrDelete("123", true, "Delete", "fieldName", "User", parentNode, null, idsByAction, idsByType, inputsById, inputsByType);
            expect.fail("Expected an error to be thrown");
        } catch (error) { /* empty */ }
    });

    it("skips placeholder when fieldName is null", () => {
        processConnectDisconnectOrDelete("123", true, "Connect", null, "User", parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        expect(idsByAction.Connect).to.deep.equal(["123"]);
        expect(idsByType.User).to.deep.equal(["123"]);
        expect(parentNode.children[0].id).to.deep.equal("123");
    });
});

describe("processInputObjectField", () => {
    let parentNode, idsByAction, idsByType, inputsById, inputsByType, inputInfo, format, closestWithId;
    const initialIdsByAction = { Create: [], Update: [], Connect: [], Disconnect: [] };
    const initialIdsByType = {};
    const initialInputsById = {};
    const initialInputsByType = {};

    beforeEach(() => {
        parentNode = new InputNode("RoutineVersion", "parentId", "Update");
        idsByAction = JSON.parse(JSON.stringify(initialIdsByAction));
        idsByType = JSON.parse(JSON.stringify(initialIdsByType));
        inputsById = JSON.parse(JSON.stringify(initialInputsById));
        inputsByType = JSON.parse(JSON.stringify(initialInputsByType));
        inputInfo = { node: parentNode, input: {} };
        format = {
            apiRelMap: {
                __typename: "User" as const,
                project: "Project",
                reports: "Report",
            },
        };
        closestWithId = { __typename: "Routine", id: "grandparentId", path: "RoutineVersion|version" };
    });

    // NOTE: We'll assume that fields with action suffixes are always 
    // relation fields, so we don't need to test the case where the field
    // doesn't contain valid relation data. It would only slow performance.
    it("handles non-relation fields (name not containing action suffix)", () => {
        const field = "description";
        const input = { description: "New Description" };

        processInputObjectField(field, input, format, parentNode, null, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        // Should only be in the inputInfo
        expect(idsByAction).to.deep.equal(initialIdsByAction);
        expect(idsByType).to.deep.equal(initialIdsByType);
        expect(inputsById).to.deep.equal(initialInputsById);
        expect(inputsByType).to.deep.equal(initialInputsByType);
        expect(inputInfo.input[field]).to.equal(input[field]);
    });

    it("handles non-array create relations with closestWithId", () => {
        const field = "projectCreate";
        const input = {
            projectCreate: { id: "123", name: "Test", someRelation: { id: "789" } },
            projectUpdate: { id: "456", name: "Test", someRelation: { id: "790" } }, // Should be ignored
        };

        processInputObjectField(field, input, format, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        expect(idsByAction.Create).to.deep.equal([input[field].id]);
        expect(idsByType.Project).to.deep.equal([input[field].id]);
        expect(inputsById[input[field].id].node.id).to.deep.equal(input[field].id);
        expect(inputsById[input[field].id].input).to.deep.equal(input[field]);
        expect(inputsByType.Project.Create[0].node.id).to.deep.equal(input[field].id);
        expect(inputsByType.Project.Create[0].input).to.deep.equal(input[field]);
        expect(parentNode.children[0].id).to.deep.equal(input[field].id);
        expect(parentNode.children[0].parent).to.equal(parentNode);
        expect(inputInfo.input[field]).to.equal(input[field].id);
    });

    it("handles non-array create relations without closestWithId", () => {
        const field = "projectCreate";
        const input = {
            projectCreate: { id: "123", name: "Test", someRelation: { id: "789" } },
            projectUpdate: { id: "456", name: "Test", someRelation: { id: "790" } }, // Should be ignored
        };

        processInputObjectField(field, input, format, parentNode, null, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        expect(idsByAction.Create).to.deep.equal([input[field].id]);
        expect(idsByType.Project).to.deep.equal([input[field].id]);
        expect(inputsById[input[field].id].node.id).to.deep.equal(input[field].id);
        expect(inputsById[input[field].id].input).to.deep.equal(input[field]);
        expect(inputsByType.Project.Create[0].node.id).to.deep.equal(input[field].id);
        expect(inputsByType.Project.Create[0].input).to.deep.equal(input[field]);
        expect(parentNode.children[0].id).to.deep.equal(input[field].id);
        expect(parentNode.children[0].parent).to.equal(parentNode);
        expect(inputInfo.input[field]).to.equal(input[field].id);
    });

    it("handles array create relations with closestWithId", () => {
        const field = "reportsCreate";
        const input = {
            reportsCreate: [
                { id: "123", name: "Test", someRelation: { id: "789" } },
                { id: "456", name: "Test", someRelation: { id: "790" } },
            ],
        };

        processInputObjectField(field, input, format, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        expect(idsByAction.Create).to.deep.equal(["123", "456"]);
        expect(idsByType.Report).to.deep.equal(["123", "456"]);
        expect(inputsById["123"].node.id).to.deep.equal("123");
        expect(inputsById["123"].input).to.deep.equal(input[field][0]);
        expect(inputsByType.Report.Create[0].node.id).to.deep.equal("123");
        expect(inputsByType.Report.Create[0].input).to.deep.equal(input[field][0]);
        expect(inputsById["456"].node.id).to.deep.equal("456");
        expect(inputsById["456"].input).to.deep.equal(input[field][1]);
        expect(inputsByType.Report.Create[1].node.id).to.deep.equal("456");
        expect(inputsByType.Report.Create[1].input).to.deep.equal(input[field][1]);
        expect(parentNode.children[0].id).to.deep.equal("123");
        expect(parentNode.children[0].parent).to.equal(parentNode);
        expect(parentNode.children[1].id).to.deep.equal("456");
        expect(parentNode.children[1].parent).to.equal(parentNode);
        expect(inputInfo.input[field]).to.deep.equal(["123", "456"]);
    });

    it("handles array create relations without closestWithId", () => {
        const field = "reportsCreate";
        const input = {
            reportsCreate: [
                { id: "123", name: "Test", someRelation: { id: "789" } },
                { id: "456", name: "Test", someRelation: { id: "790" } },
            ],
        };

        processInputObjectField(field, input, format, parentNode, null, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        expect(idsByAction.Create).to.deep.equal(["123", "456"]);
        expect(idsByType.Report).to.deep.equal(["123", "456"]);
        expect(inputsById["123"].node.id).to.deep.equal("123");
        expect(inputsById["123"].input).to.deep.equal(input[field][0]);
        expect(inputsByType.Report.Create[0].node.id).to.deep.equal("123");
        expect(inputsByType.Report.Create[0].input).to.deep.equal(input[field][0]);
        expect(inputsById["456"].node.id).to.deep.equal("456");
        expect(inputsById["456"].input).to.deep.equal(input[field][1]);
        expect(inputsByType.Report.Create[1].node.id).to.deep.equal("456");
        expect(inputsByType.Report.Create[1].input).to.deep.equal(input[field][1]);
        expect(parentNode.children[0].id).to.deep.equal("123");
        expect(parentNode.children[0].parent).to.equal(parentNode);
        expect(parentNode.children[1].id).to.deep.equal("456");
        expect(parentNode.children[1].parent).to.equal(parentNode);
        expect(inputInfo.input[field]).to.deep.equal(["123", "456"]);
    });

    it("handles non-array update relations with closestWithId", () => {
        const field = "projectUpdate";
        const input = {
            projectCreate: { id: "123", name: "Test", someRelation: { id: "789" } }, // Should be ignored
            projectUpdate: { id: "456", name: "Test", someRelation: { id: "790" } },
        };

        processInputObjectField(field, input, format, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        expect(idsByAction.Update).to.deep.equal(["456"]);
        expect(idsByType.Project).to.deep.equal(["456"]);
        expect(inputsById["456"].node.id).to.deep.equal("456");
        expect(inputsById["456"].input).to.deep.equal(input[field]);
        expect(inputsByType.Project.Update[0].node.id).to.deep.equal("456");
        expect(inputsByType.Project.Update[0].input).to.deep.equal(input[field]);
        expect(parentNode.children[0].id).to.deep.equal("456");
        expect(parentNode.children[0].parent).to.equal(parentNode);
        expect(inputInfo.input[field]).to.equal("456");
    });

    it("handles non-array update relations without closestWithId", () => {
        const field = "projectUpdate";
        const input = {
            projectCreate: { id: "123", name: "Test", someRelation: { id: "789" } }, // Should be ignored
            projectUpdate: { id: "456", name: "Test", someRelation: { id: "790" } },
        };

        // Should throw an error because closestWithId is null, 
        // and you can't update inside a Create mutation
        try {
            processInputObjectField(field, input, format, parentNode, null, idsByAction, idsByType, inputsById, inputsByType, inputInfo);
            expect.fail("Expected an error to be thrown");
        } catch (error) { /* empty */ }
    });

    it("handles array update relations with closestWithId", () => {
        const field = "reportsUpdate";
        const input = {
            reportsCreate: [
                { id: "123", name: "Test", someRelation: { id: "789" } }, // Should be ignored
                { id: "456", name: "Test", someRelation: { id: "790" } }, // Should be ignored
            ],
            reportsUpdate: [
                { id: "789", name: "Test", someRelation: { id: "791" } },
                { id: "790", name: "Test", someRelation: { id: "792" } },
            ],
        };

        processInputObjectField(field, input, format, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        expect(idsByAction.Update).to.deep.equal(["789", "790"]);
        expect(idsByType.Report).to.deep.equal(["789", "790"]);
        expect(inputsById["789"].node.id).to.deep.equal("789");
        expect(inputsById["789"].input).to.deep.equal(input[field][0]);
        expect(inputsByType.Report.Update[0].node.id).to.deep.equal("789");
        expect(inputsByType.Report.Update[0].input).to.deep.equal(input[field][0]);
        expect(inputsById["790"].node.id).to.deep.equal("790");
        expect(inputsById["790"].input).to.deep.equal(input[field][1]);
        expect(inputsByType.Report.Update[1].node.id).to.deep.equal("790");
        expect(inputsByType.Report.Update[1].input).to.deep.equal(input[field][1]);
        expect(parentNode.children[0].id).to.deep.equal("789");
        expect(parentNode.children[0].parent).to.equal(parentNode);
        expect(parentNode.children[1].id).to.deep.equal("790");
        expect(parentNode.children[1].parent).to.equal(parentNode);
        expect(inputInfo.input[field]).to.deep.equal(["789", "790"]);
    });

    it("handles array update relations without closestWithId", () => {
        const field = "reportsUpdate";
        const input = {
            reportsCreate: [
                { id: "123", name: "Test", someRelation: { id: "789" } }, // Should be ignored
                { id: "456", name: "Test", someRelation: { id: "790" } }, // Should be ignored
            ],
            reportsUpdate: [
                { id: "789", name: "Test", someRelation: { id: "791" } },
                { id: "790", name: "Test", someRelation: { id: "792" } },
            ],
        };

        // Should throw an error because closestWithId is null,
        // and you can't update inside a Create mutation
        // expect(() => {
        //     processInputObjectField(field, input, format, parentNode, null, idsByAction, idsByType, inputsById, inputsByType, inputInfo);
        // }).toThrow("InternalError");
        try {
            processInputObjectField(field, input, format, parentNode, null, idsByAction, idsByType, inputsById, inputsByType, inputInfo);
            expect.fail("Expected an error to be thrown");
        } catch (error) { /* empty */ }
    });

    it("handles non-array connect relations with closestWithId", () => {
        const field = "projectConnect";
        const input = {
            projectConnect: "123",
        };

        processInputObjectField(field, input, format, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        // Should contain both placeholder and ID, since we may be 
        // implicitly Disconnecting the previous relation
        const expectedPlaceholder = "Routine|grandparentId.RoutineVersion|version.Project|project";
        expect(idsByAction.Connect).to.deep.equal(["123"]);
        expect(idsByAction.Disconnect).to.deep.equal([expectedPlaceholder]);
        expect(idsByType.Project).to.deep.equal([expectedPlaceholder, "123"]);
        // No input data, since we're only giving an ID
        expect(inputsById[expectedPlaceholder]).to.be.undefined;
        expect(inputsById["123"]).to.be.undefined;
        expect(inputsByType.Project.Connect).to.have.lengthOf(0);
        expect(parentNode.children[0].id).to.deep.equal(expectedPlaceholder);
        expect(parentNode.children[0].action).to.deep.equal("Disconnect");
        expect(parentNode.children[0].parent).to.equal(parentNode);
        expect(parentNode.children[1].id).to.deep.equal("123");
        expect(parentNode.children[1].action).to.deep.equal("Connect");
        expect(parentNode.children[1].parent).to.equal(parentNode);
        expect(parentNode.children).to.have.lengthOf(2);
        expect(inputInfo.input[field]).to.equal("123");
    });

    it("handles non-array connect relations without closestWithId", () => {
        const field = "projectConnect";
        const input = {
            projectConnect: "123",
        };

        processInputObjectField(field, input, format, parentNode, null, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        // Should contain only ID, since we can't implicitly 
        // Disconnect within a Create mutation
        expect(idsByAction.Connect).to.deep.equal(["123"]);
        expect(idsByType.Project).to.deep.equal(["123"]);
        // No input data, since we're only giving an ID
        expect(inputsById["123"]).to.be.undefined;
        expect(inputsByType.Project.Connect).to.have.lengthOf(0);
        expect(parentNode.children[0].id).to.deep.equal("123");
        expect(parentNode.children[0].action).to.deep.equal("Connect");
        expect(parentNode.children[0].parent).to.equal(parentNode);
        expect(parentNode.children).to.have.lengthOf(1);
        expect(inputInfo.input[field]).to.equal("123");
    });

    it("handles array connect relations with closestWithId", () => {
        const field = "reportsConnect";
        const input = {
            reportsConnect: ["123", "456"],
        };

        processInputObjectField(field, input, format, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        // Should only contain IDs, since we're not implicitly Disconnecting anything
        expect(idsByAction.Connect).to.deep.equal(["123", "456"]);
        expect(idsByType.Report).to.deep.equal(["123", "456"]);
        // No input data, since we're only giving IDs
        expect(inputsById["123"]).to.be.undefined;
        expect(inputsById["456"]).to.be.undefined;
        expect(inputsByType.Report.Connect).to.have.lengthOf(0);
        expect(parentNode.children[0].id).to.deep.equal("123");
        expect(parentNode.children[0].action).to.deep.equal("Connect");
        expect(parentNode.children[0].parent).to.equal(parentNode);
        expect(parentNode.children[1].id).to.deep.equal("456");
        expect(parentNode.children[1].action).to.deep.equal("Connect");
        expect(parentNode.children[1].parent).to.equal(parentNode);
        expect(parentNode.children).to.have.lengthOf(2);
        expect(inputInfo.input[field]).to.deep.equal(["123", "456"]);
    });

    it("handles array connect relations without closestWithId", () => {
        const field = "reportsConnect";
        const input = {
            reportsConnect: ["123", "456"],
        };

        processInputObjectField(field, input, format, parentNode, null, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        // Should only contain IDs, since we can't implicitly
        // Disconnect within a Create mutation
        expect(idsByAction.Connect).to.deep.equal(["123", "456"]);
        expect(idsByType.Report).to.deep.equal(["123", "456"]);
        // No input data, since we're only giving IDs
        expect(inputsById["123"]).to.be.undefined;
        expect(inputsById["456"]).to.be.undefined;
        expect(inputsByType.Report.Connect).to.have.lengthOf(0);
        expect(parentNode.children[0].id).to.deep.equal("123");
        expect(parentNode.children[0].action).to.deep.equal("Connect");
        expect(parentNode.children[0].parent).to.equal(parentNode);
        expect(parentNode.children[1].id).to.deep.equal("456");
        expect(parentNode.children[1].action).to.deep.equal("Connect");
        expect(parentNode.children[1].parent).to.equal(parentNode);
        expect(parentNode.children).to.have.lengthOf(2);
        expect(inputInfo.input[field]).to.deep.equal(["123", "456"]);
    });

    it("handles non-array disconnect relations with closestWithId", () => {
        const field = "projectDisconnect";
        const input = {
            projectDisconnect: true,
        };

        processInputObjectField(field, input, format, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        // Should contain only placeholder, since we're implicitly Disconnecting
        // the previous relation, and didn't give an ID
        const expectedPlaceholder = "Routine|grandparentId.RoutineVersion|version.Project|project";
        expect(idsByAction.Disconnect).to.deep.equal([expectedPlaceholder]);
        expect(idsByType.Project).to.deep.equal([expectedPlaceholder]);
        // No input data, since we're only giving a boolean
        expect(inputsById[expectedPlaceholder]).to.be.undefined;
        expect(inputsByType.Project.Disconnect).to.have.lengthOf(0);
        expect(parentNode.children[0].id).to.deep.equal(expectedPlaceholder);
        expect(parentNode.children[0].action).to.deep.equal("Disconnect");
        expect(parentNode.children[0].parent).to.equal(parentNode);
        expect(parentNode.children).to.have.lengthOf(1);
        expect(inputInfo.input[field]).to.equal(true);
    });

    it("handles non-array disconnect relations without closestWithId", () => {
        const field = "projectDisconnect";
        const input = {
            projectDisconnect: true,
        };

        // Should throw an error because closestWithId is null,
        // and you can't Disconnect within a Create mutation
        try {
            processInputObjectField(field, input, format, parentNode, null, idsByAction, idsByType, inputsById, inputsByType, inputInfo);
            expect.fail("Expected an error to be thrown");
        } catch (error) { /* empty */ }
    });

    it("handles array disconnect relations with closestWithId", () => {
        const field = "reportsDisconnect";
        const input = {
            reportsDisconnect: ["123", "456"],
        };

        processInputObjectField(field, input, format, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        // Should only contain IDs, since we're not implicitly Disconnecting anything
        expect(idsByAction.Disconnect).to.deep.equal(["123", "456"]);
        expect(idsByType.Report).to.deep.equal(["123", "456"]);
        // No input data, since we're only giving IDs
        expect(inputsById["123"]).to.be.undefined;
        expect(inputsById["456"]).to.be.undefined;
        expect(inputsByType.Report.Disconnect).to.have.lengthOf(0);
        expect(parentNode.children[0].id).to.deep.equal("123");
        expect(parentNode.children[0].action).to.deep.equal("Disconnect");
        expect(parentNode.children[0].parent).to.equal(parentNode);
        expect(parentNode.children[1].id).to.deep.equal("456");
        expect(parentNode.children[1].action).to.deep.equal("Disconnect");
        expect(parentNode.children[1].parent).to.equal(parentNode);
        expect(parentNode.children).to.have.lengthOf(2);
        expect(inputInfo.input[field]).to.deep.equal(["123", "456"]);
    });

    it("handles array disconnect relations without closestWithId", () => {
        const field = "reportsDisconnect";
        const input = {
            reportsDisconnect: ["123", "456"],
        };

        // Should throw an error because closestWithId is null,
        // and you can't Disconnect within a Create mutation
        try {
            processInputObjectField(field, input, format, parentNode, null, idsByAction, idsByType, inputsById, inputsByType, inputInfo);
            expect.fail("Expected an error to be thrown");
        } catch (error) { /* empty */ }
    });

    it("handles non-array delete relations with closestWithId", () => {
        const field = "projectDelete";
        const input = {
            projectDelete: true,
        };

        processInputObjectField(field, input, format, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        // Should contain only placeholder, since we're Deleting without 
        // giving an ID
        const expectedPlaceholder = "Routine|grandparentId.RoutineVersion|version.Project|project";
        expect(idsByAction.Delete).to.deep.equal([expectedPlaceholder]);
        expect(idsByType.Project).to.deep.equal([expectedPlaceholder]);
        // Deletes actually do have input data, but in this case it'll 
        // be attached to the placeholder
        expect(inputsById[expectedPlaceholder].node.id).to.deep.equal(expectedPlaceholder);
        expect(inputsById[expectedPlaceholder].input).to.deep.equal(expectedPlaceholder);
        expect(inputsByType.Project.Delete[0].input).to.equal(expectedPlaceholder);
        expect(inputsByType.Project.Delete).to.have.lengthOf(1);
        expect(parentNode.children[0].id).to.deep.equal(expectedPlaceholder);
        expect(parentNode.children[0].action).to.deep.equal("Delete");
        expect(parentNode.children[0].parent).to.equal(parentNode);
        expect(parentNode.children).to.have.lengthOf(1);
        expect(inputInfo.input[field]).to.equal(true);
    });

    it("handles non-array delete relations without closestWithId", () => {
        const field = "projectDelete";
        const input = {
            projectDelete: true,
        };

        // Should throw an error because closestWithId is null,
        // and you can't Delete within a Create mutation
        try {
            processInputObjectField(field, input, format, parentNode, null, idsByAction, idsByType, inputsById, inputsByType, inputInfo);
            expect.fail("Expected an error to be thrown");
        } catch (error) { /* empty */ }
    });

    it("handles array delete relations with closestWithId", () => {
        const field = "reportsDelete";
        const input = {
            reportsDelete: ["123", "456"],
        };

        processInputObjectField(field, input, format, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        // Should only contain IDs, since we're not implicitly Deleting anything
        expect(idsByAction.Delete).to.deep.equal(["123", "456"]);
        expect(idsByType.Report).to.deep.equal(["123", "456"]);
        expect(inputsById["123"].node.id).to.deep.equal("123");
        expect(inputsById["123"].input).to.deep.equal(input[field][0]);
        expect(inputsById["456"].node.id).to.deep.equal("456");
        expect(inputsById["456"].input).to.deep.equal(input[field][1]);
        expect(inputsByType.Report.Delete[0].input).to.equal(input[field][0]);
        expect(inputsByType.Report.Delete[1].input).to.equal(input[field][1]);
        expect(inputsByType.Report.Delete).to.have.lengthOf(2);
        expect(parentNode.children[0].id).to.deep.equal("123");
        expect(parentNode.children[0].action).to.deep.equal("Delete");
        expect(parentNode.children[0].parent).to.equal(parentNode);
        expect(parentNode.children[1].id).to.deep.equal("456");
        expect(parentNode.children[1].action).to.deep.equal("Delete");
        expect(parentNode.children[1].parent).to.equal(parentNode);
        expect(parentNode.children).to.have.lengthOf(2);
        expect(inputInfo.input[field]).to.deep.equal(["123", "456"]);
    });

    it("handles array delete relations without closestWithId", () => {
        const field = "reportsDelete";
        const input = {
            reportsDelete: ["123", "456"],
        };

        // Should throw an error because closestWithId is null,
        // and you can't Delete within a Create mutation
        try {
            processInputObjectField(field, input, format, parentNode, null, idsByAction, idsByType, inputsById, inputsByType, inputInfo);
            expect.fail("Expected an error to be thrown");
        } catch (error) { /* empty */ }
    });
});

/**
 * Helper function to check that an object has only the expected non-empty arrays, 
 * and that every other value is an empty array.
 */
function expectOnlyTheseArrays<ArrayItem>(
    actual: Record<string, unknown>,
    expectedNonEmpty: Record<string, ArrayItem[]>,
) {
    // Check non-empty properties
    Object.entries(expectedNonEmpty).forEach(([key, value]) => {
        expect(actual[key], `Missing array value. Key: ${key}`).to.deep.include.members(value);
        expect(actual[key], `Has extra array values. Key: ${key}`).to.have.lengthOf(value.length);
    });

    // Make sure that everythign else is an empty array
    Object.entries(actual).forEach(([key, value]) => {
        if (!expectedNonEmpty[key]) {
            expect(value, `Should be empty array. Key: ${key}`).to.deep.equal([]);
        }
    });
}

describe("inputToMaps", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;
    let idsByAction, idsByType, inputsById, inputsByType, format, closestWithId;
    const initialIdsByAction = { Create: [], Update: [], Connect: [], Disconnect: [] };
    const initialIdsByType = {};
    const initialInputsById = {};
    const initialInputsByType = {};

    beforeAll(() => {
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async () => {
        idsByAction = JSON.parse(JSON.stringify(initialIdsByAction));
        idsByType = JSON.parse(JSON.stringify(initialIdsByType));
        inputsById = JSON.parse(JSON.stringify(initialInputsById));
        inputsByType = JSON.parse(JSON.stringify(initialInputsByType));
        format = {
            apiRelMap: {
                __typename: "User" as const,
                resource: "Resource",
            },
        };
        closestWithId = { __typename: "Resource", id: "grandparentId", path: "version" };
    });

    afterAll(() => {
        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    it("should initialize root node correctly for a Create action with a simple input", () => {
        const action = "Create";
        const input = { id: "1", name: "John Doe" };
        const rootNode = inputToMaps(action, input, format, "id", closestWithId, idsByAction, idsByType, inputsById, inputsByType);

        expect(rootNode.id).to.equal("1");
        expect(rootNode.__typename).to.equal("User");
        expect(rootNode.action).to.equal("Create");
        expect(idsByAction.Create).to.include("1");
        expect(idsByType.User).to.include("1");
        expect(inputsById["1"].input).to.deep.equal(input);
    });

    it("should handle nested objects for a Create action", () => {
        const action = "Create";
        const input = {
            id: "1",
            projectCreate: { id: "2", name: "Project X" },
            reportsConnect: ["3", "4"],
        };
        inputToMaps(action, input, format, "id", closestWithId, idsByAction, idsByType, inputsById, inputsByType);

        expectOnlyTheseArrays(idsByAction, {
            Connect: ["3", "4"],
            Create: ["1", "2"],
        });
        expectOnlyTheseArrays(idsByType, {
            User: ["1"],
            Project: ["2"],
            Report: ["3", "4"],
        });
        expect(inputsById["1"].input).to.deep.equal({
            ...input,
            projectCreate: "2",
        });
        expect(inputsById["2"].input).to.deep.equal(input.projectCreate);
    });

    it("should throw error for nested Updates in a Create action, as they are invalid", () => {
        const action = "Create";
        const input = {
            id: "1",
            projectUpdate: { id: "2", name: "Project X" },
        };
        try {
            inputToMaps(action, input, format, "id", closestWithId, idsByAction, idsByType, inputsById, inputsByType);
            expect.fail("Expected an error to be thrown");
        } catch (error) { /* empty */ }
    });

    it("should throw error for nested Disconnects in a Create action, as they are invalid - test 1", () => {
        const action = "Create";
        const input = {
            id: "1",
            reportsDisconnect: ["3", "4"],
        };
        try {
            inputToMaps(action, input, format, "id", closestWithId, idsByAction, idsByType, inputsById, inputsByType);
            expect.fail("Expected an error to be thrown");
        } catch (error) { /* empty */ }
    });

    it("should throw error for nested Disconnects in a Create action, as they are invalid - test 2", () => {
        const action = "Create";
        const input = {
            id: "1",
            projectDisconnect: true,
        };
        try {
            inputToMaps(action, input, format, "id", closestWithId, idsByAction, idsByType, inputsById, inputsByType);
            expect.fail("Expected an error to be thrown");
        } catch (error) { /* empty */ }
    });

    it("should throw error for nested Deleted in a Create action, as they are invalid - test 1", () => {
        const action = "Create";
        const input = {
            id: "1",
            reportsDelete: ["3", "4"],
        };
        try {
            inputToMaps(action, input, format, "id", closestWithId, idsByAction, idsByType, inputsById, inputsByType);
            expect.fail("Expected an error to be thrown");
        } catch (error) { /* empty */ }
    });

    it("should throw error for nested Deletes in a Create action, as they are invalid - test 2", () => {
        const action = "Create";
        const input = {
            id: "1",
            projectDelete: true,
        };
        try {
            inputToMaps(action, input, format, "id", closestWithId, idsByAction, idsByType, inputsById, inputsByType);
            expect.fail("Expected an error to be thrown");
        } catch (error) { /* empty */ }
    });

    it("should handle nested objects for an Update action", () => {
        const action = "Update";
        const input = {
            id: "1",
            apiDelete: true,
            codeDisconnect: true,
            projectCreate: { id: "2", name: "Project X" },
            reportsConnect: ["3", "4"],
            rolesCreate: [{ id: "5", name: "Role 1" }, { id: "6", name: "Role 2" }],
            routineUpdate: { id: "7", name: "Routine X" },
            standardConnect: "8",
        };
        inputToMaps(action, input, format, "id", closestWithId, idsByAction, idsByType, inputsById, inputsByType);

        expectOnlyTheseArrays(idsByAction, {
            Connect: ["3", "4", "8"],
            Create: ["2", "5", "6"],
            Update: ["1", "7"],
            Delete: ["User|1.Api|api"],
            Disconnect: ["User|1.Code|code", "User|1.Standard|standard"], // Existing standard relation is implicitly disconnected
        });
        expectOnlyTheseArrays(idsByType, {
            User: ["1"],
            Api: ["User|1.Api|api"],
            Code: ["User|1.Code|code"],
            Project: ["2"],
            Report: ["3", "4"],
            Role: ["5", "6"],
            Routine: ["7"],
            Standard: ["8", "User|1.Standard|standard"], // Include implicit disconnect
        });
        expect(inputsById["1"].input).to.deep.equal({
            ...input,
            apiDelete: true,
            codeDisconnect: true,
            projectCreate: "2",
            rolesCreate: ["5", "6"],
            routineUpdate: "7",
            standardConnect: "8",
        });
        expect(inputsById["2"].input).to.deep.equal(input.projectCreate);
        expect(inputsById["5"].input).to.deep.equal(input.rolesCreate[0]);
        expect(inputsById["6"].input).to.deep.equal(input.rolesCreate[1]);
        expect(inputsById["7"].input).to.deep.equal(input.routineUpdate);
    });

    it("should skip nested objects for a Connect action, since that's invalid", () => {
        const action = "Connect";
        const input = { id: "1", project: { id: "2", name: "Project X" } };
        inputToMaps(action, input, format, "id", closestWithId, idsByAction, idsByType, inputsById, inputsByType);

        expectOnlyTheseArrays(idsByAction, {
            Connect: ["1"],
        });
        expectOnlyTheseArrays(idsByType, {
            User: ["1"],
        });
        expect(inputsById["1"].input).to.deep.equal(input);
    });

    it("should skip nested objects for a Disconnect action, since that's invalid", () => {
        const action = "Disconnect";
        const input = { id: "1", project: { id: "2", name: "Project X" } };
        inputToMaps(action, input, format, "id", closestWithId, idsByAction, idsByType, inputsById, inputsByType);

        expectOnlyTheseArrays(idsByAction, {
            Disconnect: ["1"],
        });
        expectOnlyTheseArrays(idsByType, {
            User: ["1"],
        });
        expect(inputsById["1"].input).to.deep.equal(input);
    });

    it("should skip nested objects for a Delete action, since that's invalid", () => {
        const action = "Delete";
        const input = { id: "1", project: { id: "2", name: "Project X" } };
        inputToMaps(action, input, format, "id", closestWithId, idsByAction, idsByType, inputsById, inputsByType);

        expectOnlyTheseArrays(idsByAction, {
            Delete: ["1"],
        });
        expectOnlyTheseArrays(idsByType, {
            User: ["1"],
        });
        expect(inputsById["1"].input).to.deep.equal(input);
    });

    it("should handle complex nested structures", () => {
        const action = "Update";
        const input = {
            id: "1",
            name: "John Doe",
            projectUpdate: {
                id: "2",
                name: "Updated Project",
                // NOTE: We're not relying on the mock format for nested relationships, 
                // so these must be actual relationships
                issuesCreate: [{ id: "3", title: "Report 1" }, { id: "4", title: "Report 2" }],
                parentConnect: "5", // Has implicit disconnect
            },
        };
        inputToMaps(action, input, format, "id", closestWithId, idsByAction, idsByType, inputsById, inputsByType);

        expectOnlyTheseArrays(idsByAction, {
            Create: ["3", "4"],
            Connect: ["5"],
            Disconnect: ["Project|2.ProjectVersion|parent"],
            Update: ["1", "2"],
        });
        expectOnlyTheseArrays(idsByType, {
            User: ["1"],
            Project: ["2"],
            ProjectVersion: ["5", "Project|2.ProjectVersion|parent"],
            Issue: ["3", "4"],
        });
        expect(inputsById["1"].input).to.deep.equal({
            ...input,
            projectUpdate: "2",
        });
        expect(inputsById["2"].input).to.deep.equal({
            ...input.projectUpdate,
            issuesCreate: ["3", "4"],
            parentConnect: "5",
        });
        expect(inputsById["3"].input).to.deep.equal(input.projectUpdate.issuesCreate[0]);
        expect(inputsById["4"].input).to.deep.equal(input.projectUpdate.issuesCreate[1]);
    });
});
