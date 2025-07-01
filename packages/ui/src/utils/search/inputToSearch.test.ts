// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-19
// AI_CHECK: TYPE_SAFETY=eliminated-5-ts-ignore-comments | LAST: 2025-06-28
import { InputType, type FormSchema, type UrlPrimitive } from "@vrooli/shared";
import { describe, it, expect } from "vitest";
import { arrayToSearch, convertFormikForSearch, convertSearchForFormik, inputTypeToSearch, nonEmptyString, nonZeroNumber, searchToArray, searchToInputType, stringToTagObject, tagObjectToString, validBoolean, validPrimitive } from "./inputToSearch.js";

describe("arrayToSearch", () => {
    // Mock number converter. Filters out values that are not numbers or are less than 0, and rounds them to the nearest integer
    function convertNumber(item: unknown) {
        return typeof item === "number" && Number.isFinite(item) && item > 0 ? Math.round(item) : undefined;
    }
    // Mock string converter. Filters out values that are not strings or are empty
    function convertString(item: unknown) {
        return typeof item === "string" && item.trim() !== "" ? item.trim() : undefined;
    }

    it("should return undefined for non-array inputs", () => {
        expect(arrayToSearch(null, convertString)).toBeUndefined();
        expect(arrayToSearch("string", convertString)).toBeUndefined();
        expect(arrayToSearch(123, convertNumber)).toBeUndefined();
        expect(arrayToSearch({}, convertNumber)).toBeUndefined();
    });

    it("should return undefined for an empty array", () => {
        expect(arrayToSearch([], convertNumber)).toBeUndefined();
    });

    it("should apply the converter function to each item in the array", () => {
        const testArray = [1, 2.3, 3.7];
        arrayToSearch(testArray, convertNumber);
        expect(arrayToSearch(testArray, convertNumber)).toEqual([1, 2, 4]);
    });

    it("should filter out undefined values returned by the converter", () => {
        const inputArray = [1, -1, 2, 0, 3, ""];
        const expectedResult = [1, 2, 3];
        expect(arrayToSearch(inputArray, convertNumber)).toEqual(expectedResult);
    });

    it("should return undefined if all converted values are undefined", () => {
        const inputArray = [-1, 0, "", "   "];
        expect(arrayToSearch(inputArray, convertNumber)).toBeUndefined();
    });
});

describe("searchToArray", () => {
    // Mock converter functions for different types
    function convertNumber(value: UrlPrimitive) {
        const num = Number(value);
        return Number.isFinite(num) && num > 0 ? Math.round(num) : undefined;
    }

    function convertString(value: UrlPrimitive) {
        const str = String(value);
        return str.trim() !== "" ? str.trim() : undefined;
    }

    it("should return undefined for non-array inputs", () => {
        expect(searchToArray(null as unknown as UrlPrimitive[], convertString)).toBeUndefined();
        expect(searchToArray("string", convertString)).toBeUndefined();
        expect(searchToArray(123, convertNumber)).toBeUndefined();
        expect(searchToArray({}, convertNumber)).toBeUndefined();
    });

    it("should return an empty array for an empty array input", () => {
        expect(searchToArray([], convertNumber)).toEqual([]);
    });

    it("should apply the converter function to each item in the array", () => {
        const testArray = ["1", "2.3", "3.7"];
        expect(searchToArray(testArray, convertNumber)).toEqual([1, 2, 4]);
    });

    it("should filter out undefined values returned by the converter", () => {
        const inputArray = ["1", "-1", "2", "0", "3", ""];
        const expectedResult = [1, 2, 3];
        expect(searchToArray(inputArray, convertNumber)).toEqual(expectedResult);
    });

    it("should return an empty array if all converted values are undefined", () => {
        const inputArray = ["-1", "0", "", "   "];
        expect(searchToArray(inputArray, convertNumber)).toEqual([]);
    });
});

