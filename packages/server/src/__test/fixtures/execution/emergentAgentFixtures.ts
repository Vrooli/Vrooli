/**
 * Test fixtures for emergent agents and routines
 * 
 * These fixtures demonstrate how emergent capabilities develop through goal-driven
 * agents that learn from events and propose routine improvements.
 * 
 * All examples are based on the documentation in docs/architecture/execution/emergent-capabilities/agent-examples/
 */

import type { EmergentAgentConfig, EmergentSwarmConfig } from "../../../services/execution/cross-cutting/agents/agentDeploymentService.js";
import type { IntelligentEvent } from "../../../services/execution/cross-cutting/events/eventBus.js";

/**
 * Extended agent configuration that includes capabilities and learning patterns
 */
export interface ExtendedAgentConfig extends EmergentAgentConfig {
    name: string;
    capabilities?: Record<string, any>;
    learningPatterns?: Record<string, string>;
    safetyChecks?: Record<string, any>;
    complianceFrameworks?: Record<string, any>;
    fraudDetection?: Record<string, any>;
    threatDetection?: Record<string, any>;
    privacyProtections?: Record<string, any>;
    crossBorderCompliance?: Record<string, any>;
    adaptiveLearning?: Record<string, any>;
    resilienceCapabilities?: Record<string, any>;
    strategyEvolution?: Record<string, any>;
    routineAnalysis?: Record<string, any>;
    transformationRules?: Record<string, any>;
    costOptimization?: Record<string, any>;
    evolutionTracking?: Record<string, any>;
}

/**
 * Sample routine definitions that agents can reference
 */
export const TEST_ROUTINES = {
    // Performance analysis routines
    ANALYZE_PERFORMANCE_PATTERNS: {
        id: "analyze_performance_patterns",
        name: "Analyze Performance Patterns",
        description: "Analyze system performance metrics to identify bottlenecks and optimization opportunities",
        strategy: "reasoning",
        parameters: {
            metricTypes: ["response_time", "memory_usage", "cpu_usage"],
            analysisWindow: "24h",
            threshold: 0.8,
        },
    },
    
    // Quality assessment routines
    ASSESS_OUTPUT_QUALITY: {
        id: "assess_output_quality",
        name: "Assess Output Quality",
        description: "Evaluate output quality including accuracy, completeness, and bias detection",
        strategy: "reasoning",
        parameters: {
            qualityMetrics: ["accuracy", "completeness", "bias_score"],
            confidenceThreshold: 0.85,
            samplingRate: 0.1,
        },
    },
    
    // Security analysis routines
    ANALYZE_SECURITY_PATTERNS: {
        id: "analyze_security_patterns",
        name: "Analyze Security Patterns",
        description: "Analyze security events to identify threat patterns and compliance issues",
        strategy: "reasoning",
        parameters: {
            threatCategories: ["injection", "privilege_escalation", "data_exfiltration"],
            complianceFrameworks: ["gdpr", "hipaa", "sox"],
            riskThreshold: 0.7,
        },
    },
    
    // Cost optimization routines
    ANALYZE_COST_PATTERNS: {
        id: "analyze_cost_patterns",
        name: "Analyze Cost Patterns", 
        description: "Analyze operational costs to identify reduction opportunities",
        strategy: "reasoning",
        parameters: {
            costCategories: ["compute", "storage", "network", "api_calls"],
            optimizationTargets: ["rightsizing", "scheduling", "caching"],
            savingsThreshold: 0.15,
        },
    },
    
    // Error analysis routines
    ANALYZE_ERROR_PATTERNS: {
        id: "analyze_error_patterns",
        name: "Analyze Error Patterns",
        description: "Analyze error patterns to identify root causes and prevention strategies",
        strategy: "reasoning",
        parameters: {
            errorTypes: ["timeout", "validation", "resource_limit", "external_api"],
            analysisDepth: "deep",
            correlationWindow: "1h",
        },
    },
    
    // Resilience analysis routines
    ANALYZE_RESILIENCE_PATTERNS: {
        id: "analyze_resilience_patterns",
        name: "Analyze Resilience Patterns",
        description: "Analyze system failures and recovery patterns",
        strategy: "reasoning",
        parameters: {
            failureTypes: ["timeout", "connection_lost", "rate_limit", "server_error"],
            recoveryStrategies: ["retry", "circuit_breaker", "fallback", "cache"],
            analysisWindow: "24h",
        },
    },
    
    // Strategy performance analysis
    ANALYZE_STRATEGY_PERFORMANCE: {
        id: "analyze_strategy_performance",
        name: "Analyze Strategy Performance",
        description: "Analyze routine execution strategies to identify evolution opportunities",
        strategy: "reasoning",
        parameters: {
            strategies: ["conversational", "reasoning", "deterministic"],
            performanceMetrics: ["execution_time", "cost", "accuracy", "reliability"],
            evolutionThreshold: 0.8,
        },
    },
} as const;

/**
 * SECURITY AGENTS - From security-agents.md
 */
