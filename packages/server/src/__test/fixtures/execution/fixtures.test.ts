/**
 * Tests for Production Execution Fixtures
 * 
 * Validates that our implemented fixtures follow the emergent capabilities
 * philosophy and integrate properly with the three-tier architecture.
 */

import { describe, it, expect } from "vitest";
import { runComprehensiveExecutionTests } from "./validationTestUtils.js";
import {
    securityMonitoringSwarm,
    complianceAuditSwarm,
    vulnerabilityAssessmentRoutine,
    dataPrivacyComplianceRoutine,
    secureCodeExecutionContext,
    highPerformanceAnalyticsContext,
    securityResponseIntegration,
    complianceMonitoringIntegration,
    executionFixtures
} from "./fixtures.js";

// ================================================================================================
// Tier 1 Fixture Tests (Coordination Intelligence)
// ================================================================================================

describe("Tier 1: Coordination Intelligence Fixtures", () => {
    describe("Security Monitoring Swarm", () => {
        // Use automatic test generation for comprehensive validation
        runComprehensiveExecutionTests(
            securityMonitoringSwarm,
            "chat",
            "security-monitoring-swarm"
        );
        
        it("should demonstrate emergent threat detection capabilities", () => {
            expect(securityMonitoringSwarm.emergence.capabilities).toContain("adaptive_threat_detection");
            expect(securityMonitoringSwarm.emergence.capabilities).toContain("coordinated_incident_response");
            expect(securityMonitoringSwarm.emergence.capabilities).toContain("predictive_security_analysis");
        });
        
        it("should show evolution from reactive to predictive defense", () => {
            expect(securityMonitoringSwarm.emergence.evolutionPath).toBe(
                "reactive monitoring → proactive hunting → predictive defense"
            );
        });
        
        it("should coordinate across security event patterns", () => {
            const patterns = securityMonitoringSwarm.emergence.eventPatterns;
            expect(patterns).toContain("security.*");
            expect(patterns).toContain("tier2.routine.security.*");
            expect(patterns).toContain("tier3.execution.security.*");
        });
        
        it("should require minimum viable agents for coordination", () => {
            expect(securityMonitoringSwarm.swarmMetadata?.expectedAgentCount).toBeGreaterThan(1);
            expect(securityMonitoringSwarm.swarmMetadata?.minViableAgents).toBeGreaterThan(0);
            expect(securityMonitoringSwarm.swarmMetadata?.minViableAgents).toBeLessThanOrEqual(
                securityMonitoringSwarm.swarmMetadata?.expectedAgentCount || 0
            );
        });
    });
    
    describe("Compliance Audit Swarm", () => {
        runComprehensiveExecutionTests(
            complianceAuditSwarm,
            "chat", 
            "compliance-audit-swarm"
        );
        
        it("should demonstrate cross-regulatory compliance capabilities", () => {
            expect(complianceAuditSwarm.emergence.capabilities).toContain("regulatory_pattern_recognition");
            expect(complianceAuditSwarm.emergence.capabilities).toContain("cross_domain_compliance_correlation");
        });
    });
});

// ================================================================================================
// Tier 2 Fixture Tests (Process Intelligence)
// ================================================================================================

describe("Tier 2: Process Intelligence Fixtures", () => {
    describe("Vulnerability Assessment Routine", () => {
        runComprehensiveExecutionTests(
            vulnerabilityAssessmentRoutine,
            "routine",
            "vulnerability-assessment-routine"
        );
        
        it("should demonstrate adaptive security assessment capabilities", () => {
            expect(vulnerabilityAssessmentRoutine.emergence.capabilities).toContain("adaptive_scan_optimization");
            expect(vulnerabilityAssessmentRoutine.emergence.capabilities).toContain("intelligent_risk_prioritization");
            expect(vulnerabilityAssessmentRoutine.emergence.capabilities).toContain("vulnerability_pattern_learning");
        });
        
        it("should show measurable evolution metrics", () => {
            const metrics = vulnerabilityAssessmentRoutine.emergence.learningMetrics;
            expect(metrics?.performanceImprovement).toContain("60% reduction in false positives");
            expect(metrics?.adaptationTime).toContain("5 assessments");
            expect(metrics?.innovationRate).toContain("1 new assessment technique per 20 scans");
        });
        
        it("should have evolved to reasoning strategy", () => {
            expect(vulnerabilityAssessmentRoutine.evolutionStage.strategy).toBe("reasoning");
            expect(vulnerabilityAssessmentRoutine.evolutionStage.version).toBe("2.1.0");
            expect(vulnerabilityAssessmentRoutine.evolutionStage.successRate).toBeGreaterThan(0.9);
        });
        
        it("should define comprehensive workflow", () => {
            const nodes = vulnerabilityAssessmentRoutine.config.nodes;
            expect(nodes).toHaveLength(4);
            expect(nodes.map(n => n.id)).toEqual([
                "discovery-scan",
                "vulnerability-scan", 
                "risk-analysis",
                "remediation-planning"
            ]);
        });
    });
    
    describe("Data Privacy Compliance Routine", () => {
        runComprehensiveExecutionTests(
            dataPrivacyComplianceRoutine,
            "routine",
            "data-privacy-compliance-routine"
        );
        
        it("should demonstrate privacy protection capabilities", () => {
            expect(dataPrivacyComplianceRoutine.emergence.capabilities).toContain("intelligent_data_classification");
            expect(dataPrivacyComplianceRoutine.emergence.capabilities).toContain("adaptive_consent_monitoring");
        });
        
        it("should have evolved to deterministic strategy for compliance reliability", () => {
            expect(dataPrivacyComplianceRoutine.evolutionStage.strategy).toBe("deterministic");
            expect(dataPrivacyComplianceRoutine.evolutionStage.successRate).toBeGreaterThan(0.95);
        });
    });
});

