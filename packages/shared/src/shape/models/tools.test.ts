/* eslint-disable @typescript-eslint/ban-ts-comment */
import { DUMMY_ID, uuidValidate } from "../../id/uuid";
import { createOwner, createPrims, createRel, createVersion, shapeDate, shapeUpdate, shouldConnect, updateOwner, updatePrims, updateRel, updateTranslationPrims, updateVersion } from "./tools";

const mockShapeCreateModel = {
    create: (data) => ({ ...data, shaped: true }), // We add a `shaped` property to the data to test that the function is called
};

describe("createOwner function tests", () => {
    test("item with User owner", () => {
        const item = { owner: { __typename: "User", id: "user123" } };
        // @ts-ignore: Testing runtime scenario
        const result = createOwner(item);
        expect(result).toEqual({ userConnect: "user123" });
    });

    test("item with Team owner", () => {
        const item = { owner: { __typename: "Team", id: "team123" } };
        // @ts-ignore: Testing runtime scenario
        const result = createOwner(item);
        expect(result).toEqual({ teamConnect: "team123" });
    });

    test("item with different prefixes", () => {
        const item = { owner: { __typename: "User", id: "user123" } };
        // @ts-ignore: Testing runtime scenario
        const result = createOwner(item, "ownedBy");
        expect(result).toEqual({ ownedByUserConnect: "user123" });
    });

    test("item with null owner", () => {
        const item = { owner: null };
        const result = createOwner(item);
        expect(result).toEqual({});
    });

    test("item with undefined owner", () => {
        const item = { owner: undefined };
        const result = createOwner(item);
        expect(result).toEqual({});
    });

    test("item with unexpected owner type", () => {
        const item = { owner: { __typename: "OtherType", id: "other123" } };
        // @ts-ignore: Testing runtime scenario
        const result = createOwner(item);
        expect(result).toEqual({});
    });

    test("item with empty prefix", () => {
        const item = { owner: { __typename: "User", id: "user123" } };
        // @ts-ignore: Testing runtime scenario
        const result = createOwner(item, "");
        expect(result).toEqual({ userConnect: "user123" });
    });

    test("field name formatting with prefix", () => {
        const item = { owner: { __typename: "User", id: "user123" } };
        // @ts-ignore: Testing runtime scenario
        const result = createOwner(item, "OwnedBy");
        expect(result).toHaveProperty("ownedByUserConnect");
    });
});

describe("createVersion function tests", () => {

    test("root object with version data", () => {
        const root = {
            id: "123",
            versions: [{ data: "version1" }, { data: "version2" }],
        };
        const result = createVersion(root, mockShapeCreateModel);
        expect(result).toEqual({
            versionsCreate: [
                { data: "version1", root: { id: "123" }, shaped: true },
                { data: "version2", root: { id: "123" }, shaped: true },
            ],
        });
    });

    test("root object with empty versions array", () => {
        const root = { id: "123", versions: [] };
        const result = createVersion(root, mockShapeCreateModel);
        expect(result).toEqual({});
    });

    test("root object with null versions", () => {
        const root = { id: "123", versions: null };
        const result = createVersion(root, mockShapeCreateModel);
        expect(result).toEqual({});
    });

    test("root object with undefined versions", () => {
        const root = { id: "123", versions: undefined };
        const result = createVersion(root, mockShapeCreateModel);
        expect(result).toEqual({});
    });

    test("shape model modifies version data", () => {
        const root = {
            id: "123",
            versions: [{ data: "version1" }],
        };
        const result = createVersion(root, mockShapeCreateModel);
        expect(result).toEqual({
            versionsCreate: [{ data: "version1", root: { id: "123" }, shaped: true }],
        });
    });

    test("root object without id field", () => {
        const root = { versions: [{ data: "version1" }] };
        // @ts-ignore: Testing runtime scenario
        const result = createVersion(root, mockShapeCreateModel);
        expect(result).toEqual({
            versionsCreate: [{ data: "version1", root: { id: undefined }, shaped: true }],
        });
    });

    test("root object with additional fields", () => {
        const root = {
            id: "123",
            versions: [{ data: "version1" }],
            extraField: "extra",
        };
        const result = createVersion(root, mockShapeCreateModel);
        expect(result).toEqual({
            versionsCreate: [{ data: "version1", root: { id: "123" }, shaped: true }],
        });
    });
});

