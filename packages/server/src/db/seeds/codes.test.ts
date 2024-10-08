import { listToNumberedPlaintext, listToPlaintext, parseRunIOFromPlaintext, transformSearchTerms } from "./codes";

describe("parseRunIOFromPlaintext", () => {
    it("should parse inputs and outputs correctly", () => {
        const formData = {
            "input-name": true,
            "output-result": true,
        };
        const text = "name: John Doe\nresult: Success";
        const expectedResult = {
            inputs: { "name": "John Doe" },
            outputs: { "result": "Success" },
        };
        expect(parseRunIOFromPlaintext({ formData, text })).toEqual(expectedResult);
    });

    it("should ignore keys that do not match any input or output", () => {
        const formData = {
            "input-name": true,
            "output-result": true,
        };
        const text = "name: John Doe\nage: 30\nresult: Success";
        const expectedResult = {
            inputs: { "name": "John Doe" },
            outputs: { "result": "Success" },
        };
        expect(parseRunIOFromPlaintext({ formData, text })).toEqual(expectedResult);
    });

    it("should handle multiple colons in the value correctly", () => {
        const formData = {
            "input-time": true,
        };
        const text = "time: 10:30:00";
        const expectedResult = {
            inputs: { "time": "10:30:00" },
            outputs: {},
        };
        expect(parseRunIOFromPlaintext({ formData, text })).toEqual(expectedResult);
    });

    it("should return empty objects when no inputs or outputs match", () => {
        const formData = {
            "input-location": true,
        };
        const text = "name: John Doe\nresult: Success";
        const expectedResult = {
            inputs: {},
            outputs: {},
        };
        expect(parseRunIOFromPlaintext({ formData, text })).toEqual(expectedResult);
    });

    it("should handle empty text correctly", () => {
        const formData = {
            "input-name": true,
        };
        const text = "";
        const expectedResult = {
            inputs: {},
            outputs: {},
        };
        expect(parseRunIOFromPlaintext({ formData, text })).toEqual(expectedResult);
    });

    it("should handle text with only whitespaces correctly", () => {
        const formData = {
            "input-name": true,
        };
        const text = "    ";
        const expectedResult = {
            inputs: {},
            outputs: {},
        };
        expect(parseRunIOFromPlaintext({ formData, text })).toEqual(expectedResult);
    });

    it("should correctly handle mixed case input and output keys", () => {
        const formData = {
            "input-Name": true,
            "output-Result": true,
        };
        const text = "Name: John Doe\nResult: Success";
        const expectedResult = {
            inputs: { "Name": "John Doe" },
            outputs: { "Result": "Success" },
        };
        expect(parseRunIOFromPlaintext({ formData, text })).toEqual(expectedResult);
    });

    describe("parseRunIOFromPlaintext - Handling Random Text", () => {
        it("should ignore random text that does not form key-value pairs", () => {
            const formData = {
                "input-name": true,
                "output-result": true,
            };
            const text = "Sure! Here is what you asked for:\nname: John Doe\nresult: Success";
            const expectedResult = {
                inputs: { "name": "John Doe" },
                outputs: { "result": "Success" },
            };
            expect(parseRunIOFromPlaintext({ formData, text })).toEqual(expectedResult);
        });

        it("should handle random text between key-value pairs", () => {
            const formData = {
                "input-name": true,
                "output-result": true,
            };
            const text = "name: John Doe\nPlease note the following details are important.\nresult: Success";
            const expectedResult = {
                inputs: { "name": "John Doe" },
                outputs: { "result": "Success" },
            };
            expect(parseRunIOFromPlaintext({ formData, text })).toEqual(expectedResult);
        });

        it("should properly ignore lines without a colon", () => {
            const formData = {
                "input-name": true,
                "output-result": true,
            };
            const text = "name: John Doe\nThis line has no colon\nresult: Success";
            const expectedResult = {
                inputs: { "name": "John Doe" },
                outputs: { "result": "Success" },
            };
            expect(parseRunIOFromPlaintext({ formData, text })).toEqual(expectedResult);
        });

        it("should handle text with mixed legitimate and illegitimate lines", () => {
            const formData = {
                "input-name": true,
                "output-result": true,
            };
            const text = "Here's the info you requested:\nname: John Doe\nrandom statement here\nresult: Success\nEnd of message.";
            const expectedResult = {
                inputs: { "name": "John Doe" },
                outputs: { "result": "Success" },
            };
            expect(parseRunIOFromPlaintext({ formData, text })).toEqual(expectedResult);
        });
    });
});

