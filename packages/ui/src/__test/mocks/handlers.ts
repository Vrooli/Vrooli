/**
 * MSW request handlers for API mocking in tests.
 * 
 * This file provides default handlers for common API patterns and a fallback
 * for unmocked requests to prevent actual network calls during testing.
 */

import { http, HttpResponse } from "msw";

/**
 * Default MSW handlers for common API endpoints.
 * Individual tests can override these using server.use() for specific scenarios.
 */
export const handlers = [
  // Health check endpoint
  http.get("*/api/health", () => {
    return HttpResponse.json({ status: "ok", version: "1.0.0" });
  }),

  // Auth endpoints
  http.post("*/api/auth/login", () => {
    return HttpResponse.json({
      data: { user: { id: "test-user", email: "test@example.com" } },
      version: "1.0.0",
    });
  }),

  http.post("*/api/auth/signup", () => {
    return HttpResponse.json({
      data: { user: { id: "new-user", email: "new@example.com" } },
      version: "1.0.0",
    });
  }),

  // Error reporting endpoint
  http.post("*/api/error-reports", () => {
    return HttpResponse.json({
      data: { success: true, reportId: "test-report-id" },
      version: "1.0.0",
    });
  }),

  // Resources endpoint - commonly used in tests  
  http.get("http://localhost:3000/api/v2/resources", ({ request }) => {
    console.log("✅ MSW intercepted resources request:", request.url);
    const url = new URL(request.url);
    const searchParams = url.searchParams;
    const take = parseInt(searchParams.get("take") || "20");
    
    return HttpResponse.json({
      data: {
        edges: Array.from({ length: Math.min(take, 5) }, (_, i) => ({
          node: {
            id: `mock-resource-${i}`,
            name: `Mock Resource ${i + 1}`,
            description: `Mock description for resource ${i + 1}`,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
          },
        })),
        pageInfo: {
          hasNextPage: false,
          hasPreviousPage: false,
          startCursor: "start",
          endCursor: "end",
        },
      },
      version: "2.0.0",
    });
  }),

  // Also add wildcard version for resources
  http.get("*/api/v2/resources", ({ request }) => {
    console.log("✅ MSW intercepted resources request (wildcard):", request.url);
    const url = new URL(request.url);
    const searchParams = url.searchParams;
    const take = parseInt(searchParams.get("take") || "20");
    
    return HttpResponse.json({
      data: {
        edges: Array.from({ length: Math.min(take, 5) }, (_, i) => ({
          node: {
            id: `mock-resource-${i}`,
            name: `Mock Resource ${i + 1}`,
            description: `Mock description for resource ${i + 1}`,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
          },
        })),
        pageInfo: {
          hasNextPage: false,
          hasPreviousPage: false,
          startCursor: "start",
          endCursor: "end",
        },
      },
      version: "2.0.0",
    });
  }),

  // Generic v2 API endpoints
  http.get("*/api/v2/*", ({ request }) => {
    const url = new URL(request.url);
    const pathSegments = url.pathname.split("/");
    const resource = pathSegments[pathSegments.length - 1];
    
    return HttpResponse.json({
      data: {
        edges: [],
        pageInfo: {
          hasNextPage: false,
          hasPreviousPage: false,
          startCursor: null,
          endCursor: null,
        },
      },
      version: "2.0.0",
    });
  }),

  http.post("*/api/v2/*", ({ request }) => {
    const url = new URL(request.url);
    const pathSegments = url.pathname.split("/");
    const resource = pathSegments[pathSegments.length - 1];
    
    return HttpResponse.json({
      data: { 
        id: `mock-${resource}-id`,
        message: `Mock ${resource} created successfully`, 
      },
      version: "2.0.0",
    });
  }),

  // Generic REST endpoints - these provide sensible defaults
  http.get("*/api/rest/*", ({ request }) => {
    const url = new URL(request.url);
    const path = url.pathname.split("/").pop();
    
    return HttpResponse.json({
      data: { message: `Mock response for GET ${path}` },
      version: "1.0.0",
    });
  }),

  http.post("*/api/rest/*", ({ request }) => {
    const url = new URL(request.url);
    const path = url.pathname.split("/").pop();
    
    return HttpResponse.json({
      data: { message: `Mock response for POST ${path}`, id: "mock-id" },
      version: "1.0.0",
    });
  }),

  http.put("*/api/rest/*", ({ request }) => {
    const url = new URL(request.url);
    const path = url.pathname.split("/").pop();
    
    return HttpResponse.json({
      data: { message: `Mock response for PUT ${path}` },
      version: "1.0.0",
    });
  }),

  http.delete("*/api/rest/*", ({ request }) => {
    const url = new URL(request.url);
    const path = url.pathname.split("/").pop();
    
    return HttpResponse.json({
      data: { message: `Mock response for DELETE ${path}` },
      version: "1.0.0",
    });
  }),

  // Default fallback handler - prevents actual network requests
  http.all("*", ({ request }) => {
    console.warn(
      `🚨 Unhandled ${request.method} request to ${request.url}\n` +
      "Consider adding a specific MSW handler for this endpoint in your test.",
    );
    
    return HttpResponse.json(
      { 
        errors: [{ 
          message: `Network request not mocked: ${request.method} ${request.url}`,
          code: "UnmockedRequest",
        }], 
      },
      { status: 500 },
    );
  }),
];

/**
 * Utility function to create a successful API response.
 */
export const createSuccessResponse = (data: any, version = "1.0.0") => {
  return HttpResponse.json({ data, version });
};

/**
 * Utility function to create an error API response.
 */
export const createErrorResponse = (message: string, code = "TestError", status = 400) => {
  return HttpResponse.json(
    { errors: [{ message, code }] },
    { status },
  );
};

/**
 * Utility function to create a delayed response for testing loading states.
 */
export const createDelayedResponse = (data: any, delay = 100, version = "1.0.0") => {
  return new Promise(resolve => {
    setTimeout(() => {
      resolve(HttpResponse.json({ data, version }));
    }, delay);
  });
};
