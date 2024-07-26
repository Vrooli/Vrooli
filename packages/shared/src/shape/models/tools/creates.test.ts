/* eslint-disable @typescript-eslint/ban-ts-comment */
import { DUMMY_ID, uuidValidate } from "../../../id/uuid";
import { createOwner, createPrims, createRel, createVersion, shouldConnect } from "./creates";

const mockShapeModel = {
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
        const result = createVersion(root, mockShapeModel);
        expect(result).toEqual({
            versionsCreate: [
                { data: "version1", root: { id: "123" }, shaped: true },
                { data: "version2", root: { id: "123" }, shaped: true },
            ],
        });
    });

    test("root object with empty versions array", () => {
        const root = { id: "123", versions: [] };
        const result = createVersion(root, mockShapeModel);
        expect(result).toEqual({});
    });

    test("root object with null versions", () => {
        const root = { id: "123", versions: null };
        const result = createVersion(root, mockShapeModel);
        expect(result).toEqual({});
    });

    test("root object with undefined versions", () => {
        const root = { id: "123", versions: undefined };
        const result = createVersion(root, mockShapeModel);
        expect(result).toEqual({});
    });

    test("shape model modifies version data", () => {
        const root = {
            id: "123",
            versions: [{ data: "version1" }],
        };
        const result = createVersion(root, mockShapeModel);
        expect(result).toEqual({
            versionsCreate: [{ data: "version1", root: { id: "123" }, shaped: true }],
        });
    });

    test("root object without id field", () => {
        const root = { versions: [{ data: "version1" }] };
        // @ts-ignore: Testing runtime scenario
        const result = createVersion(root, mockShapeModel);
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
        const result = createVersion(root, mockShapeModel);
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
        const result = createRel(item, "relation", ["Create"], "one", mockShapeModel);
        expect(result).toEqual({ relationCreate: { data: "data", shaped: true } });
    });

    test("one-to-many relationship with Create operation", () => {
        const item = { relation: [{ data: "data1" }, { data: "data2" }] };
        const result = createRel(item, "relation", ["Create"], "many", mockShapeModel);
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
        const result = createRel(item, "relation", ["Create"], "one", mockShapeModel, preShape);
        expect(result).toEqual({ relationCreate: { data: "data", preShaped: true, shaped: true } });
    });

    test("filter items based on shouldConnect for Connect operation - id test 1", () => {
        const item = { relation: [{ id: "123" }, { data: "data" }] };
        const result = createRel(item, "relation", ["Connect", "Create"], "many", mockShapeModel);
        expect(result).toEqual({ relationConnect: ["123"], relationCreate: [{ data: "data", shaped: true }] });
    });

    test("filter items based on shouldConnect for Connect operation - id test 2", () => {
        const item = { relation: [{ __typename: "Boop", id: "123" }, { id: "456", data: "data" }] };
        const result = createRel(item, "relation", ["Connect", "Create"], "many", mockShapeModel);
        expect(result).toEqual({ relationConnect: ["123"], relationCreate: [{ id: "456", data: "data", shaped: true }] });
    });

    test("filter items based on shouldConnect for Connect operation - __connect test", () => {
        const item = { relation: [{ id: "123", data: "dataa", __connect: true }, { id: "456", data: "data" }] };
        const result = createRel(item, "relation", ["Connect", "Create"], "many", mockShapeModel);
        expect(result).toEqual({ relationConnect: ["123"], relationCreate: [{ id: "456", data: "data", shaped: true }] });
    });
});