describe("transformSearchTerms", () => {
    it("should correctly parse plain search terms without numbers", () => {
        const text = "how to cook a chicken\ncooking a chicken";
        const expected = ["how to cook a chicken", "cooking a chicken"];
        expect(transformSearchTerms(text)).toEqual(expected);
    });

    it("should parse numbered search terms and remove numbering", () => {
        const text = "1. how to cook a chicken\n2. cooking a chicken";
        const expected = ["how to cook a chicken", "cooking a chicken"];
        expect(transformSearchTerms(text)).toEqual(expected);
    });

    it("should ignore blank and whitespace-only lines", () => {
        const text = "1. how to cook a chicken\n   \n2. cooking a chicken\n";
        const expected = ["how to cook a chicken", "cooking a chicken"];
        expect(transformSearchTerms(text)).toEqual(expected);
    });

    it("should ignore random introductory or additional text", () => {
        const text = "Here are your search queries:\n1. how to cook a chicken\n2. cooking a chicken\nThat's all!";
        const expected = ["how to cook a chicken", "cooking a chicken"];
        expect(transformSearchTerms(text)).toEqual(expected);
    });

    it("should process terms with numbers not followed by a dot as regular terms", () => {
        const text = "1 how to cook a chicken\n2cooking a chicken";
        const expected = ["1 how to cook a chicken", "2cooking a chicken"];
        expect(transformSearchTerms(text)).toEqual(expected);
    });

    it("should handle empty strings gracefully", () => {
        const text = "";
        const expected = [];
        expect(transformSearchTerms(text)).toEqual(expected);
    });

    it("should trim and process terms with additional spaces before and after numbers", () => {
        const text = "   1.  how to cook a chicken   \n   2.   cooking a chicken   ";
        const expected = ["how to cook a chicken", "cooking a chicken"];
        expect(transformSearchTerms(text)).toEqual(expected);
    });

    it("should handle terms with multiple digits in numbering", () => {
        const text = "10. how to cook a chicken\n20. cooking a chicken";
        const expected = ["how to cook a chicken", "cooking a chicken"];
        expect(transformSearchTerms(text)).toEqual(expected);
    });

    it("should handle non-English characters and special characters in terms", () => {
        const text = "1. cómo cocinar un pollo\n2. cooking a chicken#";
        const expected = ["cómo cocinar un pollo", "cooking a chicken#"];
        expect(transformSearchTerms(text)).toEqual(expected);
    });

    describe("transformSearchTerms - Handling Non-List Lines", () => {
        it("should ignore lines not part of a structured numbered list", () => {
            const text = `Sure! Here are the search queries:
    1. how to cook a chicken
    2. cooking a chicken
    Please follow up if you need more info.`;
            const expected = ["how to cook a chicken", "cooking a chicken"];
            expect(transformSearchTerms(text)).toEqual(expected);
        });

        it("should only process lines in the structured list even with interruptions", () => {
            const text = `1. how to cook a chicken
    Here's something unrelated
    2. cooking a chicken
    End of list`;
            const expected = ["how to cook a chicken", "cooking a chicken"];
            expect(transformSearchTerms(text)).toEqual(expected);
        });

        it("should ignore lines that contain numbers but are not properly formatted as list items", () => {
            const text = `1 how to cook a chicken
    2: Here is an improperly formatted line
    2. cooking a chicken`;
            const expected = ["cooking a chicken"];
            expect(transformSearchTerms(text)).toEqual(expected);
        });

        it("should handle text where valid list items are scattered among random text", () => {
            const text = `Here are your instructions:
    1. how to cook a chicken
    Some random text.
    2. cooking a chicken
    More random text here.`;
            const expected = ["how to cook a chicken", "cooking a chicken"];
            expect(transformSearchTerms(text)).toEqual(expected);
        });

        it("should ignore non-list related text completely, even if it contains colons or numbers", () => {
            const text = `Introduction:
    We have some points to consider:
    1. how to cook a chicken
    2. cooking a chicken
    Conclusion: Thank you for your attention!`;
            const expected = ["how to cook a chicken", "cooking a chicken"];
            expect(transformSearchTerms(text)).toEqual(expected);
        });
    });

});

describe("listToPlaintext", () => {
    it("should convert an array of strings into a single newline-separated string", () => {
        const list = ["First item", "Second item", "Third item"];
        const expected = "First item\nSecond item\nThird item";
        expect(listToPlaintext(list)).toEqual(expected);
    });

    it("should handle an empty list", () => {
        const list = [];
        const expected = "";
        expect(listToPlaintext(list)).toEqual(expected);
    });

    it("should handle a list with empty strings", () => {
        const list = ["", "", ""];
        const expected = "\n\n";
        expect(listToPlaintext(list)).toEqual(expected);
    });

    it("should handle a list with various whitespace", () => {
        const list = ["  ", "\t", " \n "];
        const expected = "  \n\t\n \n ";
        expect(listToPlaintext(list)).toEqual(expected);
    });

    it("should handle special characters in the list items", () => {
        const list = ["Item #1", "Item @2", "Item &3"];
        const expected = "Item #1\nItem @2\nItem &3";
        expect(listToPlaintext(list)).toEqual(expected);
    });
});

describe("listToNumberedPlaintext", () => {
    it("should convert an array of strings into a numbered list separated by newlines", () => {
        const list = ["First item", "Second item", "Third item"];
        const expected = "1. First item\n2. Second item\n3. Third item";
        expect(listToNumberedPlaintext(list)).toEqual(expected);
    });

    it("should handle an empty list", () => {
        const list = [];
        const expected = "";
        expect(listToNumberedPlaintext(list)).toEqual(expected);
    });

    it("should handle a list with empty strings", () => {
        const list = ["", "", ""];
        const expected = "1. \n2. \n3. ";
        expect(listToNumberedPlaintext(list)).toEqual(expected);
    });

    it("should number correctly with large lists", () => {
        const list = new Array(100).fill("Item");
        const expected = Array.from({ length: 100 }, (v, i) => `${i + 1}. Item`).join("\n");
        expect(listToNumberedPlaintext(list)).toEqual(expected);
    });

    it("should handle special characters and maintain correct numbering", () => {
        const list = ["Item #1", "Item @2", "Item &3"];
        const expected = "1. Item #1\n2. Item @2\n3. Item &3";
        expect(listToNumberedPlaintext(list)).toEqual(expected);
    });
});
