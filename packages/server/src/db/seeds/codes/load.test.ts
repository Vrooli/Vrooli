import { expect } from "chai";
import { parseDocstrings, splitFunctions } from "./load.js";

describe("splitFunctions", () => {
    it("should split multiple functions correctly", () => {
        const input = `export function foo() {
    return 'foo';
}
export function bar(param) {
    return 'bar';
}`;
        const expected = [
            `export function foo() {
    return 'foo';
}`,
            `export function bar(param) {
    return 'bar';
}`,
        ];
        expect(splitFunctions(input)).to.deep.equal(expected);
    });

    it("should handle no function correctly", () => {
        const input = `const foo = 'Hello';
console.log(foo);`;
        const expected = [];
        expect(splitFunctions(input)).to.deep.equal(expected);
    });

    it("should handle a single function correctly", () => {
        const input = `export function baz() {
    return 'baz';
}`;
        const expected = [
            `export function baz() {
    return 'baz';
}`,
        ];
        expect(splitFunctions(input)).to.deep.equal(expected);
    });

    it("should ignore default exports and other export types", () => {
        const input = `export default function defaultFunc() {
    return 'default';
}
export const arrowFunc = () => 'arrow';
export function namedFunc() {
    return 'named';
}`;
        const expected = [
            `export function namedFunc() {
    return 'named';
}`,
        ];
        expect(splitFunctions(input)).to.deep.equal(expected);
    });

    it("should handle non-exported functions by ignoring them", () => {
        const input = `function internalFunc() {
    return 'internal';
}
export function exportedFunc() {
    return 'exported';
}`;
        const expected = [
            `export function exportedFunc() {
    return 'exported';
}`,
        ];
        expect(splitFunctions(input)).to.deep.equal(expected);
    });

    it("should handle code before functions correctly", () => {
        const input = `// Comment at the top
const setup = 'stuff';
export function first() {
    return 'first';
}`;
        const expected = [
            `export function first() {
    return 'first';
}`,
        ];
        expect(splitFunctions(input)).to.deep.equal(expected);
    });

    it("should not split on strings containing 'export function' (known limitation)", () => {
        const input = `export function tricky() {
    const str = \`
export function fake() {}
    \`;
    return str;
}
export function another() {
    return 'another';
}`;
        const expected = [
            `export function tricky() {
    const str = \`
export function fake() {}
    \`;
    return str;
}`,
            `export function another() {
    return 'another';
}`,
        ];
        expect(splitFunctions(input)).to.deep.equal(expected);
    });

    it("should extract both async and non-async functions correctly", () => {
        const input = `export async function asyncFunc() {
    return 'async';
}
export function syncFunc() {
    return 'sync';
}
`;
        const expected = [
            `export async function asyncFunc() {
    return 'async';
}`,
            `export function syncFunc() {
    return 'sync';
}`,
        ];
        expect(splitFunctions(input)).to.deep.equal(expected);
    });

    it("should handle functions using the 'const' keyword correctly", () => {
        const input = `export const asyncFunc = async () => {
    return 'async';
}

export const syncFunc = () => 'sync';
`;
        const expected = [
            `export const asyncFunc = async () => {
    return 'async';
}`,
            `export const syncFunc = () => 'sync';`,
        ];
        expect(splitFunctions(input)).to.deep.equal(expected);
    });

    it("should support functions with docstrings", () => {
        const input = `/**
 * @name Example Function
 * @description This is an example function.
 */
export function example() {
    return 'example';
}

/**
 * @name Example Function Two
 * @description This is the second example function.
 */
export function exampleTwo() {
    return 'two';
}
`;
        const expected = [
            `/**
 * @name Example Function
 * @description This is an example function.
 */
export function example() {
    return 'example';
}`,
            `/**
 * @name Example Function Two
 * @description This is the second example function.
 */
export function exampleTwo() {
    return 'two';
}`,
        ];
        expect(splitFunctions(input)).to.deep.equal(expected);
    });

    it("should handle internal functions correctly", () => {
        const input = `function outerFunc() {
    function innerFunc() {
        return 'inner';
    }
    return innerFunc();
}
`;
        const expected = [
            `function outerFunc() {
    function innerFunc() {
        return 'inner';
    }
    return innerFunc();
}`,
        ];
        expect(splitFunctions(input)).to.deep.equal(expected);
    });

    //TODO update tests to add more functions with parameters and typescript annotations
});

describe("parseDocstrings", () => {
    it("should correctly extract function metadata when proper docstrings are present", () => {
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
        expect(parseDocstrings(tsContent)).to.deep.equal(expected);
    });

    it("should handle cases where docstrings are missing annotations", () => {
        const tsContent = `
/**
 * This function has no annotations.
 */
export function noAnnotations() {
    return 'nothing';
}`;
        const expected = [];
        expect(parseDocstrings(tsContent)).to.deep.equal(expected);
    });

    it("should return an empty array when no docstrings are present", () => {
        const tsContent = `export function noDocstring() {
    return 'no docstring';
}`;
        const expected = [];
        expect(parseDocstrings(tsContent)).to.deep.equal(expected);
    });

    it("should handle improper docstring formatting gracefully", () => {
        const tsContent = `/*
* @name Badly Formatted
* @description The start and end of this docstring are incorrect.
*/
export function badlyFormatted() {
    return 'bad format';
}`;
        const expected = [];
        expect(parseDocstrings(tsContent)).to.deep.equal(expected);
    });
});