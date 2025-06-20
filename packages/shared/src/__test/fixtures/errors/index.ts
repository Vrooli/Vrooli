/**
 * Central export point for all error fixtures
 * 
 * These fixtures provide consistent error scenarios for testing error handling
 * across the application, including API errors, network errors, validation errors,
 * authentication errors, business logic errors, and system errors.
 * 
 * Upgraded to use the unified error fixture factory pattern with enhanced
 * error composition, recovery testing, and round-trip testing integration.
 */

// Core types and infrastructure
export * from "./composition.js";
export * from "./types.js";

// All error fixture files
export * from "./apiErrors.js";
export * from "./authErrors.js";
export * from "./businessErrors.js";
export * from "./networkErrors.js";
export * from "./systemErrors.js";
export * from "./validationErrors.js";

// Aggregate namespace export for convenience
export { apiErrorFixtures } from "./apiErrors.js";
export { authErrorFixtures } from "./authErrors.js";
export { businessErrorFixtures } from "./businessErrors.js";
export { networkErrorFixtures } from "./networkErrors.js";
export { systemErrorFixtures } from "./systemErrors.js";
export { validationErrorFixtures } from "./validationErrors.js";

// Enhanced error composition and utilities
export { errorComposer, errorScenarios, errorTestUtils, recoveryScenarioBuilder } from "./composition.js";

// Import all fixtures for the aggregate export
import { apiErrorFixtures } from "./apiErrors.js";
import { authErrorFixtures } from "./authErrors.js";
import { businessErrorFixtures } from "./businessErrors.js";
import { networkErrorFixtures } from "./networkErrors.js";
import { systemErrorFixtures } from "./systemErrors.js";
import { validationErrorFixtures } from "./validationErrors.js";

// Aggregate error fixtures object for easy import
export const errorFixtures = {
    api: apiErrorFixtures,
    auth: authErrorFixtures,
    business: businessErrorFixtures,
    network: networkErrorFixtures,
    system: systemErrorFixtures,
    validation: validationErrorFixtures,
};