export const SECURITY_AGENTS: Record<string, ExtendedAgentConfig> = {
    // HIPAA Compliance Agent
    HIPAA_COMPLIANCE_AGENT: {
        agentId: "hipaa_compliance_monitor",
        name: "HIPAA Compliance Monitor",
        goal: "Zero HIPAA violations across all medical AI workflows",
        initialRoutine: TEST_ROUTINES.ANALYZE_SECURITY_PATTERNS.id,
        subscriptions: [
            "ai/medical/*",
            "data/patient/*",
            "api/medical/*",
            "audit/medical/*"
        ],
        priority: 9,
        capabilities: {
            phiDetection: {
                description: "Scan for protected health information in all outputs",
                patterns: [
                    "social_security_numbers",
                    "medical_record_numbers", 
                    "patient_names_with_medical_context",
                    "detailed_medical_histories",
                    "specific_treatment_details"
                ],
                confidenceThreshold: 0.85
            },
            auditTrailGeneration: {
                description: "Automatically generate HIPAA-compliant audit documentation",
                includes: [
                    "who_accessed_what_when",
                    "purpose_of_access",
                    "data_transformation_tracking",
                    "compliance_validation_results"
                ]
            },
            violationResponse: {
                description: "Immediate response to potential HIPAA violations",
                actions: [
                    "quarantine_potentially_exposed_data",
                    "notify_privacy_officer",
                    "generate_incident_report",
                    "recommend_remediation_steps"
                ]
            }
        },
        learningPatterns: {
            medicalContextRecognition: "Learn to distinguish PHI from general medical knowledge",
            falsePositiveReduction: "Reduce alerts on legitimate medical education content",
            newPHIPatterns: "Identify emerging PHI exposure patterns in AI outputs"
        }
    },
    
    // Medical AI Quality & Safety Agent
    MEDICAL_SAFETY_AGENT: {
        agentId: "medical_ai_safety_monitor",
        name: "Medical AI Quality & Safety Monitor",
        goal: "Ensure clinical accuracy and patient safety in all AI-generated medical content",
        initialRoutine: TEST_ROUTINES.ASSESS_OUTPUT_QUALITY.id,
        subscriptions: [
            "ai/diagnosis/*",
            "ai/treatment/*",
            "ai/drug_interaction/*",
            "ai/medical_writing/*"
        ],
        priority: 9,
        safetyChecks: {
            clinicalAccuracy: {
                validationSources: [
                    "pubmed_latest_research",
                    "clinical_guidelines_database", 
                    "fda_drug_interaction_database",
                    "medical_consensus_protocols"
                ],
                accuracyThreshold: 0.95
            },
            biasDetection: {
                biasTypes: [
                    "demographic_bias_in_diagnosis",
                    "socioeconomic_treatment_bias",
                    "gender_based_symptom_interpretation",
                    "racial_disparities_in_recommendations"
                ],
                interventions: [
                    "flag_potentially_biased_recommendations",
                    "suggest_bias_neutral_alternatives",
                    "require_human_review_for_high_risk"
                ]
            },
            harmPrevention: {
                riskCategories: [
                    "contraindicated_medication_combinations",
                    "inappropriate_dosage_recommendations",
                    "missed_critical_symptoms",
                    "delayed_care_suggestions"
                ],
                responses: [
                    "immediate_alert_to_medical_team",
                    "block_dangerous_recommendations",
                    "suggest_safer_alternatives"
                ]
            }
        }
    },
    
    // Trading Fraud Detection Agent
    TRADING_FRAUD_AGENT: {
        agentId: "trading_fraud_detector",
        name: "Trading Fraud Detection System",
        goal: "Prevent financial fraud while maintaining trading performance",
        initialRoutine: TEST_ROUTINES.ANALYZE_SECURITY_PATTERNS.id,
        subscriptions: [
            "trading/order_placed",
            "trading/execution",
            "account/access/*",
            "api/trading_calls",
            "market/unusual_activity"
        ],
        priority: 9,
        fraudDetection: {
            suspiciousPatterns: [
                {
                    name: "wash_trading",
                    description: "Artificially inflating trading volume",
                    indicators: [
                        "rapid_buy_sell_same_security",
                        "minimal_price_movement",
                        "coordinated_account_activity"
                    ],
                    severity: "high",
                    action: "immediate_investigation"
                },
                {
                    name: "pump_and_dump",
                    description: "Coordinated price manipulation",
                    indicators: [
                        "sudden_volume_spike",
                        "social_media_coordination",
                        "rapid_position_building_then_selling"
                    ],
                    severity: "critical",
                    action: "freeze_positions_notify_sec"
                },
                {
                    name: "insider_trading",
                    description: "Trading on non-public information",
                    indicators: [
                        "unusual_pre_announcement_activity",
                        "employee_trading_before_earnings",
                        "atypical_options_activity"
                    ],
                    severity: "critical",
                    action: "immediate_sec_notification"
                }
            ],
            adaptiveLearning: {
                newPatternDetection: "Identify emerging fraud techniques",
                falsePositiveReduction: "Learn from legitimate trading patterns",
                crossMarketAnalysis: "Detect fraud across multiple markets"
            }
        }
    },
    
    // Financial Risk & Compliance Agent
    FINANCIAL_COMPLIANCE_AGENT: {
        agentId: "financial_compliance_monitor",
        name: "Financial Risk & Regulatory Compliance",
        goal: "Ensure all financial operations meet regulatory requirements",
        initialRoutine: TEST_ROUTINES.ANALYZE_SECURITY_PATTERNS.id,
        subscriptions: [
            "transaction/large_amount",
            "customer/kyc_check",
            "trading/cross_border",
            "compliance/rule_change"
        ],
        priority: 9,
        complianceFrameworks: {
            sox: {
                description: "Sarbanes-Oxley compliance for public companies",
                controls: [
                    "financial_reporting_accuracy",
                    "internal_control_validation",
                    "executive_certification_tracking"
                ]
            },
            bsa: {
                description: "Bank Secrecy Act compliance",
                requirements: [
                    "currency_transaction_reporting",
                    "suspicious_activity_monitoring",
                    "record_keeping_validation"
                ]
            },
            mifid: {
                description: "Markets in Financial Instruments Directive",
                compliance: [
                    "best_execution_reporting",
                    "transaction_cost_analysis",
                    "client_categorization_validation"
                ]
            }
        }
    },
    
    // API Security Agent
    API_SECURITY_AGENT: {
        agentId: "api_security_monitor",
        name: "Adaptive API Security Monitor",
        goal: "Protect APIs from abuse while maintaining legitimate access",
        initialRoutine: TEST_ROUTINES.ANALYZE_SECURITY_PATTERNS.id,
        subscriptions: [
            "api/request",
            "api/authentication",
            "api/rate_limit_hit",
            "api/error_spike",
            "security/ip_reputation"
        ],
        priority: 8,
        threatDetection: {
            ddosProtection: {
                indicators: [
                    "sudden_request_volume_spike",
                    "requests_from_many_ips_to_same_endpoint",
                    "repeated_requests_with_same_payload"
                ],
                responses: [
                    "adaptive_rate_limiting",
                    "temporary_ip_blocking", 
                    "traffic_pattern_analysis"
                ]
            },
            bruteForceDetection: {
                patterns: [
                    "repeated_failed_auth_attempts",
                    "password_spray_attacks",
                    "credential_stuffing_attempts"
                ],
                mitigations: [
                    "progressive_delay_implementation",
                    "account_lockout_procedures",
                    "suspicious_ip_flagging"
                ]
            },
            dataExfiltration: {
                anomalies: [
                    "unusual_data_access_patterns",
                    "bulk_data_requests",
                    "off_hours_high_volume_access"
                ],
                protections: [
                    "data_access_rate_limiting",
                    "unusual_pattern_alerts",
                    "data_classification_enforcement"
                ]
            }
        },
        adaptiveLearning: {
            normalTrafficPatterns: "Learn legitimate usage patterns for each API",
            seasonalAdjustments: "Adjust baselines for business cycles",
            userBehaviorProfiling: "Create behavioral profiles for regular users"
        }
    },
    
    // Data Privacy Agent
    DATA_PRIVACY_AGENT: {
        agentId: "data_privacy_compliance",
        name: "Global Data Privacy Compliance",
        goal: "Ensure GDPR, CCPA, and other privacy regulations compliance",
        initialRoutine: TEST_ROUTINES.ANALYZE_SECURITY_PATTERNS.id,
        subscriptions: [
            "data/personal_access",
            "data/cross_border",
            "user/consent_change",
            "data/retention_check"
        ],
        priority: 9,
        privacyProtections: {
            gdprCompliance: {
                dataSubjectRights: [
                    "right_to_access_monitoring",
                    "right_to_erasure_enforcement",
                    "right_to_portability_facilitation",
                    "right_to_rectification_tracking"
                ],
                legalBasisValidation: [
                    "consent_verification",
                    "legitimate_interest_assessment",
                    "contract_fulfillment_validation",
                    "legal_obligation_compliance"
                ]
            },
            ccpaCompliance: {
                californiaRights: [
                    "right_to_know_what_data_collected",
                    "right_to_delete_personal_information",
                    "right_to_opt_out_of_sale",
                    "right_to_non_discrimination"
                ]
            },
            dataMinimization: {
                principles: [
                    "collect_only_necessary_data",
                    "retain_only_as_long_as_needed",
                    "process_only_for_stated_purposes",
                    "ensure_data_accuracy_and_relevance"
                ]
            }
        },
        crossBorderCompliance: {
            adequacyAssessment: "Validate data transfer destination compliance",
            transferMechanisms: "Ensure proper safeguards for international transfers",
            dataLocalization: "Enforce data residency requirements where applicable"
        }
    }
};

