/**
 * Context Transformation Utilities
 * 
 * Handles conversion between basic execution context (simple variables)
 * and enhanced execution context (rich BPMN state management).
 */

import type { 
    EnhancedExecutionContext, 
    ContextTransformer,
    BoundaryEvent,
    IntermediateEvent,
    EventInstance,
    TimerEvent,
    ParallelBranch,
    JoinPoint,
    SubprocessContext,
    EventSubprocess,
    MessageEvent,
    WebhookEvent,
    SignalEvent,
    InclusiveGatewayState,
    ComplexGatewayState,
} from "../types.js";

/**
 * Default context transformer implementation
 */
export class DefaultContextTransformer implements ContextTransformer {
    
    /**
     * Convert basic context to enhanced context
     */
    enhance(basicContext: Record<string, unknown>): EnhancedExecutionContext {
        // Check if already enhanced
        if (this.isEnhancedContext(basicContext)) {
            return basicContext as unknown as EnhancedExecutionContext;
        }

        return {
            variables: { ...basicContext },
            events: {
                active: [],
                pending: [],
                fired: [],
                timers: [],
            },
            parallelExecution: {
                activeBranches: [],
                completedBranches: [],
                joinPoints: [],
            },
            subprocesses: {
                stack: [],
                eventSubprocesses: [],
            },
            external: {
                messageEvents: [],
                webhookEvents: [],
                signalEvents: [],
            },
            gateways: {
                inclusiveStates: [],
                complexConditions: [],
            },
        };
    }

    /**
     * Extract basic context from enhanced context
     */
    simplify(enhancedContext: EnhancedExecutionContext): Record<string, unknown> {
        return { ...enhancedContext.variables };
    }

    /**
     * Merge contexts while preserving enhanced state
     */
    merge(base: EnhancedExecutionContext, updates: Partial<EnhancedExecutionContext>): EnhancedExecutionContext {
        return {
            variables: { ...base.variables, ...updates.variables },
            events: {
                active: updates.events?.active ?? base.events.active,
                pending: updates.events?.pending ?? base.events.pending,
                fired: [...base.events.fired, ...(updates.events?.fired ?? [])],
                timers: updates.events?.timers ?? base.events.timers,
            },
            parallelExecution: {
                activeBranches: updates.parallelExecution?.activeBranches ?? base.parallelExecution.activeBranches,
                completedBranches: [
                    ...base.parallelExecution.completedBranches, 
                    ...(updates.parallelExecution?.completedBranches ?? []),
                ],
                joinPoints: updates.parallelExecution?.joinPoints ?? base.parallelExecution.joinPoints,
            },
            subprocesses: {
                stack: updates.subprocesses?.stack ?? base.subprocesses.stack,
                eventSubprocesses: updates.subprocesses?.eventSubprocesses ?? base.subprocesses.eventSubprocesses,
            },
            external: {
                messageEvents: updates.external?.messageEvents ?? base.external.messageEvents,
                webhookEvents: [...base.external.webhookEvents, ...(updates.external?.webhookEvents ?? [])],
                signalEvents: [...base.external.signalEvents, ...(updates.external?.signalEvents ?? [])],
            },
            gateways: {
                inclusiveStates: updates.gateways?.inclusiveStates ?? base.gateways.inclusiveStates,
                complexConditions: updates.gateways?.complexConditions ?? base.gateways.complexConditions,
            },
        };
    }

