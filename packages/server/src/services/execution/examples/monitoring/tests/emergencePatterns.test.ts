import { describe, it, beforeEach, afterEach, vi, expect } from "vitest";
import { createSandbox, type SinonSandbox } from "sinon";
import { type Logger } from "winston";
import { type SessionUser } from "@vrooli/shared";
import { type EventBus } from "../../../cross-cutting/events/eventBus.js";
import { 
    createPerformanceMonitorSwarm, 
    PERFORMANCE_MONITOR_SWARM_CONFIG 
} from "../swarms/performanceMonitorSwarm.js";
import { 
    createSLOGuardianSwarm, 
    SLO_GUARDIAN_SWARM_CONFIG 
} from "../swarms/sloGuardianSwarm.js";
import { 
    createPatternAnalystSwarm, 
    PATTERN_ANALYST_SWARM_CONFIG 
} from "../swarms/patternAnalystSwarm.js";
import { 
    createResourceOptimizerSwarm, 
    RESOURCE_OPTIMIZER_SWARM_CONFIG 
} from "../swarms/resourceOptimizerSwarm.js";

/**
 * Monitoring Intelligence Emergence Pattern Tests
 * 
 * These tests demonstrate how monitoring intelligence emerges from
 * swarm configurations rather than being hardcoded. They validate
 * the "two-lens" monitoring philosophy where specialized capabilities
 * develop naturally through swarm intelligence.
 */