/**
 * RESILIENCE AGENTS - From resilience-agents.md
 */
export const RESILIENCE_AGENTS: Record<string, ExtendedAgentConfig> = {
    // Pattern Learning Agent
    PATTERN_LEARNING_AGENT: {
        agentId: "resilience_pattern_learner",
        name: "Pattern Learning Agent",
        goal: "Learn from system failures to improve circuit breaker configurations",
        initialRoutine: TEST_ROUTINES.ANALYZE_RESILIENCE_PATTERNS.id,
        subscriptions: [
            "failure/*",
            "recovery/*",
            "circuit_breaker/*",
            "timeout/*"
        ],
        priority: 8,
        resilienceCapabilities: {
            patternDetection: {
                description: "Identify failure patterns and correlations",
                patterns: [
                    "cascading_failures",
                    "timeout_chains",
                    "resource_exhaustion",
                    "dependency_failures"
                ],
                analysisWindow: "24h",
                minimumOccurrences: 3
            },
            circuitBreakerOptimization: {
                description: "Dynamically adjust circuit breaker thresholds",
                parameters: [
                    "failure_threshold",
                    "success_threshold",
                    "timeout_duration",
                    "half_open_requests"
                ],
                optimizationGoals: [
                    "minimize_false_trips",
                    "maximize_availability",
                    "reduce_error_propagation"
                ]
            }
        },
        learningPatterns: {
            failureCorrelation: "Identify which failures tend to occur together",
            recoveryTimeOptimization: "Learn optimal recovery timing for different failure types",
            predictivePatterns: "Identify early warning signs of impending failures"
        }
    },
    
    // Threshold Optimization Agent
    THRESHOLD_OPTIMIZATION_AGENT: {
        agentId: "threshold_optimizer",
        name: "Threshold Optimization Agent",
        goal: "Optimize retry and timeout thresholds based on success patterns",
        initialRoutine: TEST_ROUTINES.ANALYZE_RESILIENCE_PATTERNS.id,
        subscriptions: [
            "retry/*",
            "timeout/*",
            "latency/*",
            "success_rate/*"
        ],
        priority: 7,
        resilienceCapabilities: {
            thresholdAnalysis: {
                description: "Analyze current threshold effectiveness",
                metrics: [
                    "retry_success_rate",
                    "timeout_accuracy",
                    "false_timeout_rate",
                    "resource_waste"
                ],
                optimizationTargets: [
                    "retry_count",
                    "retry_delay",
                    "timeout_values",
                    "backoff_multipliers"
                ]
            },
            adaptiveAdjustment: {
                description: "Dynamically adjust thresholds based on conditions",
                conditions: [
                    "time_of_day",
                    "system_load",
                    "dependency_health",
                    "network_conditions"
                ],
                adjustmentStrategies: [
                    "gradual_increase",
                    "rapid_decrease",
                    "oscillation_dampening"
                ]
            }
        },
        learningPatterns: {
            timeBasedPatterns: "Learn how optimal thresholds vary by time",
            loadBasedPatterns: "Understand threshold behavior under different loads",
            dependencyPatterns: "Adjust based on downstream service behavior"
        }
    },
    
    // Predictive Failure Agent
    PREDICTIVE_FAILURE_AGENT: {
        agentId: "predictive_failure_detector",
        name: "Predictive Failure Agent",
        goal: "Predict failures before they happen based on early indicators",
        initialRoutine: TEST_ROUTINES.ANALYZE_RESILIENCE_PATTERNS.id,
        subscriptions: [
            "metrics/*",
            "latency/*",
            "error_rate/*",
            "resource_usage/*"
        ],
        priority: 9,
        resilienceCapabilities: {
            earlyWarningDetection: {
                description: "Identify early indicators of impending failures",
                indicators: [
                    "latency_increase",
                    "error_rate_spike",
                    "memory_growth",
                    "connection_pool_exhaustion"
                ],
                predictionWindow: "5-30 minutes",
                confidenceThreshold: 0.75
            },
            proactiveActions: {
                description: "Take action before failures occur",
                actions: [
                    "preemptive_scaling",
                    "traffic_shedding",
                    "cache_warming",
                    "dependency_isolation"
                ],
                triggerThreshold: 0.8
            }
        },
        learningPatterns: {
            failureSequences: "Learn common sequences that lead to failures",
            anomalyBaselines: "Establish normal behavior baselines",
            seasonalPatterns: "Understand time-based failure patterns"
        }
    },
    
    // Recovery Strategy Evolution Agent
    RECOVERY_STRATEGY_EVOLUTION_AGENT: {
        agentId: "recovery_strategy_evolver",
        name: "Recovery Strategy Evolution Agent",
        goal: "Evolve recovery strategies based on what works best",
        initialRoutine: TEST_ROUTINES.ANALYZE_RESILIENCE_PATTERNS.id,
        subscriptions: [
            "recovery/*",
            "fallback/*",
            "graceful_degradation/*",
            "healing/*"
        ],
        priority: 8,
        resilienceCapabilities: {
            strategyEvaluation: {
                description: "Evaluate effectiveness of recovery strategies",
                strategies: [
                    "immediate_retry",
                    "exponential_backoff",
                    "circuit_breaker",
                    "fallback_response",
                    "graceful_degradation"
                ],
                metrics: [
                    "recovery_time",
                    "success_rate",
                    "resource_cost",
                    "user_impact"
                ]
            },
            strategyEvolution: {
                description: "Evolve strategies based on effectiveness",
                evolutionMethods: [
                    "parameter_tuning",
                    "strategy_combination",
                    "conditional_selection",
                    "hybrid_approaches"
                ],
                testingApproach: "gradual_rollout"
            }
        },
        learningPatterns: {
            contextualEffectiveness: "Learn which strategies work best in different contexts",
            combinationPatterns: "Discover effective strategy combinations",
            failureTypeMapping: "Map failure types to optimal recovery strategies"
        }
    }
};

/**
 * STRATEGY EVOLUTION AGENTS - From strategy-evolution-agents.md
 */
