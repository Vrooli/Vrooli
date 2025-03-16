/* eslint-disable @typescript-eslint/ban-ts-comment */
import { FormSchema, InputType, UrlPrimitive } from "@local/shared";
import { expect } from "chai";
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
        expect(arrayToSearch(null, convertString)).to.be.undefined;
        expect(arrayToSearch("string", convertString)).to.be.undefined;
        expect(arrayToSearch(123, convertNumber)).to.be.undefined;
        expect(arrayToSearch({}, convertNumber)).to.be.undefined;
    });

    it("should return undefined for an empty array", () => {
        expect(arrayToSearch([], convertNumber)).to.be.undefined;
    });

    it("should apply the converter function to each item in the array", () => {
        const testArray = [1, 2.3, 3.7];
        arrayToSearch(testArray, convertNumber);
        expect(arrayToSearch(testArray, convertNumber)).to.deep.equal([1, 2, 4]);
    });

    it("should filter out undefined values returned by the converter", () => {
        const inputArray = [1, -1, 2, 0, 3, ""];
        const expectedResult = [1, 2, 3];
        expect(arrayToSearch(inputArray, convertNumber)).to.deep.equal(expectedResult);
    });

    it("should return undefined if all converted values are undefined", () => {
        const inputArray = [-1, 0, "", "   "];
        expect(arrayToSearch(inputArray, convertNumber)).to.be.undefined;
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
        // @ts-ignore: Testing runtime scenario
        expect(searchToArray(null, convertString)).to.be.undefined;
        expect(searchToArray("string", convertString)).to.be.undefined;
        expect(searchToArray(123, convertNumber)).to.be.undefined;
        expect(searchToArray({}, convertNumber)).to.be.undefined;
    });

    it("should return an empty array for an empty array input", () => {
        expect(searchToArray([], convertNumber)).to.deep.equal([]);
    });

    it("should apply the converter function to each item in the array", () => {
        const testArray = ["1", "2.3", "3.7"];
        expect(searchToArray(testArray, convertNumber)).to.deep.equal([1, 2, 4]);
    });

    it("should filter out undefined values returned by the converter", () => {
        const inputArray = ["1", "-1", "2", "0", "3", ""];
        const expectedResult = [1, 2, 3];
        expect(searchToArray(inputArray, convertNumber)).to.deep.equal(expectedResult);
    });

    it("should return an empty array if all converted values are undefined", () => {
        const inputArray = ["-1", "0", "", "   "];
        expect(searchToArray(inputArray, convertNumber)).to.deep.equal([]);
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
        // @ts-ignore: Testing runtime scenario
        const reverted = searchToArray(converted, convertNumber);
        expect(reverted).to.deep.equal([1, 2, 5]);
    });

    it("should correctly round-trip convert tag objects", () => {
        const tagArray = [
            { tag: "JavaScript" },
            { tag: "React" },
            { tag: " " }, // should be filtered out
            { notTag: "Angular" }, // should be filtered out
        ];
        const converted = arrayToSearch(tagArray, tagObjectToString);
        // @ts-ignore: Testing runtime scenario
        const reverted = searchToArray(converted, stringToTagObject);
        expect(reverted).to.deep.equal([
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
        // @ts-ignore: Testing runtime scenario
        const reverted = searchToArray(converted, stringToTagObject);
        expect(reverted).to.deep.equal([{ __typename: "Tag", tag: "NodeJS" }]);
    });

    it("should return empty array for all-invalid conversions", () => {
        const invalidTags = [
            { noTag: "VueJS" },
            { tag: "   " },
            null,
            undefined,
        ];
        const converted = arrayToSearch(invalidTags, tagObjectToString);
        // @ts-ignore: Testing runtime scenario
        const reverted = searchToArray(converted, stringToTagObject);
        expect(reverted).to.deep.equal([]);
    });
});

describe("tagObjectToString", () => {
    it("should convert valid tag objects to their string representation", () => {
        expect(tagObjectToString({ tag: "JavaScript" })).to.equal("JavaScript");
        expect(tagObjectToString({ tag: " HTML5 " })).to.equal("HTML5");
    });

    it("should return undefined for invalid or empty tags", () => {
        expect(tagObjectToString({ tag: "" })).to.be.undefined;
        expect(tagObjectToString({ tag: "     " })).to.be.undefined;
        expect(tagObjectToString({})).to.be.undefined;
    });

    it("should handle null and non-object inputs gracefully", () => {
        expect(tagObjectToString(null)).to.be.undefined;
        expect(tagObjectToString(undefined)).to.be.undefined;
        expect(tagObjectToString("string")).to.be.undefined;
    });
});