describe("arrayToSearch and searchToArray", () => {
    function tagObjectToString(tag: unknown): string | undefined {
        return tag !== null && typeof tag === "object" && Object.prototype.hasOwnProperty.call(tag, "tag") ? (tag as { tag: string }).tag : undefined;
    }

    function stringToTagObject(tag: unknown): object | undefined {
        if (typeof tag !== "string" || tag.trim().length === 0) return undefined;
        return { __typename: "Tag", tag: tag.trim() } as object;
    }

    function convertNumber(item: unknown) {
        return typeof item === "number" && Number.isFinite(item) && item > 0 ? Math.round(item) : undefined;
    }

    it("should correctly round-trip convert numbers", () => {
        const numbersArray = [1, 2.3, 4.7, 0, -1];
        const converted = arrayToSearch(numbersArray, convertNumber);
        const reverted = searchToArray(converted as UrlPrimitive[], convertNumber);
        expect(reverted).toEqual([1, 2, 5]);
    });

    it("should correctly round-trip convert tag objects", () => {
        const tagArray = [
            { tag: "JavaScript" },
            { tag: "React" },
            { tag: " " }, // should be filtered out
            { notTag: "Angular" }, // should be filtered out
        ];
        const converted = arrayToSearch(tagArray, tagObjectToString);
        const reverted = searchToArray(converted as UrlPrimitive[], stringToTagObject);
        expect(reverted).toEqual([
            { __typename: "Tag", tag: "JavaScript" },
            { __typename: "Tag", tag: "React" },
        ]);
    });

    it("should handle mixed valid and invalid objects", () => {
        const mixedArray = [
            { tag: "NodeJS" },
            { tag: "" },
            { noTag: "VueJS" },
        ];
        const converted = arrayToSearch(mixedArray, tagObjectToString);
        const reverted = searchToArray(converted as UrlPrimitive[], stringToTagObject);
        expect(reverted).toEqual([{ __typename: "Tag", tag: "NodeJS" }]);
    });

    it("should return empty array for all-invalid conversions", () => {
        const invalidTags = [
            { noTag: "VueJS" },
            { tag: "   " },
            null,
            undefined,
        ];
        const converted = arrayToSearch(invalidTags, tagObjectToString);
        const reverted = searchToArray(converted as UrlPrimitive[], stringToTagObject);
        expect(reverted).toEqual([]);
    });
});

describe("tagObjectToString", () => {
    it("should convert valid tag objects to their string representation", () => {
        expect(tagObjectToString({ tag: "JavaScript" })).toBe("JavaScript");
        expect(tagObjectToString({ tag: " HTML5 " })).toBe("HTML5");
    });

    it("should return undefined for invalid or empty tags", () => {
        expect(tagObjectToString({ tag: "" })).toBeUndefined();
        expect(tagObjectToString({ tag: "     " })).toBeUndefined();
        expect(tagObjectToString({})).toBeUndefined();
    });

    it("should handle null and non-object inputs gracefully", () => {
        expect(tagObjectToString(null)).toBeUndefined();
        expect(tagObjectToString(undefined)).toBeUndefined();
        expect(tagObjectToString("string")).toBeUndefined();
    });
});

describe("stringToTagObject", () => {
    it("should convert valid strings to Tag objects", () => {
        expect(stringToTagObject("React")).toEqual({ __typename: "Tag", tag: "React" });
        expect(stringToTagObject(" VueJS ")).toEqual({ __typename: "Tag", tag: "VueJS" });
    });

    it("should return undefined for invalid or empty strings", () => {
        expect(stringToTagObject("")).toBeUndefined();
        expect(stringToTagObject("   ")).toBeUndefined();
    });

    it("should handle non-string inputs", () => {
        expect(stringToTagObject(null)).toBeUndefined();
        expect(stringToTagObject(123)).toBeUndefined();
    });
});

describe("Round-Trip Tag Conversion", () => {
    it("should preserve tag data through conversions", () => {
        const validTags = ["NodeJS", "Express", "   Angular  "];
        const roundTripResults = validTags.map(tag => stringToTagObject(tagObjectToString({ tag })));
        expect(roundTripResults).toEqual([
            { __typename: "Tag", tag: "NodeJS" },
            { __typename: "Tag", tag: "Express" },
            { __typename: "Tag", tag: "Angular" }, // Trimmed in both conversions
        ]);
    });

    it("should return undefined for invalid conversions", () => {
        const invalidTags = ["", " ", {}, { tag: "   " }, { notTag: "VueJS" }, { tag: "boop" }];
        const roundTripResults = invalidTags.map(tag => stringToTagObject(tagObjectToString(tag)));
        expect(roundTripResults).toEqual([undefined, undefined, undefined, undefined, undefined, { __typename: "Tag", tag: "boop" }]);
    });
});