export const STRATEGY_EVOLUTION_AGENTS: Record<string, ExtendedAgentConfig> = {
    // Routine Performance Agent
    ROUTINE_PERFORMANCE_AGENT: {
        agentId: "routine_performance_analyzer",
        name: "Routine Performance Agent",
        goal: "Monitor routine execution patterns and identify optimization opportunities",
        initialRoutine: TEST_ROUTINES.ANALYZE_STRATEGY_PERFORMANCE.id,
        subscriptions: [
            "routine/completed",
            "routine/failed",
            "execution/metrics",
            "strategy/performance"
        ],
        priority: 7,
        routineAnalysis: {
            performanceMetrics: {
                description: "Track key performance indicators",
                metrics: [
                    "execution_time",
                    "resource_usage",
                    "success_rate",
                    "cost_per_execution"
                ],
                aggregationWindows: ["5m", "1h", "24h", "7d"]
            },
            bottleneckDetection: {
                description: "Identify performance bottlenecks",
                analysisTypes: [
                    "step_duration_analysis",
                    "resource_contention",
                    "dependency_delays",
                    "queue_depth"
                ],
                thresholds: {
                    slowStepThreshold: "2x_average",
                    resourceContentionThreshold: 0.8,
                    queueDepthThreshold: 100
                }
            }
        },
        learningPatterns: {
            performancePatterns: "Learn typical performance patterns for different routine types",
            degradationDetection: "Identify gradual performance degradation",
            optimizationOpportunities: "Discover patterns that indicate optimization potential"
        }
    },
    
    // Deterministic Strategy Agent
    DETERMINISTIC_STRATEGY_AGENT: {
        agentId: "deterministic_strategy_detector",
        name: "Deterministic Strategy Agent",
        goal: "Identify routines that can be converted to deterministic strategies",
        initialRoutine: TEST_ROUTINES.ANALYZE_STRATEGY_PERFORMANCE.id,
        subscriptions: [
            "routine/completed",
            "output/generated",
            "pattern/detected",
            "execution/trace"
        ],
        priority: 8,
        transformationRules: {
            patternDetection: {
                description: "Detect deterministic patterns in routine execution",
                patterns: [
                    "consistent_output_for_input",
                    "fixed_execution_path",
                    "predictable_resource_usage",
                    "no_external_dependencies"
                ],
                confidenceThreshold: 0.85,
                minimumSamples: 100
            },
            deterministicConversion: {
                description: "Rules for converting to deterministic strategy",
                conversionCriteria: [
                    "pattern_confidence > 0.85",
                    "execution_variance < 0.1",
                    "no_random_elements",
                    "cacheable_results"
                ],
                optimizations: [
                    "result_caching",
                    "precomputation",
                    "lookup_tables",
                    "decision_trees"
                ]
            }
        },
        learningPatterns: {
            inputOutputMapping: "Learn stable input-output relationships",
            executionPathAnalysis: "Identify fixed execution paths",
            cacheabilityAssessment: "Determine which results can be cached"
        }
    },
    
    // Cost Optimization Agent
    COST_OPTIMIZATION_AGENT: {
        agentId: "cost_optimization_specialist",
        name: "Cost Optimization Agent",
        goal: "Reduce operational costs while maintaining quality",
        initialRoutine: TEST_ROUTINES.ANALYZE_COST_PATTERNS.id,
        subscriptions: [
            "billing/*",
            "resource_usage/*",
            "api_calls/*",
            "model_usage/*"
        ],
        priority: 7,
        costOptimization: {
            costAnalysis: {
                description: "Analyze cost drivers and patterns",
                categories: [
                    "compute_costs",
                    "storage_costs",
                    "api_costs",
                    "model_inference_costs"
                ],
                granularity: ["per_routine", "per_step", "per_resource"]
            },
            optimizationStrategies: {
                description: "Cost reduction strategies",
                strategies: [
                    "model_downsizing",
                    "batch_processing",
                    "caching_expansion",
                    "off_peak_scheduling"
                ],
                constraints: [
                    "maintain_sla",
                    "preserve_quality",
                    "ensure_reliability"
                ]
            }
        },
        learningPatterns: {
            costPatterns: "Learn cost patterns across different workloads",
            efficiencyMetrics: "Track cost-per-value delivered",
            optimizationImpact: "Measure impact of cost optimizations"
        }
    },
    
    // Evolution Learning Agent
    EVOLUTION_LEARNING_AGENT: {
        agentId: "evolution_pattern_learner",
        name: "Evolution Learning Agent",
        goal: "Learn from successful and failed strategy evolutions",
        initialRoutine: TEST_ROUTINES.ANALYZE_STRATEGY_PERFORMANCE.id,
        subscriptions: [
            "evolution/*",
            "strategy_change/*",
            "performance_delta/*",
            "rollback/*"
        ],
        priority: 8,
        evolutionTracking: {
            evolutionHistory: {
                description: "Track all strategy evolutions",
                tracked: [
                    "evolution_timestamp",
                    "before_after_metrics",
                    "evolution_reason",
                    "success_failure"
                ],
                analysisWindow: "90d"
            },
            successFactors: {
                description: "Identify what makes evolutions successful",
                factors: [
                    "gradual_vs_sudden_change",
                    "testing_coverage",
                    "rollback_capability",
                    "monitoring_adequacy"
                ],
                correlationAnalysis: true
            }
        },
        learningPatterns: {
            evolutionSuccess: "Learn what makes strategy evolutions successful",
            failurePatterns: "Identify common evolution failure patterns",
            optimalTiming: "Learn when to trigger evolutions"
        }
    },
    
    // Financial Optimization Agent (from examples)
    FINANCIAL_OPTIMIZATION_AGENT: {
        agentId: "financial_routine_optimizer",
        name: "Financial Routine Optimization Agent",
        goal: "Optimize financial analysis routines for speed and accuracy",
        initialRoutine: TEST_ROUTINES.ANALYZE_STRATEGY_PERFORMANCE.id,
        subscriptions: [
            "financial/analysis/*",
            "market_data/*",
            "execution/financial/*",
            "accuracy/financial/*"
        ],
        priority: 8,
        strategyEvolution: {
            currentPerformance: {
                executionTime: 750,
                accuracy: 0.92,
                cost: 0.75,
                strategy: "reasoning"
            },
            evolutionPath: {
                description: "Path from reasoning to hybrid deterministic",
                stages: [
                    {
                        stage: "reasoning",
                        characteristics: ["flexible", "slow", "expensive"]
                    },
                    {
                        stage: "partial_deterministic",
                        characteristics: ["common_patterns_cached", "faster", "cheaper"]
                    },
                    {
                        stage: "hybrid_deterministic",
                        characteristics: ["smart_routing", "optimal_performance", "high_accuracy"]
                    }
                ]
            }
        },
        learningPatterns: {
            marketPatterns: "Learn which market conditions suit which strategies",
            accuracyTracking: "Track accuracy across different strategies",
            costBenefitAnalysis: "Continuously evaluate cost vs benefit"
        }
    }
};

/**
 * QUALITY AGENTS - Inferred from patterns
 */
export const QUALITY_AGENTS: Record<string, ExtendedAgentConfig> = {
    // Output Quality Monitor
    OUTPUT_QUALITY_MONITOR: {
        agentId: "output_quality_monitor",
        name: "Output Quality Monitor",
        goal: "Monitor and improve output quality across all routines",
        initialRoutine: TEST_ROUTINES.ASSESS_OUTPUT_QUALITY.id,
        subscriptions: [
            "output/generated",
            "content/created",
            "validation/failed",
            "quality/score"
        ],
        priority: 8,
        capabilities: {
            qualityAssessment: {
                description: "Assess quality of generated outputs",
                metrics: [
                    "accuracy_score",
                    "completeness_score",
                    "consistency_score",
                    "bias_score"
                ],
                thresholds: {
                    minimum_acceptable: 0.7,
                    target: 0.85,
                    excellent: 0.95
                }
            },
            biasDetection: {
                description: "Detect various forms of bias",
                biasTypes: [
                    "demographic_bias",
                    "cultural_bias",
                    "linguistic_bias",
                    "selection_bias"
                ],
                mitigationStrategies: [
                    "diverse_training_data",
                    "bias_correction_algorithms",
                    "human_review_flagging"
                ]
            }
        },
        learningPatterns: {
            qualityTrends: "Track quality trends over time",
            biasPatterns: "Learn to identify subtle bias patterns",
            improvementCorrelation: "Correlate quality improvements with changes"
        }
    }
};