describe("stringToTagObject", () => {
    it("should convert valid strings to Tag objects", () => {
        expect(stringToTagObject("React")).to.deep.equal({ __typename: "Tag", tag: "React" });
        expect(stringToTagObject(" VueJS ")).to.deep.equal({ __typename: "Tag", tag: "VueJS" });
    });

    it("should return undefined for invalid or empty strings", () => {
        expect(stringToTagObject("")).to.be.undefined;
        expect(stringToTagObject("   ")).to.be.undefined;
    });

    it("should handle non-string inputs", () => {
        expect(stringToTagObject(null)).to.be.undefined;
        expect(stringToTagObject(123)).to.be.undefined;
    });
});

describe("Round-Trip Tag Conversion", () => {
    it("should preserve tag data through conversions", () => {
        const validTags = ["NodeJS", "Express", "   Angular  "];
        const roundTripResults = validTags.map(tag => stringToTagObject(tagObjectToString({ tag })));
        expect(roundTripResults).to.deep.equal([
            { __typename: "Tag", tag: "NodeJS" },
            { __typename: "Tag", tag: "Express" },
            { __typename: "Tag", tag: "Angular" }, // Trimmed in both conversions
        ]);
    });

    it("should return undefined for invalid conversions", () => {
        const invalidTags = ["", " ", {}, { tag: "   " }, { notTag: "VueJS" }, { tag: "boop" }];
        const roundTripResults = invalidTags.map(tag => stringToTagObject(tagObjectToString(tag)));
        expect(roundTripResults).to.deep.equal([undefined, undefined, undefined, undefined, undefined, { __typename: "Tag", tag: "boop" }]);
    });
});

describe("nonZeroNumber", () => {
    it("returns a number if it is non-zero", () => {
        expect(nonZeroNumber(5)).to.equal(5);
        expect(nonZeroNumber(-2)).to.equal(-2);
        expect(nonZeroNumber(3.5)).to.equal(3.5);
    });

    it("returns undefined for zero, non-number inputs, or non-finite numbers", () => {
        expect(nonZeroNumber(0)).to.be.undefined;
        expect(nonZeroNumber("string")).to.be.undefined;
        expect(nonZeroNumber(NaN)).to.be.undefined;
        expect(nonZeroNumber(Infinity)).to.be.undefined;
        expect(nonZeroNumber(-Infinity)).to.be.undefined;
    });

    it("parses strings into numbers if valid and non-zero", () => {
        expect(nonZeroNumber("10")).to.equal(10);
        expect(nonZeroNumber("-3.5")).to.equal(-3.5);
        expect(nonZeroNumber("0")).to.be.undefined;
        expect(nonZeroNumber("string")).to.be.undefined;
    });
});

describe("nonEmptyString", () => {
    it("returns the trimmed string if not empty", () => {
        expect(nonEmptyString(" Hello World ")).to.equal("Hello World");
        expect(nonEmptyString("Test")).to.equal("Test");
    });

    it("returns undefined for empty or whitespace-only strings", () => {
        expect(nonEmptyString("")).to.be.undefined;
        expect(nonEmptyString("     ")).to.be.undefined;
        expect(nonEmptyString("\n\t")).to.be.undefined;
    });

    it("handles non-string inputs gracefully", () => {
        expect(nonEmptyString(123)).to.be.undefined;
        expect(nonEmptyString(null)).to.be.undefined;
        expect(nonEmptyString(undefined)).to.be.undefined;
    });
});

describe("validBoolean", () => {
    it("returns the boolean value if input is boolean", () => {
        expect(validBoolean(true)).to.equal(true);
        expect(validBoolean(false)).to.equal(false);
    });

    it("parses \"true\" and \"false\" strings to boolean", () => {
        expect(validBoolean("true")).to.equal(true);
        expect(validBoolean("false")).to.equal(false);
    });

    it("returns undefined for non-boolean and incorrect string inputs", () => {
        expect(validBoolean("yes")).to.be.undefined;
        expect(validBoolean("no")).to.be.undefined;
        expect(validBoolean(0)).to.be.undefined;
        expect(validBoolean(1)).to.be.undefined;
        expect(validBoolean(null)).to.be.undefined;
        expect(validBoolean(undefined)).to.be.undefined;
        expect(validBoolean("")).to.be.undefined;
        expect(validBoolean(" ")).to.be.undefined;
    });
});

