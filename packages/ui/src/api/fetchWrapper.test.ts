// AI_CHECK: TEST_COVERAGE=1,TEST_QUALITY=1 | LAST: 2025-06-19
import { describe, it, expect, vi, beforeEach, afterEach, type Mock, beforeAll } from "vitest";

// Clear the global mock for fetchWrapper so we can test the actual implementation
beforeAll(() => {
    vi.unmock("./fetchWrapper.js");
    vi.unmock("../api/fetchWrapper.js");
    vi.unmock("../../api/fetchWrapper.js");
    vi.unmock("../../api/fetchWrapper");
});

import { ServerResponseParser } from "./responseParser.js";
import { PubSub } from "../utils/pubsub.js";
import { fetchData } from "./fetchData.js";
import { fetchLazyWrapper, fetchWrapper, useSubmitHelper } from "./fetchWrapper.js";
import { renderHook } from "@testing-library/react";
import i18next from "i18next";

// Mock dependencies
vi.mock("./responseParser.js", () => ({
    ServerResponseParser: {
        errorToMessage: vi.fn(),
    },
}));

vi.mock("../utils/pubsub.js", () => ({
    PubSub: {
        get: vi.fn(() => ({
            publish: vi.fn(),
        })),
    },
}));

vi.mock("./fetchData.js", () => ({
    fetchData: vi.fn(),
}));

// i18next is already mocked in setup.vitest.ts via react-i18next mock

