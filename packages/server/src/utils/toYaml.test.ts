/* eslint-disable @typescript-eslint/ban-ts-comment */
import { describe, expect, it } from "vitest";
import { arrayToYaml, objectToYaml } from "./toYaml.js";

describe("YAML Conversion", () => {

    describe("objectToYaml", () => {
        it("converts a simple object to YAML", () => {
            const obj = { key: "value", nullKey: null, undefinedKey: undefined, numberKey: 123 };
            const expectedYaml = "key: value\nnullKey: null\nnumberKey: 123\n";
            expect(objectToYaml(obj)).toBe(expectedYaml);
        });

        it("converts nested objects to YAML", () => {
            const obj = { parent: { child: "value", nullChild: null } };
            // Both children should have the same indent level
            const expectedYaml = `parent:
  child: value
  nullChild: null
`;
            expect(objectToYaml(obj)).to.deep.equal(expectedYaml);
        });

        it("handles multiple keys and nesting", () => {
            const obj = {
                first: "value1",
                second: {
                    nested: "nestedValue",
                    third: [
                        {
                            key: -420.69,
                            deeper: {
                                deepKey1: "deepValue1",
                                deepKey2: "deepValue2",
                            },
                        },
                        {
                            key1: "value1",
                            key2: "value2",
                            deeper: {
                                deepKey1: {
                                    deepestKey: "deepestValue",
                                    anotherKey: null,
                                },
                            },
                        },
                    ],
                },
                fourth: null,
            };
            const expectedYaml = `first: value1
second:
  nested: nestedValue
  third:
    - key: -420.69
      deeper:
        deepKey1: deepValue1
        deepKey2: deepValue2
    - key1: value1
      key2: value2
      deeper:
        deepKey1:
          deepestKey: deepestValue
          anotherKey: null
fourth: null
`;
            expect(objectToYaml(obj)).to.deep.equal(expectedYaml);
        });

        it("converts objects with array values to YAML", () => {
            const obj = { list: ["item1", "item2", ["subitem1", "subitem2"]] };
            const expectedYaml = `list:
  - item1
  - item2
  -
    - subitem1
    - subitem2
`;
            expect(objectToYaml(obj)).to.deep.equal(expectedYaml);
        });

        it("returns an empty string for null input", () => {
            // @ts-ignore: Testing runtime scenario
            expect(objectToYaml(null)).to.deep.equal("");
        });

        it("returns an empty string for undefined input", () => {
            // @ts-ignore: Testing runtime scenario
            expect(objectToYaml(undefined)).to.deep.equal("");
        });

        it("handles non-object inputs gracefully", () => {
            const nonObjectInputs = ["string", 123, true, []];
            nonObjectInputs.forEach(input => {
                // @ts-ignore: Testing runtime scenario
                expect(objectToYaml(input)).to.deep.equal("");
            });
        });
    });

    describe("arrayToYaml", () => {
        it("converts a simple array to YAML", () => {
            const arr = ["item1", "item2"];
            const expectedYaml = "- item1\n- item2\n";
            expect(arrayToYaml(arr)).toBe(expectedYaml);
        });

        it("converts an array of objects to YAML", () => {
            const arr = [{ key: "value" }, { anotherKey: "anotherValue" }];
            const expectedYaml = "- key: value\n- anotherKey: anotherValue\n";
            expect(arrayToYaml(arr)).toBe(expectedYaml);
        });

        it("handles nested arrays", () => {
            const arr = [["subitem1", "subitem2"], ["subitem3"]];
            const expectedYaml = `-
  - subitem1
  - subitem2
-
  - subitem3
`;
            expect(arrayToYaml(arr)).toBe(expectedYaml);
        });

        it("returns an empty string for null input", () => {
            // @ts-ignore: Testing runtime scenario
            expect(arrayToYaml(null)).to.deep.equal("");
        });

        it("returns an empty string for undefined input", () => {
            // @ts-ignore: Testing runtime scenario
            expect(arrayToYaml(undefined)).to.deep.equal("");
        });

        it("handles non-array inputs gracefully", () => {
            const nonArrayInputs = ["string", 123, true, {}];
            nonArrayInputs.forEach(input => {
                // @ts-ignore: Testing runtime scenario
                expect(arrayToYaml(input)).to.deep.equal("");
            });
        });
    });
});
