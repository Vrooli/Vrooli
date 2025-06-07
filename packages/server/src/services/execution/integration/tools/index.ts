/**
 * Tool exports and registration for execution architecture
 */

export * from "./monitoringTools.js";
export * from "./monitoringUtils.js";
export * from "./resilienceTools.js";
export * from "./securityTools.js";

import { type Logger } from "winston";
import { type Tool } from "../../../mcp/types.js";
import { MonitoringTools } from "./monitoringTools.js";
import { ResilienceTools } from "./resilienceTools.js";
import { SecurityTools } from "./securityTools.js";
import { type SessionUser } from "@vrooli/shared";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import { type RollingHistory } from "../../cross-cutting/monitoring/index.js";

/**
 * Monitoring tool definitions for MCP registry
 */
export const MONITORING_TOOL_DEFINITIONS: Tool[] = [
    {
        name: "query_metrics",
        description: "Query performance metrics from telemetry data",
        inputSchema: {
            type: "object",
            properties: {
                timeRange: {
                    type: "object",
                    properties: {
                        start: { type: "string", format: "date-time", description: "Start time (ISO 8601)" },
                        end: { type: "string", format: "date-time", description: "End time (ISO 8601)" },
                        duration: { type: "number", description: "Duration in milliseconds from now" },
                    },
                },
                eventTypes: {
                    type: "array",
                    items: { type: "string" },
                    description: "Event types to filter",
                },
                components: {
                    type: "array", 
                    items: { type: "string" },
                    description: "Components to filter",
                },
                tiers: {
                    type: "array",
                    items: { 
                        type: "string",
                        enum: ["tier1", "tier2", "tier3"],
                    },
                    description: "Tiers to filter",
                },
                limit: {
                    type: "number",
                    description: "Maximum number of results",
                },
                aggregationWindow: {
                    type: "number",
                    description: "Aggregation window in milliseconds for time-series data",
                },
            },
        },
        annotations: {
            title: "Query Metrics",
            readOnlyHint: true,
        },
    },
    {
        name: "analyze_history",
        description: "Analyze execution history for patterns and insights",
        inputSchema: {
            type: "object",
            properties: {
                patterns: {
                    type: "object",
                    properties: {
                        type: {
                            type: "string",
                            enum: ["bottleneck", "error_cluster", "resource_spike", "strategy_effectiveness"],
                            description: "Pattern type to detect",
                        },
                        threshold: {
                            type: "number",
                            description: "Detection threshold",
                        },
                    },
                },
                timeWindow: {
                    type: "number",
                    description: "Time window in milliseconds",
                },
                minOccurrence: {
                    type: "number",
                    description: "Minimum occurrences for pattern detection",
                },
            },
        },
        annotations: {
            title: "Analyze History",
            readOnlyHint: true,
        },
    },
    {
        name: "aggregate_data",
        description: "Perform statistical aggregation on metrics",
        inputSchema: {
            type: "object",
            properties: {
                operation: {
                    type: "string",
                    enum: ["sum", "avg", "min", "max", "count", "percentile"],
                    description: "Aggregation operation",
                },
                field: {
                    type: "string",
                    description: "Field path to aggregate (e.g., 'data.duration')",
                },
                groupBy: {
                    type: "string",
                    description: "Field to group results by",
                },
                bucketSize: {
                    type: "number",
                    description: "Time bucket size in milliseconds",
                },
                percentile: {
                    type: "number",
                    description: "Percentile value (0-100) if operation is percentile",
                },
                filter: {
                    type: "object",
                    properties: {
                        eventType: { type: "string" },
                        component: { type: "string" },
                        tier: { type: "string" },
                        timeRange: {
                            type: "object",
                            properties: {
                                start: { type: "string", format: "date-time" },
                                end: { type: "string", format: "date-time" },
                            },
                        },
                    },
                },
            },
            required: ["operation", "field"],
        },
        annotations: {
            title: "Aggregate Data",
            readOnlyHint: true,
        },
    },
    {
        name: "publish_report",
        description: "Publish monitoring insights and reports",
        inputSchema: {
            type: "object",
            properties: {
                type: {
                    type: "string",
                    enum: ["performance", "health", "slo", "custom"],
                    description: "Report type",
                },
                data: {
                    type: "object",
                    description: "Report data",
                },
                severity: {
                    type: "string",
                    enum: ["info", "warning", "error", "critical"],
                    description: "Severity level",
                },
                recipients: {
                    type: "array",
                    items: { type: "string" },
                    description: "Report recipients",
                },
                tags: {
                    type: "array",
                    items: { type: "string" },
                    description: "Tags for categorization",
                },
            },
            required: ["type", "data"],
        },
        annotations: {
            title: "Publish Report",
        },
    },
    {
        name: "detect_anomalies",
        description: "Identify unusual patterns in metrics",
        inputSchema: {
            type: "object",
            properties: {
                metric: {
                    type: "string",
                    description: "Metric path to analyze (e.g., 'data.credits')",
                },
                method: {
                    type: "string",
                    enum: ["zscore", "mad", "isolation_forest", "percentile"],
                    description: "Detection method",
                },
                sensitivity: {
                    type: "number",
                    minimum: 0,
                    maximum: 1,
                    description: "Sensitivity (0-1, where 1 is most sensitive)",
                },
                baselineWindow: {
                    type: "number",
                    description: "Baseline window in milliseconds",
                },
                includeContext: {
                    type: "boolean",
                    description: "Include contextual information",
                },
            },
            required: ["metric"],
        },
        annotations: {
            title: "Detect Anomalies",
            readOnlyHint: true,
        },
    },
    {
        name: "calculate_slo",
        description: "Calculate service level objectives and compliance",
        inputSchema: {
            type: "object",
            properties: {
                slo: {
                    type: "object",
                    properties: {
                        name: {
                            type: "string",
                            description: "SLO name",
                        },
                        metric: {
                            type: "string",
                            description: "Metric path to evaluate",
                        },
                        target: {
                            type: "number",
                            description: "Target value",
                        },
                        comparison: {
                            type: "string",
                            enum: ["gte", "lte", "eq"],
                            description: "Comparison operator",
                        },
                    },
                    required: ["name", "metric", "target", "comparison"],
                },
                timeWindow: {
                    type: "number",
                    description: "Time window in milliseconds",
                },
                breakdown: {
                    type: "boolean",
                    description: "Include breakdown by component",
                },
            },
            required: ["slo"],
        },
        annotations: {
            title: "Calculate SLO",
            readOnlyHint: true,
        },
    },
];

