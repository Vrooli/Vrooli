/**
 * Security Validator Test Fixtures
 * 
 * Comprehensive test fixtures for the SecurityValidator component, providing
 * validated configurations for testing security policies, threat detection,
 * and compliance validation across all tiers.
 */

import type {
    BaseConfigObject,
    RunConfigObject,
} from "@vrooli/shared";
import {
    runConfigFixtures,
} from "@vrooli/shared";
import type {
    ExecutionFixture,
    EmergenceDefinition,
    IntegrationDefinition,
} from "../types.js";
import { BaseExecutionFixtureFactory } from "../executionFactories.js";

/**
 * Security threat levels
 */
export type ThreatLevel = "info" | "low" | "medium" | "high" | "critical";

/**
 * Security validation strategies
 */
export type ValidationStrategy = "synchronous" | "asynchronous" | "hybrid" | "real-time";

/**
 * Compliance standards
 */
export type ComplianceStandard = "GDPR" | "HIPAA" | "SOC2" | "PCI-DSS" | "ISO27001" | "NIST";

/**
 * Security-specific fixture interface
 */
export interface SecurityFixture<TConfig extends BaseConfigObject = RunConfigObject> extends ExecutionFixture<TConfig> {
    security: {
        /** Security policies */
        policies: Array<{
            id: string;
            name: string;
            type: "access" | "data" | "network" | "execution" | "compliance";
            rules: Array<{
                condition: string;
                action: "allow" | "deny" | "audit" | "alert";
                severity: ThreatLevel;
            }>;
            enabled: boolean;
            enforcementMode: "enforce" | "monitor" | "disabled";
        }>;
        /** Threat detection configuration */
        threatDetection: {
            enabled: boolean;
            strategies: Array<{
                type: "anomaly" | "signature" | "behavioral" | "ml-based";
                sensitivity: number;
                updateFrequency: number;
            }>;
            threatIntelligence: {
                sources: string[];
                updateInterval: number;
                autoBlock: boolean;
            };
        };
        /** Validation configuration */
        validation: {
            strategy: ValidationStrategy;
            layers: Array<{
                name: string;
                tier: "tier1" | "tier2" | "tier3" | "all";
                checks: string[];
                blocking: boolean;
            }>;
            timeout: number;
            fallbackBehavior: "fail-closed" | "fail-open" | "degraded";
        };
        /** Compliance requirements */
        compliance: {
            standards: ComplianceStandard[];
            auditingEnabled: boolean;
            retentionDays: number;
            encryptionRequired: boolean;
            dataClassification: {
                enabled: boolean;
                levels: Array<"public" | "internal" | "confidential" | "secret">;
            };
        };
        /** Security metrics */
        metrics: {
            threatDetectionRate: number;
            falsePositiveRate: number;
            validationLatency: number;
            complianceScore: number;
            incidentResponseTime: number;
        };
    };
}

/**
 * Factory for creating security validator test fixtures
 */
export class SecurityValidatorFixtureFactory extends BaseExecutionFixtureFactory<RunConfigObject> {
    protected ConfigClass = runConfigFixtures.RunConfig;
    protected tier = "cross-tier" as const;

