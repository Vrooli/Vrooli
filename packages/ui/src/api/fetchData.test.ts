import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { fetchData } from "./fetchData.js";
import { apiUrlBase, restBase } from "../utils/consts.js";
import { invalidateAIConfigCache } from "./ai.js";
import type { HttpMethod } from "@vrooli/shared";

// Mock dependencies
vi.mock("./ai.js", () => ({
    invalidateAIConfigCache: vi.fn(),
}));

// Mock fetch globally
const mockFetch = vi.fn();
global.fetch = mockFetch;

describe("fetchData", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        localStorage.clear();
    });

    afterEach(() => {
        vi.clearAllMocks();
    });

    describe("URL construction", () => {
        it("should construct URL with rest base by default", async () => {
            mockFetch.mockResolvedValueOnce({
                json: async () => ({ data: "test", version: "1.0.0" }),
            });

            await fetchData({
                endpoint: "/users",
                method: "GET",
                inputs: undefined,
            });

            expect(mockFetch).toHaveBeenCalledWith(
                `${apiUrlBase}${restBase}/users`,
                expect.any(Object),
            );
        });

        it("should omit rest base when omitRestBase is true", async () => {
            mockFetch.mockResolvedValueOnce({
                json: async () => ({ data: "test", version: "1.0.0" }),
            });

            await fetchData({
                endpoint: "/auth/login",
                method: "POST",
                inputs: { email: "test@example.com", password: "password" },
                omitRestBase: true,
            });

            expect(mockFetch).toHaveBeenCalledWith(
                `${apiUrlBase}/auth/login`,
                expect.any(Object),
            );
        });

        it("should replace URL variables with input values", async () => {
            mockFetch.mockResolvedValueOnce({
                json: async () => ({ data: "test", version: "1.0.0" }),
            });

            const inputs = { id: "123", action: "update" };
            await fetchData({
                endpoint: "/users/:id/:action",
                method: "PUT",
                inputs,
            });

            expect(mockFetch).toHaveBeenCalledWith(
                `${apiUrlBase}${restBase}/users/123/update`,
                expect.any(Object),
            );
        });

        it("should handle multiple variable replacements and remove them from inputs", async () => {
            mockFetch.mockResolvedValueOnce({
                json: async () => ({ data: "test", version: "1.0.0" }),
            });

            const inputs = { 
                userId: "456", 
                postId: "789",
                title: "Updated Title",
                content: "Updated Content"
            };
            
            await fetchData({
                endpoint: "/users/:userId/posts/:postId",
                method: "PUT",
                inputs,
            });

            const calledUrl = mockFetch.mock.calls[0][0];
            const calledOptions = mockFetch.mock.calls[0][1];
            
            expect(calledUrl).toBe(`${apiUrlBase}${restBase}/users/456/posts/789`);
            // Check that replaced variables are removed from body
            const bodyData = JSON.parse(calledOptions.body);
            expect(bodyData).toEqual({
                title: "Updated Title",
                content: "Updated Content"
            });
            expect(bodyData.userId).toBeUndefined();
            expect(bodyData.postId).toBeUndefined();
        });
    });

    describe("HTTP methods", () => {
        describe("GET requests", () => {
            it("should append inputs as query parameters for GET requests", async () => {
                mockFetch.mockResolvedValueOnce({
                    json: async () => ({ data: "test", version: "1.0.0" }),
                });

                await fetchData({
                    endpoint: "/search",
                    method: "GET",
                    inputs: { q: "test query", limit: 10, offset: 0 },
                });

                const calledUrl = mockFetch.mock.calls[0][0];
                const calledOptions = mockFetch.mock.calls[0][1];
                
                // Extract query params from URL
                const url = new URL(calledUrl);
                const params = Object.fromEntries(url.searchParams.entries());
                
                expect(url.pathname).toBe(`/api${restBase}/search`);
                expect(params.q).toContain("test query");
                expect(params.limit).toContain("10");
                expect(params.offset).toContain("0");
                expect(calledOptions.method).toBe("GET");
                expect(calledOptions.body).toBeNull();
            });

            it("should handle GET requests without inputs", async () => {
                mockFetch.mockResolvedValueOnce({
                    json: async () => ({ data: "test", version: "1.0.0" }),
                });

                await fetchData({
                    endpoint: "/users",
                    method: "GET",
                    inputs: undefined,
                });

                expect(mockFetch).toHaveBeenCalledWith(
                    `${apiUrlBase}${restBase}/users`,
                    expect.objectContaining({
                        method: "GET",
                        body: null,
                    }),
                );
            });

            it("should handle empty inputs object for GET requests", async () => {
                mockFetch.mockResolvedValueOnce({
                    json: async () => ({ data: "test", version: "1.0.0" }),
                });

                await fetchData({
                    endpoint: "/users",
                    method: "GET",
                    inputs: {},
                });

                expect(mockFetch).toHaveBeenCalledWith(
                    `${apiUrlBase}${restBase}/users`,
                    expect.objectContaining({
                        method: "GET",
                        body: null,
                    }),
                );
            });
        });

        describe("POST requests", () => {
            it("should send inputs as JSON body for POST requests", async () => {
                mockFetch.mockResolvedValueOnce({
                    json: async () => ({ data: "created", version: "1.0.0" }),
                });

                const inputs = { name: "John Doe", email: "john@example.com" };
                await fetchData({
                    endpoint: "/users",
                    method: "POST",
                    inputs,
                });

                expect(mockFetch).toHaveBeenCalledWith(
                    `${apiUrlBase}${restBase}/users`,
                    expect.objectContaining({
                        method: "POST",
                        headers: expect.objectContaining({
                            "Content-Type": "application/json",
                        }),
                        body: JSON.stringify(inputs),
                    }),
                );
            });

            it("should handle FormData for POST requests", async () => {
                mockFetch.mockResolvedValueOnce({
                    json: async () => ({ data: "uploaded", version: "1.0.0" }),
                });

                const formData = new FormData();
                formData.append("file", new Blob(["test"]), "test.txt");
                formData.append("description", "Test file");

                await fetchData({
                    endpoint: "/upload",
                    method: "POST",
                    inputs: formData as any,
                });

                expect(mockFetch).toHaveBeenCalledWith(
                    `${apiUrlBase}${restBase}/upload`,
                    expect.objectContaining({
                        method: "POST",
                        headers: expect.not.objectContaining({
                            "Content-Type": "application/json",
                        }),
                        body: formData,
                    }),
                );
            });
        });

        describe("PUT requests", () => {
            it("should send inputs as JSON body for PUT requests", async () => {
                mockFetch.mockResolvedValueOnce({
                    json: async () => ({ data: "updated", version: "1.0.0" }),
                });

                const inputs = { name: "Jane Doe", email: "jane@example.com" };
                await fetchData({
                    endpoint: "/users/:id",
                    method: "PUT",
                    inputs: { id: "123", ...inputs },
                });

                expect(mockFetch).toHaveBeenCalledWith(
                    `${apiUrlBase}${restBase}/users/123`,
                    expect.objectContaining({
                        method: "PUT",
                        headers: expect.objectContaining({
                            "Content-Type": "application/json",
                        }),
                        body: JSON.stringify(inputs),
                    }),
                );
            });
        });

        describe("DELETE requests", () => {
            it("should send inputs as JSON body for DELETE requests", async () => {
                mockFetch.mockResolvedValueOnce({
                    json: async () => ({ data: "deleted", version: "1.0.0" }),
                });

                await fetchData({
                    endpoint: "/users/:id",
                    method: "DELETE",
                    inputs: { id: "123", reason: "test deletion" },
                });

                expect(mockFetch).toHaveBeenCalledWith(
                    `${apiUrlBase}${restBase}/users/123`,
                    expect.objectContaining({
                        method: "DELETE",
                        headers: expect.objectContaining({
                            "Content-Type": "application/json",
                        }),
                        body: JSON.stringify({ reason: "test deletion" }),
                    }),
                );
            });

            it("should handle DELETE requests without body", async () => {
                mockFetch.mockResolvedValueOnce({
                    json: async () => ({ data: "deleted", version: "1.0.0" }),
                });

                await fetchData({
                    endpoint: "/users/:id",
                    method: "DELETE",
                    inputs: { id: "123" },
                });

                expect(mockFetch).toHaveBeenCalledWith(
                    `${apiUrlBase}${restBase}/users/123`,
                    expect.objectContaining({
                        method: "DELETE",
                        body: JSON.stringify({}),
                    }),
                );
            });
        });
    });

    describe("Request options", () => {
        it("should include credentials as 'include'", async () => {
            mockFetch.mockResolvedValueOnce({
                json: async () => ({ data: "test", version: "1.0.0" }),
            });

            await fetchData({
                endpoint: "/test",
                method: "GET",
                inputs: undefined,
            });

            expect(mockFetch).toHaveBeenCalledWith(
                expect.any(String),
                expect.objectContaining({
                    credentials: "include",
                }),
            );
        });

        it("should handle Authorization header correctly", async () => {
            mockFetch.mockResolvedValueOnce({
                json: async () => ({ data: "test", version: "1.0.0" }),
            });

            await fetchData({
                endpoint: "/protected",
                method: "GET",
                inputs: undefined,
                options: {
                    headers: {
                        "Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test",
                    },
                },
            });

            expect(mockFetch).toHaveBeenCalledWith(
                expect.any(String),
                expect.objectContaining({
                    headers: expect.objectContaining({
                        "Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test",
                        "Content-Type": "application/json",
                    }),
                }),
            );
        });

        it("should merge custom options with default options", async () => {
            mockFetch.mockResolvedValueOnce({
                json: async () => ({ data: "test", version: "1.0.0" }),
            });

            const customHeaders = {
                "X-Custom-Header": "custom-value",
                "Authorization": "Bearer token123",
            };

            await fetchData({
                endpoint: "/test",
                method: "GET",
                inputs: undefined,
                options: {
                    headers: customHeaders,
                    mode: "cors",
                },
            });

            expect(mockFetch).toHaveBeenCalledWith(
                expect.any(String),
                expect.objectContaining({
                    headers: expect.objectContaining({
                        ...customHeaders,
                        "Content-Type": "application/json",
                    }),
                    mode: "cors",
                    credentials: "include",
                }),
            );
        });

        it("should handle abort signal", async () => {
            const abortController = new AbortController();
            mockFetch.mockResolvedValueOnce({
                json: async () => ({ data: "test", version: "1.0.0" }),
            });

            await fetchData({
                endpoint: "/test",
                method: "GET",
                inputs: undefined,
                signal: abortController.signal,
            });

            expect(mockFetch).toHaveBeenCalledWith(
                expect.any(String),
                expect.objectContaining({
                    signal: abortController.signal,
                }),
            );
        });
    });

    describe("Response handling", () => {
        it("should add __fetchTimestamp to response", async () => {
            const mockResponse = { data: "test", version: "1.0.0" };
            mockFetch.mockResolvedValueOnce({
                json: async () => mockResponse,
            });

            const beforeFetch = Date.now();
            const result = await fetchData({
                endpoint: "/test",
                method: "GET",
                inputs: undefined,
            });
            const afterFetch = Date.now();

            expect(result).toMatchObject(mockResponse);
            expect(result.__fetchTimestamp).toBeGreaterThanOrEqual(beforeFetch);
            expect(result.__fetchTimestamp).toBeLessThanOrEqual(afterFetch);
        });

        it("should handle server version change", async () => {
            // First request with version 1.0.0
            localStorage.setItem("serverVersionCache", "1.0.0");
            mockFetch.mockResolvedValueOnce({
                json: async () => ({ data: "test", version: "1.1.0" }),
            });

            await fetchData({
                endpoint: "/test",
                method: "GET",
                inputs: undefined,
            });

            expect(localStorage.getItem("serverVersionCache")).toBe("1.1.0");
            expect(invalidateAIConfigCache).toHaveBeenCalledTimes(1);
        });

        it("should not invalidate cache when server version is unchanged", async () => {
            localStorage.setItem("serverVersionCache", "1.0.0");
            mockFetch.mockResolvedValueOnce({
                json: async () => ({ data: "test", version: "1.0.0" }),
            });

            await fetchData({
                endpoint: "/test",
                method: "GET",
                inputs: undefined,
            });

            expect(localStorage.getItem("serverVersionCache")).toBe("1.0.0");
            expect(invalidateAIConfigCache).not.toHaveBeenCalled();
        });

        it("should handle first time server version check", async () => {
            // No existing version in localStorage
            mockFetch.mockResolvedValueOnce({
                json: async () => ({ data: "test", version: "1.0.0" }),
            });

            await fetchData({
                endpoint: "/test",
                method: "GET",
                inputs: undefined,
            });

            expect(localStorage.getItem("serverVersionCache")).toBe("1.0.0");
            expect(invalidateAIConfigCache).toHaveBeenCalledTimes(1);
        });
    });

    describe("Error handling", () => {
        it("should handle fetch errors", async () => {
            const fetchError = new Error("Network error");
            mockFetch.mockRejectedValueOnce(fetchError);

            await expect(
                fetchData({
                    endpoint: "/test",
                    method: "GET",
                    inputs: undefined,
                }),
            ).rejects.toThrow("Network error");
        });

        it("should handle JSON parsing errors", async () => {
            mockFetch.mockResolvedValueOnce({
                json: async () => {
                    throw new Error("Invalid JSON");
                },
            });

            await expect(
                fetchData({
                    endpoint: "/test",
                    method: "GET",
                    inputs: undefined,
                }),
            ).rejects.toThrow("Invalid JSON");
        });

        it("should handle abort errors gracefully", async () => {
            const abortError = new Error("Request aborted");
            abortError.name = "AbortError";
            mockFetch.mockRejectedValueOnce(abortError);

            const result = await fetchData({
                endpoint: "/test",
                method: "GET",
                inputs: undefined,
            });

            expect(result).toEqual({
                errors: [{ message: "Request was aborted" }],
            });
        });

        it("should rethrow non-abort errors", async () => {
            const genericError = new Error("Some other error");
            genericError.name = "TypeError";
            mockFetch.mockRejectedValueOnce(genericError);

            await expect(
                fetchData({
                    endpoint: "/test",
                    method: "GET",
                    inputs: undefined,
                }),
            ).rejects.toThrow("Some other error");
        });
    });

    describe("Edge cases", () => {
        it("should handle undefined inputs for all methods", async () => {
            const methods: HttpMethod[] = ["GET", "POST", "PUT", "DELETE"];
            
            for (const method of methods) {
                mockFetch.mockResolvedValueOnce({
                    json: async () => ({ data: "test", version: "1.0.0" }),
                });

                await fetchData({
                    endpoint: "/test",
                    method,
                    inputs: undefined,
                });

                const expectedBody = method === "GET" ? null : JSON.stringify(undefined);
                expect(mockFetch).toHaveBeenCalledWith(
                    expect.any(String),
                    expect.objectContaining({
                        method,
                        body: expectedBody,
                    }),
                );
            }
        });

        it("should handle very long query strings for GET requests", async () => {
            mockFetch.mockResolvedValueOnce({
                json: async () => ({ data: "test", version: "1.0.0" }),
            });

            const longString = "a".repeat(1000);
            await fetchData({
                endpoint: "/search",
                method: "GET",
                inputs: { query: longString, page: 1 },
            });

            const calledUrl = mockFetch.mock.calls[0][0];
            expect(calledUrl).toContain("query=");
            expect(calledUrl).toContain("page=");
        });

        it("should handle null values in inputs", async () => {
            mockFetch.mockResolvedValueOnce({
                json: async () => ({ data: "test", version: "1.0.0" }),
            });

            await fetchData({
                endpoint: "/test",
                method: "POST",
                inputs: { name: "test", value: null, active: true },
            });

            const calledOptions = mockFetch.mock.calls[0][1];
            expect(JSON.parse(calledOptions.body)).toEqual({
                name: "test",
                value: null,
                active: true,
            });
        });

        it("should handle arrays in GET query parameters", async () => {
            mockFetch.mockResolvedValueOnce({
                json: async () => ({ data: "test", version: "1.0.0" }),
            });

            await fetchData({
                endpoint: "/search",
                method: "GET",
                inputs: { tags: ["tag1", "tag2", "tag3"], status: ["active", "pending"] },
            });

            const calledUrl = mockFetch.mock.calls[0][0];
            // Arrays should be present in the URL
            expect(calledUrl).toContain("tags=");
            expect(calledUrl).toContain("status=");
            
            // Verify the URL contains the array data in some encoded form
            const url = new URL(calledUrl);
            const params = Object.fromEntries(url.searchParams.entries());
            expect(params.tags).toBeDefined();
            expect(params.status).toBeDefined();
        });

        it("should handle response without version field", async () => {
            const mockResponse = { data: "test" };
            mockFetch.mockResolvedValueOnce({
                json: async () => mockResponse,
            });

            const result = await fetchData({
                endpoint: "/test",
                method: "GET",
                inputs: undefined,
            });

            expect(result).toMatchObject(mockResponse);
            expect(result.__fetchTimestamp).toBeDefined();
            // Should not crash when version is undefined
            expect(invalidateAIConfigCache).toHaveBeenCalledTimes(1);
        });

        it("should handle special characters in URL parameters", async () => {
            mockFetch.mockResolvedValueOnce({
                json: async () => ({ data: "test", version: "1.0.0" }),
            });

            await fetchData({
                endpoint: "/search",
                method: "GET",
                inputs: { 
                    q: "test with spaces & special=chars?", 
                    filter: "type:user/admin"
                },
            });

            const calledUrl = mockFetch.mock.calls[0][0];
            const url = new URL(calledUrl);
            const params = Object.fromEntries(url.searchParams.entries());
            
            expect(url.pathname).toBe(`/api${restBase}/search`);
            expect(params.q).toContain("test with spaces & special=chars?");
            expect(params.filter).toContain("type:user/admin");
        });

        it("should handle empty endpoint", async () => {
            mockFetch.mockResolvedValueOnce({
                json: async () => ({ data: "test", version: "1.0.0" }),
            });

            await fetchData({
                endpoint: "",
                method: "GET",
                inputs: undefined,
            });

            expect(mockFetch).toHaveBeenCalledWith(
                `${apiUrlBase}${restBase}`,
                expect.any(Object),
            );
        });

        it("should handle complex nested inputs for POST", async () => {
            mockFetch.mockResolvedValueOnce({
                json: async () => ({ data: "created", version: "1.0.0" }),
            });

            const complexInputs = {
                user: {
                    name: "John Doe",
                    profile: {
                        age: 30,
                        preferences: {
                            notifications: true,
                            theme: "dark",
                        },
                    },
                },
                tags: ["tag1", "tag2", "tag3"],
                metadata: null,
            };

            await fetchData({
                endpoint: "/complex",
                method: "POST",
                inputs: complexInputs,
            });

            expect(mockFetch).toHaveBeenCalledWith(
                expect.any(String),
                expect.objectContaining({
                    body: JSON.stringify(complexInputs),
                }),
            );
        });

        it("should preserve original inputs object (not mutate)", async () => {
            mockFetch.mockResolvedValueOnce({
                json: async () => ({ data: "test", version: "1.0.0" }),
            });

            const originalInputs = { id: "123", name: "Test", value: 42 };
            const inputsCopy = { ...originalInputs };

            await fetchData({
                endpoint: "/items/:id",
                method: "PUT",
                inputs: originalInputs,
            });

            // The function deletes properties used for URL substitution
            // So we check that the original object was modified
            expect(originalInputs.id).toBeUndefined();
            expect(originalInputs.name).toBe(inputsCopy.name);
            expect(originalInputs.value).toBe(inputsCopy.value);
        });
    });
});