/**
 * Resilience tool definitions for MCP registry
 */
export const RESILIENCE_TOOL_DEFINITIONS: Tool[] = [
    {
        name: "classify_error",
        description: "Classify errors for intelligent recovery strategy selection",
        inputSchema: {
            type: "object",
            properties: {
                error: {
                    type: "object",
                    properties: {
                        message: { type: "string", description: "Error message" },
                        type: { type: "string", description: "Error type" },
                        code: { type: "string", description: "Error code" },
                        component: { type: "string", description: "Component where error occurred" },
                        tier: { type: "string", enum: ["tier1", "tier2", "tier3"], description: "Architecture tier" },
                        context: { type: "object", description: "Additional error context" },
                    },
                    required: ["message", "component"],
                },
                includePatterns: { type: "boolean", description: "Include historical pattern analysis" },
                historicalWindow: { type: "number", description: "Historical analysis window in milliseconds" },
            },
            required: ["error"],
        },
        annotations: {
            title: "Classify Error",
        },
    },
    {
        name: "select_recovery_strategy",
        description: "Select optimal recovery strategy based on error classification",
        inputSchema: {
            type: "object",
            properties: {
                errorClassification: {
                    type: "object",
                    properties: {
                        category: { type: "string", description: "Error category" },
                        severity: { type: "string", enum: ["low", "medium", "high", "critical"] },
                        recoverability: { type: "string", enum: ["transient", "persistent", "permanent"] },
                        component: { type: "string", description: "Affected component" },
                        tier: { type: "string", enum: ["tier1", "tier2", "tier3"] },
                    },
                    required: ["category", "severity", "recoverability", "component"],
                },
                availableResources: {
                    type: "object",
                    properties: {
                        credits: { type: "number", description: "Available credits" },
                        timeConstraints: { type: "number", description: "Time constraints in milliseconds" },
                        alternativeComponents: { type: "array", items: { type: "string" } },
                    },
                },
                minimumSuccessRate: { type: "number", description: "Minimum acceptable success rate (0-1)" },
            },
            required: ["errorClassification"],
        },
        annotations: {
            title: "Select Recovery Strategy",
        },
    },
    {
        name: "analyze_failure_patterns",
        description: "Analyze failure patterns to identify systemic issues",
        inputSchema: {
            type: "object",
            properties: {
                timeWindow: { type: "number", description: "Analysis time window in milliseconds" },
                components: { type: "array", items: { type: "string" }, description: "Components to analyze" },
                patterns: {
                    type: "array",
                    items: {
                        type: "object",
                        properties: {
                            type: { type: "string", enum: ["cascade", "recurring", "resource_exhaustion", "timeout", "dependency"] },
                            threshold: { type: "number", description: "Detection threshold" },
                            minOccurrences: { type: "number", description: "Minimum occurrences" },
                        },
                        required: ["type"],
                    },
                },
                includeCorrelations: { type: "boolean", description: "Include correlation analysis" },
            },
            required: ["patterns"],
        },
        annotations: {
            title: "Analyze Failure Patterns",
            readOnlyHint: true,
        },
    },
    {
        name: "tune_circuit_breaker",
        description: "Tune circuit breaker parameters for optimal performance",
        inputSchema: {
            type: "object",
            properties: {
                circuitBreaker: {
                    type: "object",
                    properties: {
                        name: { type: "string", description: "Circuit breaker name" },
                        component: { type: "string", description: "Component name" },
                        currentSettings: {
                            type: "object",
                            properties: {
                                failureThreshold: { type: "number", description: "Failure threshold" },
                                timeoutThreshold: { type: "number", description: "Timeout threshold in milliseconds" },
                                recoveryTime: { type: "number", description: "Recovery time in milliseconds" },
                            },
                            required: ["failureThreshold", "timeoutThreshold", "recoveryTime"],
                        },
                    },
                    required: ["name", "component", "currentSettings"],
                },
                historyWindow: { type: "number", description: "Historical analysis window in milliseconds" },
                goals: {
                    type: "object",
                    properties: {
                        minimizeLatency: { type: "boolean" },
                        maximizeAvailability: { type: "boolean" },
                        minimizeResourceUsage: { type: "boolean" },
                        customWeights: { type: "object" },
                    },
                },
            },
            required: ["circuitBreaker", "goals"],
        },
        annotations: {
            title: "Tune Circuit Breaker",
        },
    },
    {
        name: "evaluate_fallback_quality",
        description: "Evaluate fallback strategy quality and effectiveness",
        inputSchema: {
            type: "object",
            properties: {
                fallbackStrategy: {
                    type: "object",
                    properties: {
                        name: { type: "string", description: "Strategy name" },
                        type: { type: "string", enum: ["alternative_component", "degraded_service", "cached_response", "default_value"] },
                        configuration: { type: "object", description: "Strategy configuration" },
                    },
                    required: ["name", "type", "configuration"],
                },
                criteria: {
                    type: "object",
                    properties: {
                        accuracy: { type: "number", description: "Accuracy requirement (0-1)" },
                        latency: { type: "number", description: "Latency requirement in milliseconds" },
                        resourceCost: { type: "number", description: "Resource cost threshold" },
                        userExperience: { type: "string", enum: ["excellent", "good", "acceptable", "poor"] },
                    },
                },
                comparisonWindow: { type: "number", description: "Comparison window in milliseconds" },
            },
            required: ["fallbackStrategy", "criteria"],
        },
        annotations: {
            title: "Evaluate Fallback Quality",
            readOnlyHint: true,
        },
    },
    {
        name: "monitor_system_health",
        description: "Monitor system health with predictive analysis",
        inputSchema: {
            type: "object",
            properties: {
                components: { type: "array", items: { type: "string" }, description: "Components to monitor" },
                metrics: {
                    type: "array",
                    items: {
                        type: "object",
                        properties: {
                            type: { type: "string", enum: ["availability", "latency", "error_rate", "resource_usage", "custom"] },
                            name: { type: "string", description: "Custom metric name" },
                            threshold: { type: "number", description: "Alert threshold" },
                            timeWindow: { type: "number", description: "Metric time window in milliseconds" },
                        },
                        required: ["type"],
                    },
                },
                alerting: {
                    type: "object",
                    properties: {
                        enabled: { type: "boolean" },
                        severity: { type: "string", enum: ["info", "warning", "error", "critical"] },
                        recipients: { type: "array", items: { type: "string" } },
                    },
                },
                includePredictive: { type: "boolean", description: "Include predictive health analysis" },
            },
            required: ["metrics"],
        },
        annotations: {
            title: "Monitor System Health",
            readOnlyHint: true,
        },
    },
];

