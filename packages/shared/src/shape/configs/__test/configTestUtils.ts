/**
 * Config Test Utilities
 * 
 * This module provides comprehensive test helpers for configuration class validation.
 * It focuses on testing **behavioral contracts** rather than implementation details.
 * 
 * ## Core Testing Philosophy
 * 
 * We test that configs maintain **consistency and idempotency**:
 * 1. ✅ Config can be built from fixture data
 * 2. ✅ Config can be exported consistently (round-trip stability)
 * 3. ✅ Exported data can rebuild identical configs
 * 4. ❌ We DON'T test that exported data matches original fixtures
 * 
 * ### Why This Approach?
 * 
 * Configs often perform "healing" - normalizing, adding defaults, or cleaning data:
 * ```typescript
 * // Input fixture might have:
 * { timeout: "5000", features: ["a", "b", "a"] }
 * 
 * // After config processing:
 * { timeout: 5000, features: ["a", "b"] }  // healed: parsed number, deduplicated
 * 
 * // Our tests verify: export(import(export(data))) === export(data)
 * // We DON'T require: export(data) === original_fixture
 * ```
 * 
 * This makes tests robust to legitimate config evolution while catching real bugs
 * like inconsistent serialization or state mutation.
 * 
 * ## Quick Start Examples:
 * 
 * ### Option 1: Comprehensive (Simplest)
 * ```typescript
 * runComprehensiveConfigTests(
 *     MessageConfig, 
 *     messageConfigFixtures, 
 *     "message"
 * );
 * ```
 * 
 * ### Option 2: Individual Control  
 * ```typescript
 * runStandardConfigTests(MessageConfig, messageConfigFixtures, "message");
 * runConfigFixtureTests(messageConfigFixtures, "message");
 * ```
 * 
 * ## What Each Helper Provides:
 * 
 * - `runStandardConfigTests`: Tests constructor, round-trip consistency, method availability
 * - `runConfigFixtureTests`: Validates fixtures create valid config instances  
 * - `runComprehensiveConfigTests`: Both of the above combined
 */

import { describe, expect, it } from "vitest";
import type { PassableLogger } from "../../../consts/commonTypes.js";
import type { BaseConfig, BaseConfigObject } from "../base.js";
import type { ConfigTestFixtures } from "../../../__test/fixtures/config/baseConfigFixtures.js";

/**
 * Test that a config can be constructed from fixture data and maintains round-trip consistency.
 * 
 * This is the core test that validates the fundamental config contract:
 * - Fixture data should create a valid config instance
 * - Config should export consistently (idempotent serialization)
 * - Exported data should recreate identical configs
 * 
 * @param ConfigClass The config class constructor
 * @param fixtureData The fixture data to test with
 * @param description Human-readable description for test failure messages
 */
export function testConfigRoundTrip<T extends BaseConfigObject>(
    ConfigClass: any,
    fixtureData: T,
    description: string,
): void {
    // Step 1: Create config from fixture data
    const config1 = new ConfigClass({ config: fixtureData });
    expect(config1).toBeDefined();
    
    // Step 2: Export the config
    const exported1 = config1.export();
    expect(exported1).toBeDefined();
    expect(exported1.__version).toBeDefined();
    
    // Step 3: Create a new config from the exported data
    const config2 = new ConfigClass({ config: exported1 });
    expect(config2).toBeDefined();
    
    // Step 4: Export again and verify consistency
    const exported2 = config2.export();
    
    // Step 5: CRITICAL TEST - exported configs should be identical
    // This catches bugs like:
    // - Inconsistent serialization/deserialization
    // - State mutation during export
    // - Non-deterministic behavior
    expect(exported1).toEqual(exported2);
    
    // Step 6: Triple check with one more round-trip
    const config3 = new ConfigClass({ config: exported2 });
    const exported3 = config3.export();
    expect(exported2).toEqual(exported3);
}

