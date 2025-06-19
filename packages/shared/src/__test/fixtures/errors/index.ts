/**
 * Central export point for all error fixtures
 * 
 * These fixtures provide consistent error scenarios for testing error handling
 * across the application, including API errors, network errors, validation errors,
 * authentication errors, business logic errors, and system errors.
 */

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