/**
 * Security tool definitions for MCP registry
 */
export const SECURITY_TOOL_DEFINITIONS: Tool[] = [
    {
        name: "validate_security_context",
        description: "Validate security context for access control and threat prevention",
        inputSchema: {
            type: "object",
            properties: {
                context: {
                    type: "object",
                    properties: {
                        userId: { type: "string", description: "User ID" },
                        sessionId: { type: "string", description: "Session ID" },
                        ipAddress: { type: "string", description: "IP address" },
                        userAgent: { type: "string", description: "User agent" },
                        permissions: { type: "array", items: { type: "string" }, description: "User permissions" },
                        tier: { type: "string", enum: ["tier1", "tier2", "tier3"], description: "Architecture tier" },
                        component: { type: "string", description: "Component being accessed" },
                        action: { type: "string", description: "Action being performed" },
                        resourceId: { type: "string", description: "Resource ID" },
                    },
                    required: ["userId", "permissions", "tier", "component", "action"],
                },
                rules: {
                    type: "object",
                    properties: {
                        requireAuthentication: { type: "boolean" },
                        minimumPermissionLevel: { type: "string" },
                        allowedIpRanges: { type: "array", items: { type: "string" } },
                        maxSessionAge: { type: "number", description: "Maximum session age in milliseconds" },
                        requireMFA: { type: "boolean" },
                    },
                },
                checks: {
                    type: "object",
                    properties: {
                        anomalyDetection: { type: "boolean" },
                        riskAssessment: { type: "boolean" },
                        complianceValidation: { type: "boolean" },
                    },
                },
            },
            required: ["context"],
        },
        annotations: {
            title: "Validate Security Context",
        },
    },
    {
        name: "detect_threats",
        description: "Detect security threats using multiple analysis techniques",
        inputSchema: {
            type: "object",
            properties: {
                sources: {
                    type: "object",
                    properties: {
                        logs: { type: "boolean" },
                        events: { type: "boolean" },
                        userBehavior: { type: "boolean" },
                        networkTraffic: { type: "boolean" },
                    },
                },
                timeWindow: { type: "number", description: "Analysis time window in milliseconds" },
                threatTypes: {
                    type: "array",
                    items: {
                        type: "object",
                        properties: {
                            type: { type: "string", enum: ["injection", "authentication_bypass", "privilege_escalation", "data_exfiltration", "anomalous_behavior", "brute_force", "custom"] },
                            indicators: { type: "array", items: { type: "string" } },
                            severity: { type: "string", enum: ["low", "medium", "high", "critical"] },
                        },
                        required: ["type"],
                    },
                },
                sensitivity: { type: "number", minimum: 0, maximum: 1, description: "Detection sensitivity (0-1)" },
                includeThreatIntel: { type: "boolean", description: "Include threat intelligence enrichment" },
            },
            required: ["sources", "threatTypes"],
        },
        annotations: {
            title: "Detect Threats",
            readOnlyHint: true,
        },
    },
    {
        name: "audit_access_patterns",
        description: "Audit access patterns for security and compliance analysis",
        inputSchema: {
            type: "object",
            properties: {
                scope: {
                    type: "object",
                    properties: {
                        users: { type: "array", items: { type: "string" } },
                        resources: { type: "array", items: { type: "string" } },
                        actions: { type: "array", items: { type: "string" } },
                        timeRange: {
                            type: "object",
                            properties: {
                                start: { type: "string", format: "date-time" },
                                end: { type: "string", format: "date-time" },
                            },
                        },
                    },
                },
                analysis: {
                    type: "object",
                    properties: {
                        type: { type: "string", enum: ["access_frequency", "permission_usage", "resource_access", "temporal_patterns", "privilege_escalation", "all"] },
                        aggregationLevel: { type: "string", enum: ["user", "resource", "action", "time"] },
                    },
                    required: ["type"],
                },
                anomalyDetection: {
                    type: "object",
                    properties: {
                        enabled: { type: "boolean" },
                        baseline: { type: "string", enum: ["historical", "peer_group", "policy"] },
                        threshold: { type: "number" },
                    },
                },
                compliance: {
                    type: "object",
                    properties: {
                        framework: { type: "string", enum: ["SOX", "PCI", "HIPAA", "GDPR", "custom"] },
                        requirements: { type: "array", items: { type: "string" } },
                    },
                },
            },
            required: ["scope", "analysis"],
        },
        annotations: {
            title: "Audit Access Patterns",
            readOnlyHint: true,
        },
    },
    {
        name: "analyze_ai_safety",
        description: "Analyze AI safety and security for responsible AI usage",
        inputSchema: {
            type: "object",
            properties: {
                aiSystem: {
                    type: "object",
                    properties: {
                        model: { type: "string", description: "AI model name" },
                        version: { type: "string", description: "Model version" },
                        provider: { type: "string", description: "AI provider" },
                        usage: { type: "string", enum: ["execution", "coordination", "analysis", "generation"] },
                        tier: { type: "string", enum: ["tier1", "tier2", "tier3"] },
                    },
                    required: ["model", "provider", "usage", "tier"],
                },
                safetyChecks: {
                    type: "array",
                    items: {
                        type: "object",
                        properties: {
                            type: { type: "string", enum: ["prompt_injection", "jailbreaking", "bias_detection", "toxicity", "privacy_leakage", "hallucination", "custom"] },
                            enabled: { type: "boolean" },
                            threshold: { type: "number" },
                        },
                        required: ["type", "enabled"],
                    },
                },
                contentAnalysis: {
                    type: "object",
                    properties: {
                        inputs: { type: "array", items: { type: "string" } },
                        outputs: { type: "array", items: { type: "string" } },
                        context: { type: "object" },
                    },
                },
                behaviorValidation: {
                    type: "object",
                    properties: {
                        consistency: { type: "boolean" },
                        alignment: { type: "boolean" },
                        robustness: { type: "boolean" },
                    },
                },
            },
            required: ["aiSystem", "safetyChecks"],
        },
        annotations: {
            title: "Analyze AI Safety",
            readOnlyHint: true,
        },
    },
    {
        name: "assess_compliance",
        description: "Assess compliance with security frameworks and regulations",
        inputSchema: {
            type: "object",
            properties: {
                framework: {
                    type: "object",
                    properties: {
                        name: { type: "string", enum: ["SOX", "PCI_DSS", "HIPAA", "GDPR", "SOC2", "ISO27001", "custom"] },
                        version: { type: "string" },
                        scope: { type: "array", items: { type: "string" } },
                    },
                    required: ["name"],
                },
                scope: {
                    type: "object",
                    properties: {
                        components: { type: "array", items: { type: "string" } },
                        processes: { type: "array", items: { type: "string" } },
                        dataTypes: { type: "array", items: { type: "string" } },
                        timeRange: {
                            type: "object",
                            properties: {
                                start: { type: "string", format: "date-time" },
                                end: { type: "string", format: "date-time" },
                            },
                        },
                    },
                },
                criteria: {
                    type: "array",
                    items: {
                        type: "object",
                        properties: {
                            controlId: { type: "string" },
                            requirement: { type: "string" },
                            criticality: { type: "string", enum: ["low", "medium", "high", "critical"] },
                            automated: { type: "boolean" },
                        },
                        required: ["controlId", "requirement", "criticality"],
                    },
                },
                evidenceCollection: {
                    type: "object",
                    properties: {
                        logs: { type: "boolean" },
                        configurations: { type: "boolean" },
                        policies: { type: "boolean" },
                        userAccess: { type: "boolean" },
                    },
                },
            },
            required: ["framework", "scope", "criteria"],
        },
        annotations: {
            title: "Assess Compliance",
            readOnlyHint: true,
        },
    },
    {
        name: "investigate_incidents",
        description: "Investigate security incidents with forensic analysis",
        inputSchema: {
            type: "object",
            properties: {
                incident: {
                    type: "object",
                    properties: {
                        id: { type: "string", description: "Incident ID" },
                        type: { type: "string", enum: ["security_breach", "data_loss", "unauthorized_access", "system_compromise", "policy_violation", "unknown"] },
                        severity: { type: "string", enum: ["low", "medium", "high", "critical"] },
                        reportedAt: { type: "string", format: "date-time" },
                        affectedSystems: { type: "array", items: { type: "string" } },
                        initialIndicators: { type: "array", items: { type: "string" } },
                    },
                    required: ["id", "type", "severity", "reportedAt"],
                },
                scope: {
                    type: "object",
                    properties: {
                        timeWindow: { type: "number", description: "Investigation time window in milliseconds" },
                        components: { type: "array", items: { type: "string" } },
                        users: { type: "array", items: { type: "string" } },
                        expandScope: { type: "boolean", description: "Auto-expand scope based on findings" },
                    },
                },
                techniques: {
                    type: "object",
                    properties: {
                        forensicAnalysis: { type: "boolean" },
                        timelineReconstruction: { type: "boolean" },
                        rootCauseAnalysis: { type: "boolean" },
                        impactAssessment: { type: "boolean" },
                        attributionAnalysis: { type: "boolean" },
                    },
                },
                evidencePreservation: {
                    type: "object",
                    properties: {
                        enabled: { type: "boolean" },
                        retention: { type: "number", description: "Retention period in days" },
                        format: { type: "string", enum: ["structured", "raw", "both"] },
                    },
                },
            },
            required: ["incident", "scope", "techniques"],
        },
        annotations: {
            title: "Investigate Incidents",
        },
    },
];

