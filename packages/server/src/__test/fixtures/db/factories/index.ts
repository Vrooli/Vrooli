/**
 * Database fixture factories for Core Business Objects Part 2
 * 
 * This module exports factory classes for creating test data for:
 * - Routine: Complex workflow definitions with JSON configurations
 * - RoutineVersion: Version management with complex graph structures
 * - Resource: External links, documentation, tutorials
 * - ResourceVersion: Versioned content
 * - ResourceVersionRelation: Junction table connecting resources to other versioned objects
 */

// Export factory classes
export { RoutineDbFactory, createRoutineDbFactory } from "./RoutineDbFactory.js";
export { RoutineVersionDbFactory, createRoutineVersionDbFactory } from "./RoutineVersionDbFactory.js";
export { ResourceDbFactory, createResourceDbFactory } from "./ResourceDbFactory.js";
export { ResourceVersionDbFactory, createResourceVersionDbFactory } from "./ResourceVersionDbFactory.js";
export { ResourceVersionRelationDbFactory, createResourceVersionRelationDbFactory } from "./ResourceVersionRelationDbFactory.js";

// Export existing factories
export { TeamDbFactory, createTeamDbFactory } from "./TeamDbFactory.js";
export { UserDbFactory, createUserDbFactory } from "./UserDbFactory.js";

// Re-export base classes and types
export { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
export type { RelationConfig } from "../DatabaseFixtureFactory.js";