/**
 * AI Security Event Publisher
 * 
 * Minimal infrastructure for security events - intelligence emerges from security agents.
 * 
 * This component provides event publishing infrastructure for security monitoring.
 * Security detection logic, threat patterns, and validation rules emerge from 
 * specialized security agents that subscribe to security events.
 * 
 * Philosophy: Provide tools, not decisions. Security agents decide what constitutes
 * a threat based on their domain expertise and learning from security events.
 */

import type {
    ValidationResult,
    ValidationIssue,
} from "@vrooli/shared";
import { type Logger } from "winston";
import { EventPublisher } from "../../shared/EventPublisher.js";
import { type EventBus } from "../events/eventBus.js";

/**
 * Security event types that agents can subscribe to
 */
export interface SecurityEventData {
    /** Content being validated (input/output) */
    content: string;
    /** Type of content (input, output, prompt, etc.) */
    contentType: "input" | "output" | "prompt" | "response";
    /** Execution context */
    context: {
        runId?: string;
        stepId?: string;
        userId?: string;
        swarmId?: string;
    };
    /** Basic content metadata for agent analysis */
    metadata: {
        length: number;
        timestamp: Date;
        source: string;
    };
}

/**
 * Minimal security event publisher
 * 
 * Publishes security events for analysis by security agents.
 * Does NOT implement security logic - that emerges from agents.
 */
export class AISecurityEventPublisher {
    private readonly eventPublisher: EventPublisher;
    private readonly logger: Logger;

    constructor(eventBus: EventBus, logger: Logger) {
        this.eventPublisher = new EventPublisher(eventBus, logger, "AISecurityEventPublisher");
        this.logger = logger;
    }

    /**
     * Publishes AI interaction event for security agent analysis
     */
    async publishAIInteractionEvent(
        input: string,
        output: string,
        context: {
            requestId: string;
            tier: number;
            component: string;
            runId?: string;
            stepId?: string;
            userId?: string;
            swarmId?: string;
        }
    ): Promise<void> {
        const timestamp = new Date();

        // Publish input security event
        await this.eventPublisher.publish("security.ai.input", {
            content: input,
            contentType: "input" as const,
            context: {
                runId: context.runId,
                stepId: context.stepId,
                userId: context.userId,
                swarmId: context.swarmId,
            },
            metadata: {
                length: input.length,
                timestamp,
                source: context.component,
            },
        } as SecurityEventData);

        // Publish output security event
        await this.eventPublisher.publish("security.ai.output", {
            content: output,
            contentType: "output" as const,
            context: {
                runId: context.runId,
                stepId: context.stepId,
                userId: context.userId,
                swarmId: context.swarmId,
            },
            metadata: {
                length: output.length,
                timestamp,
                source: context.component,
            },
        } as SecurityEventData);

        // Publish interaction correlation event for agents to analyze patterns
        await this.eventPublisher.publish("security.ai.interaction", {
            requestId: context.requestId,
            tier: context.tier,
            component: context.component,
            inputLength: input.length,
            outputLength: output.length,
            timestamp,
            context: {
                runId: context.runId,
                stepId: context.stepId,
                userId: context.userId,
                swarmId: context.swarmId,
            },
        });
    }

    /**
     * Basic validation that just checks for completely empty content
     * Real validation logic emerges from security agents
     */
    async validateAIInteraction(
        input: string,
        output: string,
        context: {
            requestId: string;
            tier: number;
            component: string;
            runId?: string;
            stepId?: string;
            userId?: string;
            swarmId?: string;
        }
    ): Promise<ValidationResult> {
        // Publish events for security agents to analyze
        await this.publishAIInteractionEvent(input, output, context);

        // Only basic validation - agents provide the intelligence
        const issues: ValidationIssue[] = [];
        
        if (!input.trim() && !output.trim()) {
            issues.push({
                type: "empty_content",
                severity: "low",
                description: "Both input and output are empty",
                suggestion: "Provide meaningful content",
            });
        }

        return {
            isValid: issues.length === 0,
            issues,
            confidence: 1.0, // Security agents will determine actual confidence
            metadata: {
                timestamp: new Date(),
                validator: "AISecurityEventPublisher",
                eventsPublished: true,
            },
        };
    }

    /**
     * Creates a child event publisher with a specific prefix
     */
    createChild(sourcePrefix: string): AISecurityEventPublisher {
        const childPublisher = this.eventPublisher.createChild(sourcePrefix);
        return new AISecurityEventPublisher(
            childPublisher["eventBus"], // Access private field for child creation
            this.logger
        );
    }
}

// Legacy alias for backward compatibility during migration
export class AISecurityValidator extends AISecurityEventPublisher {
    constructor(eventBus: EventBus, logger: Logger) {
        super(eventBus, logger);
        this.logger.warn(
            "AISecurityValidator is deprecated. Use AISecurityEventPublisher directly. " +
            "Security logic now emerges from security agents, not hardcoded validation."
        );
    }
}