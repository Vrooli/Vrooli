import { getFunctionName, safeParse, safeStringify } from "./utils";

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

describe("safeStringify", () => {
    describe("primitive types", () => {
        test("string", () => {
            const obj = "hello";
            const expected = "\"hello\"";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("string with special characters", () => {
            const obj = "hello, world! 🌍\"Chicken coop\" \nBeep boop";
            const expected = "\"hello, world! 🌍\\\"Chicken coop\\\" \\nBeep boop\"";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("positive number", () => {
            const obj = 42;
            const expected = "42";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("negative number", () => {
            const obj = -42;
            const expected = "-42";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("decimal number", () => {
            const obj = 3.14;
            const expected = "3.14";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("NaN", () => {
            const obj = NaN;
            const expected = "__NAN__";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("Infinity", () => {
            const obj = Infinity;
            const expected = "__INFINITY__";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("boolean true", () => {
            const obj = true;
            const expected = "true";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("boolean false", () => {
            const obj = false;
            const expected = "false";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("null", () => {
            const obj = null;
            const expected = "null";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("undefined", () => {
            const obj = undefined;
            const expected = undefined;
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("BigInt", () => {
            const obj = BigInt(42);
            const expected = "__BIGINT__42";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("Symbol", () => {
            const obj = Symbol("test");
            const expected = undefined;
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("function", () => {
            // eslint-disable-next-line func-style
            const obj = function test() { };
            const expected = undefined;
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("Date", () => {
            const obj = new Date("2023-08-05T00:00:00.000Z");
            const expected = `__DATE__${obj.toISOString()}`;
            expect(safeStringify(obj)).toEqual(expected);
        });
    });

    describe("Map", () => {
        test("With primitives", () => {
            const obj = new Map<any, any>([["key1", "value1"], [42, "answer"]]);
            const expected = "__MAP__[[\"key1\",\"value1\"],[42,\"answer\"]]";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("With objects", () => {
            const obj = new Map<any, any>([["key1", { a: 1 }], [{ b: 2 }, "value2"]]);
            const expected = "__MAP__[[\"key1\",{\"a\":1}],[{\"b\":2},\"value2\"]]";
            expect(safeStringify(obj)).toEqual(expected);
        });

    });

    describe("placeholders", () => {
        test("__NAN__", () => {
            const obj = "__NAN__";
            const expected = "\"__NAN__\"";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("__INFINITY__", () => {
            const obj = "__INFINITY__";
            const expected = "\"__INFINITY__\"";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("__UNDEFINED__", () => {
            const obj = "__UNDEFINED__";
            const expected = "\"__UNDEFINED__\"";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("__BIGINT__", () => {
            const obj = "__BIGINT__42";
            const expected = "\"__BIGINT__42\"";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("__DATE__", () => {
            const obj = "__DATE__2023-08-05T00:00:00.000Z";
            const expected = "\"__DATE__2023-08-05T00:00:00.000Z\"";
            expect(safeStringify(obj)).toEqual(expected);
        });
    });

    describe("objects", () => {
        test("empty object", () => {
            const obj = {};
            const expected = "{}";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("object with one property", () => {
            const obj = { a: 1 };
            const expected = "{\"a\":1}";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("object with multiple properties", () => {
            const obj = { a: 1, b: "two", c: true };
            const expected = "{\"a\":1,\"b\":\"two\",\"c\":true}";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("object with nested object", () => {
            const obj = { a: { b: 2 } };
            const expected = "{\"a\":{\"b\":2}}";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("object with circular reference", () => {
            const obj: Record<string, any> = { a: 1 };
            obj.b = obj;
            const expected = "{\"a\":1}"; // Circular reference should be omitted
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("object with Date", () => {
            const obj = { date: new Date("2023-08-05T00:00:00.000Z") };
            const expected = "{\"date\":__DATE__2023-08-05T00:00:00.000Z}";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("object with undefined value", () => {
            const obj = { a: undefined };
            const expected = "{}";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("object with null value", () => {
            const obj = { a: null };
            const expected = "{\"a\":null}";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("object with NaN value", () => {
            const obj = { a: NaN };
            const expected = "{\"a\":__NAN__}";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("object with Infinity value", () => {
            const obj = { a: Infinity };
            const expected = "{\"a\":__INFINITY__}";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("object with function", () => {
            const obj = { a() { } };
            const expected = "{}";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("object with symbol", () => {
            const obj = { a: Symbol("test") };
            const expected = "{}";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("object with BigInt", () => {
            const obj = { a: BigInt(42) };
            const expected = "{\"a\":__BIGINT__42}";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("object with primitive array", () => {
            const obj = { a: [1, 2, 3] };
            const expected = "{\"a\":[1,2,3]}";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("object with object array", () => {
            const obj = { a: [{ b: 2 }, { c: 3 }] };
            const expected = "{\"a\":[{\"b\":2},{\"c\":3}]}";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("object with mixed array", () => {
            const obj = { a: [1, { b: 2 }, 3] };
            const expected = "{\"a\":[1,{\"b\":2},3]}";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("object with nested arrays", () => {
            const obj = { a: [[1, 2], [3, 4]] };
            const expected = "{\"a\":[[1,2],[3,4]]}";
            expect(safeStringify(obj)).toEqual(expected);
        });
    });

    describe("arrays", () => {
        test("empty array", () => {
            const obj = [];
            const expected = "[]";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("array with one element", () => {
            const obj = [1];
            const expected = "[1]";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("array with multiple elements", () => {
            const obj = [1, "two", true];
            const expected = "[1,\"two\",true]";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("array with nested array", () => {
            const obj = [[1, 2]];
            const expected = "[[1,2]]";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("array with circular reference", () => {
            const obj: any[] = [1];
            obj.push(obj);
            const expected = "[1]"; // Circular reference should be omitted
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("array with Date", () => {
            const obj = [new Date("2023-08-05T00:00:00.000Z")];
            const expected = "[__DATE__2023-08-05T00:00:00.000Z]";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("array with undefined value", () => {
            const obj = [undefined];
            const expected = "[undefined]";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("array with null value", () => {
            const obj = [null];
            const expected = "[null]";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("array with NaN value", () => {
            const obj = [NaN];
            const expected = "[__NAN__]";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("array with Infinity value", () => {
            const obj = [Infinity];
            const expected = "[__INFINITY__]";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("array with function", () => {
            const obj = [function test() { }];
            const expected = "[null]";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("array with symbol", () => {
            const obj = [Symbol("test")];
            const expected = "[null]";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("array with BigInt", () => {
            const obj = [BigInt(42)];
            const expected = "[__BIGINT__42]";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("array with primitive object", () => {
            const obj = [{ a: 1 }];
            const expected = "[{\"a\":1}]";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("array with object array", () => {
            const obj = [[{ a: 1 }, { b: 2 }]];
            const expected = "[[{\"a\":1},{\"b\":2}]]";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("array with mixed array", () => {
            const obj = [1, { a: 2 }, 3];
            const expected = "[1,{\"a\":2},3]";
            expect(safeStringify(obj)).toEqual(expected);
        });
        test("array with nested arrays", () => {
            const obj = [[1, 2], [3, 4]];
            const expected = "[[1,2],[3,4]]";
            expect(safeStringify(obj)).toEqual(expected);
        });
    });
});

describe("safeParse", () => {
    test("should parse basic JSON correctly", () => {
        const json = "{\"a\":1,\"b\":\"string\",\"c\":true,\"d\":null}";
        expect(safeParse(json)).toEqual({ a: 1, b: "string", c: true, d: null });
    });

    test("should restore Date objects from special format", () => {
        const date = new Date("2023-08-05T00:00:00.000Z");
        const json = `{"date":"__DATE__${date.toISOString()}"}`;
        expect(safeParse(json).date.toISOString()).toEqual(date.toISOString());
    });

    test("should parse \"undefined\" as undefined", () => {
        const json = "{\"a\":\"__UNDEFINED__\"}";
        expect(safeParse(json)).toEqual({});
    });

    test("should leave \"[Circular]\" as string", () => {
        const json = "{\"self\":\"[Circular]\"}";
        expect(safeParse(json)).toEqual({ self: "[Circular]" });
    });

    // Add tests for more complex objects, arrays, primitives, strings with __DATE__ or other signifiers, strings with weird characters, etc.
});
