import { describe, it, expect, vi } from "vitest";
import { stringifyObject, parseObject, LATEST_CONFIG_VERSION, type StringifyMode } from "./utils.js";
import { type PassableLogger } from "../../consts/commonTypes.js";

describe("LATEST_CONFIG_VERSION", () => {
    it("should export the correct version constant", () => {
        expect(LATEST_CONFIG_VERSION).toBe("1.0");
    });
});

describe("stringifyObject", () => {
    describe("json mode", () => {
        it("should stringify a simple object", () => {
            const obj = { name: "test", value: 123 };
            const result = stringifyObject(obj, "json");
            expect(result).toBe('{"name":"test","value":123}');
        });

        it("should stringify nested objects", () => {
            const obj = {
                user: {
                    id: 1,
                    profile: {
                        name: "John",
                        active: true
                    }
                }
            };
            const result = stringifyObject(obj, "json");
            expect(result).toBe('{"user":{"id":1,"profile":{"name":"John","active":true}}}');
        });

        it("should stringify arrays", () => {
            const obj = { items: [1, 2, 3], tags: ["tag1", "tag2"] };
            const result = stringifyObject(obj, "json");
            expect(result).toBe('{"items":[1,2,3],"tags":["tag1","tag2"]}');
        });

        it("should handle null and undefined values", () => {
            const obj = { nullValue: null, undefinedValue: undefined };
            const result = stringifyObject(obj, "json");
            expect(result).toBe('{"nullValue":null}');
        });

        it("should handle empty objects and arrays", () => {
            const obj = { emptyObj: {}, emptyArray: [] };
            const result = stringifyObject(obj, "json");
            expect(result).toBe('{"emptyObj":{},"emptyArray":[]}');
        });

        it("should handle special characters in strings", () => {
            const obj = { message: "Hello \"world\"\nNew line\tTab" };
            const result = stringifyObject(obj, "json");
            expect(result).toBe('{"message":"Hello \\"world\\"\\nNew line\\tTab"}');
        });

        it("should handle numbers, booleans, and strings", () => {
            const obj = {
                number: 42,
                float: 3.14,
                negative: -100,
                boolean: true,
                falseBool: false,
                string: "hello world"
            };
            const result = stringifyObject(obj, "json");
            expect(result).toBe('{"number":42,"float":3.14,"negative":-100,"boolean":true,"falseBool":false,"string":"hello world"}');
        });
    });

    describe("unsupported modes", () => {
        it("should throw error for unsupported stringify mode", () => {
            const obj = { test: "value" };
            expect(() => stringifyObject(obj, "yaml" as StringifyMode)).toThrow("Unsupported stringify mode: yaml");
        });

        it("should throw error for xml mode", () => {
            const obj = { test: "value" };
            expect(() => stringifyObject(obj, "xml" as StringifyMode)).toThrow("Unsupported stringify mode: xml");
        });
    });
});

