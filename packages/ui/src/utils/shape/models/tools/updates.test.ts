/* eslint-disable @typescript-eslint/ban-ts-comment */
import { DUMMY_ID, uuidValidate } from "@local/shared";
import { shapeUpdate, updateOwner, updatePrims, updateRel, updateTranslationPrims, updateVersion } from "./updates";

const mockShapeModel = {
    create: (data) => ({ ...data, shaped: "create" }),
    update: (original, updated) => ({ ...updated, shaped: "update" }),
};

describe("updateOwner function tests", () => {
    test("no owner in both original and updated items", () => {
        const originalItem = { owner: null };
        const updatedItem = { owner: null };
        const result = updateOwner(originalItem, updatedItem);
        expect(result).toEqual({});
    });

    test("same owner in both original and updated items", () => {
        const ownerData = { __typename: "User", id: "user123" };
        const originalItem = { owner: ownerData };
        const updatedItem = { owner: ownerData };
        // @ts-ignore: Testing runtime scenario
        const result = updateOwner(originalItem, updatedItem);
        expect(result).toEqual({});
    });

    test("different owners in original and updated items", () => {
        const originalItem = { owner: { __typename: "User", id: "user123" } };
        const updatedItem = { owner: { __typename: "Team", id: "team456" } };
        // @ts-ignore: Testing runtime scenario
        const result = updateOwner(originalItem, updatedItem);
        expect(result).toEqual({ teamConnect: "team456" });
    });

    test("owner present only in updated item", () => {
        const originalItem = { owner: null };
        const updatedItem = { owner: { __typename: "User", id: "user123" } };
        // @ts-ignore: Testing runtime scenario
        const result = updateOwner(originalItem, updatedItem);
        expect(result).toEqual({ userConnect: "user123" });
    });

    test("owner present only in original item", () => {
        const originalItem = { owner: { __typename: "User", id: "user123" } };
        const updatedItem = { owner: null };
        // @ts-ignore: Testing runtime scenario
        const result = updateOwner(originalItem, updatedItem);
        expect(result).toEqual({});
    });

    test("different prefixes", () => {
        const originalItem = { owner: null };
        const updatedItem = { owner: { __typename: "User", id: "user123" } };
        // @ts-ignore: Testing runtime scenario
        const result = updateOwner(originalItem, updatedItem, "managedBy");
        expect(result).toEqual({ managedByUserConnect: "user123" });
    });
});

describe("updateVersion function tests", () => {
    test("no updated versions", () => {
        const originalRoot = { id: "123", versions: [{ id: "v1", data: "version1" }] };
        const updatedRoot = { id: "123", versions: null };
        // @ts-ignore: Testing runtime scenario
        const result = updateVersion(originalRoot, updatedRoot, mockShapeModel);
        expect(result).toEqual({});
    });

    test("new versions to create", () => {
        const originalRoot = { id: "123", versions: [{ id: "v1", data: "version1" }] };
        const updatedRoot = { id: "123", versions: [{ id: "v2", data: "version2" }] };
        const result = updateVersion(originalRoot, updatedRoot, mockShapeModel);
        expect(result).toEqual({
            versionsCreate: [{ id: "v2", data: "version2", root: { id: "123" }, shaped: "create" }],
        });
    });

    test("existing versions to update", () => {
        const originalRoot = { id: "123", versions: [{ id: "v1", data: "version1" }] };
        const updatedRoot = { id: "123", versions: [{ id: "v1", data: "updated version1" }] };
        const result = updateVersion(originalRoot, updatedRoot, mockShapeModel);
        expect(result).toEqual({
            versionsUpdate: [{ id: "v1", data: "updated version1", root: { id: "123" }, shaped: "update" }],
        });
    });

    test("combination of new and updated versions", () => {
        const originalRoot = { id: "123", versions: [{ id: "v1", data: "version1" }] };
        const updatedRoot = {
            id: "123",
            versions: [
                { id: "v1", data: "updated version1" },
                { id: "v2", data: "version2" },
            ],
        };
        const result = updateVersion(originalRoot, updatedRoot, mockShapeModel);
        expect(result).toEqual({
            versionsCreate: [{ id: "v2", data: "version2", root: { id: "123" }, shaped: "create" }],
            versionsUpdate: [{ id: "v1", data: "updated version1", root: { id: "123" }, shaped: "update" }],
        });
    });

    test("no changes in versions", () => {
        const originalRoot = { id: "123", versions: [{ id: "v1", data: "version1" }] };
        const updatedRoot = { id: "123", versions: [{ id: "v1", data: "version1" }] };
        const result = updateVersion(originalRoot, updatedRoot, mockShapeModel);
        expect(result).toEqual({});
    });
});

