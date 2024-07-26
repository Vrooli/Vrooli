import { RoutineType } from "@local/shared";
import { defaultConfigFormInputMap, defaultConfigFormOutputMap, parseSchemaInput, parseSchemaOutput } from "./runUtils";

describe("Schema Parsing Functions", () => {
    describe("parseSchemaInput", () => {
        test("parses valid JSON string correctly", () => {
            const input = JSON.stringify({ containers: [{ id: 1 }], elements: [{ id: 1 }] });
            const expected = { containers: [{ id: 1 }], elements: [{ id: 1 }] };
            expect(parseSchemaInput(input, RoutineType.Informational, console)).toEqual(expected);
        });

        test("falls back to default input schema on invalid JSON string", () => {
            const input = "{ containers: [}";
            expect(parseSchemaInput(input, RoutineType.Api, console)).toEqual(defaultConfigFormInputMap[RoutineType.Api]());
        });

        test("handles non-string, non-object inputs by returning default input schema", () => {
            const input = 12345;
            expect(parseSchemaInput(input, RoutineType.Generate, console)).toEqual(defaultConfigFormInputMap[RoutineType.Generate]());
        });

        test("handles valid object input without parsing", () => {
            const input = { containers: [{ id: 2 }], elements: [{ id: 2 }] };
            expect(parseSchemaInput(input, RoutineType.SmartContract, console)).toEqual(input);
        });

        test("adds missing containers and elements as empty arrays", () => {
            const input = "{}";
            const expected = { containers: [], elements: [] };
            expect(parseSchemaInput(input, RoutineType.Code, console)).toEqual(expected);
        });

        test("replaces non-array containers and elements with empty arrays", () => {
            const input = JSON.stringify({ containers: "not-an-array", elements: "not-an-array" });
            const expected = { containers: [], elements: [] };
            expect(parseSchemaInput(input, RoutineType.Data, console)).toEqual(expected);
        });

        test("handles malformed object inputs", () => {
            const input = { someRandomKey: 123 };
            const expected = { containers: [], elements: [], someRandomKey: 123 };
            expect(parseSchemaInput(input, RoutineType.Informational, console)).toEqual(expected);
        });
    });

    describe("parseSchemaOutput", () => {
        test("parses valid JSON string correctly", () => {
            const input = JSON.stringify({ containers: [{ id: 1 }], elements: [{ id: 1 }] });
            const expected = { containers: [{ id: 1 }], elements: [{ id: 1 }] };
            expect(parseSchemaOutput(input, RoutineType.Generate, console)).toEqual(expected);
        });

        test("falls back to default output schema on invalid JSON string", () => {
            const input = "{ containers: [}";
            expect(parseSchemaOutput(input, RoutineType.Generate, console)).toEqual(defaultConfigFormOutputMap[RoutineType.Generate]());
        });

        test("handles non-string, non-object inputs by returning default output schema", () => {
            const input = 12345;
            expect(parseSchemaOutput(input, RoutineType.Action, console)).toEqual(defaultConfigFormOutputMap[RoutineType.Action]());
        });

        test("handles valid object input without parsing", () => {
            const input = { containers: [{ id: 2 }], elements: [{ id: 2 }] };
            expect(parseSchemaOutput(input, RoutineType.Data, console)).toEqual(input);
        });

        test("adds missing containers and elements as empty arrays", () => {
            const input = "{}";
            const expected = { containers: [], elements: [] };
            expect(parseSchemaOutput(input, RoutineType.Informational, console)).toEqual(expected);
        });

        test("replaces non-array containers and elements with empty arrays", () => {
            const input = JSON.stringify({ containers: "not-an-array", elements: "not-an-array" });
            const expected = { containers: [], elements: [] };
            expect(parseSchemaOutput(input, RoutineType.Code, console)).toEqual(expected);
        });

        test("handles malformed object inputs", () => {
            const input = { someRandomKey: 123 };
            const expected = { containers: [], elements: [], someRandomKey: 123 };
            expect(parseSchemaOutput(input, RoutineType.SmartContract, console)).toEqual(expected);
        });
    });
});