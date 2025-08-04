import axios, { type AxiosError, type AxiosRequestConfig } from "axios";
import FormData from "form-data";
import { io, type Socket } from "socket.io-client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "./client.js";
import { type ConfigManager } from "./config.js";
import { HTTP_STATUS } from "./constants.js";
import { logger } from "./logger.js";

// Mock dependencies
vi.mock("axios");
vi.mock("socket.io-client");
vi.mock("./logger.js", () => ({
    logger: {
        debug: vi.fn(),
        error: vi.fn(),
        warn: vi.fn(),
        info: vi.fn(),
    },
}));

describe("ApiClient", () => {
    let config: ConfigManager;
    let client: ApiClient;
    let mockAxiosInstance: any;
    let mockSocket: Socket;

    beforeEach(() => {
        vi.clearAllMocks();

        // Mock ConfigManager
        config = {
            getServerUrl: vi.fn().mockReturnValue("http://localhost:5329"),
            getAuthToken: vi.fn().mockReturnValue(null),
            setAuthToken: vi.fn(),
            setRefreshToken: vi.fn(),
            clearAuth: vi.fn(),
            getRefreshToken: vi.fn().mockReturnValue(null),
            isDebug: vi.fn().mockReturnValue(false),
        } as any;

        // Mock axios instance
        mockAxiosInstance = {
            get: vi.fn(),
            post: vi.fn(),
            put: vi.fn(),
            patch: vi.fn(),
            delete: vi.fn(),
            request: vi.fn(),
            interceptors: {
                request: {
                    use: vi.fn(),
                },
                response: {
                    use: vi.fn(),
                },
            },
        };

        vi.mocked(axios.create).mockReturnValue(mockAxiosInstance);

        // Mock socket
        mockSocket = {
            connect: vi.fn(),
            disconnect: vi.fn(),
            on: vi.fn(),
            off: vi.fn(),
            emit: vi.fn(),
            removeAllListeners: vi.fn(),
            connected: false,
        } as any;

        vi.mocked(io).mockReturnValue(mockSocket);

        // Create client instance
        client = new ApiClient(config);
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("constructor", () => {
        it("should create axios instance with correct config", () => {
            expect(axios.create).toHaveBeenCalledWith({
                baseURL: "http://localhost:5329/api/v2",
                timeout: 30000,
                headers: {
                    "Content-Type": "application/json",
                },
            });
        });

        it("should setup request and response interceptors", () => {
            expect(mockAxiosInstance.interceptors.request.use).toHaveBeenCalled();
            expect(mockAxiosInstance.interceptors.response.use).toHaveBeenCalled();
        });
    });

    describe("request interceptor", () => {
        let requestInterceptor: any;

        beforeEach(() => {
            requestInterceptor = vi.mocked(mockAxiosInstance.interceptors.request.use).mock.calls[0][0];
        });

        it("should add auth token when available", async () => {
            vi.mocked(config.getAuthToken).mockReturnValue("test-token");
            const requestConfig = { headers: {} } as AxiosRequestConfig;

            const result = await requestInterceptor(requestConfig);

            expect(result.headers.Authorization).toBe("Bearer test-token");
        });

        it("should not add auth header when no token", async () => {
            vi.mocked(config.getAuthToken).mockReturnValue(null);
            const requestConfig = { headers: {} } as AxiosRequestConfig;

            const result = await requestInterceptor(requestConfig);

            expect(result.headers.Authorization).toBeUndefined();
        });

        it("should log request details in debug mode", async () => {
            vi.mocked(config.isDebug).mockReturnValue(true);
            const requestConfig = {
                method: "get",
                url: "/test",
                headers: {},
                data: { test: "data" },
            } as AxiosRequestConfig;

            await requestInterceptor(requestConfig);

            expect(logger.debug).toHaveBeenCalledWith(
                "API Request: GET /test",
                expect.objectContaining({
                    headers: requestConfig.headers,
                    data: requestConfig.data,
                }),
            );
        });
    });

    describe("response interceptor", () => {
        let responseSuccessHandler: any;
        let responseErrorHandler: any;

        beforeEach(() => {
            const responseCalls = vi.mocked(mockAxiosInstance.interceptors.response.use).mock.calls[0];
            responseSuccessHandler = responseCalls[0];
            responseErrorHandler = responseCalls[1];
        });

        it("should handle successful responses", async () => {
            const response = { status: 200, data: { success: true }, config: { url: "/test" } };

            const result = await responseSuccessHandler(response);

            expect(result).toBe(response);
        });

        it("should log response in debug mode", async () => {
            vi.mocked(config.isDebug).mockReturnValue(true);
            const response = { status: 200, data: { success: true }, config: { url: "/test" } };

            await responseSuccessHandler(response);

            expect(logger.debug).toHaveBeenCalledWith(
                "API Response: 200 /test",
                { data: response.data },
            );
        });

        it("should handle 401 error and attempt token refresh", async () => {
            vi.mocked(config).setAuth = vi.fn();
            vi.mocked(config.getRefreshToken).mockReturnValue("refresh-token");
            mockAxiosInstance.post.mockResolvedValue({
                data: { accessToken: "new-token", refreshToken: "new-refresh-token", expiresIn: 3600 },
            });
            mockAxiosInstance.request.mockResolvedValue({ data: "retry-success" });

            const error: AxiosError = {
                response: { status: HTTP_STATUS.UNAUTHORIZED } as any,
                config: { url: "/protected" } as AxiosRequestConfig,
                isAxiosError: true,
                toJSON: () => ({}),
                name: "AxiosError",
                message: "Unauthorized",
            };

            const result = await responseErrorHandler(error);

            expect(mockAxiosInstance.post).toHaveBeenCalledWith("/auth/refresh", {
                refreshToken: "refresh-token",
            });
            expect(vi.mocked(config).setAuth).toHaveBeenCalledWith("new-token", "new-refresh-token", 3600);
            expect(mockAxiosInstance.request).toHaveBeenCalledWith(error.config);
            expect(result).toEqual({ data: "retry-success" });
        });

        it("should clear auth on refresh failure", async () => {
            vi.mocked(config.getRefreshToken).mockReturnValue("refresh-token");
            mockAxiosInstance.post.mockRejectedValue(new Error("Refresh failed"));

            // Create multiple errors to trigger the max refresh attempts
            const error1: AxiosError = {
                response: { status: HTTP_STATUS.UNAUTHORIZED } as any,
                config: { url: "/protected" } as AxiosRequestConfig,
                isAxiosError: true,
                toJSON: () => ({}),
                name: "AxiosError",
                message: "Unauthorized",
            };
            const error2: AxiosError = {
                response: { status: HTTP_STATUS.UNAUTHORIZED } as any,
                config: { url: "/protected2" } as AxiosRequestConfig,
                isAxiosError: true,
                toJSON: () => ({}),
                name: "AxiosError",
                message: "Unauthorized",
            };
            const error3: AxiosError = {
                response: { status: HTTP_STATUS.UNAUTHORIZED } as any,
                config: { url: "/protected3" } as AxiosRequestConfig,
                isAxiosError: true,
                toJSON: () => ({}),
                name: "AxiosError",
                message: "Unauthorized",
            };

            // Make 3 failed refresh attempts to trigger clearAuth
            await expect(responseErrorHandler(error1)).rejects.toBeDefined();
            await expect(responseErrorHandler(error2)).rejects.toBeDefined();
            await expect(responseErrorHandler(error3)).rejects.toBeDefined();
            
            expect(vi.mocked(config.clearAuth)).toHaveBeenCalled();
        });

        it("should not retry if already retried", async () => {
            const error: AxiosError = {
                response: { status: HTTP_STATUS.UNAUTHORIZED } as any,
                config: { url: "/protected", _retry: true } as AxiosRequestConfig & { _retry: boolean },
                isAxiosError: true,
                toJSON: () => ({}),
                name: "AxiosError",
                message: "Unauthorized",
            };

            await expect(responseErrorHandler(error)).rejects.toBeDefined();
            expect(mockAxiosInstance.post).not.toHaveBeenCalled();
        });
    });

    describe("HTTP methods", () => {
        beforeEach(() => {
            mockAxiosInstance.get.mockResolvedValue({ data: { result: "get" } });
            mockAxiosInstance.post.mockResolvedValue({ data: { result: "post" } });
            mockAxiosInstance.put.mockResolvedValue({ data: { result: "put" } });
            mockAxiosInstance.patch.mockResolvedValue({ data: { result: "patch" } });
            mockAxiosInstance.delete.mockResolvedValue({ data: { result: "delete" } });
        });

        it("should make GET request", async () => {
            const result = await client.get("/test", { params: { param: "value" } });

            expect(mockAxiosInstance.get).toHaveBeenCalledWith("/test", { params: { param: "value" } });
            expect(result).toEqual({ result: "get" });
        });

        it("should make POST request", async () => {
            const data = { test: "data" };
            const result = await client.post("/test", data);

            expect(mockAxiosInstance.post).toHaveBeenCalledWith("/test", data, undefined);
            expect(result).toEqual({ result: "post" });
        });

        it("should make PUT request", async () => {
            const data = { test: "data" };
            const result = await client.put("/test", data);

            expect(mockAxiosInstance.put).toHaveBeenCalledWith("/test", data, undefined);
            expect(result).toEqual({ result: "put" });
        });

        it("should make PATCH request", async () => {
            const data = { test: "data" };
            const result = await client.patch("/test", data);

            expect(mockAxiosInstance.patch).toHaveBeenCalledWith("/test", data, undefined);
            expect(result).toEqual({ result: "patch" });
        });

        it("should make DELETE request", async () => {
            const result = await client.delete("/test");

            expect(mockAxiosInstance.delete).toHaveBeenCalledWith("/test", undefined);
            expect(result).toEqual({ result: "delete" });
        });

        it("should handle request errors", async () => {
            const error = new Error("Network error");
            mockAxiosInstance.get.mockRejectedValue(error);

            await expect(client.get("/test")).rejects.toThrow("Network error");
        });
    });

    describe("uploadFile", () => {
        it("should upload file with FormData", async () => {
            const mockFile = Buffer.from("test file content");
            const mockResponse = { data: { fileId: "123" } };
            mockAxiosInstance.post.mockResolvedValue(mockResponse);

            const result = await client.uploadFile("/upload", mockFile, "test.txt");

            expect(mockAxiosInstance.post).toHaveBeenCalledWith(
                "/upload",
                expect.any(FormData),
                expect.objectContaining({
                    headers: expect.objectContaining({
                        "content-type": expect.stringMatching(/^multipart\/form-data/),
                    }),
                }),
            );
            expect(result).toEqual({ fileId: "123" });
        });

        it("should handle upload errors", async () => {
            const mockFile = Buffer.from("test file content");
            const error = new Error("Upload failed");
            mockAxiosInstance.post.mockRejectedValue(error);

            await expect(client.uploadFile("/upload", mockFile, "test.txt")).rejects.toThrow("Upload failed");
        });

        it("should call progress callback", async () => {
            const mockFile = Buffer.from("test file content");
            const mockResponse = { data: { fileId: "123" } };
            const onProgress = vi.fn();

            let progressHandler: any;
            mockAxiosInstance.post.mockImplementation((path, data, config) => {
                progressHandler = config?.onUploadProgress;
                return Promise.resolve(mockResponse);
            });

            const resultPromise = client.uploadFile("/upload", mockFile, "test.txt", onProgress);

            // Simulate progress events
            progressHandler({ loaded: 50, total: 100 });
            progressHandler({ loaded: 100, total: 100 });

            await resultPromise;

            expect(onProgress).toHaveBeenCalledWith(50);
            expect(onProgress).toHaveBeenCalledWith(100);
        });
    });

    describe("WebSocket operations", () => {
        it("should create socket connection when not connected", () => {
            const socket = client.connectWebSocket();

            expect(io).toHaveBeenCalledWith("http://localhost:5329", {
                auth: {
                    token: null,
                },
                transports: ["websocket"],
                autoConnect: true,
                reconnection: false,
                timeout: 10000,
            });
            expect(socket).toBe(mockSocket);
        });

        it("should include auth token in socket connection", () => {
            vi.mocked(config.getAuthToken).mockReturnValue("socket-token");

            client.connectWebSocket();

            expect(io).toHaveBeenCalledWith(
                "http://localhost:5329",
                expect.objectContaining({
                    auth: { token: "socket-token" },
                }),
            );
        });

        it("should reuse existing socket connection", () => {
            mockSocket.connected = true;
            const socket1 = client.connectWebSocket();
            const socket2 = client.connectWebSocket();

            expect(socket1).toBe(socket2);
            expect(io).toHaveBeenCalledTimes(1);
        });

        it("should disconnect socket", () => {
            client.connectWebSocket();
            client.disconnectWebSocket();

            expect(mockSocket.removeAllListeners).toHaveBeenCalled();
            expect(mockSocket.disconnect).toHaveBeenCalled();
        });

        it("should handle disconnect when no socket exists", () => {
            expect(() => client.disconnectWebSocket()).not.toThrow();
        });

        it("should setup socket event handlers", () => {
            client.connectWebSocket();

            expect(mockSocket.on).toHaveBeenCalledWith("connect", expect.any(Function));
            expect(mockSocket.on).toHaveBeenCalledWith("disconnect", expect.any(Function));
            expect(mockSocket.on).toHaveBeenCalledWith("error", expect.any(Function));
        });
    });

    describe("refreshAuth", () => {
        it("should refresh auth tokens successfully", async () => {
            // Add setAuth method to config mock
            vi.mocked(config).setAuth = vi.fn();
            vi.mocked(config.getRefreshToken).mockReturnValue("old-refresh-token");
            mockAxiosInstance.post.mockResolvedValue({
                data: { accessToken: "new-auth-token", refreshToken: "new-refresh-token", expiresIn: 3600 },
            });

            await (client as any).refreshAuth();

            expect(mockAxiosInstance.post).toHaveBeenCalledWith("/auth/refresh", {
                refreshToken: "old-refresh-token",
            });
            expect(vi.mocked(config).setAuth).toHaveBeenCalledWith("new-auth-token", "new-refresh-token", 3600);
        });

        it("should throw error when no refresh token available", async () => {
            vi.mocked(config.getRefreshToken).mockReturnValue(null);

            await expect((client as any).refreshAuth()).rejects.toThrow("No refresh token available");
        });

        it("should throw error on refresh failure", async () => {
            vi.mocked(config.getRefreshToken).mockReturnValue("old-refresh-token");
            mockAxiosInstance.post.mockRejectedValue(new Error("Refresh failed"));

            await expect((client as any).refreshAuth()).rejects.toThrow("Refresh failed");
        });
    });

    describe("paginate", () => {
        it("should paginate through results", async () => {
            const page1 = { items: [1, 2, 3], total: 7 };
            const page2 = { items: [4, 5, 6], total: 7 };
            const page3 = { items: [7], total: 7 };

            mockAxiosInstance.get
                .mockResolvedValueOnce({ data: page1 })
                .mockResolvedValueOnce({ data: page2 })
                .mockResolvedValueOnce({ data: page3 });

            const results: number[][] = [];
            for await (const items of client.paginate<number>("/items", {}, 3)) {
                results.push(items);
            }

            expect(results).toEqual([[1, 2, 3], [4, 5, 6], [7]]);
            expect(mockAxiosInstance.get).toHaveBeenCalledTimes(3);
            expect(mockAxiosInstance.get).toHaveBeenNthCalledWith(1, "/items", {
                params: { limit: 3, offset: 0 },
            });
            expect(mockAxiosInstance.get).toHaveBeenNthCalledWith(2, "/items", {
                params: { limit: 3, offset: 3 },
            });
            expect(mockAxiosInstance.get).toHaveBeenNthCalledWith(3, "/items", {
                params: { limit: 3, offset: 6 },
            });
        });

        it("should handle empty results", async () => {
            mockAxiosInstance.get.mockResolvedValueOnce({ data: { items: [], total: 0 } });

            const results: number[][] = [];
            for await (const items of client.paginate<number>("/items")) {
                results.push(items);
            }

            expect(results).toEqual([[]]);
            expect(mockAxiosInstance.get).toHaveBeenCalledTimes(1);
        });
    });

    describe("request", () => {
        it("should convert operation name to REST endpoint", async () => {
            mockAxiosInstance.post.mockResolvedValue({ data: { data: { id: "123", name: "Test" } } });

            const result = await client.request("routine_findMany", { where: { isPublic: true } });

            expect(mockAxiosInstance.post).toHaveBeenCalledWith(
                "/api/routine/findMany",
                { where: { isPublic: true } },
                undefined,
            );
            expect(result).toEqual({ id: "123", name: "Test" });
        });

        it("should handle response without data wrapper", async () => {
            mockAxiosInstance.post.mockResolvedValue({ data: { id: "123", name: "Test" } });

            const result = await client.request("routine_findOne", { id: "123" });

            expect(result).toEqual({ id: "123", name: "Test" });
        });

        it("should pass config options", async () => {
            mockAxiosInstance.post.mockResolvedValue({ data: { data: { success: true } } });

            await client.request("routine_update", { id: "123" }, { timeout: 5000 });

            expect(mockAxiosInstance.post).toHaveBeenCalledWith(
                "/api/routine/update",
                { id: "123" },
                { timeout: 5000 },
            );
        });
    });

    describe("formatError", () => {
        it("should format axios error with response data", () => {
            const error: AxiosError = {
                response: {
                    status: 400,
                    data: { message: "Bad request", code: "BAD_REQUEST", details: "Invalid input" },
                } as any,
                message: "Request failed",
                isAxiosError: true,
                toJSON: () => ({}),
                name: "AxiosError",
                config: {} as AxiosRequestConfig,
            };

            const formatted = (client as any).formatError(error);

            expect(formatted).toBeInstanceOf(Error);
            expect(formatted.message).toBe("Bad request\n  Please check the server logs for more details about this error.");
            expect((formatted as ApiError).code).toBe("BAD_REQUEST");
            expect((formatted as ApiError).details).toEqual("Invalid input");
        });

        it("should handle error field in response data", () => {
            const error: AxiosError = {
                response: {
                    status: 400,
                    data: { error: "Bad request from error field" },
                } as any,
                message: "Request failed",
                isAxiosError: true,
                toJSON: () => ({}),
                name: "AxiosError",
                config: {} as AxiosRequestConfig,
            };

            const formatted = (client as any).formatError(error);

            expect(formatted).toBeInstanceOf(Error);
            expect(formatted.message).toBe("Bad request from error field\n  Please try again. If the problem persists, check the server logs or contact support.");
        });

        it("should handle no response but request exists", () => {
            const error: AxiosError = {
                request: {},
                message: "Network error",
                isAxiosError: true,
                toJSON: () => ({}),
                name: "AxiosError",
                config: {} as AxiosRequestConfig,
            };

            const formatted = (client as any).formatError(error);

            expect(formatted).toBeInstanceOf(Error);
            expect(formatted.message).toBe("Cannot connect to server\n  Please check your internet connection and ensure the server is running. You can start the server with './scripts/main/develop.sh'.");
        });

        it("should return original error when no response or request", () => {
            const error: AxiosError = {
                message: "Network error",
                isAxiosError: true,
                toJSON: () => ({}),
                name: "AxiosError",
                config: {} as AxiosRequestConfig,
            };

            const formatted = (client as any).formatError(error);

            expect(formatted).toBe(error);
        });
    });
});
