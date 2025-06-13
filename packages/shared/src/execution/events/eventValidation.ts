/**
 * Event validation utilities
 * Provides schema validation for events
 */

import { type BaseEvent, type EventMetadata, type EventSource } from "../types/events.js";
import { EventTypeValidator } from "./eventTypes.js";

// Time constants
const EVENT_STALE_THRESHOLD_MS = 60000; // 1 minute in milliseconds

// ID generation constants
const RANDOM_ID_LENGTH = 9; // Length of random string in generated IDs
const RANDOM_ID_RADIX = 36; // Base 36 for alphanumeric string generation

/**
 * Event validation result
 */
export interface EventValidationResult {
    valid: boolean;
    errors: EventValidationError[];
    warnings: EventValidationWarning[];
}

/**
 * Validation error
 */
export interface EventValidationError {
    field: string;
    message: string;
    code: string;
}

/**
 * Validation warning
 */
export interface EventValidationWarning {
    field: string;
    message: string;
    code: string;
}

/**
 * Event validator
 */
export class EventValidator {
    private typeValidator = new EventTypeValidator();
    
    /**
     * Validate a single event
     */
    validate(event: unknown): EventValidationResult {
        const errors: EventValidationError[] = [];
        const warnings: EventValidationWarning[] = [];
        
        // Check if event is an object
        if (!event || typeof event !== "object") {
            errors.push({
                field: "event",
                message: "Event must be an object",
                code: "INVALID_TYPE",
            });
            return { valid: false, errors, warnings };
        }
        
        const e = event as Partial<BaseEvent>;
        
        // Validate required fields
        this.validateRequiredFields(e, errors);
        
        // Validate field types
        this.validateFieldTypes(e, errors);
        
        // Validate event type
        if (e.type) {
            const typeErrors = this.typeValidator.validateEvent(e as any);
            typeErrors.forEach(error => {
                errors.push({
                    field: "type",
                    message: error,
                    code: "INVALID_EVENT_TYPE",
                });
            });
        }
        
        // Validate metadata
        if (e.metadata) {
            this.validateMetadata(e.metadata, errors, warnings);
        }
        
        // Validate source
        if (e.source) {
            this.validateSource(e.source, errors);
        }
        
        // Check for warnings
        this.checkWarnings(e, warnings);
        
        return {
            valid: errors.length === 0,
            errors,
            warnings,
        };
    }
    
    /**
     * Validate a batch of events
     */
    validateBatch(events: unknown[]): EventValidationResult[] {
        return events.map(event => this.validate(event));
    }
    
    private validateRequiredFields(
        event: Partial<BaseEvent>,
        errors: EventValidationError[],
    ): void {
        const requiredFields: Array<keyof BaseEvent> = [
            "id",
            "type",
            "timestamp",
            "source",
            "correlationId",
            "metadata",
        ];
        
        for (const field of requiredFields) {
            if (!event[field]) {
                errors.push({
                    field,
                    message: `${field} is required`,
                    code: "REQUIRED_FIELD",
                });
            }
        }
    }
    
    private validateFieldTypes(
        event: Partial<BaseEvent>,
        errors: EventValidationError[],
    ): void {
        // Validate ID
        if (event.id && typeof event.id !== "string") {
            errors.push({
                field: "id",
                message: "id must be a string",
                code: "INVALID_TYPE",
            });
        }
        
        // Validate type
        if (event.type && typeof event.type !== "string") {
            errors.push({
                field: "type",
                message: "type must be a string",
                code: "INVALID_TYPE",
            });
        }
        
        // Validate timestamp
        if (event.timestamp) {
            if (!(event.timestamp instanceof Date)) {
                // Try to parse as date
                const parsed = new Date(event.timestamp as any);
                if (isNaN(parsed.getTime())) {
                    errors.push({
                        field: "timestamp",
                        message: "timestamp must be a valid Date",
                        code: "INVALID_TYPE",
                    });
                }
            }
        }
        
        // Validate correlationId
        if (event.correlationId && typeof event.correlationId !== "string") {
            errors.push({
                field: "correlationId",
                message: "correlationId must be a string",
                code: "INVALID_TYPE",
            });
        }
        
        // Validate causationId
        if (event.causationId && typeof event.causationId !== "string") {
            errors.push({
                field: "causationId",
                message: "causationId must be a string",
                code: "INVALID_TYPE",
            });
        }
    }
    
    private validateMetadata(
        metadata: Partial<EventMetadata>,
        errors: EventValidationError[],
        warnings: EventValidationWarning[],
    ): void {
        // Validate version
        if (!metadata.version) {
            warnings.push({
                field: "metadata.version",
                message: "version is recommended",
                code: "MISSING_RECOMMENDED",
            });
        } else if (typeof metadata.version !== "string") {
            errors.push({
                field: "metadata.version",
                message: "version must be a string",
                code: "INVALID_TYPE",
            });
        }
        
        // Validate tags
        if (metadata.tags && !Array.isArray(metadata.tags)) {
            errors.push({
                field: "metadata.tags",
                message: "tags must be an array",
                code: "INVALID_TYPE",
            });
        }
        
        // Validate priority
        const validPriorities = ["LOW", "NORMAL", "HIGH", "CRITICAL"];
        if (metadata.priority && !validPriorities.includes(metadata.priority)) {
            errors.push({
                field: "metadata.priority",
                message: `priority must be one of: ${validPriorities.join(", ")}`,
                code: "INVALID_VALUE",
            });
        }
        
        // Validate TTL
        if (metadata.ttl !== undefined) {
            if (typeof metadata.ttl !== "number" || metadata.ttl <= 0) {
                errors.push({
                    field: "metadata.ttl",
                    message: "ttl must be a positive number",
                    code: "INVALID_VALUE",
                });
            }
        }
    }
    
    private validateSource(
        source: Partial<EventSource>,
        errors: EventValidationError[],
    ): void {
        // Validate tier
        const validTiers = [1, 2, 3, "cross-cutting"];
        if (!validTiers.includes(source.tier as any)) {
            errors.push({
                field: "source.tier",
                message: `tier must be one of: ${validTiers.join(", ")}`,
                code: "INVALID_VALUE",
            });
        }
        
        // Validate component
        if (!source.component) {
            errors.push({
                field: "source.component",
                message: "component is required",
                code: "REQUIRED_FIELD",
            });
        } else if (typeof source.component !== "string") {
            errors.push({
                field: "source.component",
                message: "component must be a string",
                code: "INVALID_TYPE",
            });
        }
        
        // Validate instanceId
        if (!source.instanceId) {
            errors.push({
                field: "source.instanceId",
                message: "instanceId is required",
                code: "REQUIRED_FIELD",
            });
        } else if (typeof source.instanceId !== "string") {
            errors.push({
                field: "source.instanceId",
                message: "instanceId must be a string",
                code: "INVALID_TYPE",
            });
        }
    }
    
    private checkWarnings(
        event: Partial<BaseEvent>,
        warnings: EventValidationWarning[],
    ): void {
        // Check for missing recommended fields
        if (!event.causationId) {
            warnings.push({
                field: "causationId",
                message: "causationId is recommended for tracing event chains",
                code: "MISSING_RECOMMENDED",
            });
        }
        
        // Check for missing user context
        if (event.metadata && !event.metadata.userId && !event.metadata.sessionId) {
            warnings.push({
                field: "metadata",
                message: "userId or sessionId is recommended for user context",
                code: "MISSING_CONTEXT",
            });
        }
        
        // Check timestamp freshness
        if (event.timestamp) {
            const age = Date.now() - new Date(event.timestamp).getTime();
            if (age > EVENT_STALE_THRESHOLD_MS) {
                warnings.push({
                    field: "timestamp",
                    message: "Event timestamp is more than 1 minute old",
                    code: "STALE_EVENT",
                });
            }
        }
    }
}

/**
 * Event sanitizer
 * Cleans and normalizes events
 */
export class EventSanitizer {
    /**
     * Sanitize an event
     */
    sanitize(event: Partial<BaseEvent>): BaseEvent {
        const sanitized = { ...event };
        
        // Ensure timestamp is a Date
        if (sanitized.timestamp && !(sanitized.timestamp instanceof Date)) {
            sanitized.timestamp = new Date(sanitized.timestamp);
        }
        
        // Ensure metadata exists
        if (!sanitized.metadata) {
            sanitized.metadata = {} as EventMetadata;
        }
        
        // Set default priority
        if (!sanitized.metadata.priority) {
            sanitized.metadata.priority = "NORMAL" as any;
        }
        
        // Ensure tags is an array
        if (!sanitized.metadata.tags) {
            sanitized.metadata.tags = [];
        }
        
        // Set default version
        if (!sanitized.metadata.version) {
            sanitized.metadata.version = "1.0.0";
        }
        
        // Generate ID if missing
        if (!sanitized.id) {
            sanitized.id = this.generateEventId();
        }
        
        // Generate correlationId if missing
        if (!sanitized.correlationId) {
            sanitized.correlationId = this.generateCorrelationId();
        }
        
        return sanitized as BaseEvent;
    }
    
    private generateEventId(): string {
        return `evt_${Date.now()}_${Math.random().toString(RANDOM_ID_RADIX).substring(2, RANDOM_ID_LENGTH)}`;
    }
    
    private generateCorrelationId(): string {
        return `cor_${Date.now()}_${Math.random().toString(RANDOM_ID_RADIX).substring(2, RANDOM_ID_LENGTH)}`;
    }
}
