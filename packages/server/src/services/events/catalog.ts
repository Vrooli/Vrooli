/**
 * Event Catalog Registry
 * 
 * Central registry for event schemas and agent subscriptions.
 * Enables agent discovery of available events and validation.
 * 
 * This component allows agents to:
 * - Discover available event types and their schemas
 * - Propose new emergent event types without code changes
 * - Validate events against registered schemas
 * - Track event type evolution and confidence
 */

import { type Logger } from "winston";
import {
  type BaseEvent,
  type EventSchema,
  type EventTypeInfo,
  type ValidationResult,
} from "./types.js";

// Constants to avoid magic numbers
const MAX_TOP_EVENTS = 10;
const MIN_JUSTIFICATION_LENGTH = 50;

/**
 * Central registry for event schemas and agent subscriptions.
 * Enables agent discovery of available events and validation.
 */
export interface IEventCatalog {
  /**
   * Register a core event schema (infrastructure events)
   */
  registerCoreSchema<T extends BaseEvent>(
    eventType: string,
    schema: EventSchema<T>
  ): Promise<void>;
  
  /**
   * Allow agents to propose new emergent event types
   */
  proposeEmergentEvent<T extends BaseEvent>(
    proposingAgentId: string,
    eventType: string,
    schema: EventSchema<T>,
    justification: string
  ): Promise<{ accepted: boolean; reason: string }>;
  
  /**
   * Validate an event against its registered schema
   */
  validateEvent<T extends BaseEvent>(event: T): Promise<ValidationResult>;
  
  /**
   * Get available event types for agent discovery
   */
  getAvailableEvents(): Promise<{
    coreEvents: EventTypeInfo[];
    emergentEvents: EventTypeInfo[];
    agentProposed: EventTypeInfo[];
  }>;
}

/**
 * Emergent event metadata for tracking evolution
 */
interface EmergentEventMetadata {
  proposedBy: string;
  proposedAt: Date;
  confidence: number;
  usageCount: number;
  lastUsed?: Date;
  subscribers: string[];
  feedback: Array<{
    agentId: string;
    rating: number;
    comment?: string;
    timestamp: Date;
  }>;
}

/**
 * Event catalog implementation
 */
export class EventCatalog implements IEventCatalog {
  private readonly coreSchemas = new Map<string, EventSchema>();
  private readonly emergentSchemas = new Map<string, EventSchema>();
  private readonly emergentMetadata = new Map<string, EmergentEventMetadata>();
  private readonly agentProposals = new Map<string, {
    schema: EventSchema;
    proposingAgentId: string;
    justification: string;
    proposedAt: Date;
    status: "pending" | "accepted" | "rejected";
  }>();

  constructor(private readonly logger: Logger) {
    this.registerCoreSchemas();
  }

  /**
   * Register a core event schema (infrastructure events)
   */
  async registerCoreSchema<T extends BaseEvent>(
    eventType: string,
    schema: EventSchema<T>,
  ): Promise<void> {
    if (this.coreSchemas.has(eventType)) {
      throw new Error(`Core event schema '${eventType}' is already registered`);
    }

    this.coreSchemas.set(eventType, schema);
    
    this.logger.info("[EventCatalog] Registered core event schema", {
      eventType,
      description: schema.description,
    });
  }