    /**
     * Validate context structure and data
     */
    validate(context: EnhancedExecutionContext): boolean {
        try {
            // Basic structure validation
            if (!context || typeof context !== "object") return false;
            if (!context.variables || typeof context.variables !== "object") return false;
            if (!context.events || typeof context.events !== "object") return false;
            if (!context.parallelExecution || typeof context.parallelExecution !== "object") return false;
            if (!context.subprocesses || typeof context.subprocesses !== "object") return false;
            if (!context.external || typeof context.external !== "object") return false;
            if (!context.gateways || typeof context.gateways !== "object") return false;

            // Array validations
            if (!Array.isArray(context.events.active)) return false;
            if (!Array.isArray(context.events.pending)) return false;
            if (!Array.isArray(context.events.fired)) return false;
            if (!Array.isArray(context.events.timers)) return false;
            if (!Array.isArray(context.parallelExecution.activeBranches)) return false;
            if (!Array.isArray(context.parallelExecution.completedBranches)) return false;
            if (!Array.isArray(context.parallelExecution.joinPoints)) return false;
            if (!Array.isArray(context.subprocesses.stack)) return false;
            if (!Array.isArray(context.subprocesses.eventSubprocesses)) return false;
            if (!Array.isArray(context.external.messageEvents)) return false;
            if (!Array.isArray(context.external.webhookEvents)) return false;
            if (!Array.isArray(context.external.signalEvents)) return false;
            if (!Array.isArray(context.gateways.inclusiveStates)) return false;
            if (!Array.isArray(context.gateways.complexConditions)) return false;

            // Validate individual elements
            for (const event of context.events.active) {
                if (!this.validateBoundaryEvent(event)) return false;
            }

            for (const event of context.events.pending) {
                if (!this.validateIntermediateEvent(event)) return false;
            }

            for (const timer of context.events.timers) {
                if (!this.validateTimerEvent(timer)) return false;
            }

            for (const branch of context.parallelExecution.activeBranches) {
                if (!this.validateParallelBranch(branch)) return false;
            }

            for (const joinPoint of context.parallelExecution.joinPoints) {
                if (!this.validateJoinPoint(joinPoint)) return false;
            }

            return true;
        } catch {
            return false;
        }
    }

    /**
     * Prune completed/expired state from context
     */
    prune(context: EnhancedExecutionContext): EnhancedExecutionContext {
        const now = new Date();
        const maxAge = 60 * 60 * 1000; // 1 hour

        return {
            ...context,
            events: {
                ...context.events,
                // Remove expired timers
                timers: context.events.timers.filter(timer => timer.expiresAt > now),
                // Keep only recent fired events
                fired: context.events.fired.filter(event => 
                    (now.getTime() - event.firedAt.getTime()) < maxAge,
                ),
            },
            parallelExecution: {
                ...context.parallelExecution,
                // Remove completed branches that are no longer needed
                activeBranches: context.parallelExecution.activeBranches.filter(
                    branch => branch.status !== "completed" && branch.status !== "failed",
                ),
            },
            subprocesses: {
                ...context.subprocesses,
                // Remove completed event subprocesses
                eventSubprocesses: context.subprocesses.eventSubprocesses.filter(
                    subprocess => subprocess.status !== "completed",
                ),
            },
            external: {
                ...context.external,
                // Keep only recent webhook events
                webhookEvents: context.external.webhookEvents.filter(event =>
                    (now.getTime() - event.receivedAt.getTime()) < maxAge,
                ),
                // Keep only recent signal events
                signalEvents: context.external.signalEvents.filter(event =>
                    (now.getTime() - event.propagatedAt.getTime()) < maxAge,
                ),
            },
        };
    }

    // Helper methods for validation

    private isEnhancedContext(context: Record<string, unknown>): boolean {
        return !!(
            context &&
            typeof context === "object" &&
            "events" in context &&
            "parallelExecution" in context &&
            "subprocesses" in context &&
            "external" in context &&
            "gateways" in context
        );
    }

    private validateBoundaryEvent(event: BoundaryEvent): boolean {
        return !!(
            event &&
            typeof event.id === "string" &&
            typeof event.type === "string" &&
            typeof event.attachedToRef === "string" &&
            typeof event.interrupting === "boolean" &&
            event.activatedAt instanceof Date
        );
    }

    private validateIntermediateEvent(event: IntermediateEvent): boolean {
        return !!(
            event &&
            typeof event.id === "string" &&
            typeof event.type === "string" &&
            typeof event.waiting === "boolean" &&
            event.eventDefinition &&
            typeof event.eventDefinition === "object"
        );
    }

