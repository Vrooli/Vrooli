/**
 * MSW server setup for UI testing
 * 
 * This provides Mock Service Worker (MSW) setup for component tests
 * that need to mock API responses without hitting the real database.
 */

import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";

// Common API handlers for UI components
const handlers = [
    // Resources API endpoint that's frequently called by components
    http.get('http://localhost:3000/api/v2/resources', () => {
        return HttpResponse.json({
            data: {
                edges: [
                    {
                        node: {
                            id: "resource-1",
                            name: "Sample Resource",
                            description: "A sample resource for testing",
                        },
                    },
                    {
                        node: {
                            id: "resource-2", 
                            name: "Another Resource",
                            description: "Another sample resource",
                        },
                    },
                ],
                pageInfo: {
                    hasNextPage: false,
                    hasPreviousPage: false,
                },
            },
        });
    }),

    // Auth endpoints
    http.post('http://localhost:3000/api/v2/auth/emailSignup', () => {
        return HttpResponse.json({
            data: {
                __typename: 'Session',
                id: 'session-123',
                isLoggedIn: true,
                user: {
                    id: 'user-123',
                    name: 'Test User',
                    email: 'test@example.com',
                },
                roles: [],
            },
        });
    }),

    // General catch-all for unspecified API endpoints
    http.get('http://localhost:3000/api/*', () => {
        return HttpResponse.json({ data: null });
    }),

    http.post('http://localhost:3000/api/*', () => {
        return HttpResponse.json({ data: null });
    }),
];

// Create MSW server instance with default handlers
export const server = setupServer(...handlers);

export default server;