describe("nonZeroNumber", () => {
    it("returns a number if it is non-zero", () => {
        expect(nonZeroNumber(5)).toBe(5);
        expect(nonZeroNumber(-2)).toBe(-2);
        expect(nonZeroNumber(3.5)).toBe(3.5);
    });

    it("returns undefined for zero, non-number inputs, or non-finite numbers", () => {
        expect(nonZeroNumber(0)).toBeUndefined();
        expect(nonZeroNumber("string")).toBeUndefined();
        expect(nonZeroNumber(NaN)).toBeUndefined();
        expect(nonZeroNumber(Infinity)).toBeUndefined();
        expect(nonZeroNumber(-Infinity)).toBeUndefined();
    });

    it("parses strings into numbers if valid and non-zero", () => {
        expect(nonZeroNumber("10")).toBe(10);
        expect(nonZeroNumber("-3.5")).toBe(-3.5);
        expect(nonZeroNumber("0")).toBeUndefined();
        expect(nonZeroNumber("string")).toBeUndefined();
    });
});

describe("nonEmptyString", () => {
    it("returns the trimmed string if not empty", () => {
        expect(nonEmptyString(" Hello World ")).toBe("Hello World");
        expect(nonEmptyString("Test")).toBe("Test");
    });

    it("returns undefined for empty or whitespace-only strings", () => {
        expect(nonEmptyString("")).toBeUndefined();
        expect(nonEmptyString("     ")).toBeUndefined();
        expect(nonEmptyString("\n\t")).toBeUndefined();
    });

    it("handles non-string inputs gracefully", () => {
        expect(nonEmptyString(123)).toBeUndefined();
        expect(nonEmptyString(null)).toBeUndefined();
        expect(nonEmptyString(undefined)).toBeUndefined();
    });
});

describe("validBoolean", () => {
    it("returns the boolean value if input is boolean", () => {
        expect(validBoolean(true)).toBe(true);
        expect(validBoolean(false)).toBe(false);
    });

    it("parses \"true\" and \"false\" strings to boolean", () => {
        expect(validBoolean("true")).toBe(true);
        expect(validBoolean("false")).toBe(false);
    });

    it("returns undefined for non-boolean and incorrect string inputs", () => {
        expect(validBoolean("yes")).toBeUndefined();
        expect(validBoolean("no")).toBeUndefined();
        expect(validBoolean(0)).toBeUndefined();
        expect(validBoolean(1)).toBeUndefined();
        expect(validBoolean(null)).toBeUndefined();
        expect(validBoolean(undefined)).toBeUndefined();
        expect(validBoolean("")).toBeUndefined();
        expect(validBoolean(" ")).toBeUndefined();
    });
});

describe("validPrimitive", () => {
    it("returns the value for valid boolean inputs", () => {
        expect(validPrimitive(true)).toBe(true);
        expect(validPrimitive(false)).toBe(false);
    });

    it("returns the value for non-zero numbers", () => {
        expect(validPrimitive(42)).toBe(42);
        expect(validPrimitive(-3.14)).toBe(-3.14);
    });

    it("returns undefined for zero and non-finite numbers", () => {
        expect(validPrimitive(0)).toBeUndefined();
        expect(validPrimitive(NaN)).toBeUndefined();
        expect(validPrimitive(Infinity)).toBeUndefined();
    });

    it("returns the trimmed string if not empty", () => {
        expect(validPrimitive(" Hello ")).toBe("Hello");
        expect(validPrimitive("World")).toBe("World");
    });

    it("returns undefined for empty or whitespace-only strings", () => {
        expect(validPrimitive("")).toBeUndefined();
        expect(validPrimitive("    ")).toBeUndefined();
    });

    it("handles and returns non-array, non-Date objects", () => {
        const obj = { key: "value" };
        const anotherObj = new RegExp("pattern");
        expect(validPrimitive(obj)).toBe(obj);
        expect(validPrimitive(anotherObj)).toBe(anotherObj);
    });

    it("returns undefined for arrays, Dates, and null values", () => {
        expect(validPrimitive([1, 2, 3])).toBeUndefined();
        expect(validPrimitive(new Date())).toBeUndefined();
        expect(validPrimitive(null)).toBeUndefined();
    });

    it("returns undefined for unsupported types", () => {
        expect(validPrimitive(undefined)).toBeUndefined();
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        expect(validPrimitive(() => { })).toBeUndefined();
        expect(validPrimitive(Symbol("sym"))).toBeUndefined();
    });
});

