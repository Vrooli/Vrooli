import { addCountFields } from "./addCountFields";

describe("addCountFields", () => {
    it("processes valid obj and countFields correctly", () => {
        const obj = { commentsCount: 5 };
        const countFields = { commentsCount: true } as const;
        const expected = { _count: { comments: true } };
        expect(addCountFields(obj, countFields)).toEqual(expected);
    });

    it("returns obj unmodified when countFields is undefined", () => {
        const obj = { commentsCount: 5 };
        expect(addCountFields(obj, undefined)).toEqual(obj);
    });

    it("handles empty obj correctly", () => {
        const countFields = { commentsCount: true } as const;
        expect(addCountFields({}, countFields)).toEqual({});
    });

    it("does not modify obj when countFields is empty", () => {
        const obj = { commentsCount: 5 };
        expect(addCountFields(obj, {})).toEqual(obj);
    });

    it("does not modify obj if specified count fields do not exist", () => {
        const obj = { likesCount: 3 };
        const countFields = { commentsCount: true } as const;
        expect(addCountFields(obj, countFields)).toEqual(obj);
    });

    it("adds to existing _count property without overwriting", () => {
        const obj = { _count: { likes: true }, commentsCount: 5 };
        const countFields = { commentsCount: true } as const;
        const expected = { _count: { likes: true, comments: true } };
        expect(addCountFields(obj, countFields)).toEqual(expected);
    });

    it("handles multiple count fields correctly", () => {
        const obj = { commentsCount: 5, reportsCount: 2 };
        const countFields = { commentsCount: true, reportsCount: true } as const;
        const expected = { _count: { comments: true, reports: true } };
        expect(addCountFields(obj, countFields)).toEqual(expected);
    });

    it("ignores count fields not ending in \"Count\"", () => {
        const obj = { commentData: 5 };
        const countFields = { commentData: true } as const;
        expect(addCountFields(obj, countFields)).toEqual(obj);
    });
});
