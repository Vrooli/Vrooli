/* eslint-disable @typescript-eslint/ban-ts-comment */
import { renderHook, waitFor } from "@testing-library/react";
import { act } from "react";
import { describe, it, beforeEach, afterEach, vi } from "vitest";
import { expect } from "chai";
import { useFetch, useLazyFetch, uiPathToApi } from "./useFetch.js";
import { fetchData } from "../api/fetchData.js";
import { ServerResponseParser } from "../api/responseParser.js";
import { useDebounce } from "./useDebounce.js";

// Mock dependencies
vi.mock("../api/fetchData.js", () => ({
    fetchData: vi.fn(),
}));

vi.mock("../api/responseParser.js", () => ({
    ServerResponseParser: {
        displayErrors: vi.fn(),
    },
}));

vi.mock("./useDebounce.js", () => ({
    useDebounce: vi.fn(),
}));

const mockFetchData = vi.mocked(fetchData);
const mockDisplayErrors = vi.mocked(ServerResponseParser.displayErrors);
const mockUseDebounce = vi.mocked(useDebounce);

describe("useFetch", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        // Default useDebounce implementation that calls the callback immediately
        mockUseDebounce.mockImplementation((callback, delay) => {
            const debouncedCallback = vi.fn((value) => callback(value));
            return [debouncedCallback, vi.fn()];
        });
    });

    afterEach(() => {
        vi.clearAllMocks();
    });

    describe("basic functionality", () => {
        it("should initialize with default loading state", () => {
            const { result } = renderHook(() => useFetch({
                endpoint: "/test",
                inputs: {},
            }));

            expect(result.current.loading).to.equal(false);
            expect(result.current.data).to.be.undefined;
            expect(result.current.errors).to.be.undefined;
            expect(typeof result.current.refetch).to.equal("function");
        });

        it("should not make request when endpoint is undefined", () => {
            renderHook(() => useFetch({
                endpoint: undefined,
                inputs: {},
            }));

            expect(mockFetchData).not.to.have.been.called;
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

            await waitFor(() => {
                expect(mockFetchData).to.have.been.calledWith({
                    endpoint: "/test",
                    inputs: {},
                    method: "GET",
                    options: {},
                    omitRestBase: false,
                });
            });
        });
    });

    describe("data fetching with loading states", () => {
        it("should set loading to true during fetch and false after success", async () => {
            const mockResponse = {
                data: { id: 1, name: "test" },
                errors: undefined,
                version: "1.0.0",
                __fetchTimestamp: Date.now(),
            };

            // Create a promise that we can resolve manually to control timing
            let resolvePromise: (value: any) => void;
            const fetchPromise = new Promise((resolve) => {
                resolvePromise = resolve;
            });
            mockFetchData.mockReturnValueOnce(fetchPromise as any);

            const { result } = renderHook(() => useFetch({
                endpoint: "/test",
                inputs: {},
            }));

            // Initially should not be loading
            expect(result.current.loading).to.equal(false);

            // Trigger the fetch by calling the debounced callback
            await act(async () => {
                const [debouncedCallback] = mockUseDebounce.mock.calls[0];
                debouncedCallback(undefined);
            });

            // Should be loading now
            await waitFor(() => {
                expect(result.current.loading).to.equal(true);
            });

            // Resolve the promise
            await act(async () => {
                resolvePromise!(mockResponse);
                await fetchPromise;
            });

            // Should no longer be loading and have data
            await waitFor(() => {
                expect(result.current.loading).to.equal(false);
                expect(result.current.data).to.deep.equal(mockResponse.data);
                expect(result.current.errors).to.be.undefined;
            });
        });

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

            await waitFor(() => {
                expect(result.current.data).to.deep.equal(mockData);
                expect(result.current.errors).to.be.undefined;
                expect(result.current.loading).to.equal(false);
            });
        });
    });

    describe("error handling scenarios", () => {
        it("should handle response with errors and display them", async () => {
            const mockErrors = [
                { message: "Validation failed", trace: "001" },
                { message: "Invalid input", trace: "002" },
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

            await waitFor(() => {
                expect(result.current.errors).to.deep.equal(mockErrors);
                expect(result.current.data).to.equal(mockResponse.data);
                expect(mockDisplayErrors).to.have.been.calledWith(mockErrors);
            });
        });

        it("should handle missing endpoint gracefully", async () => {
            const { result } = renderHook(() => useFetch({
                endpoint: undefined,
                inputs: {},
            }));

            // Manually call refetch to trigger the no-endpoint case
            await act(async () => {
                const response = await result.current.refetch();
                // @ts-ignore: Testing error response structure
                expect(response.errors).to.deep.equal([{
                    trace: "0693",
                    message: "No endpoint provided to useLazyFetch",
                }]);
            });
        });
    });

    describe("dependency array changes", () => {
        it("should refetch when dependency array changes", async () => {
            const mockResponse = {
                data: { id: 1 },
                errors: undefined,
                version: "1.0.0",
                __fetchTimestamp: Date.now(),
            };

            mockFetchData.mockResolvedValue(mockResponse);

            const { rerender } = renderHook(
                ({ deps }) => useFetch({
                    endpoint: "/test",
                    inputs: {},
                }, deps),
                { initialProps: { deps: [1] } }
            );

            await waitFor(() => {
                expect(mockFetchData).to.have.been.calledOnce;
            });

            // Change dependency
            rerender({ deps: [2] });

            await waitFor(() => {
                expect(mockFetchData).to.have.been.calledTwice;
            });
        });
    });

    describe("different HTTP methods and options", () => {
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

            await waitFor(() => {
                expect(mockFetchData).to.have.been.calledWith({
                    endpoint: "/test",
                    inputs: { name: "test" },
                    method: "POST",
                    options: {},
                    omitRestBase: false,
                });
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

            await waitFor(() => {
                expect(mockFetchData).to.have.been.calledWith({
                    endpoint: "/auth/login",
                    inputs: {},
                    method: "GET",
                    options: {},
                    omitRestBase: true,
                });
            });
        });
    });

    describe("manual refetch functionality", () => {
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

            // Wait for initial fetch
            await waitFor(() => {
                expect(result.current.data).to.deep.equal(mockResponse1.data);
            });

            // Manual refetch with new input
            await act(async () => {
                await result.current.refetch({ id: 2 });
            });

            await waitFor(() => {
                expect(result.current.data).to.deep.equal(mockResponse2.data);
                expect(mockFetchData).to.have.been.calledTwice;
            });
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

            expect(typeof getData).to.equal("function");
            expect(state.loading).to.equal(false);
            expect(state.data).to.be.undefined;
            expect(state.errors).to.be.undefined;
        });

        it("should not make automatic request on mount", () => {
            renderHook(() => useLazyFetch({
                endpoint: "/test",
                inputs: {},
            }));

            expect(mockFetchData).not.to.have.been.called;
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
                expect(response.data).to.deep.equal(mockResponse.data);
            });

            expect(mockFetchData).to.have.been.calledWith({
                endpoint: "/test",
                inputs: {},
                method: "GET",
                options: {},
            });

            expect(result.current[1].data).to.deep.equal(mockResponse.data);
            expect(result.current[1].loading).to.equal(false);
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
                expect(response.errors).to.deep.equal(mockErrors);
            });

            expect(result.current[1].errors).to.deep.equal(mockErrors);
            expect(mockDisplayErrors).to.have.been.calledWith(mockErrors);
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
                expect(response.errors).to.deep.equal([{
                    trace: "0692",
                    message: "No endpoint provided to useLazyFetch",
                }]);
            });

            expect(result.current[1].errors).to.deep.equal([{
                trace: "0692",
                message: "No endpoint provided to useLazyFetch",
            }]);
            expect(consoleSpy).to.have.been.calledWith("No endpoint provided to useLazyFetch");

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

            // Mock two responses with different timestamps
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

            // First call returns newer response
            mockFetchData.mockResolvedValueOnce(newerResponse);
            
            await act(async () => {
                await getData();
            });

            expect(result.current[1].data).to.deep.equal(newerResponse.data);

            // Second call returns older response (should be ignored)
            mockFetchData.mockResolvedValueOnce(olderResponse);
            
            const consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

            await act(async () => {
                await getData();
            });

            // Data should still be the newer response
            expect(result.current[1].data).to.deep.equal(newerResponse.data);
            expect(consoleSpy).to.have.been.called;

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

            expect(mockFetchData).to.have.been.calledWith({
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
        expect(uiPathToApi("/users")).to.equal("/api/v2/users");
        expect(uiPathToApi("/u/@john")).to.equal("/api/v2/u/@john");
        expect(uiPathToApi("/search?q=test")).to.equal("/api/v2/search?q=test");
    });

    it("should convert UI path to API path with custom base", () => {
        expect(uiPathToApi("/users", "/api/v1")).to.equal("/api/v1/users");
        expect(uiPathToApi("/items", "/custom/api")).to.equal("/custom/api/items");
    });

    it("should handle empty path", () => {
        expect(uiPathToApi("")).to.equal("/api/v2");
        expect(uiPathToApi("", "/custom")).to.equal("/custom");
    });
});