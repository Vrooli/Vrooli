/**
 * Config Fixture Coverage Matrix
 * 
 * This file documents which config fixture variants are used in API fixtures,
 * helping ensure comprehensive test coverage and prevent duplication.
 */

export const CONFIG_FIXTURE_COVERAGE = {
    routine: {
        // Base configs
        "minimal": ["routineAPIFixtures.minimal.create"],
        "complete": ["routineAPIFixtures.complete.create"],
        "withDefaults": [], // Not currently used
        
        // Invalid configs
        "invalid.missingVersion": ["routineAPIFixtures.invalidConfigs.missingVersion"],
        "invalid.invalidVersion": ["routineAPIFixtures.invalidConfigs.invalidVersion"],
        "invalid.malformedStructure": ["routineAPIFixtures.invalidConfigs.malformedStructure"],
        "invalid.invalidTypes": ["routineAPIFixtures.invalidConfigs.invalidTypes"],
        
        // Variants
        "variants.simpleApiCall": ["routineAPIFixtures.edgeCases.apiRoutine"],
        "variants.textGeneration": ["routineAPIFixtures.edgeCases.textGenerationRoutine"],
        "variants.dataTransformation": ["routineAPIFixtures.edgeCases.dataTransformRoutine"],
        
        // Action variants
        "action.simple": ["routineAPIFixtures.customizer.default", "routineAPIFixtures.forkRoutine"],
        "action.withInputMapping": [], // Not currently used
        "action.withOutputMapping": [], // Not currently used
        "action.withMachine": [], // Not currently used
        
        // Generate variants
        "generate.basic": [], // Not currently used
        "generate.withCustomModel": ["routineAPIFixtures.complete.update"],
        "generate.withComplexPrompt": [], // Not currently used
        
        // Multi-step variants
        "multiStep.sequential": ["routineAPIFixtures.complete.create (version 2)"],
        "multiStep.withBranching": ["routineAPIFixtures.edgeCases.allComplexityLevels"],
        "multiStep.complexWorkflow": ["routineAPIFixtures.find results"],
    },
    
    message: {
        // Base configs
        "minimal": [
            "chatMessageFixtures.minimal.create",
            "chatMessageFixtures.edgeCases.maxLengthText",
            "chatMessageFixtures.edgeCases.textTooLong",
            "chatMessageFixtures.edgeCases.toolCallWithSuccessResult",
            "chatMessageFixtures.edgeCases.toolCallWithErrorResult",
            "chatMessageFixtures.edgeCases.toolCallWithoutResult",
        ],
        "complete": [
            "chatMessageFixtures.complete.create",
            "chatMessageFixtures.edgeCases.complexConfig",
        ],
        "withDefaults": [], // Not currently used
        
        // Invalid configs
        "invalid.missingVersion": ["chatMessageFixtures.invalidConfigs.missingVersion"],
        "invalid.invalidVersion": ["chatMessageFixtures.invalidConfigs.invalidVersion"],
        "invalid.malformedStructure": ["chatMessageFixtures.invalidConfigs.malformedStructure"],
        "invalid.invalidTypes": ["chatMessageFixtures.invalid.invalidToolCall"],
        
        // Variants
        "variants.userMessage": ["chatMessageFixtures.edgeCases.allRoles[0]"],
        "variants.assistantWithTools": [
            "chatMessageFixtures.complete.update",
            "chatMessageFixtures.edgeCases.allRoles[1]",
        ],
        "variants.systemMessage": ["chatMessageFixtures.edgeCases.allRoles[2]"],
        "variants.toolErrorMessage": ["chatMessageFixtures.edgeCases.allRoles[3]"],
        "variants.messageWithMultipleRuns": ["chatMessageFixtures.edgeCases.messageWithMultipleRuns"],
        "variants.broadcastMessage": ["chatMessageFixtures.edgeCases.broadcastMessage"],
        "variants.eventDrivenMessage": ["chatMessageFixtures.edgeCases.eventDrivenMessage"],
    },
};

/**
 * Get unused config fixture variants
 */
export function getUnusedConfigVariants(configType: "routine" | "message"): string[] {
    const coverage = CONFIG_FIXTURE_COVERAGE[configType];
    return Object.entries(coverage)
        .filter(([_, usage]) => usage.length === 0)
        .map(([variant]) => variant);
}

/**
 * Get config fixture usage statistics
 */
export function getConfigUsageStats(configType: "routine" | "message"): {
    total: number;
    used: number;
    unused: number;
    percentage: number;
} {
    const coverage = CONFIG_FIXTURE_COVERAGE[configType];
    const total = Object.keys(coverage).length;
    const used = Object.values(coverage).filter(usage => usage.length > 0).length;
    const unused = total - used;
    const percentage = Math.round((used / total) * 100);
    
    return { total, used, unused, percentage };
}

// Usage examples:
// console.log(getUnusedConfigVariants('routine'));
// console.log(getConfigUsageStats('message'));
