/**
 * Security-focused routine fixtures
 * 
 * Routines for HIPAA compliance, API security, data privacy, and threat detection
 */

import type {
    CallDataActionConfigObject,
    CallDataCodeConfigObject,
    FormInputConfigObject,
    FormOutputConfigObject,
    RoutineVersionConfigObject,
} from "@vrooli/shared";
import { InputType, McpToolName, ResourceSubType } from "@vrooli/shared";
import type { RoutineFixture, RoutineFixtureCollection } from "./types.js";

/**
 * HIPAA Compliance Check Routine
 * Used by: HIPAA Compliance Agent
 * Purpose: Scans AI outputs for PHI and generates compliance reports
 */
export const HIPAA_COMPLIANCE_CHECK: RoutineFixture = {
    id: "hipaa_compliance_check",
    name: "HIPAA Compliance Check",
    description: "Scans AI outputs for protected health information and generates compliance reports",
    version: "1.0.0",
    resourceSubType: ResourceSubType.RoutineCode,
    config: {
        __version: "1.0",
        callDataCode: {
            __version: "1.0",
            schema: {
                inputTemplate: {
                    text: "{{input.aiOutput}}",
                    confidenceThreshold: "{{input.confidenceThreshold}}",
                },
                outputMappings: [
                    {
                        schemaIndex: 0,
                        mapping: {
                            phiDetected: "phiDetected",
                            detectedItems: "detectedItems",
                            confidenceScore: "confidenceScore",
                            complianceStatus: "complianceStatus",
                        },
                    },
                ],
            },
        } as CallDataCodeConfigObject,
        formInput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "aiOutput",
                        id: "ai_output",
                        label: "AI Generated Output",
                        type: InputType.Text,
                        isRequired: true,
                        props: {
                            placeholder: "Paste AI output to scan for PHI...",
                        },
                    },
                    {
                        fieldName: "confidenceThreshold",
                        id: "confidence_threshold",
                        label: "Detection Confidence Threshold",
                        type: InputType.Slider,
                        isRequired: false,
                        defaultValue: 0.85,
                        props: {
                            min: 0.5,
                            max: 1.0,
                            step: 0.05,
                        },
                    },
                ],
            },
        } as FormInputConfigObject,
        formOutput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "phiDetected",
                        id: "phi_detected",
                        label: "PHI Detected",
                        type: InputType.Switch,
                        props: {
                            disabled: true,
                        },
                    },
                    {
                        fieldName: "detectedItems",
                        id: "detected_items",
                        label: "Detected PHI Items",
                        type: InputType.JSON,
                        props: {
                            disabled: true,
                        },
                    },
                    {
                        fieldName: "confidenceScore",
                        id: "confidence_score",
                        label: "Detection Confidence",
                        type: InputType.IntegerInput,
                        props: {
                            disabled: true,
                            min: 0,
                            max: 100,
                        },
                    },
                    {
                        fieldName: "complianceStatus",
                        id: "compliance_status",
                        label: "HIPAA Compliance Status",
                        type: InputType.Text,
                        props: {
                            disabled: true,
                        },
                    },
                ],
            },
        } as FormOutputConfigObject,
        executionStrategy: "deterministic" as const,
    } as RoutineVersionConfigObject,
};

/**
 * API Security Scan Routine
 * Used by: API Security Agent
 * Purpose: Scans API logs for security threats and anomalies
 */
export const API_SECURITY_SCAN: RoutineFixture = {
    id: "api_security_scan",
    name: "API Security Scan",
    description: "Analyzes API request logs to detect security threats and anomalies",
    version: "1.0.0",
    resourceSubType: ResourceSubType.RoutineCode,
    config: {
        __version: "1.0",
        callDataCode: {
            __version: "1.0",
            schema: {
                inputTemplate: {
                    logs: "{{input.apiLogs}}",
                    timeRange: "{{input.timeRange}}",
                    knownIPs: "{{input.knownIPs}}",
                },
                outputMappings: [
                    {
                        schemaIndex: 0,
                        mapping: {
                            threats: "threats",
                            anomalies: "anomalies",
                            recommendations: "recommendations",
                            statistics: "statistics",
                        },
                    },
                ],
            },
        } as CallDataCodeConfigObject,
        formInput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "apiLogs",
                        id: "api_logs",
                        label: "API Request Logs",
                        type: InputType.JSON,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "timeRange",
                        id: "time_range",
                        label: "Time Range",
                        type: InputType.Text,
                        isRequired: true,
                        defaultValue: "1h",
                        props: {},
                    },
                    {
                        fieldName: "knownIPs",
                        id: "known_ips",
                        label: "Known Safe IPs",
                        type: InputType.JSON,
                        isRequired: false,
                        defaultValue: [],
                        props: {},
                    },
                ],
            },
        } as FormInputConfigObject,
        formOutput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "threats",
                        id: "threats",
                        label: "Detected Threats",
                        type: InputType.JSON,
                        props: { disabled: true },
                    },
                    {
                        fieldName: "anomalies",
                        id: "anomalies",
                        label: "Anomalies",
                        type: InputType.JSON,
                        props: { disabled: true },
                    },
                    {
                        fieldName: "recommendations",
                        id: "recommendations",
                        label: "Security Recommendations",
                        type: InputType.Text,
                        props: { disabled: true },
                    },
                    {
                        fieldName: "statistics",
                        id: "statistics",
                        label: "Request Statistics",
                        type: InputType.JSON,
                        props: { disabled: true },
                    },
                ],
            },
        } as FormOutputConfigObject,
        executionStrategy: "deterministic" as const,
    } as RoutineVersionConfigObject,
};

