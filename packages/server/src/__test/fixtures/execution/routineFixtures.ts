/**
 * Test fixtures for execution routines
 * 
 * These fixtures demonstrate both sequential (native) and BPMN-based routines
 * that correspond to each agent example from the documentation.
 * 
 * Each routine represents a task that an agent might analyze or optimize.
 */

import { McpToolName, ResourceSubType } from "../../../../../shared/src/api/types.js";
import { InputType } from "../../../../../shared/src/consts/model.js";
import { BotStyle } from "../../../../../shared/src/run/types.js";
import type { 
    RoutineVersionConfigObject,
    CallDataActionConfigObject,
    CallDataApiConfigObject,
    CallDataCodeConfigObject,
    CallDataGenerateConfigObject,
    CallDataWebConfigObject,
    FormInputConfigObject,
    FormOutputConfigObject,
    GraphBpmnConfigObject,
} from "../../../../../shared/src/shape/configs/routine.js";

/**
 * Sequential routine configurations
 * These are simple, single-step routines using the native JSON format
 */
export const SEQUENTIAL_ROUTINES = {
    /**
     * HIPAA Compliance Check Routine
     * Used by: HIPAA Compliance Agent
     * Purpose: Scans AI outputs for PHI and generates compliance reports
     */
    HIPAA_COMPLIANCE_CHECK: {
        id: "hipaa_compliance_check",
        name: "HIPAA Compliance Check",
        description: "Scans AI outputs for protected health information and generates compliance reports",
        version: "1.0.0",
        resourceSubType: ResourceSubType.RoutineInternalAction,
        config: {
            __version: "1.0",
            callDataAction: {
                __version: "1.0",
                schema: {
                    toolName: McpToolName.TextAnalyze,
                    inputTemplate: JSON.stringify({
                        text: "{{input.aiOutput}}",
                        analysisType: "phi_detection",
                        patterns: [
                            "social_security_numbers",
                            "medical_record_numbers",
                            "patient_names",
                            "medical_histories",
                            "treatment_details"
                        ],
                        confidenceThreshold: "{{input.confidenceThreshold}}"
                    }),
                    outputMapping: {
                        phiDetected: "phiDetected",
                        detectedItems: "detectedItems",
                        confidenceScore: "confidenceScore",
                        complianceStatus: "complianceStatus"
                    }
                }
            } as CallDataActionConfigObject,
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
                                placeholder: "Paste AI output to scan for PHI..."
                            }
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
                                step: 0.05
                            }
                        }
                    ]
                }
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
                                disabled: true
                            }
                        },
                        {
                            fieldName: "detectedItems",
                            id: "detected_items",
                            label: "Detected PHI Items",
                            type: InputType.JSON,
                            props: {
                                disabled: true
                            }
                        },
                        {
                            fieldName: "confidenceScore",
                            id: "confidence_score",
                            label: "Detection Confidence",
                            type: InputType.IntegerInput,
                            props: {
                                disabled: true,
                                min: 0,
                                max: 100
                            }
                        },
                        {
                            fieldName: "complianceStatus",
                            id: "compliance_status",
                            label: "HIPAA Compliance Status",
                            type: InputType.Text,
                            props: {
                                disabled: true
                            }
                        }
                    ]
                }
            } as FormOutputConfigObject,
            executionStrategy: "deterministic" as const,
        } as RoutineVersionConfigObject,
    },

    /**
     * Medical Diagnosis Validation Routine
     * Used by: Medical Safety Agent
     * Purpose: Validates AI-generated diagnoses against clinical guidelines
     */
    MEDICAL_DIAGNOSIS_VALIDATION: {
        id: "medical_diagnosis_validation",
        name: "Medical Diagnosis Validation",
        description: "Validates AI-generated medical diagnoses against clinical guidelines and checks for bias",
        version: "1.0.0",
        resourceSubType: ResourceSubType.RoutineGenerate,
        config: {
            __version: "1.0",
            callDataGenerate: {
                __version: "1.0",
                schema: {
                    botStyle: BotStyle.Academic,
                    maxTokens: 1000,
                    model: null,
                    prompt: "Validate the following medical diagnosis:\n\nDiagnosis: {{input.diagnosis}}\nPatient Demographics: {{input.demographics}}\nSymptoms: {{input.symptoms}}\n\nCheck for:\n1. Clinical accuracy against latest guidelines\n2. Potential demographic bias\n3. Missing critical symptoms\n4. Safety concerns\n\nProvide a detailed validation report.",
                    respondingBot: null
                }
            } as CallDataGenerateConfigObject,
            formInput: {
                __version: "1.0",
                schema: {
                    containers: [],
                    elements: [
                        {
                            fieldName: "diagnosis",
                            id: "diagnosis",
                            label: "AI Diagnosis",
                            type: InputType.Text,
                            isRequired: true
                        },
                        {
                            fieldName: "demographics",
                            id: "demographics",
                            label: "Patient Demographics",
                            type: InputType.JSON,
                            isRequired: true
                        },
                        {
                            fieldName: "symptoms",
                            id: "symptoms",
                            label: "Symptoms",
                            type: InputType.Text,
                            isRequired: true
                        }
                    ]
                }
            } as FormInputConfigObject,
            formOutput: {
                __version: "1.0",
                schema: {
                    containers: [],
                    elements: [
                        {
                            fieldName: "response",
                            id: "response",
                            label: "Validation Report",
                            type: InputType.Text,
                            props: {
                                disabled: true
                            }
                        }
                    ]
                }
            } as FormOutputConfigObject,
            executionStrategy: "reasoning" as const,
        } as RoutineVersionConfigObject,
    },

    /**
     * Trading Pattern Analysis Routine
     * Used by: Trading Fraud Detection Agent
     * Purpose: Analyzes trading patterns for suspicious activity
     */
    TRADING_PATTERN_ANALYSIS: {
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
                    toolName: McpToolName.DataAnalyze,
                    inputTemplate: JSON.stringify({
                        data: "{{input.tradingData}}",
                        analysisType: "pattern_detection",
                        patterns: [
                            "wash_trading",
                            "pump_and_dump",
                            "insider_trading",
                            "spoofing",
                            "front_running"
                        ],
                        timeWindow: "{{input.timeWindow}}",
                        thresholds: {
                            volumeSpike: 3.0,
                            priceMovement: 0.05,
                            orderCancellation: 0.8
                        }
                    }),
                    outputMapping: {
                        suspiciousPatterns: "patterns",
                        riskScore: "riskScore",
                        flaggedTransactions: "flaggedTransactions",
                        recommendations: "recommendations"
                    }
                }
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
                            props: {
                                placeholder: "Paste trading data JSON..."
                            }
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
                                    { label: "7 days", value: "7d" }
                                ]
                            }
                        }
                    ]
                }
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
                            props: { disabled: true }
                        },
                        {
                            fieldName: "riskScore",
                            id: "risk_score",
                            label: "Risk Score",
                            type: InputType.IntegerInput,
                            props: { disabled: true, min: 0, max: 100 }
                        },
                        {
                            fieldName: "flaggedTransactions",
                            id: "flagged_transactions",
                            label: "Flagged Transactions",
                            type: InputType.JSON,
                            props: { disabled: true }
                        },
                        {
                            fieldName: "recommendations",
                            id: "recommendations",
                            label: "Recommendations",
                            type: InputType.Text,
                            props: { disabled: true }
                        }
                    ]
                }
            } as FormOutputConfigObject,
            executionStrategy: "deterministic" as const,
        } as RoutineVersionConfigObject,
    },

    /**
     * API Security Scan Routine
     * Used by: API Security Agent
     * Purpose: Scans API logs for security threats and anomalies
     */
    API_SECURITY_SCAN: {
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
                        knownIPs: "{{input.knownIPs}}"
                    },
                    outputMappings: [
                        {
                            schemaIndex: 0,
                            mapping: {
                                threats: "threats",
                                anomalies: "anomalies",
                                recommendations: "recommendations",
                                statistics: "statistics"
                            }
                        }
                    ]
                }
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
                            isRequired: true
                        },
                        {
                            fieldName: "timeRange",
                            id: "time_range",
                            label: "Time Range",
                            type: InputType.Text,
                            isRequired: true,
                            defaultValue: "1h"
                        },
                        {
                            fieldName: "knownIPs",
                            id: "known_ips",
                            label: "Known Safe IPs",
                            type: InputType.JSON,
                            isRequired: false,
                            defaultValue: []
                        }
                    ]
                }
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
                            props: { disabled: true }
                        },
                        {
                            fieldName: "anomalies",
                            id: "anomalies",
                            label: "Anomalies",
                            type: InputType.JSON,
                            props: { disabled: true }
                        },
                        {
                            fieldName: "recommendations",
                            id: "recommendations",
                            label: "Security Recommendations",
                            type: InputType.Text,
                            props: { disabled: true }
                        },
                        {
                            fieldName: "statistics",
                            id: "statistics",
                            label: "Request Statistics",
                            type: InputType.JSON,
                            props: { disabled: true }
                        }
                    ]
                }
            } as FormOutputConfigObject,
            executionStrategy: "deterministic" as const,
        } as RoutineVersionConfigObject,
    },

    /**
     * GDPR Data Audit Routine
     * Used by: Data Privacy Agent
     * Purpose: Audits data processing for GDPR compliance
     */
    GDPR_DATA_AUDIT: {
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
                    toolName: McpToolName.DataAudit,
                    inputTemplate: JSON.stringify({
                        dataProcessingRecords: "{{input.processingRecords}}",
                        dataSubjectRequests: "{{input.subjectRequests}}",
                        retentionPolicies: "{{input.retentionPolicies}}",
                        auditType: "gdpr_compliance",
                        checkpoints: [
                            "consent_verification",
                            "data_minimization",
                            "retention_compliance",
                            "cross_border_transfers",
                            "subject_rights"
                        ]
                    }),
                    outputMapping: {
                        complianceScore: "complianceScore",
                        violations: "violations",
                        recommendations: "recommendations",
                        auditReport: "auditReport"
                    }
                }
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
                            isRequired: true
                        },
                        {
                            fieldName: "subjectRequests",
                            id: "subject_requests",
                            label: "Data Subject Requests",
                            type: InputType.JSON,
                            isRequired: false
                        },
                        {
                            fieldName: "retentionPolicies",
                            id: "retention_policies",
                            label: "Retention Policies",
                            type: InputType.JSON,
                            isRequired: true
                        }
                    ]
                }
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
                            props: { disabled: true, min: 0, max: 100 }
                        },
                        {
                            fieldName: "violations",
                            id: "violations",
                            label: "Violations Found",
                            type: InputType.JSON,
                            props: { disabled: true }
                        },
                        {
                            fieldName: "recommendations",
                            id: "recommendations",
                            label: "Compliance Recommendations",
                            type: InputType.Text,
                            props: { disabled: true }
                        },
                        {
                            fieldName: "auditReport",
                            id: "audit_report",
                            label: "Full Audit Report",
                            type: InputType.JSON,
                            props: { disabled: true }
                        }
                    ]
                }
            } as FormOutputConfigObject,
            executionStrategy: "reasoning" as const,
        } as RoutineVersionConfigObject,
    },

    /**
     * System Failure Analysis Routine
     * Used by: Pattern Learning Agent (Resilience)
     * Purpose: Analyzes system failures to identify patterns
     */
    SYSTEM_FAILURE_ANALYSIS: {
        id: "system_failure_analysis",
        name: "System Failure Analysis",
        description: "Analyzes system failure logs to identify patterns and correlations",
        version: "1.0.0",
        resourceSubType: ResourceSubType.RoutineInternalAction,
        config: {
            __version: "1.0",
            callDataAction: {
                __version: "1.0",
                schema: {
                    toolName: McpToolName.LogAnalyze,
                    inputTemplate: JSON.stringify({
                        logs: "{{input.failureLogs}}",
                        analysisType: "failure_pattern",
                        patterns: [
                            "cascading_failures",
                            "timeout_chains",
                            "resource_exhaustion",
                            "dependency_failures"
                        ],
                        timeWindow: "{{input.analysisWindow}}",
                        correlationThreshold: 0.7
                    }),
                    outputMapping: {
                        patterns: "detectedPatterns",
                        correlations: "failureCorrelations",
                        rootCauses: "likelyRootCauses",
                        predictions: "failurePredictions"
                    }
                }
            } as CallDataActionConfigObject,
            formInput: {
                __version: "1.0",
                schema: {
                    containers: [],
                    elements: [
                        {
                            fieldName: "failureLogs",
                            id: "failure_logs",
                            label: "System Failure Logs",
                            type: InputType.JSON,
                            isRequired: true
                        },
                        {
                            fieldName: "analysisWindow",
                            id: "analysis_window",
                            label: "Analysis Window",
                            type: InputType.Text,
                            isRequired: true,
                            defaultValue: "24h"
                        }
                    ]
                }
            } as FormInputConfigObject,
            formOutput: {
                __version: "1.0",
                schema: {
                    containers: [],
                    elements: [
                        {
                            fieldName: "patterns",
                            id: "patterns",
                            label: "Detected Failure Patterns",
                            type: InputType.JSON,
                            props: { disabled: true }
                        },
                        {
                            fieldName: "correlations",
                            id: "correlations",
                            label: "Failure Correlations",
                            type: InputType.JSON,
                            props: { disabled: true }
                        },
                        {
                            fieldName: "rootCauses",
                            id: "root_causes",
                            label: "Likely Root Causes",
                            type: InputType.JSON,
                            props: { disabled: true }
                        },
                        {
                            fieldName: "predictions",
                            id: "predictions",
                            label: "Failure Predictions",
                            type: InputType.JSON,
                            props: { disabled: true }
                        }
                    ]
                }
            } as FormOutputConfigObject,
            executionStrategy: "reasoning" as const,
        } as RoutineVersionConfigObject,
    },

    /**
     * Performance Bottleneck Detection Routine
     * Used by: Routine Performance Agent
     * Purpose: Identifies performance bottlenecks in routine execution
     */
    PERFORMANCE_BOTTLENECK_DETECTION: {
        id: "performance_bottleneck_detection",
        name: "Performance Bottleneck Detection",
        description: "Analyzes routine execution metrics to identify performance bottlenecks",
        version: "1.0.0",
        resourceSubType: ResourceSubType.RoutineCode,
        config: {
            __version: "1.0",
            callDataCode: {
                __version: "1.0",
                schema: {
                    inputTemplate: {
                        executionMetrics: "{{input.metrics}}",
                        routineConfig: "{{input.config}}",
                        threshold: "{{input.threshold}}"
                    },
                    outputMappings: [
                        {
                            schemaIndex: 0,
                            mapping: {
                                bottlenecks: "bottlenecks",
                                optimizationOpportunities: "opportunities",
                                performanceScore: "score",
                                recommendations: "recommendations"
                            }
                        }
                    ]
                }
            } as CallDataCodeConfigObject,
            formInput: {
                __version: "1.0",
                schema: {
                    containers: [],
                    elements: [
                        {
                            fieldName: "metrics",
                            id: "metrics",
                            label: "Execution Metrics",
                            type: InputType.JSON,
                            isRequired: true
                        },
                        {
                            fieldName: "config",
                            id: "config",
                            label: "Routine Configuration",
                            type: InputType.JSON,
                            isRequired: true
                        },
                        {
                            fieldName: "threshold",
                            id: "threshold",
                            label: "Performance Threshold (ms)",
                            type: InputType.IntegerInput,
                            isRequired: true,
                            defaultValue: 1000
                        }
                    ]
                }
            } as FormInputConfigObject,
            formOutput: {
                __version: "1.0",
                schema: {
                    containers: [],
                    elements: [
                        {
                            fieldName: "bottlenecks",
                            id: "bottlenecks",
                            label: "Detected Bottlenecks",
                            type: InputType.JSON,
                            props: { disabled: true }
                        },
                        {
                            fieldName: "optimizationOpportunities",
                            id: "optimization_opportunities",
                            label: "Optimization Opportunities",
                            type: InputType.JSON,
                            props: { disabled: true }
                        },
                        {
                            fieldName: "performanceScore",
                            id: "performance_score",
                            label: "Performance Score",
                            type: InputType.IntegerInput,
                            props: { disabled: true, min: 0, max: 100 }
                        },
                        {
                            fieldName: "recommendations",
                            id: "recommendations",
                            label: "Recommendations",
                            type: InputType.Text,
                            props: { disabled: true }
                        }
                    ]
                }
            } as FormOutputConfigObject,
            executionStrategy: "deterministic" as const,
        } as RoutineVersionConfigObject,
    },

    /**
     * Cost Analysis Routine
     * Used by: Cost Optimization Agent
     * Purpose: Analyzes operational costs and identifies savings opportunities
     */
    COST_ANALYSIS: {
        id: "cost_analysis",
        name: "Operational Cost Analysis",
        description: "Analyzes operational costs to identify reduction opportunities",
        version: "1.0.0",
        resourceSubType: ResourceSubType.RoutineInternalAction,
        config: {
            __version: "1.0",
            callDataAction: {
                __version: "1.0",
                schema: {
                    toolName: McpToolName.CostAnalyze,
                    inputTemplate: JSON.stringify({
                        billingData: "{{input.billingData}}",
                        resourceUsage: "{{input.resourceUsage}}",
                        timeframe: "{{input.timeframe}}",
                        analysisType: "optimization",
                        categories: [
                            "compute",
                            "storage",
                            "network",
                            "api_calls",
                            "model_inference"
                        ]
                    }),
                    outputMapping: {
                        currentCost: "totalCost",
                        breakdown: "costBreakdown",
                        savingsOpportunities: "opportunities",
                        recommendations: "recommendations"
                    }
                }
            } as CallDataActionConfigObject,
            formInput: {
                __version: "1.0",
                schema: {
                    containers: [],
                    elements: [
                        {
                            fieldName: "billingData",
                            id: "billing_data",
                            label: "Billing Data",
                            type: InputType.JSON,
                            isRequired: true
                        },
                        {
                            fieldName: "resourceUsage",
                            id: "resource_usage",
                            label: "Resource Usage Data",
                            type: InputType.JSON,
                            isRequired: true
                        },
                        {
                            fieldName: "timeframe",
                            id: "timeframe",
                            label: "Analysis Timeframe",
                            type: InputType.Dropdown,
                            isRequired: true,
                            defaultValue: "30d",
                            props: {
                                options: [
                                    { label: "Last 7 days", value: "7d" },
                                    { label: "Last 30 days", value: "30d" },
                                    { label: "Last 90 days", value: "90d" }
                                ]
                            }
                        }
                    ]
                }
            } as FormInputConfigObject,
            formOutput: {
                __version: "1.0",
                schema: {
                    containers: [],
                    elements: [
                        {
                            fieldName: "currentCost",
                            id: "current_cost",
                            label: "Total Cost",
                            type: InputType.IntegerInput,
                            props: { disabled: true }
                        },
                        {
                            fieldName: "breakdown",
                            id: "breakdown",
                            label: "Cost Breakdown",
                            type: InputType.JSON,
                            props: { disabled: true }
                        },
                        {
                            fieldName: "savingsOpportunities",
                            id: "savings_opportunities",
                            label: "Savings Opportunities",
                            type: InputType.JSON,
                            props: { disabled: true }
                        },
                        {
                            fieldName: "recommendations",
                            id: "recommendations",
                            label: "Cost Reduction Recommendations",
                            type: InputType.Text,
                            props: { disabled: true }
                        }
                    ]
                }
            } as FormOutputConfigObject,
            executionStrategy: "reasoning" as const,
        } as RoutineVersionConfigObject,
    },

    /**
     * Output Quality Assessment Routine
     * Used by: Output Quality Monitor
     * Purpose: Assesses the quality of generated outputs
     */
    OUTPUT_QUALITY_ASSESSMENT: {
        id: "output_quality_assessment",
        name: "Output Quality Assessment",
        description: "Evaluates output quality including accuracy, completeness, and bias detection",
        version: "1.0.0",
        resourceSubType: ResourceSubType.RoutineGenerate,
        config: {
            __version: "1.0",
            callDataGenerate: {
                __version: "1.0",
                schema: {
                    botStyle: BotStyle.Analytical,
                    maxTokens: 800,
                    model: null,
                    prompt: "Assess the quality of the following output:\n\nOutput: {{input.output}}\nExpected Criteria: {{input.criteria}}\n\nEvaluate:\n1. Accuracy score (0-100)\n2. Completeness score (0-100)\n3. Consistency score (0-100)\n4. Bias detection (demographic, cultural, linguistic)\n5. Overall quality score\n\nProvide detailed assessment with specific examples.",
                    respondingBot: null
                }
            } as CallDataGenerateConfigObject,
            formInput: {
                __version: "1.0",
                schema: {
                    containers: [],
                    elements: [
                        {
                            fieldName: "output",
                            id: "output",
                            label: "Output to Assess",
                            type: InputType.Text,
                            isRequired: true
                        },
                        {
                            fieldName: "criteria",
                            id: "criteria",
                            label: "Quality Criteria",
                            type: InputType.JSON,
                            isRequired: true,
                            defaultValue: {
                                accuracyTarget: 0.85,
                                completenessTarget: 0.90,
                                biasThreshold: 0.15
                            }
                        }
                    ]
                }
            } as FormInputConfigObject,
            formOutput: {
                __version: "1.0",
                schema: {
                    containers: [],
                    elements: [
                        {
                            fieldName: "response",
                            id: "response",
                            label: "Quality Assessment Report",
                            type: InputType.Text,
                            props: { disabled: true }
                        }
                    ]
                }
            } as FormOutputConfigObject,
            executionStrategy: "reasoning" as const,
        } as RoutineVersionConfigObject,
    },

    /**
     * System Health Check Routine
     * Used by: System Health Monitor
     * Purpose: Monitors overall system health and predicts issues
     */
    SYSTEM_HEALTH_CHECK: {
        id: "system_health_check",
        name: "System Health Check",
        description: "Comprehensive system health monitoring and anomaly detection",
        version: "1.0.0",
        resourceSubType: ResourceSubType.RoutineWeb,
        config: {
            __version: "1.0",
            callDataWeb: {
                __version: "1.0",
                schema: {
                    queryTemplate: "system monitoring {{input.systemName}} health metrics best practices",
                    searchEngine: "google",
                    maxResults: 5,
                    outputMapping: {
                        bestPractices: "results[*].snippet",
                        resources: "results[*].link"
                    }
                }
            } as CallDataWebConfigObject,
            formInput: {
                __version: "1.0",
                schema: {
                    containers: [],
                    elements: [
                        {
                            fieldName: "systemName",
                            id: "system_name",
                            label: "System Name",
                            type: InputType.Text,
                            isRequired: true,
                            defaultValue: "kubernetes cluster"
                        }
                    ]
                }
            } as FormInputConfigObject,
            formOutput: {
                __version: "1.0",
                schema: {
                    containers: [],
                    elements: [
                        {
                            fieldName: "bestPractices",
                            id: "best_practices",
                            label: "Monitoring Best Practices",
                            type: InputType.JSON,
                            props: { disabled: true }
                        },
                        {
                            fieldName: "resources",
                            id: "resources",
                            label: "Reference Resources",
                            type: InputType.JSON,
                            props: { disabled: true }
                        }
                    ]
                }
            } as FormOutputConfigObject,
            executionStrategy: "deterministic" as const,
        } as RoutineVersionConfigObject,
    }
};

