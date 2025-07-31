/**
 * Type guards for event handling with MQTT-style subscriptions
 * 
 * This module provides type-safe event handling for the unified event system.
 * Since the event bus supports MQTT-style pattern subscriptions (e.g., "chat/*"),
 * handlers receive union types and need runtime type checking to safely access
 * event-specific properties.
 */

import type { SocketEvent, SocketEventPayloads } from "@vrooli/shared";
import { EventTypes } from "@vrooli/shared";
import type { ServiceEvent } from "./types.js";

/**
 * Type map for all official event types to their payload data
 */
export type EventTypeDataMap = SocketEventPayloads;

/**
 * Type-safe event type matcher for specific event types
 * @param event - The service event to check
 * @param eventType - The specific event type to match against
 * @returns Type predicate indicating if the event matches the specified type
 */
export function matchEventType<T extends SocketEvent>(
    event: ServiceEvent,
    eventType: T,
): event is ServiceEvent<EventTypeDataMap[T]> {
    return event.type === eventType;
}

/**
 * Production event type guards for all official EventTypes
 * These guards provide type-safe access to event data for specific event types.
 */
export const EventTypeGuards = {
    // ===== Chat Events =====
    isChatMessageAdded: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.CHAT.MESSAGE_ADDED]> =>
        event.type === EventTypes.CHAT.MESSAGE_ADDED,

    isChatMessageUpdated: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.CHAT.MESSAGE_UPDATED]> =>
        event.type === EventTypes.CHAT.MESSAGE_UPDATED,

    isChatMessageRemoved: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.CHAT.MESSAGE_REMOVED]> =>
        event.type === EventTypes.CHAT.MESSAGE_REMOVED,

    isChatParticipantsChanged: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.CHAT.PARTICIPANTS_CHANGED]> =>
        event.type === EventTypes.CHAT.PARTICIPANTS_CHANGED,

    isChatTypingUpdated: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.CHAT.TYPING_UPDATED]> =>
        event.type === EventTypes.CHAT.TYPING_UPDATED,

    isResponseStreamChunk: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.CHAT.RESPONSE_STREAM_CHUNK]> =>
        event.type === EventTypes.CHAT.RESPONSE_STREAM_CHUNK,

    isResponseStreamEnd: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.CHAT.RESPONSE_STREAM_END]> =>
        event.type === EventTypes.CHAT.RESPONSE_STREAM_END,

    isResponseStreamError: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.CHAT.RESPONSE_STREAM_ERROR]> =>
        event.type === EventTypes.CHAT.RESPONSE_STREAM_ERROR,

    isReasoningStreamChunk: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.CHAT.REASONING_STREAM_CHUNK]> =>
        event.type === EventTypes.CHAT.REASONING_STREAM_CHUNK,

    isReasoningStreamEnd: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.CHAT.REASONING_STREAM_END]> =>
        event.type === EventTypes.CHAT.REASONING_STREAM_END,

    isReasoningStreamError: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.CHAT.REASONING_STREAM_ERROR]> =>
        event.type === EventTypes.CHAT.REASONING_STREAM_ERROR,

    isLlmTasksUpdated: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.CHAT.LLM_TASKS_UPDATED]> =>
        event.type === EventTypes.CHAT.LLM_TASKS_UPDATED,

    isBotStatusUpdated: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.CHAT.BOT_STATUS_UPDATED]> =>
        event.type === EventTypes.CHAT.BOT_STATUS_UPDATED,

    isBotTypingUpdated: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.CHAT.BOT_TYPING_UPDATED]> =>
        event.type === EventTypes.CHAT.BOT_TYPING_UPDATED,

    isCancellationRequested: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.CHAT.CANCELLATION_REQUESTED]> =>
        event.type === EventTypes.CHAT.CANCELLATION_REQUESTED,

    // ===== Tool Events =====
    isToolCalled: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.TOOL.CALLED]> =>
        event.type === EventTypes.TOOL.CALLED,

    isToolCompleted: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.TOOL.COMPLETED]> =>
        event.type === EventTypes.TOOL.COMPLETED,

    isToolFailed: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.TOOL.FAILED]> =>
        event.type === EventTypes.TOOL.FAILED,

    isToolApprovalRequired: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.TOOL.APPROVAL_REQUIRED]> =>
        event.type === EventTypes.TOOL.APPROVAL_REQUIRED,

    isToolApprovalGranted: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.TOOL.APPROVAL_GRANTED]> =>
        event.type === EventTypes.TOOL.APPROVAL_GRANTED,

    isToolApprovalRejected: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.TOOL.APPROVAL_REJECTED]> =>
        event.type === EventTypes.TOOL.APPROVAL_REJECTED,

    isToolApprovalTimeout: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.TOOL.APPROVAL_TIMEOUT]> =>
        event.type === EventTypes.TOOL.APPROVAL_TIMEOUT,

    isToolExecutionRequested: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.TOOL.EXECUTION_REQUESTED]> =>
        event.type === EventTypes.TOOL.EXECUTION_REQUESTED,

    isToolExecutionCompleted: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.TOOL.EXECUTION_COMPLETED]> =>
        event.type === EventTypes.TOOL.EXECUTION_COMPLETED,

    // ===== Run Events =====
    isRunStarted: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.RUN.STARTED]> =>
        event.type === EventTypes.RUN.STARTED,

    isRunCompleted: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.RUN.COMPLETED]> =>
        event.type === EventTypes.RUN.COMPLETED,

    isRunFailed: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.RUN.FAILED]> =>
        event.type === EventTypes.RUN.FAILED,

    isRunDecisionRequested: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.RUN.DECISION_REQUESTED]> =>
        event.type === EventTypes.RUN.DECISION_REQUESTED,

    isStepStarted: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.RUN.STEP_STARTED]> =>
        event.type === EventTypes.RUN.STEP_STARTED,

    isStepCompleted: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.RUN.STEP_COMPLETED]> =>
        event.type === EventTypes.RUN.STEP_COMPLETED,

    isStepFailed: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.RUN.STEP_FAILED]> =>
        event.type === EventTypes.RUN.STEP_FAILED,

    isRunExecutionRequested: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.RUN.EXECUTION_REQUESTED]> =>
        event.type === EventTypes.RUN.EXECUTION_REQUESTED,

    isRunExecutionCompleted: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.RUN.EXECUTION_COMPLETED]> =>
        event.type === EventTypes.RUN.EXECUTION_COMPLETED,

    // ===== Swarm Events =====
    isSwarmStarted: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.SWARM.STARTED]> =>
        event.type === EventTypes.SWARM.STARTED,

    isSwarmStateChanged: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.SWARM.STATE_CHANGED]> =>
        event.type === EventTypes.SWARM.STATE_CHANGED,

    isSwarmResourceUpdated: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.SWARM.RESOURCE_UPDATED]> =>
        event.type === EventTypes.SWARM.RESOURCE_UPDATED,

    isSwarmConfigUpdated: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.SWARM.CONFIG_UPDATED]> =>
        event.type === EventTypes.SWARM.CONFIG_UPDATED,

    isSwarmTeamUpdated: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.SWARM.TEAM_UPDATED]> =>
        event.type === EventTypes.SWARM.TEAM_UPDATED,

    isSwarmGoalCreated: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.SWARM.GOAL_CREATED]> =>
        event.type === EventTypes.SWARM.GOAL_CREATED,

    isSwarmGoalUpdated: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.SWARM.GOAL_UPDATED]> =>
        event.type === EventTypes.SWARM.GOAL_UPDATED,

    isSwarmGoalCompleted: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.SWARM.GOAL_COMPLETED]> =>
        event.type === EventTypes.SWARM.GOAL_COMPLETED,

    isSwarmGoalFailed: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.SWARM.GOAL_FAILED]> =>
        event.type === EventTypes.SWARM.GOAL_FAILED,

    // ===== User Events =====
    isUserCreditsUpdated: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.USER.CREDITS_UPDATED]> =>
        event.type === EventTypes.USER.CREDITS_UPDATED,

    isUserNotificationReceived: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.USER.NOTIFICATION_RECEIVED]> =>
        event.type === EventTypes.USER.NOTIFICATION_RECEIVED,

    // ===== Data Events =====
    isDataAccessRequested: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.DATA.ACCESS_REQUESTED]> =>
        event.type === EventTypes.DATA.ACCESS_REQUESTED,

    isDataAccessCompleted: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.DATA.ACCESS_COMPLETED]> =>
        event.type === EventTypes.DATA.ACCESS_COMPLETED,

    isDataAccessDenied: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.DATA.ACCESS_DENIED]> =>
        event.type === EventTypes.DATA.ACCESS_DENIED,

    isDataModificationRequested: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.DATA.MODIFICATION_REQUESTED]> =>
        event.type === EventTypes.DATA.MODIFICATION_REQUESTED,

    isDataModificationCompleted: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.DATA.MODIFICATION_COMPLETED]> =>
        event.type === EventTypes.DATA.MODIFICATION_COMPLETED,

    // ===== Security Events =====
    isSecurityThreatDetected: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.SECURITY.THREAT_DETECTED]> =>
        event.type === EventTypes.SECURITY.THREAT_DETECTED,

    isSecurityEmergencyStop: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.SECURITY.EMERGENCY_STOP]> =>
        event.type === EventTypes.SECURITY.EMERGENCY_STOP,

    isSecurityPermissionCheck: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.SECURITY.PERMISSION_CHECK]> =>
        event.type === EventTypes.SECURITY.PERMISSION_CHECK,

    isSecurityAuditLogged: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.SECURITY.AUDIT_LOGGED]> =>
        event.type === EventTypes.SECURITY.AUDIT_LOGGED,

    isSecurityAccessBlocked: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.SECURITY.ACCESS_BLOCKED]> =>
        event.type === EventTypes.SECURITY.ACCESS_BLOCKED,

    isSecurityPolicyViolated: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.SECURITY.POLICY_VIOLATED]> =>
        event.type === EventTypes.SECURITY.POLICY_VIOLATED,

    // ===== System Events =====
    isSystemError: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.SYSTEM.ERROR]> =>
        event.type === EventTypes.SYSTEM.ERROR,

    isSystemStateChanged: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.SYSTEM.STATE_CHANGED]> =>
        event.type === EventTypes.SYSTEM.STATE_CHANGED,

    // ===== Room Events =====
    isRoomJoinRequested: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.ROOM.JOIN_REQUESTED]> =>
        event.type === EventTypes.ROOM.JOIN_REQUESTED,

    isRoomLeaveRequested: (event: ServiceEvent): event is ServiceEvent<EventTypeDataMap[typeof EventTypes.ROOM.LEAVE_REQUESTED]> =>
        event.type === EventTypes.ROOM.LEAVE_REQUESTED,
};

/**
 * Utility function to check if an event matches any of the given patterns
 * @param event - The service event to check
 * @param patterns - Array of event type patterns to match against
 * @returns True if the event matches any of the patterns
 */
export function matchesAnyPattern(event: ServiceEvent, patterns: string[]): boolean {
    return patterns.some(pattern => {
        if (pattern.includes("*")) {
            // Simple wildcard matching - replace * with .*
            const regex = new RegExp("^" + pattern.replace(/\*/g, ".*") + "$");
            return regex.test(event.type);
        }
        return event.type === pattern;
    });
}

/**
 * Helper function to extract chat ID from chat-related events
 * @param event - A chat event
 * @returns The chat ID if the event has one, undefined otherwise
 */
export function extractChatId(event: ServiceEvent): string | undefined {
    if (event.data && typeof event.data === "object" && "chatId" in event.data) {
        return event.data.chatId as string;
    }
    return undefined;
}

/**
 * Helper function to extract run ID from run-related events
 * @param event - A run event
 * @returns The run ID if the event has one, undefined otherwise
 */
export function extractRunId(event: ServiceEvent): string | undefined {
    if (event.data && typeof event.data === "object" && "runId" in event.data) {
        return event.data.runId as string;
    }
    return undefined;
}
