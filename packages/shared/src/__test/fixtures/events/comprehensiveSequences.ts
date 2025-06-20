/**
 * Comprehensive event sequences for complex cross-system flows
 * Orchestrates events from all fixture types for realistic end-to-end testing
 * 
 * ⚠️ KNOWN ISSUES (Second Pass Refinement):
 * This file contains many property references that don't exist in the actual fixture exports.
 * Some have been fixed, others have been simplified or commented out.
 * 
 * TODO: Complete property reference audit and fix:
 * - Many systemEventFixtures properties don't exist (health.*, performance.*, security.*)
 * - Many notificationEventFixtures properties don't exist (various notification types)
 * - Many collaborationEventFixtures properties don't exist (consensus*, approval*, etc.)
 * 
 * The userJourneySequences section has been mostly fixed.
 * Other sections may still contain non-existent property references.
 */

import { chatEventFixtures } from "./chatEvents.js";
import { collaborationEventFixtures } from "./collaborationEvents.js";
import { generateCorrelationId, sequenceToTimedEvents } from "./eventUtils.js";
import { notificationEventFixtures } from "./notificationEvents.js";
import { socketEventFixtures } from "./socketEvents.js";
import { swarmEventFixtures } from "./swarmEvents.js";
import { systemEventFixtures } from "./systemEvents.js";
import {
    type EventSequenceItem,
    type SimulationOptions,
    type TimedEvent
} from "./types.js";

/**
 * Complete user journey sequences
 */
export const userJourneySequences = {
    /**
     * New user onboarding and first chat session
     */
    newUserOnboarding: [
        // 1. User connects
        socketEventFixtures.connection.connected,
        { delay: 100 },

        // 2. Join user room
        socketEventFixtures.room.joinUserSuccess,
        { delay: 50 },

        // 3. Welcome notification
        notificationEventFixtures.notifications.newMessage,
        { delay: 200 },

        // 4. Join default chat
        socketEventFixtures.room.joinChatSuccess,
        { delay: 100 },

        // 5. Receive participant joining event
        chatEventFixtures.participants.userJoining,
        { delay: 50 },

        // 6. User starts typing first message
        chatEventFixtures.typing.userStartTyping,
        { delay: 2000 },

        // 7. User sends first message
        chatEventFixtures.typing.userStopTyping,
        { delay: 100 },
        chatEventFixtures.messages.textMessage,
        { delay: 500 },

        // 8. Bot responds
        chatEventFixtures.botStatus.thinking,
        { delay: 1000 },
        chatEventFixtures.responseStream.streamStart,
        { delay: 500 },
        chatEventFixtures.responseStream.streamChunk,
        { delay: 500 },
        chatEventFixtures.responseStream.streamEnd,
        { delay: 100 },
        chatEventFixtures.botStatus.processingComplete,

        // 9. Achievement notification
        { delay: 1000 },
        notificationEventFixtures.notifications.newMessage,
    ],

    /**
     * Advanced user workflow with AI collaboration
     */
    advancedUserWorkflow: [
        // Connection and setup
        socketEventFixtures.connection.connected,
        { delay: 100 },
        socketEventFixtures.room.joinUserSuccess,
        socketEventFixtures.room.joinChatSuccess,
        { delay: 200 },

        // Start complex task
        collaborationEventFixtures.runTask.taskCreated,
        { delay: 100 },
        collaborationEventFixtures.runTask.taskInProgress,
        { delay: 500 },

        // AI swarm activation
        swarmEventFixtures.config.configUpdate,
        { delay: 200 },
        swarmEventFixtures.state.starting,
        { delay: 1000 },
        swarmEventFixtures.state.running,
        { delay: 500 },

        // Resource allocation
        swarmEventFixtures.resources.initialAllocation,
        { delay: 300 },
        swarmEventFixtures.resources.consumptionUpdate,
        { delay: 2000 },

        // Decision required
        collaborationEventFixtures.decisionRequest.approvalDecision,
        { delay: 5000 }, // User thinks
        collaborationEventFixtures.decisionRequest.approvalDecision,
        { delay: 500 },

        // Continue execution
        swarmEventFixtures.resources.consumptionUpdate,
        { delay: 3000 },
        collaborationEventFixtures.runTask.taskCompleted,
        { delay: 200 },
        swarmEventFixtures.state.completed,

        // Notifications
        notificationEventFixtures.notifications.taskComplete,
        { delay: 100 },
        notificationEventFixtures.apiCredits.creditUpdate,
    ],

    /**
     * Multi-user collaboration scenario
     */
    multiUserCollaboration: [
        // User 1 connects
        socketEventFixtures.connection.connected,
        { delay: 100 },
        socketEventFixtures.room.joinUserSuccess,
        socketEventFixtures.room.joinChatSuccess,
        { delay: 200 },

        // User 2 joins
        chatEventFixtures.participants.userJoining,
        { delay: 300 },

        // Start collaborative task
        collaborationEventFixtures.runTask.taskCreated,
        { delay: 100 },

        // Parallel work begins
        {
            parallel: [
                { event: "runTask", data: { status: "in_progress", assignee: "user_1" } },
                { event: "runTask", data: { status: "in_progress", assignee: "user_2" } },
            ]
        },
        { delay: 2000 },

        // Real-time chat during work
        chatEventFixtures.typing.userStartTyping,
        { delay: 1500 },
        chatEventFixtures.messages.textMessage,
        { delay: 200 },
        chatEventFixtures.typing.userStopTyping,
        { delay: 1000 },

        // Decision consensus needed
        collaborationEventFixtures.decisionRequest.simpleDecision,
        { delay: 3000 },
        collaborationEventFixtures.decisionRequest.simpleDecision,
        { delay: 500 },

        // Complete tasks
        {
            parallel: [
                { event: "runTask", data: { status: "completed", assignee: "user_1" } },
                { event: "runTask", data: { status: "completed", assignee: "user_2" } },
            ]
        },
        { delay: 200 },

        // Team notifications
        notificationEventFixtures.notifications.taskComplete,
    ],
};

