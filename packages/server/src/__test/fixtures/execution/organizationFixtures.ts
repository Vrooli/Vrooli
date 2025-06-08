/**
 * MOISE+ Organization Fixtures
 * 
 * Complete organizational specifications demonstrating MOISE+ capabilities
 * including structural, functional, and normative dimensions.
 */

import type { 
    MOISEPlusOrganization, 
    MOISEPlusSpecification,
    StructuralSpecification,
    FunctionalSpecification,
    NormativeSpecification,
    RoleSpecification,
    GroupSpecification,
    GoalSpecification,
    MissionSpecification,
    NormSpecification
} from "./moiseTypes.js";
import { TEST_IDS } from "./testIdGenerator.js";

/**
 * Healthcare Compliance Organization
 * Demonstrates complex healthcare regulatory compliance with HIPAA focus
 */
export const HEALTHCARE_COMPLIANCE_ORG: MOISEPlusOrganization = {
    id: TEST_IDS.HEALTHCARE_ORG,
    name: "Healthcare Compliance Organization",
    description: "MOISE+ organization for comprehensive healthcare compliance management",
    
    specification: {
        // Structural Specification - Roles and Groups
        structural: {
            roles: [
                {
                    id: "compliance_director",
                    name: "Compliance Director",
                    description: "Overall compliance strategy and governance",
                    minInstances: 1,
                    maxInstances: 1,
                    requiredCapabilities: [
                        "strategic_planning",
                        "regulatory_expertise",
                        "risk_assessment",
                        "leadership"
                    ],
                    responsibilities: [
                        {
                            id: "compliance_strategy",
                            description: "Define and maintain compliance strategy",
                            priority: "critical",
                            frequency: "continuous"
                        },
                        {
                            id: "board_reporting",
                            description: "Report compliance status to board",
                            priority: "high",
                            frequency: "periodic"
                        }
                    ],
                    communicationLinks: [
                        {
                            type: "authority",
                            target: ["medical_officer", "privacy_officer", "lead_auditor"],
                            bidirectional: false
                        },
                        {
                            type: "coordination",
                            target: ["external_auditor"],
                            bidirectional: true
                        }
                    ]
                },
                {
                    id: "medical_officer",
                    name: "Chief Medical Officer",
                    description: "Medical standards and clinical compliance",
                    minInstances: 1,
                    maxInstances: 1,
                    requiredCapabilities: [
                        "medical_expertise",
                        "clinical_standards",
                        "quality_assurance",
                        "patient_safety"
                    ],
                    responsibilities: [
                        {
                            id: "medical_standards",
                            description: "Ensure medical standards compliance",
                            priority: "critical",
                            frequency: "continuous"
                        },
                        {
                            id: "clinical_protocols",
                            description: "Develop and maintain clinical protocols",
                            priority: "high",
                            frequency: "periodic"
                        }
                    ],
                    communicationLinks: [
                        {
                            type: "collaboration",
                            target: ["privacy_officer", "clinical_staff"],
                            bidirectional: true
                        },
                        {
                            type: "information",
                            target: ["compliance_analyst"],
                            bidirectional: true
                        }
                    ]
                },
                {
                    id: "privacy_officer",
                    name: "Chief Privacy Officer",
                    description: "Data privacy and HIPAA compliance",
                    minInstances: 1,
                    maxInstances: 1,
                    requiredCapabilities: [
                        "privacy_law",
                        "data_protection",
                        "hipaa_expertise",
                        "incident_response"
                    ],
                    responsibilities: [
                        {
                            id: "privacy_compliance",
                            description: "Ensure HIPAA and privacy compliance",
                            priority: "critical",
                            frequency: "continuous"
                        },
                        {
                            id: "breach_response",
                            description: "Manage data breach incidents",
                            priority: "critical",
                            frequency: "event-driven"
                        }
                    ],
                    communicationLinks: [
                        {
                            type: "authority",
                            target: ["data_analyst", "security_team"],
                            bidirectional: false
                        },
                        {
                            type: "collaboration",
                            target: ["medical_officer", "it_security"],
                            bidirectional: true
                        }
                    ]
                },
                {
                    id: "lead_auditor",
                    name: "Lead Compliance Auditor",
                    description: "Compliance auditing and verification",
                    minInstances: 1,
                    maxInstances: 2,
                    inheritance: ["auditor"],
                    requiredCapabilities: [
                        "audit_methodology",
                        "risk_assessment",
                        "regulatory_knowledge",
                        "reporting"
                    ],
                    responsibilities: [
                        {
                            id: "audit_planning",
                            description: "Plan and schedule compliance audits",
                            priority: "high",
                            frequency: "periodic"
                        },
                        {
                            id: "audit_execution",
                            description: "Execute compliance audits",
                            priority: "high",
                            frequency: "periodic"
                        }
                    ],
                    communicationLinks: [
                        {
                            type: "authority",
                            target: ["compliance_analyst"],
                            bidirectional: false
                        },
                        {
                            type: "information",
                            target: ["compliance_director"],
                            bidirectional: true
                        }
                    ]
                },
                {
                    id: "compliance_analyst",
                    name: "Compliance Analyst",
                    description: "Day-to-day compliance monitoring and analysis",
                    minInstances: 2,
                    maxInstances: 10,
                    requiredCapabilities: [
                        "data_analysis",
                        "regulatory_knowledge",
                        "documentation",
                        "monitoring_tools"
                    ],
                    responsibilities: [
                        {
                            id: "continuous_monitoring",
                            description: "Monitor compliance metrics continuously",
                            priority: "high",
                            frequency: "continuous"
                        },
                        {
                            id: "incident_detection",
                            description: "Detect and report compliance incidents",
                            priority: "critical",
                            frequency: "continuous"
                        }
                    ],
                    communicationLinks: [
                        {
                            type: "information",
                            target: ["lead_auditor", "privacy_officer"],
                            bidirectional: true
                        }
                    ]
                },
                {
                    id: "data_analyst",
                    name: "Healthcare Data Analyst",
                    description: "Analyze healthcare data for compliance",
                    minInstances: 1,
                    maxInstances: 5,
                    requiredCapabilities: [
                        "data_analytics",
                        "phi_handling",
                        "statistical_analysis",
                        "reporting_tools"
                    ],
                    responsibilities: [
                        {
                            id: "data_analysis",
                            description: "Analyze healthcare data patterns",
                            priority: "medium",
                            frequency: "continuous"
                        },
                        {
                            id: "anomaly_detection",
                            description: "Detect anomalies in data access",
                            priority: "high",
                            frequency: "continuous"
                        }
                    ],
                    communicationLinks: [
                        {
                            type: "information",
                            target: ["privacy_officer", "compliance_analyst"],
                            bidirectional: true
                        }
                    ]
                }
            ],
            
            groups: [
                {
                    id: "compliance_board",
                    name: "Compliance Board",
                    type: "committee",
                    roles: ["compliance_director", "medical_officer", "privacy_officer"],
                    minSize: 3,
                    maxSize: 5,
                    subgroups: ["audit_team", "incident_response_team"]
                },
                {
                    id: "audit_team",
                    name: "Audit Team",
                    type: "functional_unit",
                    roles: ["lead_auditor", "compliance_analyst", "data_analyst"],
                    minSize: 3,
                    maxSize: 12,
                    parentGroup: "compliance_board"
                },
                {
                    id: "incident_response_team",
                    name: "Incident Response Team",
                    type: "team",
                    roles: ["privacy_officer", "medical_officer", "compliance_analyst"],
                    minSize: 3,
                    maxSize: 8,
                    parentGroup: "compliance_board"
                }
            ],
            
            links: [
                {
                    from: "compliance_board",
                    to: "audit_team",
                    type: "composition",
                    cardinality: "1..1"
                },
                {
                    from: "compliance_board",
                    to: "incident_response_team",
                    type: "composition",
                    cardinality: "1..1"
                }
            ],
            
            compatibilities: [
                {
                    role1: "compliance_director",
                    role2: "lead_auditor",
                    type: "mutex",
                    reason: "separation_of_duties"
                },
                {
                    role1: "medical_officer",
                    role2: "privacy_officer",
                    type: "compatible",
                    reason: "complementary_expertise"
                },
                {
                    role1: "lead_auditor",
                    role2: "compliance_analyst",
                    type: "requires",
                    reason: "auditor_needs_analyst_support"
                }
            ],
            
            inheritance: [
                {
                    parent: "auditor",
                    child: "lead_auditor",
                    inheritedProperties: ["capabilities", "responsibilities"]
                }
            ]
        },
        
        // Functional Specification - Goals and Missions
        functional: {
            goals: [
                {
                    id: "ensure_hipaa_compliance",
                    type: "maintenance",
                    description: "Maintain HIPAA compliance across all systems",
                    decomposition: "AND",
                    subgoals: [
                        "monitor_phi_access",
                        "audit_data_handling",
                        "incident_response",
                        "training_compliance"
                    ],
                    successCriteria: {
                        metric: "compliance_score",
                        threshold: 0.95,
                        operator: ">=",
                        timeWindow: "monthly"
                    },
                    deadline: "continuous"
                },
                {
                    id: "monitor_phi_access",
                    type: "maintenance",
                    description: "Monitor all PHI access continuously",
                    successCriteria: {
                        metric: "monitoring_coverage",
                        threshold: 0.99,
                        operator: ">=",
                        timeWindow: "daily"
                    }
                },
                {
                    id: "audit_data_handling",
                    type: "achievement",
                    description: "Complete quarterly data handling audits",
                    successCriteria: {
                        metric: "audit_completion",
                        threshold: 1.0,
                        operator: "==",
                        timeWindow: "quarterly"
                    },
                    deadline: "end_of_quarter"
                },
                {
                    id: "incident_response",
                    type: "achievement",
                    description: "Respond to incidents within regulatory timeframes",
                    successCriteria: {
                        metric: "response_time_minutes",
                        threshold: 60,
                        operator: "<=",
                        timeWindow: "per_incident"
                    }
                },
                {
                    id: "training_compliance",
                    type: "maintenance",
                    description: "Ensure all staff complete compliance training",
                    successCriteria: {
                        metric: "training_completion_rate",
                        threshold: 1.0,
                        operator: "==",
                        timeWindow: "annual"
                    }
                }
            ],
            
            plans: [
                {
                    id: "quarterly_audit_plan",
                    name: "Quarterly Compliance Audit Plan",
                    goals: ["audit_data_handling"],
                    steps: [
                        {
                            id: "plan_audit",
                            action: "Create audit plan and schedule",
                            responsibleRole: "lead_auditor",
                            duration: "1_week",
                            dependencies: []
                        },
                        {
                            id: "notify_departments",
                            action: "Notify departments of audit schedule",
                            responsibleRole: "compliance_analyst",
                            duration: "2_days",
                            dependencies: ["plan_audit"]
                        },
                        {
                            id: "conduct_audit",
                            action: "Execute audit procedures",
                            responsibleRole: "audit_team",
                            duration: "2_weeks",
                            dependencies: ["notify_departments"]
                        },
                        {
                            id: "analyze_findings",
                            action: "Analyze audit findings",
                            responsibleRole: "data_analyst",
                            duration: "1_week",
                            dependencies: ["conduct_audit"]
                        },
                        {
                            id: "report_results",
                            action: "Prepare and present audit report",
                            responsibleRole: "lead_auditor",
                            duration: "3_days",
                            dependencies: ["analyze_findings"]
                        }
                    ],
                    preconditions: ["previous_audit_completed", "resources_available"],
                    postconditions: ["audit_report_filed", "findings_addressed"]
                },
                {
                    id: "incident_response_plan",
                    name: "Data Breach Incident Response Plan",
                    goals: ["incident_response"],
                    steps: [
                        {
                            id: "detect_incident",
                            action: "Detect and verify incident",
                            responsibleRole: "compliance_analyst",
                            duration: "15_minutes",
                            dependencies: []
                        },
                        {
                            id: "activate_team",
                            action: "Activate incident response team",
                            responsibleRole: "privacy_officer",
                            duration: "15_minutes",
                            dependencies: ["detect_incident"]
                        },
                        {
                            id: "contain_breach",
                            action: "Contain and isolate breach",
                            responsibleRole: "incident_response_team",
                            duration: "30_minutes",
                            dependencies: ["activate_team"]
                        },
                        {
                            id: "assess_impact",
                            action: "Assess breach impact and scope",
                            responsibleRole: "data_analyst",
                            duration: "2_hours",
                            dependencies: ["contain_breach"]
                        },
                        {
                            id: "notify_authorities",
                            action: "Notify regulatory authorities",
                            responsibleRole: "compliance_director",
                            duration: "4_hours",
                            dependencies: ["assess_impact"]
                        }
                    ],
                    preconditions: ["incident_detected", "team_available"],
                    postconditions: ["breach_contained", "authorities_notified", "patients_notified"]
                }
            ],
            
            missions: [
                {
                    id: "continuous_monitoring_mission",
                    name: "Continuous PHI Monitoring",
                    goals: ["monitor_phi_access"],
                    minAgents: 2,
                    maxAgents: 5,
                    preferredRoles: ["compliance_analyst", "data_analyst"],
                    deadline: "continuous"
                },
                {
                    id: "quarterly_audit_mission",
                    name: "Quarterly Compliance Audit",
                    goals: ["audit_data_handling"],
                    minAgents: 3,
                    maxAgents: 8,
                    preferredRoles: ["lead_auditor", "compliance_analyst", "data_analyst"],
                    deadline: "end_of_quarter"
                },
                {
                    id: "incident_response_mission",
                    name: "Incident Response",
                    goals: ["incident_response"],
                    minAgents: 3,
                    maxAgents: 6,
                    preferredRoles: ["privacy_officer", "medical_officer", "compliance_analyst"],
                    deadline: "immediate"
                }
            ],
            
            schemes: [
                {
                    id: "hipaa_compliance_scheme",
                    name: "HIPAA Compliance Scheme",
                    rootGoal: "ensure_hipaa_compliance",
                    missions: ["continuous_monitoring_mission", "quarterly_audit_mission", "incident_response_mission"],
                    monitoringScheme: "real_time_dashboard"
                }
            ]
        },
        
        // Normative Specification - Rules and Constraints
        normative: {
            norms: [
                {
                    id: "immediate_incident_reporting",
                    type: "obligation",
                    scope: "all_roles",
                    condition: "phi_breach_detected",
                    target: "report_to_compliance_board",
                    deadline: "15_minutes",
                    sanction: "performance_penalty",
                    priority: "critical"
                },
                {
                    id: "quarterly_audit_completion",
                    type: "obligation",
                    scope: "lead_auditor",
                    condition: "end_of_quarter_approaching",
                    target: "complete_quarterly_audit",
                    deadline: "last_day_of_quarter",
                    sanction: "escalation_to_director",
                    priority: "high"
                },
                {
                    id: "data_access_logging",
                    type: "obligation",
                    scope: "all_roles",
                    condition: "accessing_phi",
                    target: "log_access_with_reason",
                    deadline: "immediate",
                    sanction: "access_revocation",
                    priority: "critical"
                }
            ],
            
            permissions: [
                {
                    id: "phi_access_permission",
                    type: "permission",
                    scope: "medical_officer",
                    condition: "patient_care_need",
                    target: "access_patient_phi",
                    exceptions: ["audit_in_progress", "access_suspended"]
                },
                {
                    id: "audit_access_permission",
                    type: "permission",
                    scope: "lead_auditor",
                    condition: "audit_in_progress",
                    target: "access_all_phi_logs",
                    exceptions: []
                },
                {
                    id: "policy_modification_permission",
                    type: "permission",
                    scope: "compliance_director",
                    condition: "board_approval",
                    target: "modify_compliance_policies",
                    exceptions: ["regulatory_freeze_period"]
                }
            ],
            
            obligations: [
                {
                    id: "weekly_compliance_report",
                    type: "obligation",
                    scope: "compliance_director",
                    condition: "weekly_review",
                    target: "submit_compliance_report",
                    deadline: "friday_5pm",
                    sanction: "board_notification",
                    priority: "high",
                    fulfillmentMonitoring: "periodic"
                },
                {
                    id: "training_completion",
                    type: "obligation",
                    scope: "all_roles",
                    condition: "annual_training_due",
                    target: "complete_hipaa_training",
                    deadline: "30_days",
                    sanction: "access_restriction",
                    priority: "medium",
                    fulfillmentMonitoring: "continuous"
                }
            ],
            
            prohibitions: [
                {
                    id: "unauthorized_phi_sharing",
                    type: "prohibition",
                    scope: "all_roles",
                    condition: "always",
                    target: "share_phi_externally",
                    sanction: "immediate_termination",
                    priority: "critical",
                    violationDetection: "immediate"
                },
                {
                    id: "audit_record_modification",
                    type: "prohibition",
                    scope: "all_roles",
                    condition: "always",
                    target: "modify_audit_logs",
                    sanction: "criminal_prosecution",
                    priority: "critical",
                    violationDetection: "immediate"
                },
                {
                    id: "conflict_of_interest",
                    type: "prohibition",
                    scope: "lead_auditor",
                    condition: "auditing_own_work",
                    target: "conduct_self_audit",
                    sanction: "audit_invalidation",
                    priority: "high",
                    violationDetection: "periodic"
                }
            ]
        }
    },
    
    performanceMetrics: {
        goalCompletionRate: 0.94,
        normComplianceRate: 0.98,
        averageMissionDuration: 14.5, // days
        resourceEfficiency: 0.87,
        collaborationScore: 0.91
    }
};

