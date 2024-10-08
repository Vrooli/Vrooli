import { getFunctionDetails } from "./utils";

describe("getFunctionDetails", () => {
    describe("synchonous functions", () => {
        it("extracts name from traditional function declaration", () => {
            const funcStr = "function example() {}";
            const { functionName, isAsync } = getFunctionDetails(funcStr);
            expect(functionName).toBe("example");
            expect(isAsync).toBe(false);
        });

        it("extracts name from function expression", () => {
            const funcStr = "const example = function() {}";
            const { functionName, isAsync } = getFunctionDetails(funcStr);
            expect(functionName).toBe("example");
            expect(isAsync).toBe(false);
        });

        it("extracts name from arrow function", () => {
            const funcStr = "const example = () => {}";
            const { functionName, isAsync } = getFunctionDetails(funcStr);
            expect(functionName).toBe("example");
            expect(isAsync).toBe(false);
        });

        it("returns null for anonymous functions", () => {
            const funcStr = "function() {}";
            const { functionName, isAsync } = getFunctionDetails(funcStr);
            expect(functionName).toBeNull();
            expect(isAsync).toBe(false);
        });

        it("ignores block comments - test 1", () => {
            const funcStr = "/* function commentedOut() {} */ const example = function() {}";
            const { functionName, isAsync } = getFunctionDetails(funcStr);
            expect(functionName).toBe("example");
            expect(isAsync).toBe(false);
        });

        it("ignores block comments - test 2", () => {
            const funcStr = `/**
        * function commentedOut() { }
        * @param {string} param*
        * @returns {void}
        */
        const example = function() {}`;
            const { functionName, isAsync } = getFunctionDetails(funcStr);
            expect(functionName).toBe("example");
            expect(isAsync).toBe(false);
        });

        it("ignores inline comments", () => {
            const funcStr = "// function commentedOut() {}\nconst example = function() {}";
            const { functionName, isAsync } = getFunctionDetails(funcStr);
            expect(functionName).toBe("example");
            expect(isAsync).toBe(false);
        });

        it("handles complex cases with nested comments", () => {
            const funcStr = "/* Example with // nested comment */ const example = function() {} // end";
            const { functionName, isAsync } = getFunctionDetails(funcStr);
            expect(functionName).toBe("example");
            expect(isAsync).toBe(false);
        });

        it("doesn't think the function is async if 'async' is in a comment", () => {
            const funcStr = "/* async function commentedOut() {} */ const example = function() {}";
            const { functionName, isAsync } = getFunctionDetails(funcStr);
            expect(functionName).toBe("example");
            expect(isAsync).toBe(false);
        });

        it("doesn't think the function is async if 'async' is in the function name", () => {
            const funcStr = "const asyncFunc = function() {}";
            const { functionName, isAsync } = getFunctionDetails(funcStr);
            expect(functionName).toBe("asyncFunc");
            expect(isAsync).toBe(false);
        });
    });

    describe("asynchronous functions", () => {
        it("extracts name from async function", () => {
            const funcStr = "async function example() {}";
            const { functionName, isAsync } = getFunctionDetails(funcStr);
            expect(functionName).toBe("example");
            expect(isAsync).toBe(true);
        });

        it("extracts name from async function expression", () => {
            const funcStr = "const example = async function() {}";
            const { functionName, isAsync } = getFunctionDetails(funcStr);
            expect(functionName).toBe("example");
            expect(isAsync).toBe(true);
        });

        it("extracts name from async arrow function", () => {
            const funcStr = "const example = async () => {}";
            const { functionName, isAsync } = getFunctionDetails(funcStr);
            expect(functionName).toBe("example");
            expect(isAsync).toBe(true);
        });

        it("ignores async function keyword in comments", () => {
            const funcStr = "/* async function commentedOut() {} */ const example = async function() {}";
            const { functionName, isAsync } = getFunctionDetails(funcStr);
            expect(functionName).toBe("example");
            expect(isAsync).toBe(true);
        });

        it("ignores async arrow function keyword in comments", () => {
            const funcStr = "/* async commentedOut() => {} */ const example = async () => {}";
            const { functionName, isAsync } = getFunctionDetails(funcStr);
            expect(functionName).toBe("example");
            expect(isAsync).toBe(true);
        });
    });
});

describe("getFunctionDetails - Advanced Cases", () => {
    it("handles functions with newlines and spaces", () => {
        const funcStr = `
        // function should not be named
        const   exampleFunc   = function() {
          console.log("Doing something");
        }
      `;
        const { functionName, isAsync } = getFunctionDetails(funcStr);
        expect(functionName).toBe("exampleFunc");
        expect(isAsync).toBe(false);
    });

    it("handles functions with unusual characters in names", () => {
        const funcStr = "const $weird_name$ = () => {}";
        const { functionName, isAsync } = getFunctionDetails(funcStr);
        expect(functionName).toBe("$weird_name$");
        expect(isAsync).toBe(false);
    });

    it("ignores content within the functions", () => {
        const funcStr = "const example = function() { if(true) { return false; } }";
        const { functionName, isAsync } = getFunctionDetails(funcStr);
        expect(functionName).toBe("example");
        expect(isAsync).toBe(false);
    });

    it("extracts names correctly despite leading/trailing spaces", () => {
        const funcStr = "    const example    = function() {};  ";
        const { functionName, isAsync } = getFunctionDetails(funcStr);
        expect(functionName).toBe("example");
        expect(isAsync).toBe(false);
    });

    it("handles multiline declarations with complex inner content", () => {
        const funcStr = `const example = function() {
        // This is a comment
        console.log("Hello, world!");
        function innerFunction() {} // should not affect outer function name
      }`;
        const { functionName, isAsync } = getFunctionDetails(funcStr);
        expect(functionName).toBe("example");
        expect(isAsync).toBe(false);
    });

    it("correctly identifies function names in deeply nested expressions", () => {
        const funcStr = "const outer = function() { const inner = function() { const deepInner = () => {}; }; };";
        const { functionName, isAsync } = getFunctionDetails(funcStr);
        expect(functionName).toBe("outer");
        expect(isAsync).toBe(false);
    });

    it("handles async arrow functions with complex parameters", () => {
        const funcStr = "const fetchWithTimeout = async (url, { timeout = 500 } = {}) => {}";
        const { functionName, isAsync } = getFunctionDetails(funcStr);
        expect(functionName).toBe("fetchWithTimeout");
        expect(isAsync).toBe(true);
    });

    it("returns null for immediately invoked function expressions (IIFE)", () => {
        const funcStr = "(function() { console.log(\"IIFE\"); })()";
        const { functionName, isAsync } = getFunctionDetails(funcStr);
        expect(functionName).toBeNull();
        expect(isAsync).toBe(false);
    });

    it("recognizes names despite special characters in comments and strings", () => {
        const funcStr = `const test = function() {
        // This shouldn't "break" anything: const broken = () => {};
        return 'Sample text with tricky characters: }{()=;';
      };`;
        const { functionName, isAsync } = getFunctionDetails(funcStr);
        expect(functionName).toBe("test");
        expect(isAsync).toBe(false);
    });
});
