import { timeFrameToPrisma } from "./timeFrameToPrisma";

describe("timeFrameToPrisma", () => {
    const fieldName = "createdAt";

    it("returns correct structure with both before and after defined", () => {
        const timeFrame = { before: new Date("2022-01-01"), after: new Date("2021-01-01") };
        const expected = { [fieldName]: { lte: timeFrame.before, gte: timeFrame.after } };
        expect(timeFrameToPrisma(fieldName, timeFrame)).toEqual(expected);
    });

    it("handles only before defined", () => {
        const timeFrame = { before: new Date("2022-01-01") };
        const expected = { [fieldName]: { lte: timeFrame.before } };
        expect(timeFrameToPrisma(fieldName, timeFrame)).toEqual(expected);
    });

    it("handles only after defined", () => {
        const timeFrame = { after: new Date("2021-01-01") };
        const expected = { [fieldName]: { gte: timeFrame.after } };
        expect(timeFrameToPrisma(fieldName, timeFrame)).toEqual(expected);
    });

    it("returns undefined when neither before nor after is defined", () => {
        const timeFrame = {};
        expect(timeFrameToPrisma(fieldName, timeFrame)).toBeUndefined();
    });

    it("returns undefined for null time frame", () => {
        expect(timeFrameToPrisma(fieldName, null)).toBeUndefined();
    });

    it("returns undefined for undefined time frame", () => {
        expect(timeFrameToPrisma(fieldName)).toBeUndefined();
    });

    it("works with unexpected field names", () => {
        const unexpectedFieldName = "";
        const timeFrame = { before: new Date("2022-01-01"), after: new Date("2021-01-01") };
        const expected = { [unexpectedFieldName]: { lte: timeFrame.before, gte: timeFrame.after } };
        expect(timeFrameToPrisma(unexpectedFieldName, timeFrame)).toEqual(expected);
    });

    it("handles null time frame values - test 1", () => {
        const timeFrame = { before: null, after: null };
        expect(timeFrameToPrisma(fieldName, timeFrame)).toBeUndefined();
    });

    it("handles null time frame values - test 2", () => {
        const timeFrame = { before: null, after: new Date("2021-01-01") };
        const expected = { [fieldName]: { gte: timeFrame.after } };
        expect(timeFrameToPrisma(fieldName, timeFrame)).toEqual(expected);
    });
});