/**
 * MONITORING AGENTS - Inferred from patterns
 */
export const MONITORING_AGENTS: Record<string, ExtendedAgentConfig> = {
    // System Health Monitor
    SYSTEM_HEALTH_MONITOR: {
        agentId: "system_health_monitor",
        name: "System Health Monitor",
        goal: "Monitor overall system health and predict issues",
        initialRoutine: TEST_ROUTINES.ANALYZE_PERFORMANCE_PATTERNS.id,
        subscriptions: [
            "system/health/*",
            "resource/usage/*",
            "service/status/*",
            "alert/*"
        ],
        priority: 9,
        capabilities: {
            healthMetrics: {
                description: "Track system health indicators",
                metrics: [
                    "cpu_usage",
                    "memory_usage",
                    "disk_io",
                    "network_latency",
                    "service_availability"
                ],
                aggregations: ["min", "max", "avg", "p95", "p99"]
            },
            anomalyDetection: {
                description: "Detect anomalies in system behavior",
                methods: [
                    "statistical_deviation",
                    "pattern_matching",
                    "predictive_modeling",
                    "correlation_analysis"
                ],
                alertThresholds: {
                    warning: 0.7,
                    critical: 0.9,
                    emergency: 0.95
                }
            }
        },
        learningPatterns: {
            normalBehavior: "Learn normal system behavior patterns",
            anomalyPrediction: "Predict anomalies before they occur",
            correlationLearning: "Learn correlations between metrics"
        }
    }
};

/**
 * ROUTING COORDINATION AGENTS - Intelligent multi-routine coordination
 */
export const ROUTING_COORDINATION_AGENTS: Record<string, ExtendedAgentConfig> = {
    // Workflow Coordination Agent
    WORKFLOW_COORDINATION_AGENT: {
        agentId: "workflow_coordination_specialist",
        name: "Workflow Coordination Specialist",
        goal: "Identify opportunities for multi-routine coordination and propose routing strategies",
        initialRoutine: "analyze_workflow_patterns",
        subscriptions: [
            "routine/completed",
            "routine/sequence_detected",
            "performance/workflow_metrics",
            "coordination/opportunity"
        ],
        priority: 8,
        capabilities: {
            workflowAnalysis: {
                description: "Analyze execution patterns for coordination opportunities",
                patterns: [
                    "sequential_redundancy",
                    "parallelization_potential",
                    "conditional_branching_needs",
                    "resource_sharing_opportunities"
                ],
                minimumExecutions: 50,
                confidenceThreshold: 0.8
            },
            routingProposal: {
                description: "Propose intelligent routing strategies",
                strategies: [
                    "scatter_gather",
                    "pipeline",
                    "conditional_routing",
                    "hybrid_coordination"
                ],
                optimizationGoals: [
                    "minimize_total_time",
                    "maximize_parallelization",
                    "optimize_resource_usage",
                    "improve_quality_through_specialization"
                ]
            },
            contextPropagation: {
                description: "Manage context sharing between sub-routines",
                strategies: {
                    full: "Share all context data",
                    selective: "Share only relevant context",
                    isolated: "No context sharing",
                    filtered: "Share with transformations"
                },
                optimization: [
                    "minimize_data_transfer",
                    "maintain_privacy",
                    "ensure_consistency"
                ]
            }
        },
        learningPatterns: {
            executionPatterns: "Learn common workflow patterns",
            parallelizationSuccess: "Track successful parallel executions",
            contextRequirements: "Learn context dependencies between routines"
        }
    },
    
    // Parallel Optimization Agent
    PARALLEL_OPTIMIZATION_AGENT: {
        agentId: "parallel_optimization_specialist",
        name: "Parallel Execution Optimizer",
        goal: "Optimize parallel execution of independent sub-routines",
        initialRoutine: "analyze_parallelization_opportunities",
        subscriptions: [
            "routine/dependency_analysis",
            "resource/utilization",
            "performance/parallel_metrics",
            "execution/bottleneck"
        ],
        priority: 7,
        capabilities: {
            dependencyAnalysis: {
                description: "Analyze dependencies between routines",
                methods: [
                    "data_dependency_graph",
                    "resource_conflict_detection",
                    "temporal_dependency_mapping",
                    "side_effect_analysis"
                ],
                outputFormat: "directed_acyclic_graph"
            },
            parallelizationStrategy: {
                description: "Determine optimal parallelization approach",
                factors: [
                    "available_resources",
                    "data_locality",
                    "communication_overhead",
                    "load_balancing"
                ],
                strategies: {
                    static: "Pre-determined parallel groups",
                    dynamic: "Runtime-adjusted parallelism",
                    adaptive: "Learning-based parallelization"
                }
            },
            resourceAllocation: {
                description: "Allocate resources for parallel execution",
                resources: ["cpu", "memory", "network", "api_quota"],
                allocationStrategies: [
                    "fair_share",
                    "priority_based",
                    "performance_optimized",
                    "cost_optimized"
                ]
            }
        },
        learningPatterns: {
            parallelizationGains: "Learn actual vs predicted speedup",
            resourceContention: "Identify resource bottlenecks",
            optimalConcurrency: "Learn optimal parallelism levels"
        }
    },
    
    // Context Propagation Agent
    CONTEXT_PROPAGATION_AGENT: {
        agentId: "context_propagation_specialist",
        name: "Context Management Specialist",
        goal: "Optimize context sharing and transformation in multi-routine workflows",
        initialRoutine: "analyze_context_flow",
        subscriptions: [
            "context/created",
            "context/accessed",
            "context/transformed",
            "privacy/context_concern"
        ],
        priority: 6,
        capabilities: {
            contextAnalysis: {
                description: "Analyze context usage patterns",
                metrics: [
                    "access_frequency",
                    "modification_patterns",
                    "size_distribution",
                    "sensitivity_levels"
                ],
                privacyAware: true
            },
            transformationOptimization: {
                description: "Optimize context transformations",
                strategies: [
                    "lazy_transformation",
                    "cached_transformations",
                    "incremental_updates",
                    "schema_evolution"
                ],
                validationRequired: true
            },
            sharingStrategy: {
                description: "Determine optimal context sharing approach",
                methods: {
                    broadcast: "Share with all sub-routines",
                    unicast: "Share with specific routines",
                    multicast: "Share with groups",
                    publish_subscribe: "Event-based sharing"
                },
                considerations: [
                    "data_volume",
                    "update_frequency",
                    "consistency_requirements",
                    "privacy_constraints"
                ]
            }
        },
        learningPatterns: {
            contextUsage: "Learn which context is used by which routines",
            transformationPatterns: "Identify common transformations",
            sharingEfficiency: "Optimize sharing based on usage"
        }
    },
    
    // Routing Performance Agent
    ROUTING_PERFORMANCE_AGENT: {
        agentId: "routing_performance_optimizer",
        name: "Routing Performance Optimizer",
        goal: "Monitor and optimize routing strategy performance",
        initialRoutine: "analyze_routing_performance",
        subscriptions: [
            "routing/execution_complete",
            "performance/routing_metrics",
            "resource/routing_usage",
            "quality/routing_outcomes"
        ],
        priority: 8,
        capabilities: {
            performanceAnalysis: {
                description: "Analyze routing performance metrics",
                metrics: [
                    "total_execution_time",
                    "parallelization_efficiency",
                    "resource_utilization",
                    "quality_scores",
                    "error_rates"
                ],
                comparisons: [
                    "routing_vs_sequential",
                    "different_routing_strategies",
                    "predicted_vs_actual"
                ]
            },
            bottleneckDetection: {
                description: "Identify routing bottlenecks",
                types: [
                    "synchronization_points",
                    "resource_contention",
                    "data_transfer_overhead",
                    "sequential_dependencies"
                ],
                resolutionStrategies: [
                    "increase_parallelism",
                    "optimize_data_flow",
                    "reorder_operations",
                    "cache_intermediate_results"
                ]
            },
            evolutionProposal: {
                description: "Propose routing improvements",
                basedOn: [
                    "performance_history",
                    "pattern_analysis",
                    "cost_benefit_analysis",
                    "risk_assessment"
                ],
                proposalTypes: [
                    "strategy_change",
                    "parallelism_adjustment",
                    "resource_reallocation",
                    "workflow_restructuring"
                ]
            }
        },
        learningPatterns: {
            performanceTrends: "Track routing performance over time",
            strategyEffectiveness: "Learn which strategies work best",
            adaptiveOptimization: "Continuously improve routing decisions"
        }
    }
};