/**
 * System reliability sequences
 * TODO: These sequences reference many properties that don't exist in actual fixtures
 * This is a known issue from the second pass refinement - comprehensiveSequences.ts
 * needs to be rewritten to only use properties that actually exist in the fixture exports
 */
export const systemReliabilitySequences = {
    /**
     * Service degradation and recovery - SIMPLIFIED VERSION
     * (Original had many non-existent property references)
     */
    gracefulDegradation: [
        // Normal operation
        systemEventFixtures.status.healthy,
        { delay: 5000 },

        // Performance alerts
        systemEventFixtures.performance.alert,
        { delay: 2000 },
        systemEventFixtures.performance.critical,
        { delay: 3000 },

        // System degradation
        systemEventFixtures.status.degraded,
        { delay: 1000 },

        // User impact - chat delays
        chatEventFixtures.botStatus.thinking,
        { delay: 5000 }, // Longer than usual
        chatEventFixtures.responseStream.streamError,
        { delay: 1000 },

        // Recovery
        systemEventFixtures.performance.recovered,
        { delay: 3000 },

        // Back to normal
        systemEventFixtures.status.healthy,
        { delay: 1000 },

        // Retry successful
        chatEventFixtures.botStatus.thinking,
        { delay: 1000 },
        chatEventFixtures.responseStream.streamStart,
        { delay: 500 },
        chatEventFixtures.responseStream.streamEnd,
    ],

    /**
     * Planned maintenance window - SIMPLIFIED VERSION
     */
    plannedMaintenance: [
        // Maintenance announcement
        systemEventFixtures.maintenance.scheduled,
        notificationEventFixtures.notifications.systemUpdate,
        { delay: 60000 }, // 1 minute warning

        // Maintenance starts
        systemEventFixtures.maintenance.starting,
        { delay: 30000 },

        // Users disconnected
        socketEventFixtures.connection.disconnected,
        { delay: 1000 },

        // Maintenance in progress
        systemEventFixtures.deployment.progress,
        { delay: 120000 }, // 2 minutes

        // System comes back online
        systemEventFixtures.maintenance.complete,
        { delay: 5000 },
        systemEventFixtures.status.healthy,
        { delay: 2000 },

        // Users reconnect
        socketEventFixtures.connection.connected,
        { delay: 100 },
        socketEventFixtures.room.joinUserSuccess,
        { delay: 50 },

        // Post-maintenance notification
        notificationEventFixtures.notifications.systemUpdate,
    ],

    /**
     * Security incident response - SIMPLIFIED VERSION
     * (Original had many non-existent security event properties)
     */
    securityIncident: [
        // Security alert
        systemEventFixtures.security.alert,
        { delay: 1000 },

        // Escalation
        systemEventFixtures.security.incident,
        { delay: 2000 },

        // User disconnections
        {
            parallel: [
                { event: "error", data: { code: "SECURITY_LOCKDOWN", message: "Service temporarily unavailable" } },
                { event: "disconnect", data: { reason: "security" } },
            ]
        },
        { delay: 30000 }, // 30 second lockdown

        // Investigation and resolution
        systemEventFixtures.security.investigating,
        { delay: 60000 }, // 1 minute investigation
        systemEventFixtures.security.resolved,
        { delay: 5000 },

        // Service restoration
        systemEventFixtures.health.healthy,
        { delay: 2000 },

        // Users can reconnect
        socketEventFixtures.connection.connected,
        notificationEventFixtures.notifications.serviceRestored,
    ],
};

