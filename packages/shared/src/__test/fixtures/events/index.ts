/**
 * Central export point for all event fixtures
 * 
 * These fixtures provide consistent real-time event scenarios for testing
 * WebSocket, chat, swarm execution, and other live features.
 */

export * from "./socketEvents.js";
export * from "./chatEvents.js";
export * from "./swarmEvents.js";
export * from "./notificationEvents.js";
export * from "./collaborationEvents.js";
export * from "./systemEvents.js";

// Re-export all event fixtures as a namespace for convenience
export { socketEventFixtures } from "./socketEvents.js";
export { chatEventFixtures } from "./chatEvents.js";
export { swarmEventFixtures } from "./swarmEvents.js";
export { notificationEventFixtures } from "./notificationEvents.js";
export { collaborationEventFixtures } from "./collaborationEvents.js";
export { systemEventFixtures } from "./systemEvents.js";