  /**
   * Allow agents to propose new emergent event types
   */
  async proposeEmergentEvent<T extends BaseEvent>(
    proposingAgentId: string,
    eventType: string,
    schema: EventSchema<T>,
    justification: string,
  ): Promise<{ accepted: boolean; reason: string }> {
    // Check if event type already exists
    if (this.coreSchemas.has(eventType) || this.emergentSchemas.has(eventType)) {
      return {
        accepted: false,
        reason: `Event type '${eventType}' already exists`,
      };
    }

    // Validate the proposed schema
    const validation = this.validateProposedSchema(schema);
    if (!validation.valid) {
      return {
        accepted: false,
        reason: `Invalid schema: ${validation.errors?.join(", ")}`,
      };
    }

    const proposalId = `${eventType}_${Date.now()}`;
    
    // Store the proposal
    this.agentProposals.set(proposalId, {
      schema,
      proposingAgentId,
      justification,
      proposedAt: new Date(),
      status: "pending",
    });

    // Auto-accept if schema is well-formed and doesn't conflict
    if (this.shouldAutoAcceptProposal(schema, justification)) {
      await this.acceptProposal(proposalId);
      
      this.logger.info("[EventCatalog] Auto-accepted emergent event proposal", {
        eventType,
        proposingAgentId,
        justification,
      });
      
      return {
        accepted: true,
        reason: "Schema is well-formed and non-conflicting",
      };
    }

    this.logger.info("[EventCatalog] Received emergent event proposal", {
      eventType,
      proposingAgentId,
      justification,
      proposalId,
    });

    return {
      accepted: false,
      reason: "Proposal requires review before acceptance",
    };
  }

  /**
   * Validate an event against its registered schema
   */
  async validateEvent<T extends BaseEvent>(event: T): Promise<ValidationResult> {
    // Try core schemas first
    const coreSchema = this.coreSchemas.get(event.type);
    if (coreSchema) {
      return this.validateAgainstSchema(event, coreSchema);
    }

    // Try emergent schemas
    const emergentSchema = this.emergentSchemas.get(event.type);
    if (emergentSchema) {
      // Update usage tracking
      const metadata = this.emergentMetadata.get(event.type);
      if (metadata) {
        metadata.usageCount++;
        metadata.lastUsed = new Date();
      }
      
      return this.validateAgainstSchema(event, emergentSchema);
    }

    // No schema found - create basic validation
    const hasRequiredFields = (
      typeof event.id === "string" &&
      typeof event.type === "string" &&
      event.timestamp instanceof Date &&
      typeof event.source === "object" &&
      event.source !== null
    );

    if (!hasRequiredFields) {
      return {
        valid: false,
        errors: ["Event missing required fields (id, type, timestamp, source)"],
      };
    }

    return {
      valid: true,
      errors: [`No specific schema found for '${event.type}', validated against base event structure only`],
    };
  }

  /**
   * Get available event types for agent discovery
   */
  async getAvailableEvents(): Promise<{
    coreEvents: EventTypeInfo[];
    emergentEvents: EventTypeInfo[];
    agentProposed: EventTypeInfo[];
  }> {
    const coreEvents: EventTypeInfo[] = Array.from(this.coreSchemas.entries())
      .map(([type, schema]) => ({
        type,
        description: schema.description,
        deliveryGuarantee: this.inferDeliveryGuarantee(type),
        schema: schema.schema,
        examples: schema.examples,
      }));

    const emergentEvents: EventTypeInfo[] = Array.from(this.emergentSchemas.entries())
      .map(([type, schema]) => ({
        type,
        description: schema.description,
        deliveryGuarantee: this.inferDeliveryGuarantee(type),
        schema: schema.schema,
        examples: schema.examples,
      }));

    const agentProposed: EventTypeInfo[] = Array.from(this.agentProposals.values())
      .filter(proposal => proposal.status === "pending")
      .map(proposal => ({
        type: proposal.schema.eventType,
        description: proposal.schema.description,
        deliveryGuarantee: this.inferDeliveryGuarantee(proposal.schema.eventType),
        schema: proposal.schema.schema,
        examples: proposal.schema.examples,
      }));

    return { coreEvents, emergentEvents, agentProposed };
  }

  /**
   * Accept a pending proposal
   */
  async acceptProposal(proposalId: string): Promise<void> {
    const proposal = this.agentProposals.get(proposalId);
    if (!proposal) {
      throw new Error(`Proposal '${proposalId}' not found`);
    }

    if (proposal.status !== "pending") {
      throw new Error(`Proposal '${proposalId}' is not pending`);
    }

    // Move to emergent schemas
    this.emergentSchemas.set(proposal.schema.eventType, proposal.schema);
    
    // Create metadata
    this.emergentMetadata.set(proposal.schema.eventType, {
      proposedBy: proposal.proposingAgentId,
      proposedAt: proposal.proposedAt,
      confidence: 0.5, // Start with neutral confidence
      usageCount: 0,
      subscribers: [],
      feedback: [],
    });

    // Update proposal status
    proposal.status = "accepted";

    this.logger.info("[EventCatalog] Accepted emergent event proposal", {
      eventType: proposal.schema.eventType,
      proposingAgentId: proposal.proposingAgentId,
      proposalId,
    });
  }

