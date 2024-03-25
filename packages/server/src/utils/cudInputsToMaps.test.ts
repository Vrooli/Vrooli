/* eslint-disable @typescript-eslint/ban-ts-comment */
import { uuid } from "@local/shared";
import pkg from "../__mocks__/@prisma/client";
import { ModelMap } from "../models";
import { convertPlaceholders, determineModelType, fetchAndMapPlaceholder, initializeInputMaps, inputToMaps, processConnectDisconnectOrDelete, processCreateOrUpdate, processInputObjectField, replacePlaceholdersInInputsById, replacePlaceholdersInInputsByType, replacePlaceholdersInMap, updateClosestWithId } from "./cudInputsToMaps";
import { InputNode } from "./inputNode";
import { IdsByAction, IdsByType, InputsByType } from "./types";

const { PrismaClient } = pkg;

describe("fetchAndMapPlaceholder", () => {
    let placeholderToIdMap;

    beforeEach(async () => {
        // Initialize the ModelMap, which is used in fetchAndMapPlaceholder
        await ModelMap.init();

        // Reset the mock for each test
        jest.clearAllMocks();

        PrismaClient.injectData({
            User: [
                { id: "123-321", profile: { id: "456", address: { id: "789" } } },
            ],
        });

        placeholderToIdMap = {};
    });

    afterEach(() => {
        PrismaClient.clearData();
    });

    it("should fetch and return the correct ID for a new placeholder", async () => {
        const placeholder = "user|123-321.prof|profile";
        await fetchAndMapPlaceholder(placeholder, placeholderToIdMap);
        expect(placeholderToIdMap[placeholder]).toBe("456"); // The profile's ID
    });

    it("should return the cached ID for an already processed placeholder", async () => {
        placeholderToIdMap["user|123-321.prof|profile"] = "420"; // Cache an ID

        const placeholder = "user|123-321.prof|profile";
        await fetchAndMapPlaceholder(placeholder, placeholderToIdMap);

        expect(placeholderToIdMap[placeholder]).toBe("420"); // The cached ID
        // @ts-ignore Testing runtime scenario
        expect(PrismaClient.instance.user.findUnique).not.toHaveBeenCalled();
    });

    it("should return null if object is not found", async () => {
        const placeholder = "user|999.prof|profile";
        await fetchAndMapPlaceholder(placeholder, placeholderToIdMap);

        expect(placeholderToIdMap[placeholder]).toBeNull();
    });

    it("should handle nested relations correctly", async () => {
        const placeholder = "user|123-321.prof|profile.addr|address";
        await fetchAndMapPlaceholder(placeholder, placeholderToIdMap);

        expect(placeholderToIdMap[placeholder]).toBe("789"); // The address's ID
    });

    it("should handle placeholder with invalid format", async () => {
        const placeholder = "invalidFormatPlaceholder"; // No delimiters or parts
        await expect(async () => {
            await fetchAndMapPlaceholder(placeholder, placeholderToIdMap);
        }).rejects.toThrow("InternalError");
    });

    it("should handle non-existent objectType in placeholder", async () => {
        const placeholder = "nonExistentType|123.relation|value";
        await expect(async () => {
            await fetchAndMapPlaceholder(placeholder, placeholderToIdMap);
        }).rejects.toThrow("InternalError");
    });

    it("should handle valid objectType but invalid rootId", async () => {
        const placeholder = "user|invalidId.relation|value";
        await fetchAndMapPlaceholder(placeholder, placeholderToIdMap);

        expect(placeholderToIdMap[placeholder]).toBeNull(); // Valid objectType, but rootId does not exist
    });

    it("should handle non-existing relation type in placeholder", async () => {
        const placeholder = "user|123-321.nonExistingRelation|value";
        await fetchAndMapPlaceholder(placeholder, placeholderToIdMap);

        expect(placeholderToIdMap[placeholder]).toBeNull(); // Non-existing relation type
    });

    it("should handle valid relation type but invalid relation in placeholder", async () => {
        const placeholder = "user|123-321.prof|invalidRelation";
        await fetchAndMapPlaceholder(placeholder, placeholderToIdMap);

        expect(placeholderToIdMap[placeholder]).toBeNull(); // Valid relation type but invalid relation
    });

    it("should handle unnecessary placeholders (placeholders which contain the ID already)", async () => {
        const idInPlaceholder = uuid();
        const placeholder = `Note|${idInPlaceholder}`; // This check only works with valid IDs
        await fetchAndMapPlaceholder(placeholder, placeholderToIdMap);

        expect(placeholderToIdMap[placeholder]).toBe(idInPlaceholder);
    });
});

describe("replacePlaceholdersInMap", () => {
    let placeholderToIdMap;

    beforeEach(async () => {
        // Initialize the ModelMap, which is used in fetchAndMapPlaceholder
        await ModelMap.init();

        jest.clearAllMocks();

        PrismaClient.injectData({
            User: [
                { id: "123", profile: { id: "456", address: { id: "789" } } },
            ],
            Note: [
                { id: "333", owner: { id: "420" } },
            ],
        });

        placeholderToIdMap = {};
    });

    afterEach(() => {
        PrismaClient.clearData();
    });

    it("should replace placeholders with actual IDs", async () => {
        const idsMap = {
            "User": ["user|123.prof|profile"],
            "Update": ["user|123"],
        };

        await replacePlaceholdersInMap(idsMap, placeholderToIdMap);

        expect(idsMap["User"][0]).toEqual("456");
        expect(idsMap["Update"][0]).toEqual("123");
        expect(Object.keys(placeholderToIdMap)).toHaveLength(2);
        expect(placeholderToIdMap["user|123.prof|profile"]).toEqual("456");
        expect(placeholderToIdMap["user|123"]).toEqual("123");
    });

    it("should handle multiple placeholders", async () => {
        const idsMap = {
            "User": ["User|123.Prof|profile", "User|123.Prof|profile.Address|address"],
            "Update": ["User|123", "Note|333.Owner|owner"],
        };

        await replacePlaceholdersInMap(idsMap, placeholderToIdMap);

        expect(idsMap["User"]).toEqual(["456", "789"]); // Profile and Address IDs
        expect(idsMap["Update"]).toEqual(["123", "420"]); // User and Owner IDs
        expect(Object.keys(placeholderToIdMap)).toHaveLength(4);
        expect(placeholderToIdMap["User|123.Prof|profile"]).toEqual("456");
        expect(placeholderToIdMap["User|123.Prof|profile.Address|address"]).toEqual("789");
        expect(placeholderToIdMap["User|123"]).toEqual("123");
        expect(placeholderToIdMap["Note|333.Owner|owner"]).toEqual("420");
    });

    it("should leave non-placeholder IDs unchanged", async () => {
        const idsMap = {
            "User": ["123", "User|123.Prof|profile"],
            "Update": ["456"],
        };

        await replacePlaceholdersInMap(idsMap, placeholderToIdMap);

        expect(idsMap["User"]).toEqual(["123", "456"]);
        expect(idsMap["Update"]).toEqual(["456"]);
        expect(Object.keys(placeholderToIdMap)).toHaveLength(1);
        expect(placeholderToIdMap["User|123.Prof|profile"]).toEqual("456");
    });

    it("should skip querying database when placeholder is already in map", async () => {
        const idsMap = {
            "User": ["user|123.prof|profile"],
        };

        placeholderToIdMap["user|123.prof|profile"] = "420"; // Cache an ID

        await replacePlaceholdersInMap(idsMap, placeholderToIdMap);

        expect(idsMap["User"][0]).toEqual("420"); // Cached ID
        expect(Object.keys(placeholderToIdMap)).toHaveLength(1);
        expect(placeholderToIdMap["user|123.prof|profile"]).toEqual("420");
    });
});