    createMinimal(overrides?: Partial<SecurityFixture>): SecurityFixture {
        return {
            config: runConfigFixtures.minimal,
            emergence: {
                capabilities: ["basic_validation", "access_control"],
                evolutionPath: "static → dynamic → adaptive",
                ...overrides?.emergence,
            },
            integration: {
                tier: "cross-tier",
                producedEvents: ["security.validation.completed", "security.threat.detected"],
                consumedEvents: ["*.request", "*.response"],
                ...overrides?.integration,
            },
            security: {
                policies: [{
                    id: "basic_access",
                    name: "Basic Access Control",
                    type: "access",
                    rules: [{
                        condition: "authenticated === true",
                        action: "allow",
                        severity: "low",
                    }],
                    enabled: true,
                    enforcementMode: "enforce",
                }],
                threatDetection: {
                    enabled: false,
                    strategies: [],
                    threatIntelligence: {
                        sources: [],
                        updateInterval: 86400000,
                        autoBlock: false,
                    },
                },
                validation: {
                    strategy: "synchronous",
                    layers: [{
                        name: "authentication",
                        tier: "all",
                        checks: ["auth_token"],
                        blocking: true,
                    }],
                    timeout: 5000,
                    fallbackBehavior: "fail-closed",
                },
                compliance: {
                    standards: [],
                    auditingEnabled: false,
                    retentionDays: 30,
                    encryptionRequired: false,
                    dataClassification: {
                        enabled: false,
                        levels: ["public"],
                    },
                },
                metrics: {
                    threatDetectionRate: 0,
                    falsePositiveRate: 0,
                    validationLatency: 100,
                    complianceScore: 0,
                    incidentResponseTime: 60000,
                },
                ...overrides?.security,
            },
        };
    }