describe("parseObject", () => {
    let mockLogger: PassableLogger;

    beforeEach(() => {
        mockLogger = {
            error: vi.fn(),
            info: vi.fn(),
            warn: vi.fn()
        };
    });

    describe("json mode - successful parsing", () => {
        it("should parse a simple JSON string", () => {
            const jsonString = '{"name":"test","value":123}';
            const result = parseObject(jsonString, "json", mockLogger);
            expect(result).toEqual({ name: "test", value: 123 });
            expect(mockLogger.error).not.toHaveBeenCalled();
        });

        it("should parse nested objects", () => {
            const jsonString = '{"user":{"id":1,"profile":{"name":"John","active":true}}}';
            const result = parseObject(jsonString, "json", mockLogger);
            expect(result).toEqual({
                user: {
                    id: 1,
                    profile: {
                        name: "John",
                        active: true
                    }
                }
            });
            expect(mockLogger.error).not.toHaveBeenCalled();
        });

        it("should parse arrays", () => {
            const jsonString = '{"items":[1,2,3],"tags":["tag1","tag2"]}';
            const result = parseObject(jsonString, "json", mockLogger);
            expect(result).toEqual({ items: [1, 2, 3], tags: ["tag1", "tag2"] });
            expect(mockLogger.error).not.toHaveBeenCalled();
        });

        it("should handle null values", () => {
            const jsonString = '{"nullValue":null}';
            const result = parseObject(jsonString, "json", mockLogger);
            expect(result).toEqual({ nullValue: null });
            expect(mockLogger.error).not.toHaveBeenCalled();
        });

        it("should handle empty objects and arrays", () => {
            const jsonString = '{"emptyObj":{},"emptyArray":[]}';
            const result = parseObject(jsonString, "json", mockLogger);
            expect(result).toEqual({ emptyObj: {}, emptyArray: [] });
            expect(mockLogger.error).not.toHaveBeenCalled();
        });

        it("should handle special characters", () => {
            const jsonString = '{"message":"Hello \\"world\\"\\nNew line\\tTab"}';
            const result = parseObject(jsonString, "json", mockLogger);
            expect(result).toEqual({ message: "Hello \"world\"\nNew line\tTab" });
            expect(mockLogger.error).not.toHaveBeenCalled();
        });

        it("should handle various data types", () => {
            const jsonString = '{"number":42,"float":3.14,"negative":-100,"boolean":true,"falseBool":false,"string":"hello world"}';
            const result = parseObject(jsonString, "json", mockLogger);
            expect(result).toEqual({
                number: 42,
                float: 3.14,
                negative: -100,
                boolean: true,
                falseBool: false,
                string: "hello world"
            });
            expect(mockLogger.error).not.toHaveBeenCalled();
        });
    });

    describe("json mode - parsing errors", () => {
        it("should return null and log error for invalid JSON", () => {
            const invalidJson = '{"name": invalid}';
            const result = parseObject(invalidJson, "json", mockLogger);
            expect(result).toBeNull();
            expect(mockLogger.error).toHaveBeenCalledWith(
                expect.stringMatching(/^Error parsing data:/)
            );
        });

        it("should handle malformed JSON with missing quotes", () => {
            const invalidJson = '{name: "test"}';
            const result = parseObject(invalidJson, "json", mockLogger);
            expect(result).toBeNull();
            expect(mockLogger.error).toHaveBeenCalled();
        });

        it("should handle JSON with trailing commas", () => {
            const invalidJson = '{"name": "test",}';
            const result = parseObject(invalidJson, "json", mockLogger);
            expect(result).toBeNull();
            expect(mockLogger.error).toHaveBeenCalled();
        });

        it("should handle incomplete JSON", () => {
            const invalidJson = '{"name": "test"';
            const result = parseObject(invalidJson, "json", mockLogger);
            expect(result).toBeNull();
            expect(mockLogger.error).toHaveBeenCalled();
        });

        it("should handle empty string", () => {
            const result = parseObject("", "json", mockLogger);
            expect(result).toBeNull();
            expect(mockLogger.error).toHaveBeenCalled();
        });

        it("should handle non-JSON string", () => {
            const result = parseObject("this is not json", "json", mockLogger);
            expect(result).toBeNull();
            expect(mockLogger.error).toHaveBeenCalled();
        });

        it("should include error details in log message", () => {
            const invalidJson = '{"invalid": }';
            parseObject(invalidJson, "json", mockLogger);
            
            expect(mockLogger.error).toHaveBeenCalledWith(
                expect.stringMatching(/^Error parsing data: .*/)
            );
            // Verify the logged message contains error information
            const loggedMessage = (mockLogger.error as any).mock.calls[0][0];
            expect(loggedMessage).toContain("Error parsing data:");
        });
    });

    describe("unsupported modes", () => {
        it("should return null and log error for unsupported parse mode", () => {
            const data = "some data";
            const result = parseObject(data, "yaml" as StringifyMode, mockLogger);
            expect(result).toBeNull();
            expect(mockLogger.error).toHaveBeenCalledWith("Error parsing data: {}");
        });

        it("should return null and log error for xml mode", () => {
            const data = "<xml>data</xml>";
            const result = parseObject(data, "xml" as StringifyMode, mockLogger);
            expect(result).toBeNull();
            expect(mockLogger.error).toHaveBeenCalledWith("Error parsing data: {}");
        });
    });

    describe("edge cases", () => {
        it("should handle whitespace-only string", () => {
            const result = parseObject("   \n\t  ", "json", mockLogger);
            expect(result).toBeNull();
            expect(mockLogger.error).toHaveBeenCalled();
        });

        it("should parse valid JSON with extra whitespace", () => {
            const jsonString = '  \n  {"name": "test"}  \n  ';
            const result = parseObject(jsonString, "json", mockLogger);
            expect(result).toEqual({ name: "test" });
            expect(mockLogger.error).not.toHaveBeenCalled();
        });
    });

    describe("type safety", () => {
        it("should return typed result for valid JSON", () => {
            interface TestType {
                name: string;
                count: number;
            }
            
            const jsonString = '{"name":"test","count":123}';
            const result = parseObject<TestType>(jsonString, "json", mockLogger);
            
            // TypeScript should infer correct type
            if (result) {
                expect(result.name).toBe("test");
                expect(result.count).toBe(123);
            }
        });
    });
});

describe("integration tests", () => {
    let mockLogger: PassableLogger;

    beforeEach(() => {
        mockLogger = {
            error: vi.fn(),
            info: vi.fn(),
            warn: vi.fn()
        };
    });

    it("should roundtrip stringify and parse successfully", () => {
        const originalObj = {
            id: 1,
            name: "Test Object",
            metadata: {
                created: "2023-01-01",
                tags: ["tag1", "tag2"],
                active: true,
                config: {
                    setting1: "value1",
                    setting2: 42,
                    setting3: null
                }
            }
        };

        const stringified = stringifyObject(originalObj, "json");
        const parsed = parseObject(stringified, "json", mockLogger);

        expect(parsed).toEqual(originalObj);
        expect(mockLogger.error).not.toHaveBeenCalled();
    });

    it("should handle complex nested structures", () => {
        const complexObj = {
            users: [
                { id: 1, name: "John", roles: ["admin", "user"] },
                { id: 2, name: "Jane", roles: ["user"] }
            ],
            settings: {
                theme: "dark",
                notifications: {
                    email: true,
                    push: false,
                    frequency: "daily"
                }
            },
            statistics: {
                totalUsers: 2,
                activeUsers: 1,
                averageAge: 25.5
            }
        };

        const stringified = stringifyObject(complexObj, "json");
        const parsed = parseObject(stringified, "json", mockLogger);

        expect(parsed).toEqual(complexObj);
        expect(mockLogger.error).not.toHaveBeenCalled();
    });
});