describe("inputTypeToSearch", () => {
    const tagObject = { tag: "Technology" };

    it("handles Checkbox input correctly", () => {
        expect(inputTypeToSearch[InputType.Checkbox]([true, false])).toEqual([true, false]);
        expect(inputTypeToSearch[InputType.Checkbox]([])).toBeUndefined();
        expect(inputTypeToSearch[InputType.Checkbox](["not boolean"])).toBeUndefined();
    });

    it("returns undefined for Dropzone as it is not supported", () => {
        expect(inputTypeToSearch[InputType.Dropzone]("file")).toBeUndefined();
    });

    it("handles IntegerInput correctly", () => {
        expect(inputTypeToSearch[InputType.IntegerInput]("3")).toBe(3);
        expect(inputTypeToSearch[InputType.IntegerInput]("invalid")).toBeUndefined();
    });

    it("handles JSON input correctly", () => {
        expect(inputTypeToSearch[InputType.JSON]("{\"key\": \"value\"}")).toBe("{\"key\": \"value\"}");
        expect(inputTypeToSearch[InputType.JSON]("")).toBeUndefined();
    });

    it("handles LanguageInput correctly", () => {
        expect(inputTypeToSearch[InputType.LanguageInput](["English", "French"])).toEqual(["English", "French"]);
        expect(inputTypeToSearch[InputType.LanguageInput]([])).toBeUndefined();
    });

    it("handles Radio input correctly", () => {
        expect(inputTypeToSearch[InputType.Radio]("Selected")).toBe("Selected");
        expect(inputTypeToSearch[InputType.Radio]("")).toBeUndefined();
    });

    it("handles Selector input correctly", () => {
        expect(inputTypeToSearch[InputType.Selector]("valid string")).toBe("valid string");
        expect(inputTypeToSearch[InputType.Selector]({ key: "value" })).toEqual({ key: "value" });
        expect(inputTypeToSearch[InputType.Selector](true)).toBe(true);
        expect(inputTypeToSearch[InputType.Selector]("")).toBeUndefined();
    });

    it("handles Slider input correctly", () => {
        expect(inputTypeToSearch[InputType.Slider]("5.5")).toBe(5.5);
        expect(inputTypeToSearch[InputType.Slider]("invalid")).toBeUndefined();
    });

    it("handles Switch input correctly", () => {
        expect(inputTypeToSearch[InputType.Switch](true)).toBe(true);
        expect(inputTypeToSearch[InputType.Switch](false)).toBe(false);
    });

    it("handles TagSelector input correctly", () => {
        expect(inputTypeToSearch[InputType.TagSelector]([tagObject, { tag: "Science" }])).toEqual(["Technology", "Science"]);
        expect(inputTypeToSearch[InputType.TagSelector]([])).toBeUndefined();
    });

    it("handles Text input correctly", () => {
        expect(inputTypeToSearch[InputType.Text]("Hello")).toBe("Hello");
        expect(inputTypeToSearch[InputType.Text]("")).toBeUndefined();
    });
});