/**
 * API BOOTSTRAPPING AGENTS - Create new API integrations
 */
export const API_BOOTSTRAPPING_AGENTS: Record<string, ExtendedAgentConfig> = {
    // API Integration Creator
    API_INTEGRATION_CREATOR: {
        agentId: "api_integration_creator",
        name: "API Integration Bootstrap Agent",
        goal: "Discover and create new API integrations through routine composition",
        initialRoutine: "analyze_api_documentation",
        subscriptions: [
            "integration/request",
            "api/documentation_found",
            "integration/test_results",
            "routine/api_usage"
        ],
        priority: 7,
        capabilities: {
            apiDiscovery: {
                description: "Discover and analyze new API endpoints",
                methods: [
                    "openapi_spec_parsing",
                    "documentation_scraping",
                    "example_analysis",
                    "schema_inference"
                ],
                supportedFormats: ["OpenAPI", "GraphQL", "REST", "SOAP"]
            },
            integrationGeneration: {
                description: "Generate integration routines from API specs",
                outputs: [
                    "authentication_routine",
                    "endpoint_wrapper_routines",
                    "error_handling_routine",
                    "rate_limiting_routine"
                ],
                testingStrategy: "progressive_validation"
            },
            qualityAssurance: {
                description: "Ensure generated integrations work correctly",
                validations: [
                    "schema_compliance",
                    "error_handling_coverage",
                    "performance_benchmarks",
                    "security_validation"
                ]
            }
        },
        learningPatterns: {
            apiPatterns: "Learn common API design patterns",
            errorPatterns: "Track API-specific error patterns",
            optimizationOpportunities: "Identify performance improvements"
        }
    },
    
    // API Test Generator
    API_TEST_GENERATOR: {
        agentId: "api_test_generator",
        name: "API Test Suite Generator",
        goal: "Generate comprehensive test suites for API integrations",
        initialRoutine: "analyze_api_behavior",
        subscriptions: [
            "api/integration_created",
            "api/test_request",
            "test/results",
            "api/failure_detected"
        ],
        priority: 6,
        capabilities: {
            testGeneration: {
                description: "Generate various types of tests",
                testTypes: [
                    "unit_tests",
                    "integration_tests",
                    "contract_tests",
                    "performance_tests",
                    "security_tests"
                ],
                coverage: {
                    endpoints: 1.0,
                    error_cases: 0.9,
                    edge_cases: 0.8,
                    performance_scenarios: 0.7
                }
            },
            mockGeneration: {
                description: "Generate API mocks for testing",
                strategies: [
                    "response_recording",
                    "schema_based_generation",
                    "behavior_simulation",
                    "error_injection"
                ]
            },
            continuousTesting: {
                description: "Monitor and test APIs continuously",
                monitoring: [
                    "availability_checks",
                    "response_time_tracking",
                    "schema_compliance",
                    "breaking_change_detection"
                ]
            }
        },
        learningPatterns: {
            testEffectiveness: "Learn which tests catch most issues",
            apiEvolution: "Track API changes over time",
            testOptimization: "Optimize test execution"
        }
    }
};

/**
 * DATA BOOTSTRAPPING AGENTS - Document and data management
 */
export const DATA_BOOTSTRAPPING_AGENTS: Record<string, ExtendedAgentConfig> = {
    // Document Creator Agent
    DOCUMENT_CREATOR_AGENT: {
        agentId: "intelligent_document_creator",
        name: "Document Bootstrap Agent",
        goal: "Create and manage documents across external storage providers",
        initialRoutine: "analyze_document_requirements",
        subscriptions: [
            "document/create_request",
            "storage/provider_available",
            "document/collaboration_needed",
            "format/optimization_opportunity"
        ],
        priority: 7,
        capabilities: {
            formatSelection: {
                description: "Choose optimal format based on content and audience",
                strategies: {
                    presentation: ["pptx", "google_slides", "pdf"],
                    documentation: ["docx", "markdown", "confluence"],
                    data_analysis: ["xlsx", "jupyter", "tableau"],
                    collaboration: ["google_docs", "sharepoint", "notion"]
                },
                decisionFactors: [
                    "audience_technical_level",
                    "collaboration_requirements",
                    "visual_complexity",
                    "update_frequency"
                ]
            },
            storageIntegration: {
                description: "Integrate with external storage providers",
                providers: {
                    google_drive: {
                        auth: "oauth2",
                        capabilities: ["real_time_collab", "version_history", "commenting"]
                    },
                    sharepoint: {
                        auth: "oauth2",
                        capabilities: ["enterprise_permissions", "workflows", "metadata"]
                    },
                    github: {
                        auth: "token",
                        capabilities: ["version_control", "pull_requests", "ci_cd"]
                    }
                }
            },
            contentGeneration: {
                description: "Generate document content intelligently",
                features: [
                    "data_visualization",
                    "narrative_generation",
                    "template_adaptation",
                    "multi_language_support"
                ]
            }
        },
        learningPatterns: {
            audiencePreferences: "Learn format preferences by audience",
            contentEffectiveness: "Track engagement and comprehension",
            collaborationPatterns: "Optimize for team workflows"
        }
    },
    
    // Format Optimizer Agent
    FORMAT_OPTIMIZER_AGENT: {
        agentId: "format_optimization_specialist",
        name: "Document Format Optimizer",
        goal: "Optimize document formats for maximum effectiveness",
        initialRoutine: "analyze_format_performance",
        subscriptions: [
            "document/created",
            "document/accessed",
            "feedback/document",
            "conversion/request"
        ],
        priority: 6,
        capabilities: {
            formatAnalysis: {
                description: "Analyze format effectiveness",
                metrics: [
                    "readability_score",
                    "engagement_time",
                    "completion_rate",
                    "share_frequency"
                ],
                comparisons: [
                    "format_performance",
                    "audience_preferences",
                    "device_compatibility"
                ]
            },
            conversionOptimization: {
                description: "Optimize format conversions",
                preserves: [
                    "formatting",
                    "interactive_elements",
                    "metadata",
                    "accessibility_features"
                ],
                enhances: [
                    "searchability",
                    "performance",
                    "compatibility",
                    "file_size"
                ]
            },
            adaptiveFormatting: {
                description: "Adapt formatting based on context",
                factors: [
                    "device_type",
                    "network_speed",
                    "user_preferences",
                    "accessibility_needs"
                ]
            }
        },
        learningPatterns: {
            formatSuccess: "Learn which formats work best",
            conversionPatterns: "Track common conversion needs",
            audienceAdaptation: "Optimize for different audiences"
        }
    }
};

