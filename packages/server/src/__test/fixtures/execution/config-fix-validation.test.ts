/**
 * Test to validate that the config object structure mismatch has been fixed
 * 
 * This test ensures that:
 * 1. SwarmFixture creates valid ChatConfigObject instances
 * 2. RoutineFixture creates valid RunProgress instances  
 * 3. The validation utilities work with real config objects
 * 4. Round-trip validation passes for all fixture types
 */

import { describe, it, expect } from "vitest";
import { ChatConfig, RunProgressConfig } from "@vrooli/shared";
import { swarmFactory, routineFactory } from "./executionFactories.js";
import { validateConfigWithSharedFixtures } from "./executionValidationUtils.js";

describe("Config Object Structure Fix Validation", () => {
    describe("SwarmFixture with ChatConfigObject", () => {
        it("should create minimal swarm fixture with valid ChatConfigObject", async () => {
            const fixture = swarmFactory.createMinimal();
            
            // Validate that the config follows ChatConfigObject structure
            expect(fixture.config).toHaveProperty("__version");
            expect(fixture.config).toHaveProperty("stats");
            expect(fixture.config.stats).toHaveProperty("totalToolCalls");
            expect(fixture.config.stats).toHaveProperty("totalCredits");
            expect(fixture.config.stats).toHaveProperty("startedAt");
            expect(fixture.config.stats).toHaveProperty("lastProcessingCycleEndedAt");
            
            // Validate using ChatConfig class
            const chatConfig = new ChatConfig({ config: fixture.config });
            const exported = chatConfig.export();
            expect(exported).toBeTruthy();
            
            // Round-trip validation should pass
            const reimported = new ChatConfig({ config: exported });
            const reexported = reimported.export();
            expect(JSON.stringify(exported)).toBe(JSON.stringify(reexported));
        });

        it("should create complete swarm fixture with valid ChatConfigObject", async () => {
            const fixture = swarmFactory.createComplete();
            
            // Validate complex structure
            expect(fixture.config).toHaveProperty("goal");
            expect(fixture.config).toHaveProperty("subtasks");
            expect(fixture.config).toHaveProperty("blackboard");
            expect(fixture.config).toHaveProperty("resources");
            
            // Validate using ChatConfig class  
            const chatConfig = new ChatConfig({ config: fixture.config });
            const exported = chatConfig.export();
            expect(exported).toBeTruthy();
        });

        it("should pass comprehensive validation with shared fixtures", async () => {
            const fixture = swarmFactory.createMinimal();
            
            const validationResult = await validateConfigWithSharedFixtures(fixture, "chat");
            expect(validationResult.pass).toBe(true);
            if (!validationResult.pass) {
                console.error("Validation errors:", validationResult.errors);
                console.error("Validation warnings:", validationResult.warnings);
            }
        });
    });

    describe("RoutineFixture with RunProgress", () => {
        it("should create minimal routine fixture with valid RunProgress", async () => {
            const fixture = routineFactory.createMinimal();
            
            // Validate that the config follows RunProgress structure
            expect(fixture.config).toHaveProperty("__version");
            expect(fixture.config).toHaveProperty("branches");
            expect(fixture.config).toHaveProperty("config");
            
            // Validate using RunProgressConfig class
            const runConfig = new RunProgressConfig({ config: fixture.config });
            const exported = runConfig.export();
            expect(exported).toBeTruthy();
            
            // Round-trip validation should pass
            const reimported = new RunProgressConfig({ config: exported });
            const reexported = reimported.export();
            expect(JSON.stringify(exported)).toBe(JSON.stringify(reexported));
        });

        it("should create complete routine fixture with valid RunProgress", async () => {
            const fixture = routineFactory.createComplete();
            
            // Validate complex structure
            expect(fixture.config).toHaveProperty("__version");
            expect(fixture.config).toHaveProperty("branches");
            expect(fixture.config).toHaveProperty("config");
            
            // Validate using RunProgressConfig class
            const runConfig = new RunProgressConfig({ config: fixture.config });
            const exported = runConfig.export();
            expect(exported).toBeTruthy();
        });

        it("should pass comprehensive validation with shared fixtures", async () => {
            const fixture = routineFactory.createMinimal();
            
            const validationResult = await validateConfigWithSharedFixtures(fixture, "routine");
            expect(validationResult.pass).toBe(true);
            if (!validationResult.pass) {
                console.error("Validation errors:", validationResult.errors);
                console.error("Validation warnings:", validationResult.warnings);
            }
        });
    });

    describe("Factory Evolution Paths", () => {
        it("should create valid evolution path for swarm fixtures", () => {
            const evolutionPath = swarmFactory.createEvolutionPath(3);
            
            expect(evolutionPath).toHaveLength(3);
            
            evolutionPath.forEach((fixture, index) => {
                // Each fixture should have valid ChatConfigObject
                expect(fixture.config).toHaveProperty("__version");
                expect(fixture.config).toHaveProperty("stats");
                
                // Evolution should show progression
                expect(fixture.emergence.capabilities).toContain("basic_coordination");
                if (index >= 1) {
                    expect(fixture.emergence.capabilities).toContain("pattern_recognition");
                }
                if (index >= 2) {
                    expect(fixture.emergence.capabilities).toContain("predictive_optimization");
                }
            });
        });

        it("should create valid evolution path for routine fixtures", () => {
            const evolutionPath = routineFactory.createEvolutionPath(3);
            
            expect(evolutionPath).toHaveLength(3);
            
            evolutionPath.forEach((fixture, index) => {
                // Each fixture should have valid RunProgress
                expect(fixture.config).toHaveProperty("__version");
                expect(fixture.config).toHaveProperty("branches");
                
                // Evolution should show progression
                expect(fixture.emergence.capabilities).toContain("basic_execution");
                if (index >= 1) {
                    expect(fixture.emergence.capabilities).toContain("pattern_learning");
                }
                if (index >= 2) {
                    expect(fixture.emergence.capabilities).toContain("optimization");
                }
            });
        });
    });

    describe("Factory Validation Methods", () => {
        it("should validate swarm fixtures correctly", async () => {
            const fixture = swarmFactory.createMinimal();
            
            const configResult = await swarmFactory.validateConfig(fixture.config);
            expect(configResult.pass).toBe(true);
            
            const fixtureResult = await swarmFactory.validateFixture(fixture);
            expect(fixtureResult.pass).toBe(true);
        });

        it("should validate routine fixtures correctly", async () => {
            const fixture = routineFactory.createMinimal();
            
            const configResult = await routineFactory.validateConfig(fixture.config);
            expect(configResult.pass).toBe(true);
            
            const fixtureResult = await routineFactory.validateFixture(fixture);
            expect(fixtureResult.pass).toBe(true);
        });
    });

    describe("Fixture Type Guards", () => {
        it("should correctly identify valid fixtures", () => {
            const swarmFixture = swarmFactory.createMinimal();
            const routineFixture = routineFactory.createMinimal();
            
            expect(swarmFactory.isValid(swarmFixture)).toBe(true);
            expect(routineFactory.isValid(routineFixture)).toBe(true);
            
            // Should reject invalid fixtures
            expect(swarmFactory.isValid({})).toBe(false);
            expect(swarmFactory.isValid(null)).toBe(false);
            expect(routineFactory.isValid(undefined)).toBe(false);
        });
    });
});