/**
 * Central export point for the Enhanced Event Fixture System
 * 
 * Provides comprehensive real-time event simulation capabilities using the
 * factory pattern for type-safe, realistic testing scenarios.
 */

// Core infrastructure
export * from "./BaseEventFactory.js";
export * from "./eventUtils.js";
export * from "./MockSocketEmitter.js";
export * from "./types.js";

// Event fixture modules
export * from "./chatEvents.js";
export * from "./collaborationEvents.js";
export * from "./comprehensiveSequences.js";
export * from "./notificationEvents.js";
export * from "./socketEvents.js";
export * from "./swarmEvents.js";
export * from "./systemEvents.js";

// Legacy exports for backward compatibility
export { chatEventFixtures } from "./chatEvents.js";
export { collaborationEventFixtures } from "./collaborationEvents.js";
export { notificationEventFixtures } from "./notificationEvents.js";
export { socketEventFixtures } from "./socketEvents.js";
export { swarmEventFixtures } from "./swarmEvents.js";
export { systemEventFixtures } from "./systemEvents.js";

// Factory instances for advanced usage
export {
    connectionFactory,
    errorFactory, reconnectionFactory, roomFactory
} from "./socketEvents.js";

// Comprehensive sequences
export {
    allSequences, businessWorkflowSequences,
    comprehensiveScenarios, errorRecoverySequences, performanceTestSequences, SequenceOrchestrator, systemReliabilitySequences, userJourneySequences
} from "./comprehensiveSequences.js";

// Utility functions
export {
    applyNetworkDelay, collectEvents, createEventAssertions, delay, EventCorrelator, generateCorrelationId, networkPresets, sequenceToTimedEvents, shouldDropPacket, StateDiffer, TimingAnalyzer,
    waitForEvent
} from "./eventUtils.js";

// Main fixture collection for easy access
export const eventFixtures = {
    socket: socketEventFixtures,
    chat: chatEventFixtures,
    swarm: swarmEventFixtures,
    notification: notificationEventFixtures,
    collaboration: collaborationEventFixtures,
    system: systemEventFixtures,
};

// Quick access to all sequences
export const eventSequences = {
    userJourney: userJourneySequences,
    systemReliability: systemReliabilitySequences,
    comprehensive: comprehensiveScenarios,
};