describe("searchToInputType", () => {
    it("handles Checkbox input correctly", () => {
        expect(searchToInputType[InputType.Checkbox](["true", "false"])).toEqual([true, false]);
        expect(searchToInputType[InputType.Checkbox](["invalid"])).toEqual([]);
        expect(searchToInputType[InputType.Checkbox]([])).toEqual([]);
    });

    it("returns undefined for Dropzone as it is not supported", () => {
        expect(searchToInputType[InputType.Dropzone]("file")).toBeUndefined();
    });

    it("handles IntegerInput correctly", () => {
        expect(searchToInputType[InputType.IntegerInput]("3")).toBe(3);
        expect(searchToInputType[InputType.IntegerInput]("0")).toBeUndefined();
        expect(searchToInputType[InputType.IntegerInput]("invalid")).toBeUndefined();
    });

    it("handles JSON input correctly", () => {
        expect(searchToInputType[InputType.JSON]("{\"key\": \"value\"}")).toBe("{\"key\": \"value\"}");
        expect(searchToInputType[InputType.JSON]("")).toBeUndefined();
    });

    it("handles LanguageInput correctly", () => {
        expect(searchToInputType[InputType.LanguageInput](["English", "French"])).toEqual(["English", "French"]);
        expect(searchToInputType[InputType.LanguageInput]([])).toEqual([]);
    });

    it("handles Radio input correctly", () => {
        expect(searchToInputType[InputType.Radio]("Selected")).toBe("Selected");
        expect(searchToInputType[InputType.Radio]("")).toBeUndefined();
    });

    it("handles Selector input correctly", () => {
        expect(searchToInputType[InputType.Selector]("valid string")).toBe("valid string");
        expect(searchToInputType[InputType.Selector]({ key: "value" })).toEqual({ key: "value" });
        expect(searchToInputType[InputType.Selector](true)).toBe(true);
        expect(searchToInputType[InputType.Selector]("")).toBeUndefined();
    });

    it("handles Slider input correctly", () => {
        expect(searchToInputType[InputType.Slider]("5.5")).toBe(5.5);
        expect(searchToInputType[InputType.Slider]("0")).toBeUndefined();
        expect(searchToInputType[InputType.Slider]("invalid")).toBeUndefined();
    });

    it("handles Switch input correctly", () => {
        expect(searchToInputType[InputType.Switch](true)).toBe(true);
        expect(searchToInputType[InputType.Switch](false)).toBe(false);
        expect(searchToInputType[InputType.Switch]("true")).toBe(true);
        expect(searchToInputType[InputType.Switch]("false")).toBe(false);
    });

    it("handles TagSelector input correctly", () => {
        const tagStrings = ["Technology", "Science"];
        const expectedTags = [{ __typename: "Tag", tag: "Technology" }, { __typename: "Tag", tag: "Science" }];
        expect(searchToInputType[InputType.TagSelector](tagStrings)).toEqual(expectedTags);
        expect(searchToInputType[InputType.TagSelector]([])).toEqual([]);
    });

    it("handles Text input correctly", () => {
        expect(searchToInputType[InputType.Text]("Hello")).toBe("Hello");
        expect(searchToInputType[InputType.Text]("")).toBeUndefined();
    });
});

describe("convertFormikForSearch", () => {
    // Mock schema based on hypothetical form field types
    const mockSchema = {
        elements: [
            { fieldName: "checkboxField", type: InputType.Checkbox },
            { fieldName: "integerField", type: InputType.IntegerInput },
            { fieldName: "textField", type: InputType.Text },
            { fieldName: "tagSelectorField", type: InputType.TagSelector },
            { fieldName: "dropzoneField", type: InputType.Dropzone }, // Not supported
        ],
    } as FormSchema;

    it("converts Formik values according to the schema", () => {
        const formValues = {
            checkboxField: [true, false],
            integerField: 42,
            textField: "Hello world",
            tagSelectorField: [{ tag: "Technology" }, { tag: "Innovation" }],
            dropzoneField: "fileData",
        };

        const expectedResult = {
            checkboxField: [true, false],
            integerField: 42,
            textField: "Hello world",
            tagSelectorField: ["Technology", "Innovation"],
        };

        expect(convertFormikForSearch(formValues, mockSchema)).toEqual(expectedResult);
    });

    it("ignores fields that are not in Formik values", () => {
        const formValues = {
            integerField: 42,
        };

        const expectedResult = {
            integerField: 42,
        };

        expect(convertFormikForSearch(formValues, mockSchema)).toEqual(expectedResult);
    });

    it("does not include fields with unsupported types or undefined conversion results", () => {
        const formValues = {
            checkboxField: [], // Should return undefined because the array is empty
            integerField: 0, // Should return undefined because the value is zero
            dropzoneField: "fileData", // Unsupported
        };

        expect(convertFormikForSearch(formValues, mockSchema)).toEqual({});
    });

    it("correctly handles empty and null values", () => {
        const formValues = {
            checkboxField: null,
            integerField: null,
            textField: null,
            tagSelectorField: null,
        };

        expect(convertFormikForSearch(formValues, mockSchema)).toEqual({});
    });
});