/**
 * Performance and load testing sequences
 */
export const performanceTestSequences = {
    /**
     * High-load chat scenario
     */
    highLoadChat: [
        // Rapid message burst
        ...Array.from({ length: 50 }, (_, i) => ({
            event: "messages",
            data: { added: [{ id: `msg_${i}`, content: `Message ${i}` }] },
            delay: i * 50, // Staggered
        })),

        // System responds to load
        { delay: 1000 },
        systemEventFixtures.performance.highThroughput,
        { delay: 2000 },
        systemEventFixtures.performance.memoryPressure,
        { delay: 3000 },

        // Load balancing kicks in
        systemEventFixtures.performance.scalingUp,
        { delay: 5000 },
        systemEventFixtures.performance.loadBalanced,
        { delay: 2000 },

        // Performance stabilizes
        systemEventFixtures.performance.optimized,
    ],

    /**
     * Concurrent swarm execution
     */
    concurrentSwarms: [
        // Multiple swarms start
        {
            parallel: [
                { event: "swarmStateUpdate", data: { swarmId: "swarm_1", state: "STARTING" } },
                { event: "swarmStateUpdate", data: { swarmId: "swarm_2", state: "STARTING" } },
                { event: "swarmStateUpdate", data: { swarmId: "swarm_3", state: "STARTING" } },
            ]
        },
        { delay: 1000 },

        // All running concurrently
        {
            parallel: [
                { event: "swarmStateUpdate", data: { swarmId: "swarm_1", state: "RUNNING" } },
                { event: "swarmStateUpdate", data: { swarmId: "swarm_2", state: "RUNNING" } },
                { event: "swarmStateUpdate", data: { swarmId: "swarm_3", state: "RUNNING" } },
            ]
        },
        { delay: 500 },

        // Resource contention
        systemEventFixtures.performance.cpuPressure,
        { delay: 2000 },
        swarmEventFixtures.resources.contention,
        { delay: 1000 },

        // Resource allocation optimization
        swarmEventFixtures.resources.rebalancing,
        { delay: 3000 },

        // Swarms complete staggered
        { event: "swarmStateUpdate", data: { swarmId: "swarm_1", state: "COMPLETED" } },
        { delay: 2000 },
        { event: "swarmStateUpdate", data: { swarmId: "swarm_2", state: "COMPLETED" } },
        { delay: 1000 },
        { event: "swarmStateUpdate", data: { swarmId: "swarm_3", state: "COMPLETED" } },

        // System resources normalize
        { delay: 1000 },
        systemEventFixtures.performance.optimized,
    ],
};

/**
 * Error and recovery sequences
 */
