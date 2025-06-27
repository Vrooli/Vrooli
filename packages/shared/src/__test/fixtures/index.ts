/* c8 ignore start */
/**
 * Central export for all test fixtures in the shared package
 */

// Export API fixtures
export * from "./api-inputs/index.js";

// Export config fixtures
export * from "./config/index.js";

// Export error fixtures
export * from "./errors/index.js";

// Export event fixtures
export * from "./events/index.js";

// Export API response fixtures
export * from "./api-responses/index.js";

// Namespace exports for better organization
export * as apiFixtures from "./api-inputs/index.js";
export * as configFixtures from "./config/index.js";
export * as errorFixtures from "./errors/index.js";
export * as eventFixtures from "./events/index.js";
export * as apiResponseFixtures from "./api-responses/index.js";