/**
 * Financial Trading Organization
 * Demonstrates complex financial compliance with separation of duties
 */
export const FINANCIAL_TRADING_ORG: MOISEPlusOrganization = {
    id: TEST_IDS.FINANCIAL_ORG,
    name: "Financial Trading Organization",
    description: "MOISE+ organization for financial trading compliance and risk management",
    
    specification: {
        structural: {
            roles: [
                {
                    id: "chief_risk_officer",
                    name: "Chief Risk Officer",
                    description: "Overall risk management and compliance",
                    minInstances: 1,
                    maxInstances: 1,
                    requiredCapabilities: [
                        "risk_assessment",
                        "regulatory_expertise",
                        "financial_modeling",
                        "crisis_management"
                    ],
                    responsibilities: [
                        {
                            id: "risk_strategy",
                            description: "Define risk management strategy",
                            priority: "critical",
                            frequency: "continuous"
                        },
                        {
                            id: "regulatory_compliance",
                            description: "Ensure regulatory compliance",
                            priority: "critical",
                            frequency: "continuous"
                        }
                    ],
                    communicationLinks: [
                        {
                            type: "authority",
                            target: ["trading_desk_manager", "compliance_officer", "risk_analyst"],
                            bidirectional: false
                        }
                    ]
                },
                {
                    id: "trading_desk_manager",
                    name: "Trading Desk Manager",
                    description: "Manage trading operations and strategies",
                    minInstances: 1,
                    maxInstances: 5,
                    requiredCapabilities: [
                        "trading_strategies",
                        "market_analysis",
                        "team_management",
                        "risk_awareness"
                    ],
                    responsibilities: [
                        {
                            id: "trading_oversight",
                            description: "Oversee trading activities",
                            priority: "critical",
                            frequency: "continuous"
                        },
                        {
                            id: "position_management",
                            description: "Manage trading positions within limits",
                            priority: "critical",
                            frequency: "continuous"
                        }
                    ],
                    communicationLinks: [
                        {
                            type: "authority",
                            target: ["trader", "junior_trader"],
                            bidirectional: false
                        },
                        {
                            type: "coordination",
                            target: ["risk_analyst"],
                            bidirectional: true
                        }
                    ]
                },
                {
                    id: "trader",
                    name: "Senior Trader",
                    description: "Execute trades within approved limits",
                    minInstances: 2,
                    maxInstances: 20,
                    requiredCapabilities: [
                        "trade_execution",
                        "market_knowledge",
                        "risk_management",
                        "decision_making"
                    ],
                    responsibilities: [
                        {
                            id: "execute_trades",
                            description: "Execute trades within limits",
                            priority: "high",
                            frequency: "continuous"
                        },
                        {
                            id: "monitor_positions",
                            description: "Monitor and report positions",
                            priority: "high",
                            frequency: "continuous"
                        }
                    ],
                    communicationLinks: [
                        {
                            type: "information",
                            target: ["risk_analyst", "trading_desk_manager"],
                            bidirectional: true
                        }
                    ]
                },
                {
                    id: "compliance_officer",
                    name: "Compliance Officer",
                    description: "Ensure trading compliance with regulations",
                    minInstances: 1,
                    maxInstances: 3,
                    requiredCapabilities: [
                        "regulatory_knowledge",
                        "compliance_monitoring",
                        "investigation",
                        "reporting"
                    ],
                    responsibilities: [
                        {
                            id: "compliance_monitoring",
                            description: "Monitor trading compliance",
                            priority: "critical",
                            frequency: "continuous"
                        },
                        {
                            id: "regulatory_reporting",
                            description: "Submit regulatory reports",
                            priority: "high",
                            frequency: "periodic"
                        }
                    ],
                    communicationLinks: [
                        {
                            type: "authority",
                            target: ["compliance_analyst"],
                            bidirectional: false
                        },
                        {
                            type: "information",
                            target: ["chief_risk_officer", "external_auditor"],
                            bidirectional: true
                        }
                    ]
                },
                {
                    id: "risk_analyst",
                    name: "Risk Analyst",
                    description: "Analyze and monitor trading risks",
                    minInstances: 2,
                    maxInstances: 10,
                    requiredCapabilities: [
                        "quantitative_analysis",
                        "risk_modeling",
                        "data_analytics",
                        "reporting"
                    ],
                    responsibilities: [
                        {
                            id: "risk_analysis",
                            description: "Analyze portfolio risks",
                            priority: "high",
                            frequency: "continuous"
                        },
                        {
                            id: "limit_monitoring",
                            description: "Monitor trading limits",
                            priority: "critical",
                            frequency: "continuous"
                        }
                    ],
                    communicationLinks: [
                        {
                            type: "information",
                            target: ["chief_risk_officer", "trading_desk_manager"],
                            bidirectional: true
                        }
                    ]
                }
            ],
            
            groups: [
                {
                    id: "risk_committee",
                    name: "Risk Management Committee",
                    type: "committee",
                    roles: ["chief_risk_officer", "compliance_officer", "trading_desk_manager"],
                    minSize: 3,
                    maxSize: 7,
                    subgroups: ["trading_desk", "compliance_unit"]
                },
                {
                    id: "trading_desk",
                    name: "Trading Desk",
                    type: "functional_unit",
                    roles: ["trading_desk_manager", "trader", "junior_trader"],
                    minSize: 3,
                    maxSize: 25,
                    parentGroup: "risk_committee"
                },
                {
                    id: "compliance_unit",
                    name: "Compliance Unit",
                    type: "functional_unit",
                    roles: ["compliance_officer", "compliance_analyst"],
                    minSize: 2,
                    maxSize: 5,
                    parentGroup: "risk_committee"
                }
            ],
            
            links: [
                {
                    from: "risk_committee",
                    to: "trading_desk",
                    type: "aggregation",
                    cardinality: "1..*"
                },
                {
                    from: "risk_committee",
                    to: "compliance_unit",
                    type: "composition",
                    cardinality: "1..1"
                }
            ],
            
            compatibilities: [
                {
                    role1: "trader",
                    role2: "compliance_officer",
                    type: "mutex",
                    reason: "separation_of_duties"
                },
                {
                    role1: "trader",
                    role2: "risk_analyst",
                    type: "mutex",
                    reason: "independent_risk_assessment"
                },
                {
                    role1: "chief_risk_officer",
                    role2: "trading_desk_manager",
                    type: "mutex",
                    reason: "independent_oversight"
                }
            ],
            
            inheritance: []
        },
        
        functional: {
            goals: [
                {
                    id: "maintain_trading_compliance",
                    type: "maintenance",
                    description: "Maintain regulatory compliance in all trading activities",
                    decomposition: "AND",
                    subgoals: [
                        "monitor_trade_compliance",
                        "enforce_position_limits",
                        "prevent_market_manipulation",
                        "ensure_best_execution"
                    ],
                    successCriteria: {
                        metric: "compliance_violations",
                        threshold: 0,
                        operator: "==",
                        timeWindow: "daily"
                    }
                },
                {
                    id: "manage_portfolio_risk",
                    type: "optimization",
                    description: "Optimize portfolio risk-return profile",
                    decomposition: "AND",
                    subgoals: [
                        "monitor_var_limits",
                        "diversify_positions",
                        "hedge_exposures"
                    ],
                    successCriteria: {
                        metric: "risk_adjusted_return",
                        threshold: 0.15,
                        operator: ">=",
                        timeWindow: "monthly"
                    }
                }
            ],
            
            plans: [
                {
                    id: "daily_risk_review",
                    name: "Daily Risk Review Process",
                    goals: ["manage_portfolio_risk"],
                    steps: [
                        {
                            id: "calculate_var",
                            action: "Calculate Value at Risk",
                            responsibleRole: "risk_analyst",
                            duration: "30_minutes",
                            dependencies: []
                        },
                        {
                            id: "stress_test",
                            action: "Run stress test scenarios",
                            responsibleRole: "risk_analyst",
                            duration: "1_hour",
                            dependencies: ["calculate_var"]
                        },
                        {
                            id: "review_limits",
                            action: "Review limit utilization",
                            responsibleRole: "risk_analyst",
                            duration: "30_minutes",
                            dependencies: []
                        },
                        {
                            id: "prepare_report",
                            action: "Prepare risk report",
                            responsibleRole: "risk_analyst",
                            duration: "30_minutes",
                            dependencies: ["calculate_var", "stress_test", "review_limits"]
                        },
                        {
                            id: "review_meeting",
                            action: "Conduct risk review meeting",
                            responsibleRole: "chief_risk_officer",
                            duration: "1_hour",
                            dependencies: ["prepare_report"]
                        }
                    ],
                    preconditions: ["market_data_available", "positions_reconciled"],
                    postconditions: ["risk_report_distributed", "action_items_assigned"]
                }
            ],
            
            missions: [
                {
                    id: "real_time_monitoring",
                    name: "Real-time Trade Monitoring",
                    goals: ["monitor_trade_compliance", "enforce_position_limits"],
                    minAgents: 2,
                    maxAgents: 4,
                    preferredRoles: ["compliance_officer", "risk_analyst"],
                    deadline: "continuous"
                },
                {
                    id: "daily_risk_assessment",
                    name: "Daily Risk Assessment",
                    goals: ["manage_portfolio_risk"],
                    minAgents: 2,
                    maxAgents: 5,
                    preferredRoles: ["risk_analyst", "chief_risk_officer"],
                    deadline: "daily_before_market_open"
                }
            ],
            
            schemes: [
                {
                    id: "trading_compliance_scheme",
                    name: "Trading Compliance and Risk Management",
                    rootGoal: "maintain_trading_compliance",
                    missions: ["real_time_monitoring", "daily_risk_assessment"],
                    monitoringScheme: "real_time_surveillance_system"
                }
            ]
        },
        
        normative: {
            norms: [
                {
                    id: "trade_within_limits",
                    type: "obligation",
                    scope: "trader",
                    condition: "always",
                    target: "respect_position_limits",
                    deadline: "immediate",
                    sanction: "trading_suspension",
                    priority: "critical"
                },
                {
                    id: "report_violations",
                    type: "obligation",
                    scope: "all_roles",
                    condition: "violation_detected",
                    target: "report_to_compliance",
                    deadline: "immediate",
                    sanction: "disciplinary_action",
                    priority: "critical"
                }
            ],
            
            permissions: [
                {
                    id: "execute_trades",
                    type: "permission",
                    scope: "trader",
                    condition: "within_approved_limits",
                    target: "place_market_orders",
                    exceptions: ["market_closed", "account_suspended"]
                },
                {
                    id: "override_limits",
                    type: "permission",
                    scope: "chief_risk_officer",
                    condition: "exceptional_circumstances",
                    target: "temporary_limit_override",
                    exceptions: ["regulatory_restriction"]
                }
            ],
            
            obligations: [
                {
                    id: "daily_position_report",
                    type: "obligation",
                    scope: "trading_desk_manager",
                    condition: "end_of_trading_day",
                    target: "submit_position_report",
                    deadline: "market_close_plus_2_hours",
                    sanction: "escalation",
                    priority: "high",
                    fulfillmentMonitoring: "continuous"
                }
            ],
            
            prohibitions: [
                {
                    id: "insider_trading",
                    type: "prohibition",
                    scope: "all_roles",
                    condition: "always",
                    target: "trade_on_inside_information",
                    sanction: "termination_and_prosecution",
                    priority: "critical",
                    violationDetection: "immediate"
                },
                {
                    id: "market_manipulation",
                    type: "prohibition",
                    scope: "all_roles",
                    condition: "always",
                    target: "manipulate_market_prices",
                    sanction: "regulatory_action",
                    priority: "critical",
                    violationDetection: "immediate"
                },
                {
                    id: "front_running",
                    type: "prohibition",
                    scope: "trader",
                    condition: "client_order_pending",
                    target: "trade_ahead_of_client",
                    sanction: "license_revocation",
                    priority: "critical",
                    violationDetection: "immediate"
                }
            ]
        }
    },
    
    performanceMetrics: {
        goalCompletionRate: 0.97,
        normComplianceRate: 0.99,
        averageMissionDuration: 1.2, // days
        resourceEfficiency: 0.93,
        collaborationScore: 0.88
    }
};

