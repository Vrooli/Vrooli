/**
 * Domain types for Swarm Manager
 *
 * Barrel re-export — individual domain type files live alongside this file.
 * Import from here for convenience, or from specific files for precision.
 *
 * DOC: docs/concepts/ARCHITECTURE.md#domain-concepts
 * DOC: docs/internal/SEAMS.md#module-boundaries
 * DOC: docs/internal/INTENT.md#module-responsibilities
 */

export * from "./shared";
export * from "./backlog";
export * from "./initiative";
export * from "./capture";
export * from "./workshop";
export * from "./review";
export * from "./archive";
export * from "./scenario";
export * from "./agent";
export * from "./execution";
export * from "./settings";
export * from "./summary";
export * from "./prompt";