/**
 * Create monitoring tool instances
 */
export function createMonitoringToolInstances(
    user: SessionUser,
    logger: Logger,
    eventBus: EventBus,
    rollingHistory?: RollingHistory,
): Map<string, (params: any) => Promise<any>> {
    const monitoringTools = new MonitoringTools(user, logger, eventBus, rollingHistory);
    
    return new Map([
        ["query_metrics", (params) => monitoringTools.queryMetrics(params)],
        ["analyze_history", (params) => monitoringTools.analyzeHistory(params)],
        ["aggregate_data", (params) => monitoringTools.aggregateData(params)],
        ["publish_report", (params) => monitoringTools.publishReport(params)],
        ["detect_anomalies", (params) => monitoringTools.detectAnomalies(params)],
        ["calculate_slo", (params) => monitoringTools.calculateSLO(params)],
    ]);
}

/**
 * Create resilience tool instances
 */
export function createResilienceToolInstances(
    user: SessionUser,
    logger: Logger,
    eventBus: EventBus,
    rollingHistory?: RollingHistory,
): Map<string, (params: any) => Promise<any>> {
    const resilienceTools = new ResilienceTools(user, logger, eventBus, rollingHistory);
    
    return new Map([
        ["classify_error", (params) => resilienceTools.classifyError(params)],
        ["select_recovery_strategy", (params) => resilienceTools.selectRecoveryStrategy(params)],
        ["analyze_failure_patterns", (params) => resilienceTools.analyzeFailurePatterns(params)],
        ["tune_circuit_breaker", (params) => resilienceTools.tuneCircuitBreaker(params)],
        ["evaluate_fallback_quality", (params) => resilienceTools.evaluateFallbackQuality(params)],
        ["monitor_system_health", (params) => resilienceTools.monitorSystemHealth(params)],
    ]);
}