/**
 * SWARM CONFIGURATIONS
 */
export const TEST_SWARMS: Record<string, EmergentSwarmConfig> = {
    // Healthcare Security Swarm
    HEALTHCARE_SECURITY_SWARM: {
        swarmId: "healthcare_security_swarm",
        name: "Healthcare Security Ecosystem",
        description: "Comprehensive healthcare security and compliance monitoring",
        agents: [
            {
                ...SECURITY_AGENTS.HIPAA_COMPLIANCE_AGENT,
                agentId: "healthcare_hipaa_monitor"
            },
            {
                ...SECURITY_AGENTS.MEDICAL_SAFETY_AGENT,
                agentId: "healthcare_safety_monitor"
            },
            {
                ...SECURITY_AGENTS.DATA_PRIVACY_AGENT,
                agentId: "healthcare_privacy_monitor"
            }
        ],
        coordination: {
            sharedLearning: true,
            collaborativeProposals: true,
            crossAgentInsights: true
        }
    },
    
    // Financial Security Swarm
    FINANCIAL_SECURITY_SWARM: {
        swarmId: "financial_security_swarm",
        name: "Financial Security Swarm",
        description: "Comprehensive financial security and compliance monitoring",
        agents: [
            {
                ...SECURITY_AGENTS.TRADING_FRAUD_AGENT,
                agentId: "financial_fraud_detector"
            },
            {
                ...SECURITY_AGENTS.FINANCIAL_COMPLIANCE_AGENT,
                agentId: "financial_compliance_monitor"
            },
            {
                ...STRATEGY_EVOLUTION_AGENTS.FINANCIAL_OPTIMIZATION_AGENT,
                agentId: "financial_optimizer"
            }
        ],
        coordination: {
            sharedLearning: true,
            collaborativeProposals: false, // Financial decisions need isolation
            crossAgentInsights: true
        }
    },
    
    // Resilience Evolution Swarm
    RESILIENCE_EVOLUTION_SWARM: {
        swarmId: "resilience_evolution_swarm",
        name: "Resilience Evolution Swarm",
        description: "Adaptive resilience and recovery optimization",
        agents: [
            {
                ...RESILIENCE_AGENTS.PATTERN_LEARNING_AGENT,
                agentId: "resilience_pattern_learner"
            },
            {
                ...RESILIENCE_AGENTS.THRESHOLD_OPTIMIZATION_AGENT,
                agentId: "resilience_threshold_optimizer"
            },
            {
                ...RESILIENCE_AGENTS.PREDICTIVE_FAILURE_AGENT,
                agentId: "resilience_failure_predictor"
            },
            {
                ...RESILIENCE_AGENTS.RECOVERY_STRATEGY_EVOLUTION_AGENT,
                agentId: "resilience_recovery_evolver"
            }
        ],
        coordination: {
            sharedLearning: true,
            collaborativeProposals: true,
            crossAgentInsights: true
        }
    },
    
    // Strategy Optimization Swarm
    STRATEGY_OPTIMIZATION_SWARM: {
        swarmId: "strategy_optimization_swarm",
        name: "Strategy Optimization Swarm",
        description: "Continuous strategy evolution and optimization",
        agents: [
            {
                ...STRATEGY_EVOLUTION_AGENTS.ROUTINE_PERFORMANCE_AGENT,
                agentId: "strategy_performance_analyzer"
            },
            {
                ...STRATEGY_EVOLUTION_AGENTS.DETERMINISTIC_STRATEGY_AGENT,
                agentId: "strategy_deterministic_detector"
            },
            {
                ...STRATEGY_EVOLUTION_AGENTS.COST_OPTIMIZATION_AGENT,
                agentId: "strategy_cost_optimizer"
            },
            {
                ...STRATEGY_EVOLUTION_AGENTS.EVOLUTION_LEARNING_AGENT,
                agentId: "strategy_evolution_learner"
            }
        ],
        coordination: {
            sharedLearning: true,
            collaborativeProposals: true,
            crossAgentInsights: true
        }
    }
};

/**
 * Sample events that trigger agent learning and improvement proposals
 */