describe("validPrimitive", () => {
    it("returns the value for valid boolean inputs", () => {
        expect(validPrimitive(true)).to.equal(true);
        expect(validPrimitive(false)).to.equal(false);
    });

    it("returns the value for non-zero numbers", () => {
        expect(validPrimitive(42)).to.equal(42);
        expect(validPrimitive(-3.14)).to.equal(-3.14);
    });

    it("returns undefined for zero and non-finite numbers", () => {
        expect(validPrimitive(0)).to.be.undefined;
        expect(validPrimitive(NaN)).to.be.undefined;
        expect(validPrimitive(Infinity)).to.be.undefined;
    });

    it("returns the trimmed string if not empty", () => {
        expect(validPrimitive(" Hello ")).to.equal("Hello");
        expect(validPrimitive("World")).to.equal("World");
    });

    it("returns undefined for empty or whitespace-only strings", () => {
        expect(validPrimitive("")).to.be.undefined;
        expect(validPrimitive("    ")).to.be.undefined;
    });

    it("handles and returns non-array, non-Date objects", () => {
        const obj = { key: "value" };
        const anotherObj = new RegExp("pattern");
        expect(validPrimitive(obj)).to.equal(obj);
        expect(validPrimitive(anotherObj)).to.equal(anotherObj);
    });

    it("returns undefined for arrays, Dates, and null values", () => {
        expect(validPrimitive([1, 2, 3])).to.be.undefined;
        expect(validPrimitive(new Date())).to.be.undefined;
        expect(validPrimitive(null)).to.be.undefined;
    });

    it("returns undefined for unsupported types", () => {
        expect(validPrimitive(undefined)).to.be.undefined;
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        expect(validPrimitive(() => { })).to.be.undefined;
        expect(validPrimitive(Symbol("sym"))).to.be.undefined;
    });
});

describe("inputTypeToSearch", () => {
    const tagObject = { tag: "Technology" };

    it("handles Checkbox input correctly", () => {
        expect(inputTypeToSearch[InputType.Checkbox]([true, false])).to.deep.equal([true, false]);
        expect(inputTypeToSearch[InputType.Checkbox]([])).to.be.undefined;
        expect(inputTypeToSearch[InputType.Checkbox](["not boolean"])).to.be.undefined;
    });

    it("returns undefined for Dropzone as it is not supported", () => {
        expect(inputTypeToSearch[InputType.Dropzone]("file")).to.be.undefined;
    });

    it("handles IntegerInput correctly", () => {
        expect(inputTypeToSearch[InputType.IntegerInput]("3")).to.equal(3);
        expect(inputTypeToSearch[InputType.IntegerInput]("invalid")).to.be.undefined;
    });

    it("handles JSON input correctly", () => {
        expect(inputTypeToSearch[InputType.JSON]("{\"key\": \"value\"}")).to.equal("{\"key\": \"value\"}");
        expect(inputTypeToSearch[InputType.JSON]("")).to.be.undefined;
    });

    it("handles LanguageInput correctly", () => {
        expect(inputTypeToSearch[InputType.LanguageInput](["English", "French"])).to.deep.equal(["English", "French"]);
        expect(inputTypeToSearch[InputType.LanguageInput]([])).to.be.undefined;
    });

    it("handles Radio input correctly", () => {
        expect(inputTypeToSearch[InputType.Radio]("Selected")).to.equal("Selected");
        expect(inputTypeToSearch[InputType.Radio]("")).to.be.undefined;
    });

    it("handles Selector input correctly", () => {
        expect(inputTypeToSearch[InputType.Selector]("valid string")).to.equal("valid string");
        expect(inputTypeToSearch[InputType.Selector]({ key: "value" })).to.deep.equal({ key: "value" });
        expect(inputTypeToSearch[InputType.Selector](true)).to.equal(true);
        expect(inputTypeToSearch[InputType.Selector]("")).to.be.undefined;
    });

    it("handles Slider input correctly", () => {
        expect(inputTypeToSearch[InputType.Slider]("5.5")).to.equal(5.5);
        expect(inputTypeToSearch[InputType.Slider]("invalid")).to.be.undefined;
    });

    it("handles Switch input correctly", () => {
        expect(inputTypeToSearch[InputType.Switch](true)).to.equal(true);
        expect(inputTypeToSearch[InputType.Switch](false)).to.equal(false);
    });

    it("handles TagSelector input correctly", () => {
        expect(inputTypeToSearch[InputType.TagSelector]([tagObject, { tag: "Science" }])).to.deep.equal(["Technology", "Science"]);
        expect(inputTypeToSearch[InputType.TagSelector]([])).to.be.undefined;
    });

    it("handles Text input correctly", () => {
        expect(inputTypeToSearch[InputType.Text]("Hello")).to.equal("Hello");
        expect(inputTypeToSearch[InputType.Text]("")).to.be.undefined;
    });
});

