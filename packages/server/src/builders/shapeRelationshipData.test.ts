/* eslint-disable @typescript-eslint/ban-ts-comment */
import { filterFields, shapeRelationshipData } from "./shapeRelationshipData";

describe("filterFields", () => {
    it("correctly filters out specified fields", () => {
        const data = { name: "John", age: 30, country: "USA" };
        const excludes = ["age"];
        const expected = { name: "John", country: "USA" };
        expect(filterFields(data, excludes)).toEqual(expected);
    });

    it("returns original object when no exclusions", () => {
        const data = { name: "John", age: 30 };
        const excludes = [];
        expect(filterFields(data, excludes)).toEqual(data);
    });

    it("returns original object when no fields match excludes", () => {
        const data = { name: "John", age: 30 };
        const excludes = ["country"];
        expect(filterFields(data, excludes)).toEqual(data);
    });

    it("ignores non-existent fields in excludes", () => {
        const data = { name: "John", age: 30 };
        const excludes = ["country", "email"];
        expect(filterFields(data, excludes)).toEqual(data);
    });

    it("returns an empty object when data is empty", () => {
        const data = {};
        const excludes = ["name"];
        expect(filterFields(data, excludes)).toEqual({});
    });

    it("handles null data object gracefully", () => {
        const data = null;
        const excludes = ["name"];
        // @ts-ignore: Testing runtime scenario
        expect(filterFields(data, excludes)).toEqual({});
    });

    it("returns an empty object when all fields are excluded", () => {
        const data = { name: "John", age: 30 };
        const excludes = ["name", "age"];
        expect(filterFields(data, excludes)).toEqual({});
    });

    it("does not process nested objects", () => {
        const data = { user: { name: "John", age: 30 }, country: "USA" };
        const excludes = ["user"];
        const expected = { country: "USA" };
        expect(filterFields(data, excludes)).toEqual(expected);
    });
});

describe("shapeRelationshipData", () => {
    it("converts single ID string to array with one object (isOneToOne false)", () => {
        expect(shapeRelationshipData("123", [], false)).toEqual([{ id: "123" }]);
    });

    it("converts single ID string to single object (isOneToOne true)", () => {
        expect(shapeRelationshipData("123", [], true)).toEqual({ id: "123" });
    });

    it("wraps single object in array (isOneToOne false)", () => {
        const data = { id: "123", name: "John" };
        expect(shapeRelationshipData(data, [], false)).toEqual([data]);
    });

    it("returns single object as is (isOneToOne true)", () => {
        const data = { id: "123", name: "John" };
        expect(shapeRelationshipData(data, [], true)).toEqual(data);
    });

    it("converts array of ID strings to array of objects (isOneToOne false)", () => {
        expect(shapeRelationshipData(["123", "456"], [], false)).toEqual([{ id: "123" }, { id: "456" }]);
    });

    it("converts first ID in array to single object (isOneToOne true)", () => {
        expect(shapeRelationshipData(["123", "456"], [], true)).toEqual({ id: "123" });
    });

    it("converts array of objects applying exclusions (isOneToOne false)", () => {
        const data = [{ id: "123", name: "John" }, { id: "456", name: "Doe" }];
        expect(shapeRelationshipData(data, ["name"], false)).toEqual([{ id: "123" }, { id: "456" }]);
    });

    it("returns first object in array after applying exclusions (isOneToOne true)", () => {
        const data = [{ id: "123", name: "John" }, { id: "456", name: "Doe" }];
        expect(shapeRelationshipData(data, ["name"], true)).toEqual({ id: "123" });
    });

    it("handles non-object and non-string data correctly (isOneToOne false)", () => {
        expect(shapeRelationshipData(123, [], false)).toEqual([{ id: 123 }]);
    });

    it("handles non-object and non-string data correctly (isOneToOne true)", () => {
        expect(shapeRelationshipData(123, [], true)).toEqual({ id: 123 });
    });
});