/**
 * Research Team Organization
 * Demonstrates dynamic role assignment and collaborative research
 */
export const RESEARCH_TEAM_ORG: MOISEPlusOrganization = {
    id: TEST_IDS.RESEARCH_ORG,
    name: "AI Research Team Organization",
    description: "MOISE+ organization for collaborative AI research with dynamic roles",
    
    specification: {
        structural: {
            roles: [
                {
                    id: "research_director",
                    name: "Research Director",
                    description: "Lead research strategy and coordination",
                    minInstances: 1,
                    maxInstances: 1,
                    requiredCapabilities: [
                        "research_leadership",
                        "strategic_planning",
                        "grant_writing",
                        "publication_expertise"
                    ],
                    responsibilities: [
                        {
                            id: "research_strategy",
                            description: "Define research direction and priorities",
                            priority: "critical",
                            frequency: "continuous"
                        },
                        {
                            id: "resource_allocation",
                            description: "Allocate research resources",
                            priority: "high",
                            frequency: "periodic"
                        }
                    ],
                    communicationLinks: [
                        {
                            type: "authority",
                            target: ["principal_investigator", "research_coordinator"],
                            bidirectional: false
                        },
                        {
                            type: "coordination",
                            target: ["external_collaborator"],
                            bidirectional: true
                        }
                    ]
                },
                {
                    id: "principal_investigator",
                    name: "Principal Investigator",
                    description: "Lead specific research projects",
                    minInstances: 1,
                    maxInstances: 5,
                    requiredCapabilities: [
                        "domain_expertise",
                        "project_management",
                        "mentoring",
                        "publication_track_record"
                    ],
                    responsibilities: [
                        {
                            id: "project_leadership",
                            description: "Lead research project execution",
                            priority: "critical",
                            frequency: "continuous"
                        },
                        {
                            id: "student_mentoring",
                            description: "Mentor graduate students",
                            priority: "high",
                            frequency: "continuous"
                        }
                    ],
                    communicationLinks: [
                        {
                            type: "authority",
                            target: ["postdoc_researcher", "graduate_student"],
                            bidirectional: false
                        },
                        {
                            type: "collaboration",
                            target: ["principal_investigator"],
                            bidirectional: true
                        }
                    ]
                },
                {
                    id: "postdoc_researcher",
                    name: "Postdoctoral Researcher",
                    description: "Conduct advanced research",
                    minInstances: 0,
                    maxInstances: 10,
                    requiredCapabilities: [
                        "research_expertise",
                        "independent_work",
                        "technical_skills",
                        "writing_skills"
                    ],
                    responsibilities: [
                        {
                            id: "conduct_research",
                            description: "Execute research experiments",
                            priority: "high",
                            frequency: "continuous"
                        },
                        {
                            id: "paper_writing",
                            description: "Write research papers",
                            priority: "high",
                            frequency: "periodic"
                        }
                    ],
                    communicationLinks: [
                        {
                            type: "collaboration",
                            target: ["graduate_student", "postdoc_researcher"],
                            bidirectional: true
                        },
                        {
                            type: "information",
                            target: ["principal_investigator"],
                            bidirectional: true
                        }
                    ]
                },
                {
                    id: "graduate_student",
                    name: "Graduate Student Researcher",
                    description: "Learn and contribute to research",
                    minInstances: 0,
                    maxInstances: 20,
                    requiredCapabilities: [
                        "learning_ability",
                        "basic_research_skills",
                        "programming",
                        "analytical_thinking"
                    ],
                    responsibilities: [
                        {
                            id: "research_assistance",
                            description: "Assist in research tasks",
                            priority: "medium",
                            frequency: "continuous"
                        },
                        {
                            id: "thesis_work",
                            description: "Work on thesis research",
                            priority: "high",
                            frequency: "continuous"
                        }
                    ],
                    communicationLinks: [
                        {
                            type: "information",
                            target: ["postdoc_researcher", "principal_investigator"],
                            bidirectional: true
                        }
                    ]
                },
                {
                    id: "research_coordinator",
                    name: "Research Coordinator",
                    description: "Coordinate research activities and resources",
                    minInstances: 1,
                    maxInstances: 2,
                    requiredCapabilities: [
                        "project_coordination",
                        "communication_skills",
                        "resource_management",
                        "scheduling"
                    ],
                    responsibilities: [
                        {
                            id: "coordinate_activities",
                            description: "Coordinate research activities",
                            priority: "high",
                            frequency: "continuous"
                        },
                        {
                            id: "manage_resources",
                            description: "Manage lab resources and equipment",
                            priority: "medium",
                            frequency: "continuous"
                        }
                    ],
                    communicationLinks: [
                        {
                            type: "coordination",
                            target: ["all_roles"],
                            bidirectional: true
                        }
                    ]
                }
            ],
            
            groups: [
                {
                    id: "research_council",
                    name: "Research Council",
                    type: "committee",
                    roles: ["research_director", "principal_investigator"],
                    minSize: 2,
                    maxSize: 6,
                    subgroups: ["research_teams"]
                },
                {
                    id: "research_team_alpha",
                    name: "Research Team Alpha",
                    type: "team",
                    roles: ["principal_investigator", "postdoc_researcher", "graduate_student"],
                    minSize: 2,
                    maxSize: 8,
                    parentGroup: "research_council"
                },
                {
                    id: "research_team_beta",
                    name: "Research Team Beta",
                    type: "team",
                    roles: ["principal_investigator", "postdoc_researcher", "graduate_student"],
                    minSize: 2,
                    maxSize: 8,
                    parentGroup: "research_council"
                }
            ],
            
            links: [
                {
                    from: "research_council",
                    to: "research_team_alpha",
                    type: "aggregation",
                    cardinality: "1..*"
                },
                {
                    from: "research_council",
                    to: "research_team_beta",
                    type: "aggregation",
                    cardinality: "1..*"
                }
            ],
            
            compatibilities: [
                {
                    role1: "principal_investigator",
                    role2: "postdoc_researcher",
                    type: "compatible",
                    reason: "mentoring_relationship"
                },
                {
                    role1: "postdoc_researcher",
                    role2: "graduate_student",
                    type: "compatible",
                    reason: "collaborative_learning"
                }
            ],
            
            inheritance: []
        },
        
        functional: {
            goals: [
                {
                    id: "advance_ai_research",
                    type: "achievement",
                    description: "Advance the state of AI research",
                    decomposition: "OR",
                    subgoals: [
                        "publish_papers",
                        "develop_algorithms",
                        "create_datasets",
                        "build_systems"
                    ],
                    successCriteria: {
                        metric: "research_impact_score",
                        threshold: 50,
                        operator: ">=",
                        timeWindow: "annual"
                    }
                },
                {
                    id: "publish_papers",
                    type: "achievement",
                    description: "Publish high-impact research papers",
                    successCriteria: {
                        metric: "accepted_papers",
                        threshold: 5,
                        operator: ">=",
                        timeWindow: "annual"
                    }
                },
                {
                    id: "train_researchers",
                    type: "maintenance",
                    description: "Train next generation of researchers",
                    successCriteria: {
                        metric: "graduated_students",
                        threshold: 2,
                        operator: ">=",
                        timeWindow: "annual"
                    }
                }
            ],
            
            plans: [
                {
                    id: "paper_submission_plan",
                    name: "Research Paper Submission Process",
                    goals: ["publish_papers"],
                    steps: [
                        {
                            id: "research_design",
                            action: "Design research experiments",
                            responsibleRole: "principal_investigator",
                            duration: "2_weeks",
                            dependencies: []
                        },
                        {
                            id: "run_experiments",
                            action: "Execute experiments",
                            responsibleRole: "postdoc_researcher",
                            duration: "8_weeks",
                            dependencies: ["research_design"]
                        },
                        {
                            id: "analyze_results",
                            action: "Analyze experimental results",
                            responsibleRole: "graduate_student",
                            duration: "2_weeks",
                            dependencies: ["run_experiments"]
                        },
                        {
                            id: "write_paper",
                            action: "Write research paper",
                            responsibleRole: "postdoc_researcher",
                            duration: "4_weeks",
                            dependencies: ["analyze_results"]
                        },
                        {
                            id: "internal_review",
                            action: "Internal paper review",
                            responsibleRole: "principal_investigator",
                            duration: "1_week",
                            dependencies: ["write_paper"]
                        },
                        {
                            id: "submit_paper",
                            action: "Submit to conference/journal",
                            responsibleRole: "principal_investigator",
                            duration: "1_day",
                            dependencies: ["internal_review"]
                        }
                    ],
                    preconditions: ["research_idea_validated", "resources_available"],
                    postconditions: ["paper_submitted", "preprint_posted"]
                }
            ],
            
            missions: [
                {
                    id: "research_project_mission",
                    name: "Research Project Execution",
                    goals: ["advance_ai_research", "publish_papers"],
                    minAgents: 3,
                    maxAgents: 10,
                    preferredRoles: ["principal_investigator", "postdoc_researcher", "graduate_student"],
                    deadline: "project_dependent"
                },
                {
                    id: "mentoring_mission",
                    name: "Student Mentoring",
                    goals: ["train_researchers"],
                    minAgents: 2,
                    maxAgents: 6,
                    preferredRoles: ["principal_investigator", "postdoc_researcher"],
                    deadline: "continuous"
                }
            ],
            
            schemes: [
                {
                    id: "research_excellence_scheme",
                    name: "Research Excellence Scheme",
                    rootGoal: "advance_ai_research",
                    missions: ["research_project_mission", "mentoring_mission"],
                    monitoringScheme: "quarterly_progress_review"
                }
            ]
        },
        
        normative: {
            norms: [
                {
                    id: "research_ethics",
                    type: "obligation",
                    scope: "all_roles",
                    condition: "conducting_research",
                    target: "follow_ethical_guidelines",
                    deadline: "always",
                    sanction: "research_suspension",
                    priority: "critical"
                },
                {
                    id: "weekly_progress_update",
                    type: "obligation",
                    scope: "graduate_student",
                    condition: "weekly_meeting",
                    target: "present_progress",
                    deadline: "every_friday",
                    sanction: "advisor_notification",
                    priority: "medium"
                }
            ],
            
            permissions: [
                {
                    id: "use_computing_resources",
                    type: "permission",
                    scope: "all_roles",
                    condition: "approved_project",
                    target: "access_hpc_cluster",
                    exceptions: ["maintenance_window", "quota_exceeded"]
                },
                {
                    id: "submit_papers",
                    type: "permission",
                    scope: "principal_investigator",
                    condition: "paper_reviewed",
                    target: "submit_to_venues",
                    exceptions: ["embargo_period"]
                }
            ],
            
            obligations: [
                {
                    id: "publish_results",
                    type: "obligation",
                    scope: "principal_investigator",
                    condition: "project_completed",
                    target: "publish_findings",
                    deadline: "6_months",
                    sanction: "funding_review",
                    priority: "high",
                    fulfillmentMonitoring: "periodic"
                }
            ],
            
            prohibitions: [
                {
                    id: "plagiarism",
                    type: "prohibition",
                    scope: "all_roles",
                    condition: "always",
                    target: "plagiarize_work",
                    sanction: "immediate_dismissal",
                    priority: "critical",
                    violationDetection: "periodic"
                },
                {
                    id: "data_falsification",
                    type: "prohibition",
                    scope: "all_roles",
                    condition: "always",
                    target: "falsify_research_data",
                    sanction: "career_termination",
                    priority: "critical",
                    violationDetection: "periodic"
                }
            ]
        }
    },
    
    performanceMetrics: {
        goalCompletionRate: 0.85,
        normComplianceRate: 0.96,
        averageMissionDuration: 120, // days
        resourceEfficiency: 0.78,
        collaborationScore: 0.92
    }
};