    createComplete(overrides?: Partial<SecurityFixture>): SecurityFixture {
        return {
            config: runConfigFixtures.complete,
            emergence: {
                capabilities: [
                    "adaptive_threat_detection",
                    "zero_trust_validation",
                    "behavioral_analysis",
                    "predictive_security",
                    "autonomous_response",
                    "compliance_automation",
                ],
                eventPatterns: [
                    "security.*",
                    "auth.*",
                    "access.*",
                    "threat.*",
                    "compliance.*",
                ],
                evolutionPath: "static → dynamic → predictive → autonomous → self-healing",
                emergenceConditions: {
                    minEvents: 100000,
                    requiredResources: ["threat_intelligence", "ml_models", "security_analytics"],
                    timeframe: 2592000000, // 30 days
                },
                learningMetrics: {
                    performanceImprovement: "99.9% threat detection accuracy",
                    adaptationTime: "5 minutes to new threat patterns",
                    innovationRate: "20 new security patterns per month",
                },
                ...overrides?.emergence,
            },
            integration: {
                tier: "cross-tier",
                producedEvents: [
                    "security.validation.started",
                    "security.validation.completed",
                    "security.threat.detected",
                    "security.threat.mitigated",
                    "security.incident.created",
                    "security.compliance.violation",
                    "security.policy.updated",
                ],
                consumedEvents: [
                    "tier1.request.*",
                    "tier2.execution.*",
                    "tier3.operation.*",
                    "auth.attempt.*",
                    "data.access.*",
                    "network.traffic.*",
                ],
                sharedResources: [
                    "security_policies",
                    "threat_database",
                    "audit_logs",
                    "compliance_reports",
                ],
                crossTierDependencies: {
                    dependsOn: ["authentication", "authorization", "encryption"],
                    provides: ["security_validation", "threat_protection", "compliance_enforcement"],
                },
                mcpTools: ["policy_engine", "threat_analyzer", "compliance_checker"],
                ...overrides?.integration,
            },
            security: {
                policies: [
                    {
                        id: "zero_trust_access",
                        name: "Zero Trust Access Policy",
                        type: "access",
                        rules: [
                            {
                                condition: "authenticated && authorized && deviceTrusted",
                                action: "allow",
                                severity: "medium",
                            },
                            {
                                condition: "anomalyScore > 0.8",
                                action: "deny",
                                severity: "high",
                            },
                            {
                                condition: "privilegedOperation && !mfaVerified",
                                action: "deny",
                                severity: "critical",
                            },
                        ],
                        enabled: true,
                        enforcementMode: "enforce",
                    },
                    {
                        id: "data_protection",
                        name: "Data Protection Policy",
                        type: "data",
                        rules: [
                            {
                                condition: "dataClassification === 'secret'",
                                action: "audit",
                                severity: "high",
                            },
                            {
                                condition: "bulkDataAccess && !approved",
                                action: "deny",
                                severity: "high",
                            },
                        ],
                        enabled: true,
                        enforcementMode: "enforce",
                    },
                    {
                        id: "network_security",
                        name: "Network Security Policy",
                        type: "network",
                        rules: [
                            {
                                condition: "sourceIP in blacklist",
                                action: "deny",
                                severity: "critical",
                            },
                            {
                                condition: "unusualTrafficPattern",
                                action: "alert",
                                severity: "medium",
                            },
                        ],
                        enabled: true,
                        enforcementMode: "enforce",
                    },
                    {
                        id: "execution_safety",
                        name: "Execution Safety Policy",
                        type: "execution",
                        rules: [
                            {
                                condition: "untrustedCode",
                                action: "deny",
                                severity: "critical",
                            },
                            {
                                condition: "resourceLimit exceeded",
                                action: "deny",
                                severity: "medium",
                            },
                        ],
                        enabled: true,
                        enforcementMode: "enforce",
                    },
                    {
                        id: "gdpr_compliance",
                        name: "GDPR Compliance Policy",
                        type: "compliance",
                        rules: [
                            {
                                condition: "personalData && !consentGiven",
                                action: "deny",
                                severity: "high",
                            },
                            {
                                condition: "dataExport && euResident",
                                action: "audit",
                                severity: "medium",
                            },
                        ],
                        enabled: true,
                        enforcementMode: "enforce",
                    },
                ],
                threatDetection: {
                    enabled: true,
                    strategies: [
                        {
                            type: "anomaly",
                            sensitivity: 0.85,
                            updateFrequency: 3600000, // Hourly
                        },
                        {
                            type: "signature",
                            sensitivity: 0.95,
                            updateFrequency: 86400000, // Daily
                        },
                        {
                            type: "behavioral",
                            sensitivity: 0.8,
                            updateFrequency: 1800000, // 30 minutes
                        },
                        {
                            type: "ml-based",
                            sensitivity: 0.9,
                            updateFrequency: 7200000, // 2 hours
                        },
                    ],
                    threatIntelligence: {
                        sources: [
                            "internal_telemetry",
                            "threat_feeds",
                            "security_vendors",
                            "community_intelligence",
                        ],
                        updateInterval: 900000, // 15 minutes
                        autoBlock: true,
                    },
                },
                validation: {
                    strategy: "hybrid",
                    layers: [
                        {
                            name: "perimeter",
                            tier: "tier1",
                            checks: ["ip_reputation", "geo_blocking", "rate_limiting"],
                            blocking: true,
                        },
                        {
                            name: "authentication",
                            tier: "all",
                            checks: ["token_validation", "session_verification", "device_trust"],
                            blocking: true,
                        },
                        {
                            name: "authorization",
                            tier: "all",
                            checks: ["rbac", "abac", "policy_evaluation"],
                            blocking: true,
                        },
                        {
                            name: "data_validation",
                            tier: "tier3",
                            checks: ["input_sanitization", "schema_validation", "injection_prevention"],
                            blocking: true,
                        },
                        {
                            name: "behavioral",
                            tier: "all",
                            checks: ["anomaly_detection", "pattern_analysis", "risk_scoring"],
                            blocking: false,
                        },
                    ],
                    timeout: 2000,
                    fallbackBehavior: "fail-closed",
                },
                compliance: {
                    standards: ["GDPR", "SOC2", "ISO27001"],
                    auditingEnabled: true,
                    retentionDays: 365,
                    encryptionRequired: true,
                    dataClassification: {
                        enabled: true,
                        levels: ["public", "internal", "confidential", "secret"],
                    },
                },
                metrics: {
                    threatDetectionRate: 0.999,
                    falsePositiveRate: 0.001,
                    validationLatency: 50,
                    complianceScore: 0.98,
                    incidentResponseTime: 300, // 5 minutes
                },
                ...overrides?.security,
            },
            validation: {
                emergenceTests: [
                    "adaptive_threat_learning",
                    "zero_trust_enforcement",
                    "compliance_automation",
                ],
                integrationTests: [
                    "cross_tier_policy_enforcement",
                    "threat_intelligence_integration",
                    "audit_trail_completeness",
                ],
                evolutionTests: [
                    "detection_accuracy_improvement",
                    "false_positive_reduction",
                    "response_time_optimization",
                ],
                ...overrides?.validation,
            },
            metadata: {
                domain: "security",
                complexity: "complex",
                maintainer: "security-team",
                lastUpdated: new Date().toISOString(),
                ...overrides?.metadata,
            },
        };
    }