/**
 * Create security tool instances
 */
export function createSecurityToolInstances(
    user: SessionUser,
    logger: Logger,
    eventBus: EventBus,
    rollingHistory?: RollingHistory,
): Map<string, (params: any) => Promise<any>> {
    const securityTools = new SecurityTools(user, logger, eventBus, rollingHistory);
    
    return new Map([
        ["validate_security_context", (params) => securityTools.validateSecurityContext(params)],
        ["detect_threats", (params) => securityTools.detectThreats(params)],
        ["audit_access_patterns", (params) => securityTools.auditAccessPatterns(params)],
        ["analyze_ai_safety", (params) => securityTools.analyzeAiSafety(params)],
        ["assess_compliance", (params) => securityTools.assessCompliance(params)],
        ["investigate_incidents", (params) => securityTools.investigateIncidents(params)],
    ]);
}

/**
 * Register monitoring tools with the integrated tool registry
 */
export function registerMonitoringTools(
    registry: any, // IntegratedToolRegistry
    user: SessionUser,
    logger: Logger,
    eventBus: EventBus,
    rollingHistory?: RollingHistory,
): void {
    // Register tool definitions
    for (const toolDef of MONITORING_TOOL_DEFINITIONS) {
        registry.registerDynamicTool(toolDef, {
            scope: "global",
        });
    }

    // Create tool instances
    const toolInstances = createMonitoringToolInstances(user, logger, eventBus, rollingHistory);

    // Store tool instances for execution
    // This would typically be done through a more sophisticated mechanism
    // For now, we'll use the registry's internal storage
    (registry as any)._monitoringToolInstances = toolInstances;

    logger.info("[MonitoringTools] Registered monitoring tools", {
        toolCount: MONITORING_TOOL_DEFINITIONS.length,
        tools: MONITORING_TOOL_DEFINITIONS.map(t => t.name),
    });
}