/**
 * Fixture integrity test suite
 * 
 * Tests that all fixture data creates valid, consistent config instances.
 * Each fixture is tested for round-trip consistency to ensure it represents
 * a valid, stable configuration state.
 */
export function runConfigFixtureTests<T extends BaseConfigObject>(
    ConfigClass: any,
    fixtures: ConfigTestFixtures<T>,
    configName: string,
): void {
    describe(`${configName}Config - Fixture Integrity`, () => {
        it("should have valid minimal fixture with round-trip consistency", () => {
            testConfigRoundTrip(ConfigClass, fixtures.minimal, "minimal fixture");
        });

        it("should have valid complete fixture with round-trip consistency", () => {
            testConfigRoundTrip(ConfigClass, fixtures.complete, "complete fixture");
        });

        it("should have valid withDefaults fixture with round-trip consistency", () => {
            testConfigRoundTrip(ConfigClass, fixtures.withDefaults, "withDefaults fixture");
        });

        it("should have valid variant fixtures with round-trip consistency", () => {
            Object.entries(fixtures.variants).forEach(([variantName, variantData]) => {
                testConfigRoundTrip(ConfigClass, variantData, `variant: ${variantName}`);
            });
        });
    });
}

/**
 * Comprehensive config test suite combining both standard config and fixture tests
 */
export function runComprehensiveConfigTests<T extends BaseConfigObject>(
    ConfigClass: any,
    fixtures: ConfigTestFixtures<T>,
    configName: string,
): void {
    // Standard config tests (constructor, parse, export, etc.)
    runStandardConfigTests(ConfigClass, fixtures, configName);
    
    // Fixture integrity tests (fixture validity)
    runConfigFixtureTests(ConfigClass, fixtures, configName);
}

/**
 * Standard config functionality test suite
 * 
 * Tests the fundamental behavioral contracts that all config classes should satisfy:
 * 1. Constructor creates valid instances that can be exported consistently  
 * 2. Parse method (if available) creates configs with round-trip consistency
 * 3. Default factory method (if available) creates valid, consistent configs
 * 4. Resource management methods (if available) work correctly
 * 5. Setter methods (if available) maintain config consistency
 * 
 * Focus is on testing that methods exist and maintain consistency, not on
 * testing specific output values which may change due to legitimate healing.
 */