    createWithDefaults(overrides?: Partial<SecurityFixture>): SecurityFixture {
        return this.createMinimal({
            security: {
                policies: [
                    {
                        id: "standard_security",
                        name: "Standard Security Policy",
                        type: "access",
                        rules: [
                            {
                                condition: "authenticated",
                                action: "allow",
                                severity: "low",
                            },
                            {
                                condition: "suspiciousActivity",
                                action: "alert",
                                severity: "medium",
                            },
                        ],
                        enabled: true,
                        enforcementMode: "enforce",
                    },
                ],
                threatDetection: {
                    enabled: true,
                    strategies: [{
                        type: "signature",
                        sensitivity: 0.8,
                        updateFrequency: 86400000,
                    }],
                    threatIntelligence: {
                        sources: ["internal_telemetry"],
                        updateInterval: 3600000,
                        autoBlock: false,
                    },
                },
                validation: {
                    strategy: "synchronous",
                    layers: [
                        {
                            name: "basic_security",
                            tier: "all",
                            checks: ["authentication", "basic_authorization"],
                            blocking: true,
                        },
                    ],
                    timeout: 3000,
                    fallbackBehavior: "fail-closed",
                },
                compliance: {
                    standards: ["SOC2"],
                    auditingEnabled: true,
                    retentionDays: 90,
                    encryptionRequired: true,
                    dataClassification: {
                        enabled: true,
                        levels: ["public", "internal"],
                    },
                },
                metrics: {
                    threatDetectionRate: 0.95,
                    falsePositiveRate: 0.05,
                    validationLatency: 200,
                    complianceScore: 0.85,
                    incidentResponseTime: 1800, // 30 minutes
                },
            },
            ...overrides,
        });
    }

