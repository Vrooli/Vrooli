import { parseSchemaInputOutput } from "./routineUtils";

const defaultSchema = {
    containers: [],
    elements: [],
};

describe("parseSchemaInputOutput function tests", () => {
    test("parses valid JSON string correctly", () => {
        const input = JSON.stringify({ containers: [{ id: 1 }], elements: [{ id: 1 }] });
        const expected = { containers: [{ id: 1 }], elements: [{ id: 1 }] };
        expect(parseSchemaInputOutput(input, defaultSchema)).toEqual(expected);
    });

    test("falls back to default schema on invalid JSON string", () => {
        const input = "{ containers: [}";
        expect(parseSchemaInputOutput(input, defaultSchema)).toEqual(defaultSchema);
    });

    test("handles non-string, non-object inputs by returning default schema", () => {
        const input = 12345; // Non-object, non-string input
        expect(parseSchemaInputOutput(input, defaultSchema)).toEqual(defaultSchema);
    });

    test("handles valid object input without parsing", () => {
        const input = { containers: [{ id: 2 }], elements: [{ id: 2 }] };
        expect(parseSchemaInputOutput(input, defaultSchema)).toEqual(input);
    });

    test("adds missing containers and elements as empty arrays", () => {
        const input = "{}";
        const expected = { containers: [], elements: [] };
        expect(parseSchemaInputOutput(input, defaultSchema)).toEqual(expected);
    });

    test("replaces non-array containers and elements with empty arrays", () => {
        const input = JSON.stringify({ containers: "not-an-array", elements: "not-an-array" });
        const expected = { containers: [], elements: [] };
        expect(parseSchemaInputOutput(input, defaultSchema)).toEqual(expected);
    });

    test("handles malformed object inputs", () => {
        const input = { someRandomKey: 123 };
        const expected = { containers: [], elements: [], someRandomKey: 123 }; // Keeps the malformed key
        expect(parseSchemaInputOutput(input, defaultSchema)).toEqual(expected);
    });
});
