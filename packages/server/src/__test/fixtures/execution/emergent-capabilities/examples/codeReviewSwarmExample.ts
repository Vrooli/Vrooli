/**
 * Code Review Swarm Example
 * 
 * Demonstrates how code quality assurance capabilities emerge from agent collaboration.
 * The swarm starts with basic code review tasks but evolves to develop:
 * - Security vulnerability detection
 * - Performance optimization suggestions
 * - Architectural improvement recommendations
 * - Best practice enforcement
 * 
 * Key principle: We configure WHAT to review, not HOW - the methods emerge!
 */

import type { SwarmFixture } from "../../types.js";
import { swarmFactory } from "../../executionFactories.js";
import { createMeasurableCapability } from "../../executionFactories.js";

/**
 * Code Review Swarm
 * 
 * Configuration: Simple code review request
 * Emergence: Develops comprehensive quality assurance through agent specialization
 */
export const codeReviewSwarm: SwarmFixture = swarmFactory.createComplete({
    config: {
        goal: "Review pull request for code quality and potential issues",
        // Note: We don't specify review criteria - agents develop their own!
        preferredModel: "gpt-4",
        swarmTask: "Collaborative code review",
        swarmSubTasks: [
            {
                id: "initial-scan",
                description: "Scan code changes", // Vague - agents determine what to scan for
                assignedAgents: [],
                status: "pending",
            },
            {
                id: "deep-analysis",
                description: "Analyze code quality", // No specific metrics given
                assignedAgents: [],
                status: "pending",
            },
            {
                id: "recommendations",
                description: "Provide feedback", // Format and focus emerge
                assignedAgents: [],
                status: "pending",
            },
        ],
        chatLabel: "Code Review Team",
        maxTokens: 6000, // Allow detailed code discussions
    },
    emergence: {
        capabilities: [
            "security_vulnerability_detection", // Emerges from pattern recognition
            "performance_bottleneck_identification", // Emerges from execution analysis
            "code_smell_detection", // Emerges from best practice learning
            "architectural_pattern_recommendation", // Emerges from codebase understanding
            "test_coverage_analysis", // Emerges from quality focus
            "dependency_risk_assessment", // Emerges from ecosystem knowledge
        ],
        eventPatterns: [
            "review.vulnerability.detected",
            "review.performance.issue.found",
            "review.pattern.violation",
            "review.improvement.suggested",
            "agent.expertise.developed",
            "agent.review.completed",
        ],
        evolutionPath: "syntax_checking → pattern_analysis → security_awareness → architectural_guidance",
        emergenceConditions: {
            minAgents: 4, // Need specialized reviewers
            requiredResources: [
                "code_repository",
                "vulnerability_database",
                "performance_benchmarks",
                "best_practices_registry",
                "architectural_patterns",
            ],
            environmentalFactors: [
                "codebase_complexity",
                "technology_stack",
                "security_requirements",
                "performance_targets",
            ],
            minEvents: 15, // Quick to start showing expertise
            timeframe: 3600000, // 1 hour for thorough review
        },
        learningMetrics: {
            performanceImprovement: "40% more issues detected through collaborative analysis",
            adaptationTime: "3 reviews to develop specialized roles",
            innovationRate: "1 new review pattern discovered per 10 PRs",
        },
        // Measurable capabilities with concrete metrics
        measurableCapabilities: [
            createMeasurableCapability(
                "vulnerability_detection_rate",
                "vulnerabilities_found_per_kloc",
                0.5, // baseline
                2.0, // target after learning
                "vulnerabilities/KLOC",
                "Security issues found per thousand lines of code",
            ),
            createMeasurableCapability(
                "review_comprehensiveness",
                "code_coverage_percentage",
                60, // baseline
                95, // target
                "%",
                "Percentage of code actually reviewed",
            ),
            createMeasurableCapability(
                "false_positive_rate",
                "incorrect_findings_percentage",
                30, // baseline
                5, // target after calibration
                "%",
                "Percentage of findings that are false positives",
            ),
        ],
        emergenceTests: [
            {
                setup: "Present swarm with code containing subtle security vulnerability",
                trigger: "Submit PR with SQL injection risk in dynamic query",
                expectedOutcome: "Agents identify and explain the vulnerability",
                measurementMethod: "Check if vulnerability detected and properly explained",
            },
            {
                setup: "Present swarm with performant but poorly structured code",
                trigger: "Submit working code with architectural anti-patterns",
                expectedOutcome: "Agents suggest architectural improvements",
                measurementMethod: "Evaluate quality of refactoring suggestions",
            },
        ],
    },
    integration: {
        tier: "tier1",
        producedEvents: [
            "tier1.review.completed",
            "tier1.vulnerability.found",
            "tier1.improvement.suggested",
            "tier1.review.expertise.gained",
        ],
        consumedEvents: [
            "tier2.code.analysis.completed",
            "tier3.security.scan.result",
            "system.pr.submitted",
        ],
        sharedResources: [
            "review_findings_board", // Agents share discoveries
            "vulnerability_knowledge_base", // Collective security knowledge
            "performance_patterns", // Learned optimization patterns
            "review_methodology", // Evolved review processes
        ],
        crossTierDependencies: {
            dependsOn: ["tier2_static_analysis", "tier3_security_tools"],
            provides: ["review_coordination", "quality_assurance", "security_validation"],
        },
        mcpTools: ["code_analyzer", "security_scanner", "performance_profiler"],
    },
    swarmMetadata: {
        formation: "dynamic", // Agents specialize based on findings
        coordinationPattern: "emergence", // Self-organizing expertise
        expectedAgentCount: 5,
        minViableAgents: 3,
        roles: [
            { role: "security_specialist", count: 1 }, // Emerges, not assigned
            { role: "performance_analyst", count: 1 },
            { role: "architecture_reviewer", count: 1 },
            { role: "test_coverage_analyst", count: 1 },
            { role: "general_reviewer", count: 1 },
        ],
    },
    validation: {
        emergenceTests: [
            "security_expertise_development",
            "performance_pattern_recognition",
            "architectural_insight_generation",
        ],
        integrationTests: [
            "finding_consolidation",
            "cross_agent_validation",
            "knowledge_accumulation",
        ],
        evolutionTests: [
            "review_speed_improvement",
            "detection_accuracy_growth",
            "false_positive_reduction",
        ],
    },
    metadata: {
        domain: "development",
        complexity: "complex",
        maintainer: "devops-team",
        lastUpdated: new Date().toISOString(),
    },
});

