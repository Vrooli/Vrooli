import { getFunctionName } from "./utils";

describe("getFunctionName", () => {
    it("extracts name from traditional function declaration", () => {
        const funcStr = "function example() {}";
        expect(getFunctionName(funcStr)).toBe("example");
    });

    it("extracts name from function expression", () => {
        const funcStr = "const example = function() {}";
        expect(getFunctionName(funcStr)).toBe("example");
    });

    it("extracts name from arrow function", () => {
        const funcStr = "const example = () => {}";
        expect(getFunctionName(funcStr)).toBe("example");
    });

    it("extracts name from async function expression", () => {
        const funcStr = "const example = async function() {}";
        expect(getFunctionName(funcStr)).toBe("example");
    });

    it("extracts name from async arrow function", () => {
        const funcStr = "const example = async () => {}";
        expect(getFunctionName(funcStr)).toBe("example");
    });

    it("returns null for anonymous functions", () => {
        const funcStr = "function() {}";
        expect(getFunctionName(funcStr)).toBeNull();
    });

    it("ignores block comments - test 1", () => {
        const funcStr = "/* function commentedOut() {} */ const example = function() {}";
        expect(getFunctionName(funcStr)).toBe("example");
    });

    it("ignores block comments - test 2", () => {
        const funcStr = `/**
        * function commentedOut() { }
        * @param {string} param*
        * @returns {void}
        */
        const example = function() {}`;
        expect(getFunctionName(funcStr)).toBe("example");
    });

    it("ignores inline comments", () => {
        const funcStr = "// function commentedOut() {}\nconst example = function() {}";
        expect(getFunctionName(funcStr)).toBe("example");
    });

    it("handles complex cases with nested comments", () => {
        const funcStr = "/* Example with // nested comment */ const example = function() {} // end";
        expect(getFunctionName(funcStr)).toBe("example");
    });
});

describe("getFunctionName - Advanced Cases", () => {
    it("handles functions with newlines and spaces", () => {
        const funcStr = `
        // function should not be named
        const   exampleFunc   = function() {
          console.log("Doing something");
        }
      `;
        expect(getFunctionName(funcStr)).toBe("exampleFunc");
    });

    it("handles functions with unusual characters in names", () => {
        const funcStr = "const $weird_name$ = () => {}";
        expect(getFunctionName(funcStr)).toBe("$weird_name$");
    });

    it("ignores content within the functions", () => {
        const funcStr = "const example = function() { if(true) { return false; } }";
        expect(getFunctionName(funcStr)).toBe("example");
    });

    it("extracts names correctly despite leading/trailing spaces", () => {
        const funcStr = "    const example    = function() {};  ";
        expect(getFunctionName(funcStr)).toBe("example");
    });

    it("handles multiline declarations with complex inner content", () => {
        const funcStr = `const example = function() {
        // This is a comment
        console.log("Hello, world!");
        function innerFunction() {} // should not affect outer function name
      }`;
        expect(getFunctionName(funcStr)).toBe("example");
    });

    it("correctly identifies function names in deeply nested expressions", () => {
        const funcStr = "const outer = function() { const inner = function() { const deepInner = () => {}; }; };";
        expect(getFunctionName(funcStr)).toBe("outer");
    });

    it("handles async arrow functions with complex parameters", () => {
        const funcStr = "const fetchWithTimeout = async (url, { timeout = 500 } = {}) => {}";
        expect(getFunctionName(funcStr)).toBe("fetchWithTimeout");
    });

    it("returns null for immediately invoked function expressions (IIFE)", () => {
        const funcStr = "(function() { console.log(\"IIFE\"); })()";
        expect(getFunctionName(funcStr)).toBeNull();
    });

    it("recognizes names despite special characters in comments and strings", () => {
        const funcStr = `const test = function() {
        // This shouldn't "break" anything: const broken = () => {};
        return 'Sample text with tricky characters: }{()=;';
      };`;
        expect(getFunctionName(funcStr)).toBe("test");
    });
});

