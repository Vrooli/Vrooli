/* eslint-disable @typescript-eslint/ban-ts-comment */
import { shapeDate } from "./shapeDate";

describe("shapeDate function tests", () => {
    test("valid date within default range", () => {
        const result = shapeDate("2025-06-15");
        expect(result).toBe("2025-06-15T00:00:00.000Z");
    });

    test("valid date at minimum boundary", () => {
        const result = shapeDate("2023-01-01");
        expect(result).toBe("2023-01-01T00:00:00.000Z");
    });

    test("valid date at maximum boundary", () => {
        const result = shapeDate("2099-12-31");
        expect(result).toBe("2099-12-31T00:00:00.000Z");
    });

    test("invalid date string", () => {
        const result = shapeDate("invalid-date");
        expect(result).toBeNull();
    });

    test("non-string parameters", () => {
        // @ts-ignore: Testing runtime scenario
        const resultNumber = shapeDate(123);
        expect(resultNumber).toBeNull();
        // @ts-ignore: Testing runtime scenario
        const resultBoolean = shapeDate(true);
        expect(resultBoolean).toBeNull();
        // @ts-ignore: Testing runtime scenario
        const resultObject = shapeDate({});
        expect(resultObject).toBeNull();
        // @ts-ignore: Testing runtime scenario
        const resultArray = shapeDate([]);
        expect(resultArray).toBeNull();
        // @ts-ignore: Testing runtime scenario
        const resultNull = shapeDate(null);
        expect(resultNull).toBeNull();
        // @ts-ignore: Testing runtime scenario
        const resultUndefined = shapeDate(undefined);
        expect(resultUndefined).toBeNull();
    });

    test("date before minimum date", () => {
        const result = shapeDate("2022-12-31");
        expect(result).toBeNull();
    });

    test("date after maximum date", () => {
        const result = shapeDate("2100-01-02");
        expect(result).toBeNull();
    });

    test("valid date with custom min and max dates", () => {
        const minDate = new Date("2020-01-01");
        const maxDate = new Date("2024-01-01");
        const result = shapeDate("2022-06-15", minDate, maxDate);
        expect(result).toBe("2022-06-15T00:00:00.000Z");
    });

    test("date at custom minimum boundary", () => {
        const minDate = new Date("2020-01-01");
        const maxDate = new Date("2024-01-01");
        const result = shapeDate("2020-01-01", minDate, maxDate);
        expect(result).toBe("2020-01-01T00:00:00.000Z");
    });

    test("date at custom maximum boundary", () => {
        const minDate = new Date("2020-01-01");
        const maxDate = new Date("2024-01-01");
        const result = shapeDate("2024-01-01", minDate, maxDate);
        expect(result).toBe("2024-01-01T00:00:00.000Z");
    });

    test("date string format check", () => {
        const result = shapeDate("2025-06-15");
        expect(result).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d{3}Z$/);
    });
});