/**
 * Demonstrates evolution: Same swarm after learning from 100 reviews
 * Shows how capabilities have evolved through experience
 */
export const evolvedCodeReviewSwarm: SwarmFixture = swarmFactory.createComplete({
    config: {
        ...codeReviewSwarm.config,
        // Config stays the same - behavior evolves!
    },
    emergence: {
        ...codeReviewSwarm.emergence,
        capabilities: [
            ...codeReviewSwarm.emergence.capabilities,
            // New capabilities that emerged through experience:
            "zero_day_vulnerability_prediction", // Emerged from pattern correlation
            "performance_regression_prediction", // Emerged from historical analysis
            "technical_debt_quantification", // Emerged from codebase understanding
            "team_knowledge_gap_identification", // Emerged from review patterns
        ],
        evolutionPath: codeReviewSwarm.emergence.evolutionPath + " → predictive_quality_assurance",
        learningMetrics: {
            performanceImprovement: "80% faster reviews with 3x more findings",
            adaptationTime: "Instant specialization based on code characteristics",
            innovationRate: "Discovers new vulnerability patterns weekly",
        },
        measurableCapabilities: [
            createMeasurableCapability(
                "vulnerability_detection_rate",
                "vulnerabilities_found_per_kloc",
                2.0, // improved from original target
                3.5, // new target
                "vulnerabilities/KLOC",
                "Now includes zero-day pattern matching",
            ),
            createMeasurableCapability(
                "predictive_accuracy",
                "predicted_issues_materialized",
                0, // didn't exist before
                75, // high accuracy predictions
                "%",
                "Percentage of predicted issues that actually occur",
            ),
        ],
    },
    swarmMetadata: {
        ...codeReviewSwarm.swarmMetadata,
        formation: "hierarchical", // Evolved to specialized hierarchy
        coordinationPattern: "delegation", // Learned efficient delegation
        expectedAgentCount: 7, // Grew to handle specialized roles
    },
});

/**
 * Example showing how the same configuration leads to different emergent
 * capabilities in a different domain (infrastructure vs application code)
 */
export const infrastructureReviewSwarm: SwarmFixture = swarmFactory.createComplete({
    config: {
        goal: "Review infrastructure-as-code changes",
        swarmTask: "Infrastructure review",
        swarmSubTasks: codeReviewSwarm.config.swarmSubTasks, // Same tasks!
    },
    emergence: {
        capabilities: [
            // Different capabilities emerge for infrastructure:
            "cost_optimization_analysis", // Emerges from resource awareness
            "scalability_assessment", // Emerges from load pattern understanding  
            "disaster_recovery_validation", // Emerges from failure mode analysis
            "compliance_verification", // Emerges from regulatory knowledge
            "multi_cloud_compatibility", // Emerges from platform experience
        ],
        eventPatterns: [
            "infra.cost.analyzed",
            "infra.scalability.assessed",
            "infra.compliance.verified",
            "infra.optimization.found",
        ],
        evolutionPath: "resource_checking → cost_awareness → architectural_optimization → predictive_scaling",
        emergenceConditions: {
            ...codeReviewSwarm.emergence.emergenceConditions,
            requiredResources: [
                "cloud_pricing_data",
                "compliance_frameworks",
                "scalability_patterns",
                "disaster_recovery_playbooks",
            ],
        },
    },
    integration: {
        ...codeReviewSwarm.integration,
        mcpTools: ["terraform_analyzer", "cost_calculator", "compliance_checker"],
    },
});