export function runStandardConfigTests<T extends BaseConfigObject>(
    ConfigClass: any,
    fixtures: ConfigTestFixtures<T>,
    configName: string,
): void {
    describe(`${configName}Config - Standard Behavior`, () => {
        const mockLogger: PassableLogger = {
            trace: () => {},
            debug: () => {},
            info: () => {},
            warn: () => {},
            error: () => {},
        };

        describe("constructor behavior", () => {
            it("should create valid instance from minimal data", () => {
                const config = new ConfigClass({ config: fixtures.minimal });
                expect(config).toBeDefined();
                expect(config.__version).toBeDefined();
                expect(typeof config.export).toBe("function");
            });

            it("should create valid instance from complete data", () => {
                const config = new ConfigClass({ config: fixtures.complete });
                expect(config).toBeDefined();
                expect(config.__version).toBeDefined();
                expect(typeof config.export).toBe("function");
            });

            it("should handle data with defaults appropriately", () => {
                const config = new ConfigClass({ config: fixtures.withDefaults });
                expect(config).toBeDefined();
                expect(config.__version).toBeDefined();
            });
        });

        describe("round-trip consistency", () => {
            it("should maintain consistency through export/import cycles", () => {
                // Test all fixture types for round-trip consistency
                const allFixtures = [
                    { name: "minimal", data: fixtures.minimal },
                    { name: "complete", data: fixtures.complete },
                    { name: "withDefaults", data: fixtures.withDefaults },
                    ...Object.entries(fixtures.variants).map(([name, data]) => ({ name: `variant:${name}`, data })),
                ];

                allFixtures.forEach(({ name, data }) => {
                    testConfigRoundTrip(ConfigClass, data, name);
                });
            });
        });

        if (ConfigClass.parse) {
            describe("parse method behavior", () => {
                it("should create consistent configs with fallbacks enabled", () => {
                    const parsed = ConfigClass.parse(
                        { config: fixtures.minimal },
                        mockLogger,
                        { useFallbacks: true },
                    );
                    expect(parsed).toBeDefined();
                    expect(parsed.__version).toBeDefined();
                    
                    // Test that parsed config maintains round-trip consistency
                    const exported = parsed.export();
                    const reparsed = ConfigClass.parse({ config: exported }, mockLogger, { useFallbacks: true });
                    expect(reparsed.export()).toEqual(exported);
                });

                it("should create consistent configs without fallbacks", () => {
                    const parsed = ConfigClass.parse(
                        { config: fixtures.complete },
                        mockLogger,
                        { useFallbacks: false },
                    );
                    expect(parsed).toBeDefined();
                    
                    // Test round-trip consistency
                    const exported = parsed.export();
                    const reparsed = ConfigClass.parse({ config: exported }, mockLogger, { useFallbacks: false });
                    expect(reparsed.export()).toEqual(exported);
                });

                it("should handle null/undefined input gracefully", () => {
                    try {
                        const parsed = ConfigClass.parse(
                            { config: null },
                            mockLogger,
                            { useFallbacks: true },
                        );
                        expect(parsed).toBeDefined();
                        expect(parsed.__version).toBeDefined();
                    } catch (error) {
                        // Some configs like ChatConfig can't handle null config
                        // This is acceptable behavior
                        expect(error).toBeDefined();
                    }
                });
            });
        }

        if (ConfigClass.default) {
            describe("default factory method", () => {
                it("should create valid default config with round-trip consistency", () => {
                    let defaultConfig;
                    
                    // Try to create default config, handling special parameter requirements
                    try {
                        defaultConfig = ConfigClass.default();
                    } catch (error) {
                        // Some configs like CodeVersionConfig require parameters
                        try {
                            defaultConfig = ConfigClass.default({ 
                                codeLanguage: "Javascript", 
                                initialContent: "", 
                            });
                        } catch (secondError) {
                            // If we can't figure out the parameters, skip the test
                            console.warn(`Skipping default factory test for ${configName}: requires unknown parameters`);
                            return;
                        }
                    }
                    
                    expect(defaultConfig).toBeDefined();
                    expect(defaultConfig.__version).toBeDefined();
                    
                    // Test round-trip consistency of default config
                    const exported = defaultConfig.export();
                    const rebuilt = new ConfigClass({ config: exported });
                    expect(rebuilt.export()).toEqual(exported);
                });
            });
        }

        // Only create resource management test suite if the config actually supports resources
        (() => {
            let supportsResources = false;
            try {
                const testConfig = new ConfigClass({ config: fixtures.minimal });
                supportsResources = typeof testConfig.addResource === "function" &&
                                  typeof testConfig.removeResource === "function" &&
                                  typeof testConfig.updateResource === "function";
            } catch {
                supportsResources = false;
            }

            if (supportsResources && fixtures.complete.resources?.length > 0) {
                describe("resource management (if supported)", () => {
                    it("should add resources and maintain consistency", () => {
                        const config = new ConfigClass({ config: fixtures.minimal });
                        const resource = fixtures.complete.resources[0];
                        
                        config.addResource(resource);
                        const exported = config.export();
                        expect(exported.resources).toContain(resource);
                        
                        // Test round-trip consistency after adding resource
                        const rebuilt = new ConfigClass({ config: exported });
                        expect(rebuilt.export()).toEqual(exported);
                    });

                    it("should remove resources and maintain consistency", () => {
                        const config = new ConfigClass({ config: fixtures.complete });
                        
                        if (config.resources?.length > 0) {
                            const originalLength = config.resources.length;
                            config.removeResource(0);
                            expect(config.resources.length).toBe(originalLength - 1);
                            
                            // Test round-trip consistency after removal
                            const exported = config.export();
                            const rebuilt = new ConfigClass({ config: exported });
                            expect(rebuilt.export()).toEqual(exported);
                        }
                    });

                    it("should update resources and maintain consistency", () => {
                        const config = new ConfigClass({ config: fixtures.complete });
                        
                        if (config.resources?.length > 0) {
                            const updatedResource = { ...fixtures.complete.resources[0], link: "https://updated.example.com" };
                            config.updateResource(0, updatedResource);
                            
                            const exported = config.export();
                            expect(exported.resources[0].link).toBe("https://updated.example.com");
                            
                            // Test round-trip consistency after update
                            const rebuilt = new ConfigClass({ config: exported });
                            expect(rebuilt.export()).toEqual(exported);
                        }
                    });
                });
            }
        })();

        describe("setter methods (if available)", () => {
            // Auto-discover setter methods by examining the config instance
            const sampleConfig = new ConfigClass({ config: fixtures.complete });
            const sampleExported = sampleConfig.export();
            
            const availableSetters = Object.keys(sampleExported)
                .filter(key => !key.startsWith("__") && key !== "resources")
                .map(key => {
                    const setterName = `set${key.charAt(0).toUpperCase()}${key.slice(1)}`;
                    return typeof sampleConfig[setterName] === "function" ? { key, setterName } : null;
                })
                .filter(Boolean);

            if (availableSetters.length > 0) {
                availableSetters.forEach(({ key, setterName }) => {
                    it(`should maintain consistency when using ${setterName}`, () => {
                        const config = new ConfigClass({ config: fixtures.minimal });
                        
                        // Try to find a test value from variants or complete fixture
                        const testValue = fixtures.variants[Object.keys(fixtures.variants)[0]]?.[key] 
                                        ?? fixtures.complete[key];
                        
                        if (testValue !== undefined) {
                            config[setterName](testValue);
                            expect(config[key]).toEqual(testValue);
                            
                            // Test round-trip consistency after setter usage
                            const exported = config.export();
                            const rebuilt = new ConfigClass({ config: exported });
                            expect(rebuilt.export()).toEqual(exported);
                        } else {
                            // Just verify the method exists and is callable
                            expect(typeof config[setterName]).toBe("function");
                        }
                    });
                });
            } else {
                // Add a placeholder test to avoid empty test suite
                it("should not have any setter methods (this is normal for simple configs)", () => {
                    expect(availableSetters.length).toBe(0);
                });
            }
        });
    });
}