  /**
   * Provide feedback on an emergent event type
   */
  async provideFeedback(
    eventType: string,
    agentId: string,
    rating: number,
    comment?: string,
  ): Promise<void> {
    const metadata = this.emergentMetadata.get(eventType);
    if (!metadata) {
      throw new Error(`Emergent event type '${eventType}' not found`);
    }

    metadata.feedback.push({
      agentId,
      rating: Math.max(0, Math.min(1, rating)), // Clamp between 0 and 1
      comment,
      timestamp: new Date(),
    });

    // Update confidence based on feedback
    const avgRating = metadata.feedback.reduce((sum, f) => sum + f.rating, 0) / metadata.feedback.length;
    metadata.confidence = avgRating;

    this.logger.debug("[EventCatalog] Received feedback for emergent event", {
      eventType,
      agentId,
      rating,
      newConfidence: metadata.confidence,
    });
  }

  /**
   * Get statistics about event usage and confidence
   */
  getEventStatistics(): {
    coreEventCount: number;
    emergentEventCount: number;
    pendingProposals: number;
    topUsedEmergentEvents: Array<{
      eventType: string;
      usageCount: number;
      confidence: number;
    }>;
  } {
    const topUsedEmergentEvents = Array.from(this.emergentMetadata.entries())
      .map(([eventType, metadata]) => ({
        eventType,
        usageCount: metadata.usageCount,
        confidence: metadata.confidence,
      }))
      .sort((a, b) => b.usageCount - a.usageCount)
      .slice(0, MAX_TOP_EVENTS);

    const pendingProposals = Array.from(this.agentProposals.values())
      .filter(p => p.status === "pending").length;

    return {
      coreEventCount: this.coreSchemas.size,
      emergentEventCount: this.emergentSchemas.size,
      pendingProposals,
      topUsedEmergentEvents,
    };
  }

  /**
   * Private helper methods
   */

  private validateProposedSchema(schema: EventSchema): ValidationResult {
    const errors: string[] = [];

    if (!schema.eventType || typeof schema.eventType !== "string") {
      errors.push("Schema must have a valid eventType");
    }

    if (!schema.description || typeof schema.description !== "string") {
      errors.push("Schema must have a description");
    }

    if (!schema.schema || typeof schema.schema !== "object") {
      errors.push("Schema must have a schema definition");
    }

    if (!Array.isArray(schema.examples)) {
      errors.push("Schema must have examples array");
    }

    return {
      valid: errors.length === 0,
      errors: errors.length > 0 ? errors : undefined,
    };
  }

  private shouldAutoAcceptProposal(schema: EventSchema, justification: string): boolean {
    // Auto-accept if:
    // 1. Schema is well-formed
    // 2. Event type follows conventional naming patterns
    // 3. Justification is substantial
    
    const hasGoodNaming = (
      schema.eventType.includes("/") && // Hierarchical
      schema.eventType.split("/").length >= 2 && // At least 2 levels
      !schema.eventType.includes(" ") // No spaces
    );

    const hasSubstantialJustification = justification.length >= MIN_JUSTIFICATION_LENGTH;

    return hasGoodNaming && hasSubstantialJustification;
  }

  private validateAgainstSchema(event: BaseEvent, schema: EventSchema): ValidationResult {
    try {
      // Basic validation - in a full implementation, this would use a proper JSON schema validator
      const hasRequiredFields = (
        typeof event.id === "string" &&
        typeof event.type === "string" &&
        event.timestamp instanceof Date &&
        typeof event.source === "object" &&
        event.source !== null &&
        event.data !== undefined
      );

      if (!hasRequiredFields) {
        return {
          valid: false,
          errors: ["Event missing required base fields"],
        };
      }

      // Event type must match schema
      if (event.type !== schema.eventType) {
        return {
          valid: false,
          errors: [`Event type '${event.type}' does not match schema '${schema.eventType}'`],
        };
      }

      return { valid: true };

    } catch (error) {
      return {
        valid: false,
        errors: [error instanceof Error ? error.message : String(error)],
      };
    }
  }