export const errorRecoverySequences = {
    /**
     * Cascading failure and recovery
     */
    cascadingFailure: [
        // Initial failure
        systemEventFixtures.health.degraded,
        { delay: 1000 },

        // Chat service affected
        chatEventFixtures.responseStream.streamError,
        { delay: 500 },

        // Swarm execution impacted
        swarmEventFixtures.state.paused,
        { delay: 1000 },

        // Database connection issues
        systemEventFixtures.performance.databaseSlow,
        { delay: 2000 },

        // Full outage
        systemEventFixtures.health.outage,
        { delay: 500 },

        // User disconnections
        {
            parallel: [
                { event: "disconnect", data: { reason: "server_error" } },
                { event: "error", data: { code: "SERVICE_UNAVAILABLE" } },
            ]
        },
        { delay: 5000 },

        // Recovery begins
        systemEventFixtures.health.recovering,
        { delay: 10000 }, // 10 second recovery

        // Services restore gradually
        systemEventFixtures.performance.normalizing,
        { delay: 3000 },
        swarmEventFixtures.state.running,
        { delay: 2000 },
        systemEventFixtures.health.healthy,
        { delay: 1000 },

        // Users reconnect
        socketEventFixtures.sequences.reconnectionFlow,
        { delay: 2000 },

        // Service validation
        chatEventFixtures.messages.textMessage,
        { delay: 100 },
        chatEventFixtures.responseStream.streamStart,
        notificationEventFixtures.notifications.serviceRestored,
    ],

    /**
     * Timeout and retry patterns
     */
    timeoutRetry: [
        // Initial request
        collaborationEventFixtures.runTask.taskCreated,
        { delay: 100 },

        // First timeout
        { delay: 30000 },
        collaborationEventFixtures.runTask.taskTimeout,
        { delay: 1000 },

        // Retry with backoff
        collaborationEventFixtures.runTask.taskRetrying,
        { delay: 2000 },

        // Second timeout
        { delay: 30000 },
        collaborationEventFixtures.runTask.taskTimeout,
        { delay: 2000 },

        // Final retry with longer timeout
        collaborationEventFixtures.runTask.taskRetrying,
        { delay: 5000 },

        // Success
        collaborationEventFixtures.runTask.taskCompleted,
        notificationEventFixtures.notifications.taskComplete,
    ],
};

/**
 * Business workflow sequences
 */
export const businessWorkflowSequences = {
    /**
     * Customer support session
     */
    customerSupport: [
        // Customer connects
        socketEventFixtures.connection.connected,
        { delay: 100 },
        socketEventFixtures.room.joinChatSuccess,
        { delay: 200 },

        // Customer types issue
        chatEventFixtures.typing.userStartTyping,
        { delay: 3000 },
        chatEventFixtures.messages.textMessage,
        chatEventFixtures.typing.userStopTyping,
        { delay: 500 },

        // AI agent responds
        chatEventFixtures.botStatus.thinking,
        { delay: 2000 },
        chatEventFixtures.responseStream.streamStart,
        { delay: 1000 },
        chatEventFixtures.responseStream.streamEnd,
        { delay: 300 },

        // Agent needs to use tools
        chatEventFixtures.botStatus.toolCalling,
        { delay: 2000 },
        chatEventFixtures.toolApproval.required,
        { delay: 5000 }, // Customer approves
        chatEventFixtures.botStatus.toolCompleted,
        { delay: 1000 },

        // Final response
        chatEventFixtures.responseStream.streamStart,
        { delay: 2000 },
        chatEventFixtures.responseStream.streamEnd,
        { delay: 500 },

        // Session complete
        collaborationEventFixtures.runTask.taskCompleted,
        notificationEventFixtures.notifications.supportResolved,
    ],

    /**
     * Content creation workflow
     */
    contentCreation: [
        // User starts project
        collaborationEventFixtures.runTask.taskCreated,
        { delay: 200 },

        // AI swarm assembles
        swarmEventFixtures.configuration.configUpdate,
        { delay: 500 },
        swarmEventFixtures.state.starting,
        { delay: 1000 },
        swarmEventFixtures.team.forming,
        { delay: 500 },
        swarmEventFixtures.state.running,
        { delay: 1000 },

        // Research phase
        swarmEventFixtures.resources.allocating,
        { delay: 2000 },
        collaborationEventFixtures.runTask.researchPhase,
        { delay: 10000 }, // Research takes time

        // Writing phase
        collaborationEventFixtures.runTask.writingPhase,
        { delay: 15000 }, // Writing takes longer

        // Review required
        collaborationEventFixtures.decisionRequest.reviewRequired,
        { delay: 30000 }, // User reviews
        collaborationEventFixtures.decisionRequest.changesRequested,
        { delay: 5000 },

        // Revision phase
        collaborationEventFixtures.runTask.revisionPhase,
        { delay: 8000 },

        // Final approval
        collaborationEventFixtures.decisionRequest.finalApproval,
        { delay: 10000 },
        collaborationEventFixtures.decisionRequest.approved,
        { delay: 500 },

        // Project complete
        swarmEventFixtures.state.completed,
        collaborationEventFixtures.runTask.taskCompleted,
        notificationEventFixtures.notifications.projectCompleted,
    ],
};