/**
 * GDPR Data Audit Routine
 * Used by: Data Privacy Agent
 * Purpose: Audits data processing for GDPR compliance
 */
export const GDPR_DATA_AUDIT: RoutineFixture = {
    id: "gdpr_data_audit",
    name: "GDPR Data Audit",
    description: "Audits data processing activities for GDPR compliance",
    version: "1.0.0",
    resourceSubType: ResourceSubType.RoutineInternalAction,
    config: {
        __version: "1.0",
        callDataAction: {
            __version: "1.0",
            schema: {
                toolName: McpToolName.ResourceManage,
                inputTemplate: JSON.stringify({
                    op: "search",
                    resource_type: "DataProcessingRecord",
                    filters: {
                        processingRecords: "{{input.processingRecords}}",
                        subjectRequests: "{{input.subjectRequests}}",
                        retentionPolicies: "{{input.retentionPolicies}}",
                    },
                    limit: 100,
                }),
                outputMapping: {
                    complianceScore: "complianceScore",
                    violations: "violations",
                    recommendations: "recommendations",
                    auditReport: "auditReport",
                },
            },
        } as CallDataActionConfigObject,
        formInput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "processingRecords",
                        id: "processing_records",
                        label: "Data Processing Records",
                        type: InputType.JSON,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "subjectRequests",
                        id: "subject_requests",
                        label: "Data Subject Requests",
                        type: InputType.JSON,
                        isRequired: false,
                        props: {},
                    },
                    {
                        fieldName: "retentionPolicies",
                        id: "retention_policies",
                        label: "Retention Policies",
                        type: InputType.JSON,
                        isRequired: true,
                        props: {},
                    },
                ],
            },
        } as FormInputConfigObject,
        formOutput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "complianceScore",
                        id: "compliance_score",
                        label: "GDPR Compliance Score",
                        type: InputType.IntegerInput,
                        props: { disabled: true, min: 0, max: 100 },
                    },
                    {
                        fieldName: "violations",
                        id: "violations",
                        label: "Violations Found",
                        type: InputType.JSON,
                        props: { disabled: true },
                    },
                    {
                        fieldName: "recommendations",
                        id: "recommendations",
                        label: "Compliance Recommendations",
                        type: InputType.Text,
                        props: { disabled: true },
                    },
                    {
                        fieldName: "auditReport",
                        id: "audit_report",
                        label: "Full Audit Report",
                        type: InputType.JSON,
                        props: { disabled: true },
                    },
                ],
            },
        } as FormOutputConfigObject,
        executionStrategy: "reasoning" as const,
    } as RoutineVersionConfigObject,
};

/**
 * Trading Pattern Analysis Routine
 * Used by: Trading Fraud Detection Agent
 * Purpose: Analyzes trading patterns for suspicious activity
 */
export const TRADING_PATTERN_ANALYSIS: RoutineFixture = {
    id: "trading_pattern_analysis",
    name: "Trading Pattern Analysis",
    description: "Analyzes trading patterns to detect potential fraud indicators",
    version: "1.0.0",
    resourceSubType: ResourceSubType.RoutineInternalAction,
    config: {
        __version: "1.0",
        callDataAction: {
            __version: "1.0",
            schema: {
                toolName: McpToolName.RunRoutine,
                inputTemplate: JSON.stringify({
                    routineId: "pattern_analysis_routine",
                    inputs: {
                        data: "{{input.tradingData}}",
                        timeWindow: "{{input.timeWindow}}",
                    },
                    mode: "inline",
                }),
                outputMapping: {
                    suspiciousPatterns: "patterns",
                    riskScore: "riskScore",
                    flaggedTransactions: "flaggedTransactions",
                    recommendations: "recommendations",
                },
            },
        } as CallDataActionConfigObject,
        formInput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "tradingData",
                        id: "trading_data",
                        label: "Trading Data",
                        type: InputType.JSON,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "timeWindow",
                        id: "time_window",
                        label: "Analysis Time Window",
                        type: InputType.Dropdown,
                        isRequired: true,
                        defaultValue: "1h",
                        props: {
                            options: [
                                { label: "5 minutes", value: "5m" },
                                { label: "1 hour", value: "1h" },
                                { label: "24 hours", value: "24h" },
                                { label: "7 days", value: "7d" },
                            ],
                        },
                    },
                ],
            },
        } as FormInputConfigObject,
        formOutput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "suspiciousPatterns",
                        id: "suspicious_patterns",
                        label: "Detected Patterns",
                        type: InputType.JSON,
                        props: { disabled: true },
                    },
                    {
                        fieldName: "riskScore",
                        id: "risk_score",
                        label: "Risk Score",
                        type: InputType.IntegerInput,
                        props: { disabled: true, min: 0, max: 100 },
                    },
                    {
                        fieldName: "flaggedTransactions",
                        id: "flagged_transactions",
                        label: "Flagged Transactions",
                        type: InputType.JSON,
                        props: { disabled: true },
                    },
                    {
                        fieldName: "recommendations",
                        id: "recommendations",
                        label: "Recommendations",
                        type: InputType.Text,
                        props: { disabled: true },
                    },
                ],
            },
        } as FormOutputConfigObject,
        executionStrategy: "deterministic" as const,
    } as RoutineVersionConfigObject,
};

export const SECURITY_ROUTINES: RoutineFixtureCollection<
    | "HIPAA_COMPLIANCE_CHECK"
    | "API_SECURITY_SCAN" 
    | "GDPR_DATA_AUDIT"
    | "TRADING_PATTERN_ANALYSIS"
> = {
    HIPAA_COMPLIANCE_CHECK,
    API_SECURITY_SCAN,
    GDPR_DATA_AUDIT,
    TRADING_PATTERN_ANALYSIS,
} as const;
