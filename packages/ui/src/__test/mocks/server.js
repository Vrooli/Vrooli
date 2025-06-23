/**
 * MSW server setup for UI testing
 * 
 * This provides Mock Service Worker (MSW) setup for component tests
 * that need to mock API responses without hitting the real database.
 */

import { setupServer } from "msw/node";

// Create MSW server instance
export const server = setupServer();

// Setup MSW server for tests
beforeAll(() => {
    server.listen({ onUnhandledRequest: "warn" });
});

afterEach(() => {
    server.resetHandlers();
});

afterAll(() => {
    server.close();
});

export default server;