// ================================================================================================
// Tier 3 Fixture Tests (Execution Intelligence)
// ================================================================================================

describe("Tier 3: Execution Intelligence Fixtures", () => {
    describe("Secure Code Execution Context", () => {
        runComprehensiveExecutionTests(
            secureCodeExecutionContext,
            "run",
            "secure-code-execution-context"
        );
        
        it("should demonstrate intelligent security capabilities", () => {
            expect(secureCodeExecutionContext.emergence.capabilities).toContain("intelligent_threat_detection");
            expect(secureCodeExecutionContext.emergence.capabilities).toContain("adaptive_sandboxing");
            expect(secureCodeExecutionContext.emergence.capabilities).toContain("runtime_security_monitoring");
        });
        
        it("should prioritize security over performance", () => {
            expect(secureCodeExecutionContext.tools?.preferences?.costOptimization).toBe(false);
            expect(secureCodeExecutionContext.tools?.preferences?.performanceOptimization).toBe(false);
        });
        
        it("should require approval for high-risk operations", () => {
            const constraints = secureCodeExecutionContext.context.constraints;
            expect(constraints.requireApproval).toContain("code_execution");
            expect(constraints.requireApproval).toContain("file_operations");
            expect(constraints.requireApproval).toContain("network_access");
        });
        
        it("should have comprehensive security monitoring", () => {
            const safety = secureCodeExecutionContext.context.safety;
            expect(safety?.syncChecks).toContain("input_sanitization");
            expect(safety?.syncChecks).toContain("code_validation");
            expect(safety?.asyncAgents).toContain("security_monitor");
            expect(safety?.asyncAgents).toContain("threat_detector");
        });
        
        it("should define emergency stop conditions", () => {
            const emergencyStop = secureCodeExecutionContext.context.safety?.emergencyStop;
            expect(emergencyStop?.conditions).toContain("malicious_code_detected");
            expect(emergencyStop?.conditions).toContain("privilege_escalation_attempt");
            expect(emergencyStop?.actions).toContain("immediate_halt");
            expect(emergencyStop?.actions).toContain("security_alert");
        });
    });
    
    describe("High-Performance Analytics Context", () => {
        runComprehensiveExecutionTests(
            highPerformanceAnalyticsContext,
            "run",
            "high-performance-analytics-context"
        );
        
        it("should demonstrate optimization capabilities", () => {
            expect(highPerformanceAnalyticsContext.emergence.capabilities).toContain("adaptive_resource_allocation");
            expect(highPerformanceAnalyticsContext.emergence.capabilities).toContain("intelligent_query_optimization");
        });
        
        it("should use deterministic strategy for performance", () => {
            expect(highPerformanceAnalyticsContext.strategy).toBe("deterministic");
        });
    });
});

// ================================================================================================
// Integration Scenario Tests
// ================================================================================================

describe("Integration Scenarios", () => {
    describe("Security Response Integration", () => {
        it("should coordinate all three tiers for security incidents", () => {
            expect(securityResponseIntegration.tier1).toBe(securityMonitoringSwarm);
            expect(securityResponseIntegration.tier2).toBe(vulnerabilityAssessmentRoutine);
            expect(securityResponseIntegration.tier3).toBe(secureCodeExecutionContext);
        });
        
        it("should demonstrate end-to-end emergent capabilities", () => {
            const capabilities = securityResponseIntegration.emergence.capabilities;
            expect(capabilities).toContain("end_to_end_threat_response");
            expect(capabilities).toContain("coordinated_security_intelligence");
            expect(capabilities).toContain("cross_tier_security_optimization");
        });
        
        it("should define realistic performance metrics", () => {
            const metrics = securityResponseIntegration.emergence.metrics;
            expect(metrics?.latency).toBe(30000); // 30 seconds
            expect(metrics?.accuracy).toBeGreaterThan(0.9);
            expect(metrics?.reliability).toBeGreaterThan(0.95);
        });
        
        it("should include comprehensive test scenarios", () => {
            const scenarios = securityResponseIntegration.testScenarios;
            expect(scenarios).toHaveLength(2);
            
            const sqlInjectionTest = scenarios?.find(s => s.name === "SQL Injection Attack");
            expect(sqlInjectionTest).toBeDefined();
            expect(sqlInjectionTest?.successCriteria).toHaveLength(3);
            
            const codeExecutionTest = scenarios?.find(s => s.name === "Suspicious Code Execution");
            expect(codeExecutionTest).toBeDefined();
            expect(codeExecutionTest?.successCriteria).toHaveLength(3);
        });
    });
    
    describe("Compliance Monitoring Integration", () => {
        it("should coordinate compliance across all tiers", () => {
            expect(complianceMonitoringIntegration.tier1).toBe(complianceAuditSwarm);
            expect(complianceMonitoringIntegration.tier2).toBe(dataPrivacyComplianceRoutine);
            expect(complianceMonitoringIntegration.tier3).toBe(secureCodeExecutionContext);
        });
        
        it("should demonstrate holistic compliance capabilities", () => {
            const capabilities = complianceMonitoringIntegration.emergence.capabilities;
            expect(capabilities).toContain("holistic_compliance_monitoring");
            expect(capabilities).toContain("cross_regulatory_correlation");
            expect(capabilities).toContain("predictive_compliance_management");
        });
    });
});