    createVariant(variant: string, overrides?: Partial<SecurityFixture>): SecurityFixture {
        const variants: Record<string, () => SecurityFixture> = {
            zeroTrust: () => this.createComplete({
                emergence: {
                    capabilities: [
                        "continuous_verification",
                        "microsegmentation",
                        "least_privilege_enforcement",
                    ],
                },
                security: {
                    policies: [
                        {
                            id: "never_trust_always_verify",
                            name: "Zero Trust Policy",
                            type: "access",
                            rules: [
                                {
                                    condition: "true", // Always verify
                                    action: "audit",
                                    severity: "info",
                                },
                                {
                                    condition: "!continuouslyVerified",
                                    action: "deny",
                                    severity: "high",
                                },
                            ],
                            enabled: true,
                            enforcementMode: "enforce",
                        },
                    ],
                    validation: {
                        strategy: "real-time",
                        layers: [
                            {
                                name: "continuous_verification",
                                tier: "all",
                                checks: [
                                    "identity_verification",
                                    "device_posture",
                                    "network_context",
                                    "behavior_analysis",
                                ],
                                blocking: true,
                            },
                        ],
                        timeout: 1000,
                        fallbackBehavior: "fail-closed",
                    },
                },
                ...overrides,
            }),

            hipaaCompliant: () => this.createComplete({
                emergence: {
                    capabilities: [
                        "phi_protection",
                        "access_logging",
                        "encryption_enforcement",
                    ],
                },
                security: {
                    policies: [
                        {
                            id: "hipaa_phi_protection",
                            name: "HIPAA PHI Protection",
                            type: "data",
                            rules: [
                                {
                                    condition: "dataType === 'PHI'",
                                    action: "audit",
                                    severity: "high",
                                },
                                {
                                    condition: "PHI && !encrypted",
                                    action: "deny",
                                    severity: "critical",
                                },
                            ],
                            enabled: true,
                            enforcementMode: "enforce",
                        },
                    ],
                    compliance: {
                        standards: ["HIPAA"],
                        auditingEnabled: true,
                        retentionDays: 2190, // 6 years
                        encryptionRequired: true,
                        dataClassification: {
                            enabled: true,
                            levels: ["public", "internal", "confidential", "secret"],
                        },
                    },
                },
                ...overrides,
            }),

            highThreatEnvironment: () => this.createComplete({
                emergence: {
                    capabilities: [
                        "advanced_threat_hunting",
                        "automated_incident_response",
                        "deception_technology",
                    ],
                },
                security: {
                    threatDetection: {
                        enabled: true,
                        strategies: [
                            {
                                type: "ml-based",
                                sensitivity: 0.99,
                                updateFrequency: 300000, // 5 minutes
                            },
                            {
                                type: "behavioral",
                                sensitivity: 0.95,
                                updateFrequency: 600000, // 10 minutes
                            },
                        ],
                        threatIntelligence: {
                            sources: [
                                "government_feeds",
                                "premium_intelligence",
                                "dark_web_monitoring",
                                "honeypot_network",
                            ],
                            updateInterval: 60000, // 1 minute
                            autoBlock: true,
                        },
                    },
                    metrics: {
                        threatDetectionRate: 0.9999,
                        falsePositiveRate: 0.0001,
                        validationLatency: 20,
                        complianceScore: 1.0,
                        incidentResponseTime: 60, // 1 minute
                    },
                },
                ...overrides,
            }),

            developmentEnvironment: () => this.createMinimal({
                security: {
                    policies: [{
                        id: "dev_relaxed",
                        name: "Development Policy",
                        type: "access",
                        rules: [{
                            condition: "environment === 'development'",
                            action: "allow",
                            severity: "info",
                        }],
                        enabled: true,
                        enforcementMode: "monitor",
                    }],
                    validation: {
                        strategy: "asynchronous",
                        layers: [{
                            name: "dev_monitoring",
                            tier: "all",
                            checks: ["basic_auth"],
                            blocking: false,
                        }],
                        timeout: 10000,
                        fallbackBehavior: "fail-open",
                    },
                    compliance: {
                        standards: [],
                        auditingEnabled: true,
                        retentionDays: 7,
                        encryptionRequired: false,
                        dataClassification: {
                            enabled: false,
                            levels: ["public"],
                        },
                    },
                },
                ...overrides,
            }),
        };

        const factory = variants[variant];
        if (!factory) {
            throw new Error(`Unknown security variant: ${variant}`);
        }

        return factory();
    }