describe("updatePrims function tests", () => {
    test("no original or updated object", () => {
        // @ts-ignore: Testing runtime scenario
        const result = updatePrims(null, null, "id", "name", "value");
        expect(result).toEqual({});
    });

    test("no original object", () => {
        const updated = { id: "123", name: "Test", value: 10 };
        const result = updatePrims(null, updated, "id", "name", "value");
        expect(result).toEqual({ id: "123", name: "Test", value: 10 });
    });

    test("no updated object", () => {
        const original = { id: "123", name: "Test", value: 10 };
        const result = updatePrims(original, null, "id", "name", "value");
        expect(result).toEqual({ id: "123" }); // Should always include the ID
    });

    test("unchanged fields", () => {
        const original = { id: "123", name: "Test", value: 10 };
        const updated = { ...original };
        const result = updatePrims(original, updated, "id", "name", "value");
        expect(result).toEqual({ id: "123" }); // Should always include the ID
    });

    test("changed fields", () => {
        const original = { id: "123", name: "Test", value: 10 };
        const updated = { ...original, name: "Updated", value: 20 };
        const result = updatePrims(original, updated, "id", "name", "value");
        expect(result).toEqual({ id: "123", name: "Updated", value: 20 });
    });

    test("primary key handling - no changes", () => {
        const original = { id: "123", name: "Test" };
        const updated = { ...original, name: "Updated" };
        // @ts-ignore: Testing runtime scenario
        const result = updatePrims(original, updated, "name"); // Here "name" is being treated as the primary key
        expect(result).toEqual({ name: "Test" }); // Since only the ID changed, the result should be empty except for the original ID
    });

    test("primary key handling - with changes", () => {
        const original = { id: "123", name: "Test", value: "boop" };
        const updated = { ...original, name: "Updated", value: "beep" };
        // @ts-ignore: Testing runtime scenario
        const result = updatePrims(original, updated, "name", "value"); // Here "name" is being treated as the primary key
        expect(result).toEqual({ name: "Test", value: "beep" }); // A field changed, so the result should have the original ID and the updated value
    });

    test("primary key is null", () => {
        const original = { id: "123", name: "Test" };
        const updated = { ...original, name: "Updated" };
        // @ts-ignore: Testing runtime scenario
        const result = updatePrims(original, updated, null, "value");
        expect(result).toEqual({}); // Can't update without a primary key
    });

    test("field with transformation function", () => {
        const original = { id: "123", value: 10 };
        const updated = { ...original, value: 20 };
        const result = updatePrims(original, updated, "id", ["value", val => val * 2]);
        expect(result).toEqual({ id: "123", value: 40 });
    });

    test("handling of DUMMY_ID", () => {
        const original = { id: DUMMY_ID, name: "Test" };
        const updated = { ...original, name: "Updated" };
        const result = updatePrims(original, updated, "id", "name");
        expect(uuidValidate(result.id)).toBe(true);
        expect(result.name).toBe("Updated");
    });

    test("primary key as id with DUMMY_ID", () => {
        const original = { id: DUMMY_ID, name: "Test" };
        const updated = { ...original, name: "Updated" };
        const result = updatePrims(original, updated, "id", "name");
        expect(uuidValidate(result.id)).toBe(true);
        expect(result.name).toBe("Updated");
    });

    test("primary key as id without DUMMY_ID", () => {
        const original = { id: "123", name: "Test" };
        const updated = { ...original, name: "Updated" };
        const result = updatePrims(original, updated, "id", "name");
        expect(result.id).toBe("123");
        expect(result.name).toBe("Updated");
    });
});