describe("replacePlaceholdersInInputsById", () => {
    let placeholderToIdMap;
    const placeholder1 = "user|123.prof|profile";

    beforeEach(async () => {
        // Initialize the ModelMap, which is used in fetchAndMapPlaceholder
        await ModelMap.init();

        jest.clearAllMocks();

        PrismaClient.injectData({
            User: [
                { id: "123", profile: { id: "456", address: { id: "789" } } },
            ],
        });

        placeholderToIdMap = {};
    });

    afterEach(() => {
        PrismaClient.clearData();
    });

    it("should replace placeholders with string inputs", async () => {
        const inputsById = { [placeholder1]: { node: new InputNode("User", placeholder1, "Delete"), input: placeholder1 } };

        await replacePlaceholdersInInputsById(inputsById, placeholderToIdMap);

        expect(Object.keys(inputsById)).toHaveLength(1);
        expect(inputsById["456"].node.id).toEqual("456");
        expect(inputsById["456"].input).toEqual("456");
        expect(Object.keys(placeholderToIdMap)).toHaveLength(1);
        expect(placeholderToIdMap[placeholder1]).toEqual("456");
    });

    it("should replace placeholders with object inputs", async () => {
        const inputsById = {
            [placeholder1]: {
                node: new InputNode("User", placeholder1, "Update"), input: {
                    id: placeholder1,
                    name: "Test",
                },
            },
        };

        await replacePlaceholdersInInputsById(inputsById, placeholderToIdMap);

        expect(Object.keys(inputsById)).toHaveLength(1);
        expect(inputsById["456"].node.id).toEqual("456");
        expect(inputsById["456"].input).toEqual({ id: "456", name: "Test" });
        expect(Object.keys(placeholderToIdMap)).toHaveLength(1);
        expect(placeholderToIdMap[placeholder1]).toEqual("456");
    });

    it("should leave non-placeholders unchanged", async () => {
        const inputsById = {
            [placeholder1]: { node: new InputNode("User", placeholder1, "Delete"), input: placeholder1 },
            "999": { node: new InputNode("User", "999", "Delete"), input: "999" },
        };

        await replacePlaceholdersInInputsById(inputsById, placeholderToIdMap);

        expect(Object.keys(inputsById)).toHaveLength(2);
        expect(inputsById["456"].node.id).toEqual("456");
        expect(inputsById["456"].input).toEqual("456");
        expect(inputsById["999"].node.id).toEqual("999");
        expect(inputsById["999"].input).toEqual("999");
        expect(Object.keys(placeholderToIdMap)).toHaveLength(1);
        expect(placeholderToIdMap[placeholder1]).toEqual("456");
    });

    it("should skip querying database when placeholder is already in map", async () => {
        const inputsById = { [placeholder1]: { node: new InputNode("User", placeholder1, "Delete"), input: placeholder1 } };

        placeholderToIdMap[placeholder1] = "420"; // Cache an ID

        await replacePlaceholdersInInputsById(inputsById, placeholderToIdMap);

        expect(Object.keys(inputsById)).toHaveLength(1);
        expect(inputsById["420"].node.id).toEqual("420");
        expect(inputsById["420"].input).toEqual("420");
        expect(Object.keys(placeholderToIdMap)).toHaveLength(1);
        expect(placeholderToIdMap[placeholder1]).toEqual("420");
    });
});

describe("replacePlaceholdersInInputsByType", () => {
    let placeholderToIdMap;
    const placeholder1 = "user|123.prof|profile";

    beforeEach(async () => {
        // Initialize the ModelMap, which is used in fetchAndMapPlaceholder
        await ModelMap.init();

        jest.clearAllMocks();

        PrismaClient.injectData({
            User: [
                { id: "123", profile: { id: "456", address: { id: "789" } } },
            ],
        });

        placeholderToIdMap = {};
    });

    afterEach(() => {
        PrismaClient.clearData();
    });

    it("should replace placeholders with string inputs", async () => {
        const inputsByType = {
            User: {
                Connect: [],
                Create: [],
                Delete: [{ node: new InputNode("User", placeholder1, "Delete"), input: placeholder1 }],
                Disconnect: [],
                Read: [],
                Update: [],
            },
        };

        await replacePlaceholdersInInputsByType(inputsByType, placeholderToIdMap);

        expect(inputsByType.User.Delete).toHaveLength(1);
        expect(inputsByType.User.Delete[0].node.id).toEqual("456");
        expect(inputsByType.User.Delete[0].input).toEqual("456");
        expect(Object.keys(placeholderToIdMap)).toHaveLength(1);
        expect(placeholderToIdMap[placeholder1]).toEqual("456");
    });

    it("should replace placeholders with object inputs", async () => {
        const inputsByType = {
            User: {
                Connect: [],
                Create: [],
                Delete: [],
                Disconnect: [],
                Read: [],
                Update: [{ node: new InputNode("User", placeholder1, "Update"), input: { id: placeholder1, name: "Test" } }],
            },
        };

        await replacePlaceholdersInInputsByType(inputsByType, placeholderToIdMap);

        expect(inputsByType.User.Update).toHaveLength(1);
        expect(inputsByType.User.Update[0].node.id).toEqual("456");
        expect(inputsByType.User.Update[0].input.id).toEqual("456");
        expect(inputsByType.User.Update[0].input.name).toEqual("Test");
        expect(Object.keys(placeholderToIdMap)).toHaveLength(1);
        expect(placeholderToIdMap[placeholder1]).toEqual("456");
    });

    it("should leave non-placeholders unchanged", async () => {
        const nonPlaceholder = "999";
        const inputsByType = {
            User: {
                Connect: [],
                Create: [],
                Delete: [
                    { node: new InputNode("User", placeholder1, "Delete"), input: placeholder1 },
                    { node: new InputNode("User", nonPlaceholder, "Delete"), input: nonPlaceholder },
                ],
                Disconnect: [],
                Read: [],
                Update: [],
            },
        };

        await replacePlaceholdersInInputsByType(inputsByType, placeholderToIdMap);

        expect(inputsByType.User.Delete).toHaveLength(2);
        expect(inputsByType.User.Delete[0].node.id).toEqual("456");
        expect(inputsByType.User.Delete[0].input).toEqual("456");
        expect(inputsByType.User.Delete[1].node.id).toEqual(nonPlaceholder);
        expect(inputsByType.User.Delete[1].input).toEqual(nonPlaceholder);
        expect(Object.keys(placeholderToIdMap)).toHaveLength(1);
        expect(placeholderToIdMap[placeholder1]).toEqual("456");
        expect(placeholderToIdMap).not.toHaveProperty(nonPlaceholder);
    });

    it("should skip querying database when placeholder is already in map", async () => {
        placeholderToIdMap[placeholder1] = "420"; // Pre-populate the map

        const inputsByType = {
            User: {
                Connect: [],
                Create: [],
                Delete: [{ node: new InputNode("User", placeholder1, "Delete"), input: placeholder1 }],
                Disconnect: [],
                Read: [],
                Update: [],
            },
        };

        await replacePlaceholdersInInputsByType(inputsByType, placeholderToIdMap);

        expect(inputsByType.User.Delete).toHaveLength(1);
        expect(inputsByType.User.Delete[0].node.id).toEqual("420");
        expect(inputsByType.User.Delete[0].input).toEqual("420");
        expect(Object.keys(placeholderToIdMap)).toHaveLength(1);
        expect(placeholderToIdMap[placeholder1]).toEqual("420");
    });
});