describe("convertSearchForFormik", () => {
    // Mock schema based on hypothetical form field types
    const mockSchema = {
        elements: [
            { fieldName: "checkboxField", type: InputType.Checkbox },
            { fieldName: "integerField", type: InputType.IntegerInput },
            { fieldName: "textField", type: InputType.Text },
            { fieldName: "tagSelectorField", type: InputType.TagSelector },
            { fieldName: "radioField", type: InputType.Radio },
        ],
    } as FormSchema;

    it("converts URL search parameters back to Formik values", () => {
        const urlValues = {
            checkboxField: ["true", "false"],
            integerField: "42",
            textField: "Hello world",
            tagSelectorField: ["Technology", "Innovation"],
            radioField: "Selected",
        };

        const expectedResult = {
            checkboxField: [true, false],
            integerField: 42,
            textField: "Hello world",
            tagSelectorField: [{ __typename: "Tag", tag: "Technology" }, { __typename: "Tag", tag: "Innovation" }],
            radioField: "Selected",
        };

        expect(convertSearchForFormik(urlValues, mockSchema)).toEqual(expectedResult);
    });

    it("sets undefined for fields not present in URL values", () => {
        const urlValues = {
            textField: "Hello world",
        };

        const expectedResult = {
            checkboxField: undefined,
            integerField: undefined,
            textField: "Hello world",
            tagSelectorField: undefined,
            radioField: undefined,
        };

        expect(convertSearchForFormik(urlValues, mockSchema)).toEqual(expectedResult);
    });

    it("handles invalid data types correctly", () => {
        const urlValues = {
            checkboxField: ["not boolean"],
            integerField: "not a number",
            textField: "",
            tagSelectorField: 3,
            radioField: "",
        };

        const expectedResult = {
            checkboxField: [],
            integerField: undefined,
            textField: undefined,
            tagSelectorField: undefined,
            radioField: undefined,
        };

        expect(convertSearchForFormik(urlValues, mockSchema)).toEqual(expectedResult);
    });

    it("manages missing and null fields appropriately", () => {
        const urlValues = {
            checkboxField: null,
            integerField: null,
        };

        const expectedResult = {
            checkboxField: undefined,
            integerField: undefined,
            textField: undefined,
            tagSelectorField: undefined,
            radioField: undefined,
        };

        expect(convertSearchForFormik(urlValues, mockSchema)).toEqual(expectedResult);
    });
});

describe("Round-trip Data Conversion", () => {
    const mockSchema = {
        elements: [
            { fieldName: "checkboxField", type: InputType.Checkbox },
            { fieldName: "integerField", type: InputType.IntegerInput },
            { fieldName: "textField", type: InputType.Text },
            { fieldName: "tagSelectorField", type: InputType.TagSelector },
            { fieldName: "radioField", type: InputType.Radio },
        ],
    } as FormSchema;

    it("ensures data integrity through conversions for valid inputs", () => {
        const originalFormValues = {
            checkboxField: [true, false],
            integerField: 42,
            textField: "Hello world",
            tagSelectorField: [{ tag: "Technology" }, { tag: "Innovation" }],
            radioField: "Selected",
        };

        const serializedValues = convertFormikForSearch(originalFormValues, mockSchema);
        const deserializedValues = convertSearchForFormik(serializedValues, mockSchema);

        expect(deserializedValues).toEqual({
            checkboxField: [true, false],
            integerField: 42,
            textField: "Hello world",
            tagSelectorField: [{ __typename: "Tag", tag: "Technology" }, { __typename: "Tag", tag: "Innovation" }],
            radioField: "Selected",
        });
    });

    it("handles incomplete data gracefully", () => {
        const partialFormValues = {
            integerField: 42,
            textField: "Hello world",
        };

        const serializedValues = convertFormikForSearch(partialFormValues, mockSchema);
        const deserializedValues = convertSearchForFormik(serializedValues, mockSchema);

        expect(deserializedValues).toEqual({
            checkboxField: undefined,
            integerField: 42,
            textField: "Hello world",
            tagSelectorField: undefined,
            radioField: undefined,
        });
    });

    it("properly processes and discards invalid data", () => {
        const invalidFormValues = {
            checkboxField: [null, "not boolean"],
            integerField: "not a number",
            textField: "",
            tagSelectorField: [{ tag: "" }],
            radioField: "",
        };

        const serializedValues = convertFormikForSearch(invalidFormValues, mockSchema);
        const deserializedValues = convertSearchForFormik(serializedValues, mockSchema);

        expect(deserializedValues).toEqual({
            checkboxField: undefined, // Invalid values filtered out
            integerField: undefined, // Not a number
            textField: undefined, // Empty string
            tagSelectorField: undefined, // Invalid tag
            radioField: undefined, // Empty string
        });
    });
});