/**
 * Helper functions for organization fixtures
 */
export function getAllOrganizations(): MOISEPlusOrganization[] {
    return [
        HEALTHCARE_COMPLIANCE_ORG,
        FINANCIAL_TRADING_ORG,
        RESEARCH_TEAM_ORG
    ];
}

export function getOrganizationById(id: string): MOISEPlusOrganization | undefined {
    return getAllOrganizations().find(org => org.id === id);
}

export function getOrganizationByName(name: string): MOISEPlusOrganization | undefined {
    return getAllOrganizations().find(org => 
        org.name.toLowerCase().includes(name.toLowerCase())
    );
}

export function getRolesByCapability(capability: string): RoleSpecification[] {
    const roles: RoleSpecification[] = [];
    getAllOrganizations().forEach(org => {
        org.specification.structural.roles.forEach(role => {
            if (role.requiredCapabilities.includes(capability)) {
                roles.push(role);
            }
        });
    });
    return roles;
}

export function getNormsByType(type: "obligation" | "permission" | "prohibition"): NormSpecification[] {
    const norms: NormSpecification[] = [];
    getAllOrganizations().forEach(org => {
        org.specification.normative.norms
            .filter(norm => norm.type === type)
            .forEach(norm => norms.push(norm));
    });
    return norms;
}