/**
 * Emergent Security Infrastructure
 * 
 * This module provides NO hard-coded security logic.
 * ALL security decisions emerge from agent swarms responding to access events.
 * 
 * IMPORTANT: Security is fully emergent in Vrooli. Access attempts emit events,
 * and security agents make decisions based on configured policies, learned patterns,
 * and collaborative intelligence.
 */

// Legacy export for backward compatibility during migration
// TODO: Remove once all consumers have migrated to event-driven security
export { SecurityValidator } from "./securityValidator.js";

/**
 * Emergent Security Architecture:
 * 
 * 1. Access attempts emit events (no hard-coded permission checks)
 * 2. Security agents subscribe to access attempt events
 * 3. Agents evaluate requests based on:
 *    - Configured permission policies
 *    - Learned access patterns
 *    - Threat intelligence
 *    - Compliance requirements
 * 4. Agents respond with allow/deny decisions using barrier-sync events
 * 5. The system aggregates agent responses to make final decision
 * 
 * This fully emergent approach enables:
 * - Zero hard-coded security logic
 * - Adaptive security that learns and evolves
 * - Domain-specific security through specialized agents
 * - Collaborative security intelligence
 * - Security as a configurable, data-driven capability
 * 
 * Example Security Agents:
 * - Permission Policy Agent: Evaluates against stored permission rules
 * - Anomaly Detection Agent: Identifies unusual access patterns
 * - Threat Intelligence Agent: Checks against known threat indicators
 * - Compliance Agent: Ensures regulatory requirements are met
 * - Rate Limiting Agent: Prevents abuse through access frequency analysis
 */
