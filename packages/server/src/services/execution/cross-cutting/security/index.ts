/**
 * Minimal Security Infrastructure
 * 
 * This module provides ONLY the essential security infrastructure.
 * All complex security intelligence emerges from agent swarms.
 * 
 * IMPORTANT: Do NOT add threat detection, risk scoring, policy
 * enforcement, or incident response here. Create security agent
 * routines instead.
 */

export { MinimalSecurityValidator } from "./minimalSecurityValidator.js";
export type { SecurityCheckEvent } from "./minimalSecurityValidator.js";

// Legacy export for backward compatibility during migration
// TODO: Remove once all consumers have migrated to MinimalSecurityValidator
export { SecurityValidator } from "./securityValidator.js";

/**
 * Security Philosophy:
 * 
 * 1. Infrastructure provides only basic permission checks
 * 2. Infrastructure emits events about security checks
 * 3. Security agents subscribe to these events
 * 4. Agents provide all intelligence:
 *    - Threat detection
 *    - Anomaly detection
 *    - Risk scoring
 *    - Policy enforcement
 *    - Incident response
 *    - Compliance monitoring
 * 
 * This approach enables:
 * - Domain-specific security (teams deploy relevant agents)
 * - Learning security (agents improve over time)
 * - Customizable security (different agents for different needs)
 * - Emergent intelligence (complex behaviors from simple events)
 */