/* eslint-disable @typescript-eslint/ban-ts-comment */
import { applyDataTransform, fetchDataUsingUrl } from "./useObjectFromUrl";

describe("fetchDataUsingUrl", () => {
    let mockGetData;
    let mockOnError;

    beforeEach(() => {
        mockGetData = jest.fn();
        mockOnError = jest.fn();
    });

    test("should call getData with handle if it exists", () => {
        const params = { handle: "test-handle" };
        const result = fetchDataUsingUrl(params, mockGetData, mockOnError, true);
        expect(mockGetData).toHaveBeenCalledWith({ handle: "test-handle" }, { onError: mockOnError, displayError: true });
        expect(result).toBe(true);
    });

    test("should call getData with handleRoot if handle is not present but handleRoot exists", () => {
        const params = { handleRoot: "root-handle" };
        const result = fetchDataUsingUrl(params, mockGetData, mockOnError, false);
        expect(mockGetData).toHaveBeenCalledWith({ handleRoot: "root-handle" }, { onError: mockOnError, displayError: false });
        expect(result).toBe(true);
    });

    test("should call getData with id if neither handle nor handleRoot exist", () => {
        const params = { id: "test-id" };
        const result = fetchDataUsingUrl(params, mockGetData, mockOnError, true);
        expect(mockGetData).toHaveBeenCalledWith({ id: "test-id" }, { onError: mockOnError, displayError: true });
        expect(result).toBe(true);
    });

    test("should call getData with idRoot if handle, handleRoot, and id do not exist but idRoot does", () => {
        const params = { idRoot: "root-id" };
        const result = fetchDataUsingUrl(params, mockGetData, mockOnError, true);
        expect(mockGetData).toHaveBeenCalledWith({ idRoot: "root-id" }, { onError: mockOnError, displayError: true });
        expect(result).toBe(true);
    });

    test("should return false if none of the params are present", () => {
        const params = {};
        const result = fetchDataUsingUrl(params, mockGetData, mockOnError, true);
        expect(mockGetData).not.toHaveBeenCalled();
        expect(result).toBe(false);
    });

    test("should not display error when displayError is undefined", () => {
        const params = { handle: "test-handle" };
        fetchDataUsingUrl(params, mockGetData, mockOnError, undefined);
        expect(mockGetData).toHaveBeenCalledWith({ handle: "test-handle" }, { onError: mockOnError, displayError: undefined });
    });

    test("should handle other data in params", () => {
        const params = { handle: "test-handle", otherData: "other" };
        fetchDataUsingUrl(params, mockGetData, mockOnError, true);
        expect(mockGetData).toHaveBeenCalledWith({ handle: "test-handle" }, { onError: mockOnError, displayError: true });
    });
});

describe("applyDataTransform", () => {
    // Test applying a transformation function to the data
    it("applies a transformation function to the data", () => {
        const data = { name: "John", age: 30 };
        const transform = jest.fn(data => ({
            ...data,
            age: data.age + 10,
        }));

        // @ts-ignore: Testing runtime scenario
        const result = applyDataTransform(data, transform);
        expect(transform).toHaveBeenCalledWith(data);
        expect(result).toEqual({ name: "John", age: 40 });
    });

    // Test the function when no transform function is provided
    it("returns the original data if no transform function is provided", () => {
        const data = { name: "Alice", age: 25 };
        // @ts-ignore: Testing runtime scenario
        const result = applyDataTransform(data, undefined);
        expect(result).toEqual(data);
    });

    // Test the function with null data input
    it("handles null data correctly", () => {
        const transform = jest.fn(data => ({ ...data, verified: true }));
        // @ts-ignore: Testing runtime scenario
        const result = applyDataTransform(null, transform);
        expect(transform).not.toHaveBeenCalled();
        expect(result).toBeNull();
    });

    // Test the function with empty object data input
    it("handles empty object data correctly", () => {
        const data = {};
        const transform = jest.fn(data => ({ ...data, isEmpty: true }));
        const result = applyDataTransform(data, transform);
        expect(transform).toHaveBeenCalledWith(data);
        expect(result).toEqual({ isEmpty: true });
    });

    // Test the function with complex nested data structures
    it("handles complex nested data structures", () => {
        const data = { user: { name: "Bob", details: { age: 20, location: "City" } } };
        const transform = jest.fn(data => ({
            ...data,
            user: {
                ...data.user,
                details: {
                    ...data.user.details,
                    age: data.user.details.age + 5,
                },
            },
        }));

        // @ts-ignore: Testing runtime scenario
        const result = applyDataTransform(data, transform);
        expect(transform).toHaveBeenCalledWith(data);
        expect(result).toEqual({ user: { name: "Bob", details: { age: 25, location: "City" } } });
    });

    // Test the function when data is an array
    it("handles array data correctly", () => {
        const data = [{ name: "Cathy" }, { name: "Dave" }];
        const transform = jest.fn(data => data.map(item => ({ ...item, present: true })));
        // @ts-ignore: Testing runtime scenario
        const result = applyDataTransform(data, transform);
        expect(transform).toHaveBeenCalledWith(data);
        expect(result).toEqual([{ name: "Cathy", present: true }, { name: "Dave", present: true }]);
    });

    // Test the function when the transform function is non-functional (returns undefined)
    it("handles non-functional transform results correctly", () => {
        const data = { key: "value" };
        const transform = jest.fn(() => undefined);
        // @ts-ignore: Testing runtime scenario
        const result = applyDataTransform(data, transform);
        expect(transform).toHaveBeenCalledWith(data);
        expect(result).toBeUndefined();
    });
});