describe("convertPlaceholders", () => {
    let idsByAction, idsByType, inputsById, inputsByType;
    const initialIdsByAction = { Create: [], Update: [], Connect: [], Disconnect: [] };
    const initialIdsByType = {};
    const initialInputsById = {};
    const initialInputsByType = {};
    const placeholder1 = "user|123.prof|profile";

    beforeEach(() => {
        PrismaClient.injectData({
            User: [
                { id: "123", profile: { id: "456", address: { id: "789" } } },
            ],
        });
        idsByAction = JSON.parse(JSON.stringify(initialIdsByAction));
        idsByType = JSON.parse(JSON.stringify(initialIdsByType));
        inputsById = JSON.parse(JSON.stringify(initialInputsById));
        inputsByType = JSON.parse(JSON.stringify(initialInputsByType));
    });

    afterEach(() => {
        PrismaClient.clearData();
    });

    it("should replace placeholders with actual IDs", async () => {
        idsByType = { "User": [placeholder1] };
        idsByAction = { "Update": [placeholder1] };
        inputsById = { [placeholder1]: { node: new InputNode("User", placeholder1, "Delete"), input: placeholder1 } };
        inputsByType = { "User": { "Delete": [{ node: new InputNode("User", placeholder1, "Delete"), input: placeholder1 }] } };

        await convertPlaceholders({ idsByAction, idsByType, inputsById, inputsByType });

        expect(idsByType["User"]).toEqual(["456"]);
        expect(idsByAction["Update"]).toEqual(["456"]);
        expect(Object.keys(inputsById)).toHaveLength(1);
        expect(inputsById["456"].node.id).toEqual("456");
        expect(inputsById["456"].input).toEqual("456");
        expect(inputsByType["User"].Delete).toHaveLength(1);
        expect(inputsByType["User"].Delete[0].node.id).toEqual("456");
        expect(inputsByType["User"].Delete[0].input).toEqual("456");
        expect(inputsByType["User"].Delete).toHaveLength(1);
    });

    it("should replace placeholders with null if not found", async () => {
        idsByType = { "User": ["user|999.prof|profile"] };
        idsByAction = { "Create": ["user|999.prof|profile"] };
        inputsById = { "user|999.prof|profile": { node: new InputNode("User", "user|999.prof|profile", "Create"), input: "user|999.prof|profile" } };
        inputsByType = { "User": { "Create": [{ node: new InputNode("User", "user|999.prof|profile", "Create"), input: "user|999.prof|profile" }] } };

        await convertPlaceholders({ idsByAction, idsByType, inputsById, inputsByType });

        expect(idsByType["User"]).toEqual([null]);
        expect(idsByAction["Create"]).toEqual([null]);
        expect(Object.keys(inputsById)).toHaveLength(0);
        expect(inputsById["user|999.prof|profile"]).toBeUndefined();
        expect(inputsByType["User"].Create).toHaveLength(1);
        expect(inputsByType["User"].Create[0].node.id).toEqual(null);
        expect(inputsByType["User"].Create[0].input).toEqual(null);
    });

    it("should not perform any operation if there are no placeholders", async () => {
        idsByType = { "User": ["123"] };
        idsByAction = { "Create": ["123"] };
        inputsById = { "123": { node: new InputNode("User", "123", "Create"), input: "123" } };
        inputsByType = { "User": { "Create": [{ node: new InputNode("User", "123", "Create"), input: "123" }] } };

        await convertPlaceholders({ idsByAction, idsByType, inputsById, inputsByType });

        expect(idsByType["User"]).toEqual(["123"]);
        expect(idsByAction["Create"]).toEqual(["123"]);
        expect(Object.keys(inputsById)).toHaveLength(1);
        expect(inputsById["123"].node.id).toEqual("123");
        expect(inputsById["123"].input).toEqual("123");
        expect(inputsByType["User"].Create).toHaveLength(1);
        expect(inputsByType["User"].Create[0].node.id).toEqual("123");
        expect(inputsByType["User"].Create[0].input).toEqual("123");
    });

    it("should handle multiple placeholders and nested cases correctly", async () => {
        idsByType = { "User": ["user|123.prof|profile", "user|123.prof|profile.addr|address"] };
        idsByAction = { "Create": ["user|123.prof|profile.addr|address"] };
        inputsById = {
            "user|123.prof|profile": { node: new InputNode("User", "user|123.prof|profile", "Delete"), input: "user|123.prof|profile" },
            "user|123.prof|profile.addr|address": { node: new InputNode("User", "user|123.prof|profile.addr|address", "Create"), input: "user|123.prof|profile.addr|address" },
        };
        inputsByType = {
            "User": {
                "Delete": [{ node: new InputNode("User", "user|123.prof|profile", "Delete"), input: "user|123.prof|profile" }],
                "Create": [{ node: new InputNode("User", "user|123.prof|profile.addr|address", "Create"), input: "user|123.prof|profile.addr|address" }],
            },
        };

        await convertPlaceholders({ idsByAction, idsByType, inputsById, inputsByType });

        expect(idsByType["User"]).toEqual(["456", "789"]);
        expect(idsByAction["Create"]).toEqual(["789"]);
        expect(Object.keys(inputsById)).toHaveLength(2);
        expect(inputsById["456"].node.id).toEqual("456");
        expect(inputsById["456"].input).toEqual("456");
        expect(inputsById["789"].node.id).toEqual("789");
        expect(inputsById["789"].input).toEqual("789");
        expect(inputsByType["User"].Delete).toHaveLength(1);
        expect(inputsByType["User"].Delete[0].node.id).toEqual("456");
        expect(inputsByType["User"].Delete[0].input).toEqual("456");
        expect(inputsByType["User"].Create).toHaveLength(1);
        expect(inputsByType["User"].Create[0].node.id).toEqual("789");
        expect(inputsByType["User"].Create[0].input).toEqual("789");
    });

    it("should maintain data integrity across different types and actions", async () => {
        idsByType = { "User": ["user|123.prof|profile"], "Note": ["user|123.prof|profile.addr|address"] };
        idsByAction = { "Create": ["user|123.prof|profile"], "Update": ["user|123.prof|profile.addr|address"] };
        inputsById = {
            "user|123.prof|profile": { node: new InputNode("User", "user|123.prof|profile", "Delete"), input: "user|123.prof|profile" },
            "user|123.prof|profile.addr|address": { node: new InputNode("Note", "user|123.prof|profile.addr|address", "Update"), input: "user|123.prof|profile.addr|address" },
        };
        inputsByType = {
            "User": {
                "Delete": [{ node: new InputNode("User", "user|123.prof|profile", "Delete"), input: "user|123.prof|profile" }],
                "Create": [{ node: new InputNode("User", "user|123.prof|profile", "Create"), input: "user|123.prof|profile" }],
            },
            "Note": {
                "Update": [{ node: new InputNode("Note", "user|123.prof|profile.addr|address", "Update"), input: "user|123.prof|profile.addr|address" }],
            },
        };

        await convertPlaceholders({ idsByAction, idsByType, inputsById, inputsByType });

        expect(idsByType["User"]).toEqual(["456"]);
        expect(idsByType["Note"]).toEqual(["789"]);
        expect(idsByAction["Create"]).toEqual(["456"]);
        expect(idsByAction["Update"]).toEqual(["789"]);
        expect(Object.keys(inputsById)).toHaveLength(2);
        expect(inputsById["456"].node.id).toEqual("456");
        expect(inputsById["456"].input).toEqual("456");
        expect(inputsById["789"].node.id).toEqual("789");
        expect(inputsById["789"].input).toEqual("789");
        expect(inputsByType["User"].Delete).toHaveLength(1);
        expect(inputsByType["User"].Delete[0].node.id).toEqual("456");
        expect(inputsByType["User"].Delete[0].input).toEqual("456");
        expect(inputsByType["User"].Create).toHaveLength(1);
        expect(inputsByType["User"].Create[0].node.id).toEqual("456");
        expect(inputsByType["User"].Create[0].input).toEqual("456");
        expect(inputsByType["Note"].Update).toHaveLength(1);
        expect(inputsByType["Note"].Update[0].node.id).toEqual("789");
        expect(inputsByType["Note"].Update[0].input).toEqual("789");
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

        expect(idsByAction[actionType]).toHaveLength(0);
        expect(idsByType[objectType]).toHaveLength(0);
        expect(inputsByType[objectType]).toEqual({
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

        expect(idsByAction[actionType]).toEqual(["existingId"]);
    });

    // Test behavior when idsByType is already initialized
    it("should not modify idsByType if it is already initialized", () => {
        const actionType = "Update";
        const objectType = "RoutineVersion";
        idsByType[objectType] = ["existingId"];

        initializeInputMaps(actionType, objectType, idsByAction, idsByType, inputsByType);

        expect(idsByType[objectType]).toEqual(["existingId"]);
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

        expect(inputsByType[objectType]).toEqual({
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
        expect(result).toBeNull();
    });

    it("should return null if input has no ID and no relation", () => {
        const result = updateClosestWithId("Update", { name: "Test" }, idField, "User", { __typename: "User", id: "123", path: "" });
        expect(result).toBeNull();
    });

    it("should return updated path if input has no ID but has a relation", () => {
        const result = updateClosestWithId("Update", { name: "Test" }, idField, "User", { __typename: "User", id: "123", path: "profile" }, "avatar");
        expect(result).toEqual({ __typename: "User", id: "123", path: "profile.avatar" });
    });

    it("should return null if input has no ID and no closestWithId but has a relation", () => {
        const result = updateClosestWithId("Update", { name: "Test" }, idField, "User", null, "avatar");
        expect(result).toBeNull();
    });

    it("should return object with empty path if input has an ID", () => {
        const result = updateClosestWithId("Update", { id: "456", name: "Test" }, idField, "User", { __typename: "User", id: "123", path: "profile" });
        expect(result).toEqual({ __typename: "User", id: "456", path: "" });
    });

    it("should handle string inputs as IDs", () => {
        const result = updateClosestWithId("Update", "789", idField, "User", { __typename: "User", id: "123", path: "profile" });
        expect(result).toEqual({ __typename: "User", id: "789", path: "" });
    });

    it("should return null if input is a boolean (implying a Disconnect or Delete), and there is no relation", () => {
        const result = updateClosestWithId("Update", true, idField, "User", { __typename: "User", id: "123", path: "profile" });
        expect(result).toBeNull();
    });

    it("should return updated path if input is a boolean (implying a Disconnect or Delete), and there is a relation", () => {
        const result = updateClosestWithId("Update", true, idField, "User", { __typename: "User", id: "123", path: "profile" }, "avatar");
        expect(result).toEqual({ __typename: "User", id: "123", path: "profile.avatar" });
    });
});

describe("determineModelType", () => {
    const commentFormat = {
        gqlRelMap: {
            __typename: "Comment",
            owner: {
                ownedByUser: "User",
                ownedByOrganization: "Organization",
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

    const labelFormat = {
        gqlRelMap: {
            __typename: "Label",
            owner: {
                ownedByUser: "User",
                ownedByOrganization: "Organization",
            },
        },
        unionFields: {
            owner: {},
        },
    };

    it("returns correct __typename for standard fields", () => {
        const __typename = determineModelType("reportsCreate", "reports", {}, commentFormat);
        expect(__typename).toBe("Report");
    });

    it("handles simple unions correctly", () => {
        const __typename = determineModelType("ownedByUserConnect", "ownedByUser", {}, labelFormat);
        expect(__typename).toBe("User");
    });

    it("handles complex unions correctly", () => {
        const input = { createdFor: "Post" };
        const __typename = determineModelType("forConnect", "for", input, commentFormat);
        expect(__typename).toBe("Post");
    });

    it("returns null for special cases like translations", () => {
        const __typename = determineModelType("translationsCreate", "translations", {}, commentFormat);
        expect(__typename).toBeNull();
    });

    it("throws an error when field is not found in gqlRelMap", () => {
        expect(() => {
            determineModelType("nonexistentFieldCreate", "nonexistentField", {}, commentFormat);
        }).toThrowError("InternalError");
    });

    it("throws an error for missing union typeField in input", () => {
        expect(() => {
            determineModelType("forConnect", "for", {}, commentFormat);
        }).toThrowError("InternalError");
    });
});

// NOTE: These relations only exist in either a Create or Update mutation's data. 
// When `closestWithId` is null, it means that the mutation is a Create mutation.
// Some actions are not allowed in Create mutations, so they should throw an error.
describe("processCreateOrUpdate", () => {
    let idsByAction, idsByType, inputsById, inputsByType, parentNode, closestWithId;

    beforeEach(() => {
        // Initialize the maps and parentNode
        idsByAction = { Create: [], Update: [] };
        idsByType = {};
        inputsById = {};
        inputsByType = {};
        parentNode = new InputNode("RoutineVersion", "parentId", "Update");
        closestWithId = { __typename: "RoutineVersion", id: "parentId", path: "" };
    });

    it("correctly processes a 'Create' action with closestWithId", () => {
        const childFormat = { gqlRelMap: { __typename: "User" as const } };
        const childInput = { id: "childID", name: "Test", someRelation: { id: "789" } };
        const action = "Create";
        const fieldName = "child";
        const idField = "id";

        processCreateOrUpdate(action, childInput, childFormat, fieldName, idField, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);

        expect(idsByAction.Create).toEqual(["childID"]);
        expect(idsByType.User).toEqual(["childID"]);
        expect(inputsById["childID"].node.id).toEqual("childID");
        expect(inputsById["childID"].input).toEqual(childInput);
        expect(inputsByType.User.Create[0].node.id).toEqual("childID");
        expect(inputsByType.User.Create[0].input).toEqual(childInput);
        expect(parentNode.children[0].id).toEqual("childID");
        expect(parentNode.children[0].parent).toBe(parentNode);
        // Shouldn't change closestWithId, since it's not relevant outside of the recursive call
        expect(closestWithId).toBe(closestWithId);
    });

    it("correctly processes a 'Create' action without closestWithId", () => {
        const childFormat = { gqlRelMap: { __typename: "User" as const } };
        const childInput = { id: "childID", name: "Test", someRelation: { id: "789" } };
        const action = "Create";
        const fieldName = "child";
        const idField = "id";

        processCreateOrUpdate(action, childInput, childFormat, fieldName, idField, parentNode, null, idsByAction, idsByType, inputsById, inputsByType);

        expect(idsByAction.Create).toEqual(["childID"]);
        expect(idsByType.User).toEqual(["childID"]);
        expect(inputsById["childID"].node.id).toEqual("childID");
        expect(inputsById["childID"].input).toEqual(childInput);
        expect(inputsByType.User.Create[0].node.id).toEqual("childID");
        expect(inputsByType.User.Create[0].input).toEqual(childInput);
        expect(parentNode.children[0].id).toEqual("childID");
        expect(parentNode.children[0].parent).toBe(parentNode);
        // Shouldn't change closestWithId, since it's not relevant outside of the recursive call
        expect(closestWithId).toBe(closestWithId);
    });


    it("correctly processes an 'Update' action with closestWithId", () => {
        const childFormat = { gqlRelMap: { __typename: "User" as const } };
        const childInput = { id: "childID", name: "Test", someRelation: { id: "789" } };
        const action = "Update";
        const fieldName = "child";
        const idField = "id";

        processCreateOrUpdate(action, childInput, childFormat, fieldName, idField, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);

        expect(idsByAction.Update).toEqual(["childID"]);
        expect(idsByType.User).toEqual(["childID"]);
        expect(inputsById["childID"].node.id).toEqual("childID");
        expect(inputsById["childID"].input).toEqual(childInput);
        expect(inputsByType.User.Update[0].node.id).toEqual("childID");
        expect(inputsByType.User.Update[0].input).toEqual(childInput);
        expect(parentNode.children[0].id).toEqual("childID");
        expect(parentNode.children[0].parent).toBe(parentNode);
        // Shouldn't change closestWithId, since it's not relevant outside of the recursive call
        expect(closestWithId).toBe(closestWithId);
    });

    it("throws an error when processing an 'Update' action without closestWithId", () => {
        const childFormat = { gqlRelMap: { __typename: "User" as const } };
        const childInput = { id: "childID", name: "Test", someRelation: { id: "789" } };
        const action = "Update";
        const fieldName = "child";
        const idField = "id";

        expect(() => {
            processCreateOrUpdate(action, childInput, childFormat, fieldName, idField, parentNode, null, idsByAction, idsByType, inputsById, inputsByType);
        }).toThrow("InternalError");
    });

    it("throws an error for invalid actions", () => {
        const childFormat = { gqlRelMap: { __typename: "User" as const } };
        const childInput = { id: "childID", name: "Test", someRelation: { id: "789" } };
        const action = "Connect"; // This is process CREATE or UPDATE, not CONNECT
        const fieldName = "child";
        const idField = "id";

        expect(() => {
            processCreateOrUpdate(action, childInput, childFormat, fieldName, idField, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        }).toThrow("InternalError");
    });

    it("correctly processes multiple different objects", () => {
        const childFormat = { gqlRelMap: { __typename: "User" as const } };
        const firstChildInput = { id: "childID1", name: "TestUser1", someRelation: { id: "789" } };
        const secondChildInput = { id: "childID2", name: "TestUser2", someRelation: { id: "790" } };
        const action = "Create";
        const fieldName = "child";
        const idField = "id";

        // Call the function with two different input objects
        processCreateOrUpdate(action, firstChildInput, childFormat, fieldName, idField, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        processCreateOrUpdate(action, secondChildInput, childFormat, fieldName, idField, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);

        expect(idsByAction[action]).toEqual(expect.arrayContaining([firstChildInput.id, secondChildInput.id]));
        expect(idsByType[childFormat.gqlRelMap.__typename]).toEqual(expect.arrayContaining([firstChildInput.id, secondChildInput.id]));
        expect(parentNode.children).toHaveLength(2);
        // Ensure each child node has the expected properties
        expect(parentNode.children[0].id).toEqual(firstChildInput.id);
        expect(parentNode.children[1].id).toEqual(secondChildInput.id);
    });
});

describe("processConnectDisconnectOrDelete", () => {
    let idsByAction, idsByType, inputsById, inputsByType, parentNode, closestWithId;

    beforeEach(() => {
        idsByAction = {};
        idsByType = {};
        inputsById = {};
        inputsByType = {};
        parentNode = new InputNode("RoutineVersion", "parentId", "Update");
        closestWithId = { __typename: "RoutineVersion", id: "parentId", path: "" };
    });

    it("initializes input maps correctly", () => {
        processConnectDisconnectOrDelete("123", false, "Delete", "fieldName", "User", parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        expect(idsByAction.Delete).toEqual(["123"]);
        expect(idsByType.User).toEqual(["123"]);
        expect(parentNode.children[0].id).toEqual("123");
    });

    it("adds ID to maps for toMany Connect action with closestWithId", () => {
        processConnectDisconnectOrDelete("123", false, "Connect", "fieldName", "User", parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        expect(idsByAction.Connect).toEqual(["123"]);
        expect(idsByType.User).toEqual(["123"]);
        expect(parentNode.children[0].id).toEqual("123");
    });

    it("adds ID to maps for toMany Connect action without closestWithId", () => {
        processConnectDisconnectOrDelete("123", false, "Connect", "fieldName", "User", parentNode, null, idsByAction, idsByType, inputsById, inputsByType);
        expect(idsByAction.Connect).toEqual(["123"]);
        expect(idsByType.User).toEqual(["123"]);
        expect(parentNode.children[0].id).toEqual("123");
    });

    it("adds ID to maps for toMany Disconnect action with closestWithId", () => {
        processConnectDisconnectOrDelete("123", false, "Disconnect", "fieldName", "User", parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        expect(idsByAction.Disconnect).toContain("123");
        expect(idsByType.User).toContain("123");
        expect(parentNode.children[0].id).toEqual("123");
    });

    it("throws error for toMany Disconnect action without closestWithId", () => {
        expect(() => {
            processConnectDisconnectOrDelete("123", false, "Disconnect", "fieldName", "User", parentNode, null, idsByAction, idsByType, inputsById, inputsByType);
        }).toThrow("InternalError");
    });

    it("adds ID and input to maps for toMany Delete action with closestWithId", () => {
        processConnectDisconnectOrDelete("123", false, "Delete", "fieldName", "User", parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        expect(idsByAction.Delete).toContain("123");
        expect(idsByType.User).toContain("123");
        expect(inputsById["123"].node.__typename).toBe("User");
        expect(inputsById["123"].input).toBe("123");
        expect(inputsByType.User.Delete[0].node.__typename).toBe("User");
        expect(inputsByType.User.Delete[0].input).toBe("123");
        expect(parentNode.children[0].id).toEqual("123");
    });

    it("throws error for toMany Delete action without closestWithId", () => {
        expect(() => {
            processConnectDisconnectOrDelete("123", false, "Delete", "fieldName", "User", parentNode, null, idsByAction, idsByType, inputsById, inputsByType);
        }).toThrow("InternalError");
    });

    it("adds ID and placeholder for toOne Connect relationships with closestWithId", () => {
        const expectedPlaceholder = "RoutineVersion|parentId.fieldName";

        processConnectDisconnectOrDelete("123", true, "Connect", "fieldName", "User", parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        expect(idsByAction.Connect).toContain("123");
        expect(idsByAction.Disconnect).toContain(expectedPlaceholder);
        expect(idsByType.User).toContain(expectedPlaceholder);
        expect(idsByType.User).toContain("123");
        expect(parentNode.children[0].id).toEqual(expectedPlaceholder);
        expect(parentNode.children[0].action).toBe("Disconnect");
        expect(parentNode.children[1].id).toEqual("123");
        expect(parentNode.children[1].action).toBe("Connect");
        expect(parentNode.children).toHaveLength(2);
    });

    // NOTE: This only includes a placeholder because toOne Disconnects use a boolean 
    // to disconnect the relationship, not an ID. We typically pass in a blank string 
    // for the ID in reality
    it("adds placeholder for toOne Disconnect relationships with closestWithId", () => {
        const expectedPlaceholder = "RoutineVersion|parentId.fieldName";

        processConnectDisconnectOrDelete("123", true, "Disconnect", "fieldName", "User", parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        expect(idsByAction.Disconnect).toContain(expectedPlaceholder);
        expect(idsByAction.Disconnect).not.toContain("123");
        expect(idsByType.User).toContain(expectedPlaceholder);
        expect(idsByType.User).not.toContain("123");
        expect(parentNode.children[0].id).toEqual(expectedPlaceholder);
        expect(parentNode.children[0].action).toBe("Disconnect");
        expect(parentNode.children).toHaveLength(1);
    });

    it("adds only ID for toOne Connect relationships without closestWithId", () => {
        processConnectDisconnectOrDelete("123", true, "Connect", "fieldName", "User", parentNode, null, idsByAction, idsByType, inputsById, inputsByType);
        expect(idsByAction.Connect).toEqual(["123"]);
        expect(idsByType.User).toEqual(["123"]);
        expect(parentNode.children[0].id).toEqual("123");
    });

    it("throws error for toOne Disconnect relationships without closestWithId", () => {
        expect(() => {
            processConnectDisconnectOrDelete("123", true, "Disconnect", "fieldName", "User", parentNode, null, idsByAction, idsByType, inputsById, inputsByType);
        }).toThrow("InternalError");
    });

    it("adds ID and input toOne Delete action with closestWithId", () => {
        const expectedPlaceholder = "RoutineVersion|parentId.fieldName";

        processConnectDisconnectOrDelete("123", true, "Delete", "fieldName", "User", parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        expect(idsByAction.Delete).toContain(expectedPlaceholder);
        expect(idsByAction.Delete).not.toContain("123");
        expect(idsByType.User).toContain(expectedPlaceholder);
        expect(idsByType.User).not.toContain("123");
        expect(inputsById[expectedPlaceholder].node.__typename).toBe("User");
        expect(inputsById[expectedPlaceholder].input).toBe(expectedPlaceholder);
        expect(inputsById["123"]).toBeUndefined();
        expect(inputsByType.User.Delete[0].node.__typename).toBe("User");
        expect(inputsByType.User.Delete[0].input).toBe(expectedPlaceholder);
        expect(parentNode.children[0].id).toEqual(expectedPlaceholder);
        expect(parentNode.children[0].action).toBe("Delete");
        expect(parentNode.children).toHaveLength(1);
    });

    it("throws error for toOne Delete action without closestWithId", () => {
        expect(() => {
            processConnectDisconnectOrDelete("123", true, "Delete", "fieldName", "User", parentNode, null, idsByAction, idsByType, inputsById, inputsByType);
        }).toThrow("InternalError");
    });

    it("skips placeholder when fieldName is null", () => {
        processConnectDisconnectOrDelete("123", true, "Connect", null, "User", parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        expect(idsByAction.Connect).toEqual(["123"]);
        expect(idsByType.User).toEqual(["123"]);
        expect(parentNode.children[0].id).toEqual("123");
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
            gqlRelMap: {
                __typename: "User" as const,
                project: "Project",
                reports: "Report",
            },
        };
        closestWithId = { __typename: "Routine", id: "grandparentId", path: "version" };
    });

    // NOTE: We'll assume that fields with action suffixes are always 
    // relation fields, so we don't need to test the case where the field
    // doesn't contain valid relation data. It would only slow performance.
    it("handles non-relation fields (name not containing action suffix)", () => {
        const field = "description";
        const input = { description: "New Description" };

        processInputObjectField(field, input, format, parentNode, null, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        // Should only be in the inputInfo
        expect(idsByAction).toEqual(initialIdsByAction);
        expect(idsByType).toEqual(initialIdsByType);
        expect(inputsById).toEqual(initialInputsById);
        expect(inputsByType).toEqual(initialInputsByType);
        expect(inputInfo.input[field]).toBe(input[field]);
    });

    it("handles non-array create relations with closestWithId", () => {
        const field = "projectCreate";
        const input = {
            projectCreate: { id: "123", name: "Test", someRelation: { id: "789" } },
            projectUpdate: { id: "456", name: "Test", someRelation: { id: "790" } }, // Should be ignored
        };

        processInputObjectField(field, input, format, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        expect(idsByAction.Create).toEqual([input[field].id]);
        expect(idsByType.Project).toEqual([input[field].id]);
        expect(inputsById[input[field].id].node.id).toEqual(input[field].id);
        expect(inputsById[input[field].id].input).toEqual(input[field]);
        expect(inputsByType.Project.Create[0].node.id).toEqual(input[field].id);
        expect(inputsByType.Project.Create[0].input).toEqual(input[field]);
        expect(parentNode.children[0].id).toEqual(input[field].id);
        expect(parentNode.children[0].parent).toBe(parentNode);
        expect(inputInfo.input[field]).toBe(input[field].id);
    });

    it("handles non-array create relations without closestWithId", () => {
        const field = "projectCreate";
        const input = {
            projectCreate: { id: "123", name: "Test", someRelation: { id: "789" } },
            projectUpdate: { id: "456", name: "Test", someRelation: { id: "790" } }, // Should be ignored
        };

        processInputObjectField(field, input, format, parentNode, null, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        expect(idsByAction.Create).toEqual([input[field].id]);
        expect(idsByType.Project).toEqual([input[field].id]);
        expect(inputsById[input[field].id].node.id).toEqual(input[field].id);
        expect(inputsById[input[field].id].input).toEqual(input[field]);
        expect(inputsByType.Project.Create[0].node.id).toEqual(input[field].id);
        expect(inputsByType.Project.Create[0].input).toEqual(input[field]);
        expect(parentNode.children[0].id).toEqual(input[field].id);
        expect(parentNode.children[0].parent).toBe(parentNode);
        expect(inputInfo.input[field]).toBe(input[field].id);
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

        expect(idsByAction.Create).toEqual(["123", "456"]);
        expect(idsByType.Report).toEqual(["123", "456"]);
        expect(inputsById["123"].node.id).toEqual("123");
        expect(inputsById["123"].input).toEqual(input[field][0]);
        expect(inputsByType.Report.Create[0].node.id).toEqual("123");
        expect(inputsByType.Report.Create[0].input).toEqual(input[field][0]);
        expect(inputsById["456"].node.id).toEqual("456");
        expect(inputsById["456"].input).toEqual(input[field][1]);
        expect(inputsByType.Report.Create[1].node.id).toEqual("456");
        expect(inputsByType.Report.Create[1].input).toEqual(input[field][1]);
        expect(parentNode.children[0].id).toEqual("123");
        expect(parentNode.children[0].parent).toBe(parentNode);
        expect(parentNode.children[1].id).toEqual("456");
        expect(parentNode.children[1].parent).toBe(parentNode);
        expect(inputInfo.input[field]).toEqual(["123", "456"]);
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

        expect(idsByAction.Create).toEqual(["123", "456"]);
        expect(idsByType.Report).toEqual(["123", "456"]);
        expect(inputsById["123"].node.id).toEqual("123");
        expect(inputsById["123"].input).toEqual(input[field][0]);
        expect(inputsByType.Report.Create[0].node.id).toEqual("123");
        expect(inputsByType.Report.Create[0].input).toEqual(input[field][0]);
        expect(inputsById["456"].node.id).toEqual("456");
        expect(inputsById["456"].input).toEqual(input[field][1]);
        expect(inputsByType.Report.Create[1].node.id).toEqual("456");
        expect(inputsByType.Report.Create[1].input).toEqual(input[field][1]);
        expect(parentNode.children[0].id).toEqual("123");
        expect(parentNode.children[0].parent).toBe(parentNode);
        expect(parentNode.children[1].id).toEqual("456");
        expect(parentNode.children[1].parent).toBe(parentNode);
        expect(inputInfo.input[field]).toEqual(["123", "456"]);
    });

    it("handles non-array update relations with closestWithId", () => {
        const field = "projectUpdate";
        const input = {
            projectCreate: { id: "123", name: "Test", someRelation: { id: "789" } }, // Should be ignored
            projectUpdate: { id: "456", name: "Test", someRelation: { id: "790" } },
        };

        processInputObjectField(field, input, format, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        expect(idsByAction.Update).toEqual(["456"]);
        expect(idsByType.Project).toEqual(["456"]);
        expect(inputsById["456"].node.id).toEqual("456");
        expect(inputsById["456"].input).toEqual(input[field]);
        expect(inputsByType.Project.Update[0].node.id).toEqual("456");
        expect(inputsByType.Project.Update[0].input).toEqual(input[field]);
        expect(parentNode.children[0].id).toEqual("456");
        expect(parentNode.children[0].parent).toBe(parentNode);
        expect(inputInfo.input[field]).toBe("456");
    });

    it("handles non-array update relations without closestWithId", () => {
        const field = "projectUpdate";
        const input = {
            projectCreate: { id: "123", name: "Test", someRelation: { id: "789" } }, // Should be ignored
            projectUpdate: { id: "456", name: "Test", someRelation: { id: "790" } },
        };

        // Should throw an error because closestWithId is null, 
        // and you can't update inside a Create mutation
        expect(() => {
            processInputObjectField(field, input, format, parentNode, null, idsByAction, idsByType, inputsById, inputsByType, inputInfo);
        }).toThrow("InternalError");
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

        expect(idsByAction.Update).toEqual(["789", "790"]);
        expect(idsByType.Report).toEqual(["789", "790"]);
        expect(inputsById["789"].node.id).toEqual("789");
        expect(inputsById["789"].input).toEqual(input[field][0]);
        expect(inputsByType.Report.Update[0].node.id).toEqual("789");
        expect(inputsByType.Report.Update[0].input).toEqual(input[field][0]);
        expect(inputsById["790"].node.id).toEqual("790");
        expect(inputsById["790"].input).toEqual(input[field][1]);
        expect(inputsByType.Report.Update[1].node.id).toEqual("790");
        expect(inputsByType.Report.Update[1].input).toEqual(input[field][1]);
        expect(parentNode.children[0].id).toEqual("789");
        expect(parentNode.children[0].parent).toBe(parentNode);
        expect(parentNode.children[1].id).toEqual("790");
        expect(parentNode.children[1].parent).toBe(parentNode);
        expect(inputInfo.input[field]).toEqual(["789", "790"]);
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
        expect(() => {
            processInputObjectField(field, input, format, parentNode, null, idsByAction, idsByType, inputsById, inputsByType, inputInfo);
        }).toThrow("InternalError");
    });

    it("handles non-array connect relations with closestWithId", () => {
        const field = "projectConnect";
        const input = {
            projectConnect: "123",
        };

        processInputObjectField(field, input, format, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        // Should contain both placeholder and ID, since we may be 
        // implicitly Disconnecting the previous relation
        const expectedPlaceholder = "Routine|grandparentId.version.project";
        expect(idsByAction.Connect).toEqual(["123"]);
        expect(idsByAction.Disconnect).toEqual([expectedPlaceholder]);
        expect(idsByType.Project).toEqual([expectedPlaceholder, "123"]);
        // No input data, since we're only giving an ID
        expect(inputsById[expectedPlaceholder]).toBeUndefined();
        expect(inputsById["123"]).toBeUndefined();
        expect(inputsByType.Project.Connect).toHaveLength(0);
        expect(parentNode.children[0].id).toEqual(expectedPlaceholder);
        expect(parentNode.children[0].action).toEqual("Disconnect");
        expect(parentNode.children[0].parent).toBe(parentNode);
        expect(parentNode.children[1].id).toEqual("123");
        expect(parentNode.children[1].action).toEqual("Connect");
        expect(parentNode.children[1].parent).toBe(parentNode);
        expect(parentNode.children).toHaveLength(2);
        expect(inputInfo.input[field]).toBe("123");
    });

    it("handles non-array connect relations without closestWithId", () => {
        const field = "projectConnect";
        const input = {
            projectConnect: "123",
        };

        processInputObjectField(field, input, format, parentNode, null, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        // Should contain only ID, since we can't implicitly 
        // Disconnect within a Create mutation
        expect(idsByAction.Connect).toEqual(["123"]);
        expect(idsByType.Project).toEqual(["123"]);
        // No input data, since we're only giving an ID
        expect(inputsById["123"]).toBeUndefined();
        expect(inputsByType.Project.Connect).toHaveLength(0);
        expect(parentNode.children[0].id).toEqual("123");
        expect(parentNode.children[0].action).toEqual("Connect");
        expect(parentNode.children[0].parent).toBe(parentNode);
        expect(parentNode.children).toHaveLength(1);
        expect(inputInfo.input[field]).toBe("123");
    });

    it("handles array connect relations with closestWithId", () => {
        const field = "reportsConnect";
        const input = {
            reportsConnect: ["123", "456"],
        };

        processInputObjectField(field, input, format, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        // Should only contain IDs, since we're not implicitly Disconnecting anything
        expect(idsByAction.Connect).toEqual(["123", "456"]);
        expect(idsByType.Report).toEqual(["123", "456"]);
        // No input data, since we're only giving IDs
        expect(inputsById["123"]).toBeUndefined();
        expect(inputsById["456"]).toBeUndefined();
        expect(inputsByType.Report.Connect).toHaveLength(0);
        expect(parentNode.children[0].id).toEqual("123");
        expect(parentNode.children[0].action).toEqual("Connect");
        expect(parentNode.children[0].parent).toBe(parentNode);
        expect(parentNode.children[1].id).toEqual("456");
        expect(parentNode.children[1].action).toEqual("Connect");
        expect(parentNode.children[1].parent).toBe(parentNode);
        expect(parentNode.children).toHaveLength(2);
        expect(inputInfo.input[field]).toEqual(["123", "456"]);
    });

    it("handles array connect relations without closestWithId", () => {
        const field = "reportsConnect";
        const input = {
            reportsConnect: ["123", "456"],
        };

        processInputObjectField(field, input, format, parentNode, null, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        // Should only contain IDs, since we can't implicitly
        // Disconnect within a Create mutation
        expect(idsByAction.Connect).toEqual(["123", "456"]);
        expect(idsByType.Report).toEqual(["123", "456"]);
        // No input data, since we're only giving IDs
        expect(inputsById["123"]).toBeUndefined();
        expect(inputsById["456"]).toBeUndefined();
        expect(inputsByType.Report.Connect).toHaveLength(0);
        expect(parentNode.children[0].id).toEqual("123");
        expect(parentNode.children[0].action).toEqual("Connect");
        expect(parentNode.children[0].parent).toBe(parentNode);
        expect(parentNode.children[1].id).toEqual("456");
        expect(parentNode.children[1].action).toEqual("Connect");
        expect(parentNode.children[1].parent).toBe(parentNode);
        expect(parentNode.children).toHaveLength(2);
        expect(inputInfo.input[field]).toEqual(["123", "456"]);
    });

    it("handles non-array disconnect relations with closestWithId", () => {
        const field = "projectDisconnect";
        const input = {
            projectDisconnect: true,
        };

        processInputObjectField(field, input, format, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        // Should contain only placeholder, since we're implicitly Disconnecting
        // the previous relation, and didn't give an ID
        const expectedPlaceholder = "Routine|grandparentId.version.project";
        expect(idsByAction.Disconnect).toEqual([expectedPlaceholder]);
        expect(idsByType.Project).toEqual([expectedPlaceholder]);
        // No input data, since we're only giving a boolean
        expect(inputsById[expectedPlaceholder]).toBeUndefined();
        expect(inputsByType.Project.Disconnect).toHaveLength(0);
        expect(parentNode.children[0].id).toEqual(expectedPlaceholder);
        expect(parentNode.children[0].action).toEqual("Disconnect");
        expect(parentNode.children[0].parent).toBe(parentNode);
        expect(parentNode.children).toHaveLength(1);
        expect(inputInfo.input[field]).toBe(true);
    });

    it("handles non-array disconnect relations without closestWithId", () => {
        const field = "projectDisconnect";
        const input = {
            projectDisconnect: true,
        };

        // Should throw an error because closestWithId is null,
        // and you can't Disconnect within a Create mutation
        expect(() => {
            processInputObjectField(field, input, format, parentNode, null, idsByAction, idsByType, inputsById, inputsByType, inputInfo);
        }).toThrow("InternalError");
    });

    it("handles array disconnect relations with closestWithId", () => {
        const field = "reportsDisconnect";
        const input = {
            reportsDisconnect: ["123", "456"],
        };

        processInputObjectField(field, input, format, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        // Should only contain IDs, since we're not implicitly Disconnecting anything
        expect(idsByAction.Disconnect).toEqual(["123", "456"]);
        expect(idsByType.Report).toEqual(["123", "456"]);
        // No input data, since we're only giving IDs
        expect(inputsById["123"]).toBeUndefined();
        expect(inputsById["456"]).toBeUndefined();
        expect(inputsByType.Report.Disconnect).toHaveLength(0);
        expect(parentNode.children[0].id).toEqual("123");
        expect(parentNode.children[0].action).toEqual("Disconnect");
        expect(parentNode.children[0].parent).toBe(parentNode);
        expect(parentNode.children[1].id).toEqual("456");
        expect(parentNode.children[1].action).toEqual("Disconnect");
        expect(parentNode.children[1].parent).toBe(parentNode);
        expect(parentNode.children).toHaveLength(2);
        expect(inputInfo.input[field]).toEqual(["123", "456"]);
    });

    it("handles array disconnect relations without closestWithId", () => {
        const field = "reportsDisconnect";
        const input = {
            reportsDisconnect: ["123", "456"],
        };

        // Should throw an error because closestWithId is null,
        // and you can't Disconnect within a Create mutation
        expect(() => {
            processInputObjectField(field, input, format, parentNode, null, idsByAction, idsByType, inputsById, inputsByType, inputInfo);
        }).toThrow("InternalError");
    });

    it("handles non-array delete relations with closestWithId", () => {
        const field = "projectDelete";
        const input = {
            projectDelete: true,
        };

        processInputObjectField(field, input, format, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        // Should contain only placeholder, since we're Deleting without 
        // giving an ID
        const expectedPlaceholder = "Routine|grandparentId.version.project";
        expect(idsByAction.Delete).toEqual([expectedPlaceholder]);
        expect(idsByType.Project).toEqual([expectedPlaceholder]);
        // Deletes actually do have input data, but in this case it'll 
        // be attached to the placeholder
        expect(inputsById[expectedPlaceholder].node.id).toEqual(expectedPlaceholder);
        expect(inputsById[expectedPlaceholder].input).toEqual(expectedPlaceholder);
        expect(inputsByType.Project.Delete[0].input).toBe(expectedPlaceholder);
        expect(inputsByType.Project.Delete).toHaveLength(1);
        expect(parentNode.children[0].id).toEqual(expectedPlaceholder);
        expect(parentNode.children[0].action).toEqual("Delete");
        expect(parentNode.children[0].parent).toBe(parentNode);
        expect(parentNode.children).toHaveLength(1);
        expect(inputInfo.input[field]).toBe(true);
    });

    it("handles non-array delete relations without closestWithId", () => {
        const field = "projectDelete";
        const input = {
            projectDelete: true,
        };

        // Should throw an error because closestWithId is null,
        // and you can't Delete within a Create mutation
        expect(() => {
            processInputObjectField(field, input, format, parentNode, null, idsByAction, idsByType, inputsById, inputsByType, inputInfo);
        }).toThrow("InternalError");
    });

    it("handles array delete relations with closestWithId", () => {
        const field = "reportsDelete";
        const input = {
            reportsDelete: ["123", "456"],
        };

        processInputObjectField(field, input, format, parentNode, closestWithId, idsByAction, idsByType, inputsById, inputsByType, inputInfo);

        // Should only contain IDs, since we're not implicitly Deleting anything
        expect(idsByAction.Delete).toEqual(["123", "456"]);
        expect(idsByType.Report).toEqual(["123", "456"]);
        expect(inputsById["123"].node.id).toEqual("123");
        expect(inputsById["123"].input).toEqual(input[field][0]);
        expect(inputsById["456"].node.id).toEqual("456");
        expect(inputsById["456"].input).toEqual(input[field][1]);
        expect(inputsByType.Report.Delete[0].input).toBe(input[field][0]);
        expect(inputsByType.Report.Delete[1].input).toBe(input[field][1]);
        expect(inputsByType.Report.Delete).toHaveLength(2);
        expect(parentNode.children[0].id).toEqual("123");
        expect(parentNode.children[0].action).toEqual("Delete");
        expect(parentNode.children[0].parent).toBe(parentNode);
        expect(parentNode.children[1].id).toEqual("456");
        expect(parentNode.children[1].action).toEqual("Delete");
        expect(parentNode.children[1].parent).toBe(parentNode);
        expect(parentNode.children).toHaveLength(2);
        expect(inputInfo.input[field]).toEqual(["123", "456"]);
    });

    it("handles array delete relations without closestWithId", () => {
        const field = "reportsDelete";
        const input = {
            reportsDelete: ["123", "456"],
        };

        // Should throw an error because closestWithId is null,
        // and you can't Delete within a Create mutation
        expect(() => {
            processInputObjectField(field, input, format, parentNode, null, idsByAction, idsByType, inputsById, inputsByType, inputInfo);
        }).toThrow("InternalError");
    });
});

/**
 * Helper function to check that an object has only the expected non-empty arrays, 
 * and that every other value is an empty array.
 */
const expectOnlyTheseArrays = <ArrayItem>(
    actual: Record<string, unknown>,
    expectedNonEmpty: Record<string, ArrayItem[]>,
) => {
    // Check non-empty properties
    Object.entries(expectedNonEmpty).forEach(([key, value]) => {
        // @ts-ignore: expect-message
        expect(actual[key], `Missing array value. Key: ${key}`).toEqual(expect.arrayContaining(value));
        // @ts-ignore: expect-message
        expect(actual[key], `Has extra array values. Key: ${key}`).toHaveLength(value.length);
    });

    // Make sure that everythign else is an empty array
    Object.entries(actual).forEach(([key, value]) => {
        if (!expectedNonEmpty[key]) {
            // @ts-ignore: expect-message
            expect(value, `Should be empty array. Key: ${key}`).toEqual([]);
        }
    });
};

describe("inputToMaps", () => {
    let idsByAction, idsByType, inputsById, inputsByType, format, closestWithId;
    const initialIdsByAction = { Create: [], Update: [], Connect: [], Disconnect: [] };
    const initialIdsByType = {};
    const initialInputsById = {};
    const initialInputsByType = {};

    beforeEach(async () => {
        idsByAction = JSON.parse(JSON.stringify(initialIdsByAction));
        idsByType = JSON.parse(JSON.stringify(initialIdsByType));
        inputsById = JSON.parse(JSON.stringify(initialInputsById));
        inputsByType = JSON.parse(JSON.stringify(initialInputsByType));
        format = {
            gqlRelMap: {
                __typename: "User" as const,
                api: "Api",
                project: "Project",
                reports: "Report",
                roles: "Role",
                routine: "Routine",
                standard: "Standard",
                smartContract: "SmartContract",
            },
        };
        closestWithId = { __typename: "Routine", id: "grandparentId", path: "version" };

        // Initialize the ModelMap, which is used in fetchAndMapPlaceholder
        await ModelMap.init();
    });

    it("should initialize root node correctly for a Create action with a simple input", () => {
        const action = "Create";
        const input = { id: "1", name: "John Doe" };
        const rootNode = inputToMaps(action, input, format, "id", closestWithId, idsByAction, idsByType, inputsById, inputsByType);

        expect(rootNode.id).toBe("1");
        expect(rootNode.__typename).toBe("User");
        expect(rootNode.action).toBe("Create");
        expect(idsByAction.Create).toContain("1");
        expect(idsByType.User).toContain("1");
        expect(inputsById["1"].input).toEqual(input);
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
        expect(inputsById["1"].input).toEqual({
            ...input,
            projectCreate: "2",
        });
        expect(inputsById["2"].input).toEqual(input.projectCreate);
    });

    it("should throw error for nested Updates in a Create action, as they are invalid", () => {
        const action = "Create";
        const input = {
            id: "1",
            projectUpdate: { id: "2", name: "Project X" },
        };
        expect(() => {
            inputToMaps(action, input, format, "id", closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        }).toThrow("InternalError");
    });

    it("should throw error for nested Disconnects in a Create action, as they are invalid - test 1", () => {
        const action = "Create";
        const input = {
            id: "1",
            reportsDisconnect: ["3", "4"],
        };
        expect(() => {
            inputToMaps(action, input, format, "id", closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        }).toThrow("InternalError");
    });

    it("should throw error for nested Disconnects in a Create action, as they are invalid - test 2", () => {
        const action = "Create";
        const input = {
            id: "1",
            projectDisconnect: true,
        };
        expect(() => {
            inputToMaps(action, input, format, "id", closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        }).toThrow("InternalError");
    });

    it("should throw error for nested Deleted in a Create action, as they are invalid - test 1", () => {
        const action = "Create";
        const input = {
            id: "1",
            reportsDelete: ["3", "4"],
        };
        expect(() => {
            inputToMaps(action, input, format, "id", closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        }).toThrow("InternalError");
    });

    it("should throw error for nested Deletes in a Create action, as they are invalid - test 2", () => {
        const action = "Create";
        const input = {
            id: "1",
            projectDelete: true,
        };
        expect(() => {
            inputToMaps(action, input, format, "id", closestWithId, idsByAction, idsByType, inputsById, inputsByType);
        }).toThrow("InternalError");
    });

    it("should handle nested objects for an Update action", () => {
        const action = "Update";
        const input = {
            id: "1",
            apiDelete: true,
            projectCreate: { id: "2", name: "Project X" },
            reportsConnect: ["3", "4"],
            rolesCreate: [{ id: "5", name: "Role 1" }, { id: "6", name: "Role 2" }],
            routineUpdate: { id: "7", name: "Routine X" },
            standardConnect: "8",
            smartContractDisconnect: true,
        };
        inputToMaps(action, input, format, "id", closestWithId, idsByAction, idsByType, inputsById, inputsByType);

        expectOnlyTheseArrays(idsByAction, {
            Connect: ["3", "4", "8"],
            Create: ["2", "5", "6"],
            Update: ["1", "7"],
            Delete: ["User|1.api"],
            Disconnect: ["User|1.smartContract", "User|1.standard"], // Existing standard relation is implicitly disconnected
        });
        expectOnlyTheseArrays(idsByType, {
            User: ["1"],
            Api: ["User|1.api"],
            Project: ["2"],
            Report: ["3", "4"],
            Role: ["5", "6"],
            Routine: ["7"],
            Standard: ["8", "User|1.standard"], // Include implicit disconnect
            SmartContract: ["User|1.smartContract"],
        });
        expect(inputsById["1"].input).toEqual({
            ...input,
            apiDelete: true,
            projectCreate: "2",
            rolesCreate: ["5", "6"],
            routineUpdate: "7",
            standardConnect: "8",
            smartContractDisconnect: true,
        });
        expect(inputsById["2"].input).toEqual(input.projectCreate);
        expect(inputsById["5"].input).toEqual(input.rolesCreate[0]);
        expect(inputsById["6"].input).toEqual(input.rolesCreate[1]);
        expect(inputsById["7"].input).toEqual(input.routineUpdate);
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
        expect(inputsById["1"].input).toEqual(input);
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
        expect(inputsById["1"].input).toEqual(input);
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
        expect(inputsById["1"].input).toEqual(input);
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
            Disconnect: ["Project|2.parent"],
            Update: ["1", "2"],
        });
        expectOnlyTheseArrays(idsByType, {
            User: ["1"],
            Project: ["2"],
            ProjectVersion: ["5", "Project|2.parent"],
            Issue: ["3", "4"],
        });
        expect(inputsById["1"].input).toEqual({
            ...input,
            projectUpdate: "2",
        });
        expect(inputsById["2"].input).toEqual({
            ...input.projectUpdate,
            issuesCreate: ["3", "4"],
            parentConnect: "5",
        });
        expect(inputsById["3"].input).toEqual(input.projectUpdate.issuesCreate[0]);
        expect(inputsById["4"].input).toEqual(input.projectUpdate.issuesCreate[1]);
    });
});
