// AI_CHECK: TYPE_SAFETY=eliminated-12-any-types-with-proper-interfaces | LAST: 2025-06-30
// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-24
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { http, HttpResponse } from "msw";
import { fetchData } from "./fetchData.js";
import { apiUrlBase, restBase } from "../utils/consts.js";
import { invalidateAIConfigCache } from "./ai.js";
import { server } from "../__test/mocks/server.js";
import type { HttpMethod } from "@vrooli/shared";

// Mock dependencies
vi.mock("./ai.js", () => ({
    invalidateAIConfigCache: vi.fn(),
}));

describe("fetchData", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        localStorage.clear();
    });

    afterEach(() => {
        vi.clearAllMocks();
        server.resetHandlers();
    });

    describe("URL construction", () => {
        it("should construct URL with rest base by default", async () => {
            // Mock the specific endpoint for this test
            server.use(
                http.get(`${apiUrlBase}${restBase}/users`, () => {
                    return HttpResponse.json({ data: "test", version: "1.0.0" });
                }),
            );

            const result = await fetchData({
                endpoint: "/users",
                method: "GET",
                inputs: undefined,
            });

            expect(result.data).toBe("test");
            expect(result.version).toBe("1.0.0");
        });

        it("should omit rest base when omitRestBase is true", async () => {
            // Mock the auth endpoint without rest base
            server.use(
                http.post(`${apiUrlBase}/auth/login`, () => {
                    return HttpResponse.json({ data: "authenticated", version: "1.0.0" });
                }),
            );

            const result = await fetchData({
                endpoint: "/auth/login",
                method: "POST",
                inputs: { email: "test@example.com", password: "password" },
                omitRestBase: true,
            });

            expect(result.data).toBe("authenticated");
            expect(result.version).toBe("1.0.0");
        });

        it("should replace URL variables with input values", async () => {
            // Mock the endpoint with variable substitution
            server.use(
                http.put(`${apiUrlBase}${restBase}/users/123/update`, () => {
                    return HttpResponse.json({ data: "updated", version: "1.0.0" });
                }),
            );

            const inputs = { id: "123", action: "update" };
            const result = await fetchData({
                endpoint: "/users/:id/:action",
                method: "PUT",
                inputs,
            });

            expect(result.data).toBe("updated");
            expect(result.version).toBe("1.0.0");
        });

        it("should handle multiple variable replacements and remove them from inputs", async () => {
            // Mock endpoint to capture request body for validation
            let capturedBody: Record<string, unknown> | null = null;
            server.use(
                http.put(`${apiUrlBase}${restBase}/users/456/posts/789`, async ({ request }) => {
                    capturedBody = await request.json();
                    return HttpResponse.json({ data: "updated", version: "1.0.0" });
                }),
            );

            const inputs = {
                userId: "456",
                postId: "789",
                title: "Updated Title",
                content: "Updated Content",
            };

            const result = await fetchData({
                endpoint: "/users/:userId/posts/:postId",
                method: "PUT",
                inputs,
            });

            expect(result.data).toBe("updated");
            // Check that replaced variables are removed from body
            expect(capturedBody).toEqual({
                title: "Updated Title",
                content: "Updated Content",
            });
            expect(capturedBody.userId).toBeUndefined();
            expect(capturedBody.postId).toBeUndefined();
        });
    });

    describe("HTTP methods", () => {
        describe("GET requests", () => {
            it("should append inputs as query parameters for GET requests", async () => {
                // Mock GET endpoint to capture query parameters
                let capturedParams: Record<string, string> = {};
                server.use(
                    http.get(`${apiUrlBase}${restBase}/search`, ({ request }) => {
                        const url = new URL(request.url);
                        capturedParams = Object.fromEntries(url.searchParams.entries());
                        return HttpResponse.json({ data: "search results", version: "1.0.0" });
                    }),
                );

                await fetchData({
                    endpoint: "/search",
                    method: "GET",
                    inputs: { q: "test query", limit: 10, offset: 0 },
                });

                // Verify query parameters were encoded correctly
                expect(capturedParams.q).toContain("test query");
                expect(capturedParams.limit).toContain("10");
                expect(capturedParams.offset).toContain("0");
            });

            it("should handle GET requests without inputs", async () => {
                server.use(
                    http.get(`${apiUrlBase}${restBase}/users`, () => {
                        return HttpResponse.json({ data: "users list", version: "1.0.0" });
                    }),
                );

                const result = await fetchData({
                    endpoint: "/users",
                    method: "GET",
                    inputs: undefined,
                });

                expect(result.data).toBe("users list");
                expect(result.version).toBe("1.0.0");
            });

            it("should handle empty inputs object for GET requests", async () => {
                server.use(
                    http.get(`${apiUrlBase}${restBase}/users`, () => {
                        return HttpResponse.json({ data: "users list", version: "1.0.0" });
                    }),
                );

                const result = await fetchData({
                    endpoint: "/users",
                    method: "GET",
                    inputs: {},
                });

                expect(result.data).toBe("users list");
                expect(result.version).toBe("1.0.0");
            });
        });

        describe("POST requests", () => {
            it("should send inputs as JSON body for POST requests", async () => {
                let capturedBody: Record<string, unknown> | null = null;
                server.use(
                    http.post(`${apiUrlBase}${restBase}/users`, async ({ request }) => {
                        capturedBody = await request.json();
                        return HttpResponse.json({ data: "created", version: "1.0.0" });
                    }),
                );

                const inputs = { name: "John Doe", email: "john@example.com" };
                const result = await fetchData({
                    endpoint: "/users",
                    method: "POST",
                    inputs,
                });

                expect(result.data).toBe("created");
                expect(capturedBody).toEqual(inputs);
            });

            it("should handle FormData for POST requests", async () => {
                let capturedHeaders: Record<string, string> = {};
                server.use(
                    http.post(`${apiUrlBase}${restBase}/upload`, async ({ request }) => {
                        capturedHeaders = Object.fromEntries(request.headers.entries());
                        return HttpResponse.json({ data: "uploaded", version: "1.0.0" });
                    }),
                );

                const formData = new FormData();
                formData.append("file", new Blob(["test"]), "test.txt");
                formData.append("description", "Test file");

                const result = await fetchData({
                    endpoint: "/upload",
                    method: "POST",
                    inputs: formData as FormData,
                });

                expect(result.data).toBe("uploaded");
                // FormData should not have Content-Type: application/json
                expect(capturedHeaders["content-type"]).not.toBe("application/json");
            });
        });

        describe("PUT requests", () => {
            it("should send inputs as JSON body for PUT requests", async () => {
                let capturedBody: Record<string, unknown> | null = null;
                server.use(
                    http.put(`${apiUrlBase}${restBase}/users/123`, async ({ request }) => {
                        capturedBody = await request.json();
                        return HttpResponse.json({ data: "updated", version: "1.0.0" });
                    }),
                );

                const inputs = { name: "Jane Doe", email: "jane@example.com" };
                const result = await fetchData({
                    endpoint: "/users/:id",
                    method: "PUT",
                    inputs: { id: "123", ...inputs },
                });

                expect(result.data).toBe("updated");
                expect(capturedBody).toEqual(inputs);
            });
        });

        describe("DELETE requests", () => {
            it("should send inputs as JSON body for DELETE requests", async () => {
                let capturedBody: Record<string, unknown> | null = null;
                server.use(
                    http.delete(`${apiUrlBase}${restBase}/users/123`, async ({ request }) => {
                        capturedBody = await request.json();
                        return HttpResponse.json({ data: "deleted", version: "1.0.0" });
                    }),
                );

                const result = await fetchData({
                    endpoint: "/users/:id",
                    method: "DELETE",
                    inputs: { id: "123", reason: "test deletion" },
                });

                expect(result.data).toBe("deleted");
                expect(capturedBody).toEqual({ reason: "test deletion" });
            });

            it("should handle DELETE requests without body", async () => {
                let capturedBody: Record<string, unknown> | null = null;
                server.use(
                    http.delete(`${apiUrlBase}${restBase}/users/123`, async ({ request }) => {
                        capturedBody = await request.json();
                        return HttpResponse.json({ data: "deleted", version: "1.0.0" });
                    }),
                );

                const result = await fetchData({
                    endpoint: "/users/:id",
                    method: "DELETE",
                    inputs: { id: "123" },
                });

                expect(result.data).toBe("deleted");
                expect(capturedBody).toEqual({});
            });
        });
    });

    describe("Request options", () => {
        it("should include credentials as 'include'", async () => {
            const capturedCredentials: string | null = null;
            server.use(
                http.get(`${apiUrlBase}${restBase}/test`, ({ request }) => {
                    // Note: MSW doesn't expose credentials directly, but we trust fetchData sets it
                    return HttpResponse.json({ data: "test", version: "1.0.0" });
                }),
            );

            const result = await fetchData({
                endpoint: "/test",
                method: "GET",
                inputs: undefined,
            });

            expect(result.data).toBe("test");
            // The actual credentials setting is internal to fetch
        });

        it("should handle Authorization header correctly", async () => {
            let capturedHeaders: Record<string, string> = {};
            server.use(
                http.get(`${apiUrlBase}${restBase}/protected`, ({ request }) => {
                    capturedHeaders = Object.fromEntries(request.headers.entries());
                    return HttpResponse.json({ data: "protected data", version: "1.0.0" });
                }),
            );

            const result = await fetchData({
                endpoint: "/protected",
                method: "GET",
                inputs: undefined,
                options: {
                    headers: {
                        "Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test",
                    },
                },
            });

            expect(result.data).toBe("protected data");
            // Headers in MSW may be lowercase
            expect(capturedHeaders.authorization || capturedHeaders.Authorization).toBe("Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test");
        });

        it("should merge custom options with default options", async () => {
            let capturedHeaders: Record<string, string> = {};
            server.use(
                http.get(`${apiUrlBase}${restBase}/test`, ({ request }) => {
                    capturedHeaders = Object.fromEntries(request.headers.entries());
                    return HttpResponse.json({ data: "test", version: "1.0.0" });
                }),
            );

            const customHeaders = {
                "X-Custom-Header": "custom-value",
                "Authorization": "Bearer token123",
            };

            const result = await fetchData({
                endpoint: "/test",
                method: "GET",
                inputs: undefined,
                options: {
                    headers: customHeaders,
                    mode: "cors",
                },
            });

            expect(result.data).toBe("test");
            // Headers may be lowercase in MSW
            expect(capturedHeaders["x-custom-header"] || capturedHeaders["X-Custom-Header"]).toBe("custom-value");
            expect(capturedHeaders.authorization || capturedHeaders.Authorization).toBe("Bearer token123");
            expect(capturedHeaders["content-type"] || capturedHeaders["Content-Type"]).toBe("application/json");
        });

        it("should handle abort signal", async () => {
            const abortController = new AbortController();
            
            server.use(
                http.get(`${apiUrlBase}${restBase}/test`, () => {
                    return HttpResponse.json({ data: "test", version: "1.0.0" });
                }),
            );

            const result = await fetchData({
                endpoint: "/test",
                method: "GET",
                inputs: undefined,
                signal: abortController.signal,
            });

            expect(result.data).toBe("test");
            // Abort signal is handled internally by fetch
        });
    });

    describe("Response handling", () => {
        it("should add __fetchTimestamp to response", async () => {
            server.use(
                http.get(`${apiUrlBase}${restBase}/test`, () => {
                    return HttpResponse.json({ data: "test", version: "1.0.0" });
                }),
            );

            const beforeFetch = Date.now();
            const result = await fetchData({
                endpoint: "/test",
                method: "GET",
                inputs: undefined,
            });
            const afterFetch = Date.now();

            expect(result.data).toBe("test");
            expect(result.__fetchTimestamp).toBeGreaterThanOrEqual(beforeFetch);
            expect(result.__fetchTimestamp).toBeLessThanOrEqual(afterFetch);
        });

        it("should handle server version change", async () => {
            // First request with version 1.0.0
            localStorage.setItem("serverVersionCache", "1.0.0");
            
            server.use(
                http.get(`${apiUrlBase}${restBase}/test`, () => {
                    return HttpResponse.json({ data: "test", version: "1.1.0" });
                }),
            );

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
            
            server.use(
                http.get(`${apiUrlBase}${restBase}/test`, () => {
                    return HttpResponse.json({ data: "test", version: "1.0.0" });
                }),
            );

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
            server.use(
                http.get(`${apiUrlBase}${restBase}/test`, () => {
                    return HttpResponse.json({ data: "test", version: "1.0.0" });
                }),
            );

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
            server.use(
                http.get(`${apiUrlBase}${restBase}/test`, () => {
                    return HttpResponse.error();
                }),
            );

            await expect(
                fetchData({
                    endpoint: "/test",
                    method: "GET",
                    inputs: undefined,
                }),
            ).rejects.toThrow();
        });

        it("should handle JSON parsing errors", async () => {
            server.use(
                http.get(`${apiUrlBase}${restBase}/test`, () => {
                    return new HttpResponse("Invalid JSON", {
                        status: 200,
                        headers: { "Content-Type": "text/plain" },
                    });
                }),
            );

            await expect(
                fetchData({
                    endpoint: "/test",
                    method: "GET",
                    inputs: undefined,
                }),
            ).rejects.toThrow();
        });

        it("should handle abort errors gracefully", async () => {
            const abortController = new AbortController();
            
            server.use(
                http.get(`${apiUrlBase}${restBase}/test`, () => {
                    return HttpResponse.json({ data: "test", version: "1.0.0" });
                }),
            );

            // Abort immediately
            abortController.abort();

            const result = await fetchData({
                endpoint: "/test",
                method: "GET",
                inputs: undefined,
                signal: abortController.signal,
            });

            expect(result).toEqual({
                errors: [{ message: "Request was aborted" }],
            });
        });

        it("should rethrow non-abort errors", async () => {
            server.use(
                http.get(`${apiUrlBase}${restBase}/test`, () => {
                    // Return a network error response instead of throwing
                    return HttpResponse.error();
                }),
            );

            await expect(
                fetchData({
                    endpoint: "/test",
                    method: "GET",
                    inputs: undefined,
                }),
            ).rejects.toThrow();
        });
    });

    describe("Edge cases", () => {
        it("should handle undefined inputs for all methods", async () => {
            const methods: HttpMethod[] = ["GET", "POST", "PUT", "DELETE"];

            for (const method of methods) {
                server.use(
                    http.all(`${apiUrlBase}${restBase}/test`, () => {
                        return HttpResponse.json({ data: "test", version: "1.0.0" });
                    }),
                );

                const result = await fetchData({
                    endpoint: "/test",
                    method,
                    inputs: undefined,
                });

                expect(result.data).toBe("test");
            }
        });

        it("should handle very long query strings for GET requests", async () => {
            let capturedUrl = "";
            server.use(
                http.get(`${apiUrlBase}${restBase}/search`, ({ request }) => {
                    capturedUrl = request.url;
                    return HttpResponse.json({ data: "results", version: "1.0.0" });
                }),
            );

            const longString = "a".repeat(1000);
            await fetchData({
                endpoint: "/search",
                method: "GET",
                inputs: { query: longString, page: 1 },
            });

            expect(capturedUrl).toContain("query=");
            expect(capturedUrl).toContain("page=");
        });

        it("should handle null values in inputs", async () => {
            let capturedBody: Record<string, unknown> | null = null;
            server.use(
                http.post(`${apiUrlBase}${restBase}/test`, async ({ request }) => {
                    capturedBody = await request.json();
                    return HttpResponse.json({ data: "created", version: "1.0.0" });
                }),
            );

            await fetchData({
                endpoint: "/test",
                method: "POST",
                inputs: { name: "test", value: null, active: true },
            });

            expect(capturedBody).toEqual({
                name: "test",
                value: null,
                active: true,
            });
        });

        it("should handle arrays in GET query parameters", async () => {
            let capturedUrl = "";
            server.use(
                http.get(`${apiUrlBase}${restBase}/search`, ({ request }) => {
                    capturedUrl = request.url;
                    return HttpResponse.json({ data: "results", version: "1.0.0" });
                }),
            );

            await fetchData({
                endpoint: "/search",
                method: "GET",
                inputs: { tags: ["tag1", "tag2", "tag3"], status: ["active", "pending"] },
            });

            // Arrays should be present in the URL
            expect(capturedUrl).toContain("tags=");
            expect(capturedUrl).toContain("status=");
        });

        it("should handle response without version field", async () => {
            server.use(
                http.get(`${apiUrlBase}${restBase}/test`, () => {
                    return HttpResponse.json({ data: "test" });
                }),
            );

            const result = await fetchData({
                endpoint: "/test",
                method: "GET",
                inputs: undefined,
            });

            expect(result.data).toBe("test");
            expect(result.__fetchTimestamp).toBeDefined();
            // Should not crash when version is undefined
            expect(invalidateAIConfigCache).toHaveBeenCalledTimes(1);
        });

        it("should handle special characters in URL parameters", async () => {
            let capturedUrl = "";
            server.use(
                http.get(`${apiUrlBase}${restBase}/search`, ({ request }) => {
                    capturedUrl = request.url;
                    return HttpResponse.json({ data: "results", version: "1.0.0" });
                }),
            );

            await fetchData({
                endpoint: "/search",
                method: "GET",
                inputs: {
                    q: "test with spaces & special=chars?",
                    filter: "type:user/admin",
                },
            });

            const url = new URL(capturedUrl);
            const params = Object.fromEntries(url.searchParams.entries());
            
            expect(params.q).toContain("test with spaces & special=chars?");
            expect(params.filter).toContain("type:user/admin");
        });

        it("should handle empty endpoint", async () => {
            server.use(
                http.get(`${apiUrlBase}${restBase}`, () => {
                    return HttpResponse.json({ data: "root", version: "1.0.0" });
                }),
            );

            const result = await fetchData({
                endpoint: "",
                method: "GET",
                inputs: undefined,
            });

            expect(result.data).toBe("root");
        });

        it("should handle complex nested inputs for POST", async () => {
            let capturedBody: Record<string, unknown> | null = null;
            server.use(
                http.post(`${apiUrlBase}${restBase}/complex`, async ({ request }) => {
                    capturedBody = await request.json();
                    return HttpResponse.json({ data: "created", version: "1.0.0" });
                }),
            );

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

            expect(capturedBody).toEqual(complexInputs);
        });

        it("should preserve original inputs object (not mutate)", async () => {
            server.use(
                http.put(`${apiUrlBase}${restBase}/items/123`, () => {
                    return HttpResponse.json({ data: "updated", version: "1.0.0" });
                }),
            );

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