    /**
     * Create security incident scenarios
     */
    createIncidentScenario(
        incident: "data_breach" | "ddos_attack" | "insider_threat" | "zero_day",
    ): SecurityFixture {
        const scenarios = {
            data_breach: () => this.createComplete({
                security: {
                    policies: [{
                        id: "breach_response",
                        name: "Data Breach Response",
                        type: "data",
                        rules: [
                            {
                                condition: "unauthorizedDataAccess",
                                action: "deny",
                                severity: "critical",
                            },
                            {
                                condition: "dataExfiltrationAttempt",
                                action: "alert",
                                severity: "critical",
                            },
                        ],
                        enabled: true,
                        enforcementMode: "enforce",
                    }],
                    threatDetection: {
                        enabled: true,
                        strategies: [{
                            type: "anomaly",
                            sensitivity: 0.99,
                            updateFrequency: 60000,
                        }],
                        threatIntelligence: {
                            sources: ["breach_indicators"],
                            updateInterval: 60000,
                            autoBlock: true,
                        },
                    },
                },
            }),

            ddos_attack: () => this.createComplete({
                security: {
                    policies: [{
                        id: "ddos_mitigation",
                        name: "DDoS Mitigation",
                        type: "network",
                        rules: [
                            {
                                condition: "requestRate > 1000",
                                action: "deny",
                                severity: "high",
                            },
                        ],
                        enabled: true,
                        enforcementMode: "enforce",
                    }],
                    validation: {
                        strategy: "real-time",
                        layers: [{
                            name: "rate_limiting",
                            tier: "tier1",
                            checks: ["rate_limit", "geo_filter"],
                            blocking: true,
                        }],
                        timeout: 100,
                        fallbackBehavior: "fail-closed",
                    },
                },
            }),

            insider_threat: () => this.createComplete({
                security: {
                    policies: [{
                        id: "insider_detection",
                        name: "Insider Threat Detection",
                        type: "access",
                        rules: [
                            {
                                condition: "privilegedAccessAnomaly",
                                action: "alert",
                                severity: "high",
                            },
                        ],
                        enabled: true,
                        enforcementMode: "monitor",
                    }],
                    threatDetection: {
                        enabled: true,
                        strategies: [{
                            type: "behavioral",
                            sensitivity: 0.9,
                            updateFrequency: 300000,
                        }],
                        threatIntelligence: {
                            sources: ["user_behavior_analytics"],
                            updateInterval: 300000,
                            autoBlock: false,
                        },
                    },
                },
            }),

            zero_day: () => this.createComplete({
                security: {
                    threatDetection: {
                        enabled: true,
                        strategies: [{
                            type: "ml-based",
                            sensitivity: 0.95,
                            updateFrequency: 60000,
                        }],
                        threatIntelligence: {
                            sources: ["zero_day_research", "sandbox_analysis"],
                            updateInterval: 60000,
                            autoBlock: true,
                        },
                    },
                    validation: {
                        strategy: "hybrid",
                        layers: [{
                            name: "zero_day_protection",
                            tier: "all",
                            checks: ["behavior_analysis", "sandboxing"],
                            blocking: true,
                        }],
                        timeout: 5000,
                        fallbackBehavior: "fail-closed",
                    },
                },
            }),
        };

        const factory = scenarios[incident];
        if (!factory) {
            throw new Error(`Unknown incident scenario: ${incident}`);
        }

        return factory();
    }