    private validateTimerEvent(timer: TimerEvent): boolean {
        return !!(
            timer &&
            typeof timer.id === "string" &&
            typeof timer.eventId === "string" &&
            timer.expiresAt instanceof Date
        );
    }

    private validateParallelBranch(branch: ParallelBranch): boolean {
        return !!(
            branch &&
            typeof branch.id === "string" &&
            typeof branch.branchId === "string" &&
            branch.currentLocation &&
            typeof branch.currentLocation.nodeId === "string" &&
            typeof branch.status === "string" &&
            branch.startedAt instanceof Date
        );
    }

    private validateJoinPoint(joinPoint: JoinPoint): boolean {
        return !!(
            joinPoint &&
            typeof joinPoint.id === "string" &&
            typeof joinPoint.gatewayId === "string" &&
            Array.isArray(joinPoint.requiredBranches) &&
            Array.isArray(joinPoint.completedBranches) &&
            typeof joinPoint.isReady === "boolean"
        );
    }
}

/**
 * Utility functions for context management
 */
export class ContextUtils {
    /**
     * Create empty enhanced context
     */
    static createEmpty(): EnhancedExecutionContext {
        return new DefaultContextTransformer().enhance({});
    }

    /**
     * Add variable to context
     */
    static addVariable(
        context: EnhancedExecutionContext, 
        key: string, 
        value: unknown,
    ): EnhancedExecutionContext {
        return {
            ...context,
            variables: {
                ...context.variables,
                [key]: value,
            },
        };
    }

    /**
     * Add boundary event to context
     */
    static addBoundaryEvent(
        context: EnhancedExecutionContext,
        event: BoundaryEvent,
    ): EnhancedExecutionContext {
        return {
            ...context,
            events: {
                ...context.events,
                active: [...context.events.active, event],
            },
        };
    }

    /**
     * Add timer event to context
     */
    static addTimerEvent(
        context: EnhancedExecutionContext,
        timer: TimerEvent,
    ): EnhancedExecutionContext {
        return {
            ...context,
            events: {
                ...context.events,
                timers: [...context.events.timers, timer],
            },
        };
    }

    /**
     * Add parallel branch to context
     */
    static addParallelBranch(
        context: EnhancedExecutionContext,
        branch: ParallelBranch,
    ): EnhancedExecutionContext {
        return {
            ...context,
            parallelExecution: {
                ...context.parallelExecution,
                activeBranches: [...context.parallelExecution.activeBranches, branch],
            },
        };
    }

    /**
     * Complete parallel branch in context
     */
    static completeParallelBranch(
        context: EnhancedExecutionContext,
        branchId: string,
        result?: unknown,
    ): EnhancedExecutionContext {
        return {
            ...context,
            parallelExecution: {
                ...context.parallelExecution,
                activeBranches: context.parallelExecution.activeBranches.map(branch =>
                    branch.branchId === branchId
                        ? { ...branch, status: "completed" as const, completedAt: new Date(), result }
                        : branch,
                ),
                completedBranches: [...context.parallelExecution.completedBranches, branchId],
            },
        };
    }

    /**
     * Add subprocess context
     */
    static enterSubprocess(
        context: EnhancedExecutionContext,
        subprocessContext: SubprocessContext,
    ): EnhancedExecutionContext {
        return {
            ...context,
            subprocesses: {
                ...context.subprocesses,
                stack: [...context.subprocesses.stack, subprocessContext],
            },
        };
    }

    /**
     * Exit current subprocess
     */
    static exitSubprocess(context: EnhancedExecutionContext): EnhancedExecutionContext {
        return {
            ...context,
            subprocesses: {
                ...context.subprocesses,
                stack: context.subprocesses.stack.slice(0, -1),
            },
        };
    }

    /**
     * Fire an event and add to fired events
     */
    static fireEvent(
        context: EnhancedExecutionContext,
        eventInstance: EventInstance,
    ): EnhancedExecutionContext {
        return {
            ...context,
            events: {
                ...context.events,
                fired: [...context.events.fired, eventInstance],
            },
        };
    }
}

/**
 * Default context transformer instance
 */
export const contextTransformer = new DefaultContextTransformer();
