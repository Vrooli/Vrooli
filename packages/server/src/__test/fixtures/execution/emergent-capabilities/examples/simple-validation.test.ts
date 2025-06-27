/**
 * Simple validation test for new fixture examples
 * 
 * Quick test to ensure our fixture examples are correctly structured
 * and follow the emergent architecture principles.
 */

import { describe, it, expect } from "vitest";
import { codeReviewSwarm } from "./codeReviewSwarmExample.js";
import { customerServiceV1, customerServiceV4 } from "./customerServiceEvolutionExample.js";
import { validateEmergentPrinciples } from "../../emergentCapabilityHelpers.js";

describe("New Fixture Examples Validation", () => {
    describe("Code Review Swarm", () => {
        it("should have valid structure", () => {
            expect(codeReviewSwarm.config).toBeDefined();
            expect(codeReviewSwarm.emergence).toBeDefined();
            expect(codeReviewSwarm.integration).toBeDefined();
            expect(codeReviewSwarm.swarmMetadata).toBeDefined();
        });

        it("should follow emergent principles", () => {
            const validation = validateEmergentPrinciples(codeReviewSwarm.emergence);
            
            // Should not have any violations of core principles
            expect(validation.violations).toEqual([]);
            expect(validation.isValid).toBe(true);
        });

        it("should have measurable capabilities", () => {
            expect(codeReviewSwarm.emergence.capabilities).toContain("security_vulnerability_detection");
            expect(codeReviewSwarm.emergence.capabilities).toContain("performance_bottleneck_identification");
            
            // Should have concrete emergence conditions
            expect(codeReviewSwarm.emergence.emergenceConditions?.minAgents).toBeGreaterThan(0);
            expect(codeReviewSwarm.emergence.emergenceConditions?.requiredResources).toBeDefined();
        });
    });

    describe("Customer Service Evolution", () => {
        it("should show evolution from V1 to V4", () => {
            // V1 should be conversational
            expect(customerServiceV1.evolutionStage?.strategy).toBe("conversational");
            
            // V4 should be routing
            expect(customerServiceV4.evolutionStage?.strategy).toBe("routing");
            
            // V4 should have more capabilities than V1
            expect(customerServiceV4.emergence.capabilities.length).toBeGreaterThan(
                customerServiceV1.emergence.capabilities.length,
            );
        });

        it("should show performance improvements", () => {
            const v1Metrics = customerServiceV1.evolutionStage?.metrics;
            const v4Metrics = customerServiceV4.evolutionStage?.metrics;

            if (v1Metrics && v4Metrics) {
                // V4 should be faster and cheaper
                expect(v4Metrics.avgDuration).toBeLessThan(v1Metrics.avgDuration);
                expect(v4Metrics.avgCredits).toBeLessThan(v1Metrics.avgCredits);
                expect(v4Metrics.successRate).toBeGreaterThan(v1Metrics.successRate);
            }
        });

        it("should follow emergent principles for all versions", () => {
            const fixtures = [customerServiceV1, customerServiceV4];
            
            fixtures.forEach(fixture => {
                const validation = validateEmergentPrinciples(fixture.emergence);
                expect(validation.isValid).toBe(true);
                expect(validation.violations).toEqual([]);
            });
        });
    });

    describe("Data-First Configuration", () => {
        it("should demonstrate configuration-driven behavior", () => {
            const fixtures = [codeReviewSwarm, customerServiceV1];
            
            fixtures.forEach(fixture => {
                // Configuration should define goals and constraints
                expect(fixture.config.goal || fixture.config.name).toBeDefined();
                
                // Capabilities should emerge from config, not be hard-coded
                expect(fixture.emergence.capabilities.length).toBeGreaterThan(0);
                
                // Should have clear integration patterns
                expect(fixture.integration.tier).toMatch(/^(tier[123]|cross-tier)$/);
            });
        });
    });
});