    /**
     * Create evolution path showing security maturity growth
     */
    createEvolutionPath(stages = 5): SecurityFixture[] {
        const evolutionStages = ["basic", "managed", "defined", "quantified", "optimizing"];
        
        return Array.from({ length: Math.min(stages, evolutionStages.length) }, (_, i) => {
            const complianceStandards: ComplianceStandard[] = [
                [],
                ["SOC2"],
                ["SOC2", "ISO27001"],
                ["SOC2", "ISO27001", "GDPR"],
                ["SOC2", "ISO27001", "GDPR", "HIPAA", "PCI-DSS"],
            ][i] as ComplianceStandard[];

            return this.createComplete({
                emergence: {
                    capabilities: [
                        "basic_protection",
                        ...(i >= 1 ? ["policy_enforcement"] : []),
                        ...(i >= 2 ? ["threat_detection"] : []),
                        ...(i >= 3 ? ["predictive_security"] : []),
                        ...(i >= 4 ? ["autonomous_security"] : []),
                    ],
                    evolutionPath: evolutionStages.slice(0, i + 1).join(" → "),
                },
                security: {
                    policies: Array.from({ length: i + 1 }, (_, j) => ({
                        id: `policy_level_${j}`,
                        name: `Level ${j} Security Policy`,
                        type: ["access", "data", "network", "execution", "compliance"][j] as any,
                        rules: [{
                            condition: "true",
                            action: i < 2 ? "allow" : "audit" as any,
                            severity: ["info", "low", "medium", "high", "critical"][i] as ThreatLevel,
                        }],
                        enabled: true,
                        enforcementMode: i < 2 ? "monitor" : "enforce" as any,
                    })),
                    threatDetection: {
                        enabled: i >= 2,
                        strategies: i >= 2 ? [{
                            type: ["signature", "anomaly", "behavioral", "ml-based"][i - 2] as any,
                            sensitivity: 0.7 + (0.05 * i),
                            updateFrequency: 86400000 / (i || 1),
                        }] : [],
                        threatIntelligence: {
                            sources: i >= 3 ? ["internal", "external", "premium"].slice(0, i - 2) : [],
                            updateInterval: 86400000 / (i || 1),
                            autoBlock: i >= 4,
                        },
                    },
                    validation: {
                        strategy: ["synchronous", "synchronous", "asynchronous", "hybrid", "real-time"][i] as ValidationStrategy,
                        layers: Array.from({ length: Math.min(i + 1, 5) }, (_, j) => ({
                            name: ["auth", "authz", "data", "behavior", "ml"][j],
                            tier: "all" as const,
                            checks: [`check_${j}`],
                            blocking: j < 3,
                        })),
                        timeout: 5000 - (i * 800),
                        fallbackBehavior: i < 3 ? "fail-open" : "fail-closed" as any,
                    },
                    compliance: {
                        standards: complianceStandards,
                        auditingEnabled: i >= 1,
                        retentionDays: 30 * Math.pow(2, i),
                        encryptionRequired: i >= 2,
                        dataClassification: {
                            enabled: i >= 1,
                            levels: ["public", "internal", "confidential", "secret"].slice(0, i + 1) as any,
                        },
                    },
                    metrics: {
                        threatDetectionRate: 0.8 + (0.04 * i),
                        falsePositiveRate: 0.2 / (i + 1),
                        validationLatency: 500 / (i + 1),
                        complianceScore: 0.6 + (0.08 * i),
                        incidentResponseTime: 3600 / (i + 1),
                    },
                },
            });
        });
    }
}

// Export factory instance
export const securityValidatorFixtureFactory = new SecurityValidatorFixtureFactory();

// Export pre-built fixtures
export const securityValidatorFixtures = {
    minimal: securityValidatorFixtureFactory.createMinimal(),
    complete: securityValidatorFixtureFactory.createComplete(),
    withDefaults: securityValidatorFixtureFactory.createWithDefaults(),
    variants: {
        zeroTrust: securityValidatorFixtureFactory.createVariant("zeroTrust"),
        hipaaCompliant: securityValidatorFixtureFactory.createVariant("hipaaCompliant"),
        highThreatEnvironment: securityValidatorFixtureFactory.createVariant("highThreatEnvironment"),
        developmentEnvironment: securityValidatorFixtureFactory.createVariant("developmentEnvironment"),
    },
    incidents: {
        dataBreach: securityValidatorFixtureFactory.createIncidentScenario("data_breach"),
        ddosAttack: securityValidatorFixtureFactory.createIncidentScenario("ddos_attack"),
        insiderThreat: securityValidatorFixtureFactory.createIncidentScenario("insider_threat"),
        zeroDay: securityValidatorFixtureFactory.createIncidentScenario("zero_day"),
    },
    evolution: {
        basic: securityValidatorFixtureFactory.createEvolutionPath(1)[0],
        managed: securityValidatorFixtureFactory.createEvolutionPath(2)[1],
        defined: securityValidatorFixtureFactory.createEvolutionPath(3)[2],
        quantified: securityValidatorFixtureFactory.createEvolutionPath(4)[3],
        optimizing: securityValidatorFixtureFactory.createEvolutionPath(5)[4],
    },
};