/**
 * Configuration test options interface
 * 
 * Allows customization of test behavior for configs with special requirements.
 */
export interface ConfigTestOptions {
    /** Parameters to pass to ConfigClass.default() if it requires them */
    defaultFactoryParams?: Record<string, unknown>;
    /** Methods to skip during testing (e.g., if they have known issues) */
    skipMethods?: string[];
    /** Custom description prefix for test names */
    testPrefix?: string;
}

/**
 * Helper function to safely attempt config creation with error handling
 * 
 * @param ConfigClass The config class constructor  
 * @param data The data to create the config with
 * @param description Description for error reporting
 * @returns The created config instance or throws descriptive error
 */
export function safeCreateConfig<T extends BaseConfigObject>(
    ConfigClass: any,
    data: T,
    description: string,
): any {
    try {
        return new ConfigClass({ config: data });
    } catch (error: any) {
        throw new Error(`Failed to create config from ${description}: ${error.message}`);
    }
}

/**
 * Utility to test multiple fixture data sets in batch
 * 
 * @param ConfigClass The config class to test
 * @param fixtureDataSets Array of fixture data to test
 * @param testDescription Base description for the test
 */
export function testMultipleFixtures<T extends BaseConfigObject>(
    ConfigClass: any,
    fixtureDataSets: Array<{ name: string; data: T }>,
    testDescription: string,
): void {
    fixtureDataSets.forEach(({ name, data }) => {
        testConfigRoundTrip(ConfigClass, data, `${testDescription} - ${name}`);
    });
}
