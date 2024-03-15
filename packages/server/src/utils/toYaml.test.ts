/* eslint-disable @typescript-eslint/ban-ts-comment */
import { arrayToYaml, objectToYaml } from "./toYaml";

describe("YAML Conversion", () => {

    describe("objectToYaml", () => {
        it("converts a simple object to YAML", () => {
            const obj = { key: "value" };
            const expectedYaml = "key: value\n";
            expect(objectToYaml(obj)).toEqual(expectedYaml);
        });

        it("converts nested objects to YAML", () => {
            const obj = { parent: { child: "value" } };
            const expectedYaml = "parent:\n  child: value\n";
            expect(objectToYaml(obj)).toEqual(expectedYaml);
        });

        it("handles multiple keys and nesting", () => {
            const obj = {
                first: "value1",
                second: { nested: "nestedValue" },
            };
            const expectedYaml = "first: value1\nsecond:\n  nested: nestedValue\n";
            expect(objectToYaml(obj)).toEqual(expectedYaml);
        });

        it("converts objects with array values to YAML", () => {
            const obj = { list: ["item1", "item2"] };
            const expectedYaml = "list:\n  - item1\n  - item2\n";
            expect(objectToYaml(obj)).toEqual(expectedYaml);
        });

        it("returns an empty string for null input", () => {
            // @ts-ignore: Testing runtime scenario
            expect(objectToYaml(null)).toEqual("");
        });

        it("returns an empty string for undefined input", () => {
            // @ts-ignore: Testing runtime scenario
            expect(objectToYaml(undefined)).toEqual("");
        });

        it("handles non-object inputs gracefully", () => {
            const nonObjectInputs = ["string", 123, true, []];
            nonObjectInputs.forEach(input => {
                // @ts-ignore: Testing runtime scenario
                expect(objectToYaml(input)).toEqual("");
            });
        });
    });

    describe("arrayToYaml", () => {
        it("converts a simple array to YAML", () => {
            const arr = ["item1", "item2"];
            const expectedYaml = "- item1\n- item2\n";
            expect(arrayToYaml(arr)).toEqual(expectedYaml);
        });

        it("converts an array of objects to YAML", () => {
            const arr = [{ key: "value" }, { anotherKey: "anotherValue" }];
            const expectedYaml = "- key: value\n- anotherKey: anotherValue\n";
            expect(arrayToYaml(arr)).toEqual(expectedYaml);
        });

        it("handles nested arrays", () => {
            const arr = [["subitem1", "subitem2"], ["subitem3"]];
            const expectedYaml = "-\n  - subitem1\n  - subitem2\n-\n  - subitem3\n";
            expect(arrayToYaml(arr)).toEqual(expectedYaml);
        });

        it("returns an empty string for null input", () => {
            // @ts-ignore: Testing runtime scenario
            expect(arrayToYaml(null)).toEqual("");
        });

        it("returns an empty string for undefined input", () => {
            // @ts-ignore: Testing runtime scenario
            expect(arrayToYaml(undefined)).toEqual("");
        });

        it("handles non-array inputs gracefully", () => {
            const nonArrayInputs = ["string", 123, true, {}];
            nonArrayInputs.forEach(input => {
                // @ts-ignore: Testing runtime scenario
                expect(arrayToYaml(input)).toEqual("");
            });
        });
    });
});