describe("createPrims function tests", () => {
    beforeAll(() => {
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        jest.spyOn(console, "error").mockImplementation(() => { });
    });
    afterAll(() => {
        jest.restoreAllMocks();
    });

    test("simple object with specified fields", () => {
        const obj = { id: "123", name: "Test", value: null };
        const result = createPrims(obj, "id", "name");
        expect(result).toEqual({ id: "123", name: "Test" });
    });

    test("field with null value", () => {
        const obj = { id: "123", name: null };
        const result = createPrims(obj, "id", "name");
        expect(result).toEqual({ id: "123", name: null });
    });

    test("field with transformation function", () => {
        const obj = { id: "123", value: 10 };
        const result = createPrims(obj, ["value", val => val * 2]);
        expect(result).toEqual({ value: 20 });
    });

    test("empty object with no fields", () => {
        // @ts-ignore: Testing runtime scenario
        const result = createPrims({}, "id", "name");
        expect(result).toEqual({});
    });

    test("fields not present in the object", () => {
        const obj = { id: "123" };
        // @ts-ignore: Testing runtime scenario
        const result = createPrims(obj, "name");
        expect(result).toEqual({});
    });

    test("transformation function returning null", () => {
        const obj = { id: "123", value: 10 };
        const result = createPrims(obj, ["value", () => null]);
        expect(result).toEqual({ value: null });
    });

    test("handling of DUMMY_ID", () => {
        const obj = { id: DUMMY_ID, name: "Test" };
        const result = createPrims(obj, "id", "name");
        expect(result.id).not.toBe(DUMMY_ID);
        expect(uuidValidate(result.id)).toBe(true); // Assuming uuid.validate() is a method to validate UUIDs
        expect(result.name).toBe("Test");
    });

    test("mix of fields with and without transformation functions", () => {
        const obj = { id: "123", name: "Test", value: 10 };
        const result = createPrims(obj, "id", ["value", val => val * 2]);
        expect(result).toEqual({ id: "123", value: 20 });
    });

    test("non-object input should return empty object and log error", () => {
        // @ts-ignore: Testing runtime scenario
        expect(createPrims(null, "id")).toEqual({});
        // @ts-ignore: Testing runtime scenario
        expect(createPrims(undefined, "id")).toEqual({});
        // @ts-ignore: Testing runtime scenario
        expect(createPrims(123, "id")).toEqual({});
    });
});

describe("shouldConnect function tests", () => {
    const validCases = [
        { case: { id: "123", __typename: "Type" }, description: "only id and __typename" },
        { case: { __connect: true, someOtherProp: "value" }, description: "__connect: true" },
        { case: {}, description: "empty object" },
    ];

    const invalidCases = [
        { case: { id: "123", __typename: "Type", name: "Test" }, description: "extra fields" },
        { case: { id: undefined, __typename: "Type" }, description: "undefined id" },
        { case: { id: "123", __typename: undefined }, description: "undefined __typename" },
        { case: { id: "123", data: "otherData" }, description: "extra field and no __typename" },
        { case: { __connect: false }, description: "__connect not true" },
        { case: { __connect: "yes" }, description: "__connect non-boolean true" },
        { case: { id: "123", __typename: "Type", nested: { key: "value" } }, description: "nested structures" },
        { case: { id: DUMMY_ID }, description: "DUMMY_ID indicates the object is new, so it should be created instead of connected" },
        { case: "string", description: "non-object (string)" },
        { case: 123, description: "non-object (number)" },
        { case: [1, 2, 3], description: "non-object (array)" },
        { case: null, description: "null" },
        { case: undefined, description: "undefined" },
    ];

    validCases.forEach(({ case: testCase, description }) => {
        test(`should return true for ${description}`, () => {
            expect(shouldConnect(testCase)).toBe(true);
        });
    });

    invalidCases.forEach(({ case: testCase, description }) => {
        test(`should return false for ${description}`, () => {
            // @ts-ignore: Testing runtime scenario
            expect(shouldConnect(testCase)).toBe(false);
        });
    });
});