describe("searchToInputType", () => {
    it("handles Checkbox input correctly", () => {
        expect(searchToInputType[InputType.Checkbox](["true", "false"])).to.deep.equal([true, false]);
        expect(searchToInputType[InputType.Checkbox](["invalid"])).to.deep.equal([undefined]);
        expect(searchToInputType[InputType.Checkbox]([])).to.deep.equal([]);
    });

    it("returns undefined for Dropzone as it is not supported", () => {
        expect(searchToInputType[InputType.Dropzone]("file")).to.be.undefined;
    });

    it("handles IntegerInput correctly", () => {
        expect(searchToInputType[InputType.IntegerInput]("3")).to.equal(3);
        expect(searchToInputType[InputType.IntegerInput]("0")).to.be.undefined;
        expect(searchToInputType[InputType.IntegerInput]("invalid")).to.be.undefined;
    });

    it("handles JSON input correctly", () => {
        expect(searchToInputType[InputType.JSON]("{\"key\": \"value\"}")).to.equal("{\"key\": \"value\"}");
        expect(searchToInputType[InputType.JSON]("")).to.be.undefined;
    });

    it("handles LanguageInput correctly", () => {
        expect(searchToInputType[InputType.LanguageInput](["English", "French"])).to.deep.equal(["English", "French"]);
        expect(searchToInputType[InputType.LanguageInput]([])).to.deep.equal([]);
    });

    it("handles Radio input correctly", () => {
        expect(searchToInputType[InputType.Radio]("Selected")).to.equal("Selected");
        expect(searchToInputType[InputType.Radio]("")).to.be.undefined;
    });

    it("handles Selector input correctly", () => {
        expect(searchToInputType[InputType.Selector]("valid string")).to.equal("valid string");
        expect(searchToInputType[InputType.Selector]({ key: "value" })).to.deep.equal({ key: "value" });
        expect(searchToInputType[InputType.Selector](true)).to.equal(true);
        expect(searchToInputType[InputType.Selector]("")).to.be.undefined;
    });

    it("handles Slider input correctly", () => {
        expect(searchToInputType[InputType.Slider]("5.5")).to.equal(5.5);
        expect(searchToInputType[InputType.Slider]("0")).to.be.undefined;
        expect(searchToInputType[InputType.Slider]("invalid")).to.be.undefined;
    });

    it("handles Switch input correctly", () => {
        expect(searchToInputType[InputType.Switch](true)).to.equal(true);
        expect(searchToInputType[InputType.Switch](false)).to.equal(false);
        expect(searchToInputType[InputType.Switch]("true")).to.equal(true);
        expect(searchToInputType[InputType.Switch]("false")).to.equal(false);
    });

    it("handles TagSelector input correctly", () => {
        const tagStrings = ["Technology", "Science"];
        const expectedTags = [{ __typename: "Tag", tag: "Technology" }, { __typename: "Tag", tag: "Science" }];
        expect(searchToInputType[InputType.TagSelector](tagStrings)).to.deep.equal(expectedTags);
        expect(searchToInputType[InputType.TagSelector]([])).to.deep.equal([]);
    });

    it("handles Text input correctly", () => {
        expect(searchToInputType[InputType.Text]("Hello")).to.equal("Hello");
        expect(searchToInputType[InputType.Text]("")).to.be.undefined;
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

        expect(convertFormikForSearch(formValues, mockSchema)).to.deep.equal(expectedResult);
    });

    it("ignores fields that are not in Formik values", () => {
        const formValues = {
            integerField: 42,
        };

        const expectedResult = {
            integerField: 42,
        };

        expect(convertFormikForSearch(formValues, mockSchema)).to.deep.equal(expectedResult);
    });

    it("does not include fields with unsupported types or undefined conversion results", () => {
        const formValues = {
            checkboxField: [], // Should return undefined because the array is empty
            integerField: 0, // Should return undefined because the value is zero
            dropzoneField: "fileData", // Unsupported
        };

        expect(convertFormikForSearch(formValues, mockSchema)).to.deep.equal({});
    });

    it("correctly handles empty and null values", () => {
        const formValues = {
            checkboxField: null,
            integerField: null,
            textField: null,
            tagSelectorField: null,
        };

        expect(convertFormikForSearch(formValues, mockSchema)).to.deep.equal({});
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

        expect(convertSearchForFormik(urlValues, mockSchema)).to.deep.equal(expectedResult);
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

        expect(convertSearchForFormik(urlValues, mockSchema)).to.deep.equal(expectedResult);
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

        expect(convertSearchForFormik(urlValues, mockSchema)).to.deep.equal(expectedResult);
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

        expect(convertSearchForFormik(urlValues, mockSchema)).to.deep.equal(expectedResult);
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

        expect(deserializedValues).to.deep.equal({
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

        expect(deserializedValues).to.deep.equal({
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

        expect(deserializedValues).to.deep.equal({
            checkboxField: undefined, // Invalid values filtered out
            integerField: undefined, // Not a number
            textField: undefined, // Empty string
            tagSelectorField: undefined, // Invalid tag
            radioField: undefined, // Empty string
        });
    });
});
