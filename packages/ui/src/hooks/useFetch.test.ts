import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { http, HttpResponse } from "msw";
import { server } from "../__test/mocks/server.js";
import { apiUrlBase, restBase } from "../utils/consts.js";
import { PubSub } from "../utils/pubsub.js";
// AI_CHECK: TEST_QUALITY=3 | LAST: 2025-06-24 | Updated to use MSW instead of manual mocks
import { useLazyFetch, uiPathToApi } from "./useFetch.js";

// Mock PubSub to track error display calls
vi.mock("../utils/pubsub.js", () => ({
    PubSub: {
        get: vi.fn(() => ({
            publish: vi.fn(),
        })),
    },
}));

const mockPublish = vi.fn();

describe("useLazyFetch (core functionality)", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        // Set up PubSub mock
        vi.mocked(PubSub.get).mockReturnValue({
            publish: mockPublish,
        } as ReturnType<typeof PubSub.get>);
        mockPublish.mockClear();
    });

    afterEach(() => {
        vi.clearAllTimers();
        server.resetHandlers();
    });

    describe("lazy loading behavior", () => {
        it("waits for explicit trigger before loading data", () => {
            // GIVEN: A component that needs manual control over data fetching
            // WHEN: Component mounts with useLazyFetch
            const [getData, state] = renderHook(() => useLazyFetch({
                endpoint: "/test",
                inputs: {},
            })).result.current;

            // THEN: No data should be fetched automatically
            expect(typeof getData).toBe("function");
            expect(state.loading).toBe(false);
            expect(state.data).toBeUndefined();
            // No MSW handler is set up, so any fetch would fail if attempted
        });
    });

    describe("on-demand data fetching", () => {
        it("fetches data only when explicitly triggered by user action", async () => {
            // GIVEN: A form that needs to load data when user clicks a button
            const mockResponseData = { id: 1, name: "test" };
            
            // Set up MSW handler for the test endpoint
            server.use(
                http.get(`${apiUrlBase}${restBase}/test`, () => {
                    return HttpResponse.json({
                        data: mockResponseData,
                        errors: undefined,
                        version: "1.0.0",
                    });
                }),
            );

            const { result } = renderHook(() => useLazyFetch({
                endpoint: "/test",
                inputs: {},
            }));

            const [getData] = result.current;

            // WHEN: User triggers data load (e.g., button click)
            await act(async () => {
                const state = await getData();
                expect(state.data).toEqual(mockResponseData);
                expect(state.loading).toBe(false);
                expect(state.errors).toBeUndefined();
            });

            // THEN: Data should be fetched and state updated
            expect(result.current[1].data).toEqual(mockResponseData);
            expect(result.current[1].loading).toBe(false);
        });
    });

    describe("error handling", () => {
        it("should handle errors and display them", async () => {
            const mockErrors = [{ message: "Server error", trace: "500" }];
            
            // Set up MSW handler that returns an error response
            server.use(
                http.get(`${apiUrlBase}${restBase}/test`, () => {
                    return HttpResponse.json({
                        data: undefined,
                        errors: mockErrors,
                        version: "1.0.0",
                    });
                }),
            );

            const { result } = renderHook(() => useLazyFetch({
                endpoint: "/test",
                inputs: {},
            }));

            const [getData] = result.current;

            await act(async () => {
                const state = await getData();
                expect(state.errors).toEqual(mockErrors);
                expect(state.data).toBeUndefined();
                expect(state.loading).toBe(false);
            });

            expect(result.current[1].errors).toEqual(mockErrors);
            
            // Check that error was displayed via PubSub
            expect(mockPublish).toHaveBeenCalledWith("snack", {
                message: "Server error",
                severity: "Error",
            });
        });

        it("should handle missing endpoint", async () => {
            const { result } = renderHook(() => useLazyFetch({
                endpoint: undefined,
                inputs: {},
            }));

            const [getData] = result.current;
            const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => { /* ignore */ });

            await act(async () => {
                const state = await getData();
                expect(state.errors).toEqual([{
                    trace: "0692",
                    message: "No endpoint provided to useLazyFetch",
                }]);
                expect(state.data).toBeUndefined();
                expect(state.loading).toBe(false);
            });

            expect(result.current[1].errors).toEqual([{
                trace: "0692",
                message: "No endpoint provided to useLazyFetch",
            }]);
            expect(consoleSpy).toHaveBeenCalledWith("No endpoint provided to useLazyFetch");

            // Note: displayErrors is NOT called for missing endpoint - this is handled locally

            consoleSpy.mockRestore();
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