/**
 * Register resilience tools with the integrated tool registry
 */
export function registerResilienceTools(
    registry: any, // IntegratedToolRegistry
    user: SessionUser,
    logger: Logger,
    eventBus: EventBus,
    rollingHistory?: RollingHistory,
): void {
    // Register tool definitions
    for (const toolDef of RESILIENCE_TOOL_DEFINITIONS) {
        registry.registerDynamicTool(toolDef, {
            scope: "global",
        });
    }

    // Create tool instances
    const toolInstances = createResilienceToolInstances(user, logger, eventBus, rollingHistory);

    // Store tool instances for execution
    (registry as any)._resilienceToolInstances = toolInstances;

    logger.info("[ResilienceTools] Registered resilience tools", {
        toolCount: RESILIENCE_TOOL_DEFINITIONS.length,
        tools: RESILIENCE_TOOL_DEFINITIONS.map(t => t.name),
    });
}

/**
 * Register security tools with the integrated tool registry
 */
export function registerSecurityTools(
    registry: any, // IntegratedToolRegistry
    user: SessionUser,
    logger: Logger,
    eventBus: EventBus,
    rollingHistory?: RollingHistory,
): void {
    // Register tool definitions
    for (const toolDef of SECURITY_TOOL_DEFINITIONS) {
        registry.registerDynamicTool(toolDef, {
            scope: "global",
        });
    }

    // Create tool instances
    const toolInstances = createSecurityToolInstances(user, logger, eventBus, rollingHistory);

    // Store tool instances for execution
    (registry as any)._securityToolInstances = toolInstances;

    logger.info("[SecurityTools] Registered security tools", {
        toolCount: SECURITY_TOOL_DEFINITIONS.length,
        tools: SECURITY_TOOL_DEFINITIONS.map(t => t.name),
    });
}

/**
 * Register all emergent intelligence tools
 */
export function registerEmergentIntelligenceTools(
    registry: any, // IntegratedToolRegistry
    user: SessionUser,
    logger: Logger,
    eventBus: EventBus,
    rollingHistory?: RollingHistory,
): void {
    registerMonitoringTools(registry, user, logger, eventBus, rollingHistory);
    registerResilienceTools(registry, user, logger, eventBus, rollingHistory);
    registerSecurityTools(registry, user, logger, eventBus, rollingHistory);

    logger.info("[EmergentIntelligence] Registered all emergent intelligence tools", {
        monitoringTools: MONITORING_TOOL_DEFINITIONS.length,
        resilienceTools: RESILIENCE_TOOL_DEFINITIONS.length,
        securityTools: SECURITY_TOOL_DEFINITIONS.length,
        totalTools: MONITORING_TOOL_DEFINITIONS.length + RESILIENCE_TOOL_DEFINITIONS.length + SECURITY_TOOL_DEFINITIONS.length,
    });
}