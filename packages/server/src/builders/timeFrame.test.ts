import { timeFrameToPrisma, timeFrameToSql } from "./timeFrame";

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

describe("timeFrameToSql", () => {
    const fieldName = "created_at";  // Change as needed for additional field tests

    test("returns null when timeFrame is undefined", () => {
        expect(timeFrameToSql(fieldName, undefined)).toBeNull();
    });

    test("returns correct query for after date only", () => {
        const afterDate = new Date("2021-01-01");
        const expectedSeconds = Math.floor(afterDate.getTime() / 1000);
        expect(timeFrameToSql(fieldName, { after: afterDate }))
            .toBe(`EXTRACT(EPOCH FROM t."${fieldName}") >= ${expectedSeconds}`);
    });

    test("returns correct query for before date only", () => {
        const beforeDate = new Date("2022-01-01");
        const expectedSeconds = Math.floor(beforeDate.getTime() / 1000);
        expect(timeFrameToSql(fieldName, { before: beforeDate }))
            .toBe(`EXTRACT(EPOCH FROM t."${fieldName}") <= ${expectedSeconds}`);
    });

    test("returns correct query for both after and before dates", () => {
        const afterDate = new Date("2021-01-01");
        const beforeDate = new Date("2022-01-01");
        const afterSeconds = Math.floor(afterDate.getTime() / 1000);
        const beforeSeconds = Math.floor(beforeDate.getTime() / 1000);
        expect(timeFrameToSql(fieldName, { after: afterDate, before: beforeDate }))
            .toBe(`EXTRACT(EPOCH FROM t."${fieldName}") >= ${afterSeconds} AND EXTRACT(EPOCH FROM t."${fieldName}") <= ${beforeSeconds}`);
    });

    test("handles different field names", () => {
        const afterDate = new Date("2021-01-01");
        const expectedSeconds = Math.floor(afterDate.getTime() / 1000);
        const updatedFieldName = "updated_at";
        expect(timeFrameToSql(updatedFieldName, { after: afterDate }))
            .toBe(`EXTRACT(EPOCH FROM t."${updatedFieldName}") >= ${expectedSeconds}`);
    });
});