export const TEST_LEARNING_EVENTS: Record<string, IntelligentEvent> = {
    // Performance degradation event
    PERFORMANCE_DEGRADATION: {
        id: "perf_degradation_001",
        type: "execution/metrics",
        timestamp: new Date(),
        source: "tier3_executor",
        tier: 3,
        category: "performance",
        subcategory: "degradation",
        deliveryGuarantee: "reliable" as const,
        priority: "high" as const,
        data: {
            routineId: "customer_support_routine",
            executionTime: 2500, // Slow
            memoryUsage: 512000000, // High
            cpuUsage: 0.85, // High
            cost: 0.25,
            previousExecutionTime: 800, // Much faster before
            baseline: {
                averageExecutionTime: 900,
                averageMemoryUsage: 256000000,
                averageCpuUsage: 0.45,
            },
        },
    },
    
    // Quality issue event
    QUALITY_ISSUE: {
        id: "quality_issue_001",
        type: "output/generated",
        timestamp: new Date(),
        source: "conversational_strategy",
        tier: 3,
        category: "quality",
        subcategory: "accuracy",
        deliveryGuarantee: "reliable" as const,
        priority: "medium" as const,
        data: {
            routineId: "content_generation_routine",
            output: "Generated content with potential bias and factual errors",
            qualityScore: 0.65, // Low
            accuracyScore: 0.70,
            biasScore: 0.45, // High bias (lower is better)
            completenessScore: 0.80,
            expectedQuality: 0.90,
        },
    },
    
    // Security threat event
    SECURITY_THREAT: {
        id: "security_threat_001",
        type: "security/threat/detected",
        timestamp: new Date(),
        source: "api_gateway",
        tier: 1,
        category: "security",
        subcategory: "threat",
        deliveryGuarantee: "barrier_sync" as const,
        priority: "emergency" as const,
        securityLevel: "confidential",
        data: {
            threatType: "injection_attack",
            riskLevel: "high",
            affectedEndpoint: "/api/user/data",
            suspiciousPayload: "'; DROP TABLE users; --",
            sourceIP: "192.168.1.100",
            userAgent: "Suspicious Bot v1.0",
        },
    },
    
    // Cost spike event
    COST_SPIKE: {
        id: "cost_spike_001",
        type: "billing/cost_alert",
        timestamp: new Date(),
        source: "billing_monitor",
        tier: 2,
        category: "cost",
        subcategory: "spike",
        deliveryGuarantee: "reliable" as const,
        priority: "high" as const,
        data: {
            currentCost: 150.50,
            expectedCost: 45.20,
            costIncrease: 2.33, // 233% increase
            breakdown: {
                compute: 85.00,
                storage: 25.50,
                network: 30.00,
                api_calls: 10.00,
            },
            timeWindow: "1h",
        },
    },
    
    // Error pattern event
    ERROR_PATTERN: {
        id: "error_pattern_001", 
        type: "error/recurring",
        timestamp: new Date(),
        source: "error_aggregator",
        tier: 2,
        category: "error",
        subcategory: "pattern",
        deliveryGuarantee: "reliable" as const,
        priority: "medium" as const,
        data: {
            errorType: "timeout",
            errorMessage: "Request timeout after 30000ms",
            occurrences: 47,
            timeWindow: "2h",
            affectedRoutines: ["data_analysis_routine", "report_generation_routine"],
            correlatedEvents: ["high_cpu_usage", "memory_pressure"],
        },
    },
    
    // Compliance violation event
    COMPLIANCE_VIOLATION: {
        id: "compliance_violation_001",
        type: "security/compliance/violation",
        timestamp: new Date(),
        source: "compliance_monitor",
        tier: 1,
        category: "security",
        subcategory: "compliance",
        deliveryGuarantee: "barrier_sync" as const,
        priority: "critical" as const,
        securityLevel: "secret",
        complianceRequired: true,
        data: {
            violationType: "data_retention",
            framework: "GDPR",
            description: "Personal data retained beyond required period",
            affectedRecords: 1247,
            dataTypes: ["email", "profile_data", "activity_logs"],
            retentionPeriod: "2 years",
            actualRetention: "3.5 years",
        },
    },
    
    // Successful optimization event (for positive learning)
    SUCCESSFUL_OPTIMIZATION: {
        id: "optimization_success_001",
        type: "routine/completed",
        timestamp: new Date(),
        source: "deterministic_strategy",
        tier: 3,
        category: "routine",
        subcategory: "completion",
        deliveryGuarantee: "fire-and-forget" as const,
        priority: "low" as const,
        data: {
            routineId: "optimized_data_processing",
            executionTime: 450, // Much faster
            previousExecutionTime: 1200,
            improvement: 0.625, // 62.5% improvement
            cost: 0.08,
            previousCost: 0.18,
            costSavings: 0.55, // 55% cost reduction
            quality: 0.95, // Maintained high quality
            strategy: "deterministic",
            optimizationApplied: "caching_and_batching",
        },
    },
    
    // Resilience failure event
    RESILIENCE_FAILURE: {
        id: "resilience_failure_001",
        type: "failure/circuit_breaker",
        timestamp: new Date(),
        source: "tier2_orchestrator",
        tier: 2,
        category: "failure",
        subcategory: "circuit_breaker",
        deliveryGuarantee: "reliable" as const,
        priority: "high" as const,
        data: {
            serviceId: "payment_service",
            failureType: "timeout",
            failureCount: 15,
            timeWindow: "5m",
            circuitState: "open",
            lastSuccessTime: new Date(Date.now() - 300000),
        },
    },
    
    // Strategy evolution event
    STRATEGY_EVOLUTION: {
        id: "strategy_evolution_001",
        type: "evolution/proposed",
        timestamp: new Date(),
        source: "strategy_evolution_agent",
        tier: 1,
        category: "evolution",
        subcategory: "strategy",
        deliveryGuarantee: "reliable" as const,
        priority: "medium" as const,
        data: {
            routineId: "data_processing_routine",
            currentStrategy: "reasoning",
            proposedStrategy: "deterministic",
            confidence: 0.87,
            expectedImprovement: {
                performance: 0.65,
                cost: 0.72,
                quality: 0.05,
            },
        },
    },
};

/**
 * Helper function to create test events for agent learning
 */
export function createTestEvent(
    type: string,
    tier: 1 | 2 | 3,
    category: string,
    data: any,
    options?: Partial<IntelligentEvent>,
): IntelligentEvent {
    return {
        id: `test_event_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
        type,
        timestamp: new Date(),
        source: `test_${tier === 1 ? "coordinator" : tier === 2 ? "orchestrator" : "executor"}`,
        tier,
        category,
        deliveryGuarantee: "fire-and-forget",
        priority: "medium",
        data,
        ...options,
    };
}

/**
 * Helper function to create test agent configuration
 */
export function createTestAgentConfig(
    agentId: string,
    goal: string,
    initialRoutine: string,
    subscriptions: string[],
    options?: Partial<ExtendedAgentConfig>,
): ExtendedAgentConfig {
    return {
        agentId,
        name: options?.name || agentId,
        goal,
        initialRoutine,
        subscriptions,
        priority: 5,
        ...options,
    };
}

/**
 * Helper function to create test swarm configuration
 */
export function createTestSwarmConfig(
    swarmId: string,
    name: string,
    agents: Omit<ExtendedAgentConfig, "swarmId">[],
    options?: Partial<EmergentSwarmConfig>,
): EmergentSwarmConfig {
    return {
        swarmId,
        name,
        description: `Test swarm: ${name}`,
        agents,
        coordination: {
            sharedLearning: true,
            collaborativeProposals: true,
            crossAgentInsights: true,
        },
        ...options,
    };
}

/**
 * Get all agents as a flat array for easy access
 */
export function getAllAgents(): ExtendedAgentConfig[] {
    return [
        ...Object.values(SECURITY_AGENTS),
        ...Object.values(RESILIENCE_AGENTS),
        ...Object.values(STRATEGY_EVOLUTION_AGENTS),
        ...Object.values(QUALITY_AGENTS),
        ...Object.values(MONITORING_AGENTS),
        ...Object.values(ROUTING_COORDINATION_AGENTS),
        ...Object.values(API_BOOTSTRAPPING_AGENTS),
        ...Object.values(DATA_BOOTSTRAPPING_AGENTS),
    ];
}

/**
 * Get agent by ID
 */
export function getAgentById(agentId: string): ExtendedAgentConfig | undefined {
    return getAllAgents().find(agent => agent.agentId === agentId);
}

/**
 * Get agents by capability type
 */
export function getAgentsByCapability(capability: string): ExtendedAgentConfig[] {
    return getAllAgents().filter(agent => 
        Object.keys({
            ...agent.capabilities,
            ...agent.learningPatterns,
            ...agent.resilienceCapabilities,
            ...agent.strategyEvolution,
            ...agent.fraudDetection,
            ...agent.threatDetection,
        } as any).some(key => key.toLowerCase().includes(capability.toLowerCase()))
    );
}