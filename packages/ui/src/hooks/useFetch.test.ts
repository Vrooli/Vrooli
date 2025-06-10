/* eslint-disable @typescript-eslint/ban-ts-comment */
import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { useFetch, useLazyFetch, uiPathToApi } from "./useFetch.js";
import { fetchData } from "../api/fetchData.js";
import { ServerResponseParser } from "../api/responseParser.js";

// Mock dependencies with minimal overhead
vi.mock("../api/fetchData.js", () => ({
    fetchData: vi.fn(),
}));

vi.mock("../api/responseParser.js", () => ({
    ServerResponseParser: {
        displayErrors: vi.fn(),
    },
}));

// Mock useDebounce to execute immediately without delay
vi.mock("./useDebounce.js", () => ({
    useDebounce: vi.fn((callback) => [callback, vi.fn()]),
}));

const mockFetchData = vi.mocked(fetchData);
const mockDisplayErrors = vi.mocked(ServerResponseParser.displayErrors);

describe("useFetch", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    describe("basic functionality", () => {
        it("should initialize with default loading state", () => {
            const { result } = renderHook(() => useFetch({
                endpoint: "/test",
                inputs: {},
            }));

            expect(result.current.loading).toBe(false);
            expect(result.current.data).toBeUndefined();
            expect(result.current.errors).toBeUndefined();
            expect(typeof result.current.refetch).toBe("function");
        });

        it("should not make request when endpoint is undefined", () => {
            renderHook(() => useFetch({
                endpoint: undefined,
                inputs: {},
            }));

            expect(mockFetchData).not.toHaveBeenCalled();
        });

        it("should use default method GET when no method specified", async () => {
            const mockResponse = {
                data: { id: 1, name: "test" },
                errors: undefined,
                version: "1.0.0",
                __fetchTimestamp: Date.now(),
            };
            
            mockFetchData.mockResolvedValueOnce(mockResponse);

            renderHook(() => useFetch({
                endpoint: "/test",
                inputs: {},
            }));

            expect(mockFetchData).toHaveBeenCalledWith({
                endpoint: "/test",
                inputs: {},
                method: "GET",
                options: {},
                omitRestBase: false,
            });
        });
    });

    describe("data fetching", () => {
        it("should handle successful response with data", async () => {
            const mockData = { id: 1, name: "test item", active: true };
            const mockResponse = {
                data: mockData,
                errors: undefined,
                version: "1.0.0",
                __fetchTimestamp: Date.now(),
            };

            mockFetchData.mockResolvedValueOnce(mockResponse);

            const { result } = renderHook(() => useFetch({
                endpoint: "/test",
                inputs: {},
            }));

            // Wait for fetch to complete
            await act(async () => {
                await new Promise(resolve => setTimeout(resolve, 0));
            });

            expect(result.current.data).toEqual(mockData);
            expect(result.current.errors).toBeUndefined();
            expect(result.current.loading).toBe(false);
        });

        it("should handle response with errors", async () => {
            const mockErrors = [
                { message: "Validation failed", trace: "001" },
            ];
            const mockResponse = {
                data: undefined,
                errors: mockErrors,
                version: "1.0.0",
                __fetchTimestamp: Date.now(),
            };

            mockFetchData.mockResolvedValueOnce(mockResponse);

            const { result } = renderHook(() => useFetch({
                endpoint: "/test",
                inputs: {},
            }));

            await act(async () => {
                await new Promise(resolve => setTimeout(resolve, 0));
            });

            expect(result.current.errors).toEqual(mockErrors);
            expect(mockDisplayErrors).toHaveBeenCalledWith(mockErrors);
        });

        it("should handle missing endpoint gracefully", async () => {
            const { result } = renderHook(() => useFetch({
                endpoint: undefined,
                inputs: {},
            }));

            const response = await result.current.refetch();
            // @ts-ignore: Testing error response structure
            expect(response.errors).toEqual([{
                trace: "0693",
                message: "No endpoint provided to useLazyFetch",
            }]);
        });
    });

    describe("HTTP options", () => {
        it("should pass through HTTP method correctly", async () => {
            const mockResponse = {
                data: { success: true },
                errors: undefined,
                version: "1.0.0",
                __fetchTimestamp: Date.now(),
            };

            mockFetchData.mockResolvedValueOnce(mockResponse);

            renderHook(() => useFetch({
                endpoint: "/test",
                inputs: { name: "test" },
                method: "POST",
            }));

            expect(mockFetchData).toHaveBeenCalledWith({
                endpoint: "/test",
                inputs: { name: "test" },
                method: "POST",
                options: {},
                omitRestBase: false,
            });
        });

        it("should handle omitRestBase option", async () => {
            const mockResponse = {
                data: { success: true },
                errors: undefined,
                version: "1.0.0",
                __fetchTimestamp: Date.now(),
            };

            mockFetchData.mockResolvedValueOnce(mockResponse);

            renderHook(() => useFetch({
                endpoint: "/auth/login",
                inputs: {},
                omitRestBase: true,
            }));

            expect(mockFetchData).toHaveBeenCalledWith({
                endpoint: "/auth/login",
                inputs: {},
                method: "GET",
                options: {},
                omitRestBase: true,
            });
        });
    });

    describe("refetch functionality", () => {
        it("should allow manual refetch with new inputs", async () => {
            const mockResponse1 = {
                data: { id: 1 },
                errors: undefined,
                version: "1.0.0",
                __fetchTimestamp: Date.now(),
            };

            const mockResponse2 = {
                data: { id: 2 },
                errors: undefined,
                version: "1.0.0",
                __fetchTimestamp: Date.now(),
            };

            mockFetchData.mockResolvedValueOnce(mockResponse1);
            mockFetchData.mockResolvedValueOnce(mockResponse2);

            const { result } = renderHook(() => useFetch({
                endpoint: "/test",
                inputs: { id: 1 },
            }));

            await act(async () => {
                await new Promise(resolve => setTimeout(resolve, 0));
            });
            
            expect(result.current.data).toEqual(mockResponse1.data);

            await act(async () => {
                await result.current.refetch({ id: 2 });
            });

            expect(result.current.data).toEqual(mockResponse2.data);
            expect(mockFetchData).toHaveBeenCalledTimes(2);
        });
    });
});

describe("useLazyFetch", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    describe("basic functionality", () => {
        it("should initialize with default state and return fetch function", () => {
            const [getData, state] = renderHook(() => useLazyFetch({
                endpoint: "/test",
                inputs: {},
            })).result.current;

            expect(typeof getData).toBe("function");
            expect(state.loading).toBe(false);
            expect(state.data).toBeUndefined();
            expect(state.errors).toBeUndefined();
        });

        it("should not make automatic request on mount", () => {
            renderHook(() => useLazyFetch({
                endpoint: "/test",
                inputs: {},
            }));

            expect(mockFetchData).not.toHaveBeenCalled();
        });
    });

    describe("manual data fetching", () => {
        it("should fetch data when getData is called", async () => {
            const mockResponse = {
                data: { id: 1, name: "test" },
                errors: undefined,
                version: "1.0.0",
                __fetchTimestamp: Date.now(),
            };

            mockFetchData.mockResolvedValueOnce(mockResponse);

            const { result } = renderHook(() => useLazyFetch({
                endpoint: "/test",
                inputs: {},
            }));

            const [getData] = result.current;

            await act(async () => {
                const response = await getData();
                expect(response.data).toEqual(mockResponse.data);
            });

            expect(mockFetchData).toHaveBeenCalledWith({
                endpoint: "/test",
                inputs: {},
                method: "GET",
                options: {},
            });

            expect(result.current[1].data).toEqual(mockResponse.data);
            expect(result.current[1].loading).toBe(false);
        });
    });

    describe("error handling", () => {
        it("should handle errors and display them", async () => {
            const mockErrors = [{ message: "Server error", trace: "500" }];
            const mockResponse = {
                data: undefined,
                errors: mockErrors,
                version: "1.0.0",
                __fetchTimestamp: Date.now(),
            };

            mockFetchData.mockResolvedValueOnce(mockResponse);

            const { result } = renderHook(() => useLazyFetch({
                endpoint: "/test",
                inputs: {},
            }));

            const [getData] = result.current;

            await act(async () => {
                const response = await getData();
                expect(response.errors).toEqual(mockErrors);
            });

            expect(result.current[1].errors).toEqual(mockErrors);
            expect(mockDisplayErrors).toHaveBeenCalledWith(mockErrors);
        });

        it("should handle missing endpoint", async () => {
            const { result } = renderHook(() => useLazyFetch({
                endpoint: undefined,
                inputs: {},
            }));

            const [getData] = result.current;
            const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});

            await act(async () => {
                const response = await getData();
                expect(response.errors).toEqual([{
                    trace: "0692",
                    message: "No endpoint provided to useLazyFetch",
                }]);
            });

            expect(result.current[1].errors).toEqual([{
                trace: "0692",
                message: "No endpoint provided to useLazyFetch",
            }]);
            expect(consoleSpy).toHaveBeenCalledWith("No endpoint provided to useLazyFetch");

            consoleSpy.mockRestore();
        });
    });

    describe("timestamp-based response ordering", () => {
        it("should ignore outdated responses", async () => {
            const { result } = renderHook(() => useLazyFetch({
                endpoint: "/test",
                inputs: {},
            }));

            const [getData] = result.current;

            const olderResponse = {
                data: { id: 1, name: "old" },
                errors: undefined,
                version: "1.0.0",
                __fetchTimestamp: 1000,
            };

            const newerResponse = {
                data: { id: 2, name: "new" },
                errors: undefined,
                version: "1.0.0",
                __fetchTimestamp: 2000,
            };

            mockFetchData.mockResolvedValueOnce(newerResponse);
            
            await act(async () => {
                await getData();
            });

            expect(result.current[1].data).toEqual(newerResponse.data);

            mockFetchData.mockResolvedValueOnce(olderResponse);
            
            const consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

            await act(async () => {
                await getData();
            });

            expect(result.current[1].data).toEqual(newerResponse.data);
            expect(consoleSpy).toHaveBeenCalled();

            consoleSpy.mockRestore();
        });
    });

    describe("input handling", () => {
        it("should update inputs when provided to getData", async () => {
            const mockResponse = {
                data: { success: true },
                errors: undefined,
                version: "1.0.0",
                __fetchTimestamp: Date.now(),
            };

            mockFetchData.mockResolvedValueOnce(mockResponse);

            const { result } = renderHook(() => useLazyFetch({
                endpoint: "/test",
                inputs: { original: "value" },
            }));

            const [getData] = result.current;

            await act(async () => {
                await getData({ updated: "input" });
            });

            expect(mockFetchData).toHaveBeenCalledWith({
                endpoint: "/test",
                inputs: { updated: "input" },
                method: "GET",
                options: {},
            });
        });
    });
});

describe("uiPathToApi", () => {
    it("should convert UI path to API path with default base", () => {
        expect(uiPathToApi("/users")).toBe("/api/v2/users");
        expect(uiPathToApi("/u/@john")).toBe("/api/v2/u/@john");
        expect(uiPathToApi("/search?q=test")).toBe("/api/v2/search?q=test");
    });

    it("should convert UI path to API path with custom base", () => {
        expect(uiPathToApi("/users", "/api/v1")).toBe("/api/v1/users");
        expect(uiPathToApi("/items", "/custom/api")).toBe("/custom/api/items");
    });

    it("should handle empty path", () => {
        expect(uiPathToApi("")).toBe("/api/v2");
        expect(uiPathToApi("", "/custom")).toBe("/custom");
    });
});