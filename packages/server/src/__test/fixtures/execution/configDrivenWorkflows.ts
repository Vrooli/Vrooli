/**
 * Config-Driven Workflow Examples
 * 
 * Demonstrates Vrooli's core principle: complex workflows and intelligent behaviors
 * emerge from simple configuration objects, not hard-coded logic.
 * 
 * These examples prove that sophisticated AI capabilities can be described
 * entirely through configuration, enabling true emergent intelligence.
 */

import { TEST_IDS, TestIdFactory } from "./testIdGenerator.js";

/**
 * Config-Driven Workflow Interface
 * Pure configuration that drives intelligent behavior
 */
export interface ConfigDrivenWorkflow {
    id: string;
    name: string;
    description: string;
    
    // Core configuration - no code, just declarative rules
    config: {
        trigger: TriggerConfig;
        stages: StageConfig[];
        emergentCapabilities: string[];
        learningEnabled: boolean;
        adaptationRules: AdaptationRule[];
    };
    
    // What emerges from this simple config
    emergentBehaviors: EmergentBehavior[];
    
    // Performance evolution over time
    evolutionTimeline: EvolutionPoint[];
}

export interface TriggerConfig {
    type: "form_submission" | "api_call" | "schedule" | "event_pattern" | "threshold_breach";
    pattern?: string;
    schedule?: string;
    conditions?: Record<string, unknown>;
}

export interface StageConfig {
    name: string;
    agent?: string;
    strategy?: "conversational" | "reasoning" | "deterministic" | "routing";
    parallel?: boolean;
    condition?: string;
    config?: Record<string, unknown>;
    rules?: Record<string, unknown>;
    outputSchema?: Record<string, unknown>;
    promptTemplate?: string;
}

export interface AdaptationRule {
    condition: string;
    adaptation: string;
    learningWeight: number;
}

export interface EmergentBehavior {
    name: string;
    description: string;
    emergesFrom: string[];
    confidence: number;
    timeToEmerge: string;
}

export interface EvolutionPoint {
    time: string;
    capability: string;
    confidence: number;
    description: string;
}

/**
 * Medical Patient Intake Workflow
 * Complex medical workflow defined entirely through configuration
 * NO CODE - just config that creates intelligent behavior
 */
export const PATIENT_INTAKE_WORKFLOW: ConfigDrivenWorkflow = {
    id: "patient_intake_intelligent_v1",
    name: "Intelligent Patient Intake Workflow",
    description: "HIPAA-compliant patient processing with emergent medical intelligence",
    
    config: {
        trigger: {
            type: "form_submission",
            pattern: "patient_registration_form",
            conditions: {
                requiredFields: ["name", "dob", "symptoms", "insurance"],
                privacyConsent: true
            }
        },
        
        stages: [
            {
                name: "data_validation",
                agent: "medical_data_validator",
                strategy: "deterministic",
                rules: {
                    requiredFields: ["name", "dob", "symptoms", "insurance", "emergency_contact"],
                    validationPatterns: {
                        ssn: "^\\d{3}-\\d{2}-\\d{4}$",
                        phone: "^\\d{10}$",
                        email: "^[\\w-\\.]+@([\\w-]+\\.)+[\\w-]{2,4}$"
                    },
                    dataQualityChecks: {
                        dobReasonable: "age_between_0_120",
                        symptomsDescriptive: "min_length_10_chars",
                        insuranceValid: "verify_provider_network"
                    }
                }
            },
            {
                name: "hipaa_compliance_scan",
                agent: "hipaa_compliance_monitor",
                strategy: "reasoning",
                parallel: true, // Runs alongside other stages
                config: {
                    scanForPHI: true,
                    auditTrail: true,
                    encryptionRequired: true,
                    accessLogging: "detailed",
                    retentionPolicy: "7_years_medical_records"
                }
            },
            {
                name: "symptom_analysis",
                strategy: "reasoning",
                promptTemplate: `
                    Analyze patient symptoms for medical intake:
                    Patient: {{name}}, Age: {{age}}
                    Symptoms: {{symptoms}}
                    Medical History: {{medical_history}}
                    
                    Determine:
                    1. Urgency level (emergency, urgent, routine, preventive)
                    2. Recommended specialist type
                    3. Initial diagnostic tests needed
                    4. Red flags requiring immediate attention
                `,
                outputSchema: {
                    urgency: "enum:emergency,urgent,routine,preventive",
                    recommendedSpecialist: "string",
                    initialTests: "array:string",
                    redFlags: "array:string",
                    confidence: "number:0-1",
                    reasoning: "string"
                }
            },
            {
                name: "insurance_verification",
                agent: "insurance_specialist",
                strategy: "deterministic",
                parallel: true,
                config: {
                    verifyEligibility: true,
                    checkCoverage: ["office_visits", "diagnostics", "specialist_referrals"],
                    estimateCosts: true,
                    priorAuthRequired: "check_procedure_codes"
                }
            },
            {
                name: "appointment_scheduling",
                condition: "urgency !== 'emergency'",
                agent: "scheduling_coordinator",
                strategy: "reasoning",
                config: {
                    findOptimalSlots: {
                        specialist: "{{recommendedSpecialist}}",
                        urgencyWeight: "{{urgency}}",
                        patientPreferences: "{{preferences}}",
                        providerAvailability: "realtime_check",
                        insurance: "{{insurance_verification_result}}"
                    },
                    optimizationCriteria: [
                        "earliest_available_time",
                        "patient_convenience",
                        "provider_expertise_match",
                        "cost_efficiency"
                    ]
                }
            },
            {
                name: "emergency_protocol",
                condition: "urgency === 'emergency' || red_flags.length > 0",
                agent: "emergency_coordinator",
                strategy: "deterministic",
                config: {
                    immediateActions: [
                        "notify_emergency_department",
                        "prepare_emergency_record",
                        "alert_on_call_physician",
                        "coordinate_ambulance_if_needed"
                    ],
                    escalationMatrix: {
                        cardiac: "cardiologist_on_call",
                        neurological: "neurologist_on_call",
                        trauma: "trauma_team",
                        pediatric: "pediatric_emergency"
                    }
                }
            },
            {
                name: "care_coordination",
                agent: "care_coordinator",
                strategy: "reasoning",
                config: {
                    synthesizeResults: true,
                    createCarePlan: {
                        includeAllFindings: true,
                        prioritizeByUrgency: true,
                        considerInsuranceLimits: true,
                        patientEducationMaterials: true
                    },
                    communicationPlan: {
                        patientNotification: "automated_secure_message",
                        providerBriefing: "structured_handoff",
                        familyUpdates: "privacy_compliant_summary"
                    }
                }
            }
        ],
        
        emergentCapabilities: [
            "Learns optimal symptom → specialist matching patterns",
            "Develops insurance approval prediction models", 
            "Creates efficient scheduling optimization algorithms",
            "Builds personalized patient communication strategies",
            "Evolves emergency detection accuracy over time",
            "Optimizes resource allocation across the healthcare network"
        ],
        
        learningEnabled: true,
        
        adaptationRules: [
            {
                condition: "appointment_no_shows > 15%",
                adaptation: "improve_reminder_strategy",
                learningWeight: 0.8
            },
            {
                condition: "emergency_false_positives > 5%",
                adaptation: "refine_emergency_detection_criteria",
                learningWeight: 0.9
            },
            {
                condition: "patient_satisfaction < 85%",
                adaptation: "optimize_communication_approach",
                learningWeight: 0.7
            },
            {
                condition: "processing_time > 10_minutes",
                adaptation: "increase_parallelization",
                learningWeight: 0.6
            }
        ]
    },
    
    emergentBehaviors: [
        {
            name: "Predictive Emergency Detection",
            description: "System learns to predict emergencies from subtle symptom patterns",
            emergesFrom: ["symptom_analysis", "emergency_protocol", "historical_outcomes"],
            confidence: 0.0, // Grows over time
            timeToEmerge: "2-3 months with 1000+ cases"
        },
        {
            name: "Optimal Specialist Matching",
            description: "AI learns which specialists achieve best outcomes for specific symptom clusters",
            emergesFrom: ["symptom_analysis", "appointment_scheduling", "outcome_tracking"],
            confidence: 0.0,
            timeToEmerge: "1-2 months with 500+ appointments"
        },
        {
            name: "Insurance Approval Prediction",
            description: "System predicts likelihood of insurance approval for different treatment paths",
            emergesFrom: ["insurance_verification", "care_coordination", "approval_history"],
            confidence: 0.0,
            timeToEmerge: "3-4 months with 2000+ cases"
        },
        {
            name: "Personalized Patient Communication",
            description: "AI adapts communication style based on patient demographics and preferences",
            emergesFrom: ["care_coordination", "patient_feedback", "communication_effectiveness"],
            confidence: 0.0,
            timeToEmerge: "1 month with 200+ interactions"
        }
    ],
    
    evolutionTimeline: [
        { time: "T+0", capability: "Basic intake processing", confidence: 0.7, description: "Initial rule-based processing" },
        { time: "T+2 weeks", capability: "Pattern recognition", confidence: 0.75, description: "Identifies common symptom patterns" },
        { time: "T+1 month", capability: "Communication optimization", confidence: 0.82, description: "Adapts to patient communication preferences" },
        { time: "T+2 months", capability: "Specialist matching intelligence", confidence: 0.88, description: "Learns optimal specialist assignments" },
        { time: "T+3 months", capability: "Emergency prediction", confidence: 0.85, description: "Predicts emergencies from symptom patterns" },
        { time: "T+6 months", capability: "Full healthcare intelligence", confidence: 0.92, description: "Comprehensive healthcare optimization system" }
    ]
};

/**
 * Financial Trading Risk Management
 * Sophisticated trading workflow from pure configuration
 */
export const TRADING_RISK_WORKFLOW: ConfigDrivenWorkflow = {
    id: "trading_risk_management_v1",
    name: "Intelligent Trading Risk Management",
    description: "SEC-compliant trading with emergent risk intelligence",
    
    config: {
        trigger: {
            type: "api_call",
            pattern: "trading/order/submit",
            conditions: {
                marketHours: true,
                validSecurity: true,
                authorizedUser: true
            }
        },
        
        stages: [
            {
                name: "order_validation",
                strategy: "deterministic",
                rules: {
                    basicValidation: {
                        quantity: "positive_integer",
                        price: "positive_number",
                        security: "valid_ticker",
                        action: "enum:BUY,SELL,SHORT"
                    },
                    regulatoryChecks: {
                        dayTradingRules: "check_pdt_status",
                        marginRequirements: "verify_buying_power",
                        settlementRules: "t_plus_2_compliance"
                    }
                }
            },
            {
                name: "market_analysis",
                agent: "market_intelligence_analyzer",
                strategy: "reasoning",
                parallel: true,
                config: {
                    analyzeLiveData: {
                        priceAction: "last_24h_trends",
                        volume: "unusual_volume_detection",
                        volatility: "implied_vs_historical",
                        marketSentiment: "news_sentiment_analysis"
                    },
                    technicalIndicators: [
                        "relative_strength_index",
                        "moving_averages",
                        "bollinger_bands", 
                        "volume_weighted_average_price"
                    ]
                }
            },
            {
                name: "risk_assessment",
                agent: "risk_management_specialist",
                strategy: "reasoning",
                promptTemplate: `
                    Assess trading risk for order:
                    Security: {{ticker}}
                    Action: {{action}}
                    Quantity: {{quantity}}
                    Price: {{price}}
                    
                    Market Context:
                    Current Price: {{current_price}}
                    Volatility: {{volatility}}
                    Volume: {{volume}}
                    Market Sentiment: {{sentiment}}
                    
                    Portfolio Context:
                    Current Position: {{current_position}}
                    Portfolio Value: {{portfolio_value}}
                    Available Cash: {{cash}}
                    
                    Calculate:
                    1. Position size risk (% of portfolio)
                    2. Volatility-adjusted risk
                    3. Correlation risk with existing positions
                    4. Maximum potential loss
                    5. Risk-adjusted return expectation
                `,
                outputSchema: {
                    riskScore: "number:0-100",
                    positionSizeRisk: "number:0-1",
                    maxPotentialLoss: "number",
                    riskAdjustedReturn: "number",
                    recommendedAction: "enum:APPROVE,MODIFY,REJECT",
                    modifications: "array:string",
                    reasoning: "string"
                }
            },
            {
                name: "compliance_check",
                agent: "trading_compliance_monitor",
                strategy: "deterministic",
                parallel: true,
                config: {
                    regulatoryRules: {
                        sec: ["wash_sale", "insider_trading", "market_manipulation"],
                        finra: ["pattern_day_trading", "margin_requirements", "suitability"],
                        firm: ["position_limits", "concentration_limits", "restricted_securities"]
                    },
                    monitoringFlags: {
                        unusualActivity: "volume_price_divergence",
                        patternDetection: "pump_dump_schemes",
                        accountBehavior: "suspicious_trading_patterns"
                    }
                }
            },
            {
                name: "portfolio_optimization",
                condition: "risk_score < 70 && compliance_approved === true",
                agent: "portfolio_optimizer",
                strategy: "reasoning",
                config: {
                    optimizationGoals: [
                        "risk_adjusted_return_maximization",
                        "portfolio_diversification",
                        "correlation_minimization",
                        "volatility_targeting"
                    ],
                    constraints: {
                        maxPositionSize: 0.1, // 10% of portfolio
                        maxSectorConcentration: 0.25, // 25% per sector
                        minCash: 0.05, // 5% cash buffer
                        riskBudget: "dynamic_based_on_vix"
                    }
                }
            },
            {
                name: "execution_strategy",
                condition: "final_approval === true",
                agent: "execution_specialist",
                strategy: "deterministic",
                config: {
                    executionAlgorithms: {
                        largeOrders: "volume_weighted_average_price",
                        urgentOrders: "market_on_close",
                        discreteOrders: "iceberg_algorithm",
                        volatileMarkets: "implementation_shortfall"
                    },
                    slippageMinimization: {
                        enabled: true,
                        maxSlippage: 0.005, // 0.5%
                        adaptiveRouting: true
                    }
                }
            }
        ],
        
        emergentCapabilities: [
            "Learns market patterns for optimal entry/exit timing",
            "Develops risk prediction models based on market conditions",
            "Creates adaptive portfolio optimization strategies",
            "Builds execution algorithms that minimize slippage",
            "Evolves compliance monitoring with regulatory changes",
            "Optimizes risk-return profiles through machine learning"
        ],
        
        learningEnabled: true,
        
        adaptationRules: [
            {
                condition: "average_slippage > 0.3%",
                adaptation: "optimize_execution_algorithms",
                learningWeight: 0.9
            },
            {
                condition: "risk_predictions_accuracy < 80%",
                adaptation: "enhance_risk_models",
                learningWeight: 0.85
            },
            {
                condition: "portfolio_sharpe_ratio < market_benchmark",
                adaptation: "improve_optimization_strategy",
                learningWeight: 0.8
            }
        ]
    },
    
    emergentBehaviors: [
        {
            name: "Market Regime Detection",
            description: "System learns to identify different market regimes and adapt strategies",
            emergesFrom: ["market_analysis", "execution_results", "volatility_patterns"],
            confidence: 0.0,
            timeToEmerge: "3-6 months with diverse market conditions"
        },
        {
            name: "Predictive Risk Modeling",
            description: "AI develops models to predict risk before it materializes",
            emergesFrom: ["risk_assessment", "market_analysis", "historical_outcomes"],
            confidence: 0.0,
            timeToEmerge: "6-12 months with 10,000+ trades"
        },
        {
            name: "Adaptive Execution Intelligence",
            description: "Execution algorithms adapt to changing market microstructure",
            emergesFrom: ["execution_strategy", "slippage_analysis", "order_flow_data"],
            confidence: 0.0,
            timeToEmerge: "2-4 months with high-frequency data"
        }
    ],
    
    evolutionTimeline: [
        { time: "T+0", capability: "Rule-based risk assessment", confidence: 0.65, description: "Basic compliance and risk checks" },
        { time: "T+1 month", capability: "Pattern-based risk detection", confidence: 0.72, description: "Identifies recurring risk patterns" },
        { time: "T+3 months", capability: "Market regime adaptation", confidence: 0.78, description: "Adapts strategies to market conditions" },
        { time: "T+6 months", capability: "Predictive risk intelligence", confidence: 0.85, description: "Predicts risks before they materialize" },
        { time: "T+12 months", capability: "Full trading intelligence", confidence: 0.91, description: "Comprehensive trading optimization system" }
    ]
};

