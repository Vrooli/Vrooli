/**
 * Unified ID Types for Execution System
 * 
 * These nominal types provide compile-time safety while maintaining zero runtime cost.
 * They ensure that SwarmIds, BotIds, and TurnIds cannot be accidentally mixed while
 * allowing seamless conversion from strings when needed.
 * 
 * Design Principles:
 * - Zero runtime overhead (just type annotations)
 * - Compile-time type safety
 * - Easy conversion from existing string IDs
 * - Compatible with existing database schemas
 */

/**
 * Unique identifier for a swarm execution
 * Used across all tiers to identify swarm instances
 */
export type SwarmId = string & { readonly __brand: "SwarmId" };

/**
 * Unique identifier for a bot participant
 * Used to identify individual bots within conversations and swarms
 */
export type BotId = string & { readonly __brand: "BotId" };

/**
 * Unique identifier for a conversation turn
 * Used to track individual turns within conversations
 */
export type TurnId = string & { readonly __brand: "TurnId" };

/**
 * Zero-cost conversion functions for type safety
 * These functions provide type-safe conversion from strings to nominal types
 * with no runtime overhead.
 */

/**
 * Convert a string to SwarmId with type safety
 * @param id - String identifier to convert
 * @returns SwarmId with compile-time type checking
 */
export function toSwarmId(id: string): SwarmId {
    return id as SwarmId;
}

/**
 * Convert a string to BotId with type safety
 * @param id - String identifier to convert
 * @returns BotId with compile-time type checking
 */
export function toBotId(id: string): BotId {
    return id as BotId;
}

/**
 * Convert a string to TurnId with type safety
 * @param id - String identifier to convert
 * @returns TurnId with compile-time type checking
 */
export function toTurnId(id: string): TurnId {
    return id as TurnId;
}

/**
 * Type guards for runtime validation (optional use)
 * These can be used when you need runtime validation of ID format
 */

/**
 * Check if a string looks like a valid swarm ID
 * @param id - String to validate
 * @returns True if the string appears to be a valid swarm ID
 */
export function isValidSwarmId(id: string): boolean {
    return typeof id === "string" && id.length > 0 && !id.includes(" ");
}

/**
 * Check if a string looks like a valid bot ID
 * @param id - String to validate
 * @returns True if the string appears to be a valid bot ID
 */
export function isValidBotId(id: string): boolean {
    return typeof id === "string" && id.length > 0 && !id.includes(" ");
}

/**
 * Check if a string looks like a valid turn ID
 * @param id - String to validate
 * @returns True if the string appears to be a valid turn ID
 */
export function isValidTurnId(id: string): boolean {
    return typeof id === "string" && id.length > 0 && !id.includes(" ");
}

/**
 * Utility functions for ID generation and manipulation
 */

/**
 * Extract the underlying string value from a nominal ID type
 * Useful when interfacing with external systems that expect strings
 */
export function toStringId(id: SwarmId | BotId | TurnId): string {
    return id as string;
}

/**
 * Generate a new turn ID for a swarm
 * @param swarmId - The swarm this turn belongs to
 * @param timestamp - Optional timestamp (defaults to now)
 * @returns New TurnId
 */
export function generateTurnId(swarmId: SwarmId, timestamp?: Date): TurnId {
    const ts = timestamp?.getTime() || Date.now();
    return toTurnId(`turn_${swarmId}_${ts}`);
}