/**
 * BPMN-based multi-step routine configurations
 * These are complex workflows that combine multiple routines
 */
export const BPMN_ROUTINES = {
    /**
     * Comprehensive Security Audit Workflow
     * Combines multiple security checks into a comprehensive audit
     * Used by: Security swarms
     */
    COMPREHENSIVE_SECURITY_AUDIT: {
        id: "comprehensive_security_audit",
        name: "Comprehensive Security Audit Workflow",
        description: "Multi-step security audit combining API security, data privacy, and threat analysis",
        version: "1.0.0",
        resourceSubType: ResourceSubType.RoutineMultiStep,
        config: {
            __version: "1.0",
            graph: {
                __version: "1.0",
                __type: "BPMN-2.0",
                schema: {
                    __format: "xml",
                    data: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" 
                  xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" 
                  xmlns:dc="http://www.omg.org/spec/DD/20100524/DC"
                  xmlns:di="http://www.omg.org/spec/DD/20100524/DI"
                  xmlns:vrooli="http://vrooli.com/bpmn/extensions">
  <bpmn:process id="comprehensive_security_audit_process" name="Comprehensive Security Audit">
    
    <bpmn:startEvent id="start" name="Start Audit">
      <bpmn:outgoing>flow1</bpmn:outgoing>
    </bpmn:startEvent>
    
    <bpmn:parallelGateway id="split_analysis" name="Split Analysis">
      <bpmn:incoming>flow1</bpmn:incoming>
      <bpmn:outgoing>flow2</bpmn:outgoing>
      <bpmn:outgoing>flow3</bpmn:outgoing>
      <bpmn:outgoing>flow4</bpmn:outgoing>
    </bpmn:parallelGateway>
    
    <bpmn:callActivity id="api_security_check" name="API Security Check" calledElement="api_security_scan">
      <bpmn:extensionElements>
        <vrooli:ioMapping>
          <vrooli:input name="apiLogs" fromContext="root.systemLogs" />
          <vrooli:input name="timeRange" fromContext="root.auditTimeRange" />
          <vrooli:output name="threats" />
          <vrooli:output name="anomalies" />
        </vrooli:ioMapping>
      </bpmn:extensionElements>
      <bpmn:incoming>flow2</bpmn:incoming>
      <bpmn:outgoing>flow5</bpmn:outgoing>
    </bpmn:callActivity>
    
    <bpmn:callActivity id="data_privacy_check" name="GDPR Compliance Check" calledElement="gdpr_data_audit">
      <bpmn:extensionElements>
        <vrooli:ioMapping>
          <vrooli:input name="processingRecords" fromContext="root.dataProcessingRecords" />
          <vrooli:input name="retentionPolicies" fromContext="root.retentionPolicies" />
          <vrooli:output name="complianceScore" />
          <vrooli:output name="violations" />
        </vrooli:ioMapping>
      </bpmn:extensionElements>
      <bpmn:incoming>flow3</bpmn:incoming>
      <bpmn:outgoing>flow6</bpmn:outgoing>
    </bpmn:callActivity>
    
    <bpmn:callActivity id="threat_analysis" name="Threat Pattern Analysis" calledElement="trading_pattern_analysis">
      <bpmn:extensionElements>
        <vrooli:ioMapping>
          <vrooli:input name="tradingData" fromContext="root.activityData" />
          <vrooli:input name="timeWindow" fromContext="root.auditTimeRange" />
          <vrooli:output name="suspiciousPatterns" />
          <vrooli:output name="riskScore" />
        </vrooli:ioMapping>
      </bpmn:extensionElements>
      <bpmn:incoming>flow4</bpmn:incoming>
      <bpmn:outgoing>flow7</bpmn:outgoing>
    </bpmn:callActivity>
    
    <bpmn:parallelGateway id="join_analysis" name="Join Results">
      <bpmn:incoming>flow5</bpmn:incoming>
      <bpmn:incoming>flow6</bpmn:incoming>
      <bpmn:incoming>flow7</bpmn:incoming>
      <bpmn:outgoing>flow8</bpmn:outgoing>
    </bpmn:parallelGateway>
    
    <bpmn:scriptTask id="generate_report" name="Generate Audit Report">
      <bpmn:incoming>flow8</bpmn:incoming>
      <bpmn:outgoing>flow9</bpmn:outgoing>
      <bpmn:script>
        // Combine results from all security checks
        const report = {
          timestamp: new Date(),
          apiSecurity: {
            threats: api_security_check.threats,
            anomalies: api_security_check.anomalies
          },
          dataPrivacy: {
            complianceScore: data_privacy_check.complianceScore,
            violations: data_privacy_check.violations
          },
          threatAnalysis: {
            patterns: threat_analysis.suspiciousPatterns,
            riskScore: threat_analysis.riskScore
          },
          overallRiskScore: calculateOverallRisk(),
          recommendations: generateRecommendations()
        };
        return report;
      </bpmn:script>
    </bpmn:scriptTask>
    
    <bpmn:endEvent id="end" name="Audit Complete">
      <bpmn:incoming>flow9</bpmn:incoming>
    </bpmn:endEvent>
    
    <!-- Sequence Flows -->
    <bpmn:sequenceFlow id="flow1" sourceRef="start" targetRef="split_analysis" />
    <bpmn:sequenceFlow id="flow2" sourceRef="split_analysis" targetRef="api_security_check" />
    <bpmn:sequenceFlow id="flow3" sourceRef="split_analysis" targetRef="data_privacy_check" />
    <bpmn:sequenceFlow id="flow4" sourceRef="split_analysis" targetRef="threat_analysis" />
    <bpmn:sequenceFlow id="flow5" sourceRef="api_security_check" targetRef="join_analysis" />
    <bpmn:sequenceFlow id="flow6" sourceRef="data_privacy_check" targetRef="join_analysis" />
    <bpmn:sequenceFlow id="flow7" sourceRef="threat_analysis" targetRef="join_analysis" />
    <bpmn:sequenceFlow id="flow8" sourceRef="join_analysis" targetRef="generate_report" />
    <bpmn:sequenceFlow id="flow9" sourceRef="generate_report" targetRef="end" />
    
  </bpmn:process>
</bpmn:definitions>`,
                    activityMap: {
                        "api_security_check": {
                            subroutineId: SEQUENTIAL_ROUTINES.API_SECURITY_SCAN.id,
                            inputMap: {
                                "apiLogs": "apiLogs",
                                "timeRange": "timeRange"
                            },
                            outputMap: {
                                "threats": "threats",
                                "anomalies": "anomalies"
                            }
                        },
                        "data_privacy_check": {
                            subroutineId: SEQUENTIAL_ROUTINES.GDPR_DATA_AUDIT.id,
                            inputMap: {
                                "processingRecords": "processingRecords",
                                "retentionPolicies": "retentionPolicies"
                            },
                            outputMap: {
                                "complianceScore": "complianceScore",
                                "violations": "violations"
                            }
                        },
                        "threat_analysis": {
                            subroutineId: SEQUENTIAL_ROUTINES.TRADING_PATTERN_ANALYSIS.id,
                            inputMap: {
                                "tradingData": "tradingData",
                                "timeWindow": "timeWindow"
                            },
                            outputMap: {
                                "suspiciousPatterns": "suspiciousPatterns",
                                "riskScore": "riskScore"
                            }
                        }
                    },
                    rootContext: {
                        inputMap: {
                            "systemLogs": "systemLogs",
                            "auditTimeRange": "auditTimeRange",
                            "dataProcessingRecords": "dataProcessingRecords",
                            "retentionPolicies": "retentionPolicies",
                            "activityData": "activityData"
                        },
                        outputMap: {
                            "auditReport": "auditReport"
                        }
                    }
                }
            } as GraphBpmnConfigObject,
            formInput: {
                __version: "1.0",
                schema: {
                    containers: [],
                    elements: [
                        {
                            fieldName: "systemLogs",
                            id: "system_logs",
                            label: "System Logs",
                            type: InputType.JSON,
                            isRequired: true
                        },
                        {
                            fieldName: "auditTimeRange",
                            id: "audit_time_range",
                            label: "Audit Time Range",
                            type: InputType.Text,
                            isRequired: true,
                            defaultValue: "24h"
                        },
                        {
                            fieldName: "dataProcessingRecords",
                            id: "data_processing_records",
                            label: "Data Processing Records",
                            type: InputType.JSON,
                            isRequired: true
                        },
                        {
                            fieldName: "retentionPolicies",
                            id: "retention_policies",
                            label: "Data Retention Policies",
                            type: InputType.JSON,
                            isRequired: true
                        },
                        {
                            fieldName: "activityData",
                            id: "activity_data",
                            label: "User Activity Data",
                            type: InputType.JSON,
                            isRequired: true
                        }
                    ]
                }
            } as FormInputConfigObject,
            formOutput: {
                __version: "1.0",
                schema: {
                    containers: [],
                    elements: [
                        {
                            fieldName: "auditReport",
                            id: "audit_report",
                            label: "Security Audit Report",
                            type: InputType.JSON,
                            props: { disabled: true }
                        }
                    ]
                }
            } as FormOutputConfigObject,
            executionStrategy: "reasoning" as const,
            subroutineStrategies: {
                "api_security_check": "deterministic",
                "data_privacy_check": "reasoning",
                "threat_analysis": "deterministic"
            }
        } as RoutineVersionConfigObject,
    },

    /**
     * Medical Treatment Validation Workflow
     * Multi-step validation for AI-generated medical treatments
     * Used by: Medical safety swarms
     */
    MEDICAL_TREATMENT_VALIDATION: {
        id: "medical_treatment_validation",
        name: "Medical Treatment Validation Workflow",
        description: "Comprehensive validation of AI-generated medical treatment plans",
        version: "1.0.0",
        resourceSubType: ResourceSubType.RoutineMultiStep,
        config: {
            __version: "1.0",
            graph: {
                __version: "1.0",
                __type: "BPMN-2.0",
                schema: {
                    __format: "xml",
                    data: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="medical_treatment_validation_process">
    
    <bpmn:startEvent id="start" name="Start Validation">
      <bpmn:outgoing>flow1</bpmn:outgoing>
    </bpmn:startEvent>
    
    <bpmn:callActivity id="hipaa_check" name="HIPAA Compliance Check" calledElement="hipaa_compliance_check">
      <bpmn:extensionElements>
        <vrooli:ioMapping>
          <vrooli:input name="aiOutput" fromContext="root.treatmentPlan" />
          <vrooli:input name="confidenceThreshold" fromContext="root.complianceThreshold" />
          <vrooli:output name="phiDetected" />
          <vrooli:output name="complianceStatus" />
        </vrooli:ioMapping>
      </bpmn:extensionElements>
      <bpmn:incoming>flow1</bpmn:incoming>
      <bpmn:outgoing>flow2</bpmn:outgoing>
    </bpmn:callActivity>
    
    <bpmn:exclusiveGateway id="compliance_gateway" name="Compliance Check">
      <bpmn:incoming>flow2</bpmn:incoming>
      <bpmn:outgoing>flow3</bpmn:outgoing>
      <bpmn:outgoing>flow_reject</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    
    <bpmn:callActivity id="clinical_validation" name="Clinical Validation" calledElement="medical_diagnosis_validation">
      <bpmn:extensionElements>
        <vrooli:ioMapping>
          <vrooli:input name="diagnosis" fromContext="root.diagnosis" />
          <vrooli:input name="demographics" fromContext="root.patientDemographics" />
          <vrooli:input name="symptoms" fromContext="root.symptoms" />
          <vrooli:output name="validationReport" toRootContext="clinicalValidation" />
        </vrooli:ioMapping>
      </bpmn:extensionElements>
      <bpmn:incoming>flow3</bpmn:incoming>
      <bpmn:outgoing>flow4</bpmn:outgoing>
    </bpmn:callActivity>
    
    <bpmn:callActivity id="bias_check" name="Bias Detection" calledElement="output_quality_assessment">
      <bpmn:extensionElements>
        <vrooli:ioMapping>
          <vrooli:input name="output" fromContext="root.treatmentPlan" />
          <vrooli:input name="criteria" fromContext="root.biasCriteria" />
          <vrooli:output name="biasReport" toRootContext="biasAssessment" />
        </vrooli:ioMapping>
      </bpmn:extensionElements>
      <bpmn:incoming>flow4</bpmn:incoming>
      <bpmn:outgoing>flow5</bpmn:outgoing>
    </bpmn:callActivity>
    
    <bpmn:endEvent id="end_approved" name="Treatment Approved">
      <bpmn:incoming>flow5</bpmn:incoming>
    </bpmn:endEvent>
    
    <bpmn:endEvent id="end_rejected" name="Treatment Rejected">
      <bpmn:incoming>flow_reject</bpmn:incoming>
    </bpmn:endEvent>
    
    <!-- Sequence Flows -->
    <bpmn:sequenceFlow id="flow1" sourceRef="start" targetRef="hipaa_check" />
    <bpmn:sequenceFlow id="flow2" sourceRef="hipaa_check" targetRef="compliance_gateway" />
    <bpmn:sequenceFlow id="flow3" sourceRef="compliance_gateway" targetRef="clinical_validation">
      <bpmn:conditionExpression>complianceStatus == "compliant"</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="flow_reject" sourceRef="compliance_gateway" targetRef="end_rejected">
      <bpmn:conditionExpression>complianceStatus != "compliant"</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="flow4" sourceRef="clinical_validation" targetRef="bias_check" />
    <bpmn:sequenceFlow id="flow5" sourceRef="bias_check" targetRef="end_approved" />
    
  </bpmn:process>
</bpmn:definitions>`,
                    activityMap: {
                        "hipaa_check": {
                            subroutineId: SEQUENTIAL_ROUTINES.HIPAA_COMPLIANCE_CHECK.id,
                            inputMap: {
                                "aiOutput": "aiOutput",
                                "confidenceThreshold": "confidenceThreshold"
                            },
                            outputMap: {
                                "phiDetected": "phiDetected",
                                "complianceStatus": "complianceStatus"
                            }
                        },
                        "clinical_validation": {
                            subroutineId: SEQUENTIAL_ROUTINES.MEDICAL_DIAGNOSIS_VALIDATION.id,
                            inputMap: {
                                "diagnosis": "diagnosis",
                                "demographics": "demographics",
                                "symptoms": "symptoms"
                            },
                            outputMap: {
                                "validationReport": "response"
                            }
                        },
                        "bias_check": {
                            subroutineId: SEQUENTIAL_ROUTINES.OUTPUT_QUALITY_ASSESSMENT.id,
                            inputMap: {
                                "output": "output",
                                "criteria": "criteria"
                            },
                            outputMap: {
                                "biasReport": "response"
                            }
                        }
                    },
                    rootContext: {
                        inputMap: {
                            "treatmentPlan": "treatmentPlan",
                            "complianceThreshold": "complianceThreshold",
                            "diagnosis": "diagnosis",
                            "patientDemographics": "patientDemographics",
                            "symptoms": "symptoms",
                            "biasCriteria": "biasCriteria"
                        },
                        outputMap: {
                            "clinicalValidation": "clinicalValidation",
                            "biasAssessment": "biasAssessment"
                        }
                    }
                }
            } as GraphBpmnConfigObject,
            formInput: {
                __version: "1.0",
                schema: {
                    containers: [],
                    elements: [
                        {
                            fieldName: "treatmentPlan",
                            id: "treatment_plan",
                            label: "AI Treatment Plan",
                            type: InputType.Text,
                            isRequired: true
                        },
                        {
                            fieldName: "complianceThreshold",
                            id: "compliance_threshold",
                            label: "Compliance Threshold",
                            type: InputType.Slider,
                            isRequired: true,
                            defaultValue: 0.9,
                            props: { min: 0.5, max: 1.0, step: 0.05 }
                        },
                        {
                            fieldName: "diagnosis",
                            id: "diagnosis",
                            label: "Diagnosis",
                            type: InputType.Text,
                            isRequired: true
                        },
                        {
                            fieldName: "patientDemographics",
                            id: "patient_demographics",
                            label: "Patient Demographics",
                            type: InputType.JSON,
                            isRequired: true
                        },
                        {
                            fieldName: "symptoms",
                            id: "symptoms",
                            label: "Symptoms",
                            type: InputType.Text,
                            isRequired: true
                        },
                        {
                            fieldName: "biasCriteria",
                            id: "bias_criteria",
                            label: "Bias Detection Criteria",
                            type: InputType.JSON,
                            isRequired: true,
                            defaultValue: {
                                biasThreshold: 0.15,
                                checkDemographic: true,
                                checkCultural: true
                            }
                        }
                    ]
                }
            } as FormInputConfigObject,
            formOutput: {
                __version: "1.0",
                schema: {
                    containers: [],
                    elements: [
                        {
                            fieldName: "clinicalValidation",
                            id: "clinical_validation",
                            label: "Clinical Validation Report",
                            type: InputType.Text,
                            props: { disabled: true }
                        },
                        {
                            fieldName: "biasAssessment",
                            id: "bias_assessment",
                            label: "Bias Assessment Report",
                            type: InputType.Text,
                            props: { disabled: true }
                        }
                    ]
                }
            } as FormOutputConfigObject,
            executionStrategy: "reasoning" as const,
            allowStrategyOverride: true,
            subroutineStrategies: {
                "hipaa_check": "deterministic",
                "clinical_validation": "reasoning",
                "bias_check": "reasoning"
            }
        } as RoutineVersionConfigObject,
    },

    /**
     * System Resilience Optimization Workflow
     * Analyzes failures and optimizes system resilience
     * Used by: Resilience agent swarms
     */
    RESILIENCE_OPTIMIZATION_WORKFLOW: {
        id: "resilience_optimization_workflow",
        name: "System Resilience Optimization",
        description: "Comprehensive workflow for analyzing failures and optimizing system resilience",
        version: "1.0.0",
        resourceSubType: ResourceSubType.RoutineMultiStep,
        config: {
            __version: "1.0",
            graph: {
                __version: "1.0",
                __type: "BPMN-2.0",
                schema: {
                    __format: "xml",
                    data: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="resilience_optimization_process">
    
    <bpmn:startEvent id="start" name="Start Analysis">
      <bpmn:outgoing>flow1</bpmn:outgoing>
    </bpmn:startEvent>
    
    <bpmn:callActivity id="failure_analysis" name="Analyze Failures" calledElement="system_failure_analysis">
      <bpmn:extensionElements>
        <vrooli:ioMapping>
          <vrooli:input name="failureLogs" fromContext="root.systemLogs" />
          <vrooli:input name="analysisWindow" fromContext="root.timeWindow" />
          <vrooli:output name="patterns" />
          <vrooli:output name="correlations" />
          <vrooli:output name="predictions" />
        </vrooli:ioMapping>
      </bpmn:extensionElements>
      <bpmn:incoming>flow1</bpmn:incoming>
      <bpmn:outgoing>flow2</bpmn:outgoing>
    </bpmn:callActivity>
    
    <bpmn:callActivity id="performance_analysis" name="Performance Analysis" calledElement="performance_bottleneck_detection">
      <bpmn:extensionElements>
        <vrooli:ioMapping>
          <vrooli:input name="metrics" fromContext="root.performanceMetrics" />
          <vrooli:input name="config" fromContext="root.systemConfig" />
          <vrooli:input name="threshold" fromContext="root.performanceThreshold" />
          <vrooli:output name="bottlenecks" />
          <vrooli:output name="recommendations" />
        </vrooli:ioMapping>
      </bpmn:extensionElements>
      <bpmn:incoming>flow2</bpmn:incoming>
      <bpmn:outgoing>flow3</bpmn:outgoing>
    </bpmn:callActivity>
    
    <bpmn:scriptTask id="generate_optimization_plan" name="Generate Optimization Plan">
      <bpmn:incoming>flow3</bpmn:incoming>
      <bpmn:outgoing>flow4</bpmn:outgoing>
      <bpmn:script>
        const plan = {
          failurePatterns: failure_analysis.patterns,
          correlations: failure_analysis.correlations,
          predictions: failure_analysis.predictions,
          bottlenecks: performance_analysis.bottlenecks,
          recommendations: performance_analysis.recommendations,
          optimizationStrategy: generateStrategy(),
          implementationPlan: createImplementationPlan()
        };
        return plan;
      </bpmn:script>
    </bpmn:scriptTask>
    
    <bpmn:endEvent id="end" name="Plan Ready">
      <bpmn:incoming>flow4</bpmn:incoming>
    </bpmn:endEvent>
    
    <!-- Sequence Flows -->
    <bpmn:sequenceFlow id="flow1" sourceRef="start" targetRef="failure_analysis" />
    <bpmn:sequenceFlow id="flow2" sourceRef="failure_analysis" targetRef="performance_analysis" />
    <bpmn:sequenceFlow id="flow3" sourceRef="performance_analysis" targetRef="generate_optimization_plan" />
    <bpmn:sequenceFlow id="flow4" sourceRef="generate_optimization_plan" targetRef="end" />
    
  </bpmn:process>
</bpmn:definitions>`,
                    activityMap: {
                        "failure_analysis": {
                            subroutineId: SEQUENTIAL_ROUTINES.SYSTEM_FAILURE_ANALYSIS.id,
                            inputMap: {
                                "failureLogs": "failureLogs",
                                "analysisWindow": "analysisWindow"
                            },
                            outputMap: {
                                "patterns": "patterns",
                                "correlations": "correlations",
                                "predictions": "predictions"
                            }
                        },
                        "performance_analysis": {
                            subroutineId: SEQUENTIAL_ROUTINES.PERFORMANCE_BOTTLENECK_DETECTION.id,
                            inputMap: {
                                "metrics": "metrics",
                                "config": "config",
                                "threshold": "threshold"
                            },
                            outputMap: {
                                "bottlenecks": "bottlenecks",
                                "recommendations": "recommendations"
                            }
                        }
                    },
                    rootContext: {
                        inputMap: {
                            "systemLogs": "systemLogs",
                            "timeWindow": "timeWindow",
                            "performanceMetrics": "performanceMetrics",
                            "systemConfig": "systemConfig",
                            "performanceThreshold": "performanceThreshold"
                        },
                        outputMap: {
                            "optimizationPlan": "optimizationPlan"
                        }
                    }
                }
            } as GraphBpmnConfigObject,
            formInput: {
                __version: "1.0",
                schema: {
                    containers: [],
                    elements: [
                        {
                            fieldName: "systemLogs",
                            id: "system_logs",
                            label: "System Failure Logs",
                            type: InputType.JSON,
                            isRequired: true
                        },
                        {
                            fieldName: "timeWindow",
                            id: "time_window",
                            label: "Analysis Time Window",
                            type: InputType.Text,
                            isRequired: true,
                            defaultValue: "24h"
                        },
                        {
                            fieldName: "performanceMetrics",
                            id: "performance_metrics",
                            label: "Performance Metrics",
                            type: InputType.JSON,
                            isRequired: true
                        },
                        {
                            fieldName: "systemConfig",
                            id: "system_config",
                            label: "System Configuration",
                            type: InputType.JSON,
                            isRequired: true
                        },
                        {
                            fieldName: "performanceThreshold",
                            id: "performance_threshold",
                            label: "Performance Threshold (ms)",
                            type: InputType.IntegerInput,
                            isRequired: true,
                            defaultValue: 1000
                        }
                    ]
                }
            } as FormInputConfigObject,
            formOutput: {
                __version: "1.0",
                schema: {
                    containers: [],
                    elements: [
                        {
                            fieldName: "optimizationPlan",
                            id: "optimization_plan",
                            label: "Resilience Optimization Plan",
                            type: InputType.JSON,
                            props: { disabled: true }
                        }
                    ]
                }
            } as FormOutputConfigObject,
            executionStrategy: "reasoning" as const,
            subroutineStrategies: {
                "failure_analysis": "reasoning",
                "performance_analysis": "deterministic"
            }
        } as RoutineVersionConfigObject,
    }
};

/**
 * Helper function to get all routines as a flat array
 */
export function getAllRoutines() {
    return [
        ...Object.values(SEQUENTIAL_ROUTINES),
        ...Object.values(BPMN_ROUTINES)
    ];
}

/**
 * Helper function to get routine by ID
 */
export function getRoutineById(routineId: string) {
    return getAllRoutines().find(routine => routine.id === routineId);
}

/**
 * Helper function to get routines by execution strategy
 */
export function getRoutinesByStrategy(strategy: "reasoning" | "deterministic" | "conversational" | "auto") {
    return getAllRoutines().filter(routine => 
        routine.config.executionStrategy === strategy
    );
}

/**
 * Helper function to get routines by resource subtype
 */
export function getRoutinesByResourceSubType(subType: ResourceSubType) {
    return getAllRoutines().filter(routine => 
        routine.resourceSubType === subType
    );
}

/**
 * Map of agent IDs to their associated routine IDs
 */
export const AGENT_ROUTINE_MAP = {
    // Security agents
    "hipaa_compliance_monitor": [SEQUENTIAL_ROUTINES.HIPAA_COMPLIANCE_CHECK.id],
    "medical_ai_safety_monitor": [SEQUENTIAL_ROUTINES.MEDICAL_DIAGNOSIS_VALIDATION.id],
    "trading_fraud_detector": [SEQUENTIAL_ROUTINES.TRADING_PATTERN_ANALYSIS.id],
    "api_security_monitor": [SEQUENTIAL_ROUTINES.API_SECURITY_SCAN.id],
    "data_privacy_compliance": [SEQUENTIAL_ROUTINES.GDPR_DATA_AUDIT.id],
    
    // Resilience agents
    "resilience_pattern_learner": [SEQUENTIAL_ROUTINES.SYSTEM_FAILURE_ANALYSIS.id],
    
    // Strategy evolution agents
    "routine_performance_analyzer": [SEQUENTIAL_ROUTINES.PERFORMANCE_BOTTLENECK_DETECTION.id],
    "cost_optimization_specialist": [SEQUENTIAL_ROUTINES.COST_ANALYSIS.id],
    
    // Quality agents
    "output_quality_monitor": [SEQUENTIAL_ROUTINES.OUTPUT_QUALITY_ASSESSMENT.id],
    
    // Monitoring agents
    "system_health_monitor": [SEQUENTIAL_ROUTINES.SYSTEM_HEALTH_CHECK.id],
    
    // Swarm routines (multi-agent workflows)
    "healthcare_security_swarm": [BPMN_ROUTINES.MEDICAL_TREATMENT_VALIDATION.id],
    "security_swarm": [BPMN_ROUTINES.COMPREHENSIVE_SECURITY_AUDIT.id],
    "resilience_evolution_swarm": [BPMN_ROUTINES.RESILIENCE_OPTIMIZATION_WORKFLOW.id],
};