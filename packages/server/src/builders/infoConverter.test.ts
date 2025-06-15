/* eslint-disable @typescript-eslint/ban-ts-comment */
import { expect, describe, it } from "vitest";
import { CountFields } from "./infoConverter.js";

describe("CountFields", () => {
    describe("addToData", () => {
        it("returns obj unmodified when countFields is undefined", () => {
            const obj = { commentsCount: true } as any;
            expect(CountFields.addToData(obj, undefined)).toEqual(obj);
        });

        it("handles empty obj correctly", () => {
            const countFields = { commentsCount: true } as const;
            expect(CountFields.addToData({}, countFields)).toEqual({});
        });

        it("does not modify obj when countFields is empty", () => {
            const obj = { commentsCount: true };
            expect(CountFields.addToData(obj, {})).toEqual(obj);
        });

        it("does not modify obj if specified count fields do not exist", () => {
            const obj = { likesCount: true };
            const countFields = { commentsCount: true } as const;
            expect(CountFields.addToData(obj, countFields)).toEqual(obj);
        });

        it("handles multiple count fields correctly", () => {
            const obj = { commentsCount: true, reportsCount: true };
            const countFields = { commentsCount: true, reportsCount: true } as const;
            const expected = { _count: { select: { comments: true, reports: true } } };
            expect(CountFields.addToData(obj, countFields)).toEqual(expected);
        });
    });
});