describe("updateTranslationPrims function tests", () => {
    test("no original or updated object", () => {
        // @ts-ignore: Testing runtime scenario
        const result = updateTranslationPrims(null, null, "id", "name", "value");
        expect(result).toEqual({});
    });

    test("no original object", () => {
        const updated = { id: "123", name: "Test", value: 10, language: "fr" };
        const result = updateTranslationPrims(null, updated, "id", "name", "value");
        expect(result).toEqual({ id: "123", name: "Test", value: 10, language: "fr" });
    });

    test("no updated object", () => {
        const original = { id: "123", name: "Test", value: 10, language: "fr" };
        const result = updateTranslationPrims(original, null, "id", "name", "value");
        expect(result).toEqual({});
    });

    test("unchanged fields", () => {
        const original = { id: "123", name: "Test", value: 10, language: "fr" };
        const updated = { ...original };
        const result = updateTranslationPrims(original, updated, "id", "name", "value");
        expect(result).toEqual({});
    });

    test("changed fields", () => {
        const original = { id: "123", name: "Test", value: 10, language: "fr" };
        const updated = { ...original, name: "Updated", value: 20 };
        const result = updateTranslationPrims(original, updated, "id", "name", "value");
        expect(result).toEqual({ id: "123", name: "Updated", value: 20, language: "fr" });
    });

    test("primary key handling - no changes", () => {
        const original = { id: "123", name: "Test" };
        const updated = { ...original, name: "Updated" };
        // @ts-ignore: Testing runtime scenario
        const result = updateTranslationPrims(original, updated, "name"); // Here "name" is being treated as the primary key
        expect(result).toEqual({}); // Since only the ID changed, the result should be empty
    });

    test("primary key handling - with changes", () => {
        const original = { id: "123", name: "Test", value: "boop", language: "fr" };
        const updated = { ...original, name: "Updated", value: "beep" };
        // @ts-ignore: Testing runtime scenario
        const result = updateTranslationPrims(original, updated, "name", "value"); // Here "name" is being treated as the primary key
        expect(result).toEqual({ name: "Test", value: "beep", language: "fr" }); // A field changed, so the result should have the original ID and the updated value
    });

    test("primary key is null", () => {
        const original = { id: "123", name: "Test", language: "fr" };
        const updated = { ...original, name: "Updated" };
        // @ts-ignore: Testing runtime scenario
        const result = updateTranslationPrims(original, updated, null, "value");
        expect(result).toEqual({}); // Can't update without a primary key
    });

    test("field with transformation function", () => {
        const original = { id: "123", value: 10, language: "fr" };
        const updated = { ...original, value: 20, language: "fr" };
        const result = updateTranslationPrims(original, updated, "id", ["value", val => val * 2]);
        expect(result).toEqual({ id: "123", value: 40, language: "fr" });
    });

    test("handling of DUMMY_ID", () => {
        const original = { id: DUMMY_ID, name: "Test", language: "fr" };
        const updated = { ...original, name: "Updated" };
        const result = updateTranslationPrims(original, updated, "id", "name");
        expect(uuidValidate(result.id)).toBe(true);
        expect(result.name).toBe("Updated");
    });

    test("primary key as id with DUMMY_ID", () => {
        const original = { id: DUMMY_ID, name: "Test" };
        const updated = { ...original, name: "Updated", language: "fr" };
        // @ts-ignore: Testing runtime scenario
        const result = updateTranslationPrims(original, updated, "id", "name") as any;
        expect(uuidValidate(result.id)).toBe(true);
        expect(result.name).toBe("Updated");
    });

    test("primary key as id without DUMMY_ID", () => {
        const original = { id: "123", name: "Test", language: "fr" };
        const updated = { ...original, name: "Updated", language: "fr" };
        const result = updateTranslationPrims(original, updated, "id", "name");
        expect(result.id).toBe("123");
        expect(result.name).toBe("Updated");
    });

    test("no language defaults to en", () => {
        const original = { id: "123", name: "Test" };
        const updated = { ...original, name: "Updated" };
        // @ts-ignore: Testing runtime scenario
        const result = updateTranslationPrims(original, updated, "id", "name");
        expect(result.language).toBe("en");
    });
});