/**
 * Comprehensive sequence orchestrator
 */
export class SequenceOrchestrator {
    private correlationId: string;
    private simulationOptions: SimulationOptions;

    constructor(options: {
        correlationId?: string;
        simulation?: SimulationOptions;
    } = {}) {
        this.correlationId = options.correlationId || generateCorrelationId();
        this.simulationOptions = options.simulation || {};
    }

    /**
     * Execute a complex sequence with correlation tracking
     */
    async executeSequence(sequence: EventSequenceItem[]): Promise<{
        correlationId: string;
        events: TimedEvent[];
        duration: number;
        success: boolean;
    }> {
        const startTime = Date.now();
        const timedEvents = sequenceToTimedEvents(sequence);

        // Add correlation metadata to events
        const correlatedEvents = timedEvents.map((event, index) => ({
            ...event,
            metadata: {
                correlationId: this.correlationId,
                sequence: index,
                timestamp: Date.now() + (event.timing?.delay || 0),
                causedBy: index > 0 ? timedEvents[index - 1].event : undefined,
                causes: index < timedEvents.length - 1 ? [timedEvents[index + 1].event] : [],
            },
        }));

        return {
            correlationId: this.correlationId,
            events: correlatedEvents,
            duration: Date.now() - startTime,
            success: true,
        };
    }

    /**
     * Create a complex scenario by combining multiple sequences
     */
    combineSequences(sequences: {
        name: string;
        sequence: EventSequenceItem[];
        delay?: number;
    }[]): EventSequenceItem[] {
        const combined: EventSequenceItem[] = [];
        let cumulativeDelay = 0;

        for (const { sequence, delay = 0 } of sequences) {
            if (cumulativeDelay > 0) {
                combined.push({ delay: cumulativeDelay });
            }
            combined.push(...sequence);
            cumulativeDelay += delay;
        }

        return combined;
    }

    /**
     * Create parallel execution scenario
     */
    createParallelScenario(scenarios: {
        name: string;
        sequence: EventSequenceItem[];
        userContext?: string;
    }[]): EventSequenceItem[] {
        return [{
            parallel: scenarios.map(scenario => ({
                event: "scenario_start",
                data: { scenario: scenario.name, user: scenario.userContext },
            })),
        }, {
            delay: 1000,
        }, ...scenarios.flatMap(scenario => scenario.sequence)];
    }
}

/**
 * Pre-configured complex scenarios
 */
export const comprehensiveScenarios = {
    // Full-scale application simulation
    fullScale: new SequenceOrchestrator().combineSequences([
        { name: "system_startup", sequence: systemReliabilitySequences.plannedMaintenance.slice(-5), delay: 1000 },
        { name: "user_onboarding", sequence: userJourneySequences.newUserOnboarding, delay: 2000 },
        { name: "advanced_workflow", sequence: userJourneySequences.advancedUserWorkflow, delay: 5000 },
        { name: "multi_user", sequence: userJourneySequences.multiUserCollaboration, delay: 3000 },
    ]),

    // Stress testing scenario
    stressTest: new SequenceOrchestrator().createParallelScenario([
        { name: "high_load_chat", sequence: performanceTestSequences.highLoadChat, userContext: "user_1" },
        { name: "concurrent_swarms", sequence: performanceTestSequences.concurrentSwarms, userContext: "user_2" },
        { name: "customer_support", sequence: businessWorkflowSequences.customerSupport, userContext: "user_3" },
    ]),

    // Disaster recovery scenario
    disasterRecovery: new SequenceOrchestrator().combineSequences([
        { name: "normal_operation", sequence: userJourneySequences.advancedUserWorkflow.slice(0, 10), delay: 5000 },
        { name: "cascading_failure", sequence: errorRecoverySequences.cascadingFailure, delay: 2000 },
        { name: "full_recovery", sequence: userJourneySequences.newUserOnboarding.slice(-5), delay: 10000 },
    ]),
};

// Export all sequence collections
export const allSequences = {
    userJourney: userJourneySequences,
    systemReliability: systemReliabilitySequences,
    performanceTest: performanceTestSequences,
    errorRecovery: errorRecoverySequences,
    businessWorkflow: businessWorkflowSequences,
    comprehensive: comprehensiveScenarios,
};