// ================================================================================================
// Collection and Export Tests
// ================================================================================================

describe("Fixture Collection", () => {
    it("should export all fixtures in organized collection", () => {
        expect(executionFixtures.securityMonitoringSwarm).toBeDefined();
        expect(executionFixtures.complianceAuditSwarm).toBeDefined();
        expect(executionFixtures.vulnerabilityAssessmentRoutine).toBeDefined();
        expect(executionFixtures.dataPrivacyComplianceRoutine).toBeDefined();
        expect(executionFixtures.secureCodeExecutionContext).toBeDefined();
        expect(executionFixtures.highPerformanceAnalyticsContext).toBeDefined();
        expect(executionFixtures.securityResponseIntegration).toBeDefined();
        expect(executionFixtures.complianceMonitoringIntegration).toBeDefined();
    });
    
    it("should demonstrate data-first emergent capabilities philosophy", () => {
        // All fixtures should demonstrate that capabilities emerge from configuration
        Object.values(executionFixtures).forEach(fixture => {
            if ("emergence" in fixture) {
                expect(fixture.emergence.capabilities.length).toBeGreaterThan(0);
                expect(fixture.emergence.capabilities.every(cap => 
                    typeof cap === "string" && cap.length > 0
                )).toBe(true);
            }
        });
    });
    
    it("should cover all three tiers", () => {
        const tier1Fixtures = Object.values(executionFixtures).filter(f => 
            "integration" in f && f.integration.tier === "tier1"
        );
        const tier2Fixtures = Object.values(executionFixtures).filter(f =>
            "integration" in f && f.integration.tier === "tier2"  
        );
        const tier3Fixtures = Object.values(executionFixtures).filter(f =>
            "integration" in f && f.integration.tier === "tier3"
        );
        
        expect(tier1Fixtures.length).toBeGreaterThan(0);
        expect(tier2Fixtures.length).toBeGreaterThan(0);
        expect(tier3Fixtures.length).toBeGreaterThan(0);
    });
    
    it("should focus on defensive security capabilities", () => {
        const securityFixtures = Object.values(executionFixtures).filter(fixture => {
            if ("metadata" in fixture && fixture.metadata?.domain === "security") {
                return true;
            }
            if ("emergence" in fixture) {
                return fixture.emergence.capabilities.some(cap => 
                    cap.includes("security") || 
                    cap.includes("threat") || 
                    cap.includes("compliance") ||
                    cap.includes("vulnerability")
                );
            }
            return false;
        });
        
        expect(securityFixtures.length).toBeGreaterThan(4); // Most fixtures should be security-focused
    });
});

// ================================================================================================  
// Emergent Behavior Validation
// ================================================================================================

describe("Emergent Behavior Validation", () => {
    it("should demonstrate measurable emergence in all fixtures", () => {
        const fixturesWithEmergence = Object.values(executionFixtures).filter(f => "emergence" in f);
        
        fixturesWithEmergence.forEach(fixture => {
            if ("emergence" in fixture) {
                // Should have capabilities
                expect(fixture.emergence.capabilities.length).toBeGreaterThan(0);
                
                // Should show evolution if specified
                if (fixture.emergence.evolutionPath) {
                    expect(fixture.emergence.evolutionPath).toContain("→");
                }
                
                // Should have learning metrics for complex capabilities
                if (fixture.emergence.capabilities.some(cap => cap.includes("adaptive") || cap.includes("intelligent"))) {
                    expect(fixture.emergence.learningMetrics).toBeDefined();
                }
            }
        });
    });
    
    it("should show progression in evolution stages", () => {
        // Vulnerability assessment should show clear evolution
        const vulnAssessment = vulnerabilityAssessmentRoutine;
        expect(vulnAssessment.evolutionStage.strategy).toBe("reasoning");
        expect(vulnAssessment.evolutionStage.previousVersion).toBe("2.0.0");
        expect(vulnAssessment.evolutionStage.improvements?.length).toBeGreaterThan(0);
        
        // Privacy compliance should be at deterministic stage
        const privacyCompliance = dataPrivacyComplianceRoutine;
        expect(privacyCompliance.evolutionStage.strategy).toBe("deterministic");
        expect(privacyCompliance.evolutionStage.successRate).toBeGreaterThan(0.95);
    });
});