describe("createRel function tests", () => {
    test("one-to-one relationship with Connect operation", () => {
        const item = { relation: { id: "123" } };
        const result = createRel(item, "relation", ["Connect"], "one");
        expect(result).toEqual({ relationConnect: "123" });
    });

    test("one-to-many relationship with Connect operation", () => {
        const item = { relation: [{ id: "123" }, { id: "456" }] };
        const result = createRel(item, "relation", ["Connect"], "many");
        expect(result).toEqual({ relationConnect: ["123", "456"] });
    });

    test("one-to-one relationship with Create operation", () => {
        const item = { relation: { data: "data" } };
        const result = createRel(item, "relation", ["Create"], "one", mockShapeCreateModel);
        expect(result).toEqual({ relationCreate: { data: "data", shaped: true } });
    });

    test("one-to-many relationship with Create operation", () => {
        const item = { relation: [{ data: "data1" }, { data: "data2" }] };
        const result = createRel(item, "relation", ["Create"], "many", mockShapeCreateModel);
        expect(result).toEqual({ relationCreate: [{ data: "data1", shaped: true }, { data: "data2", shaped: true }] });
    });

    test("null relationship data", () => {
        const item = { relation: null };
        const result = createRel(item, "relation", ["Connect"], "one");
        expect(result).toEqual({});
    });

    test("undefined relationship data", () => {
        const item = { relation: undefined };
        const result = createRel(item, "relation", ["Connect"], "one");
        expect(result).toEqual({});
    });

    test("empty array for one-to-many relationship", () => {
        const item = { relation: [] };
        const result = createRel(item, "relation", ["Connect"], "many");
        expect(result).toEqual({});
    });

    test("missing shape model for Create operation", () => {
        const item = { relation: { data: "data" } };
        expect(() => {
            createRel(item, "relation", ["Create"], "one");
        }).toThrowError("Model is required if relTypes includes \"Create\": relation");
    });

    test("with preShape function", () => {
        const item = { relation: { data: "data" } };
        const preShape = jest.fn(data => ({ ...data, preShaped: true }));
        const result = createRel(item, "relation", ["Create"], "one", mockShapeCreateModel, preShape);
        expect(result).toEqual({ relationCreate: { data: "data", preShaped: true, shaped: true } });
    });

    test("filter items based on shouldConnect for Connect operation - id test 1", () => {
        const item = { relation: [{ id: "123" }, { data: "data" }] };
        const result = createRel(item, "relation", ["Connect", "Create"], "many", mockShapeCreateModel);
        expect(result).toEqual({ relationConnect: ["123"], relationCreate: [{ data: "data", shaped: true }] });
    });

    test("filter items based on shouldConnect for Connect operation - id test 2", () => {
        const item = { relation: [{ __typename: "Boop", id: "123" }, { id: "456", data: "data" }] };
        const result = createRel(item, "relation", ["Connect", "Create"], "many", mockShapeCreateModel);
        expect(result).toEqual({ relationConnect: ["123"], relationCreate: [{ id: "456", data: "data", shaped: true }] });
    });

    test("filter items based on shouldConnect for Connect operation - __connect test", () => {
        const item = { relation: [{ id: "123", data: "dataa", __connect: true }, { id: "456", data: "data" }] };
        const result = createRel(item, "relation", ["Connect", "Create"], "many", mockShapeCreateModel);
        expect(result).toEqual({ relationConnect: ["123"], relationCreate: [{ id: "456", data: "data", shaped: true }] });
    });
});

const mockShapeUpdateModel = {
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
        const result = updateVersion(originalRoot, updatedRoot, mockShapeUpdateModel);
        expect(result).toEqual({});
    });

    test("new versions to create", () => {
        const originalRoot = { id: "123", versions: [{ id: "v1", data: "version1" }] };
        const updatedRoot = { id: "123", versions: [{ id: "v2", data: "version2" }] };
        const result = updateVersion(originalRoot, updatedRoot, mockShapeUpdateModel);
        expect(result).toEqual({
            versionsCreate: [{ id: "v2", data: "version2", root: { id: "123" }, shaped: "create" }],
        });
    });

    test("existing versions to update", () => {
        const originalRoot = { id: "123", versions: [{ id: "v1", data: "version1" }] };
        const updatedRoot = { id: "123", versions: [{ id: "v1", data: "updated version1" }] };
        const result = updateVersion(originalRoot, updatedRoot, mockShapeUpdateModel);
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
        const result = updateVersion(originalRoot, updatedRoot, mockShapeUpdateModel);
        expect(result).toEqual({
            versionsCreate: [{ id: "v2", data: "version2", root: { id: "123" }, shaped: "create" }],
            versionsUpdate: [{ id: "v1", data: "updated version1", root: { id: "123" }, shaped: "update" }],
        });
    });

    test("no changes in versions", () => {
        const originalRoot = { id: "123", versions: [{ id: "v1", data: "version1" }] };
        const updatedRoot = { id: "123", versions: [{ id: "v1", data: "version1" }] };
        const result = updateVersion(originalRoot, updatedRoot, mockShapeUpdateModel);
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
});

describe("updateRel function tests", () => {
    test("no original item, with create operation in updated item", () => {
        const original = {};
        const updated = { relation: [{ data: "newData" }] };
        const result = updateRel(original, updated, "relation", ["Create"], "many", mockShapeUpdateModel);
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
        const result = updateRel(original, updated, "relation", ["Create"], "many", mockShapeUpdateModel);
        expect(result).toEqual({}); // Data is different, but it's the same ID that appears in the original item, so it's treated as an update. Since updates aren't allowed, the result is empty
    });

    test("create operation - test 2", () => {
        const original = { relation: [{ id: "123" }] };
        const updated = { relation: [{ id: "456", data: "newData" }] };
        const result = updateRel(original, updated, "relation", ["Create"], "many", mockShapeUpdateModel);
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
        const result = updateRel(original, updated, "relation", ["Update"], "many", mockShapeUpdateModel);
        expect(result).toEqual({ relationUpdate: [{ id: "123", data: "newData", shaped: "update" }] });
    });

    test("update operation - test 2", () => {
        const original = { relation: [{ id: "123", data: "oldData" }] };
        const updated = { relation: [{ id: "123", data: "newData" }] };
        const result = updateRel(original, updated, "relation", ["Update"], "many", mockShapeUpdateModel);
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
        const result = updateRel(original, updated, "relation", ["Create", "Connect"], "many", mockShapeUpdateModel);
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
        const result = updateRel(original, updated, "relation", ["Create", "Disconnect"], "many", mockShapeUpdateModel);
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
        const result = updateRel(original, updated, "relation", ["Create", "Delete"], "many", mockShapeUpdateModel);
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
        const result = updateRel(original, updated, "relation", ["Create"], "one", mockShapeUpdateModel);
        expect(result).toEqual({ relationCreate: { id: "456", data: "newData", shaped: "create" } });
    });

    test("one-to-one create operation - test 2", () => {
        const original = {};
        const updated = { relation: { id: "456", data: "newData" } }; // Should be Create, since relation didn't exist in original
        // @ts-ignore: Testing runtime scenario
        const result = updateRel(original, updated, "relation", ["Create"], "one", mockShapeUpdateModel);
        expect(result).toEqual({ relationCreate: { id: "456", data: "newData", shaped: "create" } });
    });

    test("one-to-one create operation - test 3", () => {
        const original = { relation: { id: "123" } };
        const updated = { relation: { id: "123", data: "newData" } }; // Should be Update (which should be ignored in this test), since it has the same ID
        // @ts-ignore: Testing runtime scenario
        const result = updateRel(original, updated, "relation", ["Create"], "one", mockShapeUpdateModel);
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
        const result = updateRel(original, updated, "relation", ["Update"], "one", mockShapeUpdateModel);
        expect(result).toEqual({ relationUpdate: { id: "123", data: "newData", shaped: "update" } });
    });

    test("one-to-one update operation - test 2", () => {
        const original = { relation: { id: "123", data: "booop", other: { random: { data: "hi" } } } };
        const result = updateRel(original, { ...original }, "relation", ["Update"], "one", mockShapeUpdateModel);
        expect(result).toEqual({});
    });

    test("one-to-one create and connect operations - test 1", () => {
        const original = { relation: { id: "123" } };
        const updated = { relation: { id: "456", data: "newData" } }; // Should be Create, since it has more data than just an ID
        const result = updateRel(original, updated, "relation", ["Create", "Connect"], "one", mockShapeUpdateModel);
        expect(result).toEqual({
            relationCreate: { id: "456", data: "newData", shaped: "create" },
        });
    });

    test("one-to-one create and connect operations - test 2", () => {
        const original = { relation: { id: "123" } };
        const updated = { relation: { __typename: "Boop", id: "456" } }; // Should be Connect, since it has only an ID and a __typename
        const result = updateRel(original, updated, "relation", ["Create", "Connect"], "one", mockShapeUpdateModel);
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
        function preShape(d) {
            return { ...d, preShape: "bloop" };
        }
        const result = updateRel(original, updated, "relation", ["Create"], "many", mockShapeUpdateModel, preShape);
        expect(result).toEqual({ relationCreate: [{ id: "456", data: "newData", preShape: "bloop", shaped: "create" }] }); // Reflects changes from both preShape and shape (mockShapeUpdateModel)
    });

    test("update with preShape function", () => {
        const original = { relation: [{ id: "123" }] };
        const updated = { relation: [{ id: "123", data: "newData" }] };
        function preShape(d) {
            return { ...d, preShape: "bloop" };
        }
        const result = updateRel(original, updated, "relation", ["Update"], "many", mockShapeUpdateModel, preShape);
        expect(result).toEqual({ relationUpdate: [{ id: "123", data: "newData", preShape: "bloop", shaped: "update" }] }); // Reflects changes from both preShape and shape (mockShapeUpdateModel)
    });
});

