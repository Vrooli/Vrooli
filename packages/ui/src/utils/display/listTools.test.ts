import { DUMMY_ID } from "@local/shared";
import { defaultYou, getYou, placeholderColor, placeholderColors, simpleHash } from "./listTools";

describe("getYou function", () => {
    beforeEach(() => {
        jest.clearAllMocks();
    });

    it("should return defaultYou when object is null or undefined", () => {
        expect(getYou(null)).toEqual(defaultYou);
        expect(getYou(undefined)).toEqual(defaultYou);
    });

    it("should handle DUMMY_ID case", () => {
        const dummyObject = {
            __typename: "Api",
            id: DUMMY_ID,
        } as const;
        const result = getYou(dummyObject);
        expect(result).toEqual({
            ...Object.keys(defaultYou).reduce((acc, key) => ({ ...acc, [key]: typeof defaultYou[key] === "boolean" ? false : defaultYou[key] }), {}),
            canDelete: true,
        });
    });

    it("should get permissions from object.you", () => {
        const object = {
            __typename: "RoutineVersion",
            you: {
                canUpdate: true,
                canDelete: false,
            },
        } as const;
        const result = getYou(object);
        expect(result.canUpdate).toBe(true);
        expect(result.canDelete).toBe(false);
    });

    it("should get permissions from object.root.you if not in object.you", () => {
        const object = {
            __typename: "CodeVersion",
            you: {},
            root: {
                __typename: "Code",
                you: {
                    canUpdate: true,
                    canDelete: false,
                },
            },
        } as const;
        const result = getYou(object);
        expect(result.canUpdate).toBe(true);
        expect(result.canDelete).toBe(false);
    });

    it("should default to false if permission is not found", () => {
        const object = {
            __typename: "Standard",
            you: {},
        } as const;
        const result = getYou(object);
        expect(result.canUpdate).toBe(false);
        expect(result.canDelete).toBe(false);
    });

    it("should filter invalid actions", () => {
        const object = {
            __typename: "InvalidType",
            you: {
                canBookmark: true,
                canComment: true,
                canCopy: true,
                canDelete: true,
                canReact: true,
                canReport: true,
            },
        } as any;
        const result = getYou(object);
        expect(result.canBookmark).toBe(false);
        expect(result.canComment).toBe(false);
        expect(result.canCopy).toBe(false);
        expect(result.canDelete).toBe(false);
        expect(result.canReact).toBe(false);
        expect(result.canReport).toBe(false);
    });
});

describe("simpleHash", () => {
    it("should return the same hash for the same input", () => {
        const input = "testString";
        expect(simpleHash(input)).toBe(simpleHash(input));
    });

    it("should return different hashes for different inputs", () => {
        const input1 = "testString1";
        const input2 = "testString2";
        expect(simpleHash(input1)).not.toBe(simpleHash(input2));
    });

    it("should handle an empty string", () => {
        expect(simpleHash("")).toBe(0);
    });
});

describe("placeholderColor", () => {
    it("should return a valid color pair from the array", () => {
        const result = placeholderColor();
        expect(placeholderColors).toContainEqual(result);
    });

    it("should return consistent results for the same seed", () => {
        const seed = "consistentSeed";
        const result1 = placeholderColor(seed);
        const result2 = placeholderColor(seed);
        expect(result1).toEqual(result2);
    });

    it("should return different results for different seeds", () => {
        const seed1 = "seed1";
        const seed2 = "seed2";
        const result1 = placeholderColor(seed1);
        const result2 = placeholderColor(seed2);
        expect(result1).not.toEqual(result2);
    });

    it("should handle numeric seeds", () => {
        const seed = 12345;
        const result1 = placeholderColor(seed);
        const result2 = placeholderColor(seed);
        expect(result1).toEqual(result2);
    });

    it("should return different results without a seed", () => {
        let isDifferent = false;
        // Try multiple times to prevent failure due to random chance
        for (let i = 0; i < 10; i++) {
            const result1 = placeholderColor();
            const result2 = placeholderColor();
            if (result1 !== result2) {
                isDifferent = true;
                break;
            }
        }
        expect(isDifferent).toBe(true);
    });
});