describe("fetchWrapper module", () => {
    const mockPubSubPublish = vi.fn();
    const mockFetchData = fetchData as Mock;
    const mockI18nT = vi.mocked(i18next.t);

    beforeEach(() => {
        vi.clearAllMocks();
        (PubSub.get as Mock).mockReturnValue({
            publish: mockPubSubPublish,
        });
        (ServerResponseParser.errorToMessage as Mock).mockReturnValue("Error message");
        mockI18nT.mockImplementation((key: string) => key);
    });

    afterEach(() => {
        vi.clearAllTimers();
    });

    describe("fetchLazyWrapper", () => {
        const mockFetch = vi.fn();

        beforeEach(() => {
            mockFetch.mockClear();
            vi.useFakeTimers();
        });

        afterEach(() => {
            vi.useRealTimers();
        });

        it("should handle successful response with default success condition", async () => {
            const mockData = { id: "123", name: "test" };
            const mockResponse = { data: mockData };
            mockFetch.mockResolvedValue(mockResponse);

            const onSuccess = vi.fn();
            const onCompleted = vi.fn();

            const result = await fetchLazyWrapper({
                fetch: mockFetch,
                inputs: { test: "input" },
                onSuccess,
                onCompleted,
            });

            expect(mockFetch).toHaveBeenCalledWith({ test: "input" }, { displayError: false });
            expect(onSuccess).toHaveBeenCalledWith(mockData);
            expect(onCompleted).toHaveBeenCalledWith(mockResponse);
            expect(result).toEqual(mockResponse);
        });

        it("should handle File inputs by converting to FormData", async () => {
            const file = new File(["content"], "test.txt", { type: "text/plain" });
            const inputs = { file, name: "test" };
            const mockResponse = { data: { success: true } };
            mockFetch.mockResolvedValue(mockResponse);

            await fetchLazyWrapper({
                fetch: mockFetch,
                inputs,
            });

            // Verify that FormData was passed instead of the original object
            const callArgs = mockFetch.mock.calls[0][0];
            expect(callArgs).toBeInstanceOf(FormData);
        });

        it("should handle custom success condition", async () => {
            const mockData = { status: "pending" };
            const mockResponse = { data: mockData };
            mockFetch.mockResolvedValue(mockResponse);

            const successCondition = vi.fn().mockReturnValue(false);
            const onError = vi.fn();

            await fetchLazyWrapper({
                fetch: mockFetch,
                successCondition,
                onError,
                showDefaultErrorSnack: false,
            });

            expect(successCondition).toHaveBeenCalledWith(mockData);
            // The implementation transforms the response when success condition fails
            expect(onError).toHaveBeenCalledWith({
                errors: [{ trace: "0694", message: "Unknown error occurred" }],
            });
        });

        it("should display success message when provided", async () => {
            const mockData = { id: "123" };
            const mockResponse = { data: mockData };
            mockFetch.mockResolvedValue(mockResponse);

            const successMessage = vi.fn().mockReturnValue({
                messageKey: "Success",
                messageVariables: { id: "123" },
            });

            await fetchLazyWrapper({
                fetch: mockFetch,
                successMessage,
            });

            expect(successMessage).toHaveBeenCalledWith(mockData);
            expect(mockPubSubPublish).toHaveBeenCalledWith("snack", {
                messageKey: "Success",
                messageVariables: { id: "123" },
                severity: "Success",
            });
        });

        it("should handle error with custom error message", async () => {
            const mockError = { errors: [{ message: "Test error" }] };
            mockFetch.mockResolvedValue(mockError);

            const errorMessage = vi.fn().mockReturnValue({
                messageKey: "CustomError",
            });
            const onError = vi.fn();

            await fetchLazyWrapper({
                fetch: mockFetch,
                errorMessage,
                onError,
                showDefaultErrorSnack: false,
            });

            expect(errorMessage).toHaveBeenCalledWith(mockError);
            expect(mockPubSubPublish).toHaveBeenCalledWith("snack", {
                messageKey: "CustomError",
                severity: "Error",
                data: mockError,
            });
            expect(onError).toHaveBeenCalledWith(mockError);
        });

        it("should display default error message when no custom message provided", async () => {
            const mockError = { errors: [{ message: "Test error" }] };
            mockFetch.mockResolvedValue(mockError);

            await fetchLazyWrapper({
                fetch: mockFetch,
                showDefaultErrorSnack: true,
            });

            expect(ServerResponseParser.errorToMessage).toHaveBeenCalledWith(mockError, ["en"]);
            expect(mockPubSubPublish).toHaveBeenCalledWith("snack", {
                message: "Error message",
                severity: "Error",
                data: mockError,
            });
        });

        it("should handle null/undefined response as error", async () => {
            mockFetch.mockResolvedValue(null);
            const onError = vi.fn();

            await fetchLazyWrapper({
                fetch: mockFetch,
                onError,
                showDefaultErrorSnack: false,
            });

            expect(onError).toHaveBeenCalledWith({
                errors: [{ trace: "0694", message: "Unknown error occurred" }],
            });
        });

        it("should handle response with count = 0 as error", async () => {
            const mockResponse = { data: { count: 0 } };
            mockFetch.mockResolvedValue(mockResponse);
            const onError = vi.fn();

            await fetchLazyWrapper({
                fetch: mockFetch,
                onError,
                showDefaultErrorSnack: false,
            });

            // The implementation transforms the response when count = 0
            expect(onError).toHaveBeenCalledWith({
                errors: [{ trace: "0694", message: "Unknown error occurred" }],
            });
        });

        it("should handle response with success = false as error", async () => {
            const mockResponse = { data: { success: false } };
            mockFetch.mockResolvedValue(mockResponse);
            const onError = vi.fn();

            await fetchLazyWrapper({
                fetch: mockFetch,
                onError,
                showDefaultErrorSnack: false,
            });

            // The implementation transforms the response when success = false
            expect(onError).toHaveBeenCalledWith({
                errors: [{ trace: "0694", message: "Unknown error occurred" }],
            });
        });

        it("should handle fetch rejection", async () => {
            const mockError = { errors: [{ message: "Network error" }] };
            mockFetch.mockRejectedValue(mockError);
            const onError = vi.fn();

            await fetchLazyWrapper({
                fetch: mockFetch,
                onError,
                showDefaultErrorSnack: false,
            });

            expect(onError).toHaveBeenCalledWith(mockError);
        });

        it("should show and hide spinner when spinnerDelay is set", async () => {
            const mockResponse = { data: { success: true } };
            mockFetch.mockResolvedValue(mockResponse);

            await fetchLazyWrapper({
                fetch: mockFetch,
                spinnerDelay: 500,
            });

            // Check that spinner was shown
            expect(mockPubSubPublish).toHaveBeenCalledWith("menu", {
                id: "full-page-spinner",
                data: { show: true, delay: 500 },
            });

            // Check that spinner was hidden
            expect(mockPubSubPublish).toHaveBeenCalledWith("menu", {
                id: "full-page-spinner",
                data: { show: false },
            });
        });

        it("should not show spinner when spinnerDelay is null", async () => {
            const mockResponse = { data: { success: true } };
            mockFetch.mockResolvedValue(mockResponse);

            await fetchLazyWrapper({
                fetch: mockFetch,
                spinnerDelay: null,
            });

            // Verify spinner was not shown/hidden
            const spinnerCalls = mockPubSubPublish.mock.calls.filter(
                call => call[0] === "menu" && call[1].id === "full-page-spinner",
            );
            expect(spinnerCalls).toHaveLength(0);
        });

        it("should handle caught error with unknown error structure", async () => {
            const unknownError = "String error";
            mockFetch.mockRejectedValue(unknownError);
            const onError = vi.fn();
            const onCompleted = vi.fn();

            await fetchLazyWrapper({
                fetch: mockFetch,
                onError,
                onCompleted,
                showDefaultErrorSnack: false,
            });

            expect(onError).toHaveBeenCalledWith({
                errors: [{ trace: "0694", message: "Unknown error occurred" }],
            });
            expect(onCompleted).toHaveBeenCalledWith({
                errors: [{ trace: "0695", message: "Unknown error occurred" }],
            });
        });
    });

    describe("fetchWrapper", () => {
        beforeEach(() => {
            mockFetchData.mockResolvedValue({ data: { success: true } });
        });

        it("should call fetchData with correct parameters", async () => {
            const endpoint = "/api/test";
            const method = "POST";
            const inputs = { test: "data" };
            const onSuccess = vi.fn();

            await fetchWrapper({
                endpoint,
                method,
                inputs,
                onSuccess,
            });

            expect(mockFetchData).toHaveBeenCalledWith({
                endpoint,
                method,
                inputs,
            });
        });

        it("should pass through all fetchLazyWrapper options", async () => {
            const endpoint = "/api/test";
            const method = "GET";
            const onSuccess = vi.fn();
            const successMessage = vi.fn().mockReturnValue({ messageKey: "Success" });

            await fetchWrapper({
                endpoint,
                method,
                onSuccess,
                successMessage,
                spinnerDelay: 1500,
            });

            expect(onSuccess).toHaveBeenCalled();
            expect(successMessage).toHaveBeenCalled();
        });
    });

    describe("useSubmitHelper", () => {
        it("should return a function", () => {
            const { result } = renderHook(() =>
                useSubmitHelper({
                    fetch: vi.fn(),
                    isCreate: true,
                }),
            );

            expect(typeof result.current).toBe("function");
        });

        it("should show unauthorized message when disabled is true", () => {
            const { result } = renderHook(() =>
                useSubmitHelper({
                    fetch: vi.fn(),
                    disabled: true,
                    isCreate: true,
                }),
            );

            result.current();

            expect(mockPubSubPublish).toHaveBeenCalledWith("snack", {
                message: "Unauthorized",
                severity: "Error",
            });
        });

        it("should show object read error when not creating and existing is empty array", () => {
            const { result } = renderHook(() =>
                useSubmitHelper({
                    fetch: vi.fn(),
                    isCreate: false,
                    existing: [],
                }),
            );

            result.current();

            expect(mockPubSubPublish).toHaveBeenCalledWith("snack", {
                message: "CouldNotReadObject",
                severity: "Error",
            });
        });

        it("should show object read error when not creating and existing is null", () => {
            const { result } = renderHook(() =>
                useSubmitHelper({
                    fetch: vi.fn(),
                    isCreate: false,
                    existing: null,
                }),
            );

            result.current();

            expect(mockPubSubPublish).toHaveBeenCalledWith("snack", {
                message: "CouldNotReadObject",
                severity: "Error",
            });
        });

        it("should call fetchLazyWrapper when all conditions are met", async () => {
            const mockFetch = vi.fn().mockResolvedValue({ data: { success: true } });
            const inputs = { test: "data" };

            const { result } = renderHook(() =>
                useSubmitHelper({
                    fetch: mockFetch,
                    inputs,
                    isCreate: true,
                    disabled: false,
                }),
            );

            await result.current();

            // Should not show any error messages
            const errorCalls = mockPubSubPublish.mock.calls.filter(
                call => call[0] === "snack" && call[1].severity === "Error",
            );
            expect(errorCalls).toHaveLength(0);
        });

        it("should allow creating when existing object does not exist", async () => {
            const mockFetch = vi.fn().mockResolvedValue({ data: { success: true } });
            
            const { result } = renderHook(() =>
                useSubmitHelper({
                    fetch: mockFetch,
                    isCreate: true,
                    existing: undefined,
                }),
            );

            await result.current();

            // Should not show any error messages
            const errorCalls = mockPubSubPublish.mock.calls.filter(
                call => call[0] === "snack" && call[1].severity === "Error",
            );
            expect(errorCalls).toHaveLength(0);
        });

        it("should allow updating when existing object exists", async () => {
            const mockFetch = vi.fn().mockResolvedValue({ data: { success: true } });
            
            const { result } = renderHook(() =>
                useSubmitHelper({
                    fetch: mockFetch,
                    isCreate: false,
                    existing: { id: "123" },
                }),
            );

            await result.current();

            // Should not show any error messages
            const errorCalls = mockPubSubPublish.mock.calls.filter(
                call => call[0] === "snack" && call[1].severity === "Error",
            );
            expect(errorCalls).toHaveLength(0);
        });
    });

    describe("objectToFormData utility (internal)", () => {
        // Since objectToFormData is not exported, we test it indirectly through fetchLazyWrapper
        it("should convert Date objects to ISO strings in FormData", async () => {
            const date = new Date("2023-01-01T00:00:00.000Z");
            const file = new File(["content"], "test.txt");
            const inputs = { 
                file,
                createdAt: date,
                name: "test", 
            };
            
            const mockFetch = vi.fn().mockResolvedValue({ data: { success: true } });

            await fetchLazyWrapper({
                fetch: mockFetch,
                inputs,
            });

            const callArgs = mockFetch.mock.calls[0][0];
            expect(callArgs).toBeInstanceOf(FormData);
            
            // Check that date was converted to ISO string
            const formData = callArgs as FormData;
            expect(formData.get("createdAt")).toBe("2023-01-01T00:00:00.000Z");
            expect(formData.get("name")).toBe("test");
        });

        it("should handle nested objects in FormData conversion", async () => {
            const file = new File(["content"], "test.txt");
            const inputs = {
                file,
                user: {
                    name: "John",
                    age: 30,
                    profile: {
                        bio: "Developer",
                    },
                },
            };
            
            const mockFetch = vi.fn().mockResolvedValue({ data: { success: true } });

            await fetchLazyWrapper({
                fetch: mockFetch,
                inputs,
            });

            const callArgs = mockFetch.mock.calls[0][0];
            expect(callArgs).toBeInstanceOf(FormData);
            
            const formData = callArgs as FormData;
            expect(formData.get("user[name]")).toBe("John");
            expect(formData.get("user[age]")).toBe("30");
            expect(formData.get("user[profile][bio]")).toBe("Developer");
        });

        it("should skip null and undefined values in FormData conversion", async () => {
            const file = new File(["content"], "test.txt");
            const inputs = {
                file,
                name: "test",
                nullValue: null,
                undefinedValue: undefined,
                emptyString: "",
            };
            
            const mockFetch = vi.fn().mockResolvedValue({ data: { success: true } });

            await fetchLazyWrapper({
                fetch: mockFetch,
                inputs,
            });

            const callArgs = mockFetch.mock.calls[0][0];
            const formData = callArgs as FormData;
            
            expect(formData.get("name")).toBe("test");
            expect(formData.get("emptyString")).toBe("");
            expect(formData.has("nullValue")).toBe(false);
            expect(formData.has("undefinedValue")).toBe(false);
        });
    });
});