describe("shapeUpdate function tests", () => {
    test("no updated object", () => {
        const result = shapeUpdate(null, {});
        expect(result).toBeUndefined();
    });

    test("shape as a function", () => {
        const updated = { name: "Test" };
        const shapeFunc = jest.fn().mockReturnValue({ name: "Updated" });
        const result = shapeUpdate(updated, shapeFunc);
        expect(shapeFunc).toHaveBeenCalled();
        expect(result).toEqual({ name: "Updated" });
    });

    test("shape as an object", () => {
        const updated = { name: "Test" };
        const shapeObj = { name: "Updated" };
        const result = shapeUpdate(updated, shapeObj);
        expect(result).toEqual({ name: "Updated" });
    });

    test("removal of undefined values", () => {
        const updated = { name: "Test", value: undefined };
        const result = shapeUpdate(updated, updated);
        expect(result).toEqual({ name: "Test" });
    });

    test("assertHasUpdate with no values updated", () => {
        const updated = { name: undefined };
        const result = shapeUpdate(updated, updated, true);
        expect(result).toBeUndefined();
    });

    test("assertHasUpdate with some values updated", () => {
        const updated = { name: "Test" };
        const result = shapeUpdate(updated, updated, true);
        expect(result).toEqual({ name: "Test" });
    });
});