  private inferDeliveryGuarantee(eventType: string): "fire-and-forget" | "reliable" | "barrier-sync" {
    // Infer delivery guarantee from event type patterns
    if (eventType.startsWith("safety/") || eventType.startsWith("emergency/")) {
      return "barrier-sync";
    }
    
    if (eventType.includes("/completed") || eventType.includes("/failed") || eventType.includes("/error")) {
      return "reliable";
    }
    
    return "fire-and-forget";
  }

  /**
   * Register core infrastructure event schemas
   */
  private registerCoreSchemas(): void {
    // Goal management events
    this.coreSchemas.set("swarm/goal/created", {
      eventType: "swarm/goal/created",
      description: "Triggered when a new swarm goal is created",
      schema: {
        type: "object",
        properties: {
          swarmId: { type: "string" },
          goalDescription: { type: "string" },
          priority: { enum: ["low", "medium", "high", "critical"] },
          estimatedCredits: { type: "string" },
          deadline: { type: "string", format: "date-time" },
        },
        required: ["swarmId", "goalDescription", "priority"],
      },
      examples: [{
        swarmId: "swarm_123",
        goalDescription: "Generate market analysis report",
        priority: "high",
        estimatedCredits: "50000",
      }],
    });

    // Tool execution events
    this.coreSchemas.set("tool/approval_required", {
      eventType: "tool/approval_required",
      description: "Triggered when a tool requires user approval before execution",
      schema: {
        type: "object",
        properties: {
          pendingId: { type: "string" },
          toolCallId: { type: "string" },
          toolName: { type: "string" },
          parameters: { type: "object" },
          callerBotId: { type: "string" },
          approvalTimeoutAt: { type: "number" },
          estimatedCost: { type: "string" },
        },
        required: ["pendingId", "toolCallId", "toolName", "callerBotId"],
      },
      examples: [{
        pendingId: "pending_123",
        toolCallId: "call_456",
        toolName: "web_search",
        parameters: { query: "market analysis", maxResults: 10 },
        callerBotId: "bot_789",
        estimatedCost: "150",
      }],
    });

    // Error events
    this.coreSchemas.set("execution/error/occurred", {
      eventType: "execution/error/occurred",
      description: "Triggered when an execution error occurs",
      schema: {
        type: "object",
        properties: {
          errorType: { type: "string" },
          message: { type: "string" },
          stack: { type: "string" },
          context: {
            type: "object",
            properties: {
              tier: { type: "number" },
              component: { type: "string" },
              operation: { type: "string" },
            },
          },
        },
        required: ["errorType", "message", "context"],
      },
      examples: [{
        errorType: "Error",
        message: "Connection timeout",
        stack: "Error: Connection timeout\n  at ...",
        context: {
          tier: 3,
          component: "LLMClient",
          operation: "chat_completion",
        },
      }],
    });

    // Routine lifecycle events
    this.coreSchemas.set("routine/completed", {
      eventType: "routine/completed",
      description: "Triggered when a routine completes successfully",
      schema: {
        type: "object",
        properties: {
          routineId: { type: "string" },
          runId: { type: "string" },
          totalDuration: { type: "number" },
          creditsUsed: { type: "string" },
          outputs: { type: "object" },
        },
        required: ["routineId", "runId", "totalDuration"],
      },
      examples: [{
        routineId: "routine_456",
        runId: "run_789",
        totalDuration: 120000,
        creditsUsed: "2500",
        outputs: { reportUrl: "https://..." },
      }],
    });

    this.logger.info("[EventCatalog] Registered core event schemas", {
      count: this.coreSchemas.size,
    });
  }
}

/**
 * Create an event catalog instance
 */
export function createEventCatalog(logger: Logger): EventCatalog {
  return new EventCatalog(logger);
}