/**
 * Customer Service Evolution Workflow
 * Shows how simple config creates sophisticated customer service AI
 */
export const CUSTOMER_SERVICE_EVOLUTION: ConfigDrivenWorkflow = {
    id: "customer_service_evolution_v1",
    name: "Evolving Customer Service Intelligence",
    description: "Customer service that improves itself through configuration-driven learning",
    
    config: {
        trigger: {
            type: "event_pattern",
            pattern: "customer/inquiry/*",
            conditions: {
                businessHours: "flexible", // Can handle after-hours
                urgency: "any",
                channel: ["email", "chat", "phone", "social"]
            }
        },
        
        stages: [
            {
                name: "inquiry_classification",
                strategy: "reasoning",
                promptTemplate: `
                    Classify customer inquiry:
                    Customer: {{customer_name}} ({{customer_tier}})
                    Channel: {{channel}}
                    Message: {{message}}
                    History: {{previous_interactions}}
                    
                    Determine:
                    1. Issue category and subcategory
                    2. Urgency level (emergency, high, medium, low)
                    3. Complexity (simple, moderate, complex)
                    4. Required expertise level
                    5. Estimated resolution time
                    6. Customer emotion state
                `,
                outputSchema: {
                    category: "string",
                    subcategory: "string", 
                    urgency: "enum:emergency,high,medium,low",
                    complexity: "enum:simple,moderate,complex",
                    expertiseRequired: "array:string",
                    estimatedTime: "number:minutes",
                    emotionState: "enum:frustrated,neutral,happy,angry",
                    confidence: "number:0-1"
                }
            },
            {
                name: "knowledge_retrieval",
                agent: "knowledge_specialist",
                strategy: "deterministic",
                parallel: true,
                config: {
                    searchStrategy: {
                        primary: "semantic_similarity",
                        fallback: "keyword_matching",
                        context: "customer_history_aware"
                    },
                    sources: [
                        "internal_knowledge_base",
                        "previous_resolutions",
                        "product_documentation",
                        "troubleshooting_guides"
                    ],
                    relevanceThreshold: 0.8
                }
            },
            {
                name: "solution_generation",
                strategy: "reasoning",
                condition: "complexity !== 'simple'",
                config: {
                    approachSelection: {
                        simple: "direct_answer_from_kb",
                        moderate: "guided_troubleshooting",
                        complex: "expert_consultation_preparation"
                    },
                    personalization: {
                        customerTier: "adjust_response_detail",
                        emotionState: "adapt_communication_tone",
                        preferredChannel: "optimize_for_medium"
                    }
                }
            },
            {
                name: "response_optimization",
                agent: "communication_optimizer",
                strategy: "reasoning",
                config: {
                    optimizationFactors: [
                        "clarity_and_comprehension",
                        "empathy_and_emotion_acknowledgment",
                        "actionability_of_steps",
                        "follow_up_prevention"
                    ],
                    adaptiveElements: {
                        tone: "match_customer_communication_style",
                        technicality: "adjust_for_customer_expertise",
                        length: "optimize_for_channel_and_urgency"
                    }
                }
            },
            {
                name: "escalation_decision",
                condition: "complexity === 'complex' || customer_escalation_requested",
                agent: "escalation_coordinator",
                strategy: "deterministic",
                config: {
                    escalationCriteria: {
                        technical: "requires_engineering_input",
                        policy: "requires_management_override",
                        billing: "requires_finance_approval",
                        legal: "requires_legal_review"
                    },
                    handoffPreparation: {
                        contextSummary: "detailed_issue_brief",
                        customerProfile: "comprehensive_history",
                        attemptedSolutions: "what_was_tried_and_results",
                        recommendedApproach: "expert_guidance"
                    }
                }
            },
            {
                name: "quality_assurance",
                agent: "quality_monitor",
                strategy: "reasoning",
                parallel: true,
                config: {
                    qualityMetrics: [
                        "resolution_accuracy",
                        "customer_satisfaction_prediction",
                        "response_completeness",
                        "brand_voice_consistency",
                        "policy_compliance"
                    ],
                    feedbackLoop: {
                        enabled: true,
                        realtime: true,
                        learningWeight: 0.7
                    }
                }
            }
        ],
        
        emergentCapabilities: [
            "Learns optimal resolution paths for each issue type",
            "Develops customer communication style preferences",
            "Creates predictive models for customer satisfaction",
            "Builds automated quality improvement mechanisms",
            "Evolves escalation decision intelligence",
            "Optimizes resource allocation across support channels"
        ],
        
        learningEnabled: true,
        
        adaptationRules: [
            {
                condition: "customer_satisfaction < 85%",
                adaptation: "improve_solution_quality",
                learningWeight: 0.9
            },
            {
                condition: "resolution_time > target_sla",
                adaptation: "optimize_process_efficiency",
                learningWeight: 0.8
            },
            {
                condition: "escalation_rate > 15%",
                adaptation: "enhance_first_level_capabilities",
                learningWeight: 0.85
            }
        ]
    },
    
    emergentBehaviors: [
        {
            name: "Predictive Issue Resolution",
            description: "System predicts solutions before full issue description",
            emergesFrom: ["inquiry_classification", "knowledge_retrieval", "resolution_patterns"],
            confidence: 0.0,
            timeToEmerge: "2-3 months with 1000+ interactions"
        },
        {
            name: "Proactive Customer Care",
            description: "AI identifies potential issues before customers report them",
            emergesFrom: ["customer_usage_patterns", "product_telemetry", "issue_trends"],
            confidence: 0.0,
            timeToEmerge: "6-9 months with comprehensive data integration"
        },
        {
            name: "Dynamic Quality Optimization",
            description: "System continuously improves response quality through learning",
            emergesFrom: ["quality_assurance", "customer_feedback", "outcome_tracking"],
            confidence: 0.0,
            timeToEmerge: "1-2 months with feedback loops"
        }
    ],
    
    evolutionTimeline: [
        { time: "T+0", capability: "Basic issue classification", confidence: 0.68, description: "Rule-based categorization" },
        { time: "T+2 weeks", capability: "Knowledge matching", confidence: 0.74, description: "Improved solution retrieval" },
        { time: "T+1 month", capability: "Communication optimization", confidence: 0.79, description: "Personalized response style" },
        { time: "T+3 months", capability: "Predictive resolution", confidence: 0.86, description: "Anticipates solutions" },
        { time: "T+6 months", capability: "Proactive care intelligence", confidence: 0.89, description: "Prevents issues before they occur" }
    ]
};

/**
 * Export all config-driven workflows
 */
export const CONFIG_DRIVEN_WORKFLOWS = {
    PATIENT_INTAKE_WORKFLOW,
    TRADING_RISK_WORKFLOW,
    CUSTOMER_SERVICE_EVOLUTION,
} as const;

/**
 * Summary: What These Examples Prove
 * 
 * 1. **No Code Required**: Complex, intelligent workflows defined purely through configuration
 * 2. **Emergent Intelligence**: Sophisticated behaviors emerge from simple rules
 * 3. **Self-Improvement**: Systems learn and adapt autonomously
 * 4. **Domain Agnostic**: Same architecture works for healthcare, finance, customer service
 * 5. **Scalable Complexity**: Simple configs can create arbitrarily complex behaviors
 */
export const CONFIG_DRIVEN_PRINCIPLES = {
    noCodeWorkflows: "Complex behaviors from pure configuration",
    emergentIntelligence: "Sophisticated capabilities emerge over time",
    autonomousLearning: "Systems improve themselves through use",
    domainAgnostic: "Same patterns work across all industries",
    scalableComplexity: "Simple rules → complex emergent behaviors"
} as const;