// AI_CHECK: TYPE_SAFETY=phase2-tests | LAST: 2025-07-04 - Type safety verification tests
import { describe, it, expect } from "vitest";
import { ModelMap } from "../models/base/index.js";
import { 
    hasProperty, 
    hasStringProperty, 
    hasId, 
    hasTypename, 
    hasUserId,
    hasStatus,
    isAuthData,
    isObject,
    extractOwnerId,
} from "../utils/typeGuards.js";

describe("Type Guards", () => {
    describe("hasProperty", () => {
        it("should correctly identify properties", () => {
            const obj = { foo: "bar", baz: undefined };
            expect(hasProperty(obj, "foo")).toBe(true);
            expect(hasProperty(obj, "baz")).toBe(true);
            expect(hasProperty(obj, "missing")).toBe(false);
        });
    });

    describe("hasStringProperty", () => {
        it("should only return true for string properties", () => {
            const obj = { str: "hello", num: 42, nul: null };
            expect(hasStringProperty(obj, "str")).toBe(true);
            expect(hasStringProperty(obj, "num")).toBe(false);
            expect(hasStringProperty(obj, "nul")).toBe(false);
            expect(hasStringProperty(obj, "missing")).toBe(false);
        });
    });

    describe("hasId", () => {
        it("should detect objects with string id", () => {
            expect(hasId({ id: "123" })).toBe(true);
            expect(hasId({ id: 123 })).toBe(false);
            expect(hasId({ no_id: "123" })).toBe(false);
        });
    });

    describe("hasTypename", () => {
        it("should detect objects with __typename", () => {
            expect(hasTypename({ __typename: "User" })).toBe(true);
            expect(hasTypename({ __typename: 123 })).toBe(false);
            expect(hasTypename({ typename: "User" })).toBe(false);
        });
    });

    describe("isAuthData", () => {
        it("should validate auth data structure", () => {
            expect(isAuthData({ __typename: "User", id: "123" })).toBe(true);
            expect(isAuthData({ __typename: "User", id: 123 })).toBe(false);
            expect(isAuthData({ id: "123" })).toBe(false);
            expect(isAuthData({ __typename: "User" })).toBe(false);
        });
    });

    describe("extractOwnerId", () => {
        it("should extract userId", () => {
            expect(extractOwnerId({ userId: "123" })).toBe("123");
        });

        it("should extract startedById", () => {
            expect(extractOwnerId({ startedById: "456" })).toBe("456");
        });

        it("should extract userData.id", () => {
            expect(extractOwnerId({ userData: { id: "789" } })).toBe("789");
        });

        it("should prioritize userId over others", () => {
            expect(extractOwnerId({ 
                userId: "123", 
                startedById: "456",
                userData: { id: "789" },
            })).toBe("123");
        });

        it("should return null for invalid data", () => {
            expect(extractOwnerId(null)).toBe(null);
            expect(extractOwnerId(undefined)).toBe(null);
            expect(extractOwnerId({})).toBe(null);
            expect(extractOwnerId({ userId: 123 })).toBe(null);
            expect(extractOwnerId({ userData: { id: 123 } })).toBe(null);
        });
    });
});

describe("ModelMap Type Safety", () => {
    beforeAll(async () => {
        await ModelMap.init();
    });

    it("should return typed model logic", () => {
        const userLogic = ModelMap.get("User");
        expect(userLogic).toBeDefined();
        expect(userLogic.__typename).toBe("User");
        expect(typeof userLogic.dbTable).toBe("string");
    });

    it("should handle missing models gracefully", () => {
        const logic = ModelMap.get("NonExistentModel" as any, false);
        expect(logic).toBeUndefined();
    });

    it("should throw for missing models when required", () => {
        expect(() => ModelMap.get("NonExistentModel" as any)).toThrow();
    });

    it("should provide type-safe getLogic", () => {
        const { dbTable, validate } = ModelMap.getLogic(["dbTable", "validate"], "User");
        expect(typeof dbTable).toBe("string");
        expect(typeof validate).toBe("function");
        
        const validator = validate();
        expect(validator).toBeDefined();
        expect(typeof validator.isPublic).toBe("function");
    });
});

describe("Type Inference", () => {
    it("should narrow types with guards", () => {
        const data: unknown = { userId: "123", status: "active" };
        
        if (isObject(data)) {
            // TypeScript knows data is Record<string, unknown>
            if (hasUserId(data)) {
                // TypeScript knows data has userId: string
                const id: string = data.userId;
                expect(id).toBe("123");
            }
            
            if (hasStatus(data)) {
                // TypeScript knows data has status: string
                const status: string = data.status;
                expect(status).toBe("active");
            }
        }
    });

    it("should work with ModelMap type inference", () => {
        const logic = ModelMap.get("User");
        // This should compile without type errors
        const validator = logic.validate();
        const displayInfo = logic.display();
        
        expect(validator).toBeDefined();
        expect(displayInfo).toBeDefined();
    });
});