describe("Monitoring Intelligence Emergence Patterns", function() {
    let sandbox: SinonSandbox;
    let mockLogger: Logger;
    let mockEventBus: EventBus;
    let mockUser: SessionUser;

    beforeEach(function() {
        sandbox = createSandbox();
        
        // Mock dependencies
        mockLogger = {
            info: vi.fn(),
            warn: vi.fn(),
            error: vi.fn(),
            debug: vi.fn()
        } as unknown as Logger;

        mockEventBus = {
            emit: vi.fn(),
            on: vi.fn(),
            off: vi.fn(),
            publish: vi.fn(),
            subscribe: vi.fn()
        } as unknown as EventBus;

        mockUser = {
            id: "test-user",
            name: "Test User",
            email: "test@example.com"
        } as SessionUser;
    });

    afterEach(function() {
        vi.restoreAllMocks();
    });

    describe("Adaptive Threshold Emergence", function() {
        it("should demonstrate adaptive threshold learning in Performance Monitor Swarm", async function() {
            // Create swarm configuration
            const swarmConfig = await createPerformanceMonitorSwarm(mockUser, mockLogger, mockEventBus);

            // Validate adaptive threshold capabilities
            expect(swarmConfig.adaptiveThresholds).to.exist;
            expect(swarmConfig.adaptiveThresholds.responseTime.learningRate).toBeGreaterThan(0);
            expect(swarmConfig.adaptiveThresholds.responseTime.adaptationWindow).toBeGreaterThan(0);

            // Verify members have learning configurations
            const responseTimeMonitor = swarmConfig.members.find(m => m.id === "response-time-monitor");
            expect(responseTimeMonitor).to.exist;
            expect(responseTimeMonitor!.learning_config.enabled).toBe(true);
            expect(responseTimeMonitor!.autonomy_level).toBe("adaptive");

            // Validate emergent behavior configuration
            expect(swarmConfig.emergentBehaviors.patternLearning.enabled).toBe(true);
            expect(swarmConfig.emergentBehaviors.adaptiveAlerting.enabled).toBe(true);
        });

        it("should show threshold adaptation based on system context", async function() {
            // Test production environment adaptation
            process.env.ENVIRONMENT = "production";
            const prodSwarm = await createPerformanceMonitorSwarm(mockUser, mockLogger, mockEventBus);
            
            // Verify production thresholds are stricter
            expect(prodSwarm.adaptiveThresholds.responseTime.baseline).toBe(3000);
            expect(prodSwarm.emergentBehaviors.adaptiveAlerting.escalationLevels[0]).toBe(0.5);

            // Reset environment
            delete process.env.ENVIRONMENT;
        });
    });

    describe("Pattern Discovery Emergence", function() {
        it("should demonstrate multi-dimensional pattern discovery capabilities", async function() {
            const swarmConfig = await createPatternAnalystSwarm(
                mockUser, 
                mockLogger, 
                mockEventBus,
                {
                    domain: "performance",
                    complexity: "complex",
                    real_time_requirements: true
                }
            );

            // Validate pattern detection configuration
            expect(swarmConfig.patternDetection.temporal_patterns.seasonality).toBe(true);
            expect(swarmConfig.patternDetection.behavioral_patterns.user_interactions).toBe(true);
            expect(swarmConfig.patternDetection.correlational_patterns.cross_metric).toBe(true);

            // Verify learning model diversity
            expect(swarmConfig.learningModels.unsupervised).to.include("isolation_forest");
            expect(swarmConfig.learningModels.supervised).to.include("neural_networks");
            expect(swarmConfig.learningModels.reinforcement).to.include("q_learning");

            // Validate adaptive capabilities
            expect(swarmConfig.adaptiveCapabilities.pattern_hierarchy_learning).toBe(true);
            expect(swarmConfig.adaptiveCapabilities.contextual_adaptation).toBe(true);
        });

        it("should adapt member resources based on complexity requirements", async function() {
            // Test simple complexity
            const simpleSwarm = await createPatternAnalystSwarm(
                mockUser, 
                mockLogger, 
                mockEventBus,
                { domain: "behavior", complexity: "simple", real_time_requirements: false }
            );

            // Test complex complexity  
            const complexSwarm = await createPatternAnalystSwarm(
                mockUser, 
                mockLogger, 
                mockEventBus,
                { domain: "behavior", complexity: "complex", real_time_requirements: true }
            );

            // Verify resource scaling
            const simpleDetector = simpleSwarm.members.find(m => m.id === "temporal-pattern-detector");
            const complexDetector = complexSwarm.members.find(m => m.id === "temporal-pattern-detector");

            expect(complexDetector!.resources.cpu).toBeGreaterThan(simpleDetector!.resources.cpu);
            expect(complexDetector!.resources.memory).toBeGreaterThan(simpleDetector!.resources.memory);

            // Verify real-time adaptation
            expect(complexDetector!.learning_config.feedback_window)
                .to.be.lessThan(simpleDetector!.learning_config.feedback_window);
        });
    });

    describe("SLO Intelligence Emergence", function() {
        it("should demonstrate adaptive SLO management based on business context", async function() {
            const criticalContext = {
                criticality: "critical" as const,
                industry: "financial",
                compliance_requirements: ["financial", "security"]
            };

            const swarmConfig = await createSLOGuardianSwarm(
                mockUser, 
                mockLogger, 
                mockEventBus, 
                criticalContext
            );

            // Verify critical business context adaptations
            expect(swarmConfig.sloDefinitions.availability.target).toBe(0.9999); // 99.99%
            expect(swarmConfig.errorBudgetManagement.burn_rate_alerting.fast_burn).toBe(6.0);

            // Verify compliance integration
            expect(swarmConfig.adaptiveGovernance.stakeholder_communication.escalation_matrix)
                .to.include("compliance_officer");
            expect(swarmConfig.quality_assurance.validation.compliance_verification).toBe(true);
        });

        it("should show emergent error budget intelligence", async function() {
            const swarmConfig = SLO_GUARDIAN_SWARM_CONFIG;

            // Validate error budget management capabilities
            expect(swarmConfig.errorBudgetManagement.allocation_strategy).toBe("risk_based");
            expect(swarmConfig.errorBudgetManagement.budget_exhaustion_actions).to.include("activate_circuit_breakers");

            // Verify budget manager member capabilities
            const budgetManager = swarmConfig.members.find(m => m.id === "budget-manager");
            expect(budgetManager).to.exist;
            expect(budgetManager!.capabilities).to.include("error_budget_tracking");
            expect(budgetManager!.capabilities).to.include("utilization_optimization");
        });
    });

    describe("Resource Optimization Intelligence Emergence", function() {
        it("should demonstrate multi-objective optimization emergence", async function() {
            const optimizationContext = {
                primary_objective: "efficiency" as const,
                budget_constraints: {
                    total_budget: 15000,
                    cost_sensitivity: "high" as const
                },
                performance_requirements: {
                    latency_tolerance: 300,
                    availability_requirement: 0.999,
                    throughput_minimum: 1500
                }
            };

            const swarmConfig = await createResourceOptimizerSwarm(
                mockUser, 
                mockLogger, 
                mockEventBus, 
                optimizationContext
            );

            // Verify efficiency-focused optimization
            expect(swarmConfig.optimizationTargets.cost.efficiency_target).toBe(0.95);
            expect(swarmConfig.optimizationTargets.cost.budget_limit).toBe(15000);

            // Verify cost-sensitive adaptations
            const costOptimizer = swarmConfig.members.find(m => m.id === "cost-optimizer");
            expect(costOptimizer).to.exist;
            expect(costOptimizer!.autonomy_level).toBe("fully_autonomous");
        });

        it("should show emergent predictive scaling intelligence", async function() {
            const swarmConfig = RESOURCE_OPTIMIZER_SWARM_CONFIG;

            // Validate predictive capabilities
            expect(swarmConfig.intelligentCapabilities.predictive_scaling).toBe(true);
            expect(swarmConfig.intelligentCapabilities.workload_forecasting).toBe(true);

            // Verify capacity planner capabilities
            const capacityPlanner = swarmConfig.members.find(m => m.id === "capacity-planner");
            expect(capacityPlanner).to.exist;
            expect(capacityPlanner!.capabilities).to.include("demand_prediction");
            expect(capacityPlanner!.capabilities).to.include("scaling_strategy_optimization");

            // Validate optimization strategies
            expect(swarmConfig.optimizationStrategies.scaling_policies).to.include("predictive_scaling");
            expect(swarmConfig.optimizationStrategies.scaling_policies).to.include("ml_driven_scaling");
        });
    });

    describe("Collaborative Intelligence Emergence", function() {
        it("should demonstrate swarm collaboration patterns", async function() {
            // Create multiple swarms
            const performanceSwarm = await createPerformanceMonitorSwarm(mockUser, mockLogger, mockEventBus);
            const sloSwarm = await createSLOGuardianSwarm(mockUser, mockLogger, mockEventBus);
            const patternSwarm = await createPatternAnalystSwarm(mockUser, mockLogger, mockEventBus);
            const resourceSwarm = await createResourceOptimizerSwarm(mockUser, mockLogger, mockEventBus);

            // Verify collaborative monitoring is enabled
            expect(performanceSwarm.emergentBehaviors.collaborativeMonitoring.enabled).toBe(true);
            expect(performanceSwarm.emergentBehaviors.collaborativeMonitoring.peerSwarms)
                .to.include("slo-guardian-swarm");

            // Verify shared communication channels
            const allSwarms = [performanceSwarm, sloSwarm, patternSwarm, resourceSwarm];
            allSwarms.forEach(swarm => {
                expect(swarm.communication.internal.protocol).toBe("event_driven");
                expect(swarm.communication.internal.bus).toBe("redis");
            });
        });

        it("should show emergent knowledge sharing patterns", async function() {
            const patternSwarm = PATTERN_ANALYST_SWARM_CONFIG;
            const resourceSwarm = RESOURCE_OPTIMIZER_SWARM_CONFIG;

            // Validate learning from peer feedback
            expect(patternSwarm.learning.feedback_sources).to.include("peer_swarms");
            expect(resourceSwarm.learning.feedback_sources).to.include("system_performance");

            // Verify adaptation triggers include collaboration
            expect(patternSwarm.learning.adaptation_triggers).to.include("feedback_signals");
            expect(resourceSwarm.learning.adaptation_triggers).to.include("new_optimization_opportunities");
        });
    });

    describe("Self-Optimization Emergence", function() {
        it("should demonstrate autonomous improvement capabilities", async function() {
            const swarms = [
                PERFORMANCE_MONITOR_SWARM_CONFIG,
                SLO_GUARDIAN_SWARM_CONFIG,
                PATTERN_ANALYST_SWARM_CONFIG,
                RESOURCE_OPTIMIZER_SWARM_CONFIG
            ];

            swarms.forEach(swarm => {
                // Verify self-monitoring capabilities
                expect(swarm.quality_assurance.self_monitoring.enabled).toBe(true);
                expect(swarm.quality_assurance.self_monitoring.metrics).to.be.an("array");
                expect(swarm.quality_assurance.self_monitoring.review_interval).toBeGreaterThan(0);

                // Verify learning and adaptation
                expect(swarm.learning.enabled).toBe(true);
                expect(swarm.learning.feedback_sources).to.be.an("array");
                expect(swarm.learning.adaptation_triggers).to.be.an("array");

                // Verify member autonomy levels
                swarm.members.forEach(member => {
                    expect(member.autonomy_level).to.be.oneOf([
                        "guided", "semi_autonomous", "adaptive", "fully_autonomous", "collaborative"
                    ]);
                    expect(member.learning_config.enabled).toBe(true);
                });
            });
        });

        it("should show progressive autonomy development", async function() {
            const swarmConfig = PERFORMANCE_MONITOR_SWARM_CONFIG;
            
            // Verify autonomy progression
            const members = swarmConfig.members;
            const responseTimeMonitor = members.find(m => m.id === "response-time-monitor");
            const patternCorrelator = members.find(m => m.id === "pattern-correlator");

            expect(responseTimeMonitor!.autonomy_level).toBe("adaptive");
            expect(patternCorrelator!.autonomy_level).toBe("fully_autonomous");

            // Verify learning configuration scales with autonomy
            expect(patternCorrelator!.learning_config.adaptation_rate)
                .to.be.greaterThan(responseTimeMonitor!.learning_config.adaptation_rate);
        });
    });

    describe("Context-Aware Adaptation", function() {
        it("should demonstrate environment-specific adaptations", async function() {
            // Test different environment configurations
            const environments = ["development", "staging", "production"];
            
            for (const env of environments) {
                process.env.ENVIRONMENT = env;
                const swarm = await createPerformanceMonitorSwarm(mockUser, mockLogger, mockEventBus);
                
                if (env === "production") {
                    // Production should have stricter thresholds
                    expect(swarm.adaptiveThresholds.responseTime.baseline).toBe(3000);
                } else {
                    // Non-production uses default thresholds
                    expect(swarm.adaptiveThresholds.responseTime.baseline).toBe(5000);
                }
            }
            
            delete process.env.ENVIRONMENT;
        });

        it("should show domain-specific pattern focus", async function() {
            const domains = ["performance", "behavior", "business", "security"];
            
            for (const domain of domains) {
                const swarm = await createPatternAnalystSwarm(
                    mockUser, 
                    mockLogger, 
                    mockEventBus,
                    { 
                        domain: domain as any, 
                        complexity: "moderate", 
                        real_time_requirements: false 
                    }
                );

                switch (domain) {
                    case "performance":
                        expect(swarm.patternDetection.temporal_patterns.trends).toBe(true);
                        expect(swarm.patternDetection.correlational_patterns.cross_metric).toBe(true);
                        break;
                    case "behavior":
                        expect(swarm.patternDetection.behavioral_patterns.user_interactions).toBe(true);
                        expect(swarm.patternDetection.behavioral_patterns.system_responses).toBe(true);
                        break;
                    case "security":
                        expect(swarm.patternDetection.behavioral_patterns.failure_cascades).toBe(true);
                        expect(swarm.patternDetection.correlational_patterns.causal_relationships).toBe(true);
                        break;
                }
            }
        });
    });

    describe("Emergent Specialization", function() {
        it("should demonstrate role-based capability emergence", async function() {
            const swarmConfigs = [
                PERFORMANCE_MONITOR_SWARM_CONFIG,
                SLO_GUARDIAN_SWARM_CONFIG,
                PATTERN_ANALYST_SWARM_CONFIG,
                RESOURCE_OPTIMIZER_SWARM_CONFIG
            ];

            swarmConfigs.forEach(swarm => {
                // Verify each member has distinct specialization
                const specializations = swarm.members.map(m => m.specialization);
                const uniqueSpecializations = new Set(specializations);
                expect(specializations.length).toBe(uniqueSpecializations.size);

                // Verify capabilities align with specializations
                swarm.members.forEach(member => {
                    expect(member.capabilities.length).toBeGreaterThan(0);
                    expect(member.specialization).to.be.a("string");
                    expect(member.role).to.be.a("string");
                });
            });
        });

        it("should show resource allocation based on specialization", async function() {
            const resourceSwarm = RESOURCE_OPTIMIZER_SWARM_CONFIG;
            
            // Verify resource allocation reflects member importance and complexity
            const capacityPlanner = resourceSwarm.members.find(m => m.id === "capacity-planner");
            const efficiencyMonitor = resourceSwarm.members.find(m => m.id === "efficiency-monitor");

            // Capacity planner should have more resources due to complexity
            expect(capacityPlanner!.resources.cpu).toBeGreaterThan(efficiencyMonitor!.resources.cpu);
            expect(capacityPlanner!.resources.memory).toBeGreaterThan(efficiencyMonitor!.resources.memory);
        });
    });
});