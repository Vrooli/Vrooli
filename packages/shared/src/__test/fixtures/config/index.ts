/* c8 ignore start */
/**
 * Export all config fixtures for easy importing in tests
 */
export * from "./apiConfigFixtures.js";
export * from "./baseConfigFixtures.js";
export * from "./botConfigFixtures.js";
export * from "./chatConfigFixtures.js";
export * from "./codeConfigFixtures.js";
export * from "./creditConfigFixtures.js";
export * from "./messageConfigFixtures.js";
export * from "./noteConfigFixtures.js";
export * from "./projectConfigFixtures.js";
export * from "./routineConfigFixtures.js";
export * from "./runConfigFixtures.js";
export * from "./standardConfigFixtures.js";
export * from "./teamConfigFixtures.js";
export * from "./utilsConfigFixtures.js";

// Import individual fixtures for namespace export
import { apiConfigFixtures } from "./apiConfigFixtures.js";
import { baseConfigFixtures } from "./baseConfigFixtures.js";
import { botConfigFixtures } from "./botConfigFixtures.js";
import { chatConfigFixtures } from "./chatConfigFixtures.js";
import { codeConfigFixtures } from "./codeConfigFixtures.js";
import { creditConfigFixtures } from "./creditConfigFixtures.js";
import { messageConfigFixtures } from "./messageConfigFixtures.js";
import { noteConfigFixtures } from "./noteConfigFixtures.js";
import { projectConfigFixtures } from "./projectConfigFixtures.js";
import { routineConfigFixtures } from "./routineConfigFixtures.js";
import { runConfigFixtures } from "./runConfigFixtures.js";
import { standardConfigFixtures } from "./standardConfigFixtures.js";
import { teamConfigFixtures } from "./teamConfigFixtures.js";
import { utilsConfigFixtures } from "./utilsConfigFixtures.js";

// Namespace export for convenient access
export const configFixtures = {
    apiConfigFixtures,
    baseConfigFixtures,
    botConfigFixtures,
    chatConfigFixtures,
    codeConfigFixtures,
    creditConfigFixtures,
    messageConfigFixtures,
    noteConfigFixtures,
    projectConfigFixtures,
    routineConfigFixtures,
    runConfigFixtures,
    standardConfigFixtures,
    teamConfigFixtures,
    utilsConfigFixtures,
};