describe("updateRel function tests", () => {
    test("no original item, with create operation in updated item", () => {
        const original = {};
        const updated = { relation: [{ data: "newData" }] };
        const result = updateRel(original, updated, "relation", ["Create"], "many", mockShapeModel);
        expect(result).toEqual({ relationCreate: [{ data: "newData", shaped: "create" }] }); // Treated as create, since there is no original item
    });

    test("no updated item", () => {
        const original = { relation: [{ id: "123" }] };
        const updated = {};
        const result = updateRel(original, updated, "relation", ["Connect"], "many");
        expect(result).toEqual({});
    });

    test("create operation - test 1", () => {
        const original = { relation: [{ id: "123", data: "oldData" }] };
        const updated = { relation: [{ id: "123", data: "newData" }] };
        // @ts-ignore: Testing runtime scenario
        const result = updateRel(original, updated, "relation", ["Create"], "many", mockShapeModel);
        expect(result).toEqual({}); // Data is different, but it's the same ID that appears in the original item, so it's treated as an update. Since updates aren't allowed, the result is empty
    });

    test("create operation - test 2", () => {
        const original = { relation: [{ id: "123" }] };
        const updated = { relation: [{ id: "456", data: "newData" }] };
        const result = updateRel(original, updated, "relation", ["Create"], "many", mockShapeModel);
        expect(result).toEqual({ relationCreate: [{ id: "456", data: "newData", shaped: "create" }] });
    });

    test("connect operation - test 1", () => {
        const original = { relation: [{ id: "456", data: "hello" }] };
        const updated = { relation: [{ id: "456", data: "hello" }] };
        const result = updateRel(original, updated, "relation", ["Connect"], "many");
        expect(result).toEqual({}); // Data appears in both original and updated items, so it's not a connect
    });

    test("connect operation - test 2", () => {
        const original = { relation: [{ id: "123" }] };
        const updated = { relation: [{ id: "456" }] };
        const result = updateRel(original, updated, "relation", ["Connect"], "many");
        expect(result).toEqual({ relationConnect: ["456"] });
    });

    test("disconnect operation", () => {
        const original = { relation: [{ id: "123" }] };
        const updated = { relation: [] };
        // @ts-ignore: Testing runtime scenario
        const result = updateRel(original, updated, "relation", ["Disconnect"], "many");
        expect(result).toEqual({ relationDisconnect: ["123"] });
    });

    test("delete operation", () => {
        const original = { relation: [{ id: "123" }] };
        const updated = { relation: [] };
        // @ts-ignore: Testing runtime scenario
        const result = updateRel(original, updated, "relation", ["Delete"], "many");
        expect(result).toEqual({ relationDelete: ["123"] });
    });

    test("update operation - test 1", () => {
        const original = { relation: [{ id: "123" }] };
        const updated = { relation: [{ id: "123", data: "newData" }] };
        const result = updateRel(original, updated, "relation", ["Update"], "many", mockShapeModel);
        expect(result).toEqual({ relationUpdate: [{ id: "123", data: "newData", shaped: "update" }] });
    });

    test("update operation - test 2", () => {
        const original = { relation: [{ id: "123", data: "oldData" }] };
        const updated = { relation: [{ id: "123", data: "newData" }] };
        const result = updateRel(original, updated, "relation", ["Update"], "many", mockShapeModel);
        expect(result).toEqual({ relationUpdate: [{ id: "123", data: "newData", shaped: "update" }] });
    });

    test("create and connect operations", () => {
        const original = { relation: [{ id: "123" }] };
        const updated = {
            relation: [
                { id: "456" }, // Should be Connect
                { id: "123", data: "newData" }, // Should be update, so it's ignored
                { id: "999", data: "newData" }, // Should be Create
                { id: "420", __connect: true, data: "boop" }, // Should be Connect
            ],
        };
        const result = updateRel(original, updated, "relation", ["Create", "Connect"], "many", mockShapeModel);
        expect(result).toEqual({
            relationCreate: [{ id: "999", data: "newData", shaped: "create" }],
            relationConnect: ["456", "420"],
        });
    });

    test("create and disconnect operations", () => {
        const original = { relation: [{ id: "123" }, { id: "456" }] };
        const updated = {
            relation: [
                { id: "456" }, // Unchanged, so it's ignored
                { id: "999", data: "newData" }, // Should be Create
                // "123" is missing, so it should be Disconnect
            ],
        };
        // @ts-ignore: Testing runtime scenario
        const result = updateRel(original, updated, "relation", ["Create", "Disconnect"], "many", mockShapeModel);
        expect(result).toEqual({
            relationCreate: [{ id: "999", data: "newData", shaped: "create" }],
            relationDisconnect: ["123"],
        });
    });

    test("create and delete operations", () => {
        const original = { relation: [{ id: "123" }, { id: "456" }] };
        const updated = {
            relation: [
                { id: "456" }, // Unchanged, so it's ignored
                { id: "999", data: "newData" }, // Should be Create
                // "123" is missing, so it should be Delete
            ],
        };
        // @ts-ignore: Testing runtime scenario
        const result = updateRel(original, updated, "relation", ["Create", "Delete"], "many", mockShapeModel);
        expect(result).toEqual({
            relationCreate: [{ id: "999", data: "newData", shaped: "create" }],
            relationDelete: ["123"],
        });
    });

    test("connect and disconnect operations", () => {
        const original = { relation: [{ id: "123" }, { id: "456" }] };
        const updated = {
            relation: [
                { id: "456" }, // Unchanged, so it's ignored
                { id: "789", beep: "Boop" }, // Should be Connect, even though it has other data like a Create. But Create is not an option, so it falls back to Connect
                // "123" is missing, so it should be Disconnect
            ],
        };
        const result = updateRel(original, updated, "relation", ["Connect", "Disconnect"], "many");
        expect(result).toEqual({
            relationConnect: ["789"],
            relationDisconnect: ["123"],
        });
    });

    test("one-to-one create operation - test 1", () => {
        const original = { relation: { id: "123" } };
        const updated = { relation: { id: "456", data: "newData" } }; // Should be Create, since it has a different ID
        // @ts-ignore: Testing runtime scenario
        const result = updateRel(original, updated, "relation", ["Create"], "one", mockShapeModel);
        expect(result).toEqual({ relationCreate: { id: "456", data: "newData", shaped: "create" } });
    });

    test("one-to-one create operation - test 2", () => {
        const original = {};
        const updated = { relation: { id: "456", data: "newData" } }; // Should be Create, since relation didn't exist in original
        // @ts-ignore: Testing runtime scenario
        const result = updateRel(original, updated, "relation", ["Create"], "one", mockShapeModel);
        expect(result).toEqual({ relationCreate: { id: "456", data: "newData", shaped: "create" } });
    });

    test("one-to-one create operation - test 3", () => {
        const original = { relation: { id: "123" } };
        const updated = { relation: { id: "123", data: "newData" } }; // Should be Update (which should be ignored in this test), since it has the same ID
        // @ts-ignore: Testing runtime scenario
        const result = updateRel(original, updated, "relation", ["Create"], "one", mockShapeModel);
        expect(result).toEqual({});
    });

    test("one-to-one connect operation - test 1", () => {
        const original = { relation: { id: "123" } };
        const updated = { relation: { id: "456" } };
        const result = updateRel(original, updated, "relation", ["Connect"], "one");
        expect(result).toEqual({ relationConnect: "456" });
    });

    test("one-to-one connect operation - test 2", () => {
        const original = { relation: { id: "123" } };
        const updated = { relation: { id: "123" } }; // Should be ignored, since it has the same ID as the original
        const result = updateRel(original, updated, "relation", ["Connect"], "one");
        expect(result).toEqual({});
    });

    test("one-to-one disconnect operation - test 1", () => {
        const original = { relation: { id: "123" } };
        const updated = { relation: null }; // Null indicates a Disconnect
        // @ts-ignore: Testing runtime scenario
        const result = updateRel(original, updated, "relation", ["Disconnect"], "one");
        expect(result).toEqual({ relationDisconnect: true }); // One-to-one disconnects use boolean values instead of IDs, since the ID is already known
    });

    test("one-to-one disconnect operation - test 2", () => {
        const original = { relation: { id: "123" } };
        const updated = { relation: { id: "456" } }; // Should be able to determine that '123' should be disconnected
        const result = updateRel(original, updated, "relation", ["Disconnect"], "one");
        expect(result).toEqual({ relationDisconnect: true }); // One-to-one disconnects use boolean values instead of IDs, since the ID is already known
    });

    test("one-to-one disconnect operation - test 3", () => {
        const original = { relation: { id: "123" } };
        const updated = { relation: { id: "123", data: "newData" } }; // Should be ignored, since it has the same ID as the original
        const result = updateRel(original, updated, "relation", ["Disconnect"], "one");
        expect(result).toEqual({});
    });

    test("one-to-one disconnect operation - test 4", () => {
        const original = { relation: { id: "123" } };
        const updated = { relation: undefined }; // Should be ignored, since only null is treated as a disconnect
        // @ts-ignore: Testing runtime scenario
        const result = updateRel(original, updated, "relation", ["Disconnect"], "one");
        expect(result).toEqual({});
    });

    test("one-to-one delete operation - test 1", () => {
        const original = { relation: { id: "123" } };
        const updated = { relation: null }; // Null indicates a Delete
        // @ts-ignore: Testing runtime scenario
        const result = updateRel(original, updated, "relation", ["Delete"], "one");
        expect(result).toEqual({ relationDelete: true }); // One-to-one deletes use boolean values instead of IDs, since the ID is already known
    });

    test("one-to-one delete operation - test 2", () => {
        const original = { relation: { id: "123" } };
        const updated = { relation: { id: "456" } }; // Should be able to determine that '123' should be deleted
        const result = updateRel(original, updated, "relation", ["Delete"], "one");
        expect(result).toEqual({ relationDelete: true }); // One-to-one deletes use boolean values instead of IDs, since the ID is already known
    });

    test("one-to-one delete operation - test 3", () => {
        const original = { relation: { id: "123" } };
        const updated = { relation: { id: "123", data: "newData" } }; // Should be ignored, since it has the same ID as the original
        const result = updateRel(original, updated, "relation", ["Delete"], "one");
        expect(result).toEqual({});
    });

    test("one-to-one delete operation - test 4", () => {
        const original = { relation: { id: "123" } };
        const updated = { relation: undefined }; // Should be ignored, since only null is treated as a delete
        // @ts-ignore: Testing runtime scenario
        const result = updateRel(original, updated, "relation", ["Delete"], "one");
        expect(result).toEqual({});
    });

    test("one-to-one update operation - test 1", () => {
        const original = { relation: { id: "123" } };
        const updated = { relation: { id: "123", data: "newData" } };
        const result = updateRel(original, updated, "relation", ["Update"], "one", mockShapeModel);
        expect(result).toEqual({ relationUpdate: { id: "123", data: "newData", shaped: "update" } });
    });

    test("one-to-one update operation - test 2", () => {
        const original = { relation: { id: "123", data: "booop", other: { random: { data: "hi" } } } };
        const result = updateRel(original, { ...original }, "relation", ["Update"], "one", mockShapeModel);
        expect(result).toEqual({});
    });

    test("one-to-one create and connect operations - test 1", () => {
        const original = { relation: { id: "123" } };
        const updated = { relation: { id: "456", data: "newData" } }; // Should be Create, since it has more data than just an ID
        const result = updateRel(original, updated, "relation", ["Create", "Connect"], "one", mockShapeModel);
        expect(result).toEqual({
            relationCreate: { id: "456", data: "newData", shaped: "create" },
        });
    });

    test("one-to-one create and connect operations - test 2", () => {
        const original = { relation: { id: "123" } };
        const updated = { relation: { __typename: "Boop", id: "456" } }; // Should be Connect, since it has only an ID and a __typename
        const result = updateRel(original, updated, "relation", ["Create", "Connect"], "one", mockShapeModel);
        expect(result).toEqual({
            relationConnect: "456",
        });
    });

    test("one-to-one connect and disconnect operations", () => {
        const original = { relation: { id: "123" } };
        const updated = { relation: { id: "456" } };
        const result = updateRel(original, updated, "relation", ["Connect", "Disconnect"], "one");
        expect(result).toEqual({
            relationConnect: "456", // The disconnect is implicitly handled by the connect, since it's a one-to-one
        });
    });

    test("create with preShape function", () => {
        const original = { relation: [{ id: "123" }] };
        const updated = { relation: [{ id: "456", data: "newData" }] };
        const preShape = (d) => ({ ...d, preShape: "yeet" });
        const result = updateRel(original, updated, "relation", ["Create"], "many", mockShapeModel, preShape);
        expect(result).toEqual({ relationCreate: [{ id: "456", data: "newData", preShape: "yeet", shaped: "create" }] }); // Reflects changes from both preShape and shape (mockShapeModel)
    });

    test("update with preShape function", () => {
        const original = { relation: [{ id: "123" }] };
        const updated = { relation: [{ id: "123", data: "newData" }] };
        const preShape = (d) => ({ ...d, preShape: "yeet" });
        const result = updateRel(original, updated, "relation", ["Update"], "many", mockShapeModel, preShape);
        expect(result).toEqual({ relationUpdate: [{ id: "123", data: "newData", preShape: "yeet", shaped: "update" }] }); // Reflects changes from both preShape and shape (mockShapeModel)
    });
});