describe("shapeDate function tests", () => {
    test("valid date within default range", () => {
        const result = shapeDate("2025-06-15");
        expect(result).toBe("2025-06-15T00:00:00.000Z");
    });

    test("valid date at minimum boundary", () => {
        const result = shapeDate("2023-01-01");
        expect(result).toBe("2023-01-01T00:00:00.000Z");
    });

    test("valid date at maximum boundary", () => {
        const result = shapeDate("2099-12-31");
        expect(result).toBe("2099-12-31T00:00:00.000Z");
    });

    test("invalid date string", () => {
        const result = shapeDate("invalid-date");
        expect(result).toBeNull();
    });

    test("non-string parameters", () => {
        // @ts-ignore: Testing runtime scenario
        const resultNumber = shapeDate(123);
        expect(resultNumber).toBeNull();
        // @ts-ignore: Testing runtime scenario
        const resultBoolean = shapeDate(true);
        expect(resultBoolean).toBeNull();
        // @ts-ignore: Testing runtime scenario
        const resultObject = shapeDate({});
        expect(resultObject).toBeNull();
        // @ts-ignore: Testing runtime scenario
        const resultArray = shapeDate([]);
        expect(resultArray).toBeNull();
        // @ts-ignore: Testing runtime scenario
        const resultNull = shapeDate(null);
        expect(resultNull).toBeNull();
        // @ts-ignore: Testing runtime scenario
        const resultUndefined = shapeDate(undefined);
        expect(resultUndefined).toBeNull();
    });

    test("date before minimum date", () => {
        const result = shapeDate("2022-12-31");
        expect(result).toBeNull();
    });

    test("date after maximum date", () => {
        const result = shapeDate("2100-01-02");
        expect(result).toBeNull();
    });

    test("valid date with custom min and max dates", () => {
        const minDate = new Date("2020-01-01");
        const maxDate = new Date("2024-01-01");
        const result = shapeDate("2022-06-15", minDate, maxDate);
        expect(result).toBe("2022-06-15T00:00:00.000Z");
    });

    test("date at custom minimum boundary", () => {
        const minDate = new Date("2020-01-01");
        const maxDate = new Date("2024-01-01");
        const result = shapeDate("2020-01-01", minDate, maxDate);
        expect(result).toBe("2020-01-01T00:00:00.000Z");
    });

    test("date at custom maximum boundary", () => {
        const minDate = new Date("2020-01-01");
        const maxDate = new Date("2024-01-01");
        const result = shapeDate("2024-01-01", minDate, maxDate);
        expect(result).toBe("2024-01-01T00:00:00.000Z");
    });

    test("date string format check", () => {
        const result = shapeDate("2025-06-15");
        expect(result).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d{3}Z$/);
    });
});
