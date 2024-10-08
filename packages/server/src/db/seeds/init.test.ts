import { parseDocstrings, splitFunctions } from "./init";

describe("splitFunctions", () => {
    test("should split multiple functions correctly", () => {
        const input = `export function foo() {
    return 'foo';
}
export function bar() {
    return 'bar';
}`;
        const expected = [
            `export function foo() {
    return 'foo';
}`,
            `export function bar() {
    return 'bar';
}`,
        ];
        expect(splitFunctions(input)).toEqual(expected);
    });

    test("should handle no function correctly", () => {
        const input = `const foo = 'Hello';
console.log(foo);`;
        const expected = [];
        expect(splitFunctions(input)).toEqual(expected);
    });

    test("should handle a single function correctly", () => {
        const input = `export function baz() {
    return 'baz';
}`;
        const expected = [
            `export function baz() {
    return 'baz';
}`,
        ];
        expect(splitFunctions(input)).toEqual(expected);
    });
});

describe("parseDocstrings", () => {
    test("should correctly extract function metadata when proper docstrings are present", () => {
        const tsContent = `
/**
 * @name Example Function One
 * @description This is the first example function.
 */
export function exampleOne() {
    return 'one';
}

/**
 * @name Example Function Two
 * @description This is the second example function.
 */
export function exampleTwo() {
    return 'two';
}`;
        const expected = [
            { name: "Example Function One", description: "This is the first example function.", functionName: "exampleOne" },
            { name: "Example Function Two", description: "This is the second example function.", functionName: "exampleTwo" },
        ];
        expect(parseDocstrings(tsContent)).toEqual(expected);
    });

    test("should handle cases where docstrings are missing annotations", () => {
        const tsContent = `
/**
 * This function has no annotations.
 */
export function noAnnotations() {
    return 'nothing';
}`;
        const expected = [];
        expect(parseDocstrings(tsContent)).toEqual(expected);
    });

    test("should return an empty array when no docstrings are present", () => {
        const tsContent = `export function noDocstring() {
    return 'no docstring';
}`;
        const expected = [];
        expect(parseDocstrings(tsContent)).toEqual(expected);
    });

    test("should handle improper docstring formatting gracefully", () => {
        const tsContent = `/*
* @name Badly Formatted
* @description The start and end of this docstring are incorrect.
*/
export function badlyFormatted() {
    return 'bad format';
}`;
        const expected = [];
        expect(parseDocstrings(tsContent)).toEqual(expected